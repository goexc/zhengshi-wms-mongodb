package ui

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"

	"zhengshi-wms-windowsapp/internal/api"
)

type materialQuoteRow struct {
	FirstDelivery string
	Customer      string
	Material      string
	Model         string
	Specification string
	Unit          string
	OrderCode     string
	Quantity      string
	FirstPrice    string
	Status        string
	QuoteNo       string
	LatestPrice   string
	Detail        api.NewCustomerMaterial
}

type materialQuoteUI struct {
	customer          *walk.ComboBox
	startDate         *walk.DateEdit
	endDate           *walk.DateEdit
	status            *walk.ComboBox
	materialName      *walk.LineEdit
	materialModel     *walk.LineEdit
	query             *walk.PushButton
	reset             *walk.PushButton
	export            *walk.PushButton
	exportStop        *walk.PushButton
	rebuild           *walk.PushButton
	rebuildStatus     *walk.PushButton
	quote             *walk.PushButton
	history           *walk.PushButton
	table             *walk.TableView
	info              *walk.Label
	pageSize          *walk.ComboBox
	prev              *walk.PushButton
	next              *walk.PushButton
	customers         []selectOption
	rows              []materialQuoteRow
	page              int
	total             int64
	generation        int
	cancel            context.CancelFunc
	exportGeneration  int
	exportCancel      context.CancelFunc
	latestTask        api.MaterialDeliveryRebuildTask
	ctx               context.Context
	disposeCancel     context.CancelFunc
	busy              bool
	exportBusy        bool
	rebuildBusy       bool
	dependenciesReady bool
	closed            atomic.Bool
}

func newMaterialQuoteUI() *materialQuoteUI {
	ctx, cancel := context.WithCancel(context.Background())
	return &materialQuoteUI{page: 1, ctx: ctx, disposeCancel: cancel}
}

func (state *materialQuoteUI) dispose() {
	if state == nil || state.closed.Swap(true) {
		return
	}
	if state.cancel != nil {
		state.cancel()
	}
	if state.exportCancel != nil {
		state.exportCancel()
	}
	if state.disposeCancel != nil {
		state.disposeCancel()
	}
}

var materialQuoteStatuses = []string{"全部状态", "未报价", "报价中", "已报价", "已定价"}

