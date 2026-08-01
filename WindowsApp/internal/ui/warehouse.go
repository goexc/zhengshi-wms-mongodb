package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"

	"zhengshi-wms-windowsapp/internal/api"
)

type warehouseKind struct {
	Key        string
	Label      string
	MenuPath   string
	Permission string
	Level      int
	Types      []string
}

type warehouseTreeEntry struct {
	ID          string
	Name        string
	Path        string
	Label       string
	Level       int
	WarehouseID string
	ZoneID      string
	RackID      string
}

type warehouseDetail struct {
	Level     string
	Warehouse string
	Parent    string
	Code      string
	Name      string
	Type      string
	Status    string
	Capacity  string
	Manager   string
	Contact   string
	Address   string
	Remark    string
	CreateBy  string
	CreatedAt int64
	UpdatedAt int64
}

type warehouseRow struct {
	Level     string
	Warehouse string
	Parent    string
	Code      string
	Name      string
	Type      string
	Status    string
	Capacity  string
	Manager   string
	Contact   string
	Remark    string
	Detail    warehouseDetail
}

type warehouseUI struct {
	kinds      []warehouseKind
	tree       *walk.ListBox
	kind       *walk.ComboBox
	typeFilter *walk.ComboBox
	name       *walk.LineEdit
	code       *walk.LineEdit
	status     *walk.ComboBox
	table      *walk.TableView
	info       *walk.Label
	query      *walk.PushButton
	reset      *walk.PushButton
	prev       *walk.PushButton
	next       *walk.PushButton
	size       *walk.ComboBox

	treeEntries    []warehouseTreeEntry
	selectedTree   *warehouseTreeEntry
	syncingTree    bool
	page           int
	total          int64
	rows           []warehouseRow
	generation     int
	treeGeneration int
	cancel         context.CancelFunc
	treeCancel     context.CancelFunc
}

type warehouseLoadResult struct {
	total int64
	rows  []warehouseRow
}

func availableWarehouseKinds(perms api.Perms) []warehouseKind {
	candidates := []warehouseKind{
		{
			Key: "warehouse", Label: "仓库", MenuPath: "/warehouse/index",
			Permission: "warehouse:warehouse:list", Level: 0,
			Types: []string{
				"分销中心", "生产仓库", "跨境仓库", "电商仓库", "冷链仓库",
				"合规仓库", "专用仓库", "跨渠道仓库", "自动化仓库", "第三方物流仓库",
			},
		},
		{Key: "zone", Label: "库区", MenuPath: "/warehouse/zone", Permission: "warehouse:zone:list", Level: 1},
		{
			Key: "rack", Label: "货架", MenuPath: "/warehouse/rack",
			Permission: "warehouse:rack:list", Level: 2,
			Types: []string{"标准货架", "重型货架", "中型货架", "轻型货架"},
		},
		{Key: "bin", Label: "货位", MenuPath: "/warehouse/bin", Permission: "warehouse:bin:list", Level: 3},
	}
	result := make([]warehouseKind, 0, len(candidates))
	for _, candidate := range candidates {
		if hasMenuPath(perms.Menus, candidate.MenuPath) && hasButton(perms.Buttons, candidate.Permission) {
			result = append(result, candidate)
		}
	}
	return result
}

func newWarehouseUI(kinds []warehouseKind) *warehouseUI {
	return &warehouseUI{kinds: append([]warehouseKind(nil), kinds...), page: 1}
}

