package ui

import (
	"errors"
	"math"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/lxn/walk"
	"github.com/xuri/excelize/v2"

	"zhengshi-wms-windowsapp/internal/api"
)

type attachmentPreviewSetterStub struct {
	events *[]string
	err    error
}

func (stub *attachmentPreviewSetterStub) SetImage(image walk.Image) error {
	if image != nil {
		panic("clearAttachmentPreview must detach the image with nil")
	}
	*stub.events = append(*stub.events, "clear")
	return stub.err
}

type attachmentPreviewResourceStub struct {
	events   *[]string
	disposed bool
}

func (stub *attachmentPreviewResourceStub) Dispose() {
	stub.disposed = true
	*stub.events = append(*stub.events, "dispose")
}

func TestClearAttachmentPreviewDetachesBeforeDisposingBitmap(t *testing.T) {
	events := make([]string, 0, 2)
	setter := &attachmentPreviewSetterStub{events: &events}
	resource := &attachmentPreviewResourceStub{events: &events}
	if err := clearAttachmentPreview(setter, resource); err != nil {
		t.Fatal(err)
	}
	if !resource.disposed {
		t.Fatal("expected current bitmap to be disposed")
	}
	if want := []string{"clear", "dispose"}; !reflect.DeepEqual(events, want) {
		t.Fatalf("events = %#v, want %#v", events, want)
	}
}

func TestClearAttachmentPreviewDisposesBitmapWhenRefreshFails(t *testing.T) {
	events := make([]string, 0, 2)
	wantErr := errors.New("invalidate failed")
	setter := &attachmentPreviewSetterStub{events: &events, err: wantErr}
	resource := &attachmentPreviewResourceStub{events: &events}
	if err := clearAttachmentPreview(setter, resource); !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want %v", err, wantErr)
	}
	if !resource.disposed {
		t.Fatal("expected current bitmap to be disposed after detach attempt")
	}
	if want := []string{"clear", "dispose"}; !reflect.DeepEqual(events, want) {
		t.Fatalf("events = %#v, want %#v", events, want)
	}
}

func TestFlattenWarehouseTreeKeepsLevelsAndParentIDs(t *testing.T) {
	tree := []api.WarehouseNode{{
		ID: "warehouse", Name: "一号仓",
		Children: []api.WarehouseNode{{
			ID: "zone", Name: "原料区",
			Children: []api.WarehouseNode{{
				ID: "rack", Name: "A架",
				Children: []api.WarehouseNode{{ID: "bin", Name: "A01"}},
			}},
		}},
	}}

	entries := flattenWarehouseTree(tree)
	if len(entries) != 4 {
		t.Fatalf("entries = %#v", entries)
	}
	bin := entries[3]
	if bin.Level != 3 || bin.Path != "一号仓 / 原料区 / A架 / A01" {
		t.Fatalf("bin = %#v", bin)
	}
	if bin.WarehouseID != "warehouse" || bin.ZoneID != "zone" || bin.RackID != "rack" {
		t.Fatalf("bin parent ids = %#v", bin)
	}
}

func TestReportCalculationsAndGroupingMatchDisplayContract(t *testing.T) {
	rows := []api.OutboundSummaryRecord{
		{OrderCode: "CK-2", MaterialID: "m2", Model: "B", Quantity: 0.1, Price: 0.2, ReceiptDate: 200, Index: 2},
		{OrderCode: "CK-1", MaterialID: "m1", Model: "A", Quantity: 2, Price: 3.125, ReceiptDate: 100, Index: 1},
		{OrderCode: "CK-1", MaterialID: "m2", Model: "B", Quantity: 1, Price: 0.2, ReceiptDate: 100, Index: 2},
	}
	if got := reportRecordAmount(rows[0]); math.Abs(got-0.02) > 1e-12 {
		t.Fatalf("amount = %.15f", got)
	}
	if got := reportTotalQuantity(rows); math.Abs(got-3.1) > 1e-12 {
		t.Fatalf("quantity = %.15f", got)
	}
	if got := reportTotalAmount(rows); math.Abs(got-6.47) > 1e-12 {
		t.Fatalf("amount = %.15f", got)
	}
	if got := distinctOutboundOrderCount(rows); got != 2 {
		t.Fatalf("order count = %d", got)
	}

	orderGroups := groupOutboundReportRecords(rows, "order")
	if len(orderGroups) != 2 || reportOrderCode(orderGroups[0].rows[0]) != "CK-1" {
		t.Fatalf("order groups = %#v", orderGroups)
	}
	materialGroups := groupOutboundReportRecords(rows, "material")
	if len(materialGroups) != 2 || materialGroups[0].rows[0].MaterialID != "m1" {
		t.Fatalf("material groups = %#v", materialGroups)
	}
}

func TestParseRequiredReportDateUsesWebMidnightContract(t *testing.T) {
	got, err := parseRequiredReportDate("2026-07-31", "截止日期")
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 7, 31, 0, 0, 0, 0, time.Local).Unix()
	if got != want {
		t.Fatalf("date = %d, want %d", got, want)
	}
	if _, err := parseRequiredReportDate("", "客户日期"); err == nil {
		t.Fatal("expected required date error")
	}
}