func (ui *mainUI) materialQuotePageWidget() TabPage {
	state := ui.materialQuote
	return TabPage{
		AssignTo: &ui.materialQuoteTab,
		Title:    closableTabTitle("新增物料报价"),
		Layout:   VBox{Margins: Margins{Left: 14, Top: 14, Right: 14, Bottom: 12}, Spacing: 10},
		Children: []Widget{
			Label{Text: "新增物料报价", Font: Font{Family: "Microsoft YaHei UI", PointSize: 15, Bold: true}},
			Label{Text: "按客户首次交付记录查询并调用现有报价接口；服务端状态和金额计算结果为最终依据。", TextColor: secondaryTextColor()},
			GroupBox{Title: "筛选条件", Layout: Grid{Columns: 8, Margins: Margins{Left: 14, Top: 12, Right: 14, Bottom: 14}, Spacing: 8}, Children: []Widget{
				Label{Text: "客户 *"}, ComboBox{AssignTo: &state.customer, Model: []string{"正在加载客户……"}, CurrentIndex: 0, MinSize: Size{Width: 200, Height: 28}},
				Label{Text: "首次交付开始 *"}, DateEdit{AssignTo: &state.startDate, Format: "yyyy-MM-dd", MinSize: Size{Width: 160, Height: 28}},
				Label{Text: "首次交付结束 *"}, DateEdit{AssignTo: &state.endDate, Format: "yyyy-MM-dd", MinSize: Size{Width: 160, Height: 28}},
				Label{Text: "报价状态"}, ComboBox{AssignTo: &state.status, Model: materialQuoteStatuses, CurrentIndex: 0, MinSize: Size{Width: 140, Height: 28}},
				Label{Text: "物料名称"}, LineEdit{AssignTo: &state.materialName, MinSize: Size{Width: 200, Height: 28}},
				Label{Text: "物料型号"}, LineEdit{AssignTo: &state.materialModel, MinSize: Size{Width: 180, Height: 28}},
				HSpacer{ColumnSpan: 4},
				Composite{ColumnSpan: 8, Layout: HBox{Spacing: 8}, Children: []Widget{
					HSpacer{},
					PushButton{AssignTo: &state.reset, Text: "重置", MinSize: Size{Width: 82, Height: 30}, OnClicked: ui.resetMaterialQuoteFilters},
					PushButton{AssignTo: &state.query, Text: "查询", Enabled: false, MinSize: Size{Width: 92, Height: 30}, OnClicked: func() { state.page = 1; ui.loadMaterialQuotes() }},
				}},
			}},
			Composite{Layout: HBox{Spacing: 8}, Children: []Widget{
				PushButton{AssignTo: &state.quote, Text: "新增/编辑报价", Enabled: false, MinSize: Size{Width: 112, Height: 30}, OnClicked: ui.openSelectedMaterialQuote},
				PushButton{AssignTo: &state.history, Text: "报价历史", Enabled: false, MinSize: Size{Width: 92, Height: 30}, OnClicked: ui.showSelectedMaterialQuoteHistory},
				PushButton{AssignTo: &state.export, Text: "导出当前筛选", Enabled: false, MinSize: Size{Width: 108, Height: 30}, OnClicked: ui.exportMaterialQuoteRows},
				PushButton{AssignTo: &state.exportStop, Text: "取消导出", Enabled: false, MinSize: Size{Width: 88, Height: 30}, Accessibility: Accessibility{Name: "取消报价 CSV 导出并清理未完成文件"}, OnClicked: ui.cancelMaterialQuoteExport},
				HSpacer{},
				PushButton{AssignTo: &state.rebuild, Text: "重建记录", Enabled: false, MinSize: Size{Width: 92, Height: 30}, ToolTipText: "调用现有接口按历史出库单重建客户新增物料记录", OnClicked: ui.rebuildMaterialQuoteDeliveries},
				PushButton{AssignTo: &state.rebuildStatus, Text: "任务状态：未读取", Enabled: false, MinSize: Size{Width: 150, Height: 30}, OnClicked: ui.showMaterialQuoteRebuildStatus},
			}},
			TableView{
				AssignTo: &state.table, Model: []materialQuoteRow{}, AlternatingRowBG: true, ColumnsOrderable: true, StretchFactor: 1,
				OnCurrentIndexChanged: ui.updateMaterialQuoteActions, OnItemActivated: ui.openSelectedMaterialQuote,
				Accessibility: Accessibility{Name: "客户新增物料报价列表", Description: "选择首次交付记录后新增或编辑报价"},
				Columns: []TableViewColumn{
					{Title: "首次交付", DataMember: "FirstDelivery", Width: 105},
					{Title: "客户", DataMember: "Customer", Width: 150},
					{Title: "物料名称", DataMember: "Material", Width: 180},
					{Title: "型号", DataMember: "Model", Width: 135},
					{Title: "规格", DataMember: "Specification", Width: 150},
					{Title: "单位", DataMember: "Unit", Width: 62},
					{Title: "首次出库单", DataMember: "OrderCode", Width: 145},
					{Title: "首次数量", DataMember: "Quantity", Width: 90},
					{Title: "首次单价", DataMember: "FirstPrice", Width: 90},
					{Title: "报价状态", DataMember: "Status", Width: 90},
					{Title: "最新报价", DataMember: "QuoteNo", Width: 135},
					{Title: "最新单价", DataMember: "LatestPrice", Width: 90},
				},
			},
			Composite{Layout: HBox{Spacing: 8}, Children: []Widget{
				Label{AssignTo: &state.info, Text: "正在加载客户选项……", TextColor: secondaryTextColor(), Accessibility: Accessibility{Name: "新增物料报价状态"}},
				HSpacer{}, Label{Text: "每页"},
				ComboBox{AssignTo: &state.pageSize, Model: pageSizeLabels, CurrentIndex: 1, MinSize: Size{Width: 92}, OnCurrentIndexChanged: func() {
					if state.pageSize != nil && state.pageSize.CurrentIndex() >= 0 && state.dependenciesReady {
						state.page = 1
						ui.loadMaterialQuotes()
					}
				}},
				PushButton{AssignTo: &state.prev, Text: "上一页", Enabled: false, OnClicked: func() {
					if state.page > 1 {
						state.page--
						ui.loadMaterialQuotes()
					}
				}},
				PushButton{AssignTo: &state.next, Text: "下一页", Enabled: false, OnClicked: func() { state.page++; ui.loadMaterialQuotes() }},
				PushButton{Text: "跳转页", Accessibility: Accessibility{Name: "跳转到指定报价结果页"}, OnClicked: func() {
					if page, ok := promptPageNumber(ui.window, state.page, state.total, selectedPageSize(state.pageSize)); ok {
						state.page = page
						ui.loadMaterialQuotes()
					}
				}},
			}},
		},
	}
}

