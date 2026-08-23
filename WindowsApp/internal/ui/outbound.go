package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"

	"zhengshi-wms-windowsapp/internal/api"
	"zhengshi-wms-windowsapp/internal/config"
)

type outboundStage struct {
	Label   string
	Status  string
	IsPack  int
	IsWeigh int
}

var outboundStages = []outboundStage{
	{Label: "全部队列", IsPack: -1, IsWeigh: -1},
	{Label: "预发货", Status: "预发货", IsPack: -1, IsWeigh: -1},
	{Label: "待拣货", Status: "待拣货", IsPack: -1, IsWeigh: -1},
	{Label: "已拣货", Status: "已拣货", IsPack: -1, IsWeigh: -1},
	{Label: "待打包", Status: "待打包", IsPack: 0, IsWeigh: -1},
	{Label: "已打包", Status: "已打包", IsPack: 1, IsWeigh: -1},
	{Label: "待称重", Status: "待称重", IsPack: -1, IsWeigh: 0},
	{Label: "已称重", Status: "已称重", IsPack: -1, IsWeigh: 1},
	{Label: "待出库", Status: "待出库", IsPack: -1, IsWeigh: -1},
	{Label: "已出库", Status: "已出库", IsPack: -1, IsWeigh: -1},
	{Label: "已签收", Status: "已签收", IsPack: -1, IsWeigh: -1},
}

var outboundPageSizeLabels = []string{"10 条/页", "20 条/页", "50 条/页"}

type outboundRow struct {
	Code      string
	Type      string
	Status    string
	Progress  string
	Business  string
	Carrier   string
	Amount    string
	Departure string
	Receipt   string
	Remark    string
}

type outboundMaterialRow struct {
	Index         string
	Name          string
	Model         string
	Specification string
	Quantity      string
	Weight        string
	Price         string
}

type outboundUI struct {
	splitter    *walk.Splitter
	search      *walk.LineEdit
	model       *walk.LineEdit
	chooseModel *walk.PushButton
	stage       *walk.ComboBox
	orderType   *walk.ComboBox
	supplier    *walk.ComboBox
	customer    *walk.ComboBox
	startDate   *walk.LineEdit
	endDate     *walk.LineEdit
	table       *walk.TableView
	materials   *walk.TableView
	info        *walk.Label
	actionHint  *walk.Label
	query       *walk.PushButton
	reset       *walk.PushButton
	prev        *walk.PushButton
	next        *walk.PushButton
	pageSize    *walk.ComboBox
	add         *walk.PushButton
	fast        *walk.PushButton
	delete      *walk.PushButton
	export      *walk.PushButton
	exportStop  *walk.PushButton
	detail      *walk.PushButton
	attachments *walk.PushButton
	revise      *walk.PushButton
	confirm     *walk.PushButton
	pick        *walk.PushButton
	pack        *walk.PushButton
	weigh       *walk.PushButton
	departure   *walk.PushButton
	receipt     *walk.PushButton

	page               int
	total              int64
	rows               []api.OutboundOrder
	selectedMaterials  []api.OutboundMaterial
	supplierOptions    []selectOption
	customerOptions    []selectOption
	generation         int
	materialGeneration int
	exportGeneration   int
	cancel             context.CancelFunc
	exportCancel       context.CancelFunc
	materialCancel     context.CancelFunc
	operationBusy      bool
	exportBusy         bool
}

func newOutboundUI() *outboundUI {
	return &outboundUI{page: 1}
}

func outboundStageLabels() []string {
	labels := make([]string, len(outboundStages))
	for index, stage := range outboundStages {
		labels[index] = stage.Label
	}
	return labels
}

