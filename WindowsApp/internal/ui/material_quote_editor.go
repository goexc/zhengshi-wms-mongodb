package ui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"

	"zhengshi-wms-windowsapp/internal/api"
)

type quoteCostSection struct {
	Code, Name string
	Defaults   []string
}

var quoteCostSections = []quoteCostSection{
	{"material", "材料成本", []string{"原材料", "辅料", "材料损耗"}},
	{"process", "加工工序成本", []string{"下料", "折弯", "焊接", "机加工", "表面处理"}},
	{"labor_equipment", "人工/设备成本", []string{"人工工时", "设备工时", "调机费"}},
	{"quality", "质量成本", []string{"检验", "返工预估"}},
	{"packing_logistics", "包装/物流成本", []string{"清点", "标签", "打包", "纸箱", "运费", "装卸费"}},
	{"management", "管理成本", []string{"管理费", "财务成本"}},
	{"tooling", "模具/治具摊销", []string{"模具摊销", "治具摊销"}},
	{"loss", "损耗成本", []string{"生产损耗", "异常损耗"}},
	{"other", "其他成本", []string{"其他"}},
}

type materialQuoteCostRow struct {
	Category string
	Index    string
	Name     string
	Enabled  string
	Amount   string
	Remark   string
	Item     api.MaterialQuoteCostItem
}

type materialQuoteEditorUI struct {
	tab          *walk.TabPage
	delivery     api.NewCustomerMaterial
	baseline     api.MaterialQuote
	baseTitle    string
	status       *walk.Label
	quoteMode    *walk.ComboBox
	currency     *walk.LineEdit
	validFrom    *walk.DateEdit
	validTo      *walk.DateEdit
	simplePrice  *walk.LineEdit
	profitAmount *walk.LineEdit
	taxRate      *walk.LineEdit
	finalPrice   *walk.LineEdit
	remark       *walk.LineEdit
	costTable    *walk.TableView
	costAdd      *walk.PushButton
	costEdit     *walk.PushButton
	costRemove   *walk.PushButton
	summary      *walk.Label
	info         *walk.Label
	save         *walk.PushButton
	submit       *walk.PushButton
	price        *walk.PushButton
	void         *walk.PushButton
	export       *walk.PushButton
	cancelButton *walk.PushButton
	costItems    []api.MaterialQuoteCostItem
	dirty        bool
	initializing bool
	busy         bool
	writeLocked  bool
	ctx          context.Context
	cancel       context.CancelFunc
	closed       atomic.Bool
}

func newMaterialQuoteEditorUI(delivery api.NewCustomerMaterial) *materialQuoteEditorUI {
	ctx, cancel := context.WithCancel(context.Background())
	return &materialQuoteEditorUI{delivery: delivery, baseTitle: "物料报价 · " + displayMaterialValue(delivery.MaterialModel), initializing: true, ctx: ctx, cancel: cancel}
}

func (state *materialQuoteEditorUI) dispose() {
	if state == nil || state.closed.Swap(true) {
		return
	}
	if state.cancel != nil {
		state.cancel()
	}
}

func defaultMaterialQuoteCostItems() []api.MaterialQuoteCostItem {
	var items []api.MaterialQuoteCostItem
	for _, section := range quoteCostSections {
		for index, name := range section.Defaults {
			items = append(items, api.MaterialQuoteCostItem{Index: index + 1, CategoryCode: section.Code, CategoryName: section.Name, Name: name, Enabled: section.Code == "material" && index == 0})
		}
	}
	return items
}

func (state *materialQuoteEditorUI) setDirty(dirty bool) {
	state.dirty = dirty
	if state.tab != nil {
		title := state.baseTitle
		if dirty {
			title += " *"
		}
		_ = state.tab.SetTitle(closableTabTitle(title))
	}
}