func (ui *mainUI) initializeMaterialQuotePage() {
	state := ui.materialQuote
	if state == nil || state.customer == nil || state.closed.Load() {
		return
	}
	now := time.Now()
	state.startDate.SetDate(time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()))
	state.endDate.SetDate(now)
	ui.loadMaterialQuoteDependencies()
}

func (ui *mainUI) releaseMaterialQuotePage() {
	state := ui.materialQuote
	if state == nil {
		return
	}
	state.generation++
	if state.cancel != nil {
		state.cancel()
	}
	state.exportGeneration++
	if state.exportCancel != nil {
		state.exportCancel()
	}
	state.customer = nil
	state.startDate = nil
	state.endDate = nil
	state.status = nil
	state.materialName = nil
	state.materialModel = nil
	state.query = nil
	state.reset = nil
	state.export = nil
	state.exportStop = nil
	state.rebuild = nil
	state.rebuildStatus = nil
	state.quote = nil
	state.history = nil
	state.table = nil
	state.info = nil
	state.pageSize = nil
	state.prev = nil
	state.next = nil
	state.rows = nil
	state.total = 0
	state.page = 1
	state.busy = false
	state.exportBusy = false
	state.rebuildBusy = false
	state.dependenciesReady = false
	ui.materialQuoteTab = nil
}

func (ui *mainUI) loadMaterialQuoteDependencies() {
	state := ui.materialQuote
	if state == nil || state.customer == nil {
		return
	}
	state.info.SetText("正在加载客户选项和重建任务状态……")
	guardedGo(func() {
		customers, customerErr := ui.session.Client.Customers(state.ctx)
		task, taskErr := ui.session.Client.LatestMaterialDeliveryRebuild(state.ctx)
		if state.ctx.Err() != nil || state.closed.Load() {
			return
		}
		ui.window.Synchronize(func() {
			if state != ui.materialQuote || state.customer == nil {
				return
			}
			if customerErr != nil {
				_ = state.customer.SetModel([]string{"客户加载失败"})
				state.info.SetText("客户加载失败：" + customerErr.Error())
				return
			}
			state.customers = make([]selectOption, 0, len(customers))
			for _, customer := range customers {
				state.customers = append(state.customers, selectOption{ID: customer.ID, Label: businessOptionLabel(customer.Name, customer.Code)})
			}
			_ = state.customer.SetModel(optionLabels("请选择客户", state.customers))
			state.customer.SetCurrentIndex(0)
			state.dependenciesReady = true
			state.query.SetEnabled(true)
			state.export.SetEnabled(true)
			state.rebuild.SetEnabled(true)
			if taskErr == nil && task.ID != "" {
				state.latestTask = task
				ui.updateMaterialQuoteTaskButton()
			} else if taskErr != nil {
				state.rebuildStatus.SetText("任务状态读取失败")
			}
			state.info.SetText(fmt.Sprintf("已加载 %d 个客户；请选择客户后查询。", len(state.customers)))
		})
	})
}

func materialQuoteStatusValue(label string) string {
	return map[string]string{"未报价": "unquoted", "报价中": "quoting", "已报价": "quoted", "已定价": "priced"}[label]
}

func materialQuoteStatusLabel(value string) string {
	return map[string]string{"unquoted": "未报价", "quoting": "报价中", "quoted": "已报价", "priced": "已定价", "draft": "草稿", "submitted": "已提交", "void": "已作废"}[value]
}

func displayQuotePrice(value float64) string {
	if value <= 0 {
		return "—"
	}
	return fmt.Sprintf("%.3f", value)
}