func TestAttachmentReadyStatusIncludesRotationText(t *testing.T) {
	if got := attachmentReadyStatus(-1); got != "附件已加载 · 已向左旋转 90°" {
		t.Fatalf("status = %q", got)
	}
}

func TestWriteOutboundReportXLSXProducesGroupedWorkbook(t *testing.T) {
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.Local).Unix()
	end := time.Date(2026, 7, 31, 0, 0, 0, 0, time.Local).Unix()
	receipt := time.Date(2026, 7, 15, 0, 0, 0, 0, time.Local).Unix()
	target := filepath.Join(t.TempDir(), "outbound-report.xlsx")
	records := []api.OutboundSummaryRecord{
		{OrderCode: "CK-1", MaterialID: "m1", Model: "A", Name: "轴承", Quantity: 2, Price: 3.125, ReceiptDate: receipt, Index: 1},
		{OrderCode: "CK-1", MaterialID: "m2", Model: "B", Name: "螺母", Quantity: 1, Price: 0.2, ReceiptDate: receipt, Index: 2},
	}
	if err := writeOutboundReportXLSX(target, "order", records, "测试客户", start, end); err != nil {
		t.Fatal(err)
	}

	workbook, err := excelize.OpenFile(target)
	if err != nil {
		t.Fatal(err)
	}
	defer workbook.Close()
	sheet := "按单据导出"
	assertCell := func(axis, want string) {
		t.Helper()
		got, cellErr := workbook.GetCellValue(sheet, axis)
		if cellErr != nil {
			t.Fatal(cellErr)
		}
		if got != want {
			t.Fatalf("%s = %q, want %q", axis, got, want)
		}
	}
	assertCell("A2", "客户：测试客户")
	assertCell("A3", "序号")
	assertCell("F4", "CK-1")
	assertCell("I4", "3")
	assertCell("K4", "6.450")

	merges, err := workbook.GetMergeCells(sheet)
	if err != nil {
		t.Fatal(err)
	}
	foundGroupMerge := false
	for index := range merges {
		if merges[index].GetStartAxis() == "A4" && merges[index].GetEndAxis() == "A5" {
			foundGroupMerge = true
			break
		}
	}
	if !foundGroupMerge {
		t.Fatalf("missing grouped order merge: %#v", merges)
	}
}

func TestWriteOutboundReportXLSXProducesMaterialWorkbook(t *testing.T) {
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.Local).Unix()
	end := time.Date(2026, 7, 31, 0, 0, 0, 0, time.Local).Unix()
	receipt := time.Date(2026, 7, 15, 0, 0, 0, 0, time.Local).Unix()
	target := filepath.Join(t.TempDir(), "material-report.xlsx")
	records := []api.OutboundSummaryRecord{
		{OrderCode: "CK-1", MaterialID: "m1", Model: "A", Name: "轴承", Specification: "6204", Quantity: 2, Price: 3.125, ReceiptDate: receipt, Index: 1},
		{OrderCode: "CK-2", MaterialID: "m1", Model: "A", Name: "轴承", Specification: "6204", Quantity: 3, Price: 3.125, ReceiptDate: receipt, Index: 1},
	}
	if err := writeOutboundReportXLSX(target, "material", records, "测试客户", start, end); err != nil {
		t.Fatal(err)
	}

	workbook, err := excelize.OpenFile(target)
	if err != nil {
		t.Fatal(err)
	}
	defer workbook.Close()
	sheet := "按物料导出"
	for axis, want := range map[string]string{
		"A2": "客户：测试客户",
		"B4": "A",
		"F4": "CK-1",
		"F5": "CK-2",
		"I4": "5",
		"K4": "15.625",
	} {
		got, cellErr := workbook.GetCellValue(sheet, axis)
		if cellErr != nil {
			t.Fatal(cellErr)
		}
		if got != want {
			t.Fatalf("%s = %q, want %q", axis, got, want)
		}
	}

	merges, err := workbook.GetMergeCells(sheet)
	if err != nil {
		t.Fatal(err)
	}
	foundModelMerge := false
	for index := range merges {
		if merges[index].GetStartAxis() == "B4" && merges[index].GetEndAxis() == "B5" {
			foundModelMerge = true
			break
		}
	}
	if !foundModelMerge {
		t.Fatalf("missing grouped material merge: %#v", merges)
	}
}

func TestWebAlignedFilterLabels(t *testing.T) {
	if got := supplierLevelLabels(); len(got) != 4 || got[1] != "一级" || got[3] != "三级" {
		t.Fatalf("supplier levels = %#v", got)
	}
	warehouse := warehouseKind{Types: []string{"生产仓库", "冷链仓库"}}
	if got := warehouseTypeLabels(warehouse); len(got) != 3 || got[0] != "全部类型" || got[2] != "冷链仓库" {
		t.Fatalf("warehouse types = %#v", got)
	}
	if got := warehouseTypeLabels(warehouseKind{}); len(got) != 1 || got[0] != "不适用" {
		t.Fatalf("unsupported warehouse types = %#v", got)
	}
}
