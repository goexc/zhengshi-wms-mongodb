package ui

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"

	"zhengshi-wms-windowsapp/internal/api"
	"zhengshi-wms-windowsapp/internal/config"
)

type inboundEditorMaterial struct {
	Index    string
	Name     string
	Model    string
	Unit     string
	Price    string
	Quantity string
	Position string
	Request  api.InboundMaterialRequest
}

type inboundEditorUI struct {
	tab              *walk.TabPage
	mode             string
	baseline         api.InboundReceipt
	code             *walk.LineEdit
	orderType        *walk.ComboBox
	supplierLabel    *walk.Label
	supplier         *walk.ComboBox
	customerLabel    *walk.Label
	customer         *walk.ComboBox
	planDate         *walk.LineEdit
	total            *walk.Label
	remark           *walk.LineEdit
	attachment       *walk.ComboBox
	selectAttachment *walk.PushButton
	upload           *walk.PushButton
	removeAttachment *walk.PushButton
	preview          *walk.PushButton
	materials        *walk.TableView
	addMaterial      *walk.PushButton
	editMaterial     *walk.PushButton
	removeMaterial   *walk.PushButton
	info             *walk.Label
	save             *walk.PushButton
	cancelButton     *walk.PushButton

	rows            []inboundEditorMaterial
	annex           []string
	supplierOptions []selectOption
	customerOptions []selectOption
	positions       []positionOption
	dirty           bool
	initializing    bool
	busy            bool
	submitted       bool
	ctx             context.Context
	cancel          context.CancelFunc
	uploadCancel    context.CancelFunc
	closed          atomic.Bool
}

func newInboundEditorUI(mode string, receipt api.InboundReceipt) *inboundEditorUI {
	ctx, cancel := context.WithCancel(context.Background())
	state := &inboundEditorUI{
		mode: mode, baseline: receipt, ctx: ctx, cancel: cancel, initializing: true,
		annex: append([]string(nil), receipt.Annex...),
	}
	for _, material := range receipt.Materials {
		state.rows = append(state.rows, inboundEditorMaterial{
			Request: api.InboundMaterialRequest{
				Index: material.Index, ID: material.ID, Price: material.Price,
				EstimatedQuantity: material.EstimatedQuantity,
			},
			Name: material.Name, Model: material.Model, Unit: material.Unit,
		})
	}
	state.refreshRows()
	return state
}

func (state *inboundEditorUI) dispose() {
	if state == nil || state.closed.Swap(true) {
		return
	}
	if state.cancel != nil {
		state.cancel()
	}
}

func (state *inboundEditorUI) refreshRows() {
	for index := range state.rows {
		state.rows[index].Request.Index = index + 1
		state.rows[index].Index = fmt.Sprint(index + 1)
		state.rows[index].Price = fmt.Sprintf("%.3f", state.rows[index].Request.Price)
		state.rows[index].Quantity = fmt.Sprintf("%g %s", state.rows[index].Request.EstimatedQuantity, state.rows[index].Unit)
		state.rows[index].Position = "未指定"
		for _, position := range state.positions {
			if stringSlicesEqual(position.IDs, state.rows[index].Request.Position) {
				state.rows[index].Position = position.Label
				break
			}
		}
	}
}

func (state *inboundEditorUI) totalAmount() float64 {
	total := 0.0
	for _, row := range state.rows {
		total += row.Request.Price * row.Request.EstimatedQuantity
	}
	return total
}