func (ui *mainUI) materialQuoteEditorPageWidget(state *materialQuoteEditorUI) TabPage {
	return TabPage{
		AssignTo: &state.tab,
		Title:    closableTabTitle(state.baseTitle),
		Layout:   VBox{Margins: Margins{Left: 14, Top: 14, Right: 14, Bottom: 12}, Spacing: 10},
		Children: []Widget{
			Composite{Layout: HBox{Spacing: 8}, Children: []Widget{
				Label{Text: "客户新增物料报价", Font: Font{Family: "Microsoft YaHei UI", PointSize: 15, Bold: true}},
				Label{AssignTo: &state.status, Text: "正在读取报价状态……", TextColor: walk.RGB(50, 95, 145)},
				HSpacer{},
				Label{Text: "服务端金额为准", TextColor: secondaryTextColor()},
			}},
			GroupBox{Title: "来源信息", Layout: Grid{Columns: 8, Margins: Margins{Left: 14, Top: 10, Right: 14, Bottom: 12}, Spacing: 8}, Children: []Widget{
				Label{Text: "客户"}, Label{Text: displayMaterialValue(state.delivery.CustomerName)},
				Label{Text: "物料"}, Label{Text: displayMaterialValue(state.delivery.MaterialName)},
				Label{Text: "型号"}, Label{Text: displayMaterialValue(state.delivery.MaterialModel)},
				Label{Text: "首次出库单"}, Label{Text: displayMaterialValue(state.delivery.FirstDeliveryOrderCode)},
			}},
			GroupBox{Title: "报价参数", Layout: Grid{Columns: 8, Margins: Margins{Left: 14, Top: 10, Right: 14, Bottom: 12}, Spacing: 8}, Children: []Widget{
				Label{Text: "报价方式 *"}, ComboBox{AssignTo: &state.quoteMode, Model: []string{"详细报价", "简单报价"}, CurrentIndex: 0, MinSize: Size{Width: 150, Height: 28}, OnCurrentIndexChanged: ui.onMaterialQuoteModeChanged},
				Label{Text: "币种"}, LineEdit{AssignTo: &state.currency, Text: "CNY", MinSize: Size{Width: 150, Height: 28}},
				Label{Text: "有效开始"}, DateEdit{AssignTo: &state.validFrom, Format: "yyyy-MM-dd", Optional: true, MinSize: Size{Width: 150, Height: 28}},
				Label{Text: "有效结束"}, DateEdit{AssignTo: &state.validTo, Format: "yyyy-MM-dd", Optional: true, MinSize: Size{Width: 150, Height: 28}},
				Label{Text: "简单非税报价"}, LineEdit{AssignTo: &state.simplePrice, Text: "0", CueBanner: "简单报价必填", MinSize: Size{Width: 150, Height: 28}},
				Label{Text: "利润金额"}, LineEdit{AssignTo: &state.profitAmount, Text: "0", CueBanner: "详细报价使用", MinSize: Size{Width: 150, Height: 28}},
				Label{Text: "税率 (%)"}, LineEdit{AssignTo: &state.taxRate, Text: "0", CueBanner: "例如 13", MinSize: Size{Width: 150, Height: 28}},
				Label{Text: "最终报价单价"}, LineEdit{AssignTo: &state.finalPrice, Text: "0", CueBanner: "0 表示采用测算值", MinSize: Size{Width: 150, Height: 28}},
				Label{Text: "备注"}, LineEdit{AssignTo: &state.remark, ColumnSpan: 7},
			}},
			GroupBox{Title: "详细报价成本项", Layout: VBox{Margins: Margins{Left: 12, Top: 10, Right: 12, Bottom: 12}, Spacing: 8}, Children: []Widget{
				Composite{Layout: HBox{Spacing: 8}, Children: []Widget{
					PushButton{AssignTo: &state.costAdd, Text: "增加成本项", MinSize: Size{Width: 100, Height: 30}, OnClicked: ui.addMaterialQuoteCostItem},
					PushButton{AssignTo: &state.costEdit, Text: "编辑成本项", Enabled: false, MinSize: Size{Width: 100, Height: 30}, OnClicked: ui.editMaterialQuoteCostItem},
					PushButton{AssignTo: &state.costRemove, Text: "移除当前项", Enabled: false, MinSize: Size{Width: 100, Height: 30}, OnClicked: ui.removeMaterialQuoteCostItem},
					HSpacer{}, Label{Text: "移除仅影响当前报价表单，不删除服务端主数据。", TextColor: secondaryTextColor()},
				}},
				TableView{AssignTo: &state.costTable, Model: []materialQuoteCostRow{}, AlternatingRowBG: true, StretchFactor: 1, MinSize: Size{Height: 210}, OnCurrentIndexChanged: ui.updateMaterialQuoteCostActions, OnItemActivated: ui.editMaterialQuoteCostItem, Columns: []TableViewColumn{
					{Title: "成本类型", DataMember: "Category", Width: 145}, {Title: "序号", DataMember: "Index", Width: 55}, {Title: "成本项", DataMember: "Name", Width: 170},
					{Title: "启用", DataMember: "Enabled", Width: 58, Alignment: AlignCenter}, {Title: "金额", DataMember: "Amount", Width: 100}, {Title: "备注", DataMember: "Remark", Width: 220},
				}},
			}},
			Label{AssignTo: &state.summary, Text: "本地测算：成本 0.000；提交后以服务端回读金额为准。", TextColor: walk.RGB(55, 85, 115), Accessibility: Accessibility{Name: "报价金额摘要"}},
			Composite{Layout: HBox{Spacing: 8}, Children: []Widget{
				Label{AssignTo: &state.info, Text: "正在读取线上报价……", TextColor: secondaryTextColor(), Accessibility: Accessibility{Name: "报价编辑状态"}},
				HSpacer{},
				PushButton{AssignTo: &state.export, Text: "导出", Enabled: false, MinSize: Size{Width: 76, Height: 30}, OnClicked: ui.exportCurrentMaterialQuote},
				PushButton{AssignTo: &state.void, Text: "作废", Enabled: false, MinSize: Size{Width: 76, Height: 30}, OnClicked: ui.voidCurrentMaterialQuote},
				PushButton{AssignTo: &state.price, Text: "最终定价", Enabled: false, MinSize: Size{Width: 88, Height: 30}, OnClicked: ui.priceCurrentMaterialQuote},
				PushButton{AssignTo: &state.submit, Text: "提交报价", Enabled: false, MinSize: Size{Width: 88, Height: 30}, OnClicked: ui.submitCurrentMaterialQuote},
				PushButton{AssignTo: &state.cancelButton, Text: "关闭", MinSize: Size{Width: 76, Height: 30}, OnClicked: func() { ui.closeMaterialQuoteEditor(false) }},
				PushButton{AssignTo: &state.save, Text: "核对并保存", Enabled: false, MinSize: Size{Width: 110, Height: 30}, OnClicked: ui.saveCurrentMaterialQuote},
			}},
		},
	}
}

