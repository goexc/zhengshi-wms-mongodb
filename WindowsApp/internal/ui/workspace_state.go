package ui

import (
	"strings"

	"github.com/lxn/walk"

	"zhengshi-wms-windowsapp/internal/config"
)

func (ui *mainUI) workspaceMatchesSession() bool {
	state := ui.cfg.Workspace
	return strings.EqualFold(strings.TrimSpace(state.APIBaseURL), strings.TrimSpace(ui.cfg.APIBaseURL)) &&
		strings.TrimSpace(state.Mobile) != "" &&
		strings.TrimSpace(state.Mobile) == strings.TrimSpace(ui.session.Login.Mobile)
}

func (ui *mainUI) initialPageSizeIndex(module string, maximum int) int {
	if maximum <= 0 {
		return 0
	}
	value := 20
	if ui.workspaceMatchesSession() {
		switch module {
		case "material":
			value = ui.cfg.Workspace.MaterialPageSize
		case "inventory":
			value = ui.cfg.Workspace.InventoryPageSize
		case "inbound":
			value = ui.cfg.Workspace.InboundPageSize
		case "outbound":
			value = ui.cfg.Workspace.OutboundPageSize
		}
	}
	for index, size := range pageSizes {
		if size == value && index < maximum {
			return index
		}
	}
	if maximum > 1 {
		return 1
	}
	return 0
}

func (ui *mainUI) initialWorkspaceFilterIndex(module string, maximum int) int {
	if maximum <= 0 || !ui.workspaceMatchesSession() {
		return 0
	}
	value := 0
	switch module {
	case "inventory_mode":
		value = ui.cfg.Workspace.InventoryModeIndex
	case "inventory_type":
		value = ui.cfg.Workspace.InventoryTypeIndex
	case "inbound_status":
		value = ui.cfg.Workspace.InboundStatusIndex
	case "inbound_type":
		value = ui.cfg.Workspace.InboundTypeIndex
	case "outbound_stage":
		value = ui.cfg.Workspace.OutboundStageIndex
	case "outbound_type":
		value = ui.cfg.Workspace.OutboundTypeIndex
	}
	if value < 0 || value >= maximum {
		return 0
	}
	return value
}

func workspaceComboIndex(combo *walk.ComboBox) int {
	if combo == nil || combo.CurrentIndex() < 0 {
		return 0
	}
	return combo.CurrentIndex()
}

func (ui *mainUI) restoreWorkspaceTabs() {
	ui.workspaceReady = false
	if ui.tabs == nil || !ui.workspaceMatchesSession() || len(ui.cfg.Workspace.OpenPages) == 0 {
		ui.workspaceReady = true
		return
	}
	desired := make(map[string]bool, len(ui.cfg.Workspace.OpenPages)+1)
	for _, key := range ui.cfg.Workspace.OpenPages {
		if stringIndex(ui.menuKeys, key) >= 0 {
			desired[key] = true
		}
	}
	desired["system"] = true
	for index := ui.tabs.Pages().Len() - 1; index >= 0; index-- {
		page := ui.tabs.Pages().At(index)
		key := ui.keyForTab(page)
		if key != "" && stringIndex(ui.menuKeys, key) >= 0 && !desired[key] {
			_ = ui.tabs.Pages().RemoveAt(index)
		}
	}
	target := ui.cfg.Workspace.CurrentPage
	page := ui.tabForKey(target)
	if page == nil || ui.tabs.Pages().Index(page) < 0 {
		for index := 0; index < ui.tabs.Pages().Len(); index++ {
			candidate := ui.tabs.Pages().At(index)
			if candidate != ui.systemTab {
				page = candidate
				break
			}
		}
		if page == nil || ui.tabs.Pages().Index(page) < 0 {
			page = ui.systemTab
		}
	}
	if page != nil {
		_ = ui.tabs.SetCurrentIndex(ui.tabs.Pages().Index(page))
	}
	ui.workspaceReady = true
}

func (ui *mainUI) saveWorkspaceState() {
	if !ui.workspaceReady || ui.tabs == nil || ui.session == nil {
		return
	}
	openPages := make([]string, 0, ui.tabs.Pages().Len())
	for index := 0; index < ui.tabs.Pages().Len(); index++ {
		key := ui.keyForTab(ui.tabs.Pages().At(index))
		if stringIndex(ui.menuKeys, key) >= 0 {
			openPages = append(openPages, key)
		}
	}
	current := ui.currentPageKey()
	if stringIndex(ui.menuKeys, current) < 0 {
		if ui.sideMenu != nil && ui.sideMenu.CurrentIndex() >= 0 && ui.sideMenu.CurrentIndex() < len(ui.menuKeys) {
			current = ui.menuKeys[ui.sideMenu.CurrentIndex()]
		} else {
			current = "system"
		}
	}
	tableLayouts, splitterLayouts := ui.captureRegisteredLayouts()
	state := config.WorkspaceState{
		LayoutVersion:      workspaceLayoutVersion,
		TableLayouts:       tableLayouts,
		SplitterLayouts:    splitterLayouts,
		APIBaseURL:         ui.cfg.APIBaseURL,
		Mobile:             strings.TrimSpace(ui.session.Login.Mobile),
		OpenPages:          openPages,
		CurrentPage:        current,
		MaterialPageSize:   selectedPageSize(ui.materialSize),
		InventoryPageSize:  selectedPageSize(ui.inventorySize),
		InboundPageSize:    selectedPageSize(ui.inboundSize),
		OutboundPageSize:   20,
		InventoryModeIndex: workspaceComboIndex(ui.inventoryMode),
		InventoryTypeIndex: workspaceComboIndex(ui.inventoryType),
		InboundStatusIndex: workspaceComboIndex(ui.inboundStatus),
		InboundTypeIndex:   workspaceComboIndex(ui.inboundType),
		SideMenuCollapsed:  ui.sideMenuCollapsed,
	}
	if ui.window != nil {
		bounds := ui.window.Bounds()
		if validSavedWindowBounds(bounds) {
			state.WindowBoundsSet = true
			state.WindowX = bounds.X
			state.WindowY = bounds.Y
			state.WindowWidth = bounds.Width
			state.WindowHeight = bounds.Height
		}
	}
	if ui.outbound != nil {
		state.OutboundPageSize = selectedPageSize(ui.outbound.pageSize)
		state.OutboundStageIndex = workspaceComboIndex(ui.outbound.stage)
		state.OutboundTypeIndex = workspaceComboIndex(ui.outbound.orderType)
	}
	ui.cfg.Workspace = state
	if err := config.Save(ui.cfg); err != nil && ui.status != nil {
		ui.status.SetText("工作区状态保存失败：" + err.Error())
	}
}
