package ui

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"

	"zhengshi-wms-windowsapp/internal/api"
	"zhengshi-wms-windowsapp/internal/config"
)

type materialEditorUI struct {
	tab              *walk.TabPage
	mode             string
	baseline         api.Material
	baseTitle        string
	category         *walk.ComboBox
	name             *walk.LineEdit
	model            *walk.LineEdit
	image            *walk.LineEdit
	selectImage      *walk.PushButton
	upload           *walk.PushButton
	preview          *walk.PushButton
	material         *walk.LineEdit
	specification    *walk.LineEdit
	surfaceTreatment *walk.LineEdit
	strengthGrade    *walk.LineEdit
	quantity         *walk.LineEdit
	unit             *walk.LineEdit
	price            *walk.LineEdit
	remark           *walk.LineEdit
	info             *walk.Label
	save             *walk.PushButton
	cancelButton     *walk.PushButton
	categories       []selectOption
	dirty            bool
	initializing     bool
	busy             bool
	submitted        bool
	ctx              context.Context
	cancel           context.CancelFunc
	uploadCancel     context.CancelFunc
	closed           atomic.Bool
}

func newMaterialEditorUI(mode string, material api.Material) *materialEditorUI {
	ctx, cancel := context.WithCancel(context.Background())
	state := &materialEditorUI{mode: mode, baseline: material, initializing: true, ctx: ctx, cancel: cancel}
	state.baseTitle = "新增物料"
	if mode == "edit" {
		state.baseTitle = "编辑物料 · " + displayMaterialValue(material.Model)
	}
	return state
}

func (state *materialEditorUI) dispose() {
	if state == nil || state.closed.Swap(true) {
		return
	}
	if state.cancel != nil {
		state.cancel()
	}
}

func (state *materialEditorUI) setDirty(dirty bool) {
	if state == nil {
		return
	}
	state.dirty = dirty
	if state.tab != nil {
		title := state.baseTitle
		if dirty {
			title += " *"
		}
		_ = state.tab.SetTitle(closableTabTitle(title))
	}
}

func (ui *mainUI) materialEditorPageWidget(state *materialEditorUI) TabPage {
	isAdd := state.mode == "add"
	return TabPage{
		AssignTo: &state.tab,
		Title:    closableTabTitle(state.baseTitle),
		Layout:   VBox{Margins: Margins{Left: 14, Top: 14, Right: 14, Bottom: 12}, Spacing: 10},
		Children: []Widget{
			Label{Text: state.baseTitle, Font: Font{Family: "Microsoft YaHei UI", PointSize: 15, Bold: true}},
			Label{Text: "字段直接映射现有物料接口；编辑时价格保持只读历史，不随物料 PUT 请求修改。", TextColor: secondaryTextColor()},
			GroupBox{
				Title:  "基本信息",
				Layout: Grid{Columns: 8, Margins: Margins{Left: 14, Top: 10, Right: 14, Bottom: 12}, Spacing: 8},
				Children: []Widget{
					Label{Text: "物料分类 *"},
					ComboBox{AssignTo: &state.category, Model: []string{"正在加载启用分类……"}, CurrentIndex: 0, MinSize: Size{Width: 190, Height: 28}, Accessibility: Accessibility{Name: "物料分类"}},
					Label{Text: "物料名称 *"},
					LineEdit{AssignTo: &state.name, MinSize: Size{Width: 190, Height: 28}, Accessibility: Accessibility{Name: "物料名称"}},
					Label{Text: "型号 *"},
					LineEdit{AssignTo: &state.model, MinSize: Size{Width: 190, Height: 28}, Accessibility: Accessibility{Name: "物料型号"}},
					Label{Text: "图纸"},
					Composite{Layout: HBox{Spacing: 4}, Children: []Widget{
						LineEdit{AssignTo: &state.image, ReadOnly: true, MinSize: Size{Width: 76, Height: 28}, CueBanner: "未选择"},
						PushButton{AssignTo: &state.selectImage, Text: "素材", MinSize: Size{Width: 52, Height: 28}, OnClicked: ui.selectMaterialEditorImage},
						PushButton{AssignTo: &state.upload, Text: "上传", MinSize: Size{Width: 52, Height: 28}, ToolTipText: "上传期间再次点击可取消", Accessibility: Accessibility{Name: "上传物料图纸图片"}, OnClicked: ui.uploadMaterialEditorImage},
						PushButton{AssignTo: &state.preview, Text: "预览", Enabled: false, MinSize: Size{Width: 52, Height: 28}, OnClicked: ui.previewMaterialEditorImage},
					}},
					Label{Text: "材质"},
					LineEdit{AssignTo: &state.material, MinSize: Size{Width: 190, Height: 28}},
					Label{Text: "规格"},
					LineEdit{AssignTo: &state.specification, MinSize: Size{Width: 190, Height: 28}},
					Label{Text: "表面处理"},
					LineEdit{AssignTo: &state.surfaceTreatment, MinSize: Size{Width: 190, Height: 28}},
					Label{Text: "强度等级"},
					LineEdit{AssignTo: &state.strengthGrade, MinSize: Size{Width: 190, Height: 28}},
					Label{Text: "安全库存"},
					LineEdit{AssignTo: &state.quantity, Text: "0", CueBanner: "非负数字", MinSize: Size{Width: 150, Height: 28}},
					Label{Text: "计量单位"},
					LineEdit{AssignTo: &state.unit, MinSize: Size{Width: 150, Height: 28}},
					Label{Text: "初始单价", Visible: isAdd},
					LineEdit{AssignTo: &state.price, Text: "0", Visible: isAdd, CueBanner: "仅新增时提交", MinSize: Size{Width: 150, Height: 28}},
					Label{Text: "备注"},
					LineEdit{AssignTo: &state.remark, ColumnSpan: 3, MinSize: Size{Height: 28}},
				},
			},
			GroupBox{
				Title:  "提交保护",
				Layout: VBox{Margins: Margins{Left: 14, Top: 10, Right: 14, Bottom: 12}, Spacing: 6},
				Children: []Widget{
					Label{Text: "编辑提交前会重新读取线上更新时间；提交后再次读取物料详情。写操作不会自动重试。", TextColor: secondaryTextColor()},
					Label{Text: "新增物料的初始单价由现有 POST 接口处理；后续价格请进入报价或核价流程。", TextColor: secondaryTextColor()},
				},
			},
			VSpacer{},
			Composite{Layout: HBox{Spacing: 8}, Children: []Widget{
				Label{AssignTo: &state.info, Text: "正在加载启用的物料分类……", TextColor: secondaryTextColor(), Accessibility: Accessibility{Name: "物料编辑状态"}},
				HSpacer{},
				PushButton{AssignTo: &state.cancelButton, Text: "取消", MinSize: Size{Width: 80, Height: 30}, OnClicked: func() { ui.closeMaterialEditor(false) }},
				PushButton{AssignTo: &state.save, Text: "核对并保存", Enabled: false, MinSize: Size{Width: 110, Height: 30}, OnClicked: ui.submitMaterialEditor},
			}},
		},
	}
}