func (ui *mainUI) inboundEditorPageWidget(state *inboundEditorUI) TabPage {
	title := "新建入库单"
	description := "填写基本信息并添加物料；保存后由服务端创建为待审核状态。"
	if state.mode == "edit" {
		title = "编辑入库单 · " + state.baseline.Code
		description = "单号和类型按现有服务端更新规则保持只读；原计划仓位不在列表响应中，需按实际需要重新选择。"
	}
	return TabPage{
		AssignTo: &state.tab,
		Title:    closableTabTitle(title),
		Layout:   VBox{Margins: Margins{Left: 14, Top: 14, Right: 14, Bottom: 12}, Spacing: 10},
		Children: []Widget{
			Label{Text: title, Font: Font{Family: "Microsoft YaHei UI", PointSize: 15, Bold: true}},
			Label{Text: description, TextColor: secondaryTextColor(), Accessibility: Accessibility{Name: "入库单编辑说明"}},
			GroupBox{
				Title:  "基本信息",
				Layout: Grid{Columns: 8, Margins: Margins{Left: 14, Top: 10, Right: 14, Bottom: 12}, Spacing: 8},
				Children: []Widget{
					Label{Text: "入库单号 *"},
					LineEdit{AssignTo: &state.code, MinSize: Size{Width: 190, Height: 28}, CueBanner: "必填，建议使用唯一编号", Accessibility: Accessibility{Name: "入库单号"}},
					Label{Text: "入库类型 *"},
					ComboBox{AssignTo: &state.orderType, Model: inboundTypes[1:], CurrentIndex: 0, MinSize: Size{Width: 130, Height: 28}, Accessibility: Accessibility{Name: "入库类型"}},
					Label{AssignTo: &state.supplierLabel, Text: "供应商"},
					ComboBox{AssignTo: &state.supplier, Model: []string{"正在加载供应商……"}, CurrentIndex: 0, MinSize: Size{Width: 190, Height: 28}, Accessibility: Accessibility{Name: "入库供应商"}},
					Label{AssignTo: &state.customerLabel, Text: "客户"},
					ComboBox{AssignTo: &state.customer, Model: []string{"正在加载客户……"}, CurrentIndex: 0, MinSize: Size{Width: 190, Height: 28}, Accessibility: Accessibility{Name: "退货入库客户"}},
					Label{Text: "预计入库日期"},
					LineEdit{AssignTo: &state.planDate, CueBanner: "YYYY-MM-DD，可留空", MinSize: Size{Width: 150, Height: 28}, Accessibility: Accessibility{Name: "预计入库日期"}},
					Label{Text: "预计金额"},
					Label{AssignTo: &state.total, Text: "0.000", Font: Font{Bold: true}, Accessibility: Accessibility{Name: "入库预计总金额"}},
					Label{Text: "备注"},
					LineEdit{AssignTo: &state.remark, ColumnSpan: 3, CueBanner: "可选，填写单据说明", Accessibility: Accessibility{Name: "入库单备注"}},
				},
			},
			GroupBox{
				Title:  "附件",
				Layout: HBox{Margins: Margins{Left: 14, Top: 10, Right: 14, Bottom: 12}, Spacing: 8},
				Children: []Widget{
					ComboBox{AssignTo: &state.attachment, Model: []string{"暂无附件"}, CurrentIndex: 0, MinSize: Size{Width: 220, Height: 28}, Accessibility: Accessibility{Name: "入库单附件列表"}},
					PushButton{AssignTo: &state.selectAttachment, Text: "选择素材", MinSize: Size{Width: 92, Height: 30}, OnClicked: ui.selectInboundEditorAttachments},
					PushButton{AssignTo: &state.upload, Text: "上传图片", MinSize: Size{Width: 92, Height: 30}, ToolTipText: "上传期间再次点击可取消", Accessibility: Accessibility{Name: "上传入库单图片附件"}, OnClicked: ui.uploadInboundEditorAttachment},
					PushButton{AssignTo: &state.preview, Text: "预览", Enabled: false, MinSize: Size{Width: 72, Height: 30}, OnClicked: ui.previewInboundEditorAttachments},
					PushButton{AssignTo: &state.removeAttachment, Text: "移除", Enabled: false, MinSize: Size{Width: 72, Height: 30}, OnClicked: ui.removeInboundEditorAttachment},
					Label{Text: "上传仅支持现有图片接口；移除只影响本次单据提交。", TextColor: secondaryTextColor()},
					HSpacer{},
				},
			},
			GroupBox{
				Title:         "物料明细 *",
				StretchFactor: 1,
				Layout:        VBox{Margins: Margins{Left: 12, Top: 10, Right: 12, Bottom: 12}, Spacing: 8},
				Children: []Widget{
					Composite{Layout: HBox{Spacing: 8}, Children: []Widget{
						Label{Text: "价格和预计数量均按现有接口提交；仓位可选。", TextColor: secondaryTextColor()},
						HSpacer{},
						PushButton{AssignTo: &state.addMaterial, Text: "添加物料", MinSize: Size{Width: 92, Height: 30}, OnClicked: ui.addInboundEditorMaterial},
						PushButton{AssignTo: &state.editMaterial, Text: "编辑当前行", Enabled: false, MinSize: Size{Width: 104, Height: 30}, OnClicked: ui.editInboundEditorMaterial},
						PushButton{AssignTo: &state.removeMaterial, Text: "移除当前行", Enabled: false, MinSize: Size{Width: 104, Height: 30}, OnClicked: ui.removeInboundEditorMaterial},
					}},
					TableView{
						AssignTo: &state.materials, Model: state.rows, AlternatingRowBG: true, ColumnsOrderable: true, StretchFactor: 1,
						Accessibility:         Accessibility{Name: "入库单物料明细", Description: "选择一行后可编辑单价、数量和计划仓位"},
						OnCurrentIndexChanged: ui.updateInboundEditorSelection,
						OnItemActivated:       ui.editInboundEditorMaterial,
						Columns: []TableViewColumn{
							{Title: "序号", DataMember: "Index", Width: 55},
							{Title: "物料名称", DataMember: "Name", Width: 190},
							{Title: "型号", DataMember: "Model", Width: 130},
							{Title: "单位", DataMember: "Unit", Width: 70},
							{Title: "单价", DataMember: "Price", Width: 100},
							{Title: "预计数量", DataMember: "Quantity", Width: 110},
							{Title: "计划仓位", DataMember: "Position", Width: 300},
						},
					},
				},
			},
			Composite{Layout: HBox{Spacing: 8}, Children: []Widget{
				Label{AssignTo: &state.info, Text: "正在加载供应商、客户和仓位选项……", TextColor: secondaryTextColor(), Accessibility: Accessibility{Name: "入库单编辑状态"}},
				HSpacer{},
				PushButton{AssignTo: &state.cancelButton, Text: "取消", MinSize: Size{Width: 80, Height: 30}, OnClicked: func() { ui.closeInboundEditor(false) }},
				PushButton{AssignTo: &state.save, Text: "核对并保存", MinSize: Size{Width: 110, Height: 30}, OnClicked: ui.submitInboundEditor},
			}},
		},
	}
}