func (ui *mainUI) outboundPageWidget() TabPage {
	if ui.outbound == nil {
		ui.outbound = newOutboundUI()
	}
	state := ui.outbound
	types := []string{
		"全部类型", "销售出库", "样品出库", "退货出库", "报废出库",
		"赠品出库", "生产用料出库", "损耗出库",
	}
	return TabPage{
		AssignTo: &ui.outboundTab,
		Title:    closableTabTitle("出库执行"),
		Layout:   VBox{Margins: Margins{Left: 14, Top: 14, Right: 14, Bottom: 12}, Spacing: 8},
		Children: []Widget{
			Label{Text: "出库执行", Font: Font{Family: "Microsoft YaHei UI", PointSize: 15, Bold: true}},
			Label{
				Text:      "状态队列直接映射现有 API；选择订单后，在下方核对物料并执行当前状态允许的操作。",
				TextColor: secondaryTextColor(), Accessibility: Accessibility{Name: "出库单操作提示"},
			},
			GroupBox{
				Title:  "筛选条件",
				Layout: Grid{Columns: 8, Margins: Margins{Left: 14, Top: 10, Right: 14, Bottom: 12}, Spacing: 8},
				Children: []Widget{
					Label{Text: "出库单号"},
					LineEdit{
						AssignTo: &state.search, MinSize: Size{Width: 170, Height: 28},
						CueBanner: "可扫码或输入单号",
					},
					Label{Text: "物料型号"},
					Composite{
						Layout: HBox{Spacing: 4},
						Children: []Widget{
							LineEdit{AssignTo: &state.model, ReadOnly: true, MinSize: Size{Width: 112, Height: 28}, CueBanner: "全部型号", Accessibility: Accessibility{Name: "出库物料精确型号"}},
							PushButton{AssignTo: &state.chooseModel, Text: "选择", MinSize: Size{Width: 54, Height: 28}, ToolTipText: "从物料列表搜索并选择精确型号", OnClicked: ui.selectOutboundFilterModel},
						},
					},
					Label{Text: "状态队列"},
					ComboBox{
						AssignTo: &state.stage, Model: outboundStageLabels(), CurrentIndex: ui.initialWorkspaceFilterIndex("outbound_stage", len(outboundStages)),
						MinSize: Size{Width: 130, Height: 28},
					},
					Label{Text: "出库类型"},
					ComboBox{
						AssignTo: &state.orderType, Model: types, CurrentIndex: ui.initialWorkspaceFilterIndex("outbound_type", len(types)),
						MinSize: Size{Width: 140, Height: 28},
					},
					Label{Text: "供应商"},
					ComboBox{
						AssignTo: &state.supplier, Model: []string{"全部供应商（正在加载）"}, CurrentIndex: 0,
						MinSize: Size{Width: 175, Height: 28},
					},
					Label{Text: "客户"},
					ComboBox{
						AssignTo: &state.customer, Model: []string{"全部客户（正在加载）"}, CurrentIndex: 0,
						MinSize: Size{Width: 175, Height: 28},
					},
					Label{Text: "签收起始"},
					LineEdit{
						AssignTo: &state.startDate, MinSize: Size{Width: 130, Height: 28},
						CueBanner: "YYYY-MM-DD",
					},
					Label{Text: "签收截止"},
					LineEdit{
						AssignTo: &state.endDate, MinSize: Size{Width: 130, Height: 28},
						CueBanner: "YYYY-MM-DD",
					},
					Composite{
						ColumnSpan: 8,
						Layout:     HBox{Spacing: 8},
						Children: []Widget{
							HSpacer{},
							PushButton{
								AssignTo: &state.reset, Text: "重置", MinSize: Size{Width: 88, Height: 30},
								OnClicked: ui.resetOutboundFilters,
							},
							PushButton{
								AssignTo: &state.query, Text: "查询", MinSize: Size{Width: 96, Height: 30},
								OnClicked: func() {
									state.page = 1
									ui.loadOutbound()
								},
							},
						},
					},
				},
			},
			VSplitter{
				AssignTo:      &state.splitter,
				HandleWidth:   4,
				StretchFactor: 1,
				Children: []Widget{
					TableView{
						AssignTo:              &state.table,
						AlternatingRowBG:      true,
						ColumnsOrderable:      true,
						StretchFactor:         3,
						OnCurrentIndexChanged: ui.outboundSelectionChanged,
						OnItemActivated:       ui.showSelectedOutboundDetail,
						Columns: []TableViewColumn{
							{Title: "出库单号", DataMember: "Code", Width: 140},
							{Title: "类型", DataMember: "Type", Width: 95},
							{Title: "状态", DataMember: "Status", Width: 85},
							{Title: "执行进度", DataMember: "Progress", Width: 110},
							{Title: "供应商/客户", DataMember: "Business", Width: 150},
							{Title: "承运商", DataMember: "Carrier", Width: 110},
							{Title: "金额", DataMember: "Amount", Width: 85},
							{Title: "出库时间", DataMember: "Departure", Width: 125},
							{Title: "签收时间", DataMember: "Receipt", Width: 125},
							{Title: "备注", DataMember: "Remark", Width: 160},
						},
					},
					Composite{
						StretchFactor: 2,
						Layout:        VBox{Margins: Margins{Left: 0, Top: 8, Right: 0, Bottom: 0}, Spacing: 6},
						Children: []Widget{
							Label{Text: "订单物料", Font: Font{Bold: true}},
							TableView{
								AssignTo:         &state.materials,
								Model:            []outboundMaterialRow{},
								AlternatingRowBG: true,
								ColumnsOrderable: true,
								StretchFactor:    1,
								Columns: []TableViewColumn{
									{Title: "序号", DataMember: "Index", Width: 55},
									{Title: "物料", DataMember: "Name", Width: 190},
									{Title: "型号", DataMember: "Model", Width: 105},
									{Title: "规格", DataMember: "Specification", Width: 120},
									{Title: "数量", DataMember: "Quantity", Width: 90},
									{Title: "重量", DataMember: "Weight", Width: 85},
									{Title: "单价", DataMember: "Price", Width: 85},
								},
							},
						},
					},
				},
			},
			Label{
				AssignTo: &state.actionHint, Text: "请选择一张出库单。",
				TextColor: secondaryTextColor(),
			},
			Composite{
				Layout: VBox{Spacing: 6},
				Children: []Widget{
					Composite{Layout: HBox{Spacing: 6}, Children: []Widget{
						Label{Text: "当前单据操作", Font: Font{Bold: true}},
						PushButton{
							AssignTo: &state.add, Text: "新建出库单",
							Enabled: hasButton(ui.session.Perms.Buttons, "outbound:order:add"),
							MinSize: Size{Width: 100, Height: 30}, Accessibility: Accessibility{Name: "新建预发货出库单"},
							OnClicked: ui.newOutboundOrder,
						},
						PushButton{
							AssignTo: &state.fast, Text: "极速出库",
							Enabled: hasButton(ui.session.Perms.Buttons, "outbound:order:add"),
							MinSize: Size{Width: 92, Height: 30}, Accessibility: Accessibility{Name: "新建并直接完成签收的极速出库单"},
							ToolTipText: "直接完成库存补足、扣减、签收和客户应收记账；提交前会再次确认",
							OnClicked:   ui.openFastOutbound,
						},
						PushButton{
							AssignTo: &state.delete, Text: "删除预发货", Enabled: false,
							Visible: hasButton(ui.session.Perms.Buttons, "outbound:order:delete"),
							MinSize: Size{Width: 100, Height: 30}, Accessibility: Accessibility{Name: "删除选中的预发货出库单"},
							OnClicked: ui.deleteSelectedOutbound,
						},
						PushButton{
							AssignTo: &state.export, Text: "导出当前筛选", MinSize: Size{Width: 108, Height: 30},
							Accessibility: Accessibility{Name: "导出当前出库筛选的全部结果到 Excel"}, OnClicked: ui.exportOutboundQuery,
						},
						PushButton{
							AssignTo: &state.exportStop, Text: "取消导出", Enabled: false, MinSize: Size{Width: 88, Height: 30},
							Accessibility: Accessibility{Name: "取消当前出库导出并清理未完成文件"}, OnClicked: ui.cancelOutboundExport,
						},
						PushButton{
							AssignTo: &state.detail, Text: "查看详情", Enabled: false,
							MinSize: Size{Width: 88, Height: 30}, Accessibility: Accessibility{Name: "查看选中出库单详情"},
							OnClicked: ui.showSelectedOutboundDetail,
						},
						PushButton{
							AssignTo: &state.attachments, Text: "查看附件", Enabled: false,
							MinSize:       Size{Width: 88, Height: 30},
							Accessibility: Accessibility{Name: "查看选中出库单的图片附件"},
							OnClicked:     ui.showSelectedOutboundAttachments,
						},
						PushButton{
							AssignTo: &state.revise, Text: "核价/调价", Enabled: false,
							Visible: hasButton(ui.session.Perms.Buttons, "outbound:order:revise"),
							MinSize: Size{Width: 88, Height: 30}, Accessibility: Accessibility{Name: "核价或调整出库单物料价格"},
							OnClicked: ui.reviseSelectedOutbound,
						},
						HSpacer{},
					}},
					Composite{Layout: HBox{Spacing: 6}, Children: []Widget{
						Label{Text: "执行流程", Font: Font{Bold: true}},
						PushButton{AssignTo: &state.confirm, Text: "确认并分配库存", Accessibility: Accessibility{Name: "确认出库单并分配库存"}, OnClicked: ui.confirmSelectedOutbound},
						PushButton{AssignTo: &state.pick, Text: "确认拣货", Accessibility: Accessibility{Name: "确认出库单拣货完成"}, OnClicked: ui.pickSelectedOutbound},
						PushButton{AssignTo: &state.pack, Text: "确认打包", Accessibility: Accessibility{Name: "确认出库单打包完成"}, OnClicked: ui.packSelectedOutbound},
						PushButton{AssignTo: &state.weigh, Text: "确认称重", Accessibility: Accessibility{Name: "确认出库单称重完成"}, OnClicked: ui.weighSelectedOutbound},
						PushButton{AssignTo: &state.departure, Text: "确认出库", Accessibility: Accessibility{Name: "确认出库单已经出库"}, OnClicked: ui.departSelectedOutbound},
						PushButton{AssignTo: &state.receipt, Text: "确认签收", Accessibility: Accessibility{Name: "确认出库单已经签收"}, OnClicked: ui.receiptSelectedOutbound},
						HSpacer{},
					}},
					Composite{Layout: HBox{Spacing: 6}, Children: []Widget{
						Label{AssignTo: &state.info, Text: "尚未加载"},
						HSpacer{},
						Label{Text: "每页"},
						ComboBox{
							AssignTo: &state.pageSize, Model: outboundPageSizeLabels, CurrentIndex: ui.initialPageSizeIndex("outbound", len(outboundPageSizeLabels)),
							MinSize: Size{Width: 92},
							OnCurrentIndexChanged: func() {
								if ui.window != nil && state.pageSize != nil && state.pageSize.CurrentIndex() >= 0 {
									state.page = 1
									ui.loadOutbound()
								}
							},
						},
						PushButton{
							AssignTo: &state.prev, Text: "上一页",
							OnClicked: func() {
								if state.page > 1 {
									state.page--
									ui.loadOutbound()
								}
							},
						},
						PushButton{
							AssignTo: &state.next, Text: "下一页",
							OnClicked: func() {
								state.page++
								ui.loadOutbound()
							},
						},
						PushButton{Text: "跳转页", Accessibility: Accessibility{Name: "跳转到指定出库结果页"}, OnClicked: func() {
							if page, ok := promptPageNumber(ui.window, state.page, state.total, selectedPageSize(state.pageSize)); ok {
								state.page = page
								ui.loadOutbound()
							}
						}},
					}},
				},
			},
		},
	}
}

