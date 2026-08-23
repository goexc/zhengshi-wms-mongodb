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
	"zhengshi-wms-windowsapp/internal/config"
)

type warehouseParentOption struct {
	ID     string
	Label  string
	Status string
}

type warehouseEditorUI struct {
	tab             *walk.TabPage
	kind            warehouseKind
	mode            string
	baseline        warehouseDetail
	defaultParentID string
	parentLabel     *walk.Label
	parent          *walk.ComboBox
	typeLabel       *walk.Label
	typeValue       *walk.ComboBox
	name            *walk.LineEdit
	code            *walk.LineEdit
	addressLabel    *walk.Label
	address         *walk.LineEdit
	capacity        *walk.LineEdit
	capacityUnit    *walk.LineEdit
	manager         *walk.LineEdit
	contact         *walk.LineEdit
	image           *walk.LineEdit
	selectImage     *walk.PushButton
	upload          *walk.PushButton
	preview         *walk.PushButton
	clearImage      *walk.PushButton
	remark          *walk.LineEdit
	info            *walk.Label
	save            *walk.PushButton
	cancelButton    *walk.PushButton

	parentOptions     []warehouseParentOption
	dependenciesReady bool
	dirty             bool
	initializing      bool
	busy              bool
	submitted         bool
	ctx               context.Context
	cancel            context.CancelFunc
	uploadCancel      context.CancelFunc
	closed            atomic.Bool
}

type warehouseEditorValues struct {
	ParentID     string
	ParentStatus string
	Type         string
	Name         string
	Code         string
	Address      string
	Capacity     float64
	CapacityUnit string
	Manager      string
	Contact      string
	Image        string
	Remark       string
}

func newWarehouseEditorUI(kind warehouseKind, mode string, baseline warehouseDetail, defaultParentID string) *warehouseEditorUI {
	ctx, cancel := context.WithCancel(context.Background())
	return &warehouseEditorUI{
		kind: kind, mode: mode, baseline: baseline, defaultParentID: defaultParentID,
		ctx: ctx, cancel: cancel, initializing: true, dependenciesReady: kind.Key == "warehouse",
	}
}

func (state *warehouseEditorUI) dispose() {
	if state == nil || state.closed.Swap(true) {
		return
	}
	if state.cancel != nil {
		state.cancel()
	}
}