func (ui *mainUI) newInboundReceipt() {
	if !hasButton(ui.session.Perms.Buttons, "inbound:receipt:add") {
		walk.MsgBox(ui.window, "没有权限", "当前账号没有新建入库单权限。", walk.MsgBoxIconWarning)
		return
	}
	ui.openInboundEditor("add", api.InboundReceipt{})
}

func (ui *mainUI) editSelectedInbound() {
	receipt, ok := ui.selectedInbound()
	if !ok {
		return
	}
	if !hasButton(ui.session.Perms.Buttons, "inbound:receipt:edit") || !canEditInbound(receipt.Status) {
		walk.MsgBox(ui.window, "当前不可编辑", fmt.Sprintf("入库单当前状态为“%s”，只有待审核或审核不通过的单据可以编辑。", receipt.Status), walk.MsgBoxIconWarning)
		return
	}
	ui.openInboundEditor("edit", receipt)
}

func (ui *mainUI) openInboundEditor(mode string, receipt api.InboundReceipt) {
	if current := ui.inboundEditor; current != nil {
		if current.busy {
			walk.MsgBox(ui.window, "正在提交", "当前入库单正在提交和复核，请等待完成。", walk.MsgBoxIconInformation)
			return
		}
		if current.mode == mode && current.baseline.ID == receipt.ID {
			_ = ui.tabs.SetCurrentIndex(ui.tabs.Pages().Index(current.tab))
			return
		}
		if current.dirty && walk.MsgBox(ui.window, "替换未提交编辑", "当前入库单还有未提交修改，是否放弃并打开另一张单据？", walk.MsgBoxYesNo|walk.MsgBoxIconWarning) != walk.DlgCmdYes {
			return
		}
		ui.closeInboundEditor(true)
	}

	state := newInboundEditorUI(mode, receipt)
	pageDecl := ui.inboundEditorPageWidget(state)
	if err := pageDecl.Create(NewBuilder(nil)); err != nil {
		state.dispose()
		walk.MsgBox(ui.window, "无法打开入库编辑页", err.Error(), walk.MsgBoxIconError)
		return
	}
	ui.inboundEditor = state
	ui.inboundEditorTab = state.tab
	insertAt := ui.tabs.Pages().Len()
	if ui.systemTab != nil {
		if index := ui.tabs.Pages().Index(ui.systemTab); index >= 0 {
			insertAt = index
		}
	}
	if ui.inboundTab != nil {
		if index := ui.tabs.Pages().Index(ui.inboundTab); index >= 0 {
			insertAt = index + 1
		}
	}
	if err := ui.tabs.Pages().Insert(insertAt, state.tab); err != nil {
		state.tab.Dispose()
		state.dispose()
		ui.inboundEditor = nil
		ui.inboundEditorTab = nil
		walk.MsgBox(ui.window, "无法打开入库编辑页", err.Error(), walk.MsgBoxIconError)
		return
	}
	_ = ui.tabs.SetCurrentIndex(ui.tabs.Pages().Index(state.tab))
	ui.initializeInboundEditor(state)
}

