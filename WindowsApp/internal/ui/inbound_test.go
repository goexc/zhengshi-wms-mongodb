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
