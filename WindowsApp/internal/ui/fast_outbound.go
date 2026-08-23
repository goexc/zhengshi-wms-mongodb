package ui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"

	"zhengshi-wms-windowsapp/internal/api"
)

var fastOutboundTypes = []string{
	api.FastOutboundSale,
	api.FastOutboundSample,
	api.FastOutboundGift,
}

type fastOutboundUI struct {
	tab          *walk.TabPage
	code         *walk.LineEdit
	orderType    *walk.ComboBox
	customer     *walk.ComboBox
	picking      *walk.DateEdit
	packing      *walk.DateEdit
	weighing     *walk.DateEdit
	departure    *walk.DateEdit
	receipt      *walk.DateEdit
	total        *walk.Label
	materials    *walk.TableView
	addMaterial  *walk.PushButton
	editMaterial *walk.PushButton
	remove       *walk.PushButton
	priceHistory *walk.PushButton
	info         *walk.Label
	save         *walk.PushButton
	cancelButton *walk.PushButton

	rows            []outboundEditorMaterial
	customerOptions []selectOption
	dirty           bool
	initializing    bool
	busy            bool
	submitted       bool
	priceGeneration int
	ctx             context.Context
	cancel          context.CancelFunc
	closed          atomic.Bool
}

type fastOutboundValidationError struct {
	Field   string
	Message string
}

func (err *fastOutboundValidationError) Error() string { return err.Message }

func newFastOutboundUI() *fastOutboundUI {
	ctx, cancel := context.WithCancel(context.Background())
	return &fastOutboundUI{ctx: ctx, cancel: cancel, initializing: true}
}

func (state *fastOutboundUI) dispose() {
	if state == nil || state.closed.Swap(true) {
		return
	}
	state.priceGeneration++
	state.cancel()
}