func (ui *mainUI) openMaterialQuoteEditor(delivery api.NewCustomerMaterial) {
	if current := ui.materialQuoteEditor; current != nil {
		if current.busy {
			walk.MsgBox(ui.window, "正在处理", "当前报价正在提交或回读，请等待完成。", walk.MsgBoxIconInformation)
			return
		}
		if current.delivery.ID == delivery.ID {
			_ = ui.tabs.SetCurrentIndex(ui.tabs.Pages().Index(current.tab))
			return
		}
		if current.dirty && walk.MsgBox(ui.window, "替换未提交报价", "当前报价有未提交修改，是否放弃并打开另一条？", walk.MsgBoxYesNo|walk.MsgBoxIconWarning) != walk.DlgCmdYes {
			return
		}
		ui.closeMaterialQuoteEditor(true)
	}
	state := newMaterialQuoteEditorUI(delivery)
	decl := ui.materialQuoteEditorPageWidget(state)
	if err := decl.Create(NewBuilder(nil)); err != nil {
		state.dispose()
		walk.MsgBox(ui.window, "无法打开报价编辑页", err.Error(), walk.MsgBoxIconError)
		return
	}
	ui.materialQuoteEditor = state
	ui.materialQuoteEditorTab = state.tab
	insertAt := ui.tabs.Pages().Len()
	if ui.systemTab != nil {
		if index := ui.tabs.Pages().Index(ui.systemTab); index >= 0 {
			insertAt = index
		}
	}
	if ui.materialQuoteTab != nil {
		if index := ui.tabs.Pages().Index(ui.materialQuoteTab); index >= 0 {
			insertAt = index + 1
		}
	}
	if err := ui.tabs.Pages().Insert(insertAt, state.tab); err != nil {
		state.tab.Dispose()
		state.dispose()
		ui.materialQuoteEditor = nil
		ui.materialQuoteEditorTab = nil
		walk.MsgBox(ui.window, "无法打开报价编辑页", err.Error(), walk.MsgBoxIconError)
		return
	}
	_ = ui.tabs.SetCurrentIndex(ui.tabs.Pages().Index(state.tab))
	ui.initializeMaterialQuoteEditor(state)
}

func (ui *mainUI) initializeMaterialQuoteEditor(state *materialQuoteEditorUI) {
	state.costItems = defaultMaterialQuoteCostItems()
	ui.refreshMaterialQuoteCostTable()
	markDirty := func() {
		if state.initializing || state.busy || state.writeLocked {
			return
		}
		state.setDirty(true)
		state.info.SetText("存在尚未提交的修改。")
		ui.refreshMaterialQuoteSummary()
		ui.updateMaterialQuoteEditorActions()
	}
	for _, edit := range []*walk.LineEdit{state.currency, state.simplePrice, state.profitAmount, state.taxRate, state.finalPrice, state.remark} {
		edit.TextChanged().Attach(markDirty)
	}
	state.quoteMode.CurrentIndexChanged().Attach(markDirty)
	state.validFrom.DateChanged().Attach(markDirty)
	state.validTo.DateChanged().Attach(markDirty)
	state.initializing = false
	ui.onMaterialQuoteModeChanged()
	ui.loadCurrentMaterialQuote(state)
}

func (ui *mainUI) loadCurrentMaterialQuote(state *materialQuoteEditorUI) {
	if state.delivery.LatestQuoteID == "" {
		state.status.SetText("状态：尚未创建")
		state.setDirty(false)
		state.info.SetText("已载入 Web 端默认成本项；请填写并保存。")
		ui.updateMaterialQuoteEditorActions()
		return
	}
	ui.setMaterialQuoteEditorBusy(state, true, "正在读取线上最新报价……")
	guardedGo(func() {
		quote, err := ui.session.Client.MaterialQuoteInfo(state.ctx, state.delivery.LatestQuoteID)
		if state.ctx.Err() != nil || state.closed.Load() {
			return
		}
		ui.window.Synchronize(func() {
			if state != ui.materialQuoteEditor || state.closed.Load() {
				return
			}
			if err != nil {
				ui.setMaterialQuoteEditorBusy(state, false, requestFailureText(err))
				state.writeLocked = true
				ui.updateMaterialQuoteEditorActions()
				return
			}
			ui.applyMaterialQuoteToEditor(state, quote)
			ui.setMaterialQuoteEditorBusy(state, false, "报价已从线上读取；服务端金额已同步。")
		})
	})
}