func (ui *mainUI) materialQuoteFilters() (api.NewCustomerMaterialFilters, error) {
	state := ui.materialQuote
	filters := api.NewCustomerMaterialFilters{CustomerID: selectedOptionID(state.customer, state.customers), MaterialName: strings.TrimSpace(state.materialName.Text()), MaterialModel: strings.TrimSpace(state.materialModel.Text())}
	if filters.CustomerID == "" {
		return filters, fmt.Errorf("请选择客户")
	}
	start := state.startDate.Date()
	end := state.endDate.Date()
	if start.IsZero() || end.IsZero() {
		return filters, fmt.Errorf("请选择首次交付日期范围")
	}
	start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location())
	end = time.Date(end.Year(), end.Month(), end.Day(), 23, 59, 59, 0, end.Location())
	if end.Before(start) {
		return filters, fmt.Errorf("首次交付结束日期不能早于开始日期")
	}
	filters.StartTime = start.Unix()
	filters.EndTime = end.Unix()
	if state.status.CurrentIndex() > 0 {
		filters.QuoteStatus = materialQuoteStatusValue(state.status.Text())
	}
	return filters, nil
}

func (ui *mainUI) loadMaterialQuotes() {
	state := ui.materialQuote
	if state == nil || state.table == nil || state.busy {
		return
	}
	filters, err := ui.materialQuoteFilters()
	if err != nil {
		state.info.SetText(err.Error())
		return
	}
	if state.cancel != nil {
		state.cancel()
	}
	ctx, cancel := context.WithCancel(state.ctx)
	state.cancel = cancel
	state.generation++
	generation := state.generation
	page := state.page
	size := selectedPageSize(state.pageSize)
	state.busy = true
	state.info.SetText("正在读取客户首次交付记录……")
	ui.updateMaterialQuoteControls()
	guardedGo(func() {
		result, requestErr := ui.session.Client.NewCustomerMaterials(ctx, page, size, filters)
		if ctx.Err() != nil || state.closed.Load() {
			return
		}
		ui.window.Synchronize(func() {
			if state != ui.materialQuote || state.table == nil || generation != state.generation {
				return
			}
			state.busy = false
			if requestErr != nil {
				state.rows = nil
				_ = state.table.SetModel([]materialQuoteRow{})
				state.info.SetText(requestFailureText(requestErr))
				ui.updateMaterialQuoteControls()
				return
			}
			rows := make([]materialQuoteRow, 0, len(result.List))
			for _, item := range result.List {
				status := materialQuoteStatusLabel(item.QuoteStatus)
				if status == "" {
					status = displayMaterialValue(item.QuoteStatus)
				}
				rows = append(rows, materialQuoteRow{
					FirstDelivery: formatUnixMinute(item.FirstDeliveryTime), Customer: item.CustomerName, Material: item.MaterialName,
					Model: item.MaterialModel, Specification: item.MaterialSpecification, Unit: item.MaterialUnit,
					OrderCode: item.FirstDeliveryOrderCode, Quantity: fmt.Sprintf("%g", item.FirstDeliveryQuantity), FirstPrice: displayQuotePrice(item.FirstDeliveryPrice),
					Status: status, QuoteNo: displayMaterialValue(item.LatestQuoteNo), LatestPrice: displayQuotePrice(item.LatestPrice), Detail: item,
				})
			}
			state.rows = rows
			state.total = result.Total
			if modelErr := state.table.SetModel(rows); modelErr != nil {
				state.info.SetText("报价表格刷新失败：" + modelErr.Error())
				return
			}
			state.info.SetText(fmt.Sprintf("第 %d 页 | 本页 %d 条 | 共 %d 条", page, len(rows), result.Total))
			ui.updateMaterialQuoteControls()
		})
	})
}

func (ui *mainUI) selectedMaterialQuoteDelivery() (api.NewCustomerMaterial, bool) {
	state := ui.materialQuote
	if state == nil || state.table == nil {
		return api.NewCustomerMaterial{}, false
	}
	index := state.table.CurrentIndex()
	if index < 0 || index >= len(state.rows) {
		return api.NewCustomerMaterial{}, false
	}
	return state.rows[index].Detail, true
}