func (ui *mainUI) initializeOutboundPage() {
	state := ui.outbound
	if ui.outboundTab == nil || state == nil || state.search == nil {
		return
	}
	state.generation++
	state.search.KeyDown().Attach(func(key walk.Key) {
		if key == walk.KeyReturn {
			state.page = 1
			ui.loadOutbound()
		}
	})
	ui.setOutboundActionButtons(nil)
	ui.loadOutboundOptions()
	ui.loadOutbound()
}

func (ui *mainUI) releaseOutboundPage() {
	state := ui.outbound
	if state == nil {
		return
	}
	state.generation++
	state.materialGeneration++
	if state.cancel != nil {
		state.cancel()
	}
	if state.materialCancel != nil {
		state.materialCancel()
	}
	if state.exportCancel != nil {
		state.exportCancel()
	}
	ui.outboundTab = nil
	ui.outbound = newOutboundUI()
}

func (ui *mainUI) loadOutboundOptions() {
	state := ui.outbound
	if state == nil {
		return
	}
	generation := state.generation
	guardedGo(func() {
		suppliers, supplierErr := ui.session.Client.Suppliers(context.Background())
		customers, customerErr := ui.session.Client.Customers(context.Background())
		ui.window.Synchronize(func() {
			if state != ui.outbound || generation != state.generation || state.supplier == nil || state.customer == nil {
				return
			}
			state.supplierOptions = make([]selectOption, 0, len(suppliers))
			for _, supplier := range suppliers {
				state.supplierOptions = append(state.supplierOptions, selectOption{
					ID: supplier.ID, Label: businessOptionLabel(supplier.Name, supplier.Code),
				})
			}
			state.customerOptions = make([]selectOption, 0, len(customers))
			for _, customer := range customers {
				state.customerOptions = append(state.customerOptions, selectOption{
					ID: customer.ID, Label: businessOptionLabel(customer.Name, customer.Code),
				})
			}
			supplierLabel := "全部供应商"
			if supplierErr != nil {
				supplierLabel = "供应商加载失败"
			}
			customerLabel := "全部客户"
			if customerErr != nil {
				customerLabel = "客户加载失败"
			}
			_ = state.supplier.SetModel(optionLabels(supplierLabel, state.supplierOptions))
			_ = state.customer.SetModel(optionLabels(customerLabel, state.customerOptions))
			state.supplier.SetCurrentIndex(0)
			state.customer.SetCurrentIndex(0)
			if supplierErr != nil || customerErr != nil {
				state.actionHint.SetText("部分往来单位筛选项加载失败，仍可使用其他条件查询。")
			}
		})
	})
}

