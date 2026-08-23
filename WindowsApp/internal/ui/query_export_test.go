package ui

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"
	"zhengshi-wms-windowsapp/internal/api"
)

func TestWriteQueryExportIncludesMetadataAndRows(t *testing.T) {
	target := filepath.Join(t.TempDir(), "inventory.xlsx")
	queriedAt := time.Date(2026, 8, 21, 10, 30, 0, 0, time.Local)
	items := []api.Inventory{{
		Type: "采购入库", ReceiptCode: "I-001", ReceiveCode: "R-001", WarehouseName: "主仓",
		WarehouseZoneName: "A区", WarehouseRackName: "A架", WarehouseBinName: "A01",
		Name: "测试物料", Model: "M-1", Unit: "件", Quantity: 10, AvailableQuantity: 8, LockedQuantity: 2,
	}}
	if err := writeQueryExportXLSX(
		target, "当前库存查询结果", "仓库=主仓", "现有库存查询接口",
		inventoryExportColumns(), inventoryExportRows(items), queriedAt,
	); err != nil {
		t.Fatal(err)
	}
	book, err := excelize.OpenFile(target)
	if err != nil {
		t.Fatal(err)
	}
	defer book.Close()
	for cell, want := range map[string]string{
		"A1": "当前库存查询结果", "A2": "查询时间：2026-08-21 10:30:00", "A6": "类型", "B7": "I-001", "H7": "测试物料",
	} {
		got, err := book.GetCellValue("查询结果", cell)
		if err != nil || got != want {
			t.Fatalf("%s=%q err=%v want=%q", cell, got, err, want)
		}
	}
	source, _ := book.GetCellValue("查询结果", "A4")
	if !strings.Contains(source, "未修改线上数据") {
		t.Fatalf("source note = %q", source)
	}
}

func TestWriteAtomicTargetDoesNotReplaceFileAfterCancellation(t *testing.T) {
	target := filepath.Join(t.TempDir(), "existing.xlsx")
	if err := os.WriteFile(target, []byte("existing"), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	err := writeAtomicTarget(ctx, target, func(temporary string) error {
		if err := os.WriteFile(temporary, []byte("partial"), 0o600); err != nil {
			return err
		}
		cancel()
		return nil
	})
	if err == nil {
		t.Fatal("cancelled atomic write unexpectedly succeeded")
	}
	data, readErr := os.ReadFile(target)
	if readErr != nil || string(data) != "existing" {
		t.Fatalf("target=%q err=%v", data, readErr)
	}
	matches, globErr := filepath.Glob(filepath.Join(filepath.Dir(target), ".zhengshi-wms-*.partial"))
	if globErr != nil || len(matches) != 0 {
		t.Fatalf("partial files=%v err=%v", matches, globErr)
	}
}

func TestQueryExportRowsPreserveServerValues(t *testing.T) {
	inboundRows := inboundExportRows([]api.InboundReceipt{{
		Code: "I-1", Materials: []api.InboundMaterial{{EstimatedQuantity: 3, ActualQuantity: 2}}, Annex: []string{"a.png"},
	}})
	if len(inboundRows) != 1 || inboundRows[0][0] != "I-1" || inboundRows[0][7] != float64(3) || inboundRows[0][8] != float64(2) {
		t.Fatalf("inbound rows = %#v", inboundRows)
	}
	outboundRows := outboundExportRows([]api.OutboundOrder{{Code: "O-1", IsPack: 1, IsWeigh: 0}})
	if len(outboundRows) != 1 || outboundRows[0][3] != "已打包 / 未称重" {
		t.Fatalf("outbound rows = %#v", outboundRows)
	}
}
