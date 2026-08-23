package ui

import (
	"testing"

	"zhengshi-wms-windowsapp/internal/api"
)

func TestInboundTraceRowsMarksSourceBatchAndAggregatesLocations(t *testing.T) {
	rows := inboundTraceRows([]api.InboundRecord{{
		Code: "B-002",
		Materials: []api.InboundRecordMaterial{
			{ActualQuantity: 2, WarehouseName: "主仓", WarehouseZoneName: "A区", WarehouseRackName: "A架", WarehouseBinName: "A01"},
			{ActualQuantity: 3, WarehouseName: "主仓", WarehouseZoneName: "A区", WarehouseRackName: "A架", WarehouseBinName: "A01"},
		},
	}}, "B-002")
	if len(rows) != 1 || rows[0].Source != "当前来源" || rows[0].Quantity != "5" || rows[0].Location != "主仓 / A区 / A架 / A01" {
		t.Fatalf("rows = %#v", rows)
	}
}