func (ui *mainUI) initializeInboundEditor(state *inboundEditorUI) {
	if state == nil || state != ui.inboundEditor {
		return
	}
	code := state.baseline.Code
	if state.mode == "add" {
		code = "I-" + time.Now().Format("2006-01-02-15-04-05-000")
	}
	state.code.SetText(code)
	if state.mode == "edit" {
		state.code.SetReadOnly(true)
	}
	typeIndex := stringIndex(inboundTypes[1:], state.baseline.Type)
	if typeIndex < 0 {
		typeIndex = 0
	}
	state.orderType.SetCurrentIndex(typeIndex)
	if state.mode == "edit" {
		state.orderType.SetEnabled(false)
	}
	if state.baseline.ReceivingDate > 0 {
		state.planDate.SetText(time.Unix(state.baseline.ReceivingDate, 0).Format("2006-01-02"))
	}
	state.remark.SetText(state.baseline.Remark)
	ui.refreshInboundEditorTable()
	ui.refreshInboundEditorAttachments()
	ui.updateInboundEditorPartnerControls()

	markDirty := func() {
		if state.initializing || state.busy || state.submitted {
			return
		}
		state.dirty = true
		state.info.SetText("存在尚未提交的修改。")
	}
	state.code.TextChanged().Attach(markDirty)
	state.planDate.TextChanged().Attach(markDirty)
	state.remark.TextChanged().Attach(markDirty)
	state.orderType.CurrentIndexChanged().Attach(func() {
		ui.updateInboundEditorPartnerControls()
		markDirty()
	})
	state.supplier.CurrentIndexChanged().Attach(markDirty)
	state.customer.CurrentIndexChanged().Attach(markDirty)
	state.initializing = false
	state.dirty = false
	ui.loadInboundEditorDependencies(state)
}

func (ui *mainUI) loadInboundEditorDependencies(state *inboundEditorUI) {
	guardedGo(func() {
		suppliers, supplierErr := ui.session.Client.Suppliers(state.ctx)
		customers, customerErr := ui.session.Client.Customers(state.ctx)
		warehouses, warehouseErr := ui.session.Client.WarehouseTree(state.ctx)
		if state.ctx.Err() != nil || state.closed.Load() {
			return
		}
		ui.window.Synchronize(func() {
			if state != ui.inboundEditor || state.closed.Load() {
				return
			}
			state.initializing = true
			state.supplierOptions = nil
			for _, supplier := range suppliers {
				state.supplierOptions = append(state.supplierOptions, selectOption{ID: supplier.ID, Label: businessOptionLabel(supplier.Name, supplier.Code)})
			}
			state.customerOptions = nil
			for _, customer := range customers {
				state.customerOptions = append(state.customerOptions, selectOption{ID: customer.ID, Label: businessOptionLabel(customer.Name, customer.Code)})
			}
			state.positions = flattenPositions(warehouses)
			supplierLabel := "请选择供应商"
			if supplierErr != nil {
				supplierLabel = "供应商加载失败"
			}
			customerLabel := "请选择客户"
			if customerErr != nil {
				customerLabel = "客户加载失败"
			}
			_ = state.supplier.SetModel(optionLabels(supplierLabel, state.supplierOptions))
			_ = state.customer.SetModel(optionLabels(customerLabel, state.customerOptions))
			state.supplier.SetCurrentIndex(optionIndexByID(state.supplierOptions, state.baseline.SupplierID))
			state.customer.SetCurrentIndex(optionIndexByID(state.customerOptions, state.baseline.CustomerID))
			state.initializing = false
			state.refreshRows()
			_ = state.materials.SetModel(state.rows)
			ui.updateInboundEditorPartnerControls()
			errors := make([]string, 0, 3)
			if supplierErr != nil {
				errors = append(errors, "供应商："+supplierErr.Error())
			}
			if customerErr != nil {
				errors = append(errors, "客户："+customerErr.Error())
			}
			if warehouseErr != nil {
				errors = append(errors, "仓位："+warehouseErr.Error())
			}
			if len(errors) > 0 {
				state.info.SetText("部分选项加载失败；可保留页面后重试打开：" + strings.Join(errors, "；"))
			} else if state.mode == "edit" {
				state.info.SetText("选项已加载。编辑响应不含原计划仓位；未重新选择的物料将按空仓位提交。")
			} else {
				state.info.SetText("供应商、客户和仓位选项已从线上加载。")
			}
		})
	})
}