func (ui *mainUI) warehouseEditorPageWidget(state *warehouseEditorUI) TabPage {
	title := "新增" + state.kind.Label
	description := "父级、类型和状态继续由服务端校验；本页只提交现有接口定义的字段。"
	if state.mode == "edit" {
		title = "编辑" + state.kind.Label + " · " + state.baseline.Name
		description = "仅激活状态可编辑；保存前会重新读取更新时间，避免覆盖已变化的资料。"
	}
	hasParent := state.kind.Key != "warehouse"
	hasType := state.kind.Key == "warehouse" || state.kind.Key == "rack"
	isWarehouse := state.kind.Key == "warehouse"
	parentName := map[string]string{"zone": "所属仓库 *", "rack": "所属库区 *", "bin": "所属货架 *"}[state.kind.Key]
	typeModel := state.kind.Types
	if len(typeModel) == 0 {
		typeModel = []string{"不适用"}
	}
	imageLabel := "图片"
	if state.kind.Key == "zone" || state.kind.Key == "rack" {
		imageLabel = "图片 *"
	}
	return TabPage{
		AssignTo: &state.tab,
		Title:    closableTabTitle(title),
		Layout:   VBox{Margins: Margins{Left: 14, Top: 14, Right: 14, Bottom: 12}, Spacing: 10},
		Children: []Widget{
			Label{Text: title, Font: Font{Family: "Microsoft YaHei UI", PointSize: 15, Bold: true}},
			Label{Text: description, TextColor: secondaryTextColor(), Accessibility: Accessibility{Name: state.kind.Label + "编辑说明"}},
			GroupBox{
				Title:  "归属与基础信息",
				Layout: Grid{Columns: 4, Margins: Margins{Left: 14, Top: 12, Right: 14, Bottom: 14}, Spacing: 10},
				Children: []Widget{
					Label{AssignTo: &state.parentLabel, Text: parentName, Visible: hasParent},
					ComboBox{AssignTo: &state.parent, Model: []string{"正在加载可选父级……"}, CurrentIndex: 0, Visible: hasParent, MinSize: Size{Width: 240, Height: 28}},
					Label{AssignTo: &state.typeLabel, Text: "类型 *", Visible: hasType},
					ComboBox{AssignTo: &state.typeValue, Model: typeModel, CurrentIndex: 0, Visible: hasType, MinSize: Size{Width: 240, Height: 28}},
					Label{Text: state.kind.Label + "名称 *"},
					LineEdit{AssignTo: &state.name, MinSize: Size{Width: 240, Height: 28}, CueBanner: "必填"},
					Label{Text: state.kind.Label + "编号 *"},
					LineEdit{AssignTo: &state.code, MinSize: Size{Width: 240, Height: 28}, CueBanner: "必填且保持唯一"},
					Label{AssignTo: &state.addressLabel, Text: "仓库地址", Visible: isWarehouse},
					LineEdit{AssignTo: &state.address, Visible: isWarehouse, MinSize: Size{Width: 240, Height: 28}, CueBanner: "可选"},
				},
			},
			GroupBox{
				Title:  "容量与负责人",
				Layout: Grid{Columns: 4, Margins: Margins{Left: 14, Top: 12, Right: 14, Bottom: 14}, Spacing: 10},
				Children: []Widget{
					Label{Text: "容量"},
					LineEdit{AssignTo: &state.capacity, MinSize: Size{Width: 240, Height: 28}, CueBanner: "非负数字，可留空"},
					Label{Text: "容量单位"},
					LineEdit{AssignTo: &state.capacityUnit, MinSize: Size{Width: 240, Height: 28}, CueBanner: "平方米、立方米、托等"},
					Label{Text: "负责人"},
					LineEdit{AssignTo: &state.manager, MinSize: Size{Width: 240, Height: 28}, CueBanner: "可选"},
					Label{Text: "联系电话"},
					LineEdit{AssignTo: &state.contact, MinSize: Size{Width: 240, Height: 28}, CueBanner: "可选手机号"},
				},
			},
			GroupBox{
				Title:  "图片与备注",
				Layout: Grid{Columns: 4, Margins: Margins{Left: 14, Top: 12, Right: 14, Bottom: 14}, Spacing: 10},
				Children: []Widget{
					Label{Text: imageLabel},
					Composite{ColumnSpan: 3, Layout: HBox{Spacing: 8}, Children: []Widget{
						LineEdit{AssignTo: &state.image, ReadOnly: true, StretchFactor: 1, CueBanner: "未选择图片"},
						PushButton{AssignTo: &state.selectImage, Text: "选择素材", MinSize: Size{Width: 92, Height: 30}, OnClicked: ui.selectWarehouseImage},
						PushButton{AssignTo: &state.upload, Text: "上传图片", MinSize: Size{Width: 92, Height: 30}, ToolTipText: "上传期间再次点击可取消", Accessibility: Accessibility{Name: "上传仓储位置图片"}, OnClicked: ui.uploadWarehouseImage},
						PushButton{AssignTo: &state.preview, Text: "预览", Enabled: false, MinSize: Size{Width: 72, Height: 30}, OnClicked: ui.previewWarehouseImage},
						PushButton{AssignTo: &state.clearImage, Text: "清除", Enabled: false, MinSize: Size{Width: 72, Height: 30}, OnClicked: ui.clearWarehouseImage},
					}},
					Label{Text: "备注"},
					LineEdit{AssignTo: &state.remark, ColumnSpan: 3, CueBanner: "可选"},
				},
			},
			VSpacer{},
			Composite{Layout: HBox{Spacing: 8}, Children: []Widget{
				Label{AssignTo: &state.info, Text: "正在准备线上父级选项……", TextColor: secondaryTextColor(), Accessibility: Accessibility{Name: state.kind.Label + "编辑状态"}},
				HSpacer{},
				PushButton{AssignTo: &state.cancelButton, Text: "取消", MinSize: Size{Width: 82, Height: 30}, OnClicked: func() { ui.closeWarehouseEditor(false) }},
				PushButton{AssignTo: &state.save, Text: "核对并保存", Enabled: state.dependenciesReady, MinSize: Size{Width: 112, Height: 30}, OnClicked: ui.submitWarehouseEditor},
			}},
		},
	}
}

func (ui *mainUI) currentWarehouseKind() (warehouseKind, bool) {
	state := ui.warehouse
	if state == nil || state.kind == nil {
		return warehouseKind{}, false
	}
	index := state.kind.CurrentIndex()
	if index < 0 || index >= len(state.kinds) {
		return warehouseKind{}, false
	}
	return state.kinds[index], true
}

func (ui *mainUI) selectedWarehouse() (warehouseDetail, warehouseKind, bool) {
	kind, ok := ui.currentWarehouseKind()
	if !ok || ui.warehouse.table == nil {
		return warehouseDetail{}, warehouseKind{}, false
	}
	index := ui.warehouse.table.CurrentIndex()
	if index < 0 || index >= len(ui.warehouse.rows) {
		return warehouseDetail{}, kind, false
	}
	return ui.warehouse.rows[index].Detail, kind, true
}

func (ui *mainUI) updateWarehouseActions() {
	state := ui.warehouse
	if state == nil || state.add == nil {
		return
	}
	kind, kindOK := ui.currentWarehouseKind()
	detail, _, selected := ui.selectedWarehouse()
	state.add.SetEnabled(kindOK && !state.busy && hasButton(ui.session.Perms.Buttons, kind.AddPermission))
	state.edit.SetEnabled(kindOK && selected && detail.Status == "激活" && !state.busy && hasButton(ui.session.Perms.Buttons, kind.EditPermission))
	state.statusAction.SetEnabled(kindOK && selected && !state.busy && hasButton(ui.session.Perms.Buttons, kind.StatusPermission))
	state.detailAction.SetEnabled(selected && !state.busy)
	if state.actionHint == nil {
		return
	}
	if state.busy {
		state.actionHint.SetText("正在读取或提交线上仓储资料，请等待完成。")
	} else if !selected {
		state.actionHint.SetText("选择一条资料后可按权限操作；只有激活状态可编辑。")
	} else if detail.Status != "激活" {
		state.actionHint.SetText("当前状态不可编辑，可在有权限时执行状态变更。")
	} else {
		state.actionHint.SetText("写操作会在提交前重新读取，并在成功后刷新层级树。")
	}
}

