package quote

import (
	"testing"

	"api/internal/types"
	"api/model"
	quoteCode "api/pkg/code"
)

func TestBuildQuoteCalculationUsesProfitMarginRate(t *testing.T) {
	calculation, msg := buildQuoteCalculation(&types.MaterialQuoteSaveRequest{
		QuoteMode:    quoteCode.QuoteModeDetailed,
		ProfitAmount: 20,
		TaxRate:      0.13,
		CostItems: []types.MaterialQuoteCostItem{
			{
				Index:        1,
				CategoryCode: "material",
				CategoryName: "材料成本",
				Name:         "材料",
				Enabled:      true,
				Amount:       100,
			},
		},
	})
	if msg != "" {
		t.Fatalf("buildQuoteCalculation returned message: %s", msg)
	}
	if calculation.profitRate != 0.16667 {
		t.Fatalf("profitRate = %v, want 0.16667", calculation.profitRate)
	}
	if calculation.totalCost != 100 {
		t.Fatalf("totalCost = %v, want 100", calculation.totalCost)
	}
	if calculation.profitAmount != 20 {
		t.Fatalf("profitAmount = %v, want 20", calculation.profitAmount)
	}
	if calculation.taxAmount != 15.6 {
		t.Fatalf("taxAmount = %v, want 15.6", calculation.taxAmount)
	}
	if calculation.finalPrice != 135.6 {
		t.Fatalf("finalPrice = %v, want 135.6", calculation.finalPrice)
	}
}

func TestToTypeQuoteRecomputesProfitRate(t *testing.T) {
	quote := toTypeQuote(model.MaterialQuote{
		TotalCost:    100,
		ProfitAmount: 20,
		ProfitRate:   0.2,
	})
	if quote.ProfitRate != 0.16667 {
		t.Fatalf("ProfitRate = %v, want 0.16667", quote.ProfitRate)
	}
}