func businessOptionLabel(name, code string) string {
	name = strings.TrimSpace(name)
	code = strings.TrimSpace(code)
	if code == "" {
		return name
	}
	return name + "（" + code + "）"
}

func (ui *mainUI) loadOutbound() {
	state := ui.outbound
	if state == nil || state.table == nil {
		return
	}
	startTime, err := parseOutboundFilterDate(state.startDate.Text(), false)
	if err != nil {
		state.info.SetText("起始日期格式错误：" + err.Error())
		state.startDate.SetFocus()
		return
	}
	endTime, err := parseOutboundFilterDate(state.endDate.Text(), true)
	if err != nil {
		state.info.SetText("截止日期格式错误：" + err.Error())
		state.endDate.SetFocus()
		return
	}
	if startTime > 0 && endTime > 0 && startTime > endTime {
		state.info.SetText("签收起始日期不能晚于截止日期。")
		state.startDate.SetFocus()
		return
	}
	if state.cancel != nil {
		state.cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	state.cancel = cancel
	page := state.page
	size := selectedPageSize(state.pageSize)
	generation := state.generation
	stage := outboundStages[0]
	if index := state.stage.CurrentIndex(); index >= 0 && index < len(outboundStages) {
		stage = outboundStages[index]
	}
	filters := api.OutboundFilters{
		Code: state.search.Text(), Model: state.model.Text(), Status: stage.Status, IsPack: stage.IsPack, IsWeigh: stage.IsWeigh,
		SupplierID: selectedOptionID(state.supplier, state.supplierOptions),
		CustomerID: selectedOptionID(state.customer, state.customerOptions),
		StartTime:  startTime, EndTime: endTime,
	}
	if state.orderType.CurrentIndex() > 0 {
		filters.Type = state.orderType.Text()
	}
	state.info.SetText("正在加载线上出库单……")
	state.query.SetEnabled(false)
	state.reset.SetEnabled(false)
	state.prev.SetEnabled(false)
	state.next.SetEnabled(false)
	ui.setOutboundActionButtons(nil)
	guardedGo(func() {
		result, requestErr := ui.session.Client.OutboundOrders(ctx, page, size, filters)
		if ctx.Err() != nil {
			return
		}
		ui.window.Synchronize(func() {
			if state != ui.outbound || generation != state.generation || state.table == nil {
				return
			}
			state.query.SetEnabled(true)
			state.reset.SetEnabled(true)
			if requestErr != nil {
				state.info.SetText(requestFailureText(requestErr))
				return
			}
			rows := make([]outboundRow, 0, len(result.List))
			for _, order := range result.List {
				business := strings.TrimSpace(order.SupplierName)
				if business == "" {
					business = strings.TrimSpace(order.CustomerName)
				}
				rows = append(rows, outboundRow{
					Code: order.Code, Type: order.Type, Status: order.Status,
					Progress: outboundProgress(order), Business: business, Carrier: order.CarrierName,
					Amount:    fmt.Sprintf("%.2f", order.TotalAmount),
					Departure: formatUnixMinute(order.DepartureTime), Receipt: formatUnixMinute(order.ReceiptTime),
					Remark: order.Remark,
				})
			}
			if modelErr := state.table.SetModel(rows); modelErr != nil {
				state.info.SetText("表格数据错误：" + modelErr.Error())
				return
			}
			state.rows = result.List
			state.total = result.Total
			state.selectedMaterials = nil
			_ = state.materials.SetModel([]outboundMaterialRow{})
			state.info.SetText(fmt.Sprintf("%s | 第 %d 页 | 本页 %d 条 | 共 %d 条",
				stage.Label, page, len(rows), result.Total))
			state.prev.SetEnabled(page > 1)
			state.next.SetEnabled(int64(page*size) < result.Total)
			state.actionHint.SetText("请选择一张出库单查看物料和可执行操作。")
		})
	})
}

func parseOutboundFilterDate(text string, endOfDay bool) (int64, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0, nil
	}
	value, err := time.ParseInLocation("2006-01-02", text, time.Local)
	if err != nil {
		return 0, fmt.Errorf("请使用 YYYY-MM-DD")
	}
	if endOfDay {
		value = value.Add(24*time.Hour - time.Second)
	}
	return value.Unix(), nil
}

