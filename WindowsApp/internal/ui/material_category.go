package ui

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"

	"zhengshi-wms-windowsapp/internal/api"
)

type materialCategoryRow struct {
	Name      string
	Parent    string
	Sort      string
	Status    string
	Remark    string
	UpdatedAt string
	Detail    api.MaterialCategory
}

type materialCategoryUI struct {
	tab        *walk.TabPage
	table      *walk.TableView
	info       *walk.Label
	refresh    *walk.PushButton
	addRoot    *walk.PushButton
	addChild   *walk.PushButton
	edit       *walk.PushButton
	rows       []materialCategoryRow
	categories []api.MaterialCategory
	ctx        context.Context
	cancel     context.CancelFunc
	generation int
	busy       bool
	closed     atomic.Bool
}

func newMaterialCategoryUI() *materialCategoryUI {
	ctx, cancel := context.WithCancel(context.Background())
	return &materialCategoryUI{ctx: ctx, cancel: cancel}
}

func (state *materialCategoryUI) dispose() {
	if state == nil || state.closed.Swap(true) {
		return
	}
	if state.cancel != nil {
		state.cancel()
	}
}

func (ui *mainUI) materialCategoryPageWidget(state *materialCategoryUI) TabPage {
	return TabPage{
		AssignTo: &state.tab,
		Title:    closableTabTitle("物料分类"),
		Layout:   VBox{Margins: Margins{Left: 14, Top: 14, Right: 14, Bottom: 12}, Spacing: 10},
		Children: []Widget{
			Label{Text: "物料分类", Font: Font{Family: "Microsoft YaHei UI", PointSize: 15, Bold: true}},
			Label{Text: "分类层级直接映射现有接口；本客户端不提供分类删除，避免误伤已关联物料。", TextColor: secondaryTextColor()},
			Composite{Layout: HBox{Spacing: 8}, Children: []Widget{
				PushButton{AssignTo: &state.addRoot, Text: "新增根分类", Visible: hasButton(ui.session.Perms.Buttons, "material:category:add"), MinSize: Size{Width: 104, Height: 30}, OnClicked: func() { ui.editMaterialCategory("add", api.MaterialCategory{}, "") }},
				PushButton{AssignTo: &state.addChild, Text: "新增子分类", Visible: hasButton(ui.session.Perms.Buttons, "material:category:add"), Enabled: false, MinSize: Size{Width: 104, Height: 30}, OnClicked: ui.addChildMaterialCategory},
				PushButton{AssignTo: &state.edit, Text: "编辑当前分类", Visible: hasButton(ui.session.Perms.Buttons, "material:category:edit"), Enabled: false, MinSize: Size{Width: 112, Height: 30}, OnClicked: ui.editSelectedMaterialCategory},
				HSpacer{},
				PushButton{AssignTo: &state.refresh, Text: "刷新", MinSize: Size{Width: 82, Height: 30}, OnClicked: ui.loadMaterialCategories},
			}},
			TableView{
				AssignTo: &state.table, Model: []materialCategoryRow{}, AlternatingRowBG: true, ColumnsOrderable: true, StretchFactor: 1,
				Accessibility:         Accessibility{Name: "物料分类层级列表", Description: "选择分类后可新增子分类或编辑"},
				OnCurrentIndexChanged: ui.updateMaterialCategoryActions,
				OnItemActivated:       ui.editSelectedMaterialCategory,
				Columns: []TableViewColumn{
					{Title: "分类层级", DataMember: "Name", Width: 280},
					{Title: "上级分类", DataMember: "Parent", Width: 180},
					{Title: "排序", DataMember: "Sort", Width: 75},
					{Title: "状态", DataMember: "Status", Width: 90},
					{Title: "备注", DataMember: "Remark", Width: 260},
					{Title: "更新时间", DataMember: "UpdatedAt", Width: 145},
				},
			},
			Composite{Layout: HBox{}, Children: []Widget{
				Label{AssignTo: &state.info, Text: "尚未加载", TextColor: secondaryTextColor(), Accessibility: Accessibility{Name: "物料分类加载状态"}},
				HSpacer{},
				Label{Text: "双击分类可编辑；写操作不会自动重试。", TextColor: secondaryTextColor()},
			}},
		},
	}
}

