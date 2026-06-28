package material

import (
	"bytes"
	"encoding/csv"
	"testing"

	"api/model"
	quoteCode "api/pkg/code"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestBuildNewDeliveryQuoteCSVFormatsCostItems(t *testing.T) {
	deliveryId := primitive.NewObjectID()
	quoteId := primitive.NewObjectID()
	body, err := buildNewDeliveryQuoteCSV(
		[]model.CustomerMaterialDelivery{
			{
				Id:                     deliveryId,
				CustomerName:           "测试客户",
				MaterialName:           "测试物料",
				MaterialUnit:           "pcs",
				FirstDeliveryQuantity:  12.3456,
				FirstDeliveryPrice:     8.9,
				QuoteStatus:            quoteCode.QuoteStatusQuoted,
				LatestPrice:            3.1724,
				FirstDeliveryOrderCode: "OUT-001",
			},
		},
		map[string]model.MaterialQuote{
			deliveryId.Hex(): {
				Id:           quoteId,
				QuoteNo:      "Q-001",
				QuoteMode:    quoteCode.QuoteModeDetailed,
				Status:       quoteCode.MaterialQuoteStatusQuoted,
				Currency:     "CNY",
				TotalCost:    2.7674,
				ProfitRate:   0.2,
				ProfitAmount: 0.3456,
				TaxRate:      0.13,
				TaxAmount:    0.4044,
				FinalPrice:   3.1724,
				CostItems: []model.MaterialQuoteCostItem{
					{Index: 2, CategoryCode: "material", Name: "辅料", Enabled: true, Amount: 1.2344},
					{Index: 1, CategoryCode: "material", Name: "成本1", Enabled: true, Amount: 0.2},
					{Index: 1, CategoryCode: "quality", Name: "质检", Enabled: true, Amount: 0.3333},
					{Index: 1, CategoryCode: "packing", Name: "打包", Enabled: true, Amount: 0.4444},
					{Index: 1, CategoryCode: "unexpected", Name: "其他项", Enabled: true, Amount: 0.5554},
					{Index: 1, CategoryCode: "management", Name: "不应导出", Enabled: false, Amount: 99},
				},
			},
		},
	)
	if err != nil {
		t.Fatalf("build csv: %v", err)
	}

	rows := readQuoteCSV(t, body)
	if len(rows) != 2 {
		t.Fatalf("expected header and one data row, got %d rows", len(rows))
	}
	index := headerIndex(rows[0])
	cell := func(name string) string {
		pos, ok := index[name]
		if !ok {
			t.Fatalf("missing header %s", name)
		}
		return rows[1][pos]
	}

	if got, want := cell("材料成本"), "成本1(0.200元)\n辅料(1.234元)"; got != want {
		t.Fatalf("材料成本 = %q, want %q", got, want)
	}
	if got, want := cell("质量成本"), "质检(0.333元)"; got != want {
		t.Fatalf("质量成本 = %q, want %q", got, want)
	}
	if got, want := cell("包装/物流成本"), "打包(0.444元)"; got != want {
		t.Fatalf("包装/物流成本 = %q, want %q", got, want)
	}
	if got, want := cell("其他成本"), "其他项(0.555元)"; got != want {
		t.Fatalf("其他成本 = %q, want %q", got, want)
	}
	if got := cell("管理成本"); got != "" {
		t.Fatalf("disabled cost should not be exported, got %q", got)
	}

	expectedNumbers := map[string]string{
		"首次交付数量": "12.346",
		"首次交付单价": "8.900",
		"成本合计":   "2.767",
		"利润率":    "11.102%",
		"利润金额":   "0.346",
		"税率":     "13.000%",
		"税额":     "0.404",
		"报价单价":   "3.172",
	}
	for name, want := range expectedNumbers {
		if got := cell(name); got != want {
			t.Fatalf("%s = %q, want %q", name, got, want)
		}
	}
	if got := cell("最终定价"); got != "" {
		t.Fatalf("未定价记录的最终定价应为空, got %q", got)
	}
}

func TestFormatExportFinalPriceRequiresPricedStatus(t *testing.T) {
	quotedDelivery := model.CustomerMaterialDelivery{
		QuoteStatus: quoteCode.QuoteStatusQuoted,
		LatestPrice: 3.1724,
	}
	quotedQuote := model.MaterialQuote{
		Status:     quoteCode.MaterialQuoteStatusQuoted,
		FinalPrice: 3.1724,
	}
	if got := formatExportFinalPrice(quotedDelivery, quotedQuote); got != "" {
		t.Fatalf("quoted final price = %q, want empty", got)
	}

	pricedQuote := model.MaterialQuote{
		Status:     quoteCode.MaterialQuoteStatusPriced,
		FinalPrice: 3.1724,
	}
	if got, want := formatExportFinalPrice(quotedDelivery, pricedQuote), "3.172"; got != want {
		t.Fatalf("priced quote final price = %q, want %q", got, want)
	}

	pricedDelivery := model.CustomerMaterialDelivery{
		QuoteStatus: quoteCode.QuoteStatusPriced,
		LatestPrice: 8.7654,
	}
	if got, want := formatExportFinalPrice(pricedDelivery, model.MaterialQuote{}), "8.765"; got != want {
		t.Fatalf("priced delivery final price = %q, want %q", got, want)
	}
}

func readQuoteCSV(t *testing.T, body []byte) [][]string {
	t.Helper()
	body = bytes.TrimPrefix(body, []byte{0xEF, 0xBB, 0xBF})
	reader := csv.NewReader(bytes.NewReader(body))
	reader.FieldsPerRecord = -1
	rows, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("read csv: %v", err)
	}
	return rows
}

func headerIndex(header []string) map[string]int {
	index := make(map[string]int, len(header))
	for i, name := range header {
		index[name] = i
	}
	return index
}