func outboundProgress(order api.OutboundOrder) string {
	pack := "未打包"
	if order.IsPack == 1 {
		pack = "已打包"
	}
	weigh := "未称重"
	if order.IsWeigh == 1 {
		weigh = "已称重"
	}
	return pack + " / " + weigh
}

func formatUnixMinute(value int64) string {
	if value <= 0 {
		return ""
	}
	return time.Unix(value, 0).Format("2006-01-02 15:04")
}

func (ui *mainUI) resetOutboundFilters() {
	state := ui.outbound
	if state == nil {
		return
	}
	state.search.SetText("")
	state.model.SetText("")
	state.stage.SetCurrentIndex(0)
	state.orderType.SetCurrentIndex(0)
	state.supplier.SetCurrentIndex(0)
	state.customer.SetCurrentIndex(0)
	state.startDate.SetText("")
	state.endDate.SetText("")
	state.page = 1
	ui.loadOutbound()
}

func (ui *mainUI) selectOutboundFilterModel() {
	state := ui.outbound
	if state == nil || state.operationBusy {
		return
	}
	material, ok := selectOutboundMaterial(ui.window, ui.session.Client)
	if !ok {
		return
	}
	state.model.SetText(strings.TrimSpace(material.Model))
	state.page = 1
	ui.loadOutbound()
}