func (ui *mainUI) applyMaterialQuoteToEditor(state *materialQuoteEditorUI, quote api.MaterialQuote) {
	state.initializing = true
	state.baseline = quote
	modeIndex := 0
	if quote.QuoteMode == "simple" {
		modeIndex = 1
	}
	state.quoteMode.SetCurrentIndex(modeIndex)
	state.currency.SetText(quote.Currency)
	if strings.TrimSpace(quote.Currency) == "" {
		state.currency.SetText("CNY")
	}
	state.simplePrice.SetText(fmt.Sprintf("%.3f", quote.SimplePrice))
	state.profitAmount.SetText(fmt.Sprintf("%.3f", quote.ProfitAmount))
	state.taxRate.SetText(fmt.Sprintf("%.3f", quote.TaxRate*100))
	state.finalPrice.SetText(fmt.Sprintf("%.3f", quote.FinalPrice))
	state.remark.SetText(quote.Remark)
	if quote.ValidFrom > 0 {
		state.validFrom.SetDate(time.Unix(quote.ValidFrom, 0))
	}
	if quote.ValidTo > 0 {
		state.validTo.SetDate(time.Unix(quote.ValidTo, 0))
	}
	if len(quote.CostItems) > 0 {
		state.costItems = append([]api.MaterialQuoteCostItem(nil), quote.CostItems...)
	} else {
		state.costItems = defaultMaterialQuoteCostItems()
	}
	state.initializing = false
	state.setDirty(false)
	ui.refreshMaterialQuoteCostTable()
	ui.onMaterialQuoteModeChanged()
	ui.updateMaterialQuoteEditorActions()
}

func (ui *mainUI) closeMaterialQuoteEditor(force bool) {
	state := ui.materialQuoteEditor
	if state == nil || state.tab == nil || ui.tabs == nil {
		return
	}
	if !force {
		if state.busy {
			walk.MsgBox(ui.window, "正在处理", "报价正在提交或回读，请等待完成。", walk.MsgBoxIconInformation)
			return
		}
		if state.dirty && walk.MsgBox(ui.window, "放弃未提交报价", "当前报价还有未提交修改，是否放弃？", walk.MsgBoxYesNo|walk.MsgBoxIconWarning) != walk.DlgCmdYes {
			return
		}
	}
	if index := ui.tabs.Pages().Index(state.tab); index >= 0 {
		if err := ui.tabs.Pages().RemoveAt(index); err != nil {
			walk.MsgBox(ui.window, "无法关闭报价编辑页", err.Error(), walk.MsgBoxIconError)
			return
		}
	}
	state.dispose()
	state.tab.Dispose()
	ui.materialQuoteEditor = nil
	ui.materialQuoteEditorTab = nil
	ui.syncNavigationFromTab()
	if ui.materialQuote != nil && ui.materialQuote.table != nil && selectedOptionID(ui.materialQuote.customer, ui.materialQuote.customers) != "" {
		ui.loadMaterialQuotes()
	}
}

func (ui *mainUI) onMaterialQuoteModeChanged() {
	state := ui.materialQuoteEditor
	if state == nil || state.quoteMode == nil {
		return
	}
	detailed := state.quoteMode.CurrentIndex() != 1
	state.simplePrice.SetEnabled(!detailed && !state.busy && !state.writeLocked)
	state.profitAmount.SetEnabled(detailed && !state.busy && !state.writeLocked)
	state.costTable.SetEnabled(detailed && !state.busy && !state.writeLocked)
	ui.updateMaterialQuoteCostActions()
	ui.refreshMaterialQuoteSummary()
}

func materialQuoteCostRows(items []api.MaterialQuoteCostItem) []materialQuoteCostRow {
	ordered := append([]api.MaterialQuoteCostItem(nil), items...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].CategoryCode == ordered[j].CategoryCode {
			return ordered[i].Index < ordered[j].Index
		}
		return quoteCostSectionOrder(ordered[i].CategoryCode) < quoteCostSectionOrder(ordered[j].CategoryCode)
	})
	rows := make([]materialQuoteCostRow, 0, len(ordered))
	for _, item := range ordered {
		enabled := "✘"
		if item.Enabled {
			enabled = "✔"
		}
		rows = append(rows, materialQuoteCostRow{item.CategoryName, strconv.Itoa(item.Index), item.Name, enabled, fmt.Sprintf("%.3f", item.Amount), item.Remark, item})
	}
	return rows
}

func quoteCostSectionOrder(code string) int {
	for index, section := range quoteCostSections {
		if section.Code == code {
			return index
		}
	}
	return len(quoteCostSections)
}

func (ui *mainUI) refreshMaterialQuoteCostTable() {
	state := ui.materialQuoteEditor
	if state == nil || state.costTable == nil {
		return
	}
	_ = state.costTable.SetModel(materialQuoteCostRows(state.costItems))
	ui.updateMaterialQuoteCostActions()
	ui.refreshMaterialQuoteSummary()
}

