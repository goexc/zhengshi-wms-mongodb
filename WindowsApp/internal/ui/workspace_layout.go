package ui

import (
	"strings"

	"github.com/lxn/walk"

	"zhengshi-wms-windowsapp/internal/config"
)

const (
	workspaceLayoutVersion = 1
	minimumColumnWidth     = 40
	maximumColumnWidth     = 800
	maximumLayoutColumns   = 64
)

type stretchFactorLayout interface {
	SetStretchFactor(widget walk.Widget, factor int) error
}

func tableColumnKey(column *walk.TableViewColumn) string {
	if column == nil {
		return ""
	}
	for _, value := range []string{column.Name(), column.DataMemberEffective(), column.TitleEffective()} {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

func cloneTableLayouts(source map[string]config.TableLayoutState) map[string]config.TableLayoutState {
	if len(source) == 0 {
		return make(map[string]config.TableLayoutState)
	}
	result := make(map[string]config.TableLayoutState, len(source))
	for key, state := range source {
		widths := make(map[string]int, len(state.ColumnWidths))
		for column, width := range state.ColumnWidths {
			widths[column] = width
		}
		result[key] = config.TableLayoutState{
			ColumnOrder:  append([]string(nil), state.ColumnOrder...),
			ColumnWidths: widths,
		}
	}
	return result
}

func cloneSplitterLayouts(source map[string][]int) map[string][]int {
	result := make(map[string][]int, len(source))
	for key, factors := range source {
		result[key] = append([]int(nil), factors...)
	}
	return result
}

func sanitizeColumnWidth(width int) int {
	if width < minimumColumnWidth {
		return minimumColumnWidth
	}
	if width > maximumColumnWidth {
		return maximumColumnWidth
	}
	return width
}

func (ui *mainUI) registerTableLayout(key string, table *walk.TableView) {
	if ui == nil || table == nil || table.IsDisposed() || strings.TrimSpace(key) == "" {
		return
	}
	ui.layoutTables[key] = table
	if ui.workspaceMatchesSession() && ui.cfg.Workspace.LayoutVersion == workspaceLayoutVersion {
		restoreTableLayout(table, ui.cfg.Workspace.TableLayouts[key])
	}
	table.Disposing().Attach(func() {
		if state, ok := captureTableLayout(table); ok {
			if ui.cfg.Workspace.TableLayouts == nil {
				ui.cfg.Workspace.TableLayouts = make(map[string]config.TableLayoutState)
			}
			ui.cfg.Workspace.TableLayouts[key] = state
		}
		if ui.layoutTables[key] == table {
			delete(ui.layoutTables, key)
		}
	})
}

func captureTableLayout(table *walk.TableView) (config.TableLayoutState, bool) {
	if table == nil || table.IsDisposed() || table.Columns() == nil {
		return config.TableLayoutState{}, false
	}
	columns := table.Columns()
	if columns.Len() == 0 || columns.Len() > maximumLayoutColumns {
		return config.TableLayoutState{}, false
	}
	state := config.TableLayoutState{ColumnWidths: make(map[string]int, columns.Len())}
	seen := make(map[string]bool, columns.Len())
	for _, column := range table.VisibleColumnsInDisplayOrder() {
		key := tableColumnKey(column)
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		state.ColumnOrder = append(state.ColumnOrder, key)
	}
	for index := 0; index < columns.Len(); index++ {
		column := columns.At(index)
		key := tableColumnKey(column)
		if key != "" {
			state.ColumnWidths[key] = sanitizeColumnWidth(column.Width())
		}
	}
	return state, len(state.ColumnWidths) > 0
}

func restoreTableLayout(table *walk.TableView, state config.TableLayoutState) {
	if table == nil || table.IsDisposed() || len(state.ColumnWidths) == 0 {
		return
	}
	columns := table.Columns()
	if columns == nil || columns.Len() == 0 || columns.Len() > maximumLayoutColumns {
		return
	}
	byKey := make(map[string]*walk.TableViewColumn, columns.Len())
	original := make([]*walk.TableViewColumn, 0, columns.Len())
	for index := 0; index < columns.Len(); index++ {
		column := columns.At(index)
		original = append(original, column)
		if key := tableColumnKey(column); key != "" {
			if _, exists := byKey[key]; !exists {
				byKey[key] = column
			}
		}
	}
	for key, width := range state.ColumnWidths {
		if column := byKey[key]; column != nil {
			_ = column.SetWidth(sanitizeColumnWidth(width))
		}
	}
	if len(state.ColumnOrder) == 0 {
		return
	}
	ordered := make([]*walk.TableViewColumn, 0, len(original))
	added := make(map[*walk.TableViewColumn]bool, len(original))
	for _, key := range state.ColumnOrder {
		if column := byKey[key]; column != nil && !added[column] {
			ordered = append(ordered, column)
			added[column] = true
		}
	}
	for _, column := range original {
		if !added[column] {
			ordered = append(ordered, column)
		}
	}
	if len(ordered) != len(original) {
		return
	}
	for index := columns.Len() - 1; index >= 0; index-- {
		if err := columns.RemoveAt(index); err != nil {
			return
		}
	}
	for _, column := range ordered {
		if err := columns.Add(column); err != nil {
			return
		}
	}
}

func (ui *mainUI) registerSplitterLayout(key string, splitter *walk.Splitter) {
	if ui == nil || splitter == nil || splitter.IsDisposed() || strings.TrimSpace(key) == "" {
		return
	}
	ui.layoutSplitters[key] = splitter
	if ui.workspaceMatchesSession() && ui.cfg.Workspace.LayoutVersion == workspaceLayoutVersion {
		restoreSplitterLayout(splitter, ui.cfg.Workspace.SplitterLayouts[key])
	}
	splitter.Disposing().Attach(func() {
		if factors, ok := captureSplitterLayout(splitter); ok {
			if ui.cfg.Workspace.SplitterLayouts == nil {
				ui.cfg.Workspace.SplitterLayouts = make(map[string][]int)
			}
			ui.cfg.Workspace.SplitterLayouts[key] = factors
		}
		if ui.layoutSplitters[key] == splitter {
			delete(ui.layoutSplitters, key)
		}
	})
}

func captureSplitterLayout(splitter *walk.Splitter) ([]int, bool) {
	if splitter == nil || splitter.IsDisposed() || splitter.Children() == nil {
		return nil, false
	}
	children := splitter.Children()
	if children.Len() == 0 {
		return nil, false
	}
	factors := make([]int, 0, children.Len()/2+1)
	for index := 0; index < children.Len(); index += 2 {
		widget := children.At(index)
		if widget == nil || !widget.Visible() {
			return nil, false
		}
		bounds := widget.BoundsPixels()
		factor := bounds.Width
		if splitter.Orientation() == walk.Vertical {
			factor = bounds.Height
		}
		if factor <= 0 {
			return nil, false
		}
		factors = append(factors, factor)
	}
	return factors, len(factors) > 1
}

func restoreSplitterLayout(splitter *walk.Splitter, factors []int) {
	if splitter == nil || splitter.IsDisposed() || len(factors) < 2 {
		return
	}
	children := splitter.Children()
	if children == nil || children.Len()/2+1 != len(factors) {
		return
	}
	layout, ok := splitter.Layout().(stretchFactorLayout)
	if !ok {
		return
	}
	for index, factor := range factors {
		if factor <= 0 || factor > 100000 {
			return
		}
		_ = layout.SetStretchFactor(children.At(index*2), factor)
	}
}

func (ui *mainUI) captureRegisteredLayouts() (map[string]config.TableLayoutState, map[string][]int) {
	tables := cloneTableLayouts(ui.cfg.Workspace.TableLayouts)
	for key, table := range ui.layoutTables {
		if state, ok := captureTableLayout(table); ok {
			tables[key] = state
		}
	}
	splitters := cloneSplitterLayouts(ui.cfg.Workspace.SplitterLayouts)
	for key, splitter := range ui.layoutSplitters {
		if factors, ok := captureSplitterLayout(splitter); ok {
			splitters[key] = factors
		}
	}
	return tables, splitters
}

func (ui *mainUI) registerPageLayouts(key string) {
	switch key {
	case "global_lookup":
		if ui.globalLookup != nil {
			ui.registerTableLayout("global_lookup.results", ui.globalLookup.table)
			ui.registerSplitterLayout("global_lookup.workspace", ui.globalLookup.splitter)
		}
	case "operations":
		if ui.operations != nil {
			ui.registerTableLayout("operations.journal", ui.operations.table)
		}
	case "material":
		ui.registerTableLayout("material.results", ui.materialTable)
	case "material_quote":
		if ui.materialQuote != nil {
			ui.registerTableLayout("material_quote.results", ui.materialQuote.table)
		}
	case "image_assets":
		if ui.imageAssets != nil {
			ui.registerTableLayout("image_assets.results", ui.imageAssets.table)
		}
	case "inventory":
		ui.registerTableLayout("inventory.results", ui.inventoryTable)
	case "inbound":
		ui.registerTableLayout("inbound.results", ui.inboundTable)
	case "outbound":
		if ui.outbound != nil {
			ui.registerTableLayout("outbound.orders", ui.outbound.table)
			ui.registerTableLayout("outbound.materials", ui.outbound.materials)
			ui.registerSplitterLayout("outbound.orders_materials", ui.outbound.splitter)
		}
	case "outbound_report":
		if ui.outboundReport != nil {
			ui.registerTableLayout("outbound_report.results", ui.outboundReport.table)
		}
	case "partner":
		if ui.partner != nil {
			ui.registerTableLayout("partner.results", ui.partner.table)
		}
	case "warehouse":
		if ui.warehouse != nil {
			ui.registerTableLayout("warehouse.results", ui.warehouse.table)
			ui.registerSplitterLayout("warehouse.workspace", ui.warehouse.splitter)
		}
	case "admin":
		if ui.admin != nil {
			ui.registerTableLayout("admin.users", ui.admin.userTable)
			ui.registerTableLayout("admin.departments", ui.admin.departmentTable)
			ui.registerTableLayout("admin.roles", ui.admin.roleTable)
			ui.registerTableLayout("admin.menu_catalog", ui.admin.menuTable)
			ui.registerTableLayout("admin.api_catalog", ui.admin.apiTable)
		}
	}
}