func (ui *mainUI) newMaterial() {
	if !hasButton(ui.session.Perms.Buttons, "material:material:add") {
		walk.MsgBox(ui.window, "没有权限", "当前账号没有新增物料权限。", walk.MsgBoxIconWarning)
		return
	}
	ui.openMaterialEditor("add", api.Material{})
}

func (ui *mainUI) editSelectedMaterial() {
	if !hasButton(ui.session.Perms.Buttons, "material:material:edit") {
		walk.MsgBox(ui.window, "没有权限", "当前账号没有编辑物料权限。", walk.MsgBoxIconWarning)
		return
	}
	material, ok := ui.selectedMaterial()
	if !ok {
		walk.MsgBox(ui.window, "请选择物料", "请先在物料表格中选择一行。", walk.MsgBoxIconInformation)
		return
	}
	ui.openMaterialEditor("edit", material)
}

func (ui *mainUI) selectedMaterial() (api.Material, bool) {
	if ui.materialTable == nil {
		return api.Material{}, false
	}
	index := ui.materialTable.CurrentIndex()
	if index < 0 || index >= len(ui.materialRows) {
		return api.Material{}, false
	}
	return ui.materialRows[index].Detail, true
}

func (ui *mainUI) updateMaterialActionButtons() {
	if ui.materialEdit == nil {
		return
	}
	_, selected := ui.selectedMaterial()
	ui.materialEdit.SetEnabled(selected && hasButton(ui.session.Perms.Buttons, "material:material:edit"))
}

