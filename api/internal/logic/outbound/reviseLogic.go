package outbound

import (
	customerfinance "api/internal/logic/customer/finance"
	"api/internal/svc"
	"api/internal/types"
	"api/model"
	financeCode "api/pkg/code"
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"github.com/zeromicro/go-zero/core/logx"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ReviseLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewReviseLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ReviseLogic {
	return &ReviseLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ReviseLogic) Revise(req *types.OutboundOrderReviseRequest) (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)

	session, err := l.svcCtx.DBClient.StartSession()
	if err != nil {
		fmt.Printf("[Error]核价出库单[%s]创建事务:%s\n", req.Code, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	defer session.EndSession(l.ctx)

	dbCtx := mongo.NewSessionContext(l.ctx, session)
	if err = session.StartTransaction(); err != nil {
		fmt.Printf("[Error]核价出库单[%s]开启事务:%s\n", req.Code, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	order, err := l.findOutboundOrder(dbCtx, strings.TrimSpace(req.Code))
	if err != nil {
		_ = session.AbortTransaction(dbCtx)
		if err == mongo.ErrNoDocuments {
			resp.Code = http.StatusBadRequest
			resp.Msg = "出库单不存在"
			return resp, nil
		}
		fmt.Printf("[Error]查询出库单[%s]:%s\n", req.Code, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	if strings.TrimSpace(req.CustomerId) != order.CustomerId {
		_ = session.AbortTransaction(dbCtx)
		resp.Code = http.StatusBadRequest
		resp.Msg = "出库单与客户不匹配"
		return resp, nil
	}

	materials, err := l.outboundMaterials(dbCtx, order.Code)
	if err != nil {
		_ = session.AbortTransaction(dbCtx)
		fmt.Printf("[Error]查询出库单[%s]物料列表:%s\n", order.Code, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	materialsMap := make(map[string]model.OutboundOrderMaterial, len(materials))
	for _, one := range materials {
		materialsMap[one.MaterialId] = one
	}

	pricePatch := make(map[string]float64, len(req.MaterialsPrice))
	for _, one := range req.MaterialsPrice {
		if _, exists := pricePatch[one.MaterialId]; exists {
			_ = session.AbortTransaction(dbCtx)
			resp.Code = http.StatusBadRequest
			resp.Msg = "物料价格不允许重复提交"
			return resp, nil
		}
		if _, ok := materialsMap[one.MaterialId]; !ok {
			_ = session.AbortTransaction(dbCtx)
			resp.Code = http.StatusBadRequest
			resp.Msg = "缺少部分物料的价格"
			return resp, nil
		}
		pricePatch[one.MaterialId] = one.Price
	}
	if len(materials) < len(req.MaterialsPrice) {
		_ = session.AbortTransaction(dbCtx)
		resp.Code = http.StatusBadRequest
		resp.Msg = "请勿提供多余物料"
		return resp, nil
	}

	// 价格补丁先写入，再在同一事务内重读完整物料清单，避免并发补价时用旧快照计算应收。
	if err = l.patchMaterialPrices(dbCtx, order.Code, req.MaterialsPrice); err != nil {
		_ = session.AbortTransaction(dbCtx)
		fmt.Printf("[Error]批量更新出库单[%s]物料单价:%s\n", order.Code, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	if err = l.upsertCustomerMaterialPrices(dbCtx, req.CustomerId, order.CustomerName, materialsMap, req.MaterialsPrice); err != nil {
		_ = session.AbortTransaction(dbCtx)
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	materials, err = l.outboundMaterials(dbCtx, order.Code)
	if err != nil {
		_ = session.AbortTransaction(dbCtx)
		fmt.Printf("[Error]重读出库单[%s]物料列表:%s\n", order.Code, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	amount, priceStatus := computeOutboundPricing(order, materials)
	outboundTypeCode := order.TypeCode
	if outboundTypeCode == "" {
		outboundTypeCode = financeCode.OutboundTypeCodeFromLabel(order.Type)
	}

	revisionNo := order.RevisionNo + 1
	nextARStatus := financeCode.OutboundARStatusNotApplicable
	shouldCreateAR := financeCode.ShouldCreateReceivable(outboundTypeCode)
	if shouldCreateAR {
		nextARStatus = financeCode.OutboundARStatusPending
	}

	// 已签收出库单只能通过追加流水修正应收，保留完整审计链。
	if order.Status == "已签收" && shouldCreateAR {
		postedAmount, originalTransactionId, hasAdjustment, e := l.orderPostedAmount(dbCtx, order)
		if e != nil {
			_ = session.AbortTransaction(dbCtx)
			if msg, isBusiness := customerfinance.IsBusinessError(e); isBusiness {
				resp.Code = http.StatusBadRequest
				resp.Msg = msg
				return resp, nil
			}
			fmt.Printf("[Error]查询出库单[%s]应收流水:%s\n", order.Code, e.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务器内部错误"
			return resp, nil
		}

		if priceStatus != financeCode.OutboundPriceStatusFullyPriced {
			if !postedAmount.IsZero() {
				_ = session.AbortTransaction(dbCtx)
				resp.Code = http.StatusBadRequest
				resp.Msg = "已入账出库单不能改为未定价或部分定价"
				return resp, nil
			}
		} else {
			insertedAdjustment := false
			amountDelta := amount.Sub(postedAmount)
			if !amountDelta.IsZero() {
				transactionType := financeCode.TransactionTypeARAdjustment
				direction := financeCode.TransactionDirectionIncrease
				remark := fmt.Sprintf("出库单[%s]核价调整", order.Code)
				idempotencyKey := financeCode.IdempotencyKey(financeCode.TransactionTypeARAdjustment, order.Id.Hex(), fmt.Sprintf("%d", revisionNo))
				if amountDelta.IsNegative() {
					direction = financeCode.TransactionDirectionDecrease
				}
				if originalTransactionId == "" && amountDelta.IsPositive() {
					transactionType = financeCode.TransactionTypeOutboundAR
					direction = financeCode.TransactionDirectionIncrease
					remark = "补齐价格后生成应收"
					idempotencyKey = financeCode.IdempotencyKey(financeCode.TransactionTypeOutboundAR, order.Id.Hex())
				}
				if transactionType == financeCode.TransactionTypeARAdjustment {
					insertedAdjustment = true
				}

				now := time.Now().Unix()
				if _, err = customerfinance.PostTransaction(dbCtx, l.svcCtx, customerfinance.PostTransactionRequest{
					TransactionType:       transactionType,
					Direction:             direction,
					Status:                financeCode.TransactionStatusConfirmed,
					Code:                  fmt.Sprintf("CT-%s-%d", time.Now().Format("2006-01-02-15-04-05"), time.Now().UnixMilli()%1000),
					OrderCode:             order.Code,
					SourceType:            financeCode.SourceTypeOutboundOrder,
					SourceId:              order.Id.Hex(),
					SourceCode:            order.Code,
					IdempotencyKey:        idempotencyKey,
					OriginalTransactionId: originalTransactionId,
					CustomerId:            order.CustomerId,
					CustomerName:          order.CustomerName,
					Amount:                amountDelta.Abs().InexactFloat64(),
					Annex:                 order.Annex,
					Remark:                remark,
					Time:                  now,
					Creator:               l.ctx.Value("uid").(string),
					CreatorName:           l.ctx.Value("name").(string),
					CreatedAt:             now,
				}); err != nil {
					_ = session.AbortTransaction(dbCtx)
					if msg, isBusiness := customerfinance.IsBusinessError(err); isBusiness {
						resp.Code = http.StatusBadRequest
						resp.Msg = msg
						return resp, nil
					}
					fmt.Printf("[Error]新增出库单[%s]核价调整流水:%s\n", order.Code, err.Error())
					resp.Code = http.StatusInternalServerError
					resp.Msg = "服务器内部错误"
					return resp, nil
				}
			}

			nextARStatus = financeCode.OutboundARStatusPosted
			if hasAdjustment || insertedAdjustment {
				nextARStatus = financeCode.OutboundARStatusAdjusted
			}
		}
	}

	update := bson.M{
		"$set": bson.M{
			"total_amount": amount.InexactFloat64(),
			"type_code":    outboundTypeCode,
			"price_status": priceStatus,
			"ar_status":    nextARStatus,
			"revision_no":  revisionNo,
			"updated_at":   time.Now().Unix(),
		},
	}
	updateRes, err := l.svcCtx.OutboundOrderModel.UpdateOne(dbCtx, outboundRevisionFilter(order.Id, order.RevisionNo), update)
	if err != nil {
		_ = session.AbortTransaction(dbCtx)
		fmt.Printf("[Error]更新出库单[%s]财务状态:%s\n", order.Code, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	if updateRes.MatchedCount != 1 {
		_ = session.AbortTransaction(dbCtx)
		resp.Code = http.StatusBadRequest
		resp.Msg = "出库单已被其他操作修改，请刷新后重试"
		return resp, nil
	}

	if err = session.CommitTransaction(dbCtx); err != nil {
		fmt.Printf("[Error]核价出库单[%s]提交事务:%s\n", req.Code, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}

func (l *ReviseLogic) findOutboundOrder(ctx context.Context, code string) (model.OutboundOrder, error) {
	var order model.OutboundOrder
	err := l.svcCtx.OutboundOrderModel.FindOne(ctx, bson.M{"code": code}).Decode(&order)
	return order, err
}

func (l *ReviseLogic) outboundMaterials(ctx context.Context, orderCode string) ([]model.OutboundOrderMaterial, error) {
	cur, err := l.svcCtx.OutboundMaterialModel.Find(ctx, bson.M{"order_code": orderCode})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var materials []model.OutboundOrderMaterial
	if err = cur.All(ctx, &materials); err != nil {
		return nil, err
	}
	return materials, nil
}

func (l *ReviseLogic) patchMaterialPrices(ctx context.Context, orderCode string, prices []types.OutboundMaterialPrice) error {
	bulkWrites := make([]mongo.WriteModel, 0, len(prices))
	for _, one := range prices {
		bulkWrite := mongo.NewUpdateOneModel()
		bulkWrite.SetFilter(bson.D{{"order_code", orderCode}, {"material_id", one.MaterialId}})
		bulkWrite.SetUpdate(bson.D{{"$set", bson.D{{"price", one.Price}}}})
		bulkWrites = append(bulkWrites, bulkWrite)
	}
	if len(bulkWrites) == 0 {
		return nil
	}
	bulkRes, err := l.svcCtx.OutboundMaterialModel.BulkWrite(ctx, bulkWrites, &options.BulkWriteOptions{})
	if err != nil {
		return err
	}
	if bulkRes.MatchedCount != int64(len(bulkWrites)) {
		return fmt.Errorf("物料价格更新数量不匹配")
	}
	return nil
}

func (l *ReviseLogic) upsertCustomerMaterialPrices(ctx context.Context, customerId, customerName string, materialsMap map[string]model.OutboundOrderMaterial, prices []types.OutboundMaterialPrice) error {
	for _, one := range prices {
		if one.Price <= 0 {
			continue
		}
		update := bson.M{
			"$set": bson.M{
				"material":      one.MaterialId,
				"customer_id":   customerId,
				"customer_name": customerName,
				"price":         one.Price,
				"creator":       l.ctx.Value("uid").(string),
				"creator_name":  l.ctx.Value("name").(string),
				"created_at":    time.Now().Unix(),
			},
		}
		if _, err := l.svcCtx.MaterialPriceModel.UpdateOne(
			ctx,
			bson.M{"material": one.MaterialId, "customer_id": customerId, "price": one.Price},
			update,
			options.Update().SetUpsert(true),
		); err != nil {
			material := materialsMap[one.MaterialId]
			fmt.Printf("[Error]存储客户[%s]物料[%s][%s]单价:%s\n", customerId, one.MaterialId, material.Model, err.Error())
			return err
		}
	}
	return nil
}

func (l *ReviseLogic) orderPostedAmount(ctx context.Context, order model.OutboundOrder) (decimal.Decimal, string, bool, error) {
	transactionFilter := bson.M{
		"$or": []bson.M{
			{"source_type": financeCode.SourceTypeOutboundOrder, "source_id": order.Id.Hex()},
			{"source_type": financeCode.SourceTypeSystemRecount, "source_id": order.Id.Hex()},
			{"order_code": order.Code},
		},
	}
	txCur, err := l.svcCtx.CustomerTransactionModel.Find(ctx, transactionFilter)
	if err != nil {
		return decimal.Zero, "", false, err
	}
	defer txCur.Close(ctx)

	var transactions []model.CustomerTransaction
	if err = txCur.All(ctx, &transactions); err != nil {
		return decimal.Zero, "", false, err
	}

	postedAmount := decimal.Zero
	originalTransactionId := ""
	hasAdjustment := false
	for _, record := range transactions {
		typeCode := record.TransactionType
		if typeCode == "" {
			if normalized, ok := financeCode.TransactionTypeCode(record.Type); ok {
				typeCode = normalized
			}
		}
		if typeCode != financeCode.TransactionTypeOutboundAR && typeCode != financeCode.TransactionTypeARAdjustment {
			continue
		}
		if !financeCode.TransactionStatusCountsInBalance(record.Status) {
			continue
		}
		if originalTransactionId == "" && typeCode == financeCode.TransactionTypeOutboundAR {
			originalTransactionId = record.Id.Hex()
		}
		if typeCode == financeCode.TransactionTypeARAdjustment {
			hasAdjustment = true
		}
		delta, ok := financeCode.TransactionBalanceDelta(typeCode, record.Direction, record.Amount)
		if !ok {
			return decimal.Zero, "", false, &customerfinance.BusinessError{Message: "出库单存在方向异常的应收流水"}
		}
		postedAmount = postedAmount.Add(decimal.NewFromFloat(delta))
	}
	return postedAmount, originalTransactionId, hasAdjustment, nil
}

func computeOutboundPricing(order model.OutboundOrder, materials []model.OutboundOrderMaterial) (decimal.Decimal, string) {
	amount := decimal.NewFromFloat(order.CarrierCost).Add(decimal.NewFromFloat(order.OtherCost))
	if len(materials) == 0 {
		return amount, financeCode.OutboundPriceStatusUnpriced
	}

	unpricedCount := 0
	for _, one := range materials {
		if one.Price <= 0 {
			unpricedCount++
		}
		amount = decimal.NewFromFloat(one.Quantity).Mul(decimal.NewFromFloat(one.Price)).Add(amount)
	}

	switch {
	case unpricedCount == len(materials):
		return amount, financeCode.OutboundPriceStatusUnpriced
	case unpricedCount > 0:
		return amount, financeCode.OutboundPriceStatusPartialPriced
	default:
		return amount, financeCode.OutboundPriceStatusFullyPriced
	}
}

func outboundRevisionFilter(orderId primitive.ObjectID, revisionNo int64) bson.M {
	if revisionNo == 0 {
		return bson.M{
			"_id": orderId,
			"$or": []bson.M{
				{"revision_no": int64(0)},
				{"revision_no": bson.M{"$exists": false}},
			},
		}
	}
	return bson.M{"_id": orderId, "revision_no": revisionNo}
}