func (ui *mainUI) selectedMaterialQuoteCostIndex() int {
	state := ui.materialQuoteEditor
	if state == nil || state.costTable == nil {
		return -1
	}
	rowIndex := state.costTable.CurrentIndex()
	rows := materialQuoteCostRows(state.costItems)
	if rowIndex < 0 || rowIndex >= len(rows) {
		return -1
	}
	selected := rows[rowIndex].Item
	for index, item := range state.costItems {
		if item.CategoryCode == selected.CategoryCode && item.Index == selected.Index && item.Name == selected.Name {
			return index
		}
	}
	return -1
}

func (ui *mainUI) updateMaterialQuoteCostActions() {
	state := ui.materialQuoteEditor
	if state == nil || state.costAdd == nil {
		return
	}
	editable := state.quoteMode.CurrentIndex() != 1 && !state.busy && !state.writeLocked && state.baseline.Status != "priced" && state.baseline.Status != "void"
	selected := ui.selectedMaterialQuoteCostIndex() >= 0
	state.costAdd.SetEnabled(editable)
	state.costEdit.SetEnabled(editable && selected)
	state.costRemove.SetEnabled(editable && selected)
}

func (ui *mainUI) addMaterialQuoteCostItem() { ui.showMaterialQuoteCostDialog(-1) }
func (ui *mainUI) editMaterialQuoteCostItem() {
	ui.showMaterialQuoteCostDialog(ui.selectedMaterialQuoteCostIndex())
}

func (ui *mainUI) showMaterialQuoteCostDialog(index int) {
	state := ui.materialQuoteEditor
	if state == nil || state.busy || state.writeLocked {
		return
	}
	item := api.MaterialQuoteCostItem{Enabled: true, Custom: true}
	if index >= 0 && index < len(state.costItems) {
		item = state.costItems[index]
	}
	var dlg *walk.Dialog
	var category, enabled *walk.ComboBox
	var name, amount, remark *walk.LineEdit
	sectionLabels := make([]string, len(quoteCostSections))
	sectionIndex := 0
	for i, section := range quoteCostSections {
		sectionLabels[i] = section.Name
		if section.Code == item.CategoryCode {
			sectionIndex = i
		}
	}
	title := "增加成本项"
	if index >= 0 {
		title = "编辑成本项"
	}
	enabledIndex := 1
	if item.Enabled {
		enabledIndex = 0
	}
	err := Dialog{
		AssignTo: &dlg,
		Title:    title,
		MinSize:  Size{Width: 560, Height: 280},
		Size:     Size{Width: 620, Height: 330},
		Layout:   VBox{Margins: Margins{Left: 16, Top: 16, Right: 16, Bottom: 14}, Spacing: 10},
		Children: []Widget{
			Label{Text: title, Font: Font{PointSize: 14, Bold: true}},
			GroupBox{
				Title:  "成本项",
				Layout: Grid{Columns: 4, Margins: Margins{Left: 12, Top: 10, Right: 12, Bottom: 12}, Spacing: 8},
				Children: []Widget{
					Label{Text: "成本类型 *"},
					ComboBox{AssignTo: &category, Model: sectionLabels, CurrentIndex: sectionIndex, MinSize: Size{Width: 190, Height: 28}},
					Label{Text: "是否启用"},
					ComboBox{AssignTo: &enabled, Model: []string{"启用", "不启用"}, CurrentIndex: enabledIndex, MinSize: Size{Width: 190, Height: 28}},
					Label{Text: "成本项名称 *"},
					LineEdit{AssignTo: &name, Text: item.Name, MinSize: Size{Width: 190, Height: 28}},
					Label{Text: "金额"},
					LineEdit{AssignTo: &amount, Text: fmt.Sprintf("%.3f", item.Amount), MinSize: Size{Width: 190, Height: 28}},
					Label{Text: "备注"},
					LineEdit{AssignTo: &remark, Text: item.Remark, ColumnSpan: 3},
				},
			},
			VSpacer{},
			Composite{Layout: HBox{Spacing: 8}, Children: []Widget{
				HSpacer{},
				PushButton{Text: "取消", OnClicked: func() { dlg.Cancel() }},
				PushButton{Text: "确定", OnClicked: func() {
					sectionAt := category.CurrentIndex()
					if sectionAt < 0 || sectionAt >= len(quoteCostSections) || strings.TrimSpace(name.Text()) == "" {
						walk.MsgBox(dlg, "请完善成本项", "请选择成本类型并填写成本项名称。", walk.MsgBoxIconWarning)
						return
					}
					value, parseErr := parseNonNegativeNumber(amount.Text(), "成本金额")
					if parseErr != nil {
						walk.MsgBox(dlg, "金额格式错误", parseErr.Error(), walk.MsgBoxIconWarning)
						return
					}
					section := quoteCostSections[sectionAt]
					updated := item
					updated.CategoryCode = section.Code
					updated.CategoryName = section.Name
					updated.Name = strings.TrimSpace(name.Text())
					updated.Enabled = enabled.CurrentIndex() == 0
					updated.Amount = value
					updated.Remark = strings.TrimSpace(remark.Text())
					updated.Custom = true
					if index >= 0 {
						state.costItems[index] = updated
					} else {
						state.costItems = append(state.costItems, updated)
					}
					normalizeMaterialQuoteCostIndexes(state.costItems)
					state.setDirty(true)
					ui.refreshMaterialQuoteCostTable()
					ui.updateMaterialQuoteEditorActions()
					dlg.Accept()
				}},
			}},
		},
	}.Create(ui.window)
	if err != nil {
		walk.MsgBox(ui.window, "成本项窗口错误", err.Error(), walk.MsgBoxIconError)
		return
	}
	dlg.Run()
}