func (state *fastOutboundUI) refreshRows() {
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

func (state *fastOutboundUI) summary() (quantity, amount float64, unpriced int) {
	for _, row := range state.rows {
		quantity += row.Request.Quantity
		amount += row.Request.Price * row.Request.Quantity
		if row.Request.Price == 0 {
			unpriced++
		}
	}
	return
}

func (ui *mainUI) fastOutboundPageWidget(state *fastOutboundUI) TabPage {
	dateMin := time.Date(2000, 1, 1, 0, 0, 0, 0, time.Local)
	dateMax := time.Now().AddDate(1, 0, 0)
	return TabPage{
		AssignTo: &state.tab,
		Title:    closableTabTitle("极速出库"),
		Layout:   VBox{Margins: Margins{Left: 14, Top: 14, Right: 14, Bottom: 12}, Spacing: 10},
		Children: []Widget{
			Label{Text: "极速出库", Font: Font{Family: "Microsoft YaHei UI", PointSize: 15, Bold: true}},
			Label{
				Text:          "一次提交完成库存补足（需要时）、库存扣减、出库签收及客户应收记账。此操作直接写入线上数据，不会自动重试。",
				TextColor:     walk.RGB(174, 50, 42),
				Accessibility: Accessibility{Name: "极速出库业务影响提示"},
			},
			GroupBox{
				Title:  "基本信息",
				Layout: Grid{Columns: 8, Margins: Margins{Left: 14, Top: 10, Right: 14, Bottom: 12}, Spacing: 8},
				Children: []Widget{
					Label{Text: "出库单号 *"},
					LineEdit{AssignTo: &state.code, CueBanner: "必填且唯一", MinSize: Size{Width: 190, Height: 28}, Accessibility: Accessibility{Name: "极速出库单号"}},
					Label{Text: "出库类型 *"},
					ComboBox{AssignTo: &state.orderType, Model: fastOutboundTypes, CurrentIndex: 0, MinSize: Size{Width: 150, Height: 28}, Accessibility: Accessibility{Name: "极速出库类型"}},
					Label{Text: "客户 *"},
					ComboBox{AssignTo: &state.customer, Model: []string{"正在加载客户……"}, CurrentIndex: 0, MinSize: Size{Width: 210, Height: 28}, Accessibility: Accessibility{Name: "极速出库客户"}},
					Label{Text: "合计金额"},
					Label{AssignTo: &state.total, Text: "0.000", Font: Font{Bold: true}, Accessibility: Accessibility{Name: "极速出库合计金额"}},
				},
			},
			GroupBox{
				Title:  "作业日期（按本地自然日 00:00 提交）",
				Layout: Grid{Columns: 8, Margins: Margins{Left: 14, Top: 10, Right: 14, Bottom: 12}, Spacing: 8},
				Children: []Widget{
					Label{Text: "拣货日期 *"},
					DateEdit{AssignTo: &state.picking, Format: "yyyy-MM-dd", MinDate: dateMin, MaxDate: dateMax, MinSize: Size{Width: 145, Height: 28}, Accessibility: Accessibility{Name: "极速出库拣货日期"}},
					Label{Text: "打包日期"},
					DateEdit{AssignTo: &state.packing, Format: "yyyy-MM-dd", Optional: true, MinDate: dateMin, MaxDate: dateMax, MinSize: Size{Width: 145, Height: 28}, Accessibility: Accessibility{Name: "极速出库打包日期，可不填"}},
					Label{Text: "称重日期"},
					DateEdit{AssignTo: &state.weighing, Format: "yyyy-MM-dd", Optional: true, MinDate: dateMin, MaxDate: dateMax, MinSize: Size{Width: 145, Height: 28}, Accessibility: Accessibility{Name: "极速出库称重日期，可不填"}},
					Label{Text: "出库日期 *"},
					DateEdit{AssignTo: &state.departure, Format: "yyyy-MM-dd", MinDate: dateMin, MaxDate: dateMax, MinSize: Size{Width: 145, Height: 28}, Accessibility: Accessibility{Name: "极速出库出库日期"}},
					Label{Text: "签收日期 *"},
					DateEdit{AssignTo: &state.receipt, Format: "yyyy-MM-dd", MinDate: dateMin, MaxDate: dateMax, MinSize: Size{Width: 145, Height: 28}, Accessibility: Accessibility{Name: "极速出库签收日期"}},
					Label{Text: "时间约束"},
					Label{ColumnSpan: 5, Text: "拣货 → 打包（可选）→ 称重（可选）→ 出库 → 签收，均不得晚于今天。", TextColor: secondaryTextColor()},
				},
			},
			GroupBox{
				Title:         "物料明细 *",
				StretchFactor: 1,
				Layout:        VBox{Margins: Margins{Left: 12, Top: 10, Right: 12, Bottom: 12}, Spacing: 8},
				Children: []Widget{
					Composite{Layout: HBox{Spacing: 8}, Children: []Widget{
						Label{Text: "服务端可能按不足数量自动补足生产入库；请重点复核数量和单价。", TextColor: secondaryTextColor()},
						HSpacer{},
						PushButton{AssignTo: &state.priceHistory, Text: "历史价格", Enabled: false, MinSize: Size{Width: 92, Height: 30}, OnClicked: ui.showFastOutboundPriceHistory},
						PushButton{AssignTo: &state.addMaterial, Text: "添加物料", MinSize: Size{Width: 92, Height: 30}, OnClicked: ui.addFastOutboundMaterial},
						PushButton{AssignTo: &state.editMaterial, Text: "编辑当前行", Enabled: false, MinSize: Size{Width: 104, Height: 30}, OnClicked: ui.editFastOutboundMaterial},
						PushButton{AssignTo: &state.remove, Text: "移除当前行", Enabled: false, MinSize: Size{Width: 104, Height: 30}, OnClicked: ui.removeFastOutboundMaterial},
					}},
					TableView{
						AssignTo: &state.materials, Model: state.rows, AlternatingRowBG: true, ColumnsOrderable: true, StretchFactor: 1,
						Accessibility:         Accessibility{Name: "极速出库物料明细", Description: "选择一行可编辑数量和单价，或查看历史价格"},
						OnCurrentIndexChanged: ui.updateFastOutboundSelection,
						OnItemActivated:       ui.editFastOutboundMaterial,
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
				Label{AssignTo: &state.info, Text: "正在加载客户选项……", TextColor: secondaryTextColor(), Accessibility: Accessibility{Name: "极速出库编辑状态"}},
				HSpacer{},
				PushButton{AssignTo: &state.cancelButton, Text: "取消", MinSize: Size{Width: 80, Height: 30}, OnClicked: func() { ui.closeFastOutbound(false) }},
				PushButton{AssignTo: &state.save, Text: "核对并极速出库", MinSize: Size{Width: 132, Height: 30}, OnClicked: ui.submitFastOutbound},
			}},
		},
	}
}

func (ui *mainUI) openFastOutbound() {
	if !hasButton(ui.session.Perms.Buttons, "outbound:order:add") {
		walk.MsgBox(ui.window, "没有权限", "当前账号没有新建出库单权限。", walk.MsgBoxIconWarning)
		return
	}
	if ui.fastOutbound != nil && ui.fastOutboundTab != nil {
		_ = ui.tabs.SetCurrentIndex(ui.tabs.Pages().Index(ui.fastOutboundTab))
		return
	}
	state := newFastOutboundUI()
	pageDecl := ui.fastOutboundPageWidget(state)
	if err := pageDecl.Create(NewBuilder(nil)); err != nil {
		state.dispose()
		walk.MsgBox(ui.window, "无法打开极速出库", err.Error(), walk.MsgBoxIconError)
		return
	}
	ui.fastOutbound = state
	ui.fastOutboundTab = state.tab
	insertAt := ui.dynamicTabInsertionIndex()
	if ui.outboundTab != nil {
		if index := ui.tabs.Pages().Index(ui.outboundTab); index >= 0 {
			insertAt = index + 1
		}
	}
	if err := ui.tabs.Pages().Insert(insertAt, state.tab); err != nil {
		state.dispose()
		state.tab.Dispose()
		ui.fastOutbound = nil
		ui.fastOutboundTab = nil
		walk.MsgBox(ui.window, "无法打开极速出库", err.Error(), walk.MsgBoxIconError)
		return
	}
	_ = ui.tabs.SetCurrentIndex(ui.tabs.Pages().Index(state.tab))
	ui.initializeFastOutbound(state)
}

func (ui *mainUI) initializeFastOutbound(state *fastOutboundUI) {
	today := time.Now()
	state.code.SetText("O-FAST-" + today.Format("20060102-150405-000"))
	state.orderType.SetCurrentIndex(0)
	_ = state.picking.SetDate(today)
	_ = state.packing.SetDate(time.Time{})
	_ = state.weighing.SetDate(time.Time{})
	_ = state.departure.SetDate(today)
	_ = state.receipt.SetDate(today)
	ui.refreshFastOutboundTable()
	markDirty := func() {
		if state.initializing || state.busy || state.submitted {
			return
		}
		state.dirty = true
		state.info.SetText("存在尚未提交的修改。")
	}
	state.code.TextChanged().Attach(markDirty)
	state.orderType.CurrentIndexChanged().Attach(markDirty)
	state.customer.CurrentIndexChanged().Attach(func() {
		ui.refreshFastOutboundPriceReferences()
		markDirty()
	})
	for _, edit := range []*walk.DateEdit{state.picking, state.packing, state.weighing, state.departure, state.receipt} {
		edit.DateChanged().Attach(markDirty)
	}
	state.initializing = false
	state.dirty = false
	ui.loadFastOutboundCustomers(state)
}

func (ui *mainUI) loadFastOutboundCustomers(state *fastOutboundUI) {
	guardedGo(func() {
		customers, err := ui.session.Client.Customers(state.ctx)
		if state.ctx.Err() != nil || state.closed.Load() {
			return
		}
		ui.window.Synchronize(func() {
			if state != ui.fastOutbound || state.closed.Load() {
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
			ui.setFastOutboundEnabled(err == nil)
			if err != nil {
				state.info.SetText("客户选项加载失败：" + requestFailureText(err) + "。请关闭页面后重试。")
			} else {
				state.info.SetText("客户选项已加载；请添加物料并核对作业日期。")
			}
		})
	})
}

func (ui *mainUI) closeFastOutbound(force bool) {
	state := ui.fastOutbound
	if state == nil || state.tab == nil || ui.tabs == nil {
		return
	}
	if !force {
		if state.busy {
			walk.MsgBox(ui.window, "正在提交", "极速出库正在提交和复核，请等待完成，避免产生未知状态。", walk.MsgBoxIconInformation)
			return
		}
		if state.dirty && !state.submitted && walk.MsgBox(ui.window, "放弃未提交修改", "当前极速出库还有未提交修改，是否放弃？", walk.MsgBoxYesNo|walk.MsgBoxIconWarning) != walk.DlgCmdYes {
			return
		}
	}
	if index := ui.tabs.Pages().Index(state.tab); index >= 0 {
		if err := ui.tabs.Pages().RemoveAt(index); err != nil {
			walk.MsgBox(ui.window, "无法关闭极速出库", err.Error(), walk.MsgBoxIconError)
			return
		}
	}
	state.dispose()
	state.tab.Dispose()
	ui.fastOutbound = nil
	ui.fastOutboundTab = nil
	ui.syncNavigationFromTab()
}

func (ui *mainUI) refreshFastOutboundTable() {
	state := ui.fastOutbound
	if state == nil || state.materials == nil {
		return
	}
	state.refreshRows()
	_ = state.materials.SetModel(state.rows)
	_, amount, _ := state.summary()
	state.total.SetText(fmt.Sprintf("%.3f", amount))
	ui.updateFastOutboundSelection()
}

func (ui *mainUI) updateFastOutboundSelection() {
	state := ui.fastOutbound
	if state == nil || state.materials == nil {
		return
	}
	index := state.materials.CurrentIndex()
	enabled := index >= 0 && index < len(state.rows) && !state.busy && !state.submitted
	state.editMaterial.SetEnabled(enabled)
	state.remove.SetEnabled(enabled)
	state.priceHistory.SetEnabled(enabled)
}

func (ui *mainUI) addFastOutboundMaterial() {
	state := ui.fastOutbound
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
	ui.refreshFastOutboundTable()
	_ = state.materials.SetCurrentIndex(len(state.rows) - 1)
	ui.refreshFastOutboundPriceReferences()
	state.info.SetText("已添加物料；修改尚未提交。")
}

func (ui *mainUI) editFastOutboundMaterial() {
	state := ui.fastOutbound
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
	ui.refreshFastOutboundTable()
	_ = state.materials.SetCurrentIndex(index)
	state.info.SetText("物料明细已修改；尚未提交。")
}

func (ui *mainUI) removeFastOutboundMaterial() {
	state := ui.fastOutbound
	if state == nil || state.busy || state.submitted {
		return
	}
	index := state.materials.CurrentIndex()
	if index < 0 || index >= len(state.rows) {
		return
	}
	if walk.MsgBox(ui.window, "移除物料", fmt.Sprintf("是否从本次极速出库中移除“%s”？", state.rows[index].Name), walk.MsgBoxYesNo|walk.MsgBoxIconWarning) != walk.DlgCmdYes {
		return
	}
	state.rows = append(state.rows[:index], state.rows[index+1:]...)
	state.dirty = true
	ui.refreshFastOutboundTable()
	ui.refreshFastOutboundPriceReferences()
	state.info.SetText("物料已从当前编辑内容移除；尚未提交。")
}

func (ui *mainUI) showFastOutboundPriceHistory() {
	state := ui.fastOutbound
	if state == nil || state.materials == nil {
		return
	}
	index := state.materials.CurrentIndex()
	if index >= 0 && index < len(state.rows) {
		ShowMaterialPriceHistory(ui.window, ui.session.Client, state.rows[index].Material)
	}
}

func (ui *mainUI) refreshFastOutboundPriceReferences() {
	state := ui.fastOutbound
	if state == nil || state.closed.Load() || len(state.rows) == 0 {
		return
	}
	state.priceGeneration++
	generation := state.priceGeneration
	customerID := selectedOptionID(state.customer, state.customerOptions)
	for index := range state.rows {
		state.rows[index].Reference = "正在查询……"
	}
	ui.refreshFastOutboundTable()
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
			} else if price, ok := latestValidMaterialPrice(prices); ok {
				references[index] = fmt.Sprintf("最近 %.3f", price)
			} else {
				references[index] = "暂无有效记录"
			}
		}
		ui.window.Synchronize(func() {
			if state != ui.fastOutbound || state.closed.Load() || generation != state.priceGeneration || len(state.rows) != len(references) {
				return
			}
			for index := range references {
				if state.rows[index].Request.MaterialID != materials[index] {
					return
				}
				state.rows[index].Reference = references[index]
			}
			ui.refreshFastOutboundTable()
		})
	})
}

