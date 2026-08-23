package ui

import (
	"testing"

	"zhengshi-wms-windowsapp/internal/api"
)

func TestOutboundCreateTypesExcludeBrokenReturnContract(t *testing.T) {
	if stringIndex(outboundCreateTypes, "退货出库") >= 0 {
		t.Fatal("return outbound must stay hidden until the existing supplier/customer contract is consistent")
	}
	if !outboundTypeRequiresCustomer("销售出库") || outboundTypeRequiresCustomer("报废出库") {
		t.Fatal("customer requirement does not match outbound add logic")
	}
}

func TestLatestValidMaterialPriceIgnoresInvalidSources(t *testing.T) {
	price, ok := latestValidMaterialPrice([]api.MaterialPrice{
		{Price: 20, Since: 300, SourceValid: false},
		{Price: 10, Since: 100, SourceValid: true},
		{Price: 12, Since: 200, SourceValid: true},
	})
	if !ok || price != 12 {
		t.Fatalf("price=%v ok=%v", price, ok)
	}
}

func TestCanDeleteOutboundOnlyAllowsPreShipment(t *testing.T) {
	if !canDeleteOutbound(api.OutboundOrder{ID: "order", Status: "预发货"}) {
		t.Fatal("pre-shipment order should be deletable")
	}
	for _, status := range []string{"", "待拣货", "已出库", "已签收"} {
		if canDeleteOutbound(api.OutboundOrder{ID: "order", Status: status}) {
			t.Fatalf("status %q must not be deletable", status)
		}
	}
	if canDeleteOutbound(api.OutboundOrder{Status: "预发货"}) {
		t.Fatal("missing id must not be deletable")
	}
}
