package ui

import (
	"reflect"
	"testing"

	"zhengshi-wms-windowsapp/internal/api"
)

func TestFlattenPositionsPreservesWarehousePath(t *testing.T) {
	tree := []api.WarehouseNode{{
		ID: "w", Name: "一号仓",
		Children: []api.WarehouseNode{{
			ID: "z", Name: "A区",
			Children: []api.WarehouseNode{{ID: "b", Name: "A-01"}},
		}},
	}}
	got := flattenPositions(tree)
	if len(got) != 1 {
		t.Fatalf("positions = %d", len(got))
	}
	if got[0].Label != "一号仓 / A区 / A-01" {
		t.Fatalf("label = %q", got[0].Label)
	}
	if !reflect.DeepEqual(got[0].IDs, []string{"w", "z", "b"}) {
		t.Fatalf("ids = %#v", got[0].IDs)
	}
}

func TestHasButtonUsesExactPermission(t *testing.T) {
	buttons := []api.Button{{Name: "批次入库", Perms: "inbound:receipt:receive"}}
	if !hasButton(buttons, "inbound:receipt:receive") {
		t.Fatal("expected receive permission")
	}
	if hasButton(buttons, "inbound:receipt:delete") {
		t.Fatal("unexpected delete permission")
	}
}

func TestParseNonNegativeNumberAllowsExistingOptionalCostContract(t *testing.T) {
	if value, err := parseNonNegativeNumber("", "运费"); err != nil || value != 0 {
		t.Fatalf("empty value = %v, %v", value, err)
	}
	if value, err := parseNonNegativeNumber("12.5", "运费"); err != nil || value != 12.5 {
		t.Fatalf("value = %v, %v", value, err)
	}
	if _, err := parseNonNegativeNumber("-1", "运费"); err == nil {
		t.Fatal("expected non-negative validation error")
	}
}

func TestInboundLifecycleStatusMatrixMatchesServerRules(t *testing.T) {
	for _, status := range []string{"待审核", "审核不通过"} {
		if !canEditInbound(status) || !canDeleteInbound(status) {
			t.Fatalf("status %q should allow edit and delete", status)
		}
	}
	if !canCheckInbound("待审核") || canCheckInbound("审核不通过") {
		t.Fatal("check status matrix mismatch")
	}
	for _, status := range []string{"待审核", "审核不通过", "作废", "入库完成"} {
		if canReceiveInbound(status) {
			t.Fatalf("status %q must not allow receiving", status)
		}
	}
	for _, status := range []string{"审核通过", "未发货", "在途", "部分入库"} {
		if !canReceiveInbound(status) {
			t.Fatalf("status %q should allow receiving", status)
		}
	}
}

func TestParsePositiveInboundNumber(t *testing.T) {
	if got, err := parsePositiveInboundNumber("0", "单价", true); err != nil || got != 0 {
		t.Fatalf("price = %v, %v", got, err)
	}
	if got, err := parsePositiveInboundNumber("1.25", "数量", false); err != nil || got != 1.25 {
		t.Fatalf("quantity = %v, %v", got, err)
	}
	if _, err := parsePositiveInboundNumber("0", "数量", false); err == nil {
		t.Fatal("zero quantity must fail")
	}
}