func normalizeMaterialQuoteCostIndexes(items []api.MaterialQuoteCostItem) {
	counts := map[string]int{}
	sort.SliceStable(items, func(i, j int) bool {
		oi, oj := quoteCostSectionOrder(items[i].CategoryCode), quoteCostSectionOrder(items[j].CategoryCode)
		if oi == oj {
			return items[i].Index < items[j].Index
		}
		return oi < oj
	})
	for index := range items {
		counts[items[index].CategoryCode]++
		items[index].Index = counts[items[index].CategoryCode]
	}
}

func (ui *mainUI) removeMaterialQuoteCostItem() {
	state := ui.materialQuoteEditor
	index := ui.selectedMaterialQuoteCostIndex()
	if state == nil || index < 0 {
		return
	}
	if walk.MsgBox(ui.window, "移除成本项", "仅从当前未提交报价表单中移除该成本项，是否继续？", walk.MsgBoxYesNo|walk.MsgBoxIconWarning) != walk.DlgCmdYes {
		return
	}
	state.costItems = append(state.costItems[:index], state.costItems[index+1:]...)
	normalizeMaterialQuoteCostIndexes(state.costItems)
	state.setDirty(true)
	ui.refreshMaterialQuoteCostTable()
	ui.updateMaterialQuoteEditorActions()
}

func parseMaterialQuoteNumber(textValue, name string) (float64, error) {
	value, err := strconv.ParseFloat(strings.TrimSpace(textValue), 64)
	if err != nil || value < 0 {
		return 0, fmt.Errorf("%s必须是非负数字", name)
	}
	return value, nil
}

func (ui *mainUI) materialQuoteEditorRequest(state *materialQuoteEditorUI) (api.MaterialQuoteSaveRequest, error) {
	request := api.MaterialQuoteSaveRequest{ID: state.baseline.ID, DeliveryID: state.delivery.ID, Currency: strings.TrimSpace(state.currency.Text()), Remark: strings.TrimSpace(state.remark.Text())}
	if request.Currency == "" {
		request.Currency = "CNY"
	}
	if state.quoteMode.CurrentIndex() == 1 {
		request.QuoteMode = "simple"
	} else {
		request.QuoteMode = "detailed"
	}
	var err error
	request.SimplePrice, err = parseMaterialQuoteNumber(state.simplePrice.Text(), "简单非税报价")
	if err != nil {
		return request, err
	}
	request.ProfitAmount, err = parseMaterialQuoteNumber(state.profitAmount.Text(), "利润金额")
	if err != nil {
		return request, err
	}
	taxPercent, err := parseMaterialQuoteNumber(state.taxRate.Text(), "税率")
	if err != nil {
		return request, err
	}
	request.TaxRate = taxPercent / 100
	request.FinalPrice, err = parseMaterialQuoteNumber(state.finalPrice.Text(), "最终报价单价")
	if err != nil {
		return request, err
	}
	from, to := state.validFrom.Date(), state.validTo.Date()
	if !from.IsZero() {
		request.ValidFrom = time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, from.Location()).Unix()
	}
	if !to.IsZero() {
		request.ValidTo = time.Date(to.Year(), to.Month(), to.Day(), 23, 59, 59, 0, to.Location()).Unix()
	}
	if request.ValidFrom > 0 && request.ValidTo > 0 && request.ValidTo < request.ValidFrom {
		return request, fmt.Errorf("有效结束日期不能早于开始日期")
	}
	if request.QuoteMode == "simple" {
		if request.SimplePrice <= 0 {
			return request, fmt.Errorf("简单报价必须填写大于 0 的非税报价")
		}
		request.CostItems = nil
		request.ProfitAmount = 0
	} else {
		request.SimplePrice = 0
		request.CostItems = append([]api.MaterialQuoteCostItem(nil), state.costItems...)
		normalizeMaterialQuoteCostIndexes(request.CostItems)
		hasCost := false
		for _, item := range request.CostItems {
			if item.Enabled && item.Amount > 0 {
				hasCost = true
				break
			}
		}
		if !hasCost && request.FinalPrice <= 0 {
			return request, fmt.Errorf("详细报价需要启用金额大于 0 的成本项，或填写最终报价单价")
		}
	}
	return request, nil
}