func (ui *mainUI) warehousePageWidget() TabPage {
	state := ui.warehouse
	statuses := []string{"全部状态", "激活", "禁用", "盘点中", "关闭"}
	initialTypes := []string{"不适用"}
	if len(state.kinds) > 0 {
		initialTypes = warehouseTypeLabels(state.kinds[0])
	}
	return TabPage{
		AssignTo: &ui.warehouseTab,
		Title:    closableTabTitle("仓储结构"),
		Layout:   VBox{Margins: Margins{Left: 14, Top: 14, Right: 14, Bottom: 12}, Spacing: 10},
		Children: []Widget{
			Label{Text: "仓储结构", Font: Font{Family: "Microsoft YaHei UI", PointSize: 15, Bold: true}},
			Label{
				Text:      "左侧展示线上仓库层级；选择节点可定位对应层级，右侧详情以分页接口返回结果为准。",
				TextColor: walk.RGB(85, 85, 85),
			},
			HSplitter{
				HandleWidth:   5,
				StretchFactor: 1,
				Children: []Widget{
					GroupBox{
						Title:         "仓储层级",
						StretchFactor: 2,
						MinSize:       Size{Width: 210, Height: 420},
						MaxSize:       Size{Width: 290},
						Layout:        VBox{Margins: Margins{Left: 10, Top: 10, Right: 10, Bottom: 10}, Spacing: 8},
						Children: []Widget{
							Label{Text: "选择节点定位详情", TextColor: walk.RGB(85, 85, 85)},
							ListBox{
								AssignTo: &state.tree, Model: []string{"正在加载线上仓储树……"},
								StretchFactor: 1, MinSize: Size{Width: 190, Height: 360},
								Accessibility: Accessibility{
									Name:        "仓库库区货架货位层级",
									Description: "选择节点后查询对应层级详情",
								},
								OnCurrentIndexChanged: ui.warehouseTreeSelectionChanged,
								OnItemActivated:       ui.warehouseTreeSelectionChanged,
							},
						},
					},
					Composite{
						StretchFactor: 8,
						Layout:        VBox{Spacing: 8},
						Children: []Widget{
							GroupBox{
								Title:  "筛选条件",
								Layout: Grid{Columns: 8, Margins: Margins{Left: 14, Top: 10, Right: 14, Bottom: 12}, Spacing: 8},
								Children: []Widget{
									Label{Text: "层级"},
									ComboBox{
										AssignTo: &state.kind, Model: warehouseKindLabels(state.kinds), CurrentIndex: 0,
										MinSize: Size{Width: 130, Height: 28}, OnCurrentIndexChanged: ui.warehouseKindChanged,
									},
									Label{Text: "类型"},
									ComboBox{
										AssignTo: &state.typeFilter, Model: initialTypes, CurrentIndex: 0,
										MinSize:       Size{Width: 150, Height: 28},
										Accessibility: Accessibility{Name: "仓储类型筛选"},
									},
									Label{Text: "名称"},
									LineEdit{AssignTo: &state.name, MinSize: Size{Width: 160, Height: 28}, CueBanner: "仓库或位置名称"},
									Label{Text: "编号"},
									LineEdit{AssignTo: &state.code, MinSize: Size{Width: 140, Height: 28}, CueBanner: "完整或部分编号"},
									Label{Text: "状态"},
									ComboBox{AssignTo: &state.status, Model: statuses, CurrentIndex: 0, MinSize: Size{Width: 120, Height: 28}},
									HSpacer{ColumnSpan: 6},
									Composite{
										ColumnSpan: 8,
										Layout:     HBox{Spacing: 8},
										Children: []Widget{
											HSpacer{},
											PushButton{
												AssignTo: &state.reset, Text: "重置", MinSize: Size{Width: 88, Height: 30},
												OnClicked: ui.resetWarehouseFilters,
											},
											PushButton{
												AssignTo: &state.query, Text: "查询", MinSize: Size{Width: 96, Height: 30},
												OnClicked: func() {
													state.page = 1
													ui.loadWarehouseDirectory()
												},
											},
										},
									},
								},
							},
							TableView{
								AssignTo:         &state.table,
								AlternatingRowBG: true,
								ColumnsOrderable: true,
								StretchFactor:    1,
								Accessibility: Accessibility{
									Name:        "仓储位置查询结果",
									Description: "双击当前行查看完整只读资料",
								},
								OnItemActivated: ui.showSelectedWarehouseDetail,
								Columns: []TableViewColumn{
									{Title: "层级", DataMember: "Level", Width: 65},
									{Title: "所属仓库", DataMember: "Warehouse", Width: 120},
									{Title: "上级位置", DataMember: "Parent", Width: 130},
									{Title: "编号", DataMember: "Code", Width: 110},
									{Title: "名称", DataMember: "Name", Width: 145},
									{Title: "类型", DataMember: "Type", Width: 95},
									{Title: "状态", DataMember: "Status", Width: 75},
									{Title: "容量", DataMember: "Capacity", Width: 95},
									{Title: "负责人", DataMember: "Manager", Width: 85},
									{Title: "联系电话", DataMember: "Contact", Width: 110},
									{Title: "备注", DataMember: "Remark", Width: 150},
								},
							},
							Composite{
								Layout: HBox{Spacing: 8},
								Children: []Widget{
									Label{AssignTo: &state.info, Text: "尚未加载"},
									HSpacer{},
									PushButton{Text: "查看详情", OnClicked: ui.showSelectedWarehouseDetail},
									Label{Text: "每页"},
									ComboBox{
										AssignTo: &state.size, Model: pageSizeLabels, CurrentIndex: 1, MinSize: Size{Width: 92},
										OnCurrentIndexChanged: func() {
											if ui.window != nil && state.size != nil && state.size.CurrentIndex() >= 0 {
												state.page = 1
												ui.loadWarehouseDirectory()
											}
										},
									},
									PushButton{AssignTo: &state.prev, Text: "上一页", OnClicked: func() {
										if state.page > 1 {
											state.page--
											ui.loadWarehouseDirectory()
										}
									}},
									PushButton{AssignTo: &state.next, Text: "下一页", OnClicked: func() {
										state.page++
										ui.loadWarehouseDirectory()
									}},
								},
							},
						},
					},
				},
			},
		},
	}
}

