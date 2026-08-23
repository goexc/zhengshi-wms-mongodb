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

var outboundCreateTypes = []string{
	"销售出库", "样品出库", "报废出库", "赠品出库", "生产用料出库", "损耗出库",
}

func outboundTypeRequiresCustomer(orderType string) bool {
	switch strings.TrimSpace(orderType) {
	case "销售出库", "样品出库", "赠品出库":
		return true
	default:
		return false
	}
}

type outboundEditorMaterial struct {
	Index         string
	Name          string
	Model         string
	Specification string
	Unit          string
	Price         string
	Quantity      string
	Amount        string
	Reference     string
	Material      api.Material
	Request       api.OutboundMaterialRequest
}

type outboundEditorUI struct {
	tab              *walk.TabPage
	code             *walk.LineEdit
	orderType        *walk.ComboBox
	customerLabel    *walk.Label
	customer         *walk.ComboBox
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
	priceHistory     *walk.PushButton
	info             *walk.Label
	save             *walk.PushButton
	cancelButton     *walk.PushButton

	rows            []outboundEditorMaterial
	annex           []string
	customerOptions []selectOption
	dirty           bool
	initializing    bool
	busy            bool
	submitted       bool
	priceGeneration int
	ctx             context.Context
	cancel          context.CancelFunc
	uploadCancel    context.CancelFunc
	closed          atomic.Bool
}

func newOutboundEditorUI() *outboundEditorUI {
	ctx, cancel := context.WithCancel(context.Background())
	return &outboundEditorUI{ctx: ctx, cancel: cancel, initializing: true}
}

func (state *outboundEditorUI) dispose() {
	if state == nil || state.closed.Swap(true) {
		return
	}
	state.priceGeneration++
	state.cancel()
}

func (state *outboundEditorUI) refreshRows() {
	for index := range state.rows {
		state.rows[index].Request.Index = index + 1
		state.rows[index].Index = fmt.Sprint(index + 1)
		state.rows[index].Price = fmt.Sprintf("%.3f", state.rows[index].Request.Price)
		state.rows[index].Quantity = fmt.Sprintf("%g %s", state.rows[index].Request.Quantity, state.rows[index].Unit)
		state.rows[index].Amount = fmt.Sprintf("%.3f", state.rows[index].Request.Price*state.rows[index].Request.Quantity)
		if strings.TrimSpace(state.rows[index].Reference) == "" {
			state.rows[index].Reference = "尚未查询"
		}
	}
}

func (state *outboundEditorUI) summary() (quantity, amount float64, unpriced int) {
	for _, row := range state.rows {
		quantity += row.Request.Quantity
		amount += row.Request.Price * row.Request.Quantity
		if row.Request.Price == 0 {
			unpriced++
		}
	}
	return
}