func (ui *mainUI) refreshMaterialQuoteSummary() {
	state := ui.materialQuoteEditor
	if state == nil || state.summary == nil {
		return
	}
	simple, _ := parseMaterialQuoteNumber(state.simplePrice.Text(), "简单报价")
	profit, _ := parseMaterialQuoteNumber(state.profitAmount.Text(), "利润")
	taxPercent, _ := parseMaterialQuoteNumber(state.taxRate.Text(), "税率")
	cost := 0.0
	for _, item := range state.costItems {
		if item.Enabled {
			cost += item.Amount
		}
	}
	base := cost + profit
	if state.quoteMode.CurrentIndex() == 1 {
		base = simple
	}
	calculated := base * (1 + taxPercent/100)
	server := "尚未保存"
	if state.baseline.ID != "" {
		server = fmt.Sprintf("服务端成本 %.3f / 税额 %.3f / 最终 %.3f", state.baseline.TotalCost, state.baseline.TaxAmount, state.baseline.FinalPrice)
	}
	state.summary.SetText(fmt.Sprintf("本地测算：成本 %.3f / 含税 %.3f；%s。", cost, calculated, server))
}

func (ui *mainUI) setMaterialQuoteEditorBusy(state *materialQuoteEditorUI, busy bool, message string) {
	state.busy = busy
	state.cancelButton.SetEnabled(!busy)
	if message != "" {
		state.info.SetText(message)
	}
	ui.onMaterialQuoteModeChanged()
	ui.updateMaterialQuoteEditorActions()
}

func (ui *mainUI) updateMaterialQuoteEditorActions() {
	state := ui.materialQuoteEditor
	if state == nil || state.save == nil {
		return
	}
	status := state.baseline.Status
	editable := !state.busy && !state.writeLocked && status != "priced" && status != "void" && state.delivery.ID != ""
	state.save.SetEnabled(editable && (state.dirty || state.baseline.ID == ""))
	state.submit.SetEnabled(!state.busy && !state.writeLocked && !state.dirty && state.baseline.ID != "" && status != "priced" && status != "void")
	state.price.SetEnabled(!state.busy && !state.writeLocked && state.baseline.ID != "" && status == "quoted")
	state.void.SetEnabled(!state.busy && !state.writeLocked && state.baseline.ID != "" && status != "priced" && status != "void")
	state.export.SetEnabled(!state.busy && state.baseline.ID != "")
	state.status.SetText("状态：" + displayMaterialValue(materialQuoteStatusLabel(status)))
	if state.baseline.ID == "" {
		state.status.SetText("状态：尚未创建")
	}
	ui.updateMaterialQuoteCostActions()
}

func (ui *mainUI) verifyCurrentMaterialQuote(state *materialQuoteEditorUI) (api.MaterialQuote, error) {
	if state.baseline.ID == "" {
		return api.MaterialQuote{}, nil
	}
	current, err := ui.session.Client.MaterialQuoteInfo(state.ctx, state.baseline.ID)
	if err != nil {
		return current, fmt.Errorf("提交前读取线上报价失败：%w", err)
	}
	if current.UpdatedAt != state.baseline.UpdatedAt || current.Status != state.baseline.Status {
		return current, fmt.Errorf("线上报价状态或更新时间已变化，请关闭编辑页并刷新后重试")
	}
	return current, nil
}

func (ui *mainUI) saveCurrentMaterialQuote() {
	state := ui.materialQuoteEditor
	if state == nil || state.busy || state.writeLocked {
		return
	}
	request, err := ui.materialQuoteEditorRequest(state)
	if err != nil {
		state.info.SetText(err.Error())
		return
	}
	if walk.MsgBox(ui.window, "确认保存报价", fmt.Sprintf("客户：%s\r\n物料：%s · %s\r\n报价方式：%s\r\n\r\n提交后立即写入线上数据，是否继续？", state.delivery.CustomerName, state.delivery.MaterialName, state.delivery.MaterialModel, state.quoteMode.Text()), walk.MsgBoxYesNo|walk.MsgBoxIconWarning) != walk.DlgCmdYes {
		return
	}
	ui.setMaterialQuoteEditorBusy(state, true, "正在刷新线上状态并保存；请求不会自动重试……")
	guardedGo(func() {
		writeApplied := false
		if state.baseline.ID != "" {
			_, err = ui.verifyCurrentMaterialQuote(state)
		}
		var saved api.MaterialQuote
		if err == nil {
			saved, err = ui.session.Client.SaveMaterialQuote(state.ctx, request)
			if err == nil {
				writeApplied = true
				saved, err = ui.session.Client.MaterialQuoteInfo(state.ctx, saved.ID)
				if err != nil {
					err = fmt.Errorf("报价已保存，但在线回读失败：%w", err)
				}
			}
		}
		if state.ctx.Err() != nil || state.closed.Load() {
			return
		}
		ui.window.Synchronize(func() {
			if state != ui.materialQuoteEditor || state.closed.Load() {
				return
			}
			if err != nil {
				if writeApplied {
					state.writeLocked = true
					state.setDirty(false)
					ui.setMaterialQuoteEditorBusy(state, false, requestFailureText(err)+"。本页已禁止重复提交，请关闭并刷新后核对。")
					return
				}
				ui.setMaterialQuoteEditorBusy(state, false, requestFailureText(err))
				return
			}
			ui.applyMaterialQuoteToEditor(state, saved)
			ui.setMaterialQuoteEditorBusy(state, false, "报价已保存并完成线上回读；服务端金额已同步。")
			if ui.materialQuote != nil && ui.materialQuote.table != nil {
				ui.loadMaterialQuotes()
			}
		})
	})
}