func (ui *mainUI) openMaterialCategoryPage() {
	if !hasMenuPath(ui.session.Perms.Menus, "/material/category") || !hasButton(ui.session.Perms.Buttons, "material:category:list") {
		walk.MsgBox(ui.window, "没有权限", "当前账号没有查看物料分类的权限。", walk.MsgBoxIconWarning)
		return
	}
	if ui.materialCategories != nil && ui.materialCategories.tab != nil {
		_ = ui.tabs.SetCurrentIndex(ui.tabs.Pages().Index(ui.materialCategories.tab))
		return
	}
	state := newMaterialCategoryUI()
	decl := ui.materialCategoryPageWidget(state)
	if err := decl.Create(NewBuilder(nil)); err != nil {
		state.dispose()
		walk.MsgBox(ui.window, "无法打开物料分类", err.Error(), walk.MsgBoxIconError)
		return
	}
	ui.materialCategories = state
	ui.materialCategoryTab = state.tab
	insertAt := ui.tabs.Pages().Len()
	if ui.systemTab != nil {
		if index := ui.tabs.Pages().Index(ui.systemTab); index >= 0 {
			insertAt = index
		}
	}
	if ui.materialTab != nil {
		if index := ui.tabs.Pages().Index(ui.materialTab); index >= 0 {
			insertAt = index + 1
		}
	}
	if err := ui.tabs.Pages().Insert(insertAt, state.tab); err != nil {
		state.tab.Dispose()
		state.dispose()
		ui.materialCategories = nil
		ui.materialCategoryTab = nil
		walk.MsgBox(ui.window, "无法打开物料分类", err.Error(), walk.MsgBoxIconError)
		return
	}
	_ = ui.tabs.SetCurrentIndex(ui.tabs.Pages().Index(state.tab))
	ui.loadMaterialCategories()
}

func (ui *mainUI) closeMaterialCategoryPage() {
	state := ui.materialCategories
	if state == nil || state.tab == nil || ui.tabs == nil {
		return
	}
	if state.busy {
		walk.MsgBox(ui.window, "正在刷新", "分类列表正在刷新，请等待完成。", walk.MsgBoxIconInformation)
		return
	}
	if index := ui.tabs.Pages().Index(state.tab); index >= 0 {
		if err := ui.tabs.Pages().RemoveAt(index); err != nil {
			walk.MsgBox(ui.window, "无法关闭物料分类", err.Error(), walk.MsgBoxIconError)
			return
		}
	}
	state.dispose()
	state.tab.Dispose()
	ui.materialCategories = nil
	ui.materialCategoryTab = nil
	ui.syncNavigationFromTab()
}

func flattenMaterialCategoryRows(categories []api.MaterialCategory, parentName, indent string) []materialCategoryRow {
	rows := make([]materialCategoryRow, 0)
	for _, category := range categories {
		rows = append(rows, materialCategoryRow{
			Name: indent + category.Name, Parent: displayMaterialValue(parentName), Sort: strconv.Itoa(category.SortID),
			Status: displayMaterialValue(category.Status), Remark: category.Remark, UpdatedAt: formatUnixMinute(category.UpdatedAt), Detail: category,
		})
		rows = append(rows, flattenMaterialCategoryRows(category.Children, category.Name, indent+"    ")...)
	}
	return rows
}

func (ui *mainUI) setMaterialCategoryBusy(state *materialCategoryUI, busy bool) {
	state.busy = busy
	state.refresh.SetEnabled(!busy)
	state.addRoot.SetEnabled(!busy && hasButton(ui.session.Perms.Buttons, "material:category:add"))
	ui.updateMaterialCategoryActions()
}

func (ui *mainUI) loadMaterialCategories() {
	state := ui.materialCategories
	if state == nil || state.closed.Load() || state.busy {
		return
	}
	state.generation++
	generation := state.generation
	ui.setMaterialCategoryBusy(state, true)
	state.info.SetText("正在读取线上分类树……")
	guardedGo(func() {
		categories, err := ui.session.Client.MaterialCategories(state.ctx)
		if state.ctx.Err() != nil || state.closed.Load() {
			return
		}
		ui.window.Synchronize(func() {
			if state != ui.materialCategories || state.closed.Load() || generation != state.generation {
				return
			}
			ui.setMaterialCategoryBusy(state, false)
			if err != nil {
				state.rows = nil
				_ = state.table.SetModel([]materialCategoryRow{})
				state.info.SetText(requestFailureText(err))
				return
			}
			state.categories = categories
			state.rows = flattenMaterialCategoryRows(categories, "", "")
			if modelErr := state.table.SetModel(state.rows); modelErr != nil {
				state.info.SetText("分类表格刷新失败：" + modelErr.Error())
				return
			}
			state.info.SetText(fmt.Sprintf("已读取 %d 个分类；仅提供新增和编辑。", len(state.rows)))
			ui.materialCategoryOptions = flattenCategoryOptions(categories, "")
			ui.categoryReady = true
			ui.categoryFailed = false
			ui.applyMaterialCategoryOptions()
			ui.updateMaterialCategoryActions()
		})
	})
}

