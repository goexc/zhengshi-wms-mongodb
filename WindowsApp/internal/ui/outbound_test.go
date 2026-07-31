package ui

import (
	"testing"
	"time"

	"zhengshi-wms-windowsapp/internal/api"
)

func TestOutboundVirtualStageMappingsMatchExistingAPI(t *testing.T) {
	checks := map[string]outboundStage{
		"待打包": {Label: "待打包", Status: "待打包", IsPack: 0, IsWeigh: -1},
		"待称重": {Label: "待称重", Status: "待称重", IsPack: -1, IsWeigh: 0},
		"待出库": {Label: "待出库", Status: "待出库", IsPack: -1, IsWeigh: -1},
	}
	for _, stage := range outboundStages {
		want, ok := checks[stage.Label]
		if !ok {
			continue
		}
		if stage != want {
			t.Fatalf("stage %q = %#v, want %#v", stage.Label, stage, want)
		}
		delete(checks, stage.Label)
	}
	if len(checks) != 0 {
		t.Fatalf("missing stages = %#v", checks)
	}
}

func TestOutboundActionAvailabilityUsesServerStateFields(t *testing.T) {
	if !canConfirmOutbound(api.OutboundOrder{Status: "预发货"}) {
		t.Fatal("预发货 should allow confirm")
	}
	if !canPickOutbound(api.OutboundOrder{Status: "待拣货"}) {
		t.Fatal("待拣货 should allow pick")
	}
	if !canPackOutbound(api.OutboundOrder{Status: "已称重", IsPack: 0}) {
		t.Fatal("unpacked weighed order should allow pack")
	}
	if canPackOutbound(api.OutboundOrder{Status: "已称重", IsPack: 1}) {
		t.Fatal("packed weighed order should not allow pack")
	}
	if !canWeighOutbound(api.OutboundOrder{Status: "已打包", IsWeigh: 0}) {
		t.Fatal("unweighed packed order should allow weigh")
	}
	if !canDepartOutbound(api.OutboundOrder{Status: "已拣货"}) {
		t.Fatal("picked order should allow departure")
	}
	if !canReceiptOutbound(api.OutboundOrder{Status: "已出库"}) {
		t.Fatal("departed order should allow receipt")
	}
}

func TestBuildOutboundConfirmRequestUsesExistingDTO(t *testing.T) {
	allocations := []outboundAllocationData{{
		Material: api.OutboundMaterial{
			MaterialID: "material", Index: 1, Name: "螺栓", Quantity: 10, Unit: "件",
		},
		Inventories: []api.Inventory{
			{ID: "inventory-a", ReceiveCode: "A", AvailableQuantity: 4},
			{ID: "inventory-b", ReceiveCode: "B", AvailableQuantity: 8},
		},
	}}
	request, _, err := buildOutboundConfirmRequest("CK-001", 123, allocations, [][]string{{"4", "3"}})
	if err != nil {
		t.Fatal(err)
	}
	if request.Code != "CK-001" || request.ConfirmTime != 123 || len(request.Materials) != 1 {
		t.Fatalf("request = %#v", request)
	}
	inventories := request.Materials[0].Inventorys
	if len(inventories) != 2 || inventories[0].ShipmentQuantity != 4 || inventories[1].ShipmentQuantity != 3 {
		t.Fatalf("inventories = %#v", inventories)
	}
}

func TestBuildOutboundConfirmRequestRejectsDisplayedAvailabilityOverflow(t *testing.T) {
	allocations := []outboundAllocationData{{
		Material: api.OutboundMaterial{MaterialID: "material", Index: 1, Name: "螺栓", Quantity: 10},
		Inventories: []api.Inventory{{
			ID: "inventory", ReceiveCode: "A", AvailableQuantity: 2,
		}},
	}}
	if _, _, err := buildOutboundConfirmRequest("CK-001", 123, allocations, [][]string{{"3"}}); err == nil {
		t.Fatal("expected availability validation error")
	}
}

func TestOutboundDateParsingUsesLocalDayBoundaries(t *testing.T) {
	start, err := parseOutboundFilterDate("2026-07-30", false)
	if err != nil {
		t.Fatal(err)
	}
	end, err := parseOutboundFilterDate("2026-07-30", true)
	if err != nil {
		t.Fatal(err)
	}
	if end-start != int64(24*time.Hour/time.Second)-1 {
		t.Fatalf("range = %d", end-start)
	}
	if _, err := parseOperationTime("2026-07-30 12:34:56"); err != nil {
		t.Fatal(err)
	}
}