func optionIndexByID(options []selectOption, id string) int {
	for index, option := range options {
		if option.ID == id {
			return index + 1
		}
	}
	return 0
}

func (ui *mainUI) updateInboundEditorPartnerControls() {
	state := ui.inboundEditor
	if state == nil || state.orderType == nil {
		return
	}
	typeName := state.orderType.Text()
	supplierRequired := typeName == "采购入库" || typeName == "外协入库"
	customerRequired := typeName == "退货入库"
	state.supplierLabel.SetText("供应商")
	if supplierRequired {
		state.supplierLabel.SetText("供应商 *")
	}
	state.customerLabel.SetText("客户")
	if customerRequired {
		state.customerLabel.SetText("客户 *")
	}
	state.supplier.SetEnabled(supplierRequired && !state.busy && !state.submitted)
	state.customer.SetEnabled(customerRequired && !state.busy && !state.submitted)
}

func (ui *mainUI) closeInboundEditor(force bool) {
	state := ui.inboundEditor
	if state == nil || state.tab == nil || ui.tabs == nil {
		return
	}
	if !force {
		if state.busy {
			walk.MsgBox(ui.window, "正在提交", "入库单正在提交和复核，请等待完成，避免产生未知状态。", walk.MsgBoxIconInformation)
			return
		}
		if state.dirty && walk.MsgBox(ui.window, "放弃未提交修改", "当前入库单还有未提交修改，是否放弃？", walk.MsgBoxYesNo|walk.MsgBoxIconWarning) != walk.DlgCmdYes {
			return
		}
	}
	index := ui.tabs.Pages().Index(state.tab)
	if index >= 0 {
		if err := ui.tabs.Pages().RemoveAt(index); err != nil {
			walk.MsgBox(ui.window, "无法关闭编辑页", err.Error(), walk.MsgBoxIconError)
			return
		}
	}
	state.dispose()
	state.tab.Dispose()
	ui.inboundEditor = nil
	ui.inboundEditorTab = nil
	ui.syncNavigationFromTab()
}

func (ui *mainUI) updateInboundEditorSelection() {
	state := ui.inboundEditor
	if state == nil || state.materials == nil {
		return
	}
	index := state.materials.CurrentIndex()
	enabled := index >= 0 && index < len(state.rows) && !state.busy && !state.submitted
	state.editMaterial.SetEnabled(enabled)
	state.removeMaterial.SetEnabled(enabled)
}

func (ui *mainUI) refreshInboundEditorTable() {
	state := ui.inboundEditor
	if state == nil || state.materials == nil {
		return
	}
	state.refreshRows()
	_ = state.materials.SetModel(state.rows)
	state.total.SetText(fmt.Sprintf("%.3f", state.totalAmount()))
	ui.updateInboundEditorSelection()
}

func (ui *mainUI) addInboundEditorMaterial() {
	state := ui.inboundEditor
	if state == nil || state.busy || state.submitted {
		return
	}
	material, ok := selectInboundMaterial(ui.window, ui.session.Client)
	if !ok {
		return
	}
	for _, row := range state.rows {
		if row.Request.ID == material.ID {
			state.info.SetText("该物料已经在明细中，请直接编辑当前行。")
			return
		}
	}
	row := inboundEditorMaterial{
		Name: material.Name, Model: material.Model, Unit: material.Unit,
		Request: api.InboundMaterialRequest{ID: material.ID, Price: 0, EstimatedQuantity: 1},
	}
	if !editInboundMaterialDetails(ui.window, &row, state.positions) {
		return
	}
	state.rows = append(state.rows, row)
	state.dirty = true
	ui.refreshInboundEditorTable()
	_ = state.materials.SetCurrentIndex(len(state.rows) - 1)
	state.info.SetText("已添加物料；修改尚未提交。")
}

func (ui *mainUI) editInboundEditorMaterial() {
	state := ui.inboundEditor
	if state == nil || state.busy || state.submitted {
		return
	}
	index := state.materials.CurrentIndex()
	if index < 0 || index >= len(state.rows) {
		walk.MsgBox(ui.window, "请选择物料", "请先选择需要编辑的物料行。", walk.MsgBoxIconInformation)
		return
	}
	row := state.rows[index]
	if !editInboundMaterialDetails(ui.window, &row, state.positions) {
		return
	}
	state.rows[index] = row
	state.dirty = true
	ui.refreshInboundEditorTable()
	_ = state.materials.SetCurrentIndex(index)
	state.info.SetText("物料明细已修改；尚未提交。")
}