func warehouseKindLabels(kinds []warehouseKind) []string {
	labels := make([]string, len(kinds))
	for index, kind := range kinds {
		labels[index] = kind.Label
	}
	return labels
}

func warehouseTypeLabels(kind warehouseKind) []string {
	if len(kind.Types) == 0 {
		return []string{"不适用"}
	}
	return append([]string{"全部类型"}, kind.Types...)
}

func (ui *mainUI) initializeWarehousePage() {
	state := ui.warehouse
	if ui.warehouseTab == nil || state == nil || state.name == nil || len(state.kinds) == 0 {
		return
	}
	state.generation++
	ui.updateWarehouseTypeFilter()
	for _, edit := range []*walk.LineEdit{state.name, state.code} {
		edit.KeyDown().Attach(func(key walk.Key) {
			if key == walk.KeyReturn {
				state.page = 1
				ui.loadWarehouseDirectory()
			}
		})
	}
	ui.loadWarehouseTree()
	ui.loadWarehouseDirectory()
}

func (ui *mainUI) releaseWarehousePage() {
	if ui.warehouse == nil {
		return
	}
	kinds := append([]warehouseKind(nil), ui.warehouse.kinds...)
	ui.warehouse.generation++
	ui.warehouse.treeGeneration++
	if ui.warehouse.cancel != nil {
		ui.warehouse.cancel()
	}
	if ui.warehouse.treeCancel != nil {
		ui.warehouse.treeCancel()
	}
	ui.warehouseTab = nil
	ui.warehouse = newWarehouseUI(kinds)
}

