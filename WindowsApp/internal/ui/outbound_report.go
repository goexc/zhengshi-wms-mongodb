package ui

import (
	"context"
	"fmt"
	"math/big"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"

	"zhengshi-wms-windowsapp/internal/api"
)

type outboundReportRow struct {
	Model         string
	Name          string
	Specification string
	DepartureDate string
	ReceiptDate   string
	OrderCode     string
	Price         string
	Quantity      string
	Unit          string
	Amount        string
}

type outboundReportUI struct {
	customer       *walk.ComboBox
	startDate      *walk.LineEdit
	endDate        *walk.LineEdit
	quickRange     *walk.ComboBox
	table          *walk.TableView
	summary        *walk.Label
	info           *walk.Label
	query          *walk.PushButton
	reset          *walk.PushButton
	exportOrder    *walk.PushButton
	exportMaterial *walk.PushButton

	customerOptions  []selectOption
	rows             []api.OutboundSummaryRecord
	searchedCustomer string
	searchedStart    int64
	searchedEnd      int64
	generation       int
	cancel           context.CancelFunc
}

func newOutboundReportUI() *outboundReportUI {
	return &outboundReportUI{}
}

func (ui *mainUI) outboundReportPageWidget() TabPage {
	state := ui.outboundReport
	return TabPage{
		AssignTo: &ui.outboundReportTab,
		Title:    closableTabTitle("出库报表"),
		Layout:   VBox{Margins: Margins{Left: 14, Top: 14, Right: 14, Bottom: 12}, Spacing: 10},
		Children: []Widget{
			Label{Text: "出库报表", Font: Font{Family: "Microsoft YaHei UI", PointSize: 15, Bold: true}},
			Label{
				Text:      "按客户和签收日期查询服务端汇总记录；统计、分组与 Excel 导出仅改变呈现方式。",
				TextColor: walk.RGB(85, 85, 85),
			},
			GroupBox{
				Title:  "筛选条件",
				Layout: Grid{Columns: 8, Margins: Margins{Left: 14, Top: 10, Right: 14, Bottom: 12}, Spacing: 8},
				Children: []Widget{
					Label{Text: "客户"},
					ComboBox{
						AssignTo: &state.customer, Model: []string{"正在加载客户……"}, CurrentIndex: 0,
						MinSize: Size{Width: 190, Height: 28},
					},
					Label{Text: "起始日期"},
					LineEdit{
						AssignTo: &state.startDate, MinSize: Size{Width: 140, Height: 28},
						CueBanner: "YYYY-MM-DD", ToolTipText: "必填，格式 YYYY-MM-DD",
					},
					Label{Text: "截止日期"},
					LineEdit{
						AssignTo: &state.endDate, MinSize: Size{Width: 140, Height: 28},
						CueBanner: "YYYY-MM-DD", ToolTipText: "必填，格式 YYYY-MM-DD",
					},
					Label{Text: "快捷范围"},
					ComboBox{
						AssignTo:     &state.quickRange,
						Model:        []string{"最近一月", "本月", "最近三月", "今年"},
						CurrentIndex: 0, MinSize: Size{Width: 130, Height: 28},
						OnCurrentIndexChanged: ui.applyOutboundReportQuickRange,
					},
					Composite{
						ColumnSpan: 8,
						Layout:     HBox{Spacing: 8},
						Children: []Widget{
							Label{
								Text:      "日期按 Web 端现有时间戳口径提交；客户端不扩展查询边界。",
								TextColor: walk.RGB(90, 90, 90),
							},
							HSpacer{},
							PushButton{
								AssignTo: &state.reset, Text: "重置", MinSize: Size{Width: 88, Height: 30},
								OnClicked: ui.resetOutboundReportFilters,
							},
							PushButton{
								AssignTo: &state.query, Text: "查询", MinSize: Size{Width: 96, Height: 30},
								OnClicked: ui.loadOutboundReport,
							},
						},
					},
				},
			},
			GroupBox{
				Title:  "结果概览",
				Layout: HBox{Margins: Margins{Left: 14, Top: 10, Right: 14, Bottom: 10}},
				Children: []Widget{
					Label{
						AssignTo: &state.summary, Text: "请选择客户并查询。",
						Font:          Font{Family: "Microsoft YaHei UI", PointSize: 11, Bold: true},
						Accessibility: Accessibility{Name: "出库报表统计概览"},
					},
					HSpacer{},
					PushButton{
						AssignTo: &state.exportOrder, Text: "按单据导出", Enabled: false,
						MinSize:       Size{Width: 110, Height: 30},
						Accessibility: Accessibility{Name: "按出库单分组导出 Excel"},
						OnClicked:     func() { ui.exportOutboundReport("order") },
					},
					PushButton{
						AssignTo: &state.exportMaterial, Text: "按物料导出", Enabled: false,
						MinSize:       Size{Width: 110, Height: 30},
						Accessibility: Accessibility{Name: "按物料分组导出 Excel"},
						OnClicked:     func() { ui.exportOutboundReport("material") },
					},
				},
			},
			TableView{
				AssignTo:         &state.table,
				AlternatingRowBG: true,
				ColumnsOrderable: true,
				StretchFactor:    1,
				Accessibility: Accessibility{
					Name:        "出库报表明细",
					Description: "每行是一条服务端返回的出库物料签收记录",
				},
				Columns: []TableViewColumn{
					{Title: "产品型号", DataMember: "Model", Width: 150},
					{Title: "名称", DataMember: "Name", Width: 150},
					{Title: "规格", DataMember: "Specification", Width: 180},
					{Title: "出库日期", DataMember: "DepartureDate", Width: 100},
					{Title: "签收日期", DataMember: "ReceiptDate", Width: 100},
					{Title: "出库单编号", DataMember: "OrderCode", Width: 140},
					{Title: "单价", DataMember: "Price", Width: 90},
					{Title: "数量", DataMember: "Quantity", Width: 90},
					{Title: "单位", DataMember: "Unit", Width: 65},
					{Title: "金额", DataMember: "Amount", Width: 105},
				},
			},
			Label{AssignTo: &state.info, Text: "尚未查询", TextColor: walk.RGB(85, 85, 85)},
		},
	}
}

