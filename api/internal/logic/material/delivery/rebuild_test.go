package delivery

import (
	"context"
	"testing"

	"api/model"
	quoteCode "api/pkg/code"
)

func TestCustomerMaterialDeliveryKey(t *testing.T) {
	if got, want := customerMaterialDeliveryKey("customer-a", "material-1"), "customer-a\x00material-1"; got != want {
		t.Fatalf("customerMaterialDeliveryKey() = %q, want %q", got, want)
	}
}

func TestShouldDeleteStaleCustomerMaterialDelivery(t *testing.T) {
	tests := []struct {
		name   string
		record model.CustomerMaterialDelivery
		want   bool
	}{
		{
			name: "unquoted without quote summary can be deleted",
			record: model.CustomerMaterialDelivery{
				QuoteStatus: quoteCode.QuoteStatusUnquoted,
			},
			want: true,
		},
		{
			name: "empty quote status without quote summary can be deleted",
			record: model.CustomerMaterialDelivery{
				QuoteStatus: "",
			},
			want: true,
		},
		{
			name: "quoted delivery should be retained and marked invalid",
			record: model.CustomerMaterialDelivery{
				QuoteStatus:   quoteCode.QuoteStatusQuoted,
				LatestQuoteId: "quote-1",
			},
			want: false,
		},
		{
			name: "latest quote number should be retained and marked invalid",
			record: model.CustomerMaterialDelivery{
				QuoteStatus:   quoteCode.QuoteStatusUnquoted,
				LatestQuoteNo: "Q-001",
			},
			want: false,
		},
		{
			name: "priced delivery should be retained and marked invalid",
			record: model.CustomerMaterialDelivery{
				QuoteStatus: quoteCode.QuoteStatusPriced,
				LatestPrice: 12.3,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldDeleteStaleCustomerMaterialDelivery(tt.record); got != tt.want {
				t.Fatalf("shouldDeleteStaleCustomerMaterialDelivery() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFindQuotedDeliveryIdsEmptyInput(t *testing.T) {
	got, err := findQuotedDeliveryIds(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("findQuotedDeliveryIds() error = %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("len(findQuotedDeliveryIds()) = %d, want 0", len(got))
	}
}