func (ui *mainUI) loadWarehouseTree() {
	state := ui.warehouse
	if state == nil || state.tree == nil {
		return
	}
	if state.treeCancel != nil {
		state.treeCancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	state.treeCancel = cancel
	state.treeGeneration++
	generation := state.treeGeneration
	go func() {
		nodes, requestErr := ui.session.Client.WarehouseTree(ctx)
		if ctx.Err() != nil {
			return
		}
		ui.window.Synchronize(func() {
			if state != ui.warehouse || generation != state.treeGeneration || state.tree == nil {
				return
			}
			if requestErr != nil {
				state.treeEntries = nil
				_ = state.tree.SetModel([]string{"仓储树加载失败，请按 F5 重试"})
				return
			}
			state.treeEntries = flattenWarehouseTree(nodes)
			labels := make([]string, len(state.treeEntries))
			for index := range state.treeEntries {
				labels[index] = state.treeEntries[index].Label
			}
			if len(labels) == 0 {
				labels = []string{"线上仓储树暂无数据"}
			}
			_ = state.tree.SetModel(labels)
		})
	}()
}

func flattenWarehouseTree(nodes []api.WarehouseNode) []warehouseTreeEntry {
	var result []warehouseTreeEntry
	var visit func([]api.WarehouseNode, int, []string, string, string, string)
	visit = func(items []api.WarehouseNode, level int, names []string, warehouseID, zoneID, rackID string) {
		for _, node := range items {
			nextNames := append(append([]string(nil), names...), node.Name)
			entry := warehouseTreeEntry{
				ID: node.ID, Name: node.Name, Path: strings.Join(nextNames, " / "), Level: level,
				Label:       strings.Repeat("    ", level) + node.Name,
				WarehouseID: warehouseID, ZoneID: zoneID, RackID: rackID,
			}
			switch level {
			case 0:
				entry.WarehouseID = node.ID
			case 1:
				entry.ZoneID = node.ID
			case 2:
				entry.RackID = node.ID
			}
			result = append(result, entry)
			visit(node.Children, level+1, nextNames, entry.WarehouseID, entry.ZoneID, entry.RackID)
		}
	}
	visit(nodes, 0, nil, "", "", "")
	return result
}

func (ui *mainUI) warehouseTreeSelectionChanged() {
	state := ui.warehouse
	if state == nil || state.tree == nil || state.kind == nil {
		return
	}
	index := state.tree.CurrentIndex()
	if index < 0 || index >= len(state.treeEntries) {
		return
	}
	entry := state.treeEntries[index]
	kindIndex := -1
	for index, kind := range state.kinds {
		if kind.Level == entry.Level {
			kindIndex = index
			break
		}
	}
	if kindIndex < 0 {
		return
	}
	state.syncingTree = true
	state.kind.SetCurrentIndex(kindIndex)
	state.name.SetText(entry.Name)
	state.syncingTree = false
	state.selectedTree = &entry
	state.page = 1
	ui.loadWarehouseDirectory()
}

func (ui *mainUI) warehouseKindChanged() {
	state := ui.warehouse
	if ui.window == nil || state == nil || state.kind == nil || state.kind.CurrentIndex() < 0 {
		return
	}
	ui.updateWarehouseTypeFilter()
	if state.syncingTree {
		return
	}
	state.selectedTree = nil
	if state.tree != nil {
		_ = state.tree.SetCurrentIndex(-1)
	}
	state.page = 1
	ui.loadWarehouseDirectory()
}

func (ui *mainUI) updateWarehouseTypeFilter() {
	state := ui.warehouse
	if state == nil || state.kind == nil || state.typeFilter == nil {
		return
	}
	index := state.kind.CurrentIndex()
	if index < 0 || index >= len(state.kinds) {
		return
	}
	hasTypes := len(state.kinds[index].Types) > 0
	_ = state.typeFilter.SetModel(warehouseTypeLabels(state.kinds[index]))
	_ = state.typeFilter.SetCurrentIndex(0)
	state.typeFilter.SetEnabled(hasTypes)
}

func (ui *mainUI) resetWarehouseFilters() {
	state := ui.warehouse
	if state == nil {
		return
	}
	state.name.SetText("")
	state.code.SetText("")
	_ = state.typeFilter.SetCurrentIndex(0)
	state.status.SetCurrentIndex(0)
	state.selectedTree = nil
	if state.tree != nil {
		_ = state.tree.SetCurrentIndex(-1)
	}
	state.page = 1
	ui.loadWarehouseDirectory()
}

func (ui *mainUI) loadWarehouseDirectory() {
	state := ui.warehouse
	if state == nil || state.table == nil || state.kind == nil || len(state.kinds) == 0 {
		return
	}
	index := state.kind.CurrentIndex()
	if index < 0 || index >= len(state.kinds) {
		return
	}
	if state.cancel != nil {
		state.cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	state.cancel = cancel
	state.generation++
	generation := state.generation
	page := state.page
	size := selectedPageSize(state.size)
	kind := state.kinds[index]
	filters := api.WarehouseFilters{Name: state.name.Text(), Code: state.code.Text()}
	if len(kind.Types) > 0 && state.typeFilter.CurrentIndex() > 0 {
		filters.Type = state.typeFilter.Text()
	}
	if state.status.CurrentIndex() > 0 {
		filters.Status = state.status.Text()
	}
	if entry := state.selectedTree; entry != nil {
		filters.WarehouseID = entry.WarehouseID
		filters.WarehouseZoneID = entry.ZoneID
		filters.WarehouseRackID = entry.RackID
		switch kind.Key {
		case "warehouse":
			filters.WarehouseID, filters.WarehouseZoneID, filters.WarehouseRackID = "", "", ""
		case "zone":
			filters.WarehouseZoneID, filters.WarehouseRackID = "", ""
		case "rack":
			filters.WarehouseRackID = ""
		}
	}
	state.info.SetText("正在加载线上" + kind.Label + "详情……")
	state.query.SetEnabled(false)
	state.reset.SetEnabled(false)
	state.prev.SetEnabled(false)
	state.next.SetEnabled(false)

	go func() {
		result, requestErr := ui.loadWarehouseRows(ctx, kind, page, size, filters)
		if ctx.Err() != nil {
			return
		}
		ui.window.Synchronize(func() {
			if state != ui.warehouse || generation != state.generation || state.table == nil {
				return
			}
			state.query.SetEnabled(true)
			state.reset.SetEnabled(true)
			if requestErr != nil {
				state.info.SetText("加载失败：" + requestErr.Error() + "。左侧仓储树不受影响。")
				return
			}
			if modelErr := state.table.SetModel(result.rows); modelErr != nil {
				state.info.SetText("表格数据错误：" + modelErr.Error())
				return
			}
			state.rows = result.rows
			state.total = result.total
			if len(result.rows) == 0 && len(state.treeEntries) > 0 {
				state.info.SetText(kind.Label + "分页接口未返回详情；左侧层级树仍可用于查看线上结构。")
			} else {
				state.info.SetText(fmt.Sprintf("%s | 第 %d 页 | 本页 %d 条 | 共 %d 条", kind.Label, page, len(result.rows), result.total))
			}
			state.prev.SetEnabled(page > 1)
			state.next.SetEnabled(int64(page*size) < result.total)
		})
	}()
}

func (ui *mainUI) loadWarehouseRows(
	ctx context.Context,
	kind warehouseKind,
	page, size int,
	filters api.WarehouseFilters,
) (warehouseLoadResult, error) {
	switch kind.Key {
	case "warehouse":
		result, err := ui.session.Client.Warehouses(ctx, page, size, filters)
		rows := make([]warehouseRow, 0, len(result.List))
		for _, item := range result.List {
			rows = append(rows, warehouseRowFromWarehouse(item))
		}
		return warehouseLoadResult{total: result.Total, rows: rows}, err
	case "zone":
		result, err := ui.session.Client.WarehouseZones(ctx, page, size, filters)
		rows := make([]warehouseRow, 0, len(result.List))
		for _, item := range result.List {
			rows = append(rows, warehouseRowFromZone(item))
		}
		return warehouseLoadResult{total: result.Total, rows: rows}, err
	case "rack":
		result, err := ui.session.Client.WarehouseRacks(ctx, page, size, filters)
		rows := make([]warehouseRow, 0, len(result.List))
		for _, item := range result.List {
			rows = append(rows, warehouseRowFromRack(item))
		}
		return warehouseLoadResult{total: result.Total, rows: rows}, err
	case "bin":
		result, err := ui.session.Client.WarehouseBins(ctx, page, size, filters)
		rows := make([]warehouseRow, 0, len(result.List))
		for _, item := range result.List {
			rows = append(rows, warehouseRowFromBin(item))
		}
		return warehouseLoadResult{total: result.Total, rows: rows}, err
	default:
		return warehouseLoadResult{}, fmt.Errorf("不支持的仓储层级：%s", kind.Label)
	}
}

func warehouseCapacity(value float64, unit string) string {
	if value == 0 && strings.TrimSpace(unit) == "" {
		return ""
	}
	return strings.TrimSpace(fmt.Sprintf("%g %s", value, unit))
}

func warehouseRowFromWarehouse(item api.Warehouse) warehouseRow {
	detail := warehouseDetail{
		Level: "仓库", Warehouse: item.Name, Code: item.Code, Name: item.Name, Type: item.Type,
		Status: item.Status, Capacity: warehouseCapacity(item.Capacity, item.CapacityUnit),
		Manager: item.Manager, Contact: item.Contact, Address: item.Address, Remark: item.Remark,
		CreateBy: item.CreateBy, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
	}
	return warehouseRowFromDetail(detail)
}

func warehouseRowFromZone(item api.WarehouseZone) warehouseRow {
	detail := warehouseDetail{
		Level: "库区", Warehouse: item.WarehouseName, Parent: item.WarehouseName,
		Code: item.Code, Name: item.Name, Status: item.Status,
		Capacity: warehouseCapacity(item.Capacity, item.CapacityUnit),
		Manager:  item.Manager, Contact: item.Contact, Remark: item.Remark,
		CreateBy: item.CreateBy, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
	}
	return warehouseRowFromDetail(detail)
}

func warehouseRowFromRack(item api.WarehouseRack) warehouseRow {
	detail := warehouseDetail{
		Level: "货架", Warehouse: item.WarehouseName, Parent: item.WarehouseZoneName,
		Code: item.Code, Name: item.Name, Type: item.Type, Status: item.Status,
		Capacity: warehouseCapacity(item.Capacity, item.CapacityUnit),
		Manager:  item.Manager, Contact: item.Contact, Remark: item.Remark,
		CreateBy: item.CreateBy, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
	}
	return warehouseRowFromDetail(detail)
}

func warehouseRowFromBin(item api.WarehouseBin) warehouseRow {
	detail := warehouseDetail{
		Level: "货位", Warehouse: item.WarehouseName,
		Parent: strings.Trim(strings.Join([]string{item.WarehouseZoneName, item.WarehouseRackName}, " / "), " /"),
		Code:   item.Code, Name: item.Name, Status: item.Status,
		Capacity: warehouseCapacity(item.Capacity, item.CapacityUnit),
		Manager:  item.Manager, Contact: item.Contact, Remark: item.Remark,
		CreateBy: item.CreateBy, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
	}
	return warehouseRowFromDetail(detail)
}

func warehouseRowFromDetail(detail warehouseDetail) warehouseRow {
	return warehouseRow{
		Level: detail.Level, Warehouse: detail.Warehouse, Parent: detail.Parent,
		Code: detail.Code, Name: detail.Name, Type: detail.Type, Status: detail.Status,
		Capacity: detail.Capacity, Manager: detail.Manager, Contact: detail.Contact,
		Remark: detail.Remark, Detail: detail,
	}
}

func (ui *mainUI) showSelectedWarehouseDetail() {
	state := ui.warehouse
	if state == nil || state.table == nil {
		return
	}
	index := state.table.CurrentIndex()
	if index < 0 || index >= len(state.rows) {
		walk.MsgBox(ui.window, "请选择仓储位置", "请先在右侧列表中选择一条仓储资料。", walk.MsgBoxIconInformation)
		return
	}
	showWarehouseDetail(ui.window, state.rows[index].Detail)
}

func showWarehouseDetail(owner walk.Form, detail warehouseDetail) {
	var dlg *walk.Dialog
	var closeButton *walk.PushButton
	err := Dialog{
		AssignTo:      &dlg,
		Title:         detail.Level + "详情 - " + detail.Name,
		DefaultButton: &closeButton,
		MinSize:       Size{Width: 580, Height: 430},
		Size:          Size{Width: 680, Height: 560},
		Layout:        VBox{Margins: Margins{Left: 18, Top: 18, Right: 18, Bottom: 16}, Spacing: 10},
		Children: []Widget{
			Label{Text: detail.Name, Font: Font{Family: "Microsoft YaHei UI", PointSize: 15, Bold: true}},
			Label{Text: detail.Level + " · " + displayMaterialValue(detail.Status), TextColor: walk.RGB(70, 70, 70)},
			GroupBox{
				Title:         "只读资料",
				StretchFactor: 1,
				Layout:        Grid{Columns: 2, Margins: Margins{Left: 14, Top: 14, Right: 14, Bottom: 14}, Spacing: 9},
				Children:      warehouseDetailWidgets(detail),
			},
			Composite{Layout: HBox{}, Children: []Widget{
				Label{Text: "详情来自当前线上分页接口。", TextColor: walk.RGB(90, 90, 90)},
				HSpacer{},
				PushButton{AssignTo: &closeButton, Text: "关闭", MinSize: Size{Width: 88, Height: 30}, OnClicked: func() { dlg.Accept() }},
			}},
		},
	}.Create(owner)
	if err != nil {
		walk.MsgBox(owner, "详情窗口错误", err.Error(), walk.MsgBoxIconError)
		return
	}
	dlg.Run()
}

func warehouseDetailWidgets(detail warehouseDetail) []Widget {
	fields := []struct {
		label string
		value string
	}{
		{"层级", detail.Level},
		{"所属仓库", detail.Warehouse},
		{"上级位置", detail.Parent},
		{"名称", detail.Name},
		{"编号", detail.Code},
		{"类型", detail.Type},
		{"状态", detail.Status},
		{"容量", detail.Capacity},
		{"负责人", detail.Manager},
		{"联系电话", detail.Contact},
		{"地址", detail.Address},
		{"备注", detail.Remark},
		{"创建人", detail.CreateBy},
		{"创建时间", formatWarehouseTime(detail.CreatedAt)},
		{"更新时间", formatWarehouseTime(detail.UpdatedAt)},
	}
	widgets := make([]Widget, 0, len(fields)*2)
	for _, field := range fields {
		value := displayMaterialValue(field.value)
		widgets = append(widgets,
			Label{Text: field.label, Font: Font{Family: "Microsoft YaHei UI", Bold: true}, Alignment: AlignHNearVNear},
			TextLabel{
				Text: value, TextAlignment: AlignHNearVNear, ToolTipText: value,
				Accessibility: Accessibility{Name: field.label + "：" + value},
			},
		)
	}
	return widgets
}

func formatWarehouseTime(value int64) string {
	if value <= 0 {
		return ""
	}
	return time.Unix(value, 0).Format("2006-01-02 15:04")
}
