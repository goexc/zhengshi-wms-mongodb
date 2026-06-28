package quote

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"api/internal/svc"
	"api/internal/types"
	"api/model"
	quoteCode "api/pkg/code"

	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type quoteService struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

type quoteCalculation struct {
	costItems    []model.MaterialQuoteCostItem
	simplePrice  float64
	totalCost    float64
	profitRate   float64
	profitAmount float64
	taxAmount    float64
	finalPrice   float64
	totalAmount  float64
}

type businessError struct {
	message string
}

func (e businessError) Error() string {
	return e.message
}

func (l *quoteService) save(req *types.MaterialQuoteSaveRequest) (resp *types.MaterialQuoteResponse, err error) {
	resp = new(types.MaterialQuoteResponse)
	if msg := validateQuoteSave(req); msg != "" {
		resp.Code = http.StatusBadRequest
		resp.Msg = msg
		return resp, nil
	}
	if strings.TrimSpace(req.Id) == "" {
		return l.create(req)
	}
	return l.update(req)
}

func (l *quoteService) create(req *types.MaterialQuoteSaveRequest) (*types.MaterialQuoteResponse, error) {
	resp := new(types.MaterialQuoteResponse)
	delivery, ok := l.findDelivery(req.DeliveryId, resp)
	if !ok {
		return resp, nil
	}

	calculation, msg := buildQuoteCalculation(req)
	if msg != "" {
		resp.Code = http.StatusBadRequest
		resp.Msg = msg
		return resp, nil
	}

	now := time.Now().Unix()
	quote := model.MaterialQuote{
		Id:                    primitive.NewObjectID(),
		QuoteNo:               newQuoteNo(),
		CustomerId:            delivery.CustomerId,
		CustomerName:          delivery.CustomerName,
		MaterialId:            delivery.MaterialId,
		MaterialName:          delivery.MaterialName,
		MaterialModel:         delivery.MaterialModel,
		MaterialSpecification: delivery.MaterialSpecification,
		MaterialUnit:          delivery.MaterialUnit,
		DeliveryId:            delivery.Id.Hex(),
		SourceOrderCode:       delivery.FirstDeliveryOrderCode,
		QuoteMode:             strings.TrimSpace(req.QuoteMode),
		Status:                quoteCode.MaterialQuoteStatusDraft,
		Currency:              quoteCurrency(req.Currency),
		CostItems:             calculation.costItems,
		SimplePrice:           calculation.simplePrice,
		TotalCost:             calculation.totalCost,
		ProfitRate:            calculation.profitRate,
		ProfitAmount:          calculation.profitAmount,
		TaxRate:               req.TaxRate,
		TaxAmount:             calculation.taxAmount,
		FinalPrice:            calculation.finalPrice,
		TotalAmount:           calculation.totalAmount,
		ValidFrom:             req.ValidFrom,
		ValidTo:               req.ValidTo,
		Remark:                strings.TrimSpace(req.Remark),
		CreatorId:             contextString(l.ctx, "uid"),
		CreatorName:           contextString(l.ctx, "name"),
		CreatedAt:             now,
		UpdatedAt:             now,
	}
	if err := withTransaction(l.ctx, l.svcCtx, func(dbCtx mongo.SessionContext) error {
		if _, err := l.svcCtx.MaterialQuoteModel.InsertOne(dbCtx, &quote); err != nil {
			return err
		}
		return refreshDeliveryQuoteState(dbCtx, l.svcCtx, quote.DeliveryId)
	}); err != nil {
		fmt.Printf("[Error]创建物料报价单失败:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	resp.Data = toTypeQuote(quote)
	return resp, nil
}

func (l *quoteService) update(req *types.MaterialQuoteSaveRequest) (*types.MaterialQuoteResponse, error) {
	resp := new(types.MaterialQuoteResponse)
	quote, ok := findQuoteById(l.ctx, l.svcCtx, req.Id, resp)
	if !ok {
		return resp, nil
	}
	if quote.Status == quoteCode.MaterialQuoteStatusPriced || quote.Status == quoteCode.MaterialQuoteStatusVoid {
		resp.Code = http.StatusBadRequest
		resp.Msg = "已定价或已作废的报价单不能修改"
		return resp, nil
	}

	calculation, msg := buildQuoteCalculation(req)
	if msg != "" {
		resp.Code = http.StatusBadRequest
		resp.Msg = msg
		return resp, nil
	}

	now := time.Now().Unix()
	update := bson.M{
		"$set": bson.M{
			"quote_mode":    strings.TrimSpace(req.QuoteMode),
			"status":        quoteCode.MaterialQuoteStatusDraft,
			"currency":      quoteCurrency(req.Currency),
			"cost_items":    calculation.costItems,
			"simple_price":  calculation.simplePrice,
			"total_cost":    calculation.totalCost,
			"profit_rate":   calculation.profitRate,
			"profit_amount": calculation.profitAmount,
			"tax_rate":      req.TaxRate,
			"tax_amount":    calculation.taxAmount,
			"final_price":   calculation.finalPrice,
			"total_amount":  calculation.totalAmount,
			"valid_from":    req.ValidFrom,
			"valid_to":      req.ValidTo,
			"remark":        strings.TrimSpace(req.Remark),
			"updated_at":    now,
		},
	}
	if err := withTransaction(l.ctx, l.svcCtx, func(dbCtx mongo.SessionContext) error {
		result, err := l.svcCtx.MaterialQuoteModel.UpdateOne(dbCtx, bson.M{
			"_id":    quote.Id,
			"status": bson.M{"$nin": []string{quoteCode.MaterialQuoteStatusPriced, quoteCode.MaterialQuoteStatusVoid}},
		}, update)
		if err != nil {
			return err
		}
		if result.MatchedCount == 0 {
			return businessError{message: "报价单状态已变化，不能修改"}
		}
		return refreshDeliveryQuoteState(dbCtx, l.svcCtx, quote.DeliveryId)
	}); err != nil {
		fmt.Printf("[Error]更新物料报价单[%s]失败:%s\n", req.Id, err.Error())
		fillMutationError(resp, err)
		return resp, nil
	}

	quote.QuoteMode = strings.TrimSpace(req.QuoteMode)
	quote.Status = quoteCode.MaterialQuoteStatusDraft
	quote.Currency = quoteCurrency(req.Currency)
	quote.CostItems = calculation.costItems
	quote.SimplePrice = calculation.simplePrice
	quote.TotalCost = calculation.totalCost
	quote.ProfitRate = calculation.profitRate
	quote.ProfitAmount = calculation.profitAmount
	quote.TaxRate = req.TaxRate
	quote.TaxAmount = calculation.taxAmount
	quote.FinalPrice = calculation.finalPrice
	quote.TotalAmount = calculation.totalAmount
	quote.ValidFrom = req.ValidFrom
	quote.ValidTo = req.ValidTo
	quote.Remark = strings.TrimSpace(req.Remark)
	quote.UpdatedAt = now

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	resp.Data = toTypeQuote(quote)
	return resp, nil
}

func (l *quoteService) findDelivery(id string, resp *types.MaterialQuoteResponse) (model.CustomerMaterialDelivery, bool) {
	deliveryId, _ := primitive.ObjectIDFromHex(id)
	var delivery model.CustomerMaterialDelivery
	err := l.svcCtx.CustomerMaterialDeliveryModel.FindOne(l.ctx, bson.M{"_id": deliveryId}).Decode(&delivery)
	switch err {
	case nil:
		return delivery, true
	case mongo.ErrNoDocuments:
		resp.Code = http.StatusBadRequest
		resp.Msg = "客户新增物料记录不存在"
		return delivery, false
	default:
		fmt.Printf("[Error]查询客户新增物料记录[%s]失败:%s\n", id, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return delivery, false
	}
}

func (l *quoteService) page(req *types.MaterialQuotePageRequest) (resp *types.MaterialQuotePageResponse, err error) {
	resp = new(types.MaterialQuotePageResponse)
	filter := bson.M{}
	if strings.TrimSpace(req.CustomerId) != "" {
		filter["customer_id"] = strings.TrimSpace(req.CustomerId)
	}
	if strings.TrimSpace(req.MaterialId) != "" {
		filter["material_id"] = strings.TrimSpace(req.MaterialId)
	}
	if strings.TrimSpace(req.DeliveryId) != "" {
		filter["delivery_id"] = strings.TrimSpace(req.DeliveryId)
	}
	if strings.TrimSpace(req.Status) != "" {
		filter["status"] = strings.TrimSpace(req.Status)
	}
	if strings.TrimSpace(req.QuoteMode) != "" {
		filter["quote_mode"] = strings.TrimSpace(req.QuoteMode)
	}
	if strings.TrimSpace(req.MaterialName) != "" {
		filter["material_name"] = primitive.Regex{Pattern: strings.TrimSpace(req.MaterialName), Options: "i"}
	}
	if strings.TrimSpace(req.MaterialModel) != "" {
		filter["material_model"] = primitive.Regex{Pattern: strings.TrimSpace(req.MaterialModel), Options: "i"}
	}

	total, err := l.svcCtx.MaterialQuoteModel.CountDocuments(l.ctx, filter)
	if err != nil {
		fmt.Printf("[Error]查询物料报价单数量失败:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	opts := options.Find().
		SetSort(bson.D{{Key: "updated_at", Value: -1}, {Key: "created_at", Value: -1}}).
		SetSkip((req.Page - 1) * req.Size).
		SetLimit(req.Size)
	cur, err := l.svcCtx.MaterialQuoteModel.Find(l.ctx, filter, opts)
	if err != nil {
		fmt.Printf("[Error]查询物料报价单失败:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}
	defer cur.Close(l.ctx)

	var quotes []model.MaterialQuote
	if err = cur.All(l.ctx, &quotes); err != nil {
		fmt.Printf("[Error]解析物料报价单列表失败:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	resp.Data.Total = total
	for _, one := range quotes {
		resp.Data.List = append(resp.Data.List, toTypeQuote(one))
	}
	return resp, nil
}

func (l *quoteService) info(req *types.MaterialQuoteIdRequest) (resp *types.MaterialQuoteResponse, err error) {
	resp = new(types.MaterialQuoteResponse)
	quote, ok := findQuoteById(l.ctx, l.svcCtx, req.Id, resp)
	if !ok {
		return resp, nil
	}
	resp.Code = http.StatusOK
	resp.Msg = "成功"
	resp.Data = toTypeQuote(quote)
	return resp, nil
}

func (l *quoteService) submit(req *types.MaterialQuoteIdRequest) (resp *types.MaterialQuoteResponse, err error) {
	resp = new(types.MaterialQuoteResponse)
	quote, ok := findQuoteById(l.ctx, l.svcCtx, req.Id, resp)
	if !ok {
		return resp, nil
	}
	if quote.Status == quoteCode.MaterialQuoteStatusVoid {
		resp.Code = http.StatusBadRequest
		resp.Msg = "已作废的报价单不能提交"
		return resp, nil
	}
	if quote.Status == quoteCode.MaterialQuoteStatusPriced {
		resp.Code = http.StatusBadRequest
		resp.Msg = "已定价的报价单不能重复提交"
		return resp, nil
	}

	now := time.Now().Unix()
	if err := withTransaction(l.ctx, l.svcCtx, func(dbCtx mongo.SessionContext) error {
		result, err := l.svcCtx.MaterialQuoteModel.UpdateOne(dbCtx, bson.M{
			"_id":    quote.Id,
			"status": bson.M{"$nin": []string{quoteCode.MaterialQuoteStatusPriced, quoteCode.MaterialQuoteStatusVoid}},
		}, bson.M{
			"$set": bson.M{
				"status":     quoteCode.MaterialQuoteStatusQuoted,
				"updated_at": now,
			},
		})
		if err != nil {
			return err
		}
		if result.MatchedCount == 0 {
			return businessError{message: "报价单状态已变化，不能提交"}
		}
		quote.Status = quoteCode.MaterialQuoteStatusQuoted
		quote.UpdatedAt = now
		return updateDeliveryFromQuote(dbCtx, l.svcCtx, quote, quoteCode.QuoteStatusQuoted)
	}); err != nil {
		fmt.Printf("[Error]提交物料报价单[%s]失败:%s\n", req.Id, err.Error())
		fillMutationError(resp, err)
		return resp, nil
	}
	quote.Status = quoteCode.MaterialQuoteStatusQuoted
	quote.UpdatedAt = now

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	resp.Data = toTypeQuote(quote)
	return resp, nil
}

func (l *quoteService) export(req *types.MaterialQuoteIdRequest) (fileName string, body []byte, resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)
	quoteResp := new(types.MaterialQuoteResponse)
	quote, ok := findQuoteById(l.ctx, l.svcCtx, req.Id, quoteResp)
	if !ok {
		resp.Code = quoteResp.Code
		resp.Msg = quoteResp.Msg
		return "", nil, resp, nil
	}

	var buf bytes.Buffer
	buf.Write([]byte{0xEF, 0xBB, 0xBF})
	writer := csv.NewWriter(&buf)
	rows := [][]string{
		{"报价单号", quote.QuoteNo},
		{"客户", quote.CustomerName},
		{"物料", quote.MaterialName},
		{"型号", quote.MaterialModel},
		{"规格", quote.MaterialSpecification},
		{"单位", quote.MaterialUnit},
		{"来源出库单", quote.SourceOrderCode},
		{"报价方式", quote.QuoteMode},
		{"最终报价单价", fmt.Sprintf("%.4f", quote.FinalPrice)},
		{"备注", quote.Remark},
		{},
	}
	for _, row := range rows {
		if err = writer.Write(row); err != nil {
			return "", nil, nil, err
		}
	}

	if quote.QuoteMode == quoteCode.QuoteModeDetailed {
		if err = writer.Write([]string{"类型", "序号", "成本项", "是否启用", "金额", "备注"}); err != nil {
			return "", nil, nil, err
		}
		costItems := append([]model.MaterialQuoteCostItem(nil), quote.CostItems...)
		sortModelCostItems(costItems)
		for _, item := range costItems {
			if err = writer.Write([]string{
				item.CategoryName,
				fmt.Sprintf("%d", item.Index),
				item.Name,
				boolText(item.Enabled),
				fmt.Sprintf("%.4f", item.Amount),
				item.Remark,
			}); err != nil {
				return "", nil, nil, err
			}
		}
	} else {
		if err = writer.Write([]string{"简单报价单价", fmt.Sprintf("%.4f", quote.SimplePrice)}); err != nil {
			return "", nil, nil, err
		}
	}
	writer.Flush()
	if err = writer.Error(); err != nil {
		return "", nil, nil, err
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return quote.QuoteNo + ".csv", buf.Bytes(), resp, nil
}

func (l *quoteService) price(req *types.MaterialQuotePriceRequest) (resp *types.MaterialQuoteResponse, err error) {
	resp = new(types.MaterialQuoteResponse)
	if req.FinalPrice <= 0 {
		resp.Code = http.StatusBadRequest
		resp.Msg = "最终定价必须大于0"
		return resp, nil
	}
	quote, ok := findQuoteById(l.ctx, l.svcCtx, req.Id, resp)
	if !ok {
		return resp, nil
	}
	if quote.Status == quoteCode.MaterialQuoteStatusVoid {
		resp.Code = http.StatusBadRequest
		resp.Msg = "已作废的报价单不能定价"
		return resp, nil
	}
	if quote.Status == quoteCode.MaterialQuoteStatusPriced {
		resp.Code = http.StatusBadRequest
		resp.Msg = "报价单已定价"
		return resp, nil
	}
	if quote.Status != quoteCode.MaterialQuoteStatusQuoted {
		resp.Code = http.StatusBadRequest
		resp.Msg = "请先提交报价后再转最终定价"
		return resp, nil
	}

	now := time.Now().Unix()
	effectiveAt := req.EffectiveAt
	if effectiveAt <= 0 {
		effectiveAt = now
	}
	totalAmount := roundMoney(req.FinalPrice)
	priceUpdate := bson.M{
		"$set": bson.M{
			"material":      quote.MaterialId,
			"customer_id":   quote.CustomerId,
			"customer_name": quote.CustomerName,
			"price":         req.FinalPrice,
			"creator":       contextString(l.ctx, "uid"),
			"creator_name":  contextString(l.ctx, "name"),
			"created_at":    effectiveAt,
		},
	}
	updateSet := bson.M{
		"status":       quoteCode.MaterialQuoteStatusPriced,
		"final_price":  req.FinalPrice,
		"total_amount": totalAmount,
		"updated_at":   now,
	}
	if strings.TrimSpace(req.Remark) != "" {
		updateSet["remark"] = strings.TrimSpace(req.Remark)
		quote.Remark = strings.TrimSpace(req.Remark)
	}
	quote.Status = quoteCode.MaterialQuoteStatusPriced
	quote.FinalPrice = req.FinalPrice
	quote.TotalAmount = totalAmount
	quote.UpdatedAt = now
	if err := withTransaction(l.ctx, l.svcCtx, func(dbCtx mongo.SessionContext) error {
		if _, err := l.svcCtx.MaterialPriceModel.UpdateOne(
			dbCtx,
			bson.M{"material": quote.MaterialId, "customer_id": quote.CustomerId, "price": req.FinalPrice},
			priceUpdate,
			options.Update().SetUpsert(true),
		); err != nil {
			return err
		}
		result, err := l.svcCtx.MaterialQuoteModel.UpdateOne(dbCtx, bson.M{
			"_id":    quote.Id,
			"status": quoteCode.MaterialQuoteStatusQuoted,
		}, bson.M{"$set": updateSet})
		if err != nil {
			return err
		}
		if result.MatchedCount == 0 {
			return businessError{message: "报价单状态已变化，不能定价"}
		}
		return updateDeliveryFromQuote(dbCtx, l.svcCtx, quote, quoteCode.QuoteStatusPriced)
	}); err != nil {
		fmt.Printf("[Error]报价转最终定价失败:%s\n", err.Error())
		fillMutationError(resp, err)
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	resp.Data = toTypeQuote(quote)
	return resp, nil
}

func (l *quoteService) void(req *types.MaterialQuoteIdRequest) (resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)
	quote, ok := findQuoteById(l.ctx, l.svcCtx, req.Id, &types.MaterialQuoteResponse{})
	if !ok {
		resp.Code = http.StatusBadRequest
		resp.Msg = "报价单不存在"
		return resp, nil
	}
	if quote.Status == quoteCode.MaterialQuoteStatusPriced {
		resp.Code = http.StatusBadRequest
		resp.Msg = "已定价的报价单不能作废"
		return resp, nil
	}
	if quote.Status == quoteCode.MaterialQuoteStatusVoid {
		resp.Code = http.StatusOK
		resp.Msg = "成功"
		return resp, nil
	}

	if err := withTransaction(l.ctx, l.svcCtx, func(dbCtx mongo.SessionContext) error {
		result, err := l.svcCtx.MaterialQuoteModel.UpdateOne(dbCtx, bson.M{
			"_id":    quote.Id,
			"status": bson.M{"$nin": []string{quoteCode.MaterialQuoteStatusPriced, quoteCode.MaterialQuoteStatusVoid}},
		}, bson.M{
			"$set": bson.M{
				"status":     quoteCode.MaterialQuoteStatusVoid,
				"updated_at": time.Now().Unix(),
			},
		})
		if err != nil {
			return err
		}
		if result.MatchedCount == 0 {
			return businessError{message: "报价单状态已变化，不能作废"}
		}
		return refreshDeliveryQuoteState(dbCtx, l.svcCtx, quote.DeliveryId)
	}); err != nil {
		fmt.Printf("[Error]作废物料报价单[%s]失败:%s\n", req.Id, err.Error())
		if isBusinessError(err) {
			resp.Code = http.StatusBadRequest
			resp.Msg = err.Error()
		} else {
			resp.Code = http.StatusInternalServerError
			resp.Msg = "服务器内部错误"
		}
		return resp, nil
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return resp, nil
}

func withTransaction(ctx context.Context, svcCtx *svc.ServiceContext, fn func(mongo.SessionContext) error) error {
	session, err := svcCtx.DBClient.StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)

	dbCtx := mongo.NewSessionContext(ctx, session)
	if err = session.StartTransaction(); err != nil {
		return err
	}
	if err = fn(dbCtx); err != nil {
		_ = session.AbortTransaction(dbCtx)
		return err
	}
	if err = session.CommitTransaction(dbCtx); err != nil {
		_ = session.AbortTransaction(dbCtx)
		return err
	}
	return nil
}

func fillMutationError(resp *types.MaterialQuoteResponse, err error) {
	if isBusinessError(err) {
		resp.Code = http.StatusBadRequest
		resp.Msg = err.Error()
		return
	}
	resp.Code = http.StatusInternalServerError
	resp.Msg = "服务器内部错误"
}

func isBusinessError(err error) bool {
	var target businessError
	return errors.As(err, &target)
}

func validateQuoteSave(req *types.MaterialQuoteSaveRequest) string {
	if !quoteCode.ValidQuoteMode(req.QuoteMode) {
		return "报价方式不正确"
	}
	if req.ValidFrom > 0 && req.ValidTo > 0 && req.ValidTo < req.ValidFrom {
		return "报价有效期结束时间不能早于开始时间"
	}
	if req.ProfitAmount < 0 || req.TaxRate < 0 {
		return "利润金额和税率不能小于0"
	}
	return ""
}

func buildQuoteCalculation(req *types.MaterialQuoteSaveRequest) (quoteCalculation, string) {
	var result quoteCalculation
	if req.QuoteMode == quoteCode.QuoteModeSimple {
		simplePrice := decimal.NewFromFloat(req.SimplePrice)
		if !simplePrice.GreaterThan(decimal.Zero) {
			return result, "简单报价必须填写大于0的报价"
		}
		taxAmount := simplePrice.Mul(decimal.NewFromFloat(req.TaxRate))
		computedFinalPrice := simplePrice.Add(taxAmount)
		finalPrice := computedFinalPrice
		if req.FinalPrice > 0 {
			finalPrice = decimal.NewFromFloat(req.FinalPrice)
		}
		if !finalPrice.GreaterThan(decimal.Zero) {
			return result, "简单报价最终定价必须大于0"
		}
		result.simplePrice = roundMoney(simplePrice.InexactFloat64())
		result.taxAmount = roundMoney(taxAmount.InexactFloat64())
		result.finalPrice = roundMoney(finalPrice.InexactFloat64())
		result.totalAmount = roundMoney(finalPrice.InexactFloat64())
		return result, ""
	}

	items, totalCost, msg := buildModelCostItems(req.CostItems)
	if msg != "" {
		return result, msg
	}
	totalCostDecimal := decimal.NewFromFloat(totalCost)
	profitAmount := decimal.NewFromFloat(req.ProfitAmount)
	taxAmount := totalCostDecimal.Add(profitAmount).Mul(decimal.NewFromFloat(req.TaxRate))
	computedFinalPrice := totalCostDecimal.Add(profitAmount).Add(taxAmount)
	finalPrice := computedFinalPrice
	if req.FinalPrice > 0 {
		finalPrice = decimal.NewFromFloat(req.FinalPrice)
	}
	if !finalPrice.GreaterThan(decimal.Zero) {
		return result, "详细报价必须存在启用成本项或填写最终报价"
	}

	result.costItems = items
	result.totalCost = roundMoney(totalCost)
	result.profitRate = profitRateFromAmounts(totalCost, req.ProfitAmount)
	result.profitAmount = roundMoney(profitAmount.InexactFloat64())
	result.taxAmount = roundMoney(taxAmount.InexactFloat64())
	result.finalPrice = roundMoney(finalPrice.InexactFloat64())
	result.totalAmount = roundMoney(finalPrice.InexactFloat64())
	return result, ""
}

func buildModelCostItems(input []types.MaterialQuoteCostItem) ([]model.MaterialQuoteCostItem, float64, string) {
	result := make([]model.MaterialQuoteCostItem, 0, len(input))
	totalCost := decimal.Zero
	for _, one := range input {
		if one.Amount < 0 {
			return nil, 0, "成本项金额不能小于0"
		}
		if one.Enabled && strings.TrimSpace(one.Name) == "" {
			return nil, 0, "启用的成本项必须填写名称"
		}
		amount := decimal.NewFromFloat(one.Amount)
		item := model.MaterialQuoteCostItem{
			Index:        one.Index,
			CategoryCode: strings.TrimSpace(one.CategoryCode),
			CategoryName: strings.TrimSpace(one.CategoryName),
			Name:         strings.TrimSpace(one.Name),
			Enabled:      one.Enabled,
			Custom:       one.Custom,
			Amount:       roundMoney(amount.InexactFloat64()),
			Remark:       strings.TrimSpace(one.Remark),
		}
		if item.CategoryCode == "" {
			item.CategoryCode = "other"
		}
		if item.CategoryName == "" {
			item.CategoryName = categoryName(item.CategoryCode)
		}
		if item.Enabled {
			totalCost = totalCost.Add(decimal.NewFromFloat(item.Amount))
		}
		result = append(result, item)
	}
	sortModelCostItems(result)
	return result, roundMoney(totalCost.InexactFloat64()), ""
}

func findQuoteById(ctx context.Context, svcCtx *svc.ServiceContext, id string, resp *types.MaterialQuoteResponse) (model.MaterialQuote, bool) {
	quoteId, _ := primitive.ObjectIDFromHex(id)
	var quote model.MaterialQuote
	err := svcCtx.MaterialQuoteModel.FindOne(ctx, bson.M{"_id": quoteId}).Decode(&quote)
	switch err {
	case nil:
		return quote, true
	case mongo.ErrNoDocuments:
		resp.Code = http.StatusBadRequest
		resp.Msg = "报价单不存在"
		return quote, false
	default:
		fmt.Printf("[Error]查询物料报价单[%s]失败:%s\n", id, err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "服务器内部错误"
		return quote, false
	}
}

func refreshDeliveryQuoteState(ctx context.Context, svcCtx *svc.ServiceContext, deliveryId string) error {
	opts := options.FindOne().SetSort(bson.D{{Key: "updated_at", Value: -1}, {Key: "created_at", Value: -1}})
	var latest model.MaterialQuote
	err := svcCtx.MaterialQuoteModel.FindOne(ctx, bson.M{
		"delivery_id": deliveryId,
		"status":      bson.M{"$ne": quoteCode.MaterialQuoteStatusVoid},
	}, opts).Decode(&latest)
	switch err {
	case nil:
		return updateDeliveryFromQuote(ctx, svcCtx, latest, deliveryQuoteStatus(latest.Status))
	case mongo.ErrNoDocuments:
		deliveryObjectId, e := primitive.ObjectIDFromHex(deliveryId)
		if e != nil {
			return e
		}
		_, e = svcCtx.CustomerMaterialDeliveryModel.UpdateByID(ctx, deliveryObjectId, bson.M{
			"$set": bson.M{
				"quote_status":    quoteCode.QuoteStatusUnquoted,
				"latest_quote_id": "",
				"latest_quote_no": "",
				"latest_price":    float64(0),
				"updated_at":      time.Now().Unix(),
			},
		})
		return e
	default:
		return err
	}
}

func updateDeliveryFromQuote(ctx context.Context, svcCtx *svc.ServiceContext, quote model.MaterialQuote, quoteStatus string) error {
	deliveryId, err := primitive.ObjectIDFromHex(quote.DeliveryId)
	if err != nil {
		return err
	}
	latestPrice := quote.FinalPrice
	if latestPrice <= 0 {
		latestPrice = quote.SimplePrice
	}
	_, err = svcCtx.CustomerMaterialDeliveryModel.UpdateByID(ctx, deliveryId, bson.M{
		"$set": bson.M{
			"quote_status":    quoteStatus,
			"latest_quote_id": quote.Id.Hex(),
			"latest_quote_no": quote.QuoteNo,
			"latest_price":    latestPrice,
			"updated_at":      time.Now().Unix(),
		},
	})
	return err
}

func deliveryQuoteStatus(quoteStatus string) string {
	switch quoteStatus {
	case quoteCode.MaterialQuoteStatusQuoted:
		return quoteCode.QuoteStatusQuoted
	case quoteCode.MaterialQuoteStatusPriced:
		return quoteCode.QuoteStatusPriced
	default:
		return quoteCode.QuoteStatusQuoting
	}
}

func toTypeQuote(one model.MaterialQuote) types.MaterialQuote {
	return types.MaterialQuote{
		Id:                    one.Id.Hex(),
		QuoteNo:               one.QuoteNo,
		CustomerId:            one.CustomerId,
		CustomerName:          one.CustomerName,
		MaterialId:            one.MaterialId,
		MaterialName:          one.MaterialName,
		MaterialModel:         one.MaterialModel,
		MaterialSpecification: one.MaterialSpecification,
		MaterialUnit:          one.MaterialUnit,
		DeliveryId:            one.DeliveryId,
		SourceOrderCode:       one.SourceOrderCode,
		QuoteMode:             one.QuoteMode,
		Status:                one.Status,
		Currency:              one.Currency,
		CostItems:             toTypeCostItems(one.CostItems),
		SimplePrice:           one.SimplePrice,
		TotalCost:             one.TotalCost,
		ProfitRate:            profitRateFromAmounts(one.TotalCost, one.ProfitAmount),
		ProfitAmount:          one.ProfitAmount,
		TaxRate:               one.TaxRate,
		TaxAmount:             one.TaxAmount,
		FinalPrice:            one.FinalPrice,
		TotalAmount:           one.TotalAmount,
		ValidFrom:             one.ValidFrom,
		ValidTo:               one.ValidTo,
		Remark:                one.Remark,
		CreatorId:             one.CreatorId,
		CreatorName:           one.CreatorName,
		CreatedAt:             one.CreatedAt,
		UpdatedAt:             one.UpdatedAt,
	}
}

func toTypeCostItems(input []model.MaterialQuoteCostItem) []types.MaterialQuoteCostItem {
	items := append([]model.MaterialQuoteCostItem(nil), input...)
	sortModelCostItems(items)
	result := make([]types.MaterialQuoteCostItem, 0, len(items))
	for _, one := range items {
		result = append(result, types.MaterialQuoteCostItem{
			Index:        one.Index,
			CategoryCode: one.CategoryCode,
			CategoryName: one.CategoryName,
			Name:         one.Name,
			Enabled:      one.Enabled,
			Custom:       one.Custom,
			Amount:       one.Amount,
			Remark:       one.Remark,
		})
	}
	return result
}

func sortModelCostItems(items []model.MaterialQuoteCostItem) {
	sort.SliceStable(items, func(i, j int) bool {
		left, right := categoryRank(items[i].CategoryCode), categoryRank(items[j].CategoryCode)
		if left != right {
			return left < right
		}
		if items[i].Index != items[j].Index {
			return items[i].Index < items[j].Index
		}
		return items[i].Name < items[j].Name
	})
}

func categoryRank(code string) int {
	switch strings.TrimSpace(code) {
	case "material":
		return 10
	case "process":
		return 20
	case "labor_equipment":
		return 30
	case "quality":
		return 40
	case "packing", "freight", "packing_logistics":
		return 50
	case "management":
		return 70
	case "tooling":
		return 80
	case "loss":
		return 90
	case "other":
		return 110
	default:
		return 999
	}
}

func categoryName(code string) string {
	switch strings.TrimSpace(code) {
	case "material":
		return "材料成本"
	case "process":
		return "加工工序成本"
	case "labor_equipment":
		return "人工/设备成本"
	case "quality":
		return "质量成本"
	case "packing", "freight", "packing_logistics":
		return "包装/物流成本"
	case "management":
		return "管理成本"
	case "tooling":
		return "模具/治具摊销"
	case "loss":
		return "损耗成本"
	default:
		return "其他成本"
	}
}

func quoteCurrency(currency string) string {
	currency = strings.TrimSpace(currency)
	if currency == "" {
		return "CNY"
	}
	return currency
}

func newQuoteNo() string {
	objectId := primitive.NewObjectID().Hex()
	return "MQ-" + time.Now().Format("20060102150405") + "-" + objectId[len(objectId)-6:]
}

func contextString(ctx context.Context, key string) string {
	value, ok := ctx.Value(key).(string)
	if !ok {
		return ""
	}
	return value
}

func roundMoney(value float64) float64 {
	return decimal.NewFromFloat(value).Round(4).InexactFloat64()
}

func roundRate(value float64) float64 {
	return decimal.NewFromFloat(value).Round(5).InexactFloat64()
}

func profitRateFromAmounts(totalCost float64, profitAmount float64) float64 {
	profitBase := decimal.NewFromFloat(totalCost).Add(decimal.NewFromFloat(profitAmount))
	if !profitBase.GreaterThan(decimal.Zero) {
		return 0
	}
	return roundRate(decimal.NewFromFloat(profitAmount).Div(profitBase).InexactFloat64())
}

func boolText(value bool) string {
	if value {
		return "是"
	}
	return "否"
}