func (ui *mainUI) openMaterialEditor(mode string, material api.Material) {
	if current := ui.materialEditor; current != nil {
		if current.busy {
			walk.MsgBox(ui.window, "正在提交", "当前物料正在提交和复核，请等待完成。", walk.MsgBoxIconInformation)
			return
		}
		if current.mode == mode && current.baseline.ID == material.ID {
			_ = ui.tabs.SetCurrentIndex(ui.tabs.Pages().Index(current.tab))
			return
		}
		if current.dirty && walk.MsgBox(ui.window, "替换未提交编辑", "当前物料还有未提交修改，是否放弃并打开另一项？", walk.MsgBoxYesNo|walk.MsgBoxIconWarning) != walk.DlgCmdYes {
			return
		}
		ui.closeMaterialEditor(true)
	}
	state := newMaterialEditorUI(mode, material)
	pageDecl := ui.materialEditorPageWidget(state)
	if err := pageDecl.Create(NewBuilder(nil)); err != nil {
		state.dispose()
		walk.MsgBox(ui.window, "无法打开物料编辑页", err.Error(), walk.MsgBoxIconError)
		return
	}
	ui.materialEditor = state
	ui.materialEditorTab = state.tab
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
		ui.materialEditor = nil
		ui.materialEditorTab = nil
		walk.MsgBox(ui.window, "无法打开物料编辑页", err.Error(), walk.MsgBoxIconError)
		return
	}
	_ = ui.tabs.SetCurrentIndex(ui.tabs.Pages().Index(state.tab))
	ui.initializeMaterialEditor(state)
}

func (ui *mainUI) initializeMaterialEditor(state *materialEditorUI) {
	if state == nil || state != ui.materialEditor {
		return
	}
	state.name.SetText(state.baseline.Name)
	state.model.SetText(state.baseline.Model)
	state.image.SetText(state.baseline.Image)
	state.material.SetText(state.baseline.Material)
	state.specification.SetText(state.baseline.Specification)
	state.surfaceTreatment.SetText(state.baseline.SurfaceTreatment)
	state.strengthGrade.SetText(state.baseline.StrengthGrade)
	state.quantity.SetText(fmt.Sprintf("%g", state.baseline.Quantity))
	state.unit.SetText(state.baseline.Unit)
	state.remark.SetText(state.baseline.Remark)
	state.preview.SetEnabled(strings.TrimSpace(state.baseline.Image) != "")
	markDirty := func() {
		if state.initializing || state.busy || state.submitted {
			return
		}
		state.setDirty(true)
		state.info.SetText("存在尚未提交的修改。")
	}
	for _, edit := range []*walk.LineEdit{state.name, state.model, state.material, state.specification, state.surfaceTreatment, state.strengthGrade, state.quantity, state.unit, state.price, state.remark} {
		edit.TextChanged().Attach(markDirty)
	}
	state.category.CurrentIndexChanged().Attach(markDirty)
	state.initializing = false
	state.setDirty(false)
	ui.loadMaterialEditorCategories(state)
}

func enabledMaterialCategoryOptions(categories []api.MaterialCategory, parent string) []selectOption {
	var result []selectOption
	for _, category := range categories {
		label := category.Name
		if parent != "" {
			label = parent + " / " + category.Name
		}
		if strings.TrimSpace(category.Status) == "" || category.Status == "启用" {
			result = append(result, selectOption{ID: category.ID, Label: label})
		}
		result = append(result, enabledMaterialCategoryOptions(category.Children, label)...)
	}
	return result
}

func (ui *mainUI) loadMaterialEditorCategories(state *materialEditorUI) {
	guardedGo(func() {
		categories, err := ui.session.Client.MaterialCategories(state.ctx)
		if state.ctx.Err() != nil || state.closed.Load() {
			return
		}
		ui.window.Synchronize(func() {
			if state != ui.materialEditor || state.closed.Load() {
				return
			}
			if err != nil {
				state.info.SetText("物料分类加载失败：" + err.Error())
				return
			}
			state.initializing = true
			state.categories = enabledMaterialCategoryOptions(categories, "")
			_ = state.category.SetModel(optionLabels("请选择启用分类", state.categories))
			state.category.SetCurrentIndex(optionIndexByID(state.categories, state.baseline.CategoryID))
			state.initializing = false
			state.save.SetEnabled(len(state.categories) > 0)
			state.info.SetText(fmt.Sprintf("已加载 %d 个启用分类。", len(state.categories)))
		})
	})
}

func (ui *mainUI) closeMaterialEditor(force bool) {
	state := ui.materialEditor
	if state == nil || state.tab == nil || ui.tabs == nil {
		return
	}
	if !force {
		if state.busy {
			walk.MsgBox(ui.window, "正在提交", "物料正在提交和复核，请等待完成。", walk.MsgBoxIconInformation)
			return
		}
		if state.dirty && walk.MsgBox(ui.window, "放弃未提交修改", "当前物料还有未提交修改，是否放弃？", walk.MsgBoxYesNo|walk.MsgBoxIconWarning) != walk.DlgCmdYes {
			return
		}
	}
	index := ui.tabs.Pages().Index(state.tab)
	if index >= 0 {
		if err := ui.tabs.Pages().RemoveAt(index); err != nil {
			walk.MsgBox(ui.window, "无法关闭物料编辑页", err.Error(), walk.MsgBoxIconError)
			return
		}
	}
	state.dispose()
	state.tab.Dispose()
	ui.materialEditor = nil
	ui.materialEditorTab = nil
	ui.syncNavigationFromTab()
}