func (ui *mainUI) outboundEditorPageWidget(state *outboundEditorUI) TabPage {
	return TabPage{
		AssignTo: &state.tab,
		Title:    closableTabTitle("新建出库单"),
		Layout:   VBox{Margins: Margins{Left: 14, Top: 14, Right: 14, Bottom: 12}, Spacing: 10},
		Children: []Widget{
			Label{Text: "新建出库单", Font: Font{Family: "Microsoft YaHei UI", PointSize: 15, Bold: true}},
			Label{
				Text:      "仅创建预发货单；退货出库因现有接口的客户/供应商约束不一致，客户端暂不提供。",
				TextColor: secondaryTextColor(), Accessibility: Accessibility{Name: "出库建单范围说明"},
			},
			GroupBox{
				Title:  "基本信息",
				Layout: Grid{Columns: 8, Margins: Margins{Left: 14, Top: 10, Right: 14, Bottom: 12}, Spacing: 8},
				Children: []Widget{
					Label{Text: "出库单号 *"},
					LineEdit{AssignTo: &state.code, MinSize: Size{Width: 190, Height: 28}, CueBanner: "必填且唯一", Accessibility: Accessibility{Name: "出库单号"}},
					Label{Text: "出库类型 *"},
					ComboBox{AssignTo: &state.orderType, Model: outboundCreateTypes, CurrentIndex: 0, MinSize: Size{Width: 150, Height: 28}, Accessibility: Accessibility{Name: "出库类型"}},
					Label{AssignTo: &state.customerLabel, Text: "客户 *"},
					ComboBox{AssignTo: &state.customer, Model: []string{"正在加载客户……"}, CurrentIndex: 0, MinSize: Size{Width: 190, Height: 28}, Accessibility: Accessibility{Name: "出库客户"}},
					Label{Text: "合计金额"},
					Label{AssignTo: &state.total, Text: "0.000", Font: Font{Bold: true}, Accessibility: Accessibility{Name: "出库单合计金额"}},
					Label{Text: "备注"},
					LineEdit{AssignTo: &state.remark, ColumnSpan: 7, CueBanner: "可选，填写出库说明", Accessibility: Accessibility{Name: "出库单备注"}},
				},
			},
			GroupBox{
				Title:  "附件",
				Layout: HBox{Margins: Margins{Left: 14, Top: 10, Right: 14, Bottom: 12}, Spacing: 8},
				Children: []Widget{
					ComboBox{AssignTo: &state.attachment, Model: []string{"暂无附件"}, CurrentIndex: 0, MinSize: Size{Width: 220, Height: 28}, Accessibility: Accessibility{Name: "出库单附件列表"}},
					PushButton{AssignTo: &state.selectAttachment, Text: "选择素材", MinSize: Size{Width: 92, Height: 30}, OnClicked: ui.selectOutboundEditorAttachments},
					PushButton{AssignTo: &state.upload, Text: "上传图片", MinSize: Size{Width: 92, Height: 30}, ToolTipText: "上传期间再次点击可取消", Accessibility: Accessibility{Name: "上传出库单图片附件"}, OnClicked: ui.uploadOutboundEditorAttachment},
					PushButton{AssignTo: &state.preview, Text: "预览", Enabled: false, MinSize: Size{Width: 72, Height: 30}, OnClicked: ui.previewOutboundEditorAttachments},
					PushButton{AssignTo: &state.removeAttachment, Text: "移除", Enabled: false, MinSize: Size{Width: 72, Height: 30}, OnClicked: ui.removeOutboundEditorAttachment},
					Label{Text: "移除只影响本次提交，不删除服务器图片。", TextColor: secondaryTextColor()},
					HSpacer{},
				},
			},
			GroupBox{
				Title:         "物料明细 *",
				StretchFactor: 1,
				Layout:        VBox{Margins: Margins{Left: 12, Top: 10, Right: 12, Bottom: 12}, Spacing: 8},
				Children: []Widget{
					Composite{Layout: HBox{Spacing: 8}, Children: []Widget{
						Label{Text: "单价允许为 0，但提交前会明确提示未核价行。", TextColor: secondaryTextColor()},
						HSpacer{},
						PushButton{AssignTo: &state.priceHistory, Text: "历史价格", Enabled: false, MinSize: Size{Width: 92, Height: 30}, OnClicked: ui.showOutboundEditorPriceHistory},
						PushButton{AssignTo: &state.addMaterial, Text: "添加物料", MinSize: Size{Width: 92, Height: 30}, OnClicked: ui.addOutboundEditorMaterial},
						PushButton{AssignTo: &state.editMaterial, Text: "编辑当前行", Enabled: false, MinSize: Size{Width: 104, Height: 30}, OnClicked: ui.editOutboundEditorMaterial},
						PushButton{AssignTo: &state.removeMaterial, Text: "移除当前行", Enabled: false, MinSize: Size{Width: 104, Height: 30}, OnClicked: ui.removeOutboundEditorMaterial},
					}},
					TableView{
						AssignTo: &state.materials, Model: state.rows, AlternatingRowBG: true, ColumnsOrderable: true, StretchFactor: 1,
						Accessibility:         Accessibility{Name: "出库单物料明细", Description: "选择一行可编辑数量和单价，或查看历史价格"},
						OnCurrentIndexChanged: ui.updateOutboundEditorSelection,
						OnItemActivated:       ui.editOutboundEditorMaterial,
						Columns: []TableViewColumn{
							{Title: "序号", DataMember: "Index", Width: 55},
							{Title: "物料名称", DataMember: "Name", Width: 180},
							{Title: "型号", DataMember: "Model", Width: 120},
							{Title: "规格", DataMember: "Specification", Width: 130},
							{Title: "单位", DataMember: "Unit", Width: 65},
							{Title: "单价", DataMember: "Price", Width: 90},
							{Title: "数量", DataMember: "Quantity", Width: 100},
							{Title: "小计", DataMember: "Amount", Width: 95},
							{Title: "价格参考", DataMember: "Reference", Width: 150},
						},
					},
				},
			},
			Composite{Layout: HBox{Spacing: 8}, Children: []Widget{
				Label{AssignTo: &state.info, Text: "正在加载客户选项……", TextColor: secondaryTextColor(), Accessibility: Accessibility{Name: "出库单编辑状态"}},
				HSpacer{},
				PushButton{AssignTo: &state.cancelButton, Text: "取消", MinSize: Size{Width: 80, Height: 30}, OnClicked: func() { ui.closeOutboundEditor(false) }},
				PushButton{AssignTo: &state.save, Text: "核对并创建", MinSize: Size{Width: 110, Height: 30}, OnClicked: ui.submitOutboundEditor},
			}},
		},
	}
}