func (ui *mainUI) newWarehouseEntity() {
	kind, ok := ui.currentWarehouseKind()
	if !ok || !hasButton(ui.session.Perms.Buttons, kind.AddPermission) {
		walk.MsgBox(ui.window, "没有权限", "当前账号没有新增该仓储层级的明确按钮权限。", walk.MsgBoxIconWarning)
		return
	}
	defaultParentID := ""
	if entry := ui.warehouse.selectedTree; entry != nil {
		switch kind.Key {
		case "zone":
			if entry.Level == 0 {
				defaultParentID = entry.ID
			}
		case "rack":
			if entry.Level == 1 {
				defaultParentID = entry.ID
			}
		case "bin":
			if entry.Level == 2 {
				defaultParentID = entry.ID
			}
		}
	}
	ui.openWarehouseEditor(kind, "add", warehouseDetail{}, defaultParentID)
}

func (ui *mainUI) editSelectedWarehouse() {
	detail, kind, ok := ui.selectedWarehouse()
	if !ok {
		walk.MsgBox(ui.window, "请选择仓储资料", "请先选择需要编辑的仓储资料。", walk.MsgBoxIconInformation)
		return
	}
	if detail.Status != "激活" {
		walk.MsgBox(ui.window, "当前不可编辑", "只有激活状态的仓储资料可以编辑。", walk.MsgBoxIconWarning)
		return
	}
	if !hasButton(ui.session.Perms.Buttons, kind.EditPermission) {
		walk.MsgBox(ui.window, "没有权限", "当前账号没有编辑该仓储层级的权限。", walk.MsgBoxIconWarning)
		return
	}
	parentID := map[string]string{"zone": detail.WarehouseID, "rack": detail.ZoneID, "bin": detail.RackID}[kind.Key]
	ui.openWarehouseEditor(kind, "edit", detail, parentID)
}

func (ui *mainUI) openWarehouseEditor(kind warehouseKind, mode string, baseline warehouseDetail, defaultParentID string) {
	if current := ui.warehouseEditor; current != nil {
		if current.busy {
			walk.MsgBox(ui.window, "正在提交", "当前仓储资料正在提交和复核，请等待完成。", walk.MsgBoxIconInformation)
			return
		}
		if current.kind.Key == kind.Key && current.mode == mode && current.baseline.ID == baseline.ID {
			_ = ui.tabs.SetCurrentIndex(ui.tabs.Pages().Index(current.tab))
			return
		}
		if current.dirty && walk.MsgBox(ui.window, "替换未提交编辑", "当前仓储资料还有未提交修改，是否放弃并打开另一条资料？", walk.MsgBoxYesNo|walk.MsgBoxIconWarning) != walk.DlgCmdYes {
			return
		}
		ui.closeWarehouseEditor(true)
	}
	state := newWarehouseEditorUI(kind, mode, baseline, defaultParentID)
	decl := ui.warehouseEditorPageWidget(state)
	if err := decl.Create(NewBuilder(nil)); err != nil {
		state.dispose()
		walk.MsgBox(ui.window, "无法打开编辑页", err.Error(), walk.MsgBoxIconError)
		return
	}
	ui.warehouseEditor = state
	ui.warehouseEditorTab = state.tab
	insertAt := ui.tabs.Pages().Len()
	if ui.systemTab != nil {
		if index := ui.tabs.Pages().Index(ui.systemTab); index >= 0 {
			insertAt = index
		}
	}
	if ui.warehouseTab != nil {
		if index := ui.tabs.Pages().Index(ui.warehouseTab); index >= 0 {
			insertAt = index + 1
		}
	}
	if err := ui.tabs.Pages().Insert(insertAt, state.tab); err != nil {
		state.tab.Dispose()
		state.dispose()
		ui.warehouseEditor = nil
		ui.warehouseEditorTab = nil
		walk.MsgBox(ui.window, "无法打开编辑页", err.Error(), walk.MsgBoxIconError)
		return
	}
	_ = ui.tabs.SetCurrentIndex(ui.tabs.Pages().Index(state.tab))
	ui.initializeWarehouseEditor(state)
}

func (ui *mainUI) initializeWarehouseEditor(state *warehouseEditorUI) {
	if state == nil || state != ui.warehouseEditor {
		return
	}
	baseline := state.baseline
	if len(state.kind.Types) > 0 {
		index := stringIndex(state.kind.Types, baseline.Type)
		if index < 0 {
			index = 0
		}
		state.typeValue.SetCurrentIndex(index)
	}
	state.name.SetText(baseline.Name)
	state.code.SetText(baseline.Code)
	state.address.SetText(baseline.Address)
	if baseline.CapacityValue != 0 {
		state.capacity.SetText(fmt.Sprintf("%g", baseline.CapacityValue))
	}
	state.capacityUnit.SetText(baseline.CapacityUnit)
	state.manager.SetText(baseline.Manager)
	state.contact.SetText(baseline.Contact)
	state.image.SetText(baseline.Image)
	state.remark.SetText(baseline.Remark)
	ui.refreshWarehouseImageActions()
	markDirty := func() {
		if state.initializing || state.busy || state.submitted {
			return
		}
		state.dirty = true
		state.info.SetText("存在尚未提交的修改。")
	}
	for _, edit := range []*walk.LineEdit{state.name, state.code, state.address, state.capacity, state.capacityUnit, state.manager, state.contact, state.remark} {
		edit.TextChanged().Attach(markDirty)
	}
	state.typeValue.CurrentIndexChanged().Attach(markDirty)
	state.parent.CurrentIndexChanged().Attach(markDirty)
	state.initializing = false
	state.dirty = false
	state.name.SetFocus()
	if state.kind.Key == "warehouse" {
		state.info.SetText("请核对必填项后保存。")
		return
	}
	state.save.SetEnabled(false)
	ui.loadWarehouseParentOptions(state)
}