func (ui *mainUI) removeInboundEditorMaterial() {
	state := ui.inboundEditor
	if state == nil || state.busy || state.submitted {
		return
	}
	index := state.materials.CurrentIndex()
	if index < 0 || index >= len(state.rows) {
		return
	}
	if walk.MsgBox(ui.window, "移除物料", fmt.Sprintf("是否从本次入库单中移除“%s”？", state.rows[index].Name), walk.MsgBoxYesNo|walk.MsgBoxIconWarning) != walk.DlgCmdYes {
		return
	}
	state.rows = append(state.rows[:index], state.rows[index+1:]...)
	state.dirty = true
	ui.refreshInboundEditorTable()
	state.info.SetText("物料已从当前编辑内容移除；尚未提交。")
}

func (ui *mainUI) refreshInboundEditorAttachments() {
	state := ui.inboundEditor
	if state == nil || state.attachment == nil {
		return
	}
	labels := []string{"暂无附件"}
	if len(state.annex) > 0 {
		labels = make([]string, len(state.annex))
		for index := range state.annex {
			labels[index] = fmt.Sprintf("附件 %d / %d", index+1, len(state.annex))
		}
	}
	_ = state.attachment.SetModel(labels)
	state.attachment.SetCurrentIndex(0)
	hasAttachments := len(state.annex) > 0
	state.preview.SetEnabled(hasAttachments && !state.busy)
	state.removeAttachment.SetEnabled(hasAttachments && !state.busy && !state.submitted)
}

func (ui *mainUI) uploadInboundEditorAttachment() {
	state := ui.inboundEditor
	if state == nil {
		return
	}
	if state.uploadCancel != nil {
		state.uploadCancel()
		state.upload.SetText("正在取消")
		state.upload.SetEnabled(false)
		state.info.SetText("正在取消附件上传……")
		return
	}
	if state.busy || state.submitted {
		return
	}
	dialog := new(walk.FileDialog)
	dialog.Title = "选择入库单图片附件"
	dialog.Filter = "图片文件 (*.png;*.jpg;*.jpeg;*.gif)|*.png;*.jpg;*.jpeg;*.gif|所有文件 (*.*)|*.*"
	if ok, err := dialog.ShowOpen(ui.window); err != nil {
		walk.MsgBox(ui.window, "无法选择附件", err.Error(), walk.MsgBoxIconError)
	} else if !ok {
		return
	}
	state.busy = true
	uploadCtx, uploadCancel := context.WithCancel(state.ctx)
	state.uploadCancel = uploadCancel
	ui.setInboundEditorEnabled(false)
	state.upload.SetText("取消上传")
	state.upload.SetEnabled(true)
	state.info.SetText("正在上传图片附件……")
	filePath := dialog.FilePath
	guardedGo(func() {
		defer uploadCancel()
		reference, err := ui.session.Client.UploadImage(uploadCtx, filePath)
		if state.ctx.Err() != nil || state.closed.Load() {
			return
		}
		canceled := uploadCtx.Err() != nil
		ui.window.Synchronize(func() {
			if state != ui.inboundEditor || state.closed.Load() {
				return
			}
			state.uploadCancel = nil
			state.busy = false
			ui.setInboundEditorEnabled(true)
			state.upload.SetText("上传图片")
			if canceled {
				state.info.SetText("附件上传已取消，当前单据未加入新附件。")
				return
			}
			if err != nil {
				state.info.SetText("附件上传失败：" + err.Error())
				return
			}
			state.annex = append(state.annex, reference)
			state.dirty = true
			ui.refreshInboundEditorAttachments()
			state.info.SetText("附件已上传并加入当前单据；保存前仍可移除。")
		})
	})
}

func (ui *mainUI) selectInboundEditorAttachments() {
	state := ui.inboundEditor
	if state == nil || state.busy || state.submitted {
		return
	}
	references, ok := SelectImageAssets(ui.window, ui.session.Client, config.ImageBaseURL(), 10)
	if !ok || len(references) == 0 {
		return
	}
	state.annex = appendUniqueReferences(state.annex, references, len(state.annex)+len(references))
	state.dirty = true
	ui.refreshInboundEditorAttachments()
	state.info.SetText(fmt.Sprintf("已从素材库加入 %d 张附件；保存前仍可移除。", len(references)))
}