func (ui *mainUI) newOutboundOrder() {
	if !hasButton(ui.session.Perms.Buttons, "outbound:order:add") {
		walk.MsgBox(ui.window, "没有权限", "当前账号没有新建出库单权限。", walk.MsgBoxIconWarning)
		return
	}
	if ui.outboundEditor != nil {
		_ = ui.tabs.SetCurrentIndex(ui.tabs.Pages().Index(ui.outboundEditor.tab))
		return
	}
	state := newOutboundEditorUI()
	pageDecl := ui.outboundEditorPageWidget(state)
	if err := pageDecl.Create(NewBuilder(nil)); err != nil {
		state.dispose()
		walk.MsgBox(ui.window, "无法打开出库编辑页", err.Error(), walk.MsgBoxIconError)
		return
	}
	ui.outboundEditor = state
	ui.outboundEditorTab = state.tab
	insertAt := ui.tabs.Pages().Len()
	if ui.systemTab != nil {
		if index := ui.tabs.Pages().Index(ui.systemTab); index >= 0 {
			insertAt = index
		}
	}
	if ui.outboundTab != nil {
		if index := ui.tabs.Pages().Index(ui.outboundTab); index >= 0 {
			insertAt = index + 1
		}
	}
	if err := ui.tabs.Pages().Insert(insertAt, state.tab); err != nil {
		state.tab.Dispose()
		state.dispose()
		ui.outboundEditor = nil
		ui.outboundEditorTab = nil
		walk.MsgBox(ui.window, "无法打开出库编辑页", err.Error(), walk.MsgBoxIconError)
		return
	}
	_ = ui.tabs.SetCurrentIndex(ui.tabs.Pages().Index(state.tab))
	ui.initializeOutboundEditor(state)
}

func (ui *mainUI) initializeOutboundEditor(state *outboundEditorUI) {
	state.code.SetText("O-WIN-" + time.Now().Format("20060102-150405-000"))
	state.orderType.SetCurrentIndex(0)
	ui.refreshOutboundEditorTable()
	ui.refreshOutboundEditorAttachments()
	ui.updateOutboundEditorPartnerControls()
	markDirty := func() {
		if state.initializing || state.busy || state.submitted {
			return
		}
		state.dirty = true
		state.info.SetText("存在尚未提交的修改。")
	}
	state.code.TextChanged().Attach(markDirty)
	state.remark.TextChanged().Attach(markDirty)
	state.orderType.CurrentIndexChanged().Attach(func() {
		ui.updateOutboundEditorPartnerControls()
		ui.refreshOutboundEditorPriceReferences()
		markDirty()
	})
	state.customer.CurrentIndexChanged().Attach(func() {
		ui.refreshOutboundEditorPriceReferences()
		markDirty()
	})
	state.initializing = false
	state.dirty = false
	ui.loadOutboundEditorCustomers(state)
}