func (ui *mainUI) loadWarehouseParentOptions(state *warehouseEditorUI) {
	guardedGo(func() {
		options, err := ui.fetchWarehouseParentOptions(state.ctx, state.kind.Key)
		if state.ctx.Err() != nil || state.closed.Load() {
			return
		}
		ui.window.Synchronize(func() {
			if state != ui.warehouseEditor || state.closed.Load() {
				return
			}
			if err != nil {
				state.info.SetText("父级选项加载失败：" + err.Error() + "。请关闭后重试。")
				_ = state.parent.SetModel([]string{"父级加载失败"})
				state.parent.SetCurrentIndex(0)
				return
			}
			state.initializing = true
			state.parentOptions = options
			labels := make([]string, len(options)+1)
			labels[0] = "请选择父级"
			for index, option := range options {
				labels[index+1] = option.Label + " · " + displayMaterialValue(option.Status)
			}
			_ = state.parent.SetModel(labels)
			state.parent.SetCurrentIndex(warehouseParentIndex(options, state.defaultParentID))
			state.dependenciesReady = true
			state.initializing = false
			state.save.SetEnabled(!state.busy && !state.submitted)
			if len(options) == 0 {
				state.info.SetText("线上没有可选父级，无法新增或编辑该层级。")
			} else {
				state.info.SetText("父级选项已从线上加载；只有激活父级可提交。")
			}
		})
	})
}

func warehouseParentIndex(options []warehouseParentOption, id string) int {
	for index, option := range options {
		if option.ID == id {
			return index + 1
		}
	}
	return 0
}

func (ui *mainUI) fetchWarehouseParentOptions(ctx context.Context, kind string) ([]warehouseParentOption, error) {
	var options []warehouseParentOption
	for page := 1; ; page++ {
		switch kind {
		case "zone":
			result, err := ui.session.Client.Warehouses(ctx, page, 100, api.WarehouseFilters{})
			if err != nil {
				return nil, err
			}
			for _, item := range result.List {
				options = append(options, warehouseParentOption{ID: item.ID, Label: businessOptionLabel(item.Name, item.Code), Status: item.Status})
			}
			if int64(len(options)) >= result.Total || len(result.List) == 0 {
				return options, nil
			}
		case "rack":
			result, err := ui.session.Client.WarehouseZones(ctx, page, 100, api.WarehouseFilters{})
			if err != nil {
				return nil, err
			}
			for _, item := range result.List {
				options = append(options, warehouseParentOption{ID: item.ID, Label: strings.Trim(strings.Join([]string{item.WarehouseName, businessOptionLabel(item.Name, item.Code)}, " / "), " /"), Status: item.Status})
			}
			if int64(len(options)) >= result.Total || len(result.List) == 0 {
				return options, nil
			}
		case "bin":
			result, err := ui.session.Client.WarehouseRacks(ctx, page, 100, api.WarehouseFilters{})
			if err != nil {
				return nil, err
			}
			for _, item := range result.List {
				label := strings.Trim(strings.Join([]string{item.WarehouseName, item.WarehouseZoneName, businessOptionLabel(item.Name, item.Code)}, " / "), " /")
				options = append(options, warehouseParentOption{ID: item.ID, Label: label, Status: item.Status})
			}
			if int64(len(options)) >= result.Total || len(result.List) == 0 {
				return options, nil
			}
		default:
			return nil, nil
		}
	}
}

func (ui *mainUI) closeWarehouseEditor(force bool) {
	state := ui.warehouseEditor
	if state == nil || state.tab == nil || ui.tabs == nil {
		return
	}
	if !force {
		if state.busy {
			walk.MsgBox(ui.window, "正在提交", "资料正在提交和复核，请等待完成。", walk.MsgBoxIconInformation)
			return
		}
		if state.dirty && walk.MsgBox(ui.window, "放弃未提交修改", "当前仓储资料还有未提交修改，是否放弃？", walk.MsgBoxYesNo|walk.MsgBoxIconWarning) != walk.DlgCmdYes {
			return
		}
	}
	if index := ui.tabs.Pages().Index(state.tab); index >= 0 {
		if err := ui.tabs.Pages().RemoveAt(index); err != nil {
			walk.MsgBox(ui.window, "无法关闭编辑页", err.Error(), walk.MsgBoxIconError)
			return
		}
	}
	state.dispose()
	state.tab.Dispose()
	ui.warehouseEditor = nil
	ui.warehouseEditorTab = nil
	ui.syncNavigationFromTab()
}