func (ui *mainUI) updateMaterialQuoteActions() { ui.updateMaterialQuoteControls() }

func (ui *mainUI) updateMaterialQuoteControls() {
	state := ui.materialQuote
	if state == nil || state.query == nil {
		return
	}
	_, selected := ui.selectedMaterialQuoteDelivery()
	state.query.SetEnabled(state.dependenciesReady && !state.busy && !state.rebuildBusy)
	state.reset.SetEnabled(!state.busy && !state.rebuildBusy)
	state.export.SetEnabled(state.dependenciesReady && !state.busy && !state.exportBusy && !state.rebuildBusy)
	if state.exportStop != nil {
		state.exportStop.SetEnabled(state.exportBusy)
	}
	state.quote.SetEnabled(selected && !state.busy && !state.rebuildBusy)
	state.history.SetEnabled(selected && !state.busy && !state.rebuildBusy)
	state.prev.SetEnabled(!state.busy && state.page > 1)
	state.next.SetEnabled(!state.busy && int64(state.page*selectedPageSize(state.pageSize)) < state.total)
	state.rebuild.SetEnabled(state.dependenciesReady && !state.busy && !state.rebuildBusy && state.latestTask.Status != "queued" && state.latestTask.Status != "running")
}

func (ui *mainUI) resetMaterialQuoteFilters() {
	state := ui.materialQuote
	if state == nil {
		return
	}
	now := time.Now()
	state.customer.SetCurrentIndex(0)
	state.startDate.SetDate(time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()))
	state.endDate.SetDate(now)
	state.status.SetCurrentIndex(0)
	state.materialName.SetText("")
	state.materialModel.SetText("")
	state.page = 1
	state.total = 0
	state.rows = nil
	_ = state.table.SetModel([]materialQuoteRow{})
	state.info.SetText("筛选条件已重置；请选择客户后查询。")
	ui.updateMaterialQuoteControls()
}

func (ui *mainUI) openSelectedMaterialQuote() {
	delivery, ok := ui.selectedMaterialQuoteDelivery()
	if !ok {
		walk.MsgBox(ui.window, "请选择记录", "请先选择一条客户首次交付记录。", walk.MsgBoxIconInformation)
		return
	}
	ui.openMaterialQuoteEditor(delivery)
}

func (ui *mainUI) showSelectedMaterialQuoteHistory() {
	delivery, ok := ui.selectedMaterialQuoteDelivery()
	if !ok {
		return
	}
	ShowMaterialQuoteHistory(ui.window, ui.session.Client, delivery)
}

func (ui *mainUI) exportMaterialQuoteRows() {
	state := ui.materialQuote
	if state == nil || state.exportBusy {
		return
	}
	filters, err := ui.materialQuoteFilters()
	if err != nil {
		state.info.SetText(err.Error())
		return
	}
	dialog := new(walk.FileDialog)
	dialog.Title = "导出客户新增物料报价"
	dialog.Filter = "CSV 文件 (*.csv)|*.csv"
	dialog.FilePath = "客户新增物料报价-" + time.Now().Format("20060102") + ".csv"
	ok, chooseErr := dialog.ShowSave(ui.window)
	if chooseErr != nil {
		state.info.SetText("选择导出位置失败：" + chooseErr.Error())
		return
	}
	if !ok {
		return
	}
	target := strings.TrimSpace(dialog.FilePath)
	if filepath.Ext(target) == "" {
		target += ".csv"
	}
	if state.exportCancel != nil {
		state.exportCancel()
	}
	ctx, cancel := context.WithCancel(state.ctx)
	state.exportCancel = cancel
	state.exportGeneration++
	generation := state.exportGeneration
	state.exportBusy = true
	state.info.SetText("正在由服务端生成报价 CSV……")
	ui.updateMaterialQuoteControls()
	request := api.NewCustomerMaterialExportRequest{CustomerID: filters.CustomerID, StartTime: filters.StartTime, EndTime: filters.EndTime, QuoteStatus: filters.QuoteStatus, MaterialName: filters.MaterialName, MaterialModel: filters.MaterialModel}
	finishTask := ui.startBackgroundTask("报价 CSV 导出", func() {
		if state == ui.materialQuote && generation == state.exportGeneration {
			ui.cancelMaterialQuoteExport()
		}
	})
	guardedGo(func() {
		defer finishTask()
		data, _, requestErr := ui.session.Client.ExportNewCustomerMaterialQuotes(ctx, request)
		if requestErr == nil {
			requestErr = writeFileAtomic(ctx, target, data, 0o600)
		}
		if ctx.Err() != nil || state.closed.Load() {
			return
		}
		ui.window.Synchronize(func() {
			if state != ui.materialQuote || generation != state.exportGeneration {
				return
			}
			state.exportBusy = false
			state.exportCancel = nil
			ui.updateMaterialQuoteControls()
			if requestErr != nil {
				state.info.SetText("导出失败：" + requestErr.Error() + "；可修正后重试。")
				return
			}
			state.info.SetText("报价 CSV 已保存：" + target)
		})
	})
}

