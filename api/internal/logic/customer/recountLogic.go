package customer

import (
	customerfinance "api/internal/logic/customer/finance"
	"api/internal/svc"
	"api/internal/types"
	"api/model"
	financeCode "api/pkg/code"
	"context"
	"fmt"
	"math"
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

const recountAmountTolerance = 0.000001

type RecountLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// orderPostingSnapshot records how much receivable has already been posted for
// an outbound order. The original transaction id is kept so later adjustments
// can be linked back to the first receivable entry.
type orderPostingSnapshot struct {
	PostedAmount          decimal.Decimal
	OriginalTransactionId string
	HasAdjustment         bool
}

func NewRecountLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RecountLogic {
	return &RecountLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// Recount is an immediate repair action. It intentionally does not create a
// preview, confirmation, or approval record; the caller triggers the repair
// directly and this logic writes missing receivable adjustments plus balance
// caches in customer-sized transactions.
func (l *RecountLogic) Recount() (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)

	customers, _, _, err := l.activeCustomers(l.ctx)
	if err != nil {
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	gapWarnings := make([]string, 0)
	for idx, customer := range customers {
		warnings, e := l.recountCustomer(customer.Id)
		if e != nil {
			if msg, isBusiness := customerfinance.IsBusinessError(e); isBusiness {
				resp.Code = http.StatusBadRequest
				resp.Msg = fmt.Sprintf("已处理%d个客户，客户[%s]重算失败：%s", idx, customer.Name, msg)
				return resp, nil
			}
			fmt.Printf("[Error]客户[%s]应收重算:%s\n", customer.Name, e.Error())
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务器内部错误"
			return resp, nil
		}
		gapWarnings = append(gapWarnings, warnings...)
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	if len(gapWarnings) > 0 {
		samples := gapWarnings
		if len(samples) > 3 {
			samples = samples[:3]
		}
		resp.Msg = fmt.Sprintf("成功，已按流水重写客户余额；发现%d个客户缓存差异，示例：%s", len(gapWarnings), strings.Join(samples, "；"))
	}
	return resp, nil
}

func (l *RecountLogic) recountCustomer(customerId primitive.ObjectID) ([]string, error) {
	session, err := l.svcCtx.DBClient.StartSession()
	if err != nil {
		return nil, err
	}
	defer session.EndSession(l.ctx)

	dbCtx := mongo.NewSessionContext(l.ctx, session)
	if err = session.StartTransaction(); err != nil {
		return nil, err
	}

	customer, err := l.activeCustomerByID(dbCtx, customerId)
	if err != nil {
		_ = session.AbortTransaction(dbCtx)
		return nil, err
	}

	orders, err := l.receivableOrdersByCustomer(dbCtx, customer.Id.Hex())
	if err != nil {
		_ = session.AbortTransaction(dbCtx)
		return nil, err
	}

	orderCodes := make([]string, 0, len(orders))
	orderIdByCode := make(map[string]string, len(orders))
	for _, order := range orders {
		orderCodes = append(orderCodes, order.Code)
		orderIdByCode[order.Code] = order.Id.Hex()
	}

	materialsByOrder, err := l.outboundMaterialsByOrder(dbCtx, orderCodes)
	if err != nil {
		_ = session.AbortTransaction(dbCtx)
		return nil, err
	}

	records, err := l.customerTransactions(dbCtx, []string{customer.Id.Hex()})
	if err != nil {
		_ = session.AbortTransaction(dbCtx)
		return nil, err
	}
	postingByOrder, err := buildOrderPostingSnapshots(records, orderIdByCode)
	if err != nil {
		_ = session.AbortTransaction(dbCtx)
		return nil, err
	}

	for _, order := range orders {
		outboundTypeCode := order.TypeCode
		if outboundTypeCode == "" {
			outboundTypeCode = financeCode.OutboundTypeCodeFromLabel(order.Type)
		}
		shouldCreateAR := financeCode.ShouldCreateReceivable(outboundTypeCode)
		totalAmount, priceStatus := computeOutboundAmount(order, materialsByOrder[order.Code])
		nextARStatus := financeCode.OutboundARStatusNotApplicable
		if shouldCreateAR {
			nextARStatus = financeCode.OutboundARStatusPending
		}

		snapshot := postingByOrder[order.Id.Hex()]
		if shouldCreateAR && priceStatus != financeCode.OutboundPriceStatusFullyPriced && !snapshot.PostedAmount.IsZero() {
			_ = session.AbortTransaction(dbCtx)
			return nil, &customerfinance.BusinessError{Message: fmt.Sprintf("出库单[%s]未完整定价但已存在应收流水，请先补齐物料价格或人工核对", order.Code)}
		}

		wroteTransaction := false
		if shouldCreateAR && priceStatus == financeCode.OutboundPriceStatusFullyPriced {
			delta := totalAmount.Sub(snapshot.PostedAmount)
			if !delta.IsZero() {
				transactionType := financeCode.TransactionTypeARAdjustment
				direction := financeCode.TransactionDirectionIncrease
				remark := fmt.Sprintf("重算修复出库单[%s]应收差异", order.Code)
				idempotencyKey := financeCode.IdempotencyKey(financeCode.TransactionTypeARAdjustment, order.Id.Hex(), "recount", fmt.Sprintf("%d", time.Now().UnixNano()))
				if delta.IsNegative() {
					direction = financeCode.TransactionDirectionDecrease
				}
				if snapshot.OriginalTransactionId == "" && delta.IsPositive() {
					transactionType = financeCode.TransactionTypeOutboundAR
					direction = financeCode.TransactionDirectionIncrease
					remark = "重算补充出库应收"
					idempotencyKey = financeCode.IdempotencyKey(financeCode.TransactionTypeOutboundAR, order.Id.Hex())
				}

				now := time.Now().Unix()
				_, err = customerfinance.PostTransaction(dbCtx, l.svcCtx, customerfinance.PostTransactionRequest{
					TransactionType:       transactionType,
					Direction:             direction,
					Status:                financeCode.TransactionStatusConfirmed,
					Code:                  fmt.Sprintf("CT-RECOUNT-%s-%d", order.Code, time.Now().UnixMilli()%1000),
					OrderCode:             order.Code,
					SourceType:            financeCode.SourceTypeSystemRecount,
					SourceId:              order.Id.Hex(),
					SourceCode:            order.Code,
					IdempotencyKey:        idempotencyKey,
					OriginalTransactionId: snapshot.OriginalTransactionId,
					CustomerId:            order.CustomerId,
					CustomerName:          order.CustomerName,
					Amount:                math.Abs(delta.InexactFloat64()),
					Annex:                 order.Annex,
					Remark:                remark,
					Time:                  now,
					Creator:               l.ctx.Value("uid").(string),
					CreatorName:           l.ctx.Value("name").(string),
					CreatedAt:             now,
				})
				if err != nil {
					_ = session.AbortTransaction(dbCtx)
					return nil, err
				}
				wroteTransaction = true
				snapshot.HasAdjustment = snapshot.HasAdjustment || transactionType == financeCode.TransactionTypeARAdjustment
			}

			nextARStatus = financeCode.OutboundARStatusPosted
			if snapshot.HasAdjustment {
				nextARStatus = financeCode.OutboundARStatusAdjusted
			}
		}

		if !orderFinanceStateChanged(order, outboundTypeCode, totalAmount, priceStatus, nextARStatus) && !wroteTransaction {
			continue
		}

		update := bson.M{
			"$set": bson.M{
				"type_code":    outboundTypeCode,
				"total_amount": totalAmount.InexactFloat64(),
				"price_status": priceStatus,
				"ar_status":    nextARStatus,
				"updated_at":   time.Now().Unix(),
			},
			"$inc": bson.M{"revision_no": int64(1)},
		}
		updateRes, e := l.svcCtx.OutboundOrderModel.UpdateOne(dbCtx, recountRevisionFilter(order.Id, order.RevisionNo), update)
		if e != nil {
			_ = session.AbortTransaction(dbCtx)
			return nil, e
		}
		if updateRes.MatchedCount != 1 {
			_ = session.AbortTransaction(dbCtx)
			return nil, &customerfinance.BusinessError{Message: fmt.Sprintf("出库单[%s]已被其他操作修改，请刷新后重试", order.Code)}
		}
	}

	records, err = l.customerTransactions(dbCtx, []string{customer.Id.Hex()})
	if err != nil {
		_ = session.AbortTransaction(dbCtx)
		return nil, err
	}

	customer, err = l.activeCustomerByID(dbCtx, customer.Id)
	if err != nil {
		_ = session.AbortTransaction(dbCtx)
		return nil, err
	}

	gapWarnings, err := balanceCacheGapSummary([]model.Customer{customer}, records)
	if err != nil {
		_ = session.AbortTransaction(dbCtx)
		return nil, err
	}

	if err = l.rewriteCustomerBalances(dbCtx, []model.Customer{customer}, records); err != nil {
		_ = session.AbortTransaction(dbCtx)
		return nil, err
	}

	if err = session.CommitTransaction(dbCtx); err != nil {
		return nil, err
	}
	return gapWarnings, nil
}

func (l *RecountLogic) activeCustomers(ctx context.Context) ([]model.Customer, []string, map[string]struct{}, error) {
	cur, err := l.svcCtx.CustomerModel.Find(ctx, bson.M{"is_deleted": bson.M{"$ne": true}, "status": bson.M{"$ne": "删除"}})
	if err != nil {
		fmt.Printf("[Error]查询客户列表:%s\n", err.Error())
		return nil, nil, nil, err
	}
	defer cur.Close(ctx)

	var customers []model.Customer
	if err = cur.All(ctx, &customers); err != nil {
		fmt.Printf("[Error]解析客户列表:%s\n", err.Error())
		return nil, nil, nil, err
	}

	activeCustomerIds := make([]string, 0, len(customers))
	activeCustomers := make(map[string]struct{}, len(customers))
	for _, customer := range customers {
		activeCustomerIds = append(activeCustomerIds, customer.Id.Hex())
		activeCustomers[customer.Id.Hex()] = struct{}{}
	}
	return customers, activeCustomerIds, activeCustomers, nil
}

func (l *RecountLogic) activeCustomerByID(ctx context.Context, id primitive.ObjectID) (model.Customer, error) {
	var customer model.Customer
	err := l.svcCtx.CustomerModel.FindOne(ctx, bson.M{"_id": id, "is_deleted": bson.M{"$ne": true}, "status": bson.M{"$ne": "删除"}}).Decode(&customer)
	if err == mongo.ErrNoDocuments {
		return customer, &customerfinance.BusinessError{Message: "客户不存在或已删除"}
	}
	return customer, err
}

func (l *RecountLogic) receivableOrders(ctx context.Context) ([]model.OutboundOrder, error) {
	return l.receivableOrdersByCustomer(ctx, "")
}

func (l *RecountLogic) receivableOrdersByCustomer(ctx context.Context, customerId string) ([]model.OutboundOrder, error) {
	filter := bson.M{
		"status": "已签收",
		"$or": []bson.M{
			{"type_code": bson.M{"$in": []string{financeCode.OutboundTypeCodeSales, financeCode.OutboundTypeCodeSample}}},
			{"type_code": "", "type": bson.M{"$in": []string{"销售出库", "样品出库"}}},
			{"type_code": bson.M{"$exists": false}, "type": bson.M{"$in": []string{"销售出库", "样品出库"}}},
		},
		"customer_id": bson.M{"$ne": ""},
	}
	if customerId != "" {
		filter["customer_id"] = customerId
	}

	cur, err := l.svcCtx.OutboundOrderModel.Find(ctx, filter)
	if err != nil {
		fmt.Printf("[Error]查询订单列表:%s\n", err.Error())
		return nil, err
	}
	defer cur.Close(ctx)

	var orders []model.OutboundOrder
	if err = cur.All(ctx, &orders); err != nil {
		fmt.Printf("[Error]解析订单列表:%s\n", err.Error())
		return nil, err
	}
	return orders, nil
}

func (l *RecountLogic) outboundMaterialsByOrder(ctx context.Context, orderCodes []string) (map[string][]model.OutboundOrderMaterial, error) {
	grouped := make(map[string][]model.OutboundOrderMaterial)
	if len(orderCodes) == 0 {
		return grouped, nil
	}

	cur, err := l.svcCtx.OutboundMaterialModel.Find(ctx, bson.M{"order_code": bson.M{"$in": orderCodes}})
	if err != nil {
		fmt.Printf("[Error]查询出库单物料:%s\n", err.Error())
		return nil, err
	}
	defer cur.Close(ctx)

	var materials []model.OutboundOrderMaterial
	if err = cur.All(ctx, &materials); err != nil {
		fmt.Printf("[Error]解析出库单物料:%s\n", err.Error())
		return nil, err
	}
	for _, material := range materials {
		grouped[material.OrderCode] = append(grouped[material.OrderCode], material)
	}
	return grouped, nil
}

func (l *RecountLogic) customerTransactions(ctx context.Context, activeCustomerIds []string) ([]model.CustomerTransaction, error) {
	if len(activeCustomerIds) == 0 {
		return nil, nil
	}
	cur, err := l.svcCtx.CustomerTransactionModel.Find(ctx, bson.M{
		"customer_id": bson.M{"$in": activeCustomerIds},
		"$or": []bson.M{
			{"status": financeCode.TransactionStatusConfirmed},
			{"status": ""},
			{"status": bson.M{"$exists": false}},
		},
	})
	if err != nil {
		fmt.Printf("[Error]查询客户交易流水:%s\n", err.Error())
		return nil, err
	}
	defer cur.Close(ctx)

	var records []model.CustomerTransaction
	if err = cur.All(ctx, &records); err != nil {
		fmt.Printf("[Error]解析客户交易流水:%s\n", err.Error())
		return nil, err
	}
	return records, nil
}

func computeOutboundAmount(order model.OutboundOrder, materials []model.OutboundOrderMaterial) (decimal.Decimal, string) {
	totalAmount := decimal.NewFromFloat(order.CarrierCost).Add(decimal.NewFromFloat(order.OtherCost))
	if len(materials) == 0 {
		return totalAmount, financeCode.OutboundPriceStatusUnpriced
	}

	unpricedCount := 0
	for _, material := range materials {
		if material.Price <= 0 {
			unpricedCount++
		}
		totalAmount = decimal.NewFromFloat(material.Quantity).Mul(decimal.NewFromFloat(material.Price)).Add(totalAmount)
	}

	switch {
	case unpricedCount == len(materials):
		return totalAmount, financeCode.OutboundPriceStatusUnpriced
	case unpricedCount > 0:
		return totalAmount, financeCode.OutboundPriceStatusPartialPriced
	default:
		return totalAmount, financeCode.OutboundPriceStatusFullyPriced
	}
}

func buildOrderPostingSnapshots(records []model.CustomerTransaction, orderIdByCode map[string]string) (map[string]orderPostingSnapshot, error) {
	result := make(map[string]orderPostingSnapshot)
	for _, record := range records {
		typeCode := record.TransactionType
		if typeCode == "" {
			if normalized, ok := financeCode.TransactionTypeCode(record.Type); ok {
				typeCode = normalized
			}
		}
		if typeCode != financeCode.TransactionTypeOutboundAR && typeCode != financeCode.TransactionTypeARAdjustment {
			continue
		}

		orderId := ""
		if (record.SourceType == financeCode.SourceTypeOutboundOrder || record.SourceType == financeCode.SourceTypeSystemRecount) && record.SourceId != "" {
			orderId = record.SourceId
		} else {
			orderId = orderIdByCode[record.OrderCode]
		}
		if orderId == "" {
			continue
		}

		delta, ok := financeCode.TransactionBalanceDelta(typeCode, record.Direction, record.Amount)
		if !ok {
			return nil, &customerfinance.BusinessError{Message: "存在方向异常的出库应收流水"}
		}

		snapshot := result[orderId]
		snapshot.PostedAmount = snapshot.PostedAmount.Add(decimal.NewFromFloat(delta))
		if typeCode == financeCode.TransactionTypeOutboundAR && snapshot.OriginalTransactionId == "" {
			snapshot.OriginalTransactionId = record.Id.Hex()
		}
		if typeCode == financeCode.TransactionTypeARAdjustment {
			snapshot.HasAdjustment = true
		}
		result[orderId] = snapshot
	}
	return result, nil
}

func customerBalanceSnapshots(records []model.CustomerTransaction) (map[string]decimal.Decimal, error) {
	balances := make(map[string]decimal.Decimal)
	for _, record := range records {
		if !financeCode.TransactionStatusCountsInBalance(record.Status) {
			continue
		}

		typeCode := record.TransactionType
		if typeCode == "" {
			if normalized, ok := financeCode.TransactionTypeCode(record.Type); ok {
				typeCode = normalized
			}
		}
		delta, ok := financeCode.TransactionBalanceDelta(typeCode, record.Direction, record.Amount)
		if !ok {
			return nil, &customerfinance.BusinessError{Message: "存在未知类型或方向异常的交易流水"}
		}
		balances[record.CustomerId] = balances[record.CustomerId].Add(decimal.NewFromFloat(delta))
	}
	return balances, nil
}

func balanceCacheGapSummary(customers []model.Customer, records []model.CustomerTransaction) ([]string, error) {
	balances, err := customerBalanceSnapshots(records)
	if err != nil {
		return nil, err
	}

	warnings := make([]string, 0)
	for _, customer := range customers {
		ledgerNet := balances[customer.Id.Hex()]
		cachedNet := decimal.NewFromFloat(customer.ReceivableBalance).Sub(decimal.NewFromFloat(customer.CreditBalance))
		if cachedNet.Sub(ledgerNet).Abs().LessThanOrEqual(decimal.NewFromFloat(recountAmountTolerance)) {
			continue
		}

		receivable, credit := splitCustomerNetBalance(ledgerNet)
		warnings = append(warnings, fmt.Sprintf("%s(原净额%.4f,流水净额%.4f,应收%.4f,贷项%.4f)",
			customer.Name,
			cachedNet.InexactFloat64(),
			ledgerNet.InexactFloat64(),
			receivable.InexactFloat64(),
			credit.InexactFloat64(),
		))
	}
	return warnings, nil
}

func (l *RecountLogic) rewriteCustomerBalances(ctx context.Context, customers []model.Customer, records []model.CustomerTransaction) error {
	balances, err := customerBalanceSnapshots(records)
	if err != nil {
		return err
	}

	bulkWrites := make([]mongo.WriteModel, 0, len(customers))
	for _, customer := range customers {
		receivable, credit := splitCustomerNetBalance(balances[customer.Id.Hex()])
		bulkWrite := mongo.NewUpdateOneModel()
		bulkWrite.SetFilter(bson.D{
			{"_id", customer.Id},
			{"is_deleted", bson.M{"$ne": true}},
			{"status", bson.M{"$ne": "删除"}},
		})
		bulkWrite.SetUpdate(bson.D{{"$set", bson.D{
			{"receivable_balance", receivable.InexactFloat64()},
			{"credit_balance", credit.InexactFloat64()},
		}}})
		bulkWrites = append(bulkWrites, bulkWrite)
	}
	if len(bulkWrites) == 0 {
		return nil
	}

	res, err := l.svcCtx.CustomerModel.BulkWrite(ctx, bulkWrites, &options.BulkWriteOptions{})
	if err != nil {
		fmt.Printf("[Error]批量更新客户应收账款:%s\n", err.Error())
		return err
	}
	if res.MatchedCount != int64(len(bulkWrites)) {
		return &customerfinance.BusinessError{Message: "部分客户状态已变化，请刷新后重试"}
	}
	return nil
}

func splitCustomerNetBalance(net decimal.Decimal) (decimal.Decimal, decimal.Decimal) {
	tolerance := decimal.NewFromFloat(recountAmountTolerance)
	switch {
	case net.GreaterThan(tolerance):
		return net, decimal.Zero
	case net.LessThan(tolerance.Neg()):
		return decimal.Zero, net.Abs()
	default:
		return decimal.Zero, decimal.Zero
	}
}

func orderFinanceStateChanged(order model.OutboundOrder, typeCode string, total decimal.Decimal, priceStatus, arStatus string) bool {
	return order.TypeCode != typeCode ||
		math.Abs(order.TotalAmount-total.InexactFloat64()) > recountAmountTolerance ||
		order.PriceStatus != priceStatus ||
		order.ArStatus != arStatus
}

func recountRevisionFilter(orderId primitive.ObjectID, revisionNo int64) bson.M {
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