func (ui *mainUI) initializeOutboundReportPage() {
	state := ui.outboundReport
	if ui.outboundReportTab == nil || state == nil || state.customer == nil {
		return
	}
	state.generation++
	ui.setDefaultOutboundReportDates()
	for _, edit := range []*walk.LineEdit{state.startDate, state.endDate} {
		edit.KeyDown().Attach(func(key walk.Key) {
			if key == walk.KeyReturn {
				ui.loadOutboundReport()
			}
		})
	}
	ui.loadOutboundReportCustomers()
}

func (ui *mainUI) releaseOutboundReportPage() {
	if ui.outboundReport == nil {
		return
	}
	ui.outboundReport.generation++
	if ui.outboundReport.cancel != nil {
		ui.outboundReport.cancel()
	}
	ui.outboundReportTab = nil
	ui.outboundReport = newOutboundReportUI()
}

func (ui *mainUI) loadOutboundReportCustomers() {
	state := ui.outboundReport
	generation := state.generation
	go func() {
		customers, requestErr := ui.session.Client.Customers(context.Background())
		ui.window.Synchronize(func() {
			if state != ui.outboundReport || generation != state.generation || state.customer == nil {
				return
			}
			if requestErr != nil {
				state.info.SetText("客户列表加载失败：" + requestErr.Error() + "。按 F5 可重试。")
				_ = state.customer.SetModel([]string{"客户加载失败"})
				return
			}
			state.customerOptions = make([]selectOption, 0, len(customers))
			for _, customer := range customers {
				state.customerOptions = append(state.customerOptions, selectOption{
					ID: customer.ID, Label: customer.Name,
				})
			}
			_ = state.customer.SetModel(optionLabels("请选择客户", state.customerOptions))
			state.customer.SetCurrentIndex(0)
			state.info.SetText("客户列表已加载，请选择客户后查询。")
		})
	}()
}

