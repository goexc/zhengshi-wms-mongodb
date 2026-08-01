package ui

import (
	"strings"
	"testing"

	"zhengshi-wms-windowsapp/internal/api"
)

func TestInboundBatchRowsPreserveBatchMetadataAndMaterialLocation(t *testing.T) {
	record := api.InboundRecord{
		Code: "WIN-1", CarrierName: "承运商", CarrierCost: 8.5, OtherCost: 1.5,
		TotalAmount: 30, Annex: []string{"a.png", "b.png"}, CreatorName: "张三",
		Materials: []api.InboundRecordMaterial{{
			Index: 2, Name: "轴承", Model: "6204", Unit: "件", ActualQuantity: 3, Status: "入库完成",
			WarehouseName: "一号仓", WarehouseZoneName: "原料区", WarehouseRackName: "A架", WarehouseBinName: "A01",
		}},
	}
	batches := inboundBatchRows([]api.InboundRecord{record})
	if len(batches) != 1 || batches[0].Carrier != "承运商" || batches[0].CarrierCost != "8.50" || batches[0].Attachments != "2" {
		t.Fatalf("batches = %#v", batches)
	}
	materials := inboundBatchMaterialRows(record)
	if len(materials) != 1 || materials[0].Index != "3" || materials[0].Quantity != "3 件" || materials[0].Location != "一号仓 / 原料区 / A架 / A01" {
		t.Fatalf("materials = %#v", materials)
	}
}

func TestMaterialPriceRowsSortAndExposeSourceValidity(t *testing.T) {
	rows := materialPriceRows([]api.MaterialPrice{
		{Price: 10, Since: 100, CustomerName: "甲", SourceType: "manual", SourceValid: true},
		{Price: 12, Since: 200, CustomerName: "乙", SourceType: "material_quote", SourceValid: false, SourceInvalidReason: "报价已作废"},
	})
	if len(rows) != 2 || rows[0].Price != "12.000" || rows[0].Source != "物料报价" || rows[0].Valid != "无效" || rows[0].Reason != "报价已作废" {
		t.Fatalf("rows = %#v", rows)
	}
	if rows[1].Source != "人工维护" || rows[1].Valid != "有效" {
		t.Fatalf("rows = %#v", rows)
	}
}

func TestCustomerTransactionRowsNormalizeLabelsAndAttachments(t *testing.T) {
	rows := customerTransactionRows([]api.CustomerTransaction{{
		TransactionType: "payment", Direction: "receivable_decrease", SourceCode: "CK-1",
		Amount: 12.5, Annex: " a.png, ,b.png ",
	}})
	if len(rows) != 1 || rows[0].Type != "回款" || rows[0].Direction != "应收减少" || rows[0].Attachments != "2" {
		t.Fatalf("rows = %#v", rows)
	}
	attachments := customerTransactionAttachments(" a.png, ,b.png ")
	if len(attachments) != 2 || attachments[0] != "a.png" || attachments[1] != "b.png" {
		t.Fatalf("attachments = %#v", attachments)
	}
}

func TestBuildOutboundReviseRequestIncludesEveryMaterialAndChangedSummary(t *testing.T) {
	order := api.OutboundOrder{Code: "CK-1", CustomerID: "customer", CustomerName: "客户"}
	rows := buildOutboundReviseRows([]api.OutboundMaterial{
		{MaterialID: "m1", Name: "轴承", Quantity: 2, Unit: "件", Price: 3.125},
		{MaterialID: "m2", Name: "螺母", Quantity: 1, Unit: "件", Price: 0.2},
	})
	rows[0].NewValue = 3.5
	refreshOutboundRevisionRow(&rows[0])
	request, summary, err := buildOutboundReviseRequest(order, rows)
	if err != nil {
		t.Fatal(err)
	}
	if len(request.MaterialsPrice) != 2 || request.MaterialsPrice[0].Price != 3.5 || request.MaterialsPrice[1].Price != 0.2 {
		t.Fatalf("request = %#v", request)
	}
	if !strings.Contains(summary, "轴承：3.125 → 3.500") || !strings.Contains(summary, "变更项：1") {
		t.Fatalf("summary = %q", summary)
	}
}

func TestOutboundReviseSnapshotAndVerificationDetectConflicts(t *testing.T) {
	order := api.OutboundOrder{Code: "CK-1", CustomerID: "customer", Status: "已出库"}
	baseline := []api.OutboundMaterial{{MaterialID: "m1", Name: "轴承", Quantity: 2, Price: 3}}
	if err := validateOutboundRevisionSnapshot(order, order, baseline, baseline); err != nil {
		t.Fatal(err)
	}
	changed := append([]api.OutboundMaterial(nil), baseline...)
	changed[0].Price = 4
	if err := validateOutboundRevisionSnapshot(order, order, baseline, changed); err == nil {
		t.Fatal("expected concurrent price conflict")
	}
	request := api.OutboundReviseRequest{
		Code: "CK-1", CustomerID: "customer",
		MaterialsPrice: []api.OutboundMaterialPrice{{MaterialID: "m1", Price: 4}},
	}
	verifiedOrder := api.OutboundOrder{CarrierCost: 1, OtherCost: 2, TotalAmount: 11}
	if err := verifyOutboundRevision(verifiedOrder, changed, request); err != nil {
		t.Fatal(err)
	}
	changed[0].Price = 5
	if err := verifyOutboundRevision(verifiedOrder, changed, request); err == nil {
		t.Fatal("expected verification mismatch")
	}
	changed[0].Price = 4
	verifiedOrder.TotalAmount = 12
	if err := verifyOutboundRevision(verifiedOrder, changed, request); err == nil {
		t.Fatal("expected total amount verification mismatch")
	}
}

func TestParseOutboundRevisionPriceRejectsInvalidValues(t *testing.T) {
	if value, err := parseOutboundRevisionPrice("12.345"); err != nil || value != 12.345 {
		t.Fatalf("value=%v err=%v", value, err)
	}
	for _, value := range []string{"", "-1", "NaN", "Inf"} {
		if _, err := parseOutboundRevisionPrice(value); err == nil {
			t.Fatalf("expected error for %q", value)
		}
	}
}
