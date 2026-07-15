package outbound

import (
	customerfinance "api/internal/logic/customer/finance"
	materialdelivery "api/internal/logic/material/delivery"
	"api/model"
	financeCode "api/pkg/code"
	"context"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"api/internal/svc"
	"api/internal/types"

	"github.com/shopspring/decimal"
	"github.com/zeromicro/go-zero/core/logx"
)

const (
	fastOutboundCoverTimes = 20
	fastOutboundLotSize    = 10000
	fastOutboundMinInbound = 10000
)

var fastOutboundCustomerTypes = map[string]struct{}{
	"销售出库": {},
	"样品出库": {},
	"赠品出库": {},
}

type FastDepartureLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 极速出库
func NewFastDepartureLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FastDepartureLogic {
	return &FastDepartureLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func fastReplenishQuantity(outboundQuantity float64) float64 {
	target := outboundQuantity * fastOutboundCoverTimes
	if target < fastOutboundMinInbound {
		target = fastOutboundMinInbound
	}

	return math.Ceil(target/fastOutboundLotSize) * fastOutboundLotSize
}

func (l *FastDepartureLogic) FastDeparture(req *types.FastOutboundRequest) (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)
	now := time.Now().Unix()
	uid := l.ctx.Value("uid").(string)
	name := l.ctx.Value("name").(string)
	code := strings.TrimSpace(req.Code)
	outboundType := strings.TrimSpace(req.Type)
	outboundTypeCode := financeCode.OutboundTypeCodeFromLabel(outboundType)
	customerIdText := strings.TrimSpace(req.CustomerId)

	if _, ok := fastOutboundCustomerTypes[outboundType]; !ok {
		resp.Code = http.StatusBadRequest
		resp.Msg = "极速出库只允许销售出库、样品出库、赠品出库"
		return resp, nil
	}

	checkTimeNotFuture := func(value int64, label string) bool {
		if value > now {
			resp.Code = http.StatusBadRequest
			resp.Msg = fmt.Sprintf("%s不能超过当前时间", label)
			return false
		}
		return true
	}
	if !checkTimeNotFuture(req.PickingTime, "拣货时间") ||
		!checkTimeNotFuture(req.PackingTime, "打包时间") ||
		!checkTimeNotFuture(req.WeighingTime, "称重时间") ||
		!checkTimeNotFuture(req.DepartureTime, "出库时间") ||
		!checkTimeNotFuture(req.ReceiptTime, "签收时间") {
		return resp, nil
	}

	departureTime := req.DepartureTime
	if departureTime == 0 {
		departureTime = now
	}
	pickingTime := req.PickingTime
	if pickingTime == 0 {
		pickingTime = departureTime
	}
	receiptTime := req.ReceiptTime
	if receiptTime == 0 {
		receiptTime = now
	}

	type timePoint struct {
		label string
		value int64
	}
	timeChain := []timePoint{
		{label: "拣货时间", value: pickingTime},
	}
	if req.PackingTime > 0 {
		timeChain = append(timeChain, timePoint{label: "打包时间", value: req.PackingTime})
	}
	if req.WeighingTime > 0 {
		timeChain = append(timeChain, timePoint{label: "称重时间", value: req.WeighingTime})
	}
	timeChain = append(timeChain,
		timePoint{label: "出库时间", value: departureTime},
		timePoint{label: "签收时间", value: receiptTime},
	)
	for index := 1; index < len(timeChain); index++ {
		if timeChain[index].value < timeChain[index-1].value {
			resp.Code = http.StatusBadRequest
			resp.Msg = fmt.Sprintf("%s不能早于%s", timeChain[index].label, timeChain[index-1].label)
			return resp, nil
		}
	}
	if receiptTime < departureTime {
		resp.Code = http.StatusBadRequest
		resp.Msg = "签收时间不能早于出库时间"
		return resp, nil
	}

	// 1. 出库单号是否冲突
	count, err := l.svcCtx.OutboundOrderModel.CountDocuments(l.ctx, bson.M{"code": code, "status": bson.M{"$ne": "删除"}})
	if err != nil {
		fmt.Printf("[Error]查询出库单是否冲突:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	if count > 0 {
		resp.Code = http.StatusBadRequest
		resp.Msg = "出库单号已占用"
		return resp, nil
	}

	// 2. 客户是否存在
	var customer model.Customer
	customerId, _ := primitive.ObjectIDFromHex(customerIdText)
	err = l.svcCtx.CustomerModel.FindOne(l.ctx, bson.M{"_id": customerId}).Decode(&customer)
	if err != nil {
		fmt.Printf("[Error]未找到客户:%s\n", err.Error())
		resp.Code = http.StatusBadRequest
		resp.Msg = "客户不存在"
		return resp, nil
	}

	// 3. 收集物料信息
	var materialIds []primitive.ObjectID
	materialQuantities := make(map[string]float64)
	materialIndexes := make(map[string]int)
	materialPrices := make(map[string]float64)
	for index, one := range req.Materials {
		materialId := strings.TrimSpace(one.MaterialId)
		mid, _ := primitive.ObjectIDFromHex(materialId)
		if _, ok := materialQuantities[materialId]; !ok {
			materialIds = append(materialIds, mid)
			materialIndexes[materialId] = index + 1
			materialPrices[materialId] = one.Price
		} else {
			resp.Code = http.StatusBadRequest
			resp.Msg = "出库物料不允许重复"
			return resp, nil
		}
		materialQuantities[materialId] += one.Quantity
	}

	cur, err := l.svcCtx.MaterialModel.Find(l.ctx, bson.M{"_id": bson.M{"$in": materialIds}})
	if err != nil {
		resp.Code = http.StatusInternalServerError
		resp.Msg = "查询物料信息失败"
		return resp, nil
	}
	defer cur.Close(l.ctx)

	var ms []model.Material
	if err = cur.All(l.ctx, &ms); err != nil {
		resp.Code = http.StatusInternalServerError
		resp.Msg = "解析物料信息失败"
		return resp, nil
	}

	materialsMap := make(map[string]model.Material)
	for _, m := range ms {
		materialsMap[m.Id.Hex()] = m
	}

	if len(materialsMap) != len(materialQuantities) {
		resp.Code = http.StatusBadRequest
		resp.Msg = "部分物料不存在"
		return resp, nil
	}

	// 4. 查询并聚合库存
	var inventories []model.Inventory
	invCur, err := l.svcCtx.InventoryModel.Find(l.ctx, bson.M{"material_id": bson.M{"$in": mapKeys(materialQuantities)}})
	if err != nil {
		resp.Code = http.StatusInternalServerError
		resp.Msg = "查询库存失败"
		return resp, nil
	}
	if err = invCur.All(l.ctx, &inventories); err != nil {
		_ = invCur.Close(l.ctx)
		resp.Code = http.StatusInternalServerError
		resp.Msg = "解析库存失败"
		return resp, nil
	}
	_ = invCur.Close(l.ctx)

	inventoryMap := make(map[string]float64)
	for _, inv := range inventories {
		inventoryMap[inv.MaterialId] += inv.AvailableQuantity
	}

	var inboundMaterials []model.InboundMaterial
	var inboundInventories []interface{}
	unpricedCount := 0

	// 5. 库存不足时，按本次出库量补足约 20 次出库需求，并按 10000 批量向上取整。
	for materialId, quantity := range materialQuantities {
		available := inventoryMap[materialId]
		if available >= quantity {
			continue
		}

		material := materialsMap[materialId]
		inboundQty := fastReplenishQuantity(quantity)

		inboundMaterials = append(inboundMaterials, model.InboundMaterial{
			Id:                material.Id.Hex(),
			Index:             materialIndexes[materialId],
			Price:             0,
			Name:              material.Name,
			Model:             material.Model,
			EstimatedQuantity: inboundQty,
			ActualQuantity:    inboundQty,
			Unit:              material.Unit,
			Status:            "入库完成",
		})

		newInv := model.Inventory{
			Id:                primitive.NewObjectID(),
			Type:              "生产入库",
			ReceiptCode:       fmt.Sprintf("AUTO_IN_%s", code),
			ReceiveCode:       fmt.Sprintf("BATCH_%s_%d", code, materialIndexes[materialId]),
			EntryTime:         now,
			MaterialId:        material.Id.Hex(),
			Name:              material.Name,
			Price:             0,
			Model:             material.Model,
			Unit:              material.Unit,
			Quantity:          inboundQty,
			AvailableQuantity: inboundQty,
			CreatorId:         uid,
			CreatorName:       name,
			CreatedAt:         now,
		}
		inboundInventories = append(inboundInventories, newInv)
		inventories = append(inventories, newInv)
	}

	outboundMaterials := make([]interface{}, 0, len(materialQuantities))
	outboundMaterialDocs := make([]model.OutboundOrderMaterial, 0, len(materialQuantities))
	bulkWrites := make([]mongo.WriteModel, 0)
	var totalAmount decimal.Decimal

	for materialId, quantity := range materialQuantities {
		material := materialsMap[materialId]
		price := materialPrices[materialId]
		if price <= 0 {
			unpricedCount++
		}
		qtyToDeduct := quantity
		deductedInventories := make([]model.OutboundMaterialInventory, 0)

		for _, inv := range inventories {
			if inv.MaterialId != materialId || inv.AvailableQuantity <= 0 || qtyToDeduct <= 0 {
				continue
			}

			take := math.Min(qtyToDeduct, inv.AvailableQuantity)
			qtyToDeduct = decimal.NewFromFloat(qtyToDeduct).Sub(decimal.NewFromFloat(take)).InexactFloat64()
			deductedInventories = append(deductedInventories, model.OutboundMaterialInventory{
				InventoryId:      inv.Id.Hex(),
				ShipmentQuantity: take,
			})

			bulkWrite := mongo.NewUpdateOneModel()
			bulkWrite.SetFilter(bson.M{"_id": inv.Id, "available_quantity": bson.M{"$gte": take}})
			bulkWrite.SetUpdate(bson.M{"$inc": bson.M{"available_quantity": -take}})
			bulkWrites = append(bulkWrites, bulkWrite)
		}

		if qtyToDeduct > 0 {
			resp.Code = http.StatusBadRequest
			resp.Msg = fmt.Sprintf("物料[%s]库存不足", material.Name)
			return resp, nil
		}

		totalAmount = decimal.NewFromFloat(quantity).Mul(decimal.NewFromFloat(price)).Add(totalAmount)

		outboundMaterial := model.OutboundOrderMaterial{
			OrderCode:     code,
			MaterialId:    material.Id.Hex(),
			Index:         materialIndexes[materialId],
			Price:         price,
			Name:          material.Name,
			Model:         material.Model,
			Specification: material.Specification,
			Quantity:      quantity,
			Unit:          material.Unit,
			Inventorys:    deductedInventories,
		}
		outboundMaterialDocs = append(outboundMaterialDocs, outboundMaterial)
		outboundMaterials = append(outboundMaterials, outboundMaterial)
	}

	session, err := l.svcCtx.DBClient.StartSession()
	if err != nil {
		fmt.Printf("[Error]极速出库创建事务:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	defer session.EndSession(l.ctx)

	dbCtx := mongo.NewSessionContext(l.ctx, session)
	if err = session.StartTransaction(); err != nil {
		fmt.Printf("[Error]极速出库开启事务:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	if len(inboundMaterials) > 0 {
		inboundReceipt := model.InboundReceipt{
			Status:        "入库完成",
			Type:          "生产入库",
			Code:          fmt.Sprintf("AUTO_IN_%s", code),
			ReceivingDate: now,
			TotalAmount:   0,
			Materials:     inboundMaterials,
			Remark:        fmt.Sprintf("极速出库自动补入库，来源出库单:%s", code),
			CreatorId:     uid,
			CreatorName:   name,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		if _, err = l.svcCtx.InboundReceiptModel.InsertOne(dbCtx, &inboundReceipt); err != nil {
			_ = session.AbortTransaction(dbCtx)
			if mongo.IsDuplicateKeyError(err) {
				resp.Code = http.StatusBadRequest
				resp.Msg = "出库单号已占用"
			} else {
				resp.Code = http.StatusInternalServerError
				resp.Msg = "自动生成入库单失败"
			}
			return resp, nil
		}

		if len(inboundInventories) > 0 {
			if _, err = l.svcCtx.InventoryModel.InsertMany(dbCtx, inboundInventories); err != nil {
				_ = session.AbortTransaction(dbCtx)
				resp.Code = http.StatusInternalServerError
				resp.Msg = "增加库存失败"
				return resp, nil
			}
		}
	}

	if len(bulkWrites) > 0 {
		bulkRes, e := l.svcCtx.InventoryModel.BulkWrite(dbCtx, bulkWrites)
		if e != nil {
			_ = session.AbortTransaction(dbCtx)
			resp.Code = http.StatusInternalServerError
			resp.Msg = "扣减库存失败"
			return resp, nil
		}
		if bulkRes.ModifiedCount != int64(len(bulkWrites)) {
			_ = session.AbortTransaction(dbCtx)
			resp.Code = http.StatusBadRequest
			resp.Msg = "库存已发生变化，请重试"
			return resp, nil
		}
	}

	outboundOrder := model.OutboundOrder{
		Id:            primitive.NewObjectID(),
		Status:        "已签收",
		IsPack:        boolToInt(req.PackingTime > 0),
		IsWeigh:       boolToInt(req.WeighingTime > 0),
		Type:          outboundType,
		TypeCode:      outboundTypeCode,
		Code:          code,
		CustomerId:    customer.Id.Hex(),
		CustomerName:  customer.Name,
		TotalAmount:   totalAmount.InexactFloat64(),
		PriceStatus:   financeCode.OutboundPriceStatusFullyPriced,
		ArStatus:      financeCode.OutboundARStatusNotApplicable,
		CreatorId:     uid,
		CreatorName:   name,
		ConfirmTime:   pickingTime,
		PickingTime:   pickingTime,
		PackingTime:   req.PackingTime,
		WeighingTime:  req.WeighingTime,
		DepartureTime: departureTime,
		ReceiptTime:   receiptTime,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	switch {
	case unpricedCount == len(materialQuantities):
		outboundOrder.PriceStatus = financeCode.OutboundPriceStatusUnpriced
	case unpricedCount > 0:
		outboundOrder.PriceStatus = financeCode.OutboundPriceStatusPartialPriced
	default:
		outboundOrder.PriceStatus = financeCode.OutboundPriceStatusFullyPriced
	}
	if financeCode.ShouldCreateReceivable(outboundTypeCode) {
		outboundOrder.ArStatus = financeCode.OutboundARStatusPending
	}

	if _, err = l.svcCtx.OutboundOrderModel.InsertOne(dbCtx, &outboundOrder); err != nil {
		_ = session.AbortTransaction(dbCtx)
		if mongo.IsDuplicateKeyError(err) {
			resp.Code = http.StatusBadRequest
			resp.Msg = "出库单号已占用"
		} else {
			resp.Code = http.StatusInternalServerError
			resp.Msg = "新增出库单失败"
		}
		return resp, nil
	}

	if len(outboundMaterials) > 0 {
		if _, err = l.svcCtx.OutboundMaterialModel.InsertMany(dbCtx, outboundMaterials); err != nil {
			_ = session.AbortTransaction(dbCtx)
			resp.Code = http.StatusInternalServerError
			resp.Msg = "新增出库单物料失败"
			return resp, nil
		}
	}

	if err = materialdelivery.SyncCustomerMaterialDeliveries(dbCtx, l.svcCtx, outboundOrder, outboundMaterialDocs); err != nil {
		_ = session.AbortTransaction(dbCtx)
		fmt.Printf("[Error]极速出库[%s]维护客户物料首次交付记录:%s\n", code, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "维护客户物料首次交付记录失败"
		return resp, nil
	}

	for materialId, price := range materialPrices {
		if price <= 0 {
			continue
		}

		update := bson.M{
			"$set": bson.M{
				"material":              materialId,
				"customer_id":           customer.Id.Hex(),
				"customer_name":         customer.Name,
				"price":                 price,
				"creator":               uid,
				"creator_name":          name,
				"source_type":           financeCode.MaterialPriceSourceOutbound,
				"source_quote_id":       "",
				"source_delivery_id":    "",
				"source_valid":          true,
				"source_invalid_reason": "",
				"created_at":            now,
			},
		}
		_, err = l.svcCtx.MaterialPriceModel.UpdateOne(
			dbCtx,
			bson.M{"material": materialId, "customer_id": customer.Id.Hex(), "price": price},
			update,
			options.Update().SetUpsert(true),
		)
		if err != nil {
			_ = session.AbortTransaction(dbCtx)
			resp.Code = http.StatusInternalServerError
			resp.Msg = "记录物料价格失败"
			return resp, nil
		}
	}

	if financeCode.ShouldCreateReceivable(outboundTypeCode) && outboundOrder.PriceStatus == financeCode.OutboundPriceStatusFullyPriced {
		if _, err = customerfinance.PostTransaction(dbCtx, l.svcCtx, customerfinance.PostTransactionRequest{
			TransactionType: financeCode.TransactionTypeOutboundAR,
			Status:          financeCode.TransactionStatusConfirmed,
			Code:            fmt.Sprintf("CT-%s-%d", time.Now().Format("2006-01-02-15-04-05"), time.Now().UnixMilli()%1000),
			OrderCode:       code,
			SourceType:      financeCode.SourceTypeOutboundOrder,
			SourceId:        outboundOrder.Id.Hex(),
			SourceCode:      code,
			IdempotencyKey:  financeCode.IdempotencyKey(financeCode.TransactionTypeOutboundAR, outboundOrder.Id.Hex()),
			CustomerId:      customer.Id.Hex(),
			CustomerName:    customer.Name,
			Amount:          totalAmount.InexactFloat64(),
			Remark:          "极速出库自动签收",
			Time:            receiptTime,
			Creator:         uid,
			CreatorName:     name,
			CreatedAt:       now,
		}); err != nil {
			_ = session.AbortTransaction(dbCtx)
			if msg, isBusiness := customerfinance.IsBusinessError(err); isBusiness {
				resp.Code = http.StatusBadRequest
				resp.Msg = msg
				return resp, nil
			}
			resp.Code = http.StatusInternalServerError
			resp.Msg = "生成客户交易流水失败"
			return resp, nil
		}

		if _, err = l.svcCtx.OutboundOrderModel.UpdateByID(dbCtx, outboundOrder.Id, bson.M{"$set": bson.M{"ar_status": financeCode.OutboundARStatusPosted}}); err != nil {
			_ = session.AbortTransaction(dbCtx)
			resp.Code = http.StatusInternalServerError
			resp.Msg = "更新出库单应收状态失败"
			return resp, nil
		}
	}

	if err = session.CommitTransaction(dbCtx); err != nil {
		resp.Code = http.StatusInternalServerError
		resp.Msg = "极速出库事务提交失败"
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "极速出库成功"
	return resp, nil
}

func mapKeys(m map[string]float64) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	return keys
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