func (ui *mainUI) refreshOutboundReportPage() {
	state := ui.outboundReport
	if state == nil || state.customer == nil {
		return
	}
	if len(state.customerOptions) == 0 {
		ui.loadOutboundReportCustomers()
		return
	}
	ui.loadOutboundReport()
}

func (ui *mainUI) setDefaultOutboundReportDates() {
	state := ui.outboundReport
	if state == nil || state.startDate == nil || state.endDate == nil {
		return
	}
	now := time.Now()
	state.startDate.SetText(now.AddDate(0, -1, 0).Format("2006-01-02"))
	state.endDate.SetText(now.Format("2006-01-02"))
}

func (ui *mainUI) applyOutboundReportQuickRange() {
	state := ui.outboundReport
	if ui.window == nil || state == nil || state.quickRange == nil || state.startDate == nil {
		return
	}
	now := time.Now()
	start := now.AddDate(0, -1, 0)
	switch state.quickRange.CurrentIndex() {
	case 1:
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	case 2:
		start = now.AddDate(0, -3, 0)
	case 3:
		start = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
	}
	state.startDate.SetText(start.Format("2006-01-02"))
	state.endDate.SetText(now.Format("2006-01-02"))
}

func (ui *mainUI) resetOutboundReportFilters() {
	state := ui.outboundReport
	if state == nil {
		return
	}
	state.customer.SetCurrentIndex(0)
	state.quickRange.SetCurrentIndex(0)
	ui.setDefaultOutboundReportDates()
	state.rows = nil
	state.searchedCustomer = ""
	state.searchedStart = 0
	state.searchedEnd = 0
	_ = state.table.SetModel([]outboundReportRow{})
	state.summary.SetText("请选择客户并查询。")
	state.info.SetText("筛选条件已重置。")
	state.exportOrder.SetEnabled(false)
	state.exportMaterial.SetEnabled(false)
}

func (ui *mainUI) loadOutboundReport() {
	state := ui.outboundReport
	if state == nil || state.table == nil {
		return
	}
	customerID := selectedOptionID(state.customer, state.customerOptions)
	if customerID == "" {
		state.info.SetText("请选择客户后再查询。")
		state.customer.SetFocus()
		return
	}
	start, err := parseRequiredReportDate(state.startDate.Text(), "起始日期")
	if err != nil {
		state.info.SetText(err.Error())
		state.startDate.SetFocus()
		return
	}
	end, err := parseRequiredReportDate(state.endDate.Text(), "截止日期")
	if err != nil {
		state.info.SetText(err.Error())
		state.endDate.SetFocus()
		return
	}
	if start > end {
		state.info.SetText("起始日期不能晚于截止日期。")
		state.startDate.SetFocus()
		return
	}
	if state.cancel != nil {
		state.cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	state.cancel = cancel
	state.generation++
	generation := state.generation
	customerName := state.customer.Text()
	state.info.SetText("正在加载线上出库汇总……")
	state.query.SetEnabled(false)
	state.reset.SetEnabled(false)
	state.exportOrder.SetEnabled(false)
	state.exportMaterial.SetEnabled(false)

	go func() {
		records, requestErr := ui.session.Client.OutboundSummary(ctx, customerID, start, end)
		if ctx.Err() != nil {
			return
		}
		ui.window.Synchronize(func() {
			if state != ui.outboundReport || generation != state.generation || state.table == nil {
				return
			}
			state.query.SetEnabled(true)
			state.reset.SetEnabled(true)
			if requestErr != nil {
				state.info.SetText("加载失败：" + requestErr.Error() + "。请核对客户和日期后重试。")
				return
			}
			sort.SliceStable(records, func(i, j int) bool {
				return records[i].Model > records[j].Model
			})
			rows := make([]outboundReportRow, 0, len(records))
			for _, record := range records {
				rows = append(rows, outboundReportRow{
					Model: record.Model, Name: record.Name, Specification: record.Specification,
					DepartureDate: formatReportDate(record.DepartureDate),
					ReceiptDate:   formatReportDate(record.ReceiptDate),
					OrderCode:     reportOrderCode(record),
					Price:         formatReportNumber(record.Price, 3),
					Quantity:      formatReportNumber(record.Quantity, 3),
					Unit:          record.Unit,
					Amount:        formatReportNumber(reportRecordAmount(record), 3),
				})
			}
			if modelErr := state.table.SetModel(rows); modelErr != nil {
				state.info.SetText("表格数据错误：" + modelErr.Error())
				return
			}
			state.rows = records
			state.searchedCustomer = customerName
			state.searchedStart = start
			state.searchedEnd = end
			orderCount := distinctOutboundOrderCount(records)
			quantity := reportTotalQuantity(records)
			amount := reportTotalAmount(records)
			state.summary.SetText(fmt.Sprintf(
				"单据 %d 张    明细 %d 条    数量 %s    金额 %s 元",
				orderCount, len(records), formatReportNumber(quantity, 3), formatReportNumber(amount, 3),
			))
			if len(records) == 0 {
				state.info.SetText("当前客户和日期范围内没有出库汇总记录。")
			} else {
				state.info.SetText(fmt.Sprintf("已加载 %d 条服务端汇总记录。", len(records)))
			}
			state.exportOrder.SetEnabled(len(records) > 0)
			state.exportMaterial.SetEnabled(len(records) > 0)
		})
	}()
}

func parseRequiredReportDate(text, field string) (int64, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0, fmt.Errorf("%s不能为空。", field)
	}
	value, err := time.ParseInLocation("2006-01-02", text, time.Local)
	if err != nil {
		return 0, fmt.Errorf("%s格式错误，请使用 YYYY-MM-DD。", field)
	}
	return value.Unix(), nil
}