func (ui *mainUI) mutateCurrentMaterialQuote(action string) {
	state := ui.materialQuoteEditor
	if state == nil || state.busy || state.writeLocked || state.baseline.ID == "" {
		return
	}
	title, message := "确认提交报价", "提交后状态变为“已报价”，是否继续？"
	if action == "price" {
		title = "确认最终定价"
		message = "最终定价只提交当前最终报价单价和备注，并写入现有客户物料价格记录；未保存的成本项、税率等修改不会随本操作提交。是否继续？"
	}
	if action == "void" {
		title = "确认作废报价"
		message = "作废后报价不能修改或定价，是否继续？"
	}
	if walk.MsgBox(ui.window, title, message, walk.MsgBoxYesNo|walk.MsgBoxIconWarning) != walk.DlgCmdYes {
		return
	}
	ui.setMaterialQuoteEditorBusy(state, true, "正在核对线上状态并执行操作；请求不会自动重试……")
	guardedGo(func() {
		_, err := ui.verifyCurrentMaterialQuote(state)
		writeApplied := false
		var result api.MaterialQuote
		if err == nil {
			switch action {
			case "submit":
				result, err = ui.session.Client.SubmitMaterialQuote(state.ctx, state.baseline.ID)
			case "price":
				finalPrice, parseErr := parseMaterialQuoteNumber(state.finalPrice.Text(), "最终定价")
				if parseErr != nil || finalPrice <= 0 {
					err = fmt.Errorf("最终定价必须大于 0")
				} else {
					result, err = ui.session.Client.PriceMaterialQuote(state.ctx, api.MaterialQuotePriceRequest{ID: state.baseline.ID, FinalPrice: finalPrice, EffectiveAt: time.Now().Unix(), Remark: strings.TrimSpace(state.remark.Text())})
				}
			case "void":
				err = ui.session.Client.VoidMaterialQuote(state.ctx, state.baseline.ID)
			}
			if err == nil {
				writeApplied = true
				result, err = ui.session.Client.MaterialQuoteInfo(state.ctx, state.baseline.ID)
				if err != nil {
					err = fmt.Errorf("操作已提交，但在线回读失败：%w", err)
				}
			}
		}
		if state.ctx.Err() != nil || state.closed.Load() {
			return
		}
		ui.window.Synchronize(func() {
			if state != ui.materialQuoteEditor || state.closed.Load() {
				return
			}
			if err != nil {
				if writeApplied {
					state.writeLocked = true
					state.setDirty(false)
					ui.setMaterialQuoteEditorBusy(state, false, requestFailureText(err)+"。本页已禁止重复操作，请关闭并刷新后核对。")
					return
				}
				ui.setMaterialQuoteEditorBusy(state, false, requestFailureText(err))
				return
			}
			ui.applyMaterialQuoteToEditor(state, result)
			ui.setMaterialQuoteEditorBusy(state, false, "操作成功并完成线上回读。")
			if ui.materialQuote != nil && ui.materialQuote.table != nil {
				ui.loadMaterialQuotes()
			}
		})
	})
}

func (ui *mainUI) submitCurrentMaterialQuote() { ui.mutateCurrentMaterialQuote("submit") }
func (ui *mainUI) priceCurrentMaterialQuote()  { ui.mutateCurrentMaterialQuote("price") }
func (ui *mainUI) voidCurrentMaterialQuote()   { ui.mutateCurrentMaterialQuote("void") }

func (ui *mainUI) exportCurrentMaterialQuote() {
	state := ui.materialQuoteEditor
	if state == nil || state.busy || state.baseline.ID == "" {
		return
	}
	dialog := new(walk.FileDialog)
	dialog.Title = "导出物料报价"
	dialog.Filter = "CSV 文件 (*.csv)|*.csv"
	dialog.FilePath = state.baseline.QuoteNo + ".csv"
	ok, err := dialog.ShowSave(ui.window)
	if err != nil || !ok {
		return
	}
	target := strings.TrimSpace(dialog.FilePath)
	if filepath.Ext(target) == "" {
		target += ".csv"
	}
	ui.setMaterialQuoteEditorBusy(state, true, "正在由服务端生成报价 CSV……")
	guardedGo(func() {
		data, _, requestErr := ui.session.Client.ExportMaterialQuote(state.ctx, state.baseline.ID)
		if requestErr == nil {
			requestErr = os.WriteFile(target, data, 0600)
		}
		if state.ctx.Err() != nil || state.closed.Load() {
			return
		}
		ui.window.Synchronize(func() {
			if state != ui.materialQuoteEditor || state.closed.Load() {
				return
			}
			if requestErr != nil {
				ui.setMaterialQuoteEditorBusy(state, false, "导出失败："+requestErr.Error()+"；可修正后重试。")
				return
			}
			ui.setMaterialQuoteEditorBusy(state, false, "报价 CSV 已保存："+target)
		})
	})
}