func (ui *mainUI) uploadWarehouseImage() {
	state := ui.warehouseEditor
	if state == nil {
		return
	}
	if state.uploadCancel != nil {
		state.uploadCancel()
		state.upload.SetText("正在取消")
		state.upload.SetEnabled(false)
		state.info.SetText("正在取消图片上传……")
		return
	}
	if state.busy || state.submitted {
		return
	}
	dialog := new(walk.FileDialog)
	dialog.Title = "选择" + state.kind.Label + "图片"
	dialog.Filter = "图片文件 (*.png;*.jpg;*.jpeg;*.gif;*.svg)|*.png;*.jpg;*.jpeg;*.gif;*.svg|所有文件 (*.*)|*.*"
	if ok, err := dialog.ShowOpen(ui.window); err != nil {
		walk.MsgBox(ui.window, "无法选择图片", err.Error(), walk.MsgBoxIconError)
		return
	} else if !ok {
		return
	}
	previous := state.image.Text()
	state.busy = true
	uploadCtx, uploadCancel := context.WithCancel(state.ctx)
	state.uploadCancel = uploadCancel
	ui.setWarehouseEditorEnabled(false)
	state.upload.SetText("取消上传")
	state.upload.SetEnabled(true)
	state.info.SetText("正在上传图片；失败时将保留原图片。")
	guardedGo1(dialog.FilePath, func(filePath string) {
		defer uploadCancel()
		reference, err := ui.session.Client.UploadImage(uploadCtx, filePath)
		if state.ctx.Err() != nil || state.closed.Load() {
			return
		}
		canceled := uploadCtx.Err() != nil
		ui.window.Synchronize(func() {
			if state != ui.warehouseEditor || state.closed.Load() {
				return
			}
			state.uploadCancel = nil
			state.busy = false
			ui.setWarehouseEditorEnabled(true)
			state.upload.SetText("上传图片")
			if canceled {
				state.image.SetText(previous)
				state.info.SetText("图片上传已取消，原图片已保留。")
				return
			}
			if err != nil {
				state.image.SetText(previous)
				state.info.SetText("图片上传失败：" + err.Error() + "。原图片已保留，可重新选择。")
				return
			}
			state.image.SetText(reference)
			state.dirty = true
			ui.refreshWarehouseImageActions()
			state.info.SetText("图片已上传并加入当前资料；保存后才会关联到仓储位置。")
		})
	})
}

func (ui *mainUI) selectWarehouseImage() {
	state := ui.warehouseEditor
	if state == nil || state.busy || state.submitted {
		return
	}
	references, ok := SelectImageAssets(ui.window, ui.session.Client, config.ImageBaseURL(), 1)
	if !ok || len(references) == 0 {
		return
	}
	state.image.SetText(references[0])
	state.dirty = true
	ui.refreshWarehouseImageActions()
	state.info.SetText("已选择线上图片素材，保存资料后生效。")
}

func (ui *mainUI) previewWarehouseImage() {
	state := ui.warehouseEditor
	if state == nil || strings.TrimSpace(state.image.Text()) == "" {
		return
	}
	ShowOrderAttachments(ui.window, ui.session.Client, config.ImageBaseURL(), state.kind.Label+" "+state.name.Text(), []string{state.image.Text()})
}

func (ui *mainUI) clearWarehouseImage() {
	state := ui.warehouseEditor
	if state == nil || state.busy || state.submitted || strings.TrimSpace(state.image.Text()) == "" {
		return
	}
	state.image.SetText("")
	state.dirty = true
	ui.refreshWarehouseImageActions()
	state.info.SetText("图片已从当前提交内容清除；服务端图片库文件不会被删除。")
}

func (ui *mainUI) refreshWarehouseImageActions() {
	state := ui.warehouseEditor
	if state == nil || state.image == nil {
		return
	}
	hasImage := strings.TrimSpace(state.image.Text()) != ""
	state.preview.SetEnabled(hasImage && !state.busy)
	state.clearImage.SetEnabled(hasImage && !state.busy && !state.submitted)
}

func (ui *mainUI) setWarehouseEditorEnabled(enabled bool) {
	state := ui.warehouseEditor
	if state == nil {
		return
	}
	canEdit := enabled && !state.submitted
	for _, edit := range []*walk.LineEdit{state.name, state.code, state.address, state.capacity, state.capacityUnit, state.manager, state.contact, state.remark} {
		edit.SetEnabled(canEdit)
	}
	state.parent.SetEnabled(canEdit)
	state.typeValue.SetEnabled(canEdit)
	state.upload.SetEnabled(canEdit)
	state.selectImage.SetEnabled(canEdit)
	state.save.SetEnabled(canEdit && state.dependenciesReady)
	state.cancelButton.SetEnabled(enabled)
	ui.refreshWarehouseImageActions()
}