func (ui *mainUI) uploadMaterialEditorImage() {
	state := ui.materialEditor
	if state == nil {
		return
	}
	if state.uploadCancel != nil {
		state.uploadCancel()
		state.upload.SetText("正在取消")
		state.upload.SetEnabled(false)
		state.info.SetText("正在取消图纸上传……")
		return
	}
	if state.busy {
		return
	}
	dialog := new(walk.FileDialog)
	dialog.Title = "选择物料图纸图片"
	dialog.Filter = "图片文件 (*.png;*.jpg;*.jpeg;*.gif)|*.png;*.jpg;*.jpeg;*.gif"
	if ok, err := dialog.ShowOpen(ui.window); err != nil || !ok {
		return
	}
	state.busy = true
	uploadCtx, uploadCancel := context.WithCancel(state.ctx)
	state.uploadCancel = uploadCancel
	state.upload.SetText("取消上传")
	state.upload.SetEnabled(true)
	state.save.SetEnabled(false)
	state.info.SetText("正在上传图纸图片……")
	guardedGo1(dialog.FilePath, func(filePath string) {
		defer uploadCancel()
		reference, err := ui.session.Client.UploadImage(uploadCtx, filePath)
		if state.ctx.Err() != nil || state.closed.Load() {
			return
		}
		canceled := uploadCtx.Err() != nil
		ui.window.Synchronize(func() {
			if state != ui.materialEditor || state.closed.Load() {
				return
			}
			state.uploadCancel = nil
			state.busy = false
			state.upload.SetText("上传")
			state.upload.SetEnabled(true)
			state.save.SetEnabled(len(state.categories) > 0)
			if canceled {
				state.info.SetText("图纸上传已取消，原值未改变。")
				return
			}
			if err != nil {
				state.info.SetText("图纸上传失败，已保留原值：" + err.Error())
				return
			}
			state.image.SetText(reference)
			state.preview.SetEnabled(true)
			state.setDirty(true)
			state.info.SetText("图纸上传成功，保存物料后生效。")
		})
	})
}

func (ui *mainUI) selectMaterialEditorImage() {
	state := ui.materialEditor
	if state == nil || state.busy || state.submitted {
		return
	}
	references, ok := SelectImageAssets(ui.window, ui.session.Client, config.ImageBaseURL(), 1)
	if !ok || len(references) == 0 {
		return
	}
	state.image.SetText(references[0])
	state.preview.SetEnabled(true)
	state.setDirty(true)
	state.info.SetText("已选择线上图片素材，保存物料后生效。")
}

func (ui *mainUI) previewMaterialEditorImage() {
	state := ui.materialEditor
	if state == nil || strings.TrimSpace(state.image.Text()) == "" {
		return
	}
	material := state.baseline
	material.Name = strings.TrimSpace(state.name.Text())
	material.Model = strings.TrimSpace(state.model.Text())
	material.Image = strings.TrimSpace(state.image.Text())
	ShowMaterialDetail(ui.window, ui.session.Client, config.ImageBaseURL(), material)
}

func (ui *mainUI) materialEditorRequest(state *materialEditorUI) (api.MaterialRequest, error) {
	request := api.MaterialRequest{
		ID: state.baseline.ID, CategoryID: selectedOptionID(state.category, state.categories),
		Name: strings.TrimSpace(state.name.Text()), Model: strings.TrimSpace(state.model.Text()), Image: strings.TrimSpace(state.image.Text()),
		Material: strings.TrimSpace(state.material.Text()), Specification: strings.TrimSpace(state.specification.Text()),
		SurfaceTreatment: strings.TrimSpace(state.surfaceTreatment.Text()), StrengthGrade: strings.TrimSpace(state.strengthGrade.Text()),
		Unit: strings.TrimSpace(state.unit.Text()), Remark: strings.TrimSpace(state.remark.Text()),
	}
	if request.CategoryID == "" {
		return request, fmt.Errorf("请选择启用的物料分类")
	}
	if request.Name == "" {
		return request, fmt.Errorf("请填写物料名称")
	}
	if request.Model == "" {
		return request, fmt.Errorf("请填写物料型号")
	}
	quantity, err := parseNonNegativeNumber(state.quantity.Text(), "安全库存")
	if err != nil {
		return request, err
	}
	request.Quantity = quantity
	if state.mode == "add" {
		price, parseErr := parseNonNegativeNumber(state.price.Text(), "初始单价")
		if parseErr != nil {
			return request, parseErr
		}
		request.Price = price
	}
	return request, nil
}

