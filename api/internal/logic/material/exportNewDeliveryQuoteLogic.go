// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package material

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"api/internal/svc"
	"api/internal/types"
	"api/model"
	quoteCode "api/pkg/code"

	"github.com/zeromicro/go-zero/core/logx"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ExportNewDeliveryQuoteLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 导出客户新增物料报价
func NewExportNewDeliveryQuoteLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ExportNewDeliveryQuoteLogic {
	return &ExportNewDeliveryQuoteLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ExportNewDeliveryQuoteLogic) ExportNewDeliveryQuote(req *types.NewCustomerMaterialExportRequest) (fileName string, body []byte, resp *types.BaseResponse, err error) {
	resp = new(types.BaseResponse)
	if req.EndTime < req.StartTime {
		resp.Code = http.StatusBadRequest
		resp.Msg = "结束时间不能早于开始时间"
		return "", nil, resp, nil
	}

	deliveries, err := l.findExportDeliveries(req)
	if err != nil {
		fmt.Printf("[Error]查询客户新增物料报价导出数据失败:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "查询客户新增物料报价导出数据失败"
		return "", nil, resp, nil
	}

	quoteMap, err := l.findLatestQuotes(deliveries)
	if err != nil {
		fmt.Printf("[Error]查询客户新增物料最新报价失败:%s\n", err.Error())
		resp.Code = http.StatusInternalServerError
		resp.Msg = "查询客户新增物料最新报价失败"
		return "", nil, resp, nil
	}

	body, err = buildNewDeliveryQuoteCSV(deliveries, quoteMap)
	if err != nil {
		return "", nil, nil, err
	}

	resp.Code = http.StatusOK
	resp.Msg = "成功"
	return l.newDeliveryQuoteExportFileName(req, deliveries), body, resp, nil
}

func (l *ExportNewDeliveryQuoteLogic) newDeliveryQuoteExportFileName(req *types.NewCustomerMaterialExportRequest, deliveries []model.CustomerMaterialDelivery) string {
	customerName := ""
	for _, delivery := range deliveries {
		if strings.TrimSpace(delivery.CustomerName) != "" {
			customerName = delivery.CustomerName
			break
		}
	}
	if strings.TrimSpace(customerName) == "" {
		if customerId, err := primitive.ObjectIDFromHex(req.CustomerId); err == nil {
			var customer model.Customer
			if err = l.svcCtx.CustomerModel.FindOne(l.ctx, bson.M{"_id": customerId}).Decode(&customer); err == nil {
				customerName = customer.Name
			}
		}
	}
	customerName = sanitizeExportFileName(customerName)
	if customerName == "" {
		customerName = "客户"
	}
	dateRange := fmt.Sprintf("%s至%s", formatExportDate(req.StartTime), formatExportDate(req.EndTime))
	return fmt.Sprintf("%s-新增物料报价-%s.csv", customerName, dateRange)
}

func sanitizeExportFileName(name string) string {
	name = strings.NewReplacer(
		"\\", "_",
		"/", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
	).Replace(strings.TrimSpace(name))
	name = strings.Map(func(r rune) rune {
		if r < 32 {
			return -1
		}
		return r
	}, name)
	return strings.TrimSpace(name)
}

func (l *ExportNewDeliveryQuoteLogic) findExportDeliveries(req *types.NewCustomerMaterialExportRequest) ([]model.CustomerMaterialDelivery, error) {
	filter := bson.M{
		"customer_id": req.CustomerId,
		"first_delivery_time": bson.M{
			"$gte": req.StartTime,
			"$lte": req.EndTime,
		},
	}
	if strings.TrimSpace(req.QuoteStatus) != "" {
		filter["quote_status"] = strings.TrimSpace(req.QuoteStatus)
	}
	if strings.TrimSpace(req.MaterialName) != "" {
		filter["material_name"] = primitive.Regex{Pattern: strings.TrimSpace(req.MaterialName), Options: "i"}
	}
	if strings.TrimSpace(req.MaterialModel) != "" {
		filter["material_model"] = primitive.Regex{Pattern: strings.TrimSpace(req.MaterialModel), Options: "i"}
	}

	cur, err := l.svcCtx.CustomerMaterialDeliveryModel.Find(
		l.ctx,
		filter,
		options.Find().SetSort(bson.D{{Key: "first_delivery_time", Value: -1}, {Key: "created_at", Value: -1}}),
	)
	if err != nil {
		return nil, err
	}
	defer cur.Close(l.ctx)

	var deliveries []model.CustomerMaterialDelivery
	if err = cur.All(l.ctx, &deliveries); err != nil {
		return nil, err
	}
	return deliveries, nil
}

func (l *ExportNewDeliveryQuoteLogic) findLatestQuotes(deliveries []model.CustomerMaterialDelivery) (map[string]model.MaterialQuote, error) {
	ids := make([]primitive.ObjectID, 0, len(deliveries))
	quoteIdToDeliveryId := make(map[string]string, len(deliveries))
	for _, delivery := range deliveries {
		if strings.TrimSpace(delivery.LatestQuoteId) == "" {
			continue
		}
		quoteId, err := primitive.ObjectIDFromHex(delivery.LatestQuoteId)
		if err != nil {
			continue
		}
		ids = append(ids, quoteId)
		quoteIdToDeliveryId[quoteId.Hex()] = delivery.Id.Hex()
	}
	if len(ids) == 0 {
		return map[string]model.MaterialQuote{}, nil
	}

	cur, err := l.svcCtx.MaterialQuoteModel.Find(l.ctx, bson.M{"_id": bson.M{"$in": ids}})
	if err != nil {
		return nil, err
	}
	defer cur.Close(l.ctx)

	quotes := make(map[string]model.MaterialQuote, len(ids))
	for cur.Next(l.ctx) {
		var quote model.MaterialQuote
		if err = cur.Decode(&quote); err != nil {
			return nil, err
		}
		deliveryId := quoteIdToDeliveryId[quote.Id.Hex()]
		if deliveryId != "" {
			quotes[deliveryId] = quote
		}
	}
	if err = cur.Err(); err != nil {
		return nil, err
	}
	return quotes, nil
}

func buildNewDeliveryQuoteCSV(deliveries []model.CustomerMaterialDelivery, quoteMap map[string]model.MaterialQuote) ([]byte, error) {
	var buf bytes.Buffer
	buf.Write([]byte{0xEF, 0xBB, 0xBF})
	writer := csv.NewWriter(&buf)
	header := []string{
		"客户",
		"物料名称",
		"物料型号",
		"物料规格",
		"单位",
		"首次交付时间",
		"首次交付出库单",
		"首次交付数量",
		"首次交付单价",
		"报价状态",
		"最新报价单号",
		"报价方式",
		"报价单状态",
		"币种",
	}
	for _, category := range quoteCostExportCategories {
		header = append(header, category.name)
	}
	header = append(header,
		"成本合计",
		"利润率",
		"利润金额",
		"税率",
		"税额",
		"报价单价",
		"最终定价",
		"有效期开始",
		"有效期结束",
		"报价备注",
	)
	if err := writer.Write(header); err != nil {
		return nil, err
	}

	for _, delivery := range deliveries {
		quote := quoteMap[delivery.Id.Hex()]
		costs := quoteCostCategoryCells(quote)
		row := []string{
			delivery.CustomerName,
			delivery.MaterialName,
			delivery.MaterialModel,
			delivery.MaterialSpecification,
			delivery.MaterialUnit,
			formatExportTime(delivery.FirstDeliveryTime),
			delivery.FirstDeliveryOrderCode,
			formatExportNumber(delivery.FirstDeliveryQuantity),
			formatExportNumber(delivery.FirstDeliveryPrice),
			deliveryQuoteStatusLabel(delivery.QuoteStatus),
			quote.QuoteNo,
			quoteModeLabel(quote.QuoteMode),
			materialQuoteStatusLabel(quote.Status),
			quote.Currency,
		}
		for _, category := range quoteCostExportCategories {
			row = append(row, costs[category.code])
		}
		row = append(row,
			formatExportNumber(quote.TotalCost),
			formatExportRate(exportQuoteProfitRate(quote)),
			formatExportNumber(quote.ProfitAmount),
			formatExportRate(quote.TaxRate),
			formatExportNumber(quote.TaxAmount),
			formatExportNumber(exportQuoteUnitPrice(quote)),
			formatExportFinalPrice(delivery, quote),
			formatExportTime(quote.ValidFrom),
			formatExportTime(quote.ValidTo),
			quote.Remark,
		)
		if err := writer.Write(row); err != nil {
			return nil, err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type quoteCostExportCategory struct {
	code string
	name string
}

var quoteCostExportCategories = []quoteCostExportCategory{
	{code: "material", name: "材料成本"},
	{code: "process", name: "加工工序成本"},
	{code: "labor_equipment", name: "人工/设备成本"},
	{code: "quality", name: "质量成本"},
	{code: "packing_logistics", name: "包装/物流成本"},
	{code: "management", name: "管理成本"},
	{code: "tooling", name: "模具/治具摊销"},
	{code: "loss", name: "损耗成本"},
	{code: "other", name: "其他成本"},
}

func quoteCostCategoryCells(quote model.MaterialQuote) map[string]string {
	costs := make(map[string][]string, len(quoteCostExportCategories))
	for _, category := range quoteCostExportCategories {
		costs[category.code] = nil
	}

	items := append([]model.MaterialQuoteCostItem(nil), quote.CostItems...)
	sortQuoteCostItems(items)
	for _, item := range items {
		if !item.Enabled {
			continue
		}
		categoryCode := quoteCostExportCategoryCode(item.CategoryCode)
		name := strings.TrimSpace(item.Name)
		if name == "" {
			name = fmt.Sprintf("成本%d", len(costs[categoryCode])+1)
		}
		costs[categoryCode] = append(costs[categoryCode], fmt.Sprintf("%s(%s元)", name, formatExportAmount(item.Amount)))
	}

	cells := make(map[string]string, len(costs))
	for categoryCode, lines := range costs {
		if len(lines) > 0 {
			cells[categoryCode] = strings.Join(lines, "\n")
		}
	}
	return cells
}

func sortQuoteCostItems(items []model.MaterialQuoteCostItem) {
	sort.SliceStable(items, func(i, j int) bool {
		left, right := quoteCostExportCategoryRank(items[i].CategoryCode), quoteCostExportCategoryRank(items[j].CategoryCode)
		if left != right {
			return left < right
		}
		if items[i].Index != items[j].Index {
			return items[i].Index < items[j].Index
		}
		return items[i].Name < items[j].Name
	})
}

func quoteCostExportCategoryCode(code string) string {
	switch strings.TrimSpace(code) {
	case "material", "process", "labor_equipment", "quality", "management", "tooling", "loss", "other":
		return strings.TrimSpace(code)
	case "packing", "freight", "packing_logistics":
		return "packing_logistics"
	default:
		return "other"
	}
}

func quoteCostExportCategoryRank(code string) int {
	switch quoteCostExportCategoryCode(code) {
	case "material":
		return 10
	case "process":
		return 20
	case "labor_equipment":
		return 30
	case "quality":
		return 40
	case "packing_logistics":
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

func exportQuoteUnitPrice(quote model.MaterialQuote) float64 {
	if quote.FinalPrice > 0 {
		return quote.FinalPrice
	}
	return quote.SimplePrice
}

func exportQuoteProfitRate(quote model.MaterialQuote) float64 {
	profitBase := quote.TotalCost + quote.ProfitAmount
	if profitBase <= 0 {
		return 0
	}
	return quote.ProfitAmount / profitBase
}

func formatExportFinalPrice(delivery model.CustomerMaterialDelivery, quote model.MaterialQuote) string {
	if quote.Status == quoteCode.MaterialQuoteStatusPriced {
		return formatExportNumber(quote.FinalPrice)
	}
	if delivery.QuoteStatus == quoteCode.QuoteStatusPriced {
		return formatExportNumber(delivery.LatestPrice)
	}
	return ""
}

func formatExportNumber(value float64) string {
	if value == 0 {
		return ""
	}
	return formatExportAmount(value)
}

func formatExportRate(value float64) string {
	if value == 0 {
		return ""
	}
	return fmt.Sprintf("%.3f%%", value*100)
}

func formatExportAmount(value float64) string {
	return fmt.Sprintf("%.3f", value)
}

func formatExportTime(value int64) string {
	if value <= 0 {
		return ""
	}
	return time.Unix(value, 0).Format("2006-01-02 15:04:05")
}

func formatExportDate(value int64) string {
	if value <= 0 {
		return ""
	}
	return time.Unix(value, 0).Format("20060102")
}

func deliveryQuoteStatusLabel(status string) string {
	switch status {
	case quoteCode.QuoteStatusUnquoted:
		return "未报价"
	case quoteCode.QuoteStatusQuoting:
		return "报价中"
	case quoteCode.QuoteStatusQuoted:
		return "已报价"
	case quoteCode.QuoteStatusPriced:
		return "已定价"
	default:
		return status
	}
}

func quoteModeLabel(mode string) string {
	switch mode {
	case quoteCode.QuoteModeDetailed:
		return "详细报价"
	case quoteCode.QuoteModeSimple:
		return "简单报价"
	default:
		return mode
	}
}

func materialQuoteStatusLabel(status string) string {
	switch status {
	case quoteCode.MaterialQuoteStatusDraft:
		return "草稿"
	case quoteCode.MaterialQuoteStatusSubmitted:
		return "已提交"
	case quoteCode.MaterialQuoteStatusQuoted:
		return "已报价"
	case quoteCode.MaterialQuoteStatusPriced:
		return "已定价"
	case quoteCode.MaterialQuoteStatusVoid:
		return "已作废"
	default:
		return status
	}
}