func (ui *mainUI) previewInboundEditorAttachments() {
	state := ui.inboundEditor
	if state == nil {
		return
	}
	ShowOrderAttachments(ui.window, ui.session.Client, config.ImageBaseURL(), "入库单 "+state.code.Text(), state.annex)
}

func (ui *mainUI) removeInboundEditorAttachment() {
	state := ui.inboundEditor
	if state == nil || len(state.annex) == 0 || state.busy || state.submitted {
		return
	}
	index := state.attachment.CurrentIndex()
	if index < 0 || index >= len(state.annex) {
		index = 0
	}
	if walk.MsgBox(ui.window, "移除附件", fmt.Sprintf("是否从当前入库单中移除附件 %d？已上传的图片文件不会从服务端图片库删除。", index+1), walk.MsgBoxYesNo|walk.MsgBoxIconWarning) != walk.DlgCmdYes {
		return
	}
	state.annex = append(state.annex[:index], state.annex[index+1:]...)
	state.dirty = true
	ui.refreshInboundEditorAttachments()
	state.info.SetText("附件已从当前提交内容移除。")
}

func (ui *mainUI) setInboundEditorEnabled(enabled bool) {
	state := ui.inboundEditor
	if state == nil {
		return
	}
	canEdit := enabled && !state.submitted
	state.code.SetEnabled(canEdit)
	if state.mode == "edit" {
		state.code.SetReadOnly(true)
	}
	state.orderType.SetEnabled(canEdit && state.mode == "add")
	state.planDate.SetEnabled(canEdit)
	state.remark.SetEnabled(canEdit)
	state.upload.SetEnabled(canEdit)
	state.selectAttachment.SetEnabled(canEdit)
	state.addMaterial.SetEnabled(canEdit)
	state.save.SetEnabled(canEdit)
	state.cancelButton.SetEnabled(enabled)
	ui.updateInboundEditorPartnerControls()
	ui.updateInboundEditorSelection()
	ui.refreshInboundEditorAttachments()
}

func buildInboundReceiptRequest(state *inboundEditorUI) (api.InboundReceiptRequest, error) {
	code := strings.TrimSpace(state.code.Text())
	if code == "" {
		return api.InboundReceiptRequest{}, fmt.Errorf("入库单号不能为空")
	}
	typeName := strings.TrimSpace(state.orderType.Text())
	if stringIndex(inboundTypes[1:], typeName) < 0 {
		return api.InboundReceiptRequest{}, fmt.Errorf("请选择有效的入库类型")
	}
	receivingDate := int64(0)
	dateText := strings.TrimSpace(state.planDate.Text())
	if dateText != "" {
		date, err := time.ParseInLocation("2006-01-02", dateText, time.Local)
		if err != nil {
			return api.InboundReceiptRequest{}, fmt.Errorf("预计入库日期必须使用 YYYY-MM-DD 格式")
		}
		receivingDate = date.Unix()
	}
	request := api.InboundReceiptRequest{
		ID: state.baseline.ID, Code: code, Type: typeName,
		ReceivingDate: receivingDate, Remark: strings.TrimSpace(state.remark.Text()),
		Annex: append([]string(nil), state.annex...),
	}
	if typeName == "采购入库" || typeName == "外协入库" {
		request.SupplierID = selectedOptionID(state.supplier, state.supplierOptions)
		if request.SupplierID == "" {
			return api.InboundReceiptRequest{}, fmt.Errorf("%s必须选择供应商", typeName)
		}
	}
	if typeName == "退货入库" {
		request.CustomerID = selectedOptionID(state.customer, state.customerOptions)
		if request.CustomerID == "" {
			return api.InboundReceiptRequest{}, fmt.Errorf("退货入库必须选择客户")
		}
	}
	if len(state.rows) == 0 {
		return api.InboundReceiptRequest{}, fmt.Errorf("至少需要添加一条物料")
	}
	seen := make(map[string]bool, len(state.rows))
	for index, row := range state.rows {
		material := row.Request
		material.Index = index + 1
		material.ID = strings.TrimSpace(material.ID)
		if material.ID == "" {
			return api.InboundReceiptRequest{}, fmt.Errorf("第 %d 行物料编号为空", index+1)
		}
		if seen[material.ID] {
			return api.InboundReceiptRequest{}, fmt.Errorf("物料“%s”重复", row.Name)
		}
		seen[material.ID] = true
		if material.Price < 0 {
			return api.InboundReceiptRequest{}, fmt.Errorf("物料“%s”的单价不能小于 0", row.Name)
		}
		if material.EstimatedQuantity <= 0 {
			return api.InboundReceiptRequest{}, fmt.Errorf("物料“%s”的预计数量必须大于 0", row.Name)
		}
		material.Position = append([]string(nil), material.Position...)
		request.Materials = append(request.Materials, material)
		request.TotalAmount += material.Price * material.EstimatedQuantity
	}
	return request, nil
}