func (ui *mainUI) selectedMaterialCategory() (api.MaterialCategory, bool) {
	state := ui.materialCategories
	if state == nil || state.table == nil {
		return api.MaterialCategory{}, false
	}
	index := state.table.CurrentIndex()
	if index < 0 || index >= len(state.rows) {
		return api.MaterialCategory{}, false
	}
	return state.rows[index].Detail, true
}

func (ui *mainUI) updateMaterialCategoryActions() {
	state := ui.materialCategories
	if state == nil || state.addChild == nil {
		return
	}
	_, selected := ui.selectedMaterialCategory()
	state.addChild.SetEnabled(selected && !state.busy && hasButton(ui.session.Perms.Buttons, "material:category:add"))
	state.edit.SetEnabled(selected && !state.busy && hasButton(ui.session.Perms.Buttons, "material:category:edit"))
}

func (ui *mainUI) addChildMaterialCategory() {
	category, ok := ui.selectedMaterialCategory()
	if !ok {
		walk.MsgBox(ui.window, "请选择分类", "请先选择上级分类。", walk.MsgBoxIconInformation)
		return
	}
	ui.editMaterialCategory("add", api.MaterialCategory{}, category.ID)
}

func (ui *mainUI) editSelectedMaterialCategory() {
	category, ok := ui.selectedMaterialCategory()
	if !ok {
		return
	}
	ui.editMaterialCategory("edit", category, category.ParentID)
}

func flattenMaterialCategoryOptions(categories []api.MaterialCategory, prefix string, excluded map[string]bool) []selectOption {
	var options []selectOption
	for _, category := range categories {
		label := category.Name
		if prefix != "" {
			label = prefix + " / " + category.Name
		}
		if !excluded[category.ID] {
			options = append(options, selectOption{ID: category.ID, Label: label})
			options = append(options, flattenMaterialCategoryOptions(category.Children, label, excluded)...)
		}
	}
	return options
}

func collectMaterialCategoryIDs(category api.MaterialCategory, ids map[string]bool) {
	ids[category.ID] = true
	for _, child := range category.Children {
		collectMaterialCategoryIDs(child, ids)
	}
}

func findMaterialCategory(categories []api.MaterialCategory, id string) (api.MaterialCategory, bool) {
	for _, category := range categories {
		if category.ID == id {
			return category, true
		}
		if found, ok := findMaterialCategory(category.Children, id); ok {
			return found, true
		}
	}
	return api.MaterialCategory{}, false
}