func (ui *mainUI) loadOutboundEditorCustomers(state *outboundEditorUI) {
	guardedGo(func() {
		customers, err := ui.session.Client.Customers(state.ctx)
		if state.ctx.Err() != nil || state.closed.Load() {
			return
		}
		ui.window.Synchronize(func() {
			if state != ui.outboundEditor || state.closed.Load() {
				return
			}
			state.initializing = true
			for _, customer := range customers {
				state.customerOptions = append(state.customerOptions, selectOption{ID: customer.ID, Label: businessOptionLabel(customer.Name, customer.Code)})
			}
			label := "请选择客户"
			if err != nil {
				label = "客户加载失败"
			}
			_ = state.customer.SetModel(optionLabels(label, state.customerOptions))
			state.customer.SetCurrentIndex(0)
			state.initializing = false
			ui.updateOutboundEditorPartnerControls()
			if err != nil {
				state.info.SetText("客户选项加载失败：" + err.Error() + "。销售、样品和赠品出库暂不能提交，请关闭后重试。")
			} else {
				state.info.SetText("客户选项已加载；请添加出库物料。")
			}
		})
	})
}

func (ui *mainUI) updateOutboundEditorPartnerControls() {
	state := ui.outboundEditor
	if state == nil || state.orderType == nil {
		return
	}
	required := outboundTypeRequiresCustomer(state.orderType.Text())
	if required {
		state.customerLabel.SetText("客户 *")
	} else {
		state.customerLabel.SetText("客户（不适用）")
	}
	state.customer.SetEnabled(required && !state.busy && !state.submitted)
}