func (ui *mainUI) cancelMaterialQuoteExport() {
	state := ui.materialQuote
	if state == nil || !state.exportBusy {
		return
	}
	if state.exportCancel != nil {
		state.exportCancel()
	}
	state.exportGeneration++
	state.exportCancel = nil
	state.exportBusy = false
	state.info.SetText("报价 CSV 导出已取消；目标文件不会被半成品覆盖，临时文件正在清理。")
	ui.updateMaterialQuoteControls()
}

func materialRebuildStatusLabel(status string) string {
	label := map[string]string{"queued": "等待执行", "running": "执行中", "success": "执行成功", "failed": "执行失败"}[status]
	if label == "" {
		return displayMaterialValue(status)
	}
	return label
}

func (ui *mainUI) updateMaterialQuoteTaskButton() {
	state := ui.materialQuote
	if state == nil || state.rebuildStatus == nil {
		return
	}
	if state.latestTask.ID == "" {
		state.rebuildStatus.SetText("任务状态：无记录")
		state.rebuildStatus.SetEnabled(false)
		return
	}
	state.rebuildStatus.SetText("任务状态：" + materialRebuildStatusLabel(state.latestTask.Status))
	state.rebuildStatus.SetEnabled(true)
	ui.updateMaterialQuoteControls()
}

func (ui *mainUI) rebuildMaterialQuoteDeliveries() {
	state := ui.materialQuote
	if state == nil || state.rebuildBusy {
		return
	}
	if walk.MsgBox(ui.window, "确认重建记录", "将调用现有接口，根据历史出库单重建客户新增物料记录；报价状态由服务端尽量保留。是否继续？", walk.MsgBoxYesNo|walk.MsgBoxIconWarning) != walk.DlgCmdYes {
		return
	}
	state.rebuildBusy = true
	state.info.SetText("正在提交重建任务；请求不会自动重试……")
	ui.updateMaterialQuoteControls()
	guardedGo(func() {
		task, err := ui.session.Client.StartMaterialDeliveryRebuild(state.ctx)
		if state.ctx.Err() != nil || state.closed.Load() {
			return
		}
		ui.window.Synchronize(func() {
			if state != ui.materialQuote || state.info == nil {
				return
			}
			state.rebuildBusy = false
			if err != nil {
				state.info.SetText(requestFailureText(err))
				ui.updateMaterialQuoteControls()
				return
			}
			state.latestTask = task
			state.info.SetText("重建任务已提交；可点击任务状态查看并手动刷新。")
			ui.updateMaterialQuoteTaskButton()
		})
	})
}

func (ui *mainUI) showMaterialQuoteRebuildStatus() {
	state := ui.materialQuote
	if state == nil || state.latestTask.ID == "" {
		return
	}
	message := fmt.Sprintf("状态：%s\r\n订单数：%d\r\n生成记录：%d\r\n消息：%s\r\n错误：%s\r\n更新时间：%s", materialRebuildStatusLabel(state.latestTask.Status), state.latestTask.OrderCount, state.latestTask.DeliveryCount, displayMaterialValue(state.latestTask.Message), displayMaterialValue(state.latestTask.ErrorMessage), formatUnixMinute(state.latestTask.UpdatedAt))
	if walk.MsgBox(ui.window, "重建任务状态", message+"\r\n\r\n是否刷新线上任务状态？", walk.MsgBoxYesNo|walk.MsgBoxIconInformation) != walk.DlgCmdYes {
		return
	}
	state.info.SetText("正在刷新重建任务状态……")
	guardedGo(func() {
		task, err := ui.session.Client.LatestMaterialDeliveryRebuild(state.ctx)
		if state.ctx.Err() != nil || state.closed.Load() {
			return
		}
		ui.window.Synchronize(func() {
			if state != ui.materialQuote || state.info == nil {
				return
			}
			if err != nil {
				state.info.SetText(requestFailureText(err))
				return
			}
			state.latestTask = task
			ui.updateMaterialQuoteTaskButton()
			state.info.SetText("重建任务状态已刷新。")
			if task.Status == "success" && selectedOptionID(state.customer, state.customers) != "" {
				state.page = 1
				ui.loadMaterialQuotes()
			}
		})
	})
}

