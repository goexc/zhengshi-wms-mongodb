package ui

import (
	"strings"
	"testing"

	"zhengshi-wms-windowsapp/internal/api"
	"zhengshi-wms-windowsapp/internal/config"
)

func TestAvailableGlobalLookupKindsFollowExistingMenus(t *testing.T) {
	labels, keys := availableGlobalLookupKinds(api.Perms{Menus: []api.Menu{{Path: "/material/list"}, {Path: "/outbound/receipt"}}})
	if strings.Join(labels, ",") != "自动识别,物料,出库单" || strings.Join(keys, ",") != "auto,material,outbound" {
		t.Fatalf("labels=%v keys=%v", labels, keys)
	}
}

func TestAvailableDocumentKindsDoNotExposeUnavailableModules(t *testing.T) {
	labels, keys := availableDocumentKinds(api.Perms{Menus: []api.Menu{{Path: "/inbound/receipt"}}})
	if strings.Join(labels, ",") != "入库收货核对单,入库单号标签" || strings.Join(keys, ",") != "inbound_checklist,inbound_label" {
		t.Fatalf("labels=%v keys=%v", labels, keys)
	}
}

func TestInboundChecklistDocumentShowsQuantityReconciliation(t *testing.T) {
	document := inboundChecklistDocument(api.InboundReceipt{
		Code: "IN-1", Type: "采购入库", Status: "部分入库", SupplierName: "供应商",
		Materials: []api.InboundMaterial{{Index: 0, Name: "螺栓", Model: "M8", EstimatedQuantity: 10, ActualQuantity: 4, Unit: "个", Status: "部分入库"}},
	}, []api.InboundRecord{{Code: "BATCH-1"}})
	preview := documentPreviewText(document)
	for _, expected := range []string{"IN-1", "已记录收货批次：1", "计划 10 个", "已收 4", "剩余 6"} {
		if !strings.Contains(preview, expected) {
			t.Fatalf("preview missing %q: %s", expected, preview)
		}
	}
}

func TestOperationLabelsKeepUnknownDistinctFromFailure(t *testing.T) {
	if got := operationOutcomeText(api.OperationPending); got != "正在提交" {
		t.Fatalf("pending = %q", got)
	}
	if got := operationOutcomeText(api.OperationUnknown); got != "结果待确认" {
		t.Fatalf("unknown = %q", got)
	}
	if got := operationOutcomeText(api.OperationFailed); got != "服务端拒绝" {
		t.Fatalf("failed = %q", got)
	}
}

func TestWorkspacePageSizeIsScopedToAccountAndAPI(t *testing.T) {
	ui := &mainUI{
		cfg: config.Config{APIBaseURL: "https://api.example", Workspace: config.WorkspaceState{
			APIBaseURL: "https://api.example", Mobile: "18800000000", MaterialPageSize: 50,
		}},
		session: &Session{Login: api.LoginData{Mobile: "18800000000"}},
	}
	if got := ui.initialPageSizeIndex("material", 4); got != 2 {
		t.Fatalf("matching index = %d", got)
	}
	ui.session.Login.Mobile = "19900000000"
	if got := ui.initialPageSizeIndex("material", 4); got != 1 {
		t.Fatalf("other account index = %d", got)
	}
}

func TestWorkspaceEnumFiltersAreScopedAndBoundsChecked(t *testing.T) {
	ui := &mainUI{
		cfg: config.Config{APIBaseURL: "https://api.example", Workspace: config.WorkspaceState{
			APIBaseURL: "https://api.example", Mobile: "18800000000", OutboundStageIndex: 4,
		}},
		session: &Session{Login: api.LoginData{Mobile: "18800000000"}},
	}
	if got := ui.initialWorkspaceFilterIndex("outbound_stage", 10); got != 4 {
		t.Fatalf("stage index = %d", got)
	}
	if got := ui.initialWorkspaceFilterIndex("outbound_stage", 3); got != 0 {
		t.Fatalf("out-of-range stage index = %d", got)
	}
	ui.session.Login.Mobile = "19900000000"
	if got := ui.initialWorkspaceFilterIndex("outbound_stage", 10); got != 0 {
		t.Fatalf("other account stage index = %d", got)
	}
}