func (ui *mainUI) closeOutboundEditor(force bool) {
	state := ui.outboundEditor
	if state == nil || state.tab == nil || ui.tabs == nil {
		return
	}
	if !force {
		if state.busy {
			walk.MsgBox(ui.window, "正在提交", "出库单正在提交和复核，请等待完成，避免产生未知状态。", walk.MsgBoxIconInformation)
			return
		}
		if state.dirty && walk.MsgBox(ui.window, "放弃未提交修改", "当前出库单还有未提交修改，是否放弃？", walk.MsgBoxYesNo|walk.MsgBoxIconWarning) != walk.DlgCmdYes {
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
	ui.outboundEditor = nil
	ui.outboundEditorTab = nil
	ui.syncNavigationFromTab()
}

func (ui *mainUI) refreshOutboundEditorTable() {
	state := ui.outboundEditor
	if state == nil || state.materials == nil {
		return
	}
	state.refreshRows()
	_ = state.materials.SetModel(state.rows)
	_, amount, _ := state.summary()
	state.total.SetText(fmt.Sprintf("%.3f", amount))
	ui.updateOutboundEditorSelection()
}

func (ui *mainUI) updateOutboundEditorSelection() {
	state := ui.outboundEditor
	if state == nil || state.materials == nil {
		return
	}
	index := state.materials.CurrentIndex()
	enabled := index >= 0 && index < len(state.rows) && !state.busy && !state.submitted
	state.editMaterial.SetEnabled(enabled)
	state.removeMaterial.SetEnabled(enabled)
	state.priceHistory.SetEnabled(enabled)
}

func (ui *mainUI) addOutboundEditorMaterial() {
	state := ui.outboundEditor
	if state == nil || state.busy || state.submitted {
		return
	}
	material, ok := selectOutboundMaterial(ui.window, ui.session.Client)
	if !ok {
		return
	}
	for _, row := range state.rows {
		if row.Request.MaterialID == material.ID {
			state.info.SetText("该物料已经在明细中，请直接编辑当前行。")
			return
		}
	}
	row := outboundEditorMaterial{
		Name: material.Name, Model: material.Model, Specification: material.Specification, Unit: material.Unit, Material: material,
		Request: api.OutboundMaterialRequest{MaterialID: material.ID, Quantity: 1},
	}
	if !editOutboundMaterialDetails(ui.window, ui.session.Client, &row) {
		return
	}
	state.rows = append(state.rows, row)
	state.dirty = true
	ui.refreshOutboundEditorTable()
	_ = state.materials.SetCurrentIndex(len(state.rows) - 1)
	ui.refreshOutboundEditorPriceReferences()
	state.info.SetText("已添加物料；修改尚未提交。")
}

func (ui *mainUI) editOutboundEditorMaterial() {
	state := ui.outboundEditor
	if state == nil || state.busy || state.submitted {
		return
	}
	index := state.materials.CurrentIndex()
	if index < 0 || index >= len(state.rows) {
		walk.MsgBox(ui.window, "请选择物料", "请先选择需要编辑的物料行。", walk.MsgBoxIconInformation)
		return
	}
	row := state.rows[index]
	if !editOutboundMaterialDetails(ui.window, ui.session.Client, &row) {
		return
	}
	state.rows[index] = row
	state.dirty = true
	ui.refreshOutboundEditorTable()
	_ = state.materials.SetCurrentIndex(index)
	state.info.SetText("物料明细已修改；尚未提交。")
}

func (ui *mainUI) removeOutboundEditorMaterial() {
	state := ui.outboundEditor
	if state == nil || state.busy || state.submitted {
		return
	}
	index := state.materials.CurrentIndex()
	if index < 0 || index >= len(state.rows) {
		return
	}
	if walk.MsgBox(ui.window, "移除物料", fmt.Sprintf("是否从本次出库单中移除“%s”？", state.rows[index].Name), walk.MsgBoxYesNo|walk.MsgBoxIconWarning) != walk.DlgCmdYes {
		return
	}
	state.rows = append(state.rows[:index], state.rows[index+1:]...)
	state.dirty = true
	ui.refreshOutboundEditorTable()
	ui.refreshOutboundEditorPriceReferences()
	state.info.SetText("物料已从当前编辑内容移除；尚未提交。")
}

func (ui *mainUI) showOutboundEditorPriceHistory() {
	state := ui.outboundEditor
	if state == nil || state.materials == nil {
		return
	}
	index := state.materials.CurrentIndex()
	if index >= 0 && index < len(state.rows) {
		ShowMaterialPriceHistory(ui.window, ui.session.Client, state.rows[index].Material)
	}
}

func latestValidMaterialPrice(prices []api.MaterialPrice) (float64, bool) {
	var latest api.MaterialPrice
	found := false
	for _, price := range prices {
		if !price.SourceValid {
			continue
		}
		if !found || price.Since > latest.Since {
			latest = price
			found = true
		}
	}
	return latest.Price, found
}

func (ui *mainUI) refreshOutboundEditorPriceReferences() {
	state := ui.outboundEditor
	if state == nil || state.closed.Load() {
		return
	}
	state.priceGeneration++
	generation := state.priceGeneration
	if len(state.rows) == 0 {
		return
	}
	customerID := ""
	if outboundTypeRequiresCustomer(state.orderType.Text()) {
		customerID = selectedOptionID(state.customer, state.customerOptions)
	}
	for index := range state.rows {
		state.rows[index].Reference = "正在查询……"
	}
	ui.refreshOutboundEditorTable()
	materials := make([]string, len(state.rows))
	for index := range state.rows {
		materials[index] = state.rows[index].Request.MaterialID
	}
	guardedGo(func() {
		references := make([]string, len(materials))
		for index, materialID := range materials {
			prices, err := ui.session.Client.MaterialPrices(state.ctx, materialID, customerID)
			if state.ctx.Err() != nil || state.closed.Load() {
				return
			}
			if err != nil {
				references[index] = "查询失败"
				continue
			}
			if price, ok := latestValidMaterialPrice(prices); ok {
				references[index] = fmt.Sprintf("最近 %.3f", price)
			} else {
				references[index] = "暂无有效记录"
			}
		}
		ui.window.Synchronize(func() {
			if state != ui.outboundEditor || state.closed.Load() || generation != state.priceGeneration || len(state.rows) != len(references) {
				return
			}
			for index := range references {
				if state.rows[index].Request.MaterialID != materials[index] {
					return
				}
				state.rows[index].Reference = references[index]
			}
			ui.refreshOutboundEditorTable()
		})
	})
}

func editOutboundMaterialDetails(owner walk.Form, client *api.Client, row *outboundEditorMaterial) bool {
	var dlg *walk.Dialog
	var priceEdit, quantityEdit *walk.LineEdit
	var info *walk.Label
	var applyButton *walk.PushButton
	accepted := false
	apply := func() {}
	err := Dialog{
		AssignTo: &dlg,
		Title:    "编辑出库物料 - " + row.Name,
		MinSize:  Size{Width: 540, Height: 280},
		Size:     Size{Width: 650, Height: 340},
		Layout:   VBox{Margins: Margins{Left: 16, Top: 16, Right: 16, Bottom: 14}, Spacing: 10},
		Children: []Widget{
			Label{Text: row.Name, Font: Font{Family: "Microsoft YaHei UI", PointSize: 14, Bold: true}},
			Label{Text: fmt.Sprintf("型号：%s    规格：%s    单位：%s", displayMaterialValue(row.Model), displayMaterialValue(row.Specification), displayMaterialValue(row.Unit)), TextColor: secondaryTextColor()},
			GroupBox{
				Title:  "本次出库",
				Layout: Grid{Columns: 2, Margins: Margins{Left: 12, Top: 10, Right: 12, Bottom: 12}, Spacing: 8},
				Children: []Widget{
					Label{Text: "单价 *"},
					LineEdit{AssignTo: &priceEdit, Text: fmt.Sprintf("%g", row.Request.Price), CueBanner: "非负数字", Accessibility: Accessibility{Name: "物料单价"}},
					Label{Text: "出库数量 *"},
					LineEdit{AssignTo: &quantityEdit, Text: fmt.Sprintf("%g", row.Request.Quantity), CueBanner: "大于 0", Accessibility: Accessibility{Name: "物料出库数量"}},
				},
			},
			Label{AssignTo: &info, Text: "单价可以为 0；最终合法性仍由服务端校验。", TextColor: secondaryTextColor()},
			VSpacer{},
			Composite{Layout: HBox{Spacing: 8}, Children: []Widget{
				PushButton{Text: "历史价格", MinSize: Size{Width: 92, Height: 30}, OnClicked: func() { ShowMaterialPriceHistory(dlg, client, row.Material) }},
				HSpacer{},
				PushButton{Text: "取消", MinSize: Size{Width: 80, Height: 30}, OnClicked: func() { dlg.Cancel() }},
				PushButton{AssignTo: &applyButton, Text: "应用", MinSize: Size{Width: 88, Height: 30}},
			}},
		},
	}.Create(owner)
	if err != nil {
		walk.MsgBox(owner, "无法编辑物料", err.Error(), walk.MsgBoxIconError)
		return false
	}
	apply = func() {
		price, err := parseNonNegativeNumber(priceEdit.Text(), "单价")
		if err != nil {
			info.SetText(err.Error())
			priceEdit.SetFocus()
			return
		}
		quantity, err := parsePositiveNumber(quantityEdit.Text(), "出库数量")
		if err != nil {
			info.SetText(err.Error())
			quantityEdit.SetFocus()
			return
		}
		row.Request.Price = price
		row.Request.Quantity = quantity
		accepted = true
		dlg.Accept()
	}
	applyButton.Clicked().Attach(apply)
	priceEdit.KeyDown().Attach(func(key walk.Key) {
		if key == walk.KeyReturn {
			quantityEdit.SetFocus()
		}
	})
	quantityEdit.KeyDown().Attach(func(key walk.Key) {
		if key == walk.KeyReturn {
			apply()
		}
	})
	dlg.Run()
	return accepted
}

func parsePositiveNumber(text, field string) (float64, error) {
	value, err := strconv.ParseFloat(strings.TrimSpace(text), 64)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s必须是大于 0 的数字", field)
	}
	return value, nil
}

func (ui *mainUI) refreshOutboundEditorAttachments() {
	state := ui.outboundEditor
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

func (ui *mainUI) uploadOutboundEditorAttachment() {
	state := ui.outboundEditor
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
	dialog.Title = "选择出库单图片附件"
	dialog.Filter = "图片文件 (*.png;*.jpg;*.jpeg;*.gif)|*.png;*.jpg;*.jpeg;*.gif|所有文件 (*.*)|*.*"
	if ok, err := dialog.ShowOpen(ui.window); err != nil {
		walk.MsgBox(ui.window, "无法选择附件", err.Error(), walk.MsgBoxIconError)
		return
	} else if !ok {
		return
	}
	state.busy = true
	uploadCtx, uploadCancel := context.WithCancel(state.ctx)
	state.uploadCancel = uploadCancel
	ui.setOutboundEditorEnabled(false)
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
			if state != ui.outboundEditor || state.closed.Load() {
				return
			}
			state.uploadCancel = nil
			state.busy = false
			ui.setOutboundEditorEnabled(true)
			state.upload.SetText("上传图片")
			if canceled {
				state.info.SetText("附件上传已取消，当前单据未加入新附件。")
				return
			}
			if err != nil {
				state.info.SetText("附件上传失败：" + requestFailureText(err))
				return
			}
			state.annex = append(state.annex, reference)
			state.dirty = true
			ui.refreshOutboundEditorAttachments()
			state.info.SetText("附件已上传并加入当前单据；创建前仍可移除。")
		})
	})
}

