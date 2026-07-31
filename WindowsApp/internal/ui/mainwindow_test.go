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
	inventory := new(walk.TabPage)
	inbound := new(walk.TabPage)
	outbound := new(walk.TabPage)
	system := new(walk.TabPage)
	ui := &mainUI{
		materialTab:  material,
		inventoryTab: inventory,
		inboundTab:   inbound,
		outboundTab:  outbound,
		systemTab:    system,
	}

	for key, page := range map[string]*walk.TabPage{
		"material":  material,
		"inventory": inventory,
		"inbound":   inbound,
		"outbound":  outbound,
		"system":    system,
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