func (ui *mainUI) selectedOutbound() (api.OutboundOrder, bool) {
	state := ui.outbound
	if state == nil || state.table == nil {
		return api.OutboundOrder{}, false
	}
	index := state.table.CurrentIndex()
	if index < 0 || index >= len(state.rows) {
		walk.MsgBox(ui.window, "请选择出库单", "请先在出库队列中选择一张出库单。", walk.MsgBoxIconInformation)
		return api.OutboundOrder{}, false
	}
	return state.rows[index], true
}

func (ui *mainUI) outboundSelectionChanged() {
	state := ui.outbound
	if state == nil || state.table == nil {
		return
	}
	index := state.table.CurrentIndex()
	if index < 0 || index >= len(state.rows) {
		state.selectedMaterials = nil
		_ = state.materials.SetModel([]outboundMaterialRow{})
		ui.setOutboundActionButtons(nil)
		return
	}
	order := state.rows[index]
	ui.setOutboundActionButtons(&order)
	state.actionHint.SetText(outboundActionHint(order))
	ui.loadSelectedOutboundMaterials(order)
}

func (ui *mainUI) loadSelectedOutboundMaterials(order api.OutboundOrder) {
	state := ui.outbound
	if state.materialCancel != nil {
		state.materialCancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	state.materialCancel = cancel
	state.materialGeneration++
	generation := state.materialGeneration
	state.actionHint.SetText("正在加载出库单物料……")
	guardedGo(func() {
		materials, err := ui.session.Client.OutboundMaterials(ctx, order.Code)
		if ctx.Err() != nil {
			return
		}
		ui.window.Synchronize(func() {
			if state != ui.outbound || generation != state.materialGeneration || state.materials == nil {
				return
			}
			if err != nil {
				state.actionHint.SetText("物料加载失败：" + err.Error())
				return
			}
			rows := make([]outboundMaterialRow, 0, len(materials))
			for _, material := range materials {
				rows = append(rows, outboundMaterialRow{
					Index: fmt.Sprint(material.Index), Name: material.Name, Model: material.Model,
					Specification: material.Specification,
					Quantity:      fmt.Sprintf("%g %s", material.Quantity, material.Unit),
					Weight:        fmt.Sprintf("%g", material.Weight),
					Price:         fmt.Sprintf("%.3f", material.Price),
				})
			}
			if modelErr := state.materials.SetModel(rows); modelErr != nil {
				state.actionHint.SetText("物料展示失败：" + modelErr.Error())
				return
			}
			state.selectedMaterials = materials
			state.actionHint.SetText(outboundActionHint(order))
		})
	})
}

func outboundActionHint(order api.OutboundOrder) string {
	action := "当前状态没有可执行的仓库动作"
	switch {
	case order.Status == "预发货":
		action = "可确认订单并按库存批次分配出货数量"
	case order.Status == "待拣货":
		action = "可确认拣货"
	case order.Status == "已拣货":
		action = "可打包、称重或直接确认出库"
	case order.Status == "已称重" && order.IsPack == 0:
		action = "可补打包或确认出库"
	case order.Status == "已打包" && order.IsWeigh == 0:
		action = "可补称重或确认出库"
	case order.Status == "已打包", order.Status == "已称重":
		action = "可确认出库"
	case order.Status == "已出库":
		action = "可上传签收附件并确认签收"
	}
	return fmt.Sprintf("当前：%s，%s。服务端状态和权限仍是最终依据。", order.Status, action)
}

func canConfirmOutbound(order api.OutboundOrder) bool {
	return order.Status == "预发货"
}

func canPickOutbound(order api.OutboundOrder) bool {
	return order.Status == "待拣货"
}

func canPackOutbound(order api.OutboundOrder) bool {
	return order.Status == "已拣货" || (order.Status == "已称重" && order.IsPack == 0)
}

func canWeighOutbound(order api.OutboundOrder) bool {
	return order.Status == "已拣货" || (order.Status == "已打包" && order.IsWeigh == 0)
}

func canDepartOutbound(order api.OutboundOrder) bool {
	return order.Status == "已拣货" || order.Status == "已打包" || order.Status == "已称重"
}

func canReceiptOutbound(order api.OutboundOrder) bool {
	return order.Status == "已出库"
}

func (ui *mainUI) setOutboundActionButtons(order *api.OutboundOrder) {
	state := ui.outbound
	if state == nil || state.confirm == nil {
		return
	}
	enabled := order != nil && !state.operationBusy
	var value api.OutboundOrder
	if order != nil {
		value = *order
	}
	if state.attachments != nil {
		state.attachments.SetEnabled(enabled && len(value.Annex) > 0)
		if len(value.Annex) > 0 {
			state.attachments.SetText(fmt.Sprintf("查看附件 (%d)", len(value.Annex)))
		} else {
			state.attachments.SetText("查看附件")
		}
	}
	if state.detail != nil {
		state.detail.SetEnabled(enabled)
	}
	if state.add != nil {
		state.add.SetEnabled(!state.operationBusy && hasButton(ui.session.Perms.Buttons, "outbound:order:add"))
	}
	if state.fast != nil {
		state.fast.SetEnabled(!state.operationBusy && hasButton(ui.session.Perms.Buttons, "outbound:order:add"))
	}
	if state.export != nil {
		state.export.SetEnabled(!state.operationBusy && !state.exportBusy)
	}
	if state.exportStop != nil {
		state.exportStop.SetEnabled(state.exportBusy)
	}
	state.confirm.SetEnabled(enabled && hasButton(ui.session.Perms.Buttons, "outbound:order:confirm") && canConfirmOutbound(value))
	state.pick.SetEnabled(enabled && hasButton(ui.session.Perms.Buttons, "outbound:order:pick") && canPickOutbound(value))
	state.pack.SetEnabled(enabled && hasButton(ui.session.Perms.Buttons, "outbound:order:pack") && canPackOutbound(value))
	state.weigh.SetEnabled(enabled && hasButton(ui.session.Perms.Buttons, "outbound:order:weigh") && canWeighOutbound(value))
	state.departure.SetEnabled(enabled && hasButton(ui.session.Perms.Buttons, "outbound:order:departure") && canDepartOutbound(value))
	state.receipt.SetEnabled(enabled && hasButton(ui.session.Perms.Buttons, "outbound:order:receipt") && canReceiptOutbound(value))
	if state.revise != nil {
		state.revise.SetEnabled(enabled && hasButton(ui.session.Perms.Buttons, "outbound:order:revise") && strings.TrimSpace(value.CustomerID) != "")
	}
	if state.delete != nil {
		state.delete.SetEnabled(enabled && hasButton(ui.session.Perms.Buttons, "outbound:order:delete") && canDeleteOutbound(value))
	}
}

func (ui *mainUI) showSelectedOutboundAttachments() {
	order, ok := ui.selectedOutbound()
	if !ok {
		return
	}
	ShowOrderAttachments(
		ui.window, ui.session.Client, config.ImageBaseURL(), "出库单 "+order.Code, order.Annex,
	)
}

func (ui *mainUI) setOutboundOperationBusy(busy bool, message string) {
	state := ui.outbound
	if state == nil {
		return
	}
	state.operationBusy = busy
	if message != "" {
		state.actionHint.SetText(message)
	}
	order, ok := ui.selectedOutboundWithoutMessage()
	if !ok {
		ui.setOutboundActionButtons(nil)
		return
	}
	ui.setOutboundActionButtons(&order)
}

func (ui *mainUI) selectedOutboundWithoutMessage() (api.OutboundOrder, bool) {
	state := ui.outbound
	if state == nil || state.table == nil {
		return api.OutboundOrder{}, false
	}
	index := state.table.CurrentIndex()
	if index < 0 || index >= len(state.rows) {
		return api.OutboundOrder{}, false
	}
	return state.rows[index], true
}