func collectWarehouseEditorValues(state *warehouseEditorUI) (warehouseEditorValues, error) {
	capacity := 0.0
	if text := strings.TrimSpace(state.capacity.Text()); text != "" {
		value, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return warehouseEditorValues{}, fmt.Errorf("容量必须是非负数字")
		}
		capacity = value
	}
	values := warehouseEditorValues{
		Type: state.typeValue.Text(), Name: state.name.Text(), Code: state.code.Text(), Address: state.address.Text(),
		Capacity: capacity, CapacityUnit: state.capacityUnit.Text(), Manager: state.manager.Text(), Contact: state.contact.Text(),
		Image: state.image.Text(), Remark: state.remark.Text(),
	}
	if state.kind.Key != "warehouse" {
		index := state.parent.CurrentIndex() - 1
		if index >= 0 && index < len(state.parentOptions) {
			values.ParentID = state.parentOptions[index].ID
			values.ParentStatus = state.parentOptions[index].Status
		}
	}
	return values, validateWarehouseEditorValues(state.kind, values)
}

func validateWarehouseEditorValues(kind warehouseKind, values warehouseEditorValues) error {
	if kind.Key != "warehouse" {
		if strings.TrimSpace(values.ParentID) == "" {
			return fmt.Errorf("请选择所属父级")
		}
		if values.ParentStatus != "激活" {
			return fmt.Errorf("所选父级状态为“%s”，只有激活父级可以提交", displayMaterialValue(values.ParentStatus))
		}
	}
	if len(kind.Types) > 0 && stringIndex(kind.Types, strings.TrimSpace(values.Type)) < 0 {
		return fmt.Errorf("请选择有效类型")
	}
	if strings.TrimSpace(values.Name) == "" {
		return fmt.Errorf("名称不能为空")
	}
	if strings.TrimSpace(values.Code) == "" {
		return fmt.Errorf("编号不能为空")
	}
	if values.Capacity < 0 {
		return fmt.Errorf("容量不能小于 0")
	}
	if contact := strings.TrimSpace(values.Contact); contact != "" && !mobilePattern.MatchString(contact) {
		return fmt.Errorf("联系电话必须是有效手机号")
	}
	if (kind.Key == "zone" || kind.Key == "rack") && strings.TrimSpace(values.Image) == "" {
		return fmt.Errorf("%s图片为接口必填项", kind.Label)
	}
	return nil
}

func (ui *mainUI) submitWarehouseEditor() {
	state := ui.warehouseEditor
	if state == nil || state.busy || state.submitted {
		return
	}
	permission := state.kind.AddPermission
	if state.mode == "edit" {
		permission = state.kind.EditPermission
	}
	if !hasButton(ui.session.Perms.Buttons, permission) {
		state.info.SetText("当前账号缺少本次操作权限，未发送请求。")
		return
	}
	values, err := collectWarehouseEditorValues(state)
	if err != nil {
		state.info.SetText("无法提交：" + err.Error())
		return
	}
	action := "新增"
	if state.mode == "edit" {
		action = "编辑"
	}
	message := fmt.Sprintf("操作：%s%s\r\n编号：%s\r\n名称：%s", action, state.kind.Label, strings.TrimSpace(values.Code), strings.TrimSpace(values.Name))
	if values.ParentID != "" {
		message += "\r\n父级：" + state.parent.Text()
	}
	message += "\r\n\r\n是否提交到线上服务？"
	if walk.MsgBox(ui.window, "核对仓储资料", message, walk.MsgBoxYesNo|walk.MsgBoxIconQuestion) != walk.DlgCmdYes {
		return
	}
	state.busy = true
	ui.setWarehouseEditorEnabled(false)
	state.info.SetText("正在重新读取线上资料并提交……")
	guardedGo(func() {
		requestErr := ui.writeWarehouseEntity(state.ctx, state.kind, state.mode, state.baseline, values)
		var verifyErr error
		if requestErr == nil {
			verifyErr = ui.verifyWarehouseWrite(state.ctx, state.kind, state.baseline.ID, values.Code, values.Name)
		}
		if state.ctx.Err() != nil || state.closed.Load() {
			return
		}
		ui.window.Synchronize(func() {
			if state != ui.warehouseEditor || state.closed.Load() {
				return
			}
			state.busy = false
			if requestErr != nil {
				ui.setWarehouseEditorEnabled(true)
				state.info.SetText("提交失败：" + requestErr.Error() + "。请刷新线上资料后重试。")
				return
			}
			state.dirty = false
			if verifyErr != nil {
				state.submitted = true
				ui.setWarehouseEditorEnabled(false)
				state.cancelButton.SetEnabled(true)
				state.info.SetText("服务端已返回成功，但回读失败：" + verifyErr.Error() + "。请勿重复提交，关闭后刷新列表核对。")
				walk.MsgBox(ui.window, "提交后复核失败", state.info.Text(), walk.MsgBoxIconWarning)
				ui.loadWarehouseTree()
				ui.loadWarehouseDirectory()
				return
			}
			ui.notifyStatusSuccess("仓储资料已保存，并完成线上回读复核。")
			ui.loadWarehouseTree()
			ui.loadWarehouseDirectory()
			ui.loadFilterOptions()
			ui.closeWarehouseEditor(true)
		})
	})
}