func (ui *mainUI) setMaterialEditorBusy(state *materialEditorUI, busy bool, message string) {
	state.busy = busy
	state.save.SetEnabled(!busy && len(state.categories) > 0 && !state.submitted)
	state.cancelButton.SetEnabled(!busy)
	state.upload.SetEnabled(!busy)
	state.selectImage.SetEnabled(!busy && !state.submitted)
	state.category.SetEnabled(!busy)
	if message != "" {
		state.info.SetText(message)
	}
}

func (ui *mainUI) submitMaterialEditor() {
	state := ui.materialEditor
	if state == nil || state.busy || state.submitted {
		return
	}
	request, err := ui.materialEditorRequest(state)
	if err != nil {
		state.info.SetText(err.Error())
		return
	}
	action := "新增"
	if state.mode == "edit" {
		action = "更新"
	}
	if walk.MsgBox(ui.window, "确认"+action+"物料", fmt.Sprintf("物料：%s\r\n型号：%s\r\n\r\n提交后将立即写入线上生产数据，是否继续？", request.Name, request.Model), walk.MsgBoxYesNo|walk.MsgBoxIconWarning) != walk.DlgCmdYes {
		return
	}
	ui.setMaterialEditorBusy(state, true, "正在刷新线上物料并提交……")
	guardedGo(func() {
		var writeErr error
		var verified api.Material
		writeApplied := false
		if state.mode == "edit" {
			current, currentErr := ui.session.Client.MaterialInfo(state.ctx, state.baseline.ID)
			if currentErr != nil {
				writeErr = fmt.Errorf("提交前读取线上物料失败：%w", currentErr)
			} else if current.UpdatedAt != state.baseline.UpdatedAt {
				writeErr = fmt.Errorf("线上物料已被其他用户修改，请关闭编辑页并刷新后重试")
			} else if writeErr = ui.session.Client.UpdateMaterial(state.ctx, request); writeErr == nil {
				writeApplied = true
				verified, writeErr = ui.session.Client.MaterialInfo(state.ctx, state.baseline.ID)
				if writeErr != nil {
					writeErr = fmt.Errorf("物料已提交，但在线回读复核失败：%w", writeErr)
				}
			}
		} else {
			writeErr = ui.session.Client.CreateMaterial(state.ctx, request)
			if writeErr == nil {
				writeApplied = true
				page, verifyErr := ui.session.Client.Materials(state.ctx, 1, 100, api.MaterialFilters{Name: request.Name, Model: request.Model})
				if verifyErr != nil {
					writeErr = fmt.Errorf("物料已提交，但在线复核失败：%w", verifyErr)
				} else {
					for _, item := range page.List {
						if strings.EqualFold(strings.TrimSpace(item.Name), request.Name) && strings.EqualFold(strings.TrimSpace(item.Model), request.Model) {
							verified = item
							break
						}
					}
					if verified.ID == "" {
						writeErr = fmt.Errorf("物料已提交，但未能在查询结果中完成在线复核；请刷新物料列表，勿重复提交")
					}
				}
			}
		}
		if state.ctx.Err() != nil || state.closed.Load() {
			return
		}
		ui.window.Synchronize(func() {
			if state != ui.materialEditor || state.closed.Load() {
				return
			}
			if writeErr != nil {
				if writeApplied {
					state.submitted = true
					state.setDirty(false)
					ui.setMaterialEditorBusy(state, false, requestFailureText(writeErr)+"。写入结果未知或已生效，本页已禁止重复提交；请关闭并刷新物料列表核对。")
					walk.MsgBox(ui.window, "提交后复核失败", "写请求已经成功返回，但在线回读复核失败。为避免重复新增或覆盖，本页已禁止再次提交；请关闭编辑页并刷新物料列表核对。", walk.MsgBoxIconWarning)
					return
				}
				ui.setMaterialEditorBusy(state, false, requestFailureText(writeErr))
				return
			}
			state.baseline = verified
			state.submitted = true
			state.setDirty(false)
			ui.setMaterialEditorBusy(state, false, "物料已保存并完成线上回读复核。")
			ui.notifyStatusSuccess("物料已保存，并完成线上回读复核。")
			ui.closeMaterialEditor(true)
			ui.materialPage = 1
			ui.loadMaterials()
		})
	})
}