func (ui *mainUI) selectOutboundEditorAttachments() {
	state := ui.outboundEditor
	if state == nil || state.busy || state.submitted {
		return
	}
	references, ok := SelectImageAssets(ui.window, ui.session.Client, config.ImageBaseURL(), 10)
	if !ok || len(references) == 0 {
		return
	}
	state.annex = appendUniqueReferences(state.annex, references, len(state.annex)+len(references))
	state.dirty = true
	ui.refreshOutboundEditorAttachments()
	state.info.SetText(fmt.Sprintf("已从素材库加入 %d 张附件；创建前仍可移除。", len(references)))
}

func (ui *mainUI) previewOutboundEditorAttachments() {
	state := ui.outboundEditor
	if state != nil {
		ShowOrderAttachments(ui.window, ui.session.Client, config.ImageBaseURL(), "出库单 "+state.code.Text(), state.annex)
	}
}

func (ui *mainUI) removeOutboundEditorAttachment() {
	state := ui.outboundEditor
	if state == nil || len(state.annex) == 0 || state.busy || state.submitted {
		return
	}
	index := state.attachment.CurrentIndex()
	if index < 0 || index >= len(state.annex) {
		index = 0
	}
	if walk.MsgBox(ui.window, "移除附件", fmt.Sprintf("是否从当前出库单中移除附件 %d？已上传图片不会从服务端图片库删除。", index+1), walk.MsgBoxYesNo|walk.MsgBoxIconWarning) != walk.DlgCmdYes {
		return
	}
	state.annex = append(state.annex[:index], state.annex[index+1:]...)
	state.dirty = true
	ui.refreshOutboundEditorAttachments()
	state.info.SetText("附件已从当前提交内容移除。")
}