func (ui *mainUI) writeWarehouseEntity(ctx context.Context, kind warehouseKind, mode string, baseline warehouseDetail, values warehouseEditorValues) error {
	if mode == "edit" {
		updatedAt, status, found, err := ui.warehouseCurrent(ctx, kind, baseline.ID, baseline.Code)
		if err != nil {
			return fmt.Errorf("提交前重新读取失败：%w", err)
		}
		if !found {
			return fmt.Errorf("资料已不存在")
		}
		if status != "激活" {
			return fmt.Errorf("资料状态已变为“%s”，不能继续编辑", status)
		}
		if baseline.UpdatedAt > 0 && updatedAt != baseline.UpdatedAt {
			return fmt.Errorf("资料已被其他用户更新，请关闭编辑页并刷新后重试")
		}
	}
	switch kind.Key {
	case "warehouse":
		request := api.WarehouseRequest{ID: baseline.ID, Type: strings.TrimSpace(values.Type), Name: strings.TrimSpace(values.Name), Code: strings.TrimSpace(values.Code), Image: strings.TrimSpace(values.Image), Address: strings.TrimSpace(values.Address), Capacity: values.Capacity, CapacityUnit: strings.TrimSpace(values.CapacityUnit), Manager: strings.TrimSpace(values.Manager), Contact: strings.TrimSpace(values.Contact), Remark: strings.TrimSpace(values.Remark)}
		if mode == "add" {
			return ui.session.Client.CreateWarehouse(ctx, request)
		}
		return ui.session.Client.UpdateWarehouse(ctx, request)
	case "zone":
		request := api.WarehouseZoneRequest{ID: baseline.ID, WarehouseID: values.ParentID, Name: strings.TrimSpace(values.Name), Code: strings.TrimSpace(values.Code), Image: strings.TrimSpace(values.Image), Capacity: values.Capacity, CapacityUnit: strings.TrimSpace(values.CapacityUnit), Manager: strings.TrimSpace(values.Manager), Contact: strings.TrimSpace(values.Contact), Remark: strings.TrimSpace(values.Remark)}
		if mode == "add" {
			return ui.session.Client.CreateWarehouseZone(ctx, request)
		}
		return ui.session.Client.UpdateWarehouseZone(ctx, request)
	case "rack":
		request := api.WarehouseRackRequest{ID: baseline.ID, WarehouseZoneID: values.ParentID, Type: strings.TrimSpace(values.Type), Name: strings.TrimSpace(values.Name), Code: strings.TrimSpace(values.Code), Image: strings.TrimSpace(values.Image), Capacity: values.Capacity, CapacityUnit: strings.TrimSpace(values.CapacityUnit), Manager: strings.TrimSpace(values.Manager), Contact: strings.TrimSpace(values.Contact), Remark: strings.TrimSpace(values.Remark)}
		if mode == "add" {
			return ui.session.Client.CreateWarehouseRack(ctx, request)
		}
		return ui.session.Client.UpdateWarehouseRack(ctx, request)
	case "bin":
		request := api.WarehouseBinRequest{ID: baseline.ID, WarehouseRackID: values.ParentID, Name: strings.TrimSpace(values.Name), Code: strings.TrimSpace(values.Code), Image: strings.TrimSpace(values.Image), Capacity: values.Capacity, CapacityUnit: strings.TrimSpace(values.CapacityUnit), Manager: strings.TrimSpace(values.Manager), Contact: strings.TrimSpace(values.Contact), Remark: strings.TrimSpace(values.Remark)}
		if mode == "add" {
			return ui.session.Client.CreateWarehouseBin(ctx, request)
		}
		return ui.session.Client.UpdateWarehouseBin(ctx, request)
	default:
		return fmt.Errorf("不支持的仓储层级：%s", kind.Label)
	}
}

func (ui *mainUI) warehouseCurrent(ctx context.Context, kind warehouseKind, id, code string) (int64, string, bool, error) {
	switch kind.Key {
	case "warehouse":
		item, found, err := ui.session.Client.FindWarehouse(ctx, id, code)
		return item.UpdatedAt, item.Status, found, err
	case "zone":
		item, found, err := ui.session.Client.FindWarehouseZone(ctx, id, code)
		return item.UpdatedAt, item.Status, found, err
	case "rack":
		item, found, err := ui.session.Client.FindWarehouseRack(ctx, id, code)
		return item.UpdatedAt, item.Status, found, err
	case "bin":
		item, found, err := ui.session.Client.FindWarehouseBin(ctx, id, code)
		return item.UpdatedAt, item.Status, found, err
	default:
		return 0, "", false, fmt.Errorf("不支持的仓储层级")
	}
}

func (ui *mainUI) verifyWarehouseWrite(ctx context.Context, kind warehouseKind, id, code, name string) error {
	if strings.TrimSpace(id) != "" {
		_, _, found, err := ui.warehouseCurrent(ctx, kind, id, code)
		if err != nil {
			return err
		}
		if !found {
			return fmt.Errorf("未查询到编号 %s 的资料", code)
		}
		return nil
	}
	result, err := ui.loadWarehouseRows(ctx, kind, 1, 100, api.WarehouseFilters{Code: code})
	if err != nil {
		return err
	}
	for _, row := range result.rows {
		if strings.EqualFold(strings.TrimSpace(row.Code), strings.TrimSpace(code)) && strings.TrimSpace(row.Name) == strings.TrimSpace(name) {
			return nil
		}
	}
	return fmt.Errorf("未查询到新增后的资料 %s", code)
}

func warehouseStatuses() []string { return []string{"激活", "禁用", "盘点中", "关闭"} }

