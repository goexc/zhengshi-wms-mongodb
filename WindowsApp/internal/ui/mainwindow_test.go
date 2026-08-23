package ui

import (
	"testing"

	"github.com/lxn/walk"

	"zhengshi-wms-windowsapp/internal/api"
)

func TestHasMenuFindsNestedBusinessModule(t *testing.T) {
	menus := []api.Menu{{
		Name: "仓库管理",
		Path: "/warehouse",
		Children: []api.Menu{{
			Name: "库存查询",
			Path: "/inventory/index",
		}},
	}}
	if !hasMenu(menus, "库存", "/inventory") {
		t.Fatal("expected nested inventory menu to be found")
	}
	if hasMenu(menus, "物料", "/material") {
		t.Fatal("unexpected material permission")
	}
	if !hasMenuPath(menus, "/inventory/index/") {
		t.Fatal("expected exact normalized inventory path")
	}
	if hasMenuPath(menus, "/inventory") {
		t.Fatal("parent-like path must not match an exact leaf permission")
	}
}

func TestFlattenCategoryOptionsKeepsHierarchyInLabel(t *testing.T) {
	categories := []api.MaterialCategory{{
		ID:   "parent",
		Name: "标准件",
		Children: []api.MaterialCategory{{
			ID:   "child",
			Name: "螺栓",
		}},
	}}

	options := flattenCategoryOptions(categories, "")
	if len(options) != 2 {
		t.Fatalf("len(options) = %d", len(options))
	}
	if options[0].ID != "parent" || options[0].Label != "标准件" {
		t.Fatalf("parent option = %#v", options[0])
	}
	if options[1].ID != "child" || options[1].Label != "标准件 / 螺栓" {
		t.Fatalf("child option = %#v", options[1])
	}
}

func TestSelectedPageSizeDefaultsToTwenty(t *testing.T) {
	if got := selectedPageSize(nil); got != 20 {
		t.Fatalf("selectedPageSize(nil) = %d", got)
	}
}

func TestWorkspaceTabMappings(t *testing.T) {
	material := new(walk.TabPage)
	dashboard := new(walk.TabPage)
	globalLookup := new(walk.TabPage)
	operations := new(walk.TabPage)
	documents := new(walk.TabPage)
	inventory := new(walk.TabPage)
	inbound := new(walk.TabPage)
	inboundEditor := new(walk.TabPage)
	outbound := new(walk.TabPage)
	outboundReport := new(walk.TabPage)
	partner := new(walk.TabPage)
	partnerEditor := new(walk.TabPage)
	warehouse := new(walk.TabPage)
	warehouseEditor := new(walk.TabPage)
	profile := new(walk.TabPage)
	system := new(walk.TabPage)
	ui := &mainUI{
		dashboardTab:       dashboard,
		globalLookupTab:    globalLookup,
		operationTab:       operations,
		documentTab:        documents,
		materialTab:        material,
		inventoryTab:       inventory,
		inboundTab:         inbound,
		inboundEditorTab:   inboundEditor,
		outboundTab:        outbound,
		outboundReportTab:  outboundReport,
		partnerTab:         partner,
		partnerEditorTab:   partnerEditor,
		warehouseTab:       warehouse,
		warehouseEditorTab: warehouseEditor,
		profileTab:         profile,
		systemTab:          system,
	}

	for key, page := range map[string]*walk.TabPage{
		"dashboard":        dashboard,
		"global_lookup":    globalLookup,
		"operations":       operations,
		"documents":        documents,
		"material":         material,
		"inventory":        inventory,
		"inbound":          inbound,
		"inbound_editor":   inboundEditor,
		"outbound":         outbound,
		"outbound_report":  outboundReport,
		"partner":          partner,
		"partner_editor":   partnerEditor,
		"warehouse":        warehouse,
		"warehouse_editor": warehouseEditor,
		"profile":          profile,
		"system":           system,
	} {
		if got := ui.tabForKey(key); got != page {
			t.Fatalf("tabForKey(%q) = %p, want %p", key, got, page)
		}
		if got := ui.keyForTab(page); got != key {
			t.Fatalf("keyForTab(%q) = %q", key, got)
		}
	}
	if got := stringIndex([]string{"material", "inventory"}, "inventory"); got != 1 {
		t.Fatalf("stringIndex = %d", got)
	}
}

func TestDashboardCardsRequireExactListPermissions(t *testing.T) {
	perms := api.Perms{
		Menus:   []api.Menu{{Path: "/inbound/receipt"}, {Path: "/outbound/receipt"}},
		Buttons: []api.Button{{Perms: "inbound:receipt:list"}, {Perms: "outbound:order:list"}},
	}
	cards := dashboardCardSpecs(perms)
	if len(cards) != 8 {
		t.Fatalf("cards = %#v", cards)
	}
	if cards[0].Module != "inbound" || cards[0].Status != "待审核" || cards[7].Status != "已出库" {
		t.Fatalf("cards = %#v", cards)
	}
	if got := dashboardCardSpecs(api.Perms{Menus: perms.Menus}); len(got) != 0 {
		t.Fatalf("cards without buttons = %#v", got)
	}
}

func TestAvailableReadOnlyKindsRequireExactMenuAndListButton(t *testing.T) {
	perms := api.Perms{
		Menus: []api.Menu{
			{Path: "/business_partner/supplier"},
			{Path: "/business_partner/customer"},
			{Path: "/warehouse/index"},
			{Path: "/warehouse/bin"},
		},
		Buttons: []api.Button{
			{Perms: "business_partner:supplier:list"},
			{Perms: "warehouse:warehouse:list"},
			{Perms: "warehouse:bin:list"},
		},
	}
	partners := availablePartnerKinds(perms)
	if len(partners) != 1 || partners[0].Key != "supplier" {
		t.Fatalf("partner kinds = %#v", partners)
	}
	warehouses := availableWarehouseKinds(perms)
	if len(warehouses) != 2 || warehouses[0].Key != "warehouse" || warehouses[1].Key != "bin" {
		t.Fatalf("warehouse kinds = %#v", warehouses)
	}
}