func reportOrderCode(record api.OutboundSummaryRecord) string {
	if value := strings.TrimSpace(record.OrderCode); value != "" {
		return value
	}
	return strings.TrimSpace(record.Code)
}

func formatReportDate(value int64) string {
	if value <= 0 {
		return ""
	}
	return time.Unix(value, 0).Format("2006-01-02")
}

func formatReportNumber(value float64, decimals int) string {
	text := strconv.FormatFloat(value, 'f', decimals, 64)
	text = strings.TrimRight(strings.TrimRight(text, "0"), ".")
	if text == "" || text == "-0" {
		return "0"
	}
	return text
}

func decimalRat(value float64) *big.Rat {
	text := strconv.FormatFloat(value, 'f', -1, 64)
	result := new(big.Rat)
	if _, ok := result.SetString(text); ok {
		return result
	}
	return new(big.Rat)
}

func reportRecordAmount(record api.OutboundSummaryRecord) float64 {
	value, _ := new(big.Rat).Mul(decimalRat(record.Quantity), decimalRat(record.Price)).Float64()
	return value
}

func reportTotalQuantity(records []api.OutboundSummaryRecord) float64 {
	total := new(big.Rat)
	for _, record := range records {
		total.Add(total, decimalRat(record.Quantity))
	}
	value, _ := total.Float64()
	return value
}

func reportTotalAmount(records []api.OutboundSummaryRecord) float64 {
	total := new(big.Rat)
	for _, record := range records {
		total.Add(total, new(big.Rat).Mul(decimalRat(record.Quantity), decimalRat(record.Price)))
	}
	value, _ := total.Float64()
	return value
}

func distinctOutboundOrderCount(records []api.OutboundSummaryRecord) int {
	values := make(map[string]struct{})
	for _, record := range records {
		if code := reportOrderCode(record); code != "" {
			values[code] = struct{}{}
		}
	}
	return len(values)
}

func (ui *mainUI) exportOutboundReport(mode string) {
	state := ui.outboundReport
	if state == nil || len(state.rows) == 0 {
		walk.MsgBox(ui.window, "暂无可导出数据", "请先查询出库报表。", walk.MsgBoxIconInformation)
		return
	}
	exported, err := exportOutboundReportXLSX(
		ui.window, mode, state.rows, state.searchedCustomer, state.searchedStart, state.searchedEnd,
	)
	if err != nil {
		walk.MsgBox(ui.window, "导出失败", err.Error(), walk.MsgBoxIconError)
		return
	}
	if exported {
		state.info.SetText("Excel 报表已导出。")
	}
}