func (ui *mainUI) setFastOutboundEnabled(enabled bool) {
	state := ui.fastOutbound
	if state == nil {
		return
	}
	canEdit := enabled && !state.submitted
	for _, control := range []interface{ SetEnabled(bool) }{
		state.code, state.orderType, state.customer, state.picking, state.packing, state.weighing, state.departure, state.receipt,
		state.addMaterial, state.save,
	} {
		control.SetEnabled(canEdit)
	}
	state.cancelButton.SetEnabled(enabled)
	ui.updateFastOutboundSelection()
}

func fastOutboundDayUnix(value time.Time) int64 {
	if value.IsZero() {
		return 0
	}
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.Local).Unix()
}

func buildFastOutboundRequest(state *fastOutboundUI, now time.Time) (api.FastOutboundRequest, int, error) {
	request := api.FastOutboundRequest{
		Code:          strings.TrimSpace(state.code.Text()),
		Type:          strings.TrimSpace(state.orderType.Text()),
		CustomerID:    selectedOptionID(state.customer, state.customerOptions),
		PickingTime:   fastOutboundDayUnix(state.picking.Date()),
		PackingTime:   fastOutboundDayUnix(state.packing.Date()),
		WeighingTime:  fastOutboundDayUnix(state.weighing.Date()),
		DepartureTime: fastOutboundDayUnix(state.departure.Date()),
		ReceiptTime:   fastOutboundDayUnix(state.receipt.Date()),
	}
	if request.Code == "" {
		return request, 0, &fastOutboundValidationError{Field: "code", Message: "出库单号不能为空"}
	}
	if stringIndex(fastOutboundTypes, request.Type) < 0 {
		return request, 0, &fastOutboundValidationError{Field: "type", Message: "请选择销售、样品或赠品出库"}
	}
	if request.CustomerID == "" {
		return request, 0, &fastOutboundValidationError{Field: "customer", Message: "必须选择客户"}
	}
	if request.PickingTime == 0 {
		return request, 0, &fastOutboundValidationError{Field: "picking", Message: "必须选择拣货日期"}
	}
	if request.DepartureTime == 0 {
		return request, 0, &fastOutboundValidationError{Field: "departure", Message: "必须选择出库日期"}
	}
	if request.ReceiptTime == 0 {
		return request, 0, &fastOutboundValidationError{Field: "receipt", Message: "必须选择签收日期"}
	}
	if err := validateFastOutboundRequest(request, now); err != nil {
		return request, 0, err
	}
	if len(state.rows) == 0 {
		return request, 0, &fastOutboundValidationError{Field: "materials", Message: "至少需要添加一条物料"}
	}
	seen := make(map[string]bool, len(state.rows))
	unpriced := 0
	for index, row := range state.rows {
		materialID := strings.TrimSpace(row.Request.MaterialID)
		if materialID == "" {
			return request, unpriced, &fastOutboundValidationError{Field: "materials", Message: fmt.Sprintf("第 %d 行物料编号为空", index+1)}
		}
		if seen[materialID] {
			return request, unpriced, &fastOutboundValidationError{Field: "materials", Message: fmt.Sprintf("物料“%s”重复", row.Name)}
		}
		seen[materialID] = true
		if row.Request.Price < 0 {
			return request, unpriced, &fastOutboundValidationError{Field: "materials", Message: fmt.Sprintf("物料“%s”的单价不能小于 0", row.Name)}
		}
		if row.Request.Quantity <= 0 {
			return request, unpriced, &fastOutboundValidationError{Field: "materials", Message: fmt.Sprintf("物料“%s”的出库数量必须大于 0", row.Name)}
		}
		if row.Request.Price == 0 {
			unpriced++
		}
		request.Materials = append(request.Materials, api.FastOutboundMaterialRequest{
			MaterialID: materialID, Price: row.Request.Price, Quantity: row.Request.Quantity,
		})
	}
	return request, unpriced, nil
}