func (ui *mainUI) changeSelectedWarehouseStatus() {
	detail, kind, ok := ui.selectedWarehouse()
	if !ok {
		walk.MsgBox(ui.window, "请选择仓储资料", "请先选择需要变更状态的仓储资料。", walk.MsgBoxIconInformation)
		return
	}
	if !hasButton(ui.session.Perms.Buttons, kind.StatusPermission) {
		walk.MsgBox(ui.window, "没有权限", "当前账号没有变更该仓储层级状态的权限。", walk.MsgBoxIconWarning)
		return
	}
	statuses := warehouseStatuses()
	var dlg *walk.Dialog
	var target *walk.ComboBox
	var okButton *walk.PushButton
	accepted := false
	err := Dialog{
		AssignTo: &dlg, Title: "变更" + kind.Label + "状态", DefaultButton: &okButton,
		MinSize: Size{Width: 480, Height: 260}, Size: Size{Width: 520, Height: 300},
		Layout: VBox{Margins: Margins{Left: 20, Top: 18, Right: 20, Bottom: 18}, Spacing: 12},
		Children: []Widget{
			Label{Text: detail.Name, Font: Font{Family: "Microsoft YaHei UI", PointSize: 13, Bold: true}},
			Label{Text: "当前状态：" + displayMaterialValue(detail.Status)},
			GroupBox{Title: "目标状态", Layout: VBox{Margins: Margins{Left: 12, Top: 10, Right: 12, Bottom: 12}}, Children: []Widget{
				ComboBox{AssignTo: &target, Model: statuses, CurrentIndex: stringIndex(statuses, detail.Status), MinSize: Size{Height: 30}},
			}},
			Label{Text: "本阶段不提供删除状态；下级和父级约束由服务端最终校验。", TextColor: secondaryTextColor()},
			Composite{Layout: HBox{Spacing: 8}, Children: []Widget{HSpacer{}, PushButton{Text: "取消", OnClicked: func() { dlg.Cancel() }}, PushButton{AssignTo: &okButton, Text: "继续", OnClicked: func() { accepted = true; dlg.Accept() }}}},
		},
	}.Create(ui.window)
	if err != nil {
		walk.MsgBox(ui.window, "无法打开状态窗口", err.Error(), walk.MsgBoxIconError)
		return
	}
	if target.CurrentIndex() < 0 {
		target.SetCurrentIndex(0)
	}
	dlg.Run()
	if !accepted {
		return
	}
	status := target.Text()
	if status == detail.Status {
		walk.MsgBox(ui.window, "状态未变化", "请选择不同于当前状态的目标状态。", walk.MsgBoxIconInformation)
		return
	}
	if stringIndex(statuses, status) < 0 {
		walk.MsgBox(ui.window, "状态无效", "请选择接口允许的非删除状态。", walk.MsgBoxIconWarning)
		return
	}
	if walk.MsgBox(ui.window, "确认变更状态", fmt.Sprintf("%s：%s\r\n当前状态：%s\r\n目标状态：%s\r\n\r\n是否提交到线上服务？", kind.Label, detail.Name, detail.Status, status), walk.MsgBoxYesNo|walk.MsgBoxIconWarning) != walk.DlgCmdYes {
		return
	}
	state := ui.warehouse
	state.busy = true
	ui.updateWarehouseActions()
	guardedGo(func() {
		updatedAt, _, found, requestErr := ui.warehouseCurrent(context.Background(), kind, detail.ID, detail.Code)
		if requestErr == nil && !found {
			requestErr = fmt.Errorf("资料已不存在")
		}
		if requestErr == nil && detail.UpdatedAt > 0 && updatedAt != detail.UpdatedAt {
			requestErr = fmt.Errorf("资料已被其他用户更新，请刷新后重试")
		}
		if requestErr == nil {
			switch kind.Key {
			case "warehouse":
				requestErr = ui.session.Client.UpdateWarehouseStatus(context.Background(), detail.ID, status)
			case "zone":
				requestErr = ui.session.Client.UpdateWarehouseZoneStatus(context.Background(), detail.ID, status)
			case "rack":
				requestErr = ui.session.Client.UpdateWarehouseRackStatus(context.Background(), detail.ID, status)
			case "bin":
				requestErr = ui.session.Client.UpdateWarehouseBinStatus(context.Background(), detail.ID, status)
			}
		}
		if requestErr == nil {
			_, actual, found, err := ui.warehouseCurrent(context.Background(), kind, detail.ID, detail.Code)
			if err != nil {
				requestErr = err
			} else if !found {
				requestErr = fmt.Errorf("回读时未找到资料")
			} else if actual != status {
				requestErr = fmt.Errorf("回读状态为 %s", actual)
			}
		}
		ui.window.Synchronize(func() {
			if state != ui.warehouse {
				return
			}
			state.busy = false
			ui.updateWarehouseActions()
			if requestErr != nil {
				state.info.SetText("状态变更失败或复核失败：" + requestErr.Error() + "。请刷新后核对，勿重复提交。")
				walk.MsgBox(ui.window, "状态变更未确认", state.info.Text(), walk.MsgBoxIconWarning)
				return
			}
			ui.notifyStatusSuccess("仓储资料状态已变更，并完成线上回读复核。")
			ui.loadWarehouseTree()
			ui.loadWarehouseDirectory()
			ui.loadFilterOptions()
		})
	})
}