func (ui *mainUI) editMaterialCategory(mode string, baseline api.MaterialCategory, defaultParentID string) {
	permission := "material:category:add"
	if mode == "edit" {
		permission = "material:category:edit"
	}
	if !hasButton(ui.session.Perms.Buttons, permission) {
		walk.MsgBox(ui.window, "没有权限", "当前账号没有执行该分类操作的权限。", walk.MsgBoxIconWarning)
		return
	}
	state := ui.materialCategories
	if state == nil || state.busy {
		return
	}
	excluded := map[string]bool{}
	if mode == "edit" {
		collectMaterialCategoryIDs(baseline, excluded)
	}
	parentOptions := flattenMaterialCategoryOptions(state.categories, "", excluded)
	var dlg *walk.Dialog
	var parent *walk.ComboBox
	var name, sortValue, remark *walk.LineEdit
	var status *walk.ComboBox
	var info *walk.Label
	var save, cancelButton *walk.PushButton
	var submitting bool
	var allowClose bool
	title := "新增物料分类"
	if mode == "edit" {
		title = "编辑物料分类"
	}
	initialSort := baseline.SortID
	if mode == "add" && initialSort == 0 {
		initialSort = 1
	}
	err := Dialog{
		AssignTo: &dlg, Title: title, MinSize: Size{Width: 620, Height: 330}, Size: Size{Width: 680, Height: 380},
		Layout: VBox{Margins: Margins{Left: 16, Top: 16, Right: 16, Bottom: 14}, Spacing: 10},
		Children: []Widget{
			Label{Text: title, Font: Font{Family: "Microsoft YaHei UI", PointSize: 14, Bold: true}},
			Label{Text: "编辑提交前会重新拉取分类树；当前分类及其子级不能被选为上级。", TextColor: secondaryTextColor()},
			GroupBox{Title: "分类信息", Layout: Grid{Columns: 4, Margins: Margins{Left: 12, Top: 10, Right: 12, Bottom: 12}, Spacing: 8}, Children: []Widget{
				Label{Text: "上级分类"}, ComboBox{AssignTo: &parent, Model: optionLabels("无（根分类）", parentOptions), CurrentIndex: optionIndexByID(parentOptions, defaultParentID), MinSize: Size{Width: 220, Height: 28}},
				Label{Text: "分类名称 *"}, LineEdit{AssignTo: &name, Text: baseline.Name, MinSize: Size{Width: 220, Height: 28}},
				Label{Text: "排序"}, LineEdit{AssignTo: &sortValue, Text: strconv.Itoa(initialSort), CueBanner: "非负整数", MinSize: Size{Width: 220, Height: 28}},
				Label{Text: "状态 *"}, ComboBox{AssignTo: &status, Model: []string{"启用", "停用"}, CurrentIndex: stringIndex([]string{"启用", "停用"}, baseline.Status), MinSize: Size{Width: 220, Height: 28}},
				Label{Text: "备注"}, LineEdit{AssignTo: &remark, Text: baseline.Remark, ColumnSpan: 3},
			}},
			VSpacer{},
			Composite{Layout: HBox{Spacing: 8}, Children: []Widget{
				Label{AssignTo: &info, Text: "写操作不会自动重试。", TextColor: secondaryTextColor(), Accessibility: Accessibility{Name: "分类编辑状态"}},
				HSpacer{},
				PushButton{AssignTo: &cancelButton, Text: "取消", MinSize: Size{Width: 80, Height: 30}, OnClicked: func() { dlg.Cancel() }},
				PushButton{AssignTo: &save, Text: "核对并保存", MinSize: Size{Width: 110, Height: 30}},
			}},
		},
	}.Create(ui.window)
	if err != nil {
		walk.MsgBox(ui.window, "分类窗口错误", err.Error(), walk.MsgBoxIconError)
		return
	}
	if status.CurrentIndex() < 0 {
		status.SetCurrentIndex(0)
	}
	if strings.TrimSpace(sortValue.Text()) == "" {
		sortValue.SetText("0")
	}
	dlg.Closing().Attach(func(canceled *bool, _ walk.CloseReason) {
		if !allowClose && submitting {
			*canceled = true
			walk.MsgBox(dlg, "正在提交", "分类正在提交和复核，请等待完成。", walk.MsgBoxIconInformation)
		}
	})
	save.Clicked().Attach(func() {
		if submitting {
			return
		}
		categoryName := strings.TrimSpace(name.Text())
		if categoryName == "" {
			info.SetText("请填写分类名称。")
			name.SetFocus()
			return
		}
		sortID, parseErr := strconv.Atoi(strings.TrimSpace(sortValue.Text()))
		if parseErr != nil || sortID < 0 {
			info.SetText("排序必须是非负整数。")
			sortValue.SetFocus()
			return
		}
		request := api.MaterialCategoryRequest{ID: baseline.ID, ParentID: selectedOptionID(parent, parentOptions), SortID: sortID, Name: categoryName, Status: status.Text(), Remark: strings.TrimSpace(remark.Text())}
		if walk.MsgBox(dlg, "确认保存分类", fmt.Sprintf("分类：%s\r\n状态：%s\r\n\r\n提交后将立即写入线上数据，是否继续？", request.Name, request.Status), walk.MsgBoxYesNo|walk.MsgBoxIconWarning) != walk.DlgCmdYes {
			return
		}
		submitting = true
		save.SetEnabled(false)
		cancelButton.SetEnabled(false)
		info.SetText("正在刷新线上分类并提交……")
		guardedGo(func() {
			fresh, requestErr := ui.session.Client.MaterialCategories(state.ctx)
			if requestErr == nil && mode == "edit" {
				current, found := findMaterialCategory(fresh, baseline.ID)
				if !found {
					requestErr = fmt.Errorf("线上分类已不存在，请关闭窗口并刷新")
				} else if current.UpdatedAt != baseline.UpdatedAt {
					requestErr = fmt.Errorf("线上分类已被其他用户修改，请关闭窗口并刷新后重试")
				}
			}
			if requestErr == nil {
				if mode == "edit" {
					requestErr = ui.session.Client.UpdateMaterialCategory(state.ctx, request)
				} else {
					requestErr = ui.session.Client.CreateMaterialCategory(state.ctx, request)
				}
			}
			if state.ctx.Err() != nil || state.closed.Load() {
				return
			}
			dlg.Synchronize(func() {
				if requestErr != nil {
					submitting = false
					save.SetEnabled(true)
					cancelButton.SetEnabled(true)
					info.SetText(requestFailureText(requestErr))
					return
				}
				allowClose = true
				dlg.Accept()
			})
		})
	})
	if dlg.Run() == walk.DlgCmdOK {
		ui.loadMaterialCategories()
		if ui.materialTab != nil {
			ui.materialPage = 1
			ui.loadMaterials()
		}
	}
}