func validateFastOutboundRequest(request api.FastOutboundRequest, now time.Time) error {
	latest := now.Unix()
	fields := []struct {
		name  string
		label string
		value int64
	}{
		{name: "picking", label: "拣货日期", value: request.PickingTime},
		{name: "packing", label: "打包日期", value: request.PackingTime},
		{name: "weighing", label: "称重日期", value: request.WeighingTime},
		{name: "departure", label: "出库日期", value: request.DepartureTime},
		{name: "receipt", label: "签收日期", value: request.ReceiptTime},
	}
	previous := int64(0)
	previousLabel := ""
	for _, field := range fields {
		if field.value == 0 {
			continue
		}
		if field.value > latest {
			return &fastOutboundValidationError{Field: field.name, Message: field.label + "不能晚于今天"}
		}
		if previous > field.value {
			return &fastOutboundValidationError{Field: field.name, Message: field.label + "不能早于" + previousLabel}
		}
		previous = field.value
		previousLabel = field.label
	}
	return nil
}

func (ui *mainUI) focusFastOutboundError(err error) {
	state := ui.fastOutbound
	var validationErr *fastOutboundValidationError
	if state == nil || !errors.As(err, &validationErr) {
		return
	}
	switch validationErr.Field {
	case "code":
		state.code.SetFocus()
	case "type":
		state.orderType.SetFocus()
	case "customer":
		state.customer.SetFocus()
	case "picking":
		state.picking.SetFocus()
	case "packing":
		state.packing.SetFocus()
	case "weighing":
		state.weighing.SetFocus()
	case "departure":
		state.departure.SetFocus()
	case "receipt":
		state.receipt.SetFocus()
	case "materials":
		state.materials.SetFocus()
	}
}