func (ui *mainUI) setOutboundEditorEnabled(enabled bool) {
	state := ui.outboundEditor
	if state == nil {
		return
	}
	canEdit := enabled && !state.submitted
	state.code.SetEnabled(canEdit)
	state.orderType.SetEnabled(canEdit)
	state.remark.SetEnabled(canEdit)
	state.upload.SetEnabled(canEdit)
	state.selectAttachment.SetEnabled(canEdit)
	state.addMaterial.SetEnabled(canEdit)
	state.save.SetEnabled(canEdit)
	state.cancelButton.SetEnabled(enabled)
	ui.updateOutboundEditorPartnerControls()
	ui.updateOutboundEditorSelection()
	ui.refreshOutboundEditorAttachments()
}

func buildOutboundOrderRequest(state *outboundEditorUI) (api.OutboundOrderRequest, int, error) {
	code := strings.TrimSpace(state.code.Text())
	if code == "" {
		return api.OutboundOrderRequest{}, 0, fmt.Errorf("出库单号不能为空")
	}
	orderType := strings.TrimSpace(state.orderType.Text())
	if stringIndex(outboundCreateTypes, orderType) < 0 {
		return api.OutboundOrderRequest{}, 0, fmt.Errorf("请选择客户端支持的出库类型")
	}
	request := api.OutboundOrderRequest{
		Code: code, Type: orderType, Remark: strings.TrimSpace(state.remark.Text()), Annex: append([]string(nil), state.annex...),
	}
	if outboundTypeRequiresCustomer(orderType) {
		request.CustomerID = selectedOptionID(state.customer, state.customerOptions)
		if request.CustomerID == "" {
			return api.OutboundOrderRequest{}, 0, fmt.Errorf("%s必须选择客户", orderType)
		}
	}
	if len(state.rows) == 0 {
		return api.OutboundOrderRequest{}, 0, fmt.Errorf("至少需要添加一条物料")
	}
	seen := make(map[string]bool, len(state.rows))
	unpriced := 0
	for index, row := range state.rows {
		material := row.Request
		material.Index = index + 1
		material.MaterialID = strings.TrimSpace(material.MaterialID)
		if material.MaterialID == "" {
			return api.OutboundOrderRequest{}, 0, fmt.Errorf("第 %d 行物料编号为空", index+1)
		}
		if seen[material.MaterialID] {
			return api.OutboundOrderRequest{}, 0, fmt.Errorf("物料“%s”重复", row.Name)
		}
		seen[material.MaterialID] = true
		if material.Price < 0 {
			return api.OutboundOrderRequest{}, 0, fmt.Errorf("物料“%s”的单价不能小于 0", row.Name)
		}
		if material.Quantity <= 0 {
			return api.OutboundOrderRequest{}, 0, fmt.Errorf("物料“%s”的出库数量必须大于 0", row.Name)
		}
		if material.Price == 0 {
			unpriced++
		}
		request.Materials = append(request.Materials, material)
		request.TotalAmount += material.Price * material.Quantity
	}
	return request, unpriced, nil
}