func ShowMaterialQuoteHistory(owner walk.Form, client *api.Client, delivery api.NewCustomerMaterial) {
	var dlg *walk.Dialog
	var table *walk.TableView
	var info *walk.Label
	type historyRow struct{ QuoteNo, Status, Mode, Currency, Price, Validity, Creator, Updated string }
	err := Dialog{
		AssignTo: &dlg, Title: "报价历史 · " + delivery.MaterialName, MinSize: Size{Width: 900, Height: 500}, Size: Size{Width: 1040, Height: 620},
		Layout: VBox{Margins: Margins{Left: 14, Top: 14, Right: 14, Bottom: 12}, Spacing: 10},
		Children: []Widget{
			Label{Text: delivery.CustomerName + " · " + delivery.MaterialName + " · " + delivery.MaterialModel, Font: Font{Bold: true}},
			TableView{AssignTo: &table, Model: []historyRow{}, AlternatingRowBG: true, StretchFactor: 1, Columns: []TableViewColumn{
				{Title: "报价单号", DataMember: "QuoteNo", Width: 150}, {Title: "状态", DataMember: "Status", Width: 85}, {Title: "方式", DataMember: "Mode", Width: 85},
				{Title: "币种", DataMember: "Currency", Width: 70}, {Title: "最终单价", DataMember: "Price", Width: 100}, {Title: "有效期", DataMember: "Validity", Width: 230},
				{Title: "创建人", DataMember: "Creator", Width: 100}, {Title: "更新时间", DataMember: "Updated", Width: 140},
			}},
			Composite{Layout: HBox{}, Children: []Widget{Label{AssignTo: &info, Text: "正在读取报价历史……"}, HSpacer{}, PushButton{Text: "关闭", OnClicked: func() { dlg.Accept() }}}},
		},
	}.Create(owner)
	if err != nil {
		walk.MsgBox(owner, "报价历史窗口错误", err.Error(), walk.MsgBoxIconError)
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	dlg.Disposing().Attach(func() { cancel() })
	guardedGo(func() {
		result, requestErr := client.MaterialQuotes(ctx, 1, 100, api.MaterialQuoteFilters{DeliveryID: delivery.ID})
		if ctx.Err() != nil {
			return
		}
		dlg.Synchronize(func() {
			if requestErr != nil {
				info.SetText(requestFailureText(requestErr))
				return
			}
			rows := make([]historyRow, 0, len(result.List))
			for _, quote := range result.List {
				status := materialQuoteStatusLabel(quote.Status)
				if status == "" {
					status = displayMaterialValue(quote.Status)
				}
				mode := map[string]string{"detailed": "详细报价", "simple": "简单报价"}[quote.QuoteMode]
				if mode == "" {
					mode = displayMaterialValue(quote.QuoteMode)
				}
				validity := "—"
				if quote.ValidFrom > 0 || quote.ValidTo > 0 {
					validity = formatUnixMinute(quote.ValidFrom) + " 至 " + formatUnixMinute(quote.ValidTo)
				}
				rows = append(rows, historyRow{quote.QuoteNo, status, mode, quote.Currency, displayQuotePrice(quote.FinalPrice), validity, quote.CreatorName, formatUnixMinute(quote.UpdatedAt)})
			}
			_ = table.SetModel(rows)
			info.SetText(fmt.Sprintf("已读取 %d 条报价历史。", len(rows)))
		})
	})
	dlg.Run()
}