func (ui *mainUI) submitFastOutbound() {
	state := ui.fastOutbound
	if state == nil || state.busy || state.submitted {
		return
	}
	request, unpriced, err := buildFastOutboundRequest(state, time.Now())
	if err != nil {
		state.info.SetText("无法提交：" + err.Error())
		ui.focusFastOutboundError(err)
		return
	}
	quantity, amount, _ := state.summary()
	message := fmt.Sprintf("操作：极速出库（直接完成签收）\r\n单号：%s\r\n类型：%s\r\n客户：%s\r\n物料：%d 种 / 数量 %g\r\n金额：%.3f 元\r\n未核价：%d 行\r\n\r\n服务端可能自动补足生产入库、扣减库存并生成客户应收。", request.Code, request.Type, state.customer.Text(), len(request.Materials), quantity, amount, unpriced)
	if unpriced > 0 {
		message += "\r\n\r\n未核价物料将以 0 单价提交。"
	}
	message += "\r\n\r\n此写操作不会自动重试，是否提交到线上服务？"
	if walk.MsgBox(ui.window, "核对极速出库", message, walk.MsgBoxYesNo|walk.MsgBoxIconWarning) != walk.DlgCmdYes {
		return
	}
	state.busy = true
	ui.setFastOutboundEnabled(false)
	state.info.SetText("正在执行极速出库并在线回读复核……")
	guardedGo(func() {
		requestErr := ui.session.Client.FastOutbound(state.ctx, request)
		verified := false
		var verifyErr error
		if requestErr == nil {
			_, verifyErr = ui.session.Client.FindOutboundByCode(state.ctx, request.Code)
			verified = verifyErr == nil
		} else {
			var transportErr *api.TransportError
			if errors.As(requestErr, &transportErr) {
				_, verifyErr = ui.session.Client.FindOutboundByCode(state.ctx, request.Code)
				verified = verifyErr == nil
			}
		}
		if state.ctx.Err() != nil || state.closed.Load() {
			return
		}
		ui.window.Synchronize(func() {
			if state != ui.fastOutbound || state.closed.Load() {
				return
			}
			state.busy = false
			if requestErr != nil && !verified {
				var transportErr *api.TransportError
				if errors.As(requestErr, &transportErr) {
					state.submitted = true
					state.dirty = false
					ui.setFastOutboundEnabled(false)
					state.cancelButton.SetEnabled(true)
					state.info.SetText(requestFailureText(requestErr) + "；回读也未确认结果。本页已禁止重复提交，请关闭并刷新出库列表核对。")
					walk.MsgBox(ui.window, "极速出库结果未确认", state.info.Text(), walk.MsgBoxIconWarning)
					ui.refreshAfterFastOutbound(request.CustomerID)
					return
				}
				ui.setFastOutboundEnabled(true)
				state.info.SetText("极速出库失败：" + requestFailureText(requestErr) + "。服务端未返回成功，请修正后手动重试。")
				return
			}
			state.dirty = false
			if requestErr == nil && !verified {
				state.submitted = true
				ui.setFastOutboundEnabled(false)
				state.cancelButton.SetEnabled(true)
				state.info.SetText("服务端已返回成功，但在线回读复核失败：" + verifyErr.Error() + "。本页已禁止重复提交，请关闭并刷新核对。")
				walk.MsgBox(ui.window, "极速出库复核失败", state.info.Text(), walk.MsgBoxIconWarning)
				ui.refreshAfterFastOutbound(request.CustomerID)
				return
			}
			message := "服务端已返回成功，并已在线重新查询到该出库单。"
			if requestErr != nil && verified {
				message = "提交响应未确认，但已在线查询到同单号出库单；客户端未重复提交。"
			}
			walk.MsgBox(ui.window, "极速出库完成", message, walk.MsgBoxIconInformation)
			ui.refreshAfterFastOutbound(request.CustomerID)
			ui.closeFastOutbound(true)
		})
	})
}

func (ui *mainUI) refreshAfterFastOutbound(customerID string) {
	ui.loadOutbound()
	ui.loadDashboard()
	if ui.inventoryTab != nil {
		ui.loadInventory()
	}
	if ui.customerFinance != nil && strings.TrimSpace(ui.customerFinance.customer.ID) == strings.TrimSpace(customerID) {
		ui.loadCustomerFinanceTransactions()
	}
}