func (ui *mainUI) submitOutboundEditor() {
	state := ui.outboundEditor
	if state == nil || state.busy || state.submitted {
		return
	}
	request, unpriced, err := buildOutboundOrderRequest(state)
	if err != nil {
		state.info.SetText("无法提交：" + err.Error())
		return
	}
	quantity, _, _ := state.summary()
	message := fmt.Sprintf("操作：新建预发货单\r\n单号：%s\r\n类型：%s\r\n物料：%d 种 / 数量 %g\r\n金额：%.3f 元\r\n未核价：%d 行", request.Code, request.Type, len(request.Materials), quantity, request.TotalAmount, unpriced)
	if unpriced > 0 {
		message += "\r\n\r\n未核价物料将以 0 单价提交，请确认这是预期操作。"
	}
	message += "\r\n\r\n是否提交到线上服务？"
	if walk.MsgBox(ui.window, "核对出库单", message, walk.MsgBoxYesNo|walk.MsgBoxIconQuestion) != walk.DlgCmdYes {
		return
	}
	state.busy = true
	ui.setOutboundEditorEnabled(false)
	state.info.SetText("正在创建出库单并在线复核……")
	guardedGo(func() {
		requestErr := ui.session.Client.CreateOutbound(state.ctx, request)
		var verifyErr error
		if requestErr == nil {
			_, verifyErr = ui.session.Client.FindOutboundByCode(state.ctx, request.Code)
		}
		if state.ctx.Err() != nil || state.closed.Load() {
			return
		}
		ui.window.Synchronize(func() {
			if state != ui.outboundEditor || state.closed.Load() {
				return
			}
			state.busy = false
			if requestErr != nil {
				ui.setOutboundEditorEnabled(true)
				state.info.SetText("创建失败：" + requestFailureText(requestErr) + "。写操作不会自动重试，请核对后再决定。")
				return
			}
			state.dirty = false
			if verifyErr != nil {
				state.submitted = true
				ui.setOutboundEditorEnabled(false)
				state.cancelButton.SetEnabled(true)
				state.info.SetText("服务端已返回成功，但在线复核失败：" + verifyErr.Error() + "。请勿重复提交，关闭后刷新列表人工核对。")
				walk.MsgBox(ui.window, "创建后复核失败", state.info.Text(), walk.MsgBoxIconWarning)
				ui.loadOutbound()
				ui.loadDashboard()
				return
			}
			ui.notifyStatusSuccess("出库单已创建，并完成线上回读复核。")
			ui.loadOutbound()
			ui.loadDashboard()
			ui.closeOutboundEditor(true)
		})
	})
}