func (ui *mainUI) submitInboundEditor() {
	state := ui.inboundEditor
	if state == nil || state.busy || state.submitted {
		return
	}
	request, err := buildInboundReceiptRequest(state)
	if err != nil {
		state.info.SetText("无法提交：" + err.Error())
		return
	}
	action := "新建"
	if state.mode == "edit" {
		action = "更新"
	}
	message := fmt.Sprintf("操作：%s入库单\r\n单号：%s\r\n类型：%s\r\n物料：%d 条\r\n预计金额：%.3f\r\n\r\n是否提交到线上服务？", action, request.Code, request.Type, len(request.Materials), request.TotalAmount)
	if walk.MsgBox(ui.window, "核对入库单", message, walk.MsgBoxYesNo|walk.MsgBoxIconQuestion) != walk.DlgCmdYes {
		return
	}
	state.busy = true
	ui.setInboundEditorEnabled(false)
	state.info.SetText("正在重新核对状态并提交到线上服务……")
	guardedGo(func() {
		var requestErr error
		if state.mode == "edit" {
			current, found, findErr := ui.session.Client.FindInboundReceipt(state.ctx, state.baseline.ID, state.baseline.Code)
			switch {
			case findErr != nil:
				requestErr = fmt.Errorf("提交前重新读取入库单失败：%w", findErr)
			case !found:
				requestErr = fmt.Errorf("入库单已不存在，请关闭编辑页并刷新列表")
			case !canEditInbound(current.Status):
				requestErr = fmt.Errorf("入库单状态已变为“%s”，不能继续编辑", current.Status)
			default:
				requestErr = ui.session.Client.UpdateInboundReceipt(state.ctx, request)
			}
		} else {
			requestErr = ui.session.Client.CreateInboundReceipt(state.ctx, request)
		}

		var verifyErr error
		if requestErr == nil {
			page, err := ui.session.Client.InboundReceipts(state.ctx, 1, 100, api.InboundFilters{Code: request.Code})
			if err != nil {
				verifyErr = err
			} else {
				found := false
				for _, receipt := range page.List {
					if receipt.Code == request.Code && (request.ID == "" || receipt.ID == request.ID) {
						found = true
						break
					}
				}
				if !found {
					verifyErr = fmt.Errorf("列表中尚未查到单据 %s", request.Code)
				}
			}
		}
		if state.ctx.Err() != nil || state.closed.Load() {
			return
		}
		ui.window.Synchronize(func() {
			if state != ui.inboundEditor || state.closed.Load() {
				return
			}
			state.busy = false
			if requestErr != nil {
				ui.setInboundEditorEnabled(true)
				state.info.SetText("提交失败：" + requestErr.Error())
				return
			}
			state.dirty = false
			if verifyErr != nil {
				state.submitted = true
				ui.setInboundEditorEnabled(false)
				state.cancelButton.SetEnabled(true)
				state.info.SetText("服务端已返回成功，但在线复核失败：" + verifyErr.Error() + "。请勿重复提交，关闭后刷新列表人工核对。")
				walk.MsgBox(ui.window, "提交后复核失败", state.info.Text(), walk.MsgBoxIconWarning)
				ui.loadInbound()
				ui.loadDashboard()
				return
			}
			ui.notifyStatusSuccess("入库单已保存，并完成线上回读复核。")
			ui.loadInbound()
			ui.loadDashboard()
			ui.closeInboundEditor(true)
		})
	})
}

func stringSlicesEqual(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func parsePositiveInboundNumber(text, field string, allowZero bool) (float64, error) {
	value, err := strconv.ParseFloat(strings.TrimSpace(text), 64)
	if err != nil || value < 0 || (!allowZero && value <= 0) {
		if allowZero {
			return 0, fmt.Errorf("%s必须是非负数字", field)
		}
		return 0, fmt.Errorf("%s必须是大于 0 的数字", field)
	}
	return value, nil
}
