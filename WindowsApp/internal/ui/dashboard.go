package ui

import (
	"context"
	"fmt"
	"sync"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"

	"zhengshi-wms-windowsapp/internal/api"
)

type dashboardCardSpec struct {
	Key      string
	Label    string
	Module   string
	Status   string
	Hint     string
	MenuPath string
}

type dashboardCard struct {
	spec   dashboardCardSpec
	count  *walk.Label
	status *walk.Label
	open   *walk.PushButton
}

type dashboardUI struct {
	cards      []*dashboardCard
	info       *walk.Label
	refresh    *walk.PushButton
	generation int
	cancel     context.CancelFunc
}

type dashboardLoadResult struct {
	index int
	total int64
	err   error
}

func dashboardCardSpecs(perms api.Perms) []dashboardCardSpec {
	var result []dashboardCardSpec
	if hasMenuPath(perms.Menus, "/inbound/receipt") && hasButton(perms.Buttons, "inbound:receipt:list") {
		result = append(result,
			dashboardCardSpec{Key: "inbound_pending", Label: "入库待审核", Module: "inbound", Status: "待审核", Hint: "等待业务审核", MenuPath: "/inbound/receipt"},
			dashboardCardSpec{Key: "inbound_rejected", Label: "入库审核不通过", Module: "inbound", Status: "审核不通过", Hint: "需要修改或删除", MenuPath: "/inbound/receipt"},
			dashboardCardSpec{Key: "inbound_partial", Label: "入库部分入库", Module: "inbound", Status: "部分入库", Hint: "等待继续收货", MenuPath: "/inbound/receipt"},
		)
	}
	if hasAnyMenuPath(perms.Menus, "/outbound/receipt", "/outbound/receipt2") && hasButton(perms.Buttons, "outbound:order:list") {
		result = append(result,
			dashboardCardSpec{Key: "outbound_pre", Label: "出库预发货", Module: "outbound", Status: "预发货", Hint: "等待确认", MenuPath: "/outbound/receipt"},
			dashboardCardSpec{Key: "outbound_pick", Label: "出库待拣货", Module: "outbound", Status: "待拣货", Hint: "等待拣货", MenuPath: "/outbound/receipt"},
			dashboardCardSpec{Key: "outbound_pack", Label: "出库待打包", Module: "outbound", Status: "待打包", Hint: "等待打包", MenuPath: "/outbound/receipt"},
			dashboardCardSpec{Key: "outbound_weigh", Label: "出库待称重", Module: "outbound", Status: "待称重", Hint: "等待称重", MenuPath: "/outbound/receipt"},
			dashboardCardSpec{Key: "outbound_receipt", Label: "出库待签收", Module: "outbound", Status: "已出库", Hint: "等待客户签收", MenuPath: "/outbound/receipt"},
		)
	}
	return result
}

func newDashboardUI(specs []dashboardCardSpec) *dashboardUI {
	state := &dashboardUI{}
	for _, spec := range specs {
		state.cards = append(state.cards, &dashboardCard{spec: spec})
	}
	return state
}

func (ui *mainUI) dashboardPageWidget() TabPage {
	state := ui.dashboard
	cardWidgets := make([]Widget, 0, len(state.cards))
	for _, card := range state.cards {
		current := card
		cardWidgets = append(cardWidgets, GroupBox{
			Title:   current.spec.Label,
			MinSize: Size{Width: 210, Height: 132},
			Layout:  VBox{Margins: Margins{Left: 14, Top: 10, Right: 14, Bottom: 12}, Spacing: 6},
			Children: []Widget{
				Label{
					AssignTo: &current.count, Text: "—",
					Font:          Font{Family: "Microsoft YaHei UI", PointSize: 22, Bold: true},
					Accessibility: Accessibility{Name: current.spec.Label + "数量"},
				},
				Label{
					AssignTo: &current.status, Text: current.spec.Hint,
					TextColor: secondaryTextColor(),
				},
				HSpacer{},
				PushButton{
					AssignTo: &current.open, Text: "打开对应队列", MinSize: Size{Width: 118, Height: 30},
					Accessibility: Accessibility{Name: "打开" + current.spec.Label + "队列"},
					OnClicked:     func() { ui.openDashboardCard(current.spec) },
				},
			},
		})
	}
	return TabPage{
		AssignTo: &ui.dashboardTab,
		Title:    closableTabTitle("仓库作业台"),
		Layout:   VBox{Margins: Margins{Left: 14, Top: 14, Right: 14, Bottom: 12}, Spacing: 10},
		Children: []Widget{
			Label{Text: "仓库作业台", Font: Font{Family: "Microsoft YaHei UI", PointSize: 15, Bold: true}},
			Label{
				Text:          "按当前账号权限汇总现有入库、出库接口中的待办状态；点击卡片进入对应筛选队列。",
				TextColor:     secondaryTextColor(),
				Accessibility: Accessibility{Name: "仓库作业台说明"},
			},
			Composite{
				Layout:   Grid{Columns: 4, Spacing: 12},
				Children: cardWidgets,
			},
			VSpacer{},
			Composite{Layout: HBox{Spacing: 8}, Children: []Widget{
				Label{AssignTo: &state.info, Text: "尚未刷新", TextColor: secondaryTextColor()},
				HSpacer{},
				PushButton{
					AssignTo: &state.refresh, Text: "刷新待办", MinSize: Size{Width: 96, Height: 30},
					Accessibility: Accessibility{Name: "刷新仓库作业待办"},
					OnClicked:     ui.loadDashboard,
				},
			}},
		},
	}
}

func (ui *mainUI) initializeDashboardPage() {
	if ui.dashboardTab == nil || ui.dashboard == nil || ui.dashboard.info == nil {
		return
	}
	ui.dashboard.generation++
	ui.loadDashboard()
}

func (ui *mainUI) releaseDashboardPage() {
	state := ui.dashboard
	if state == nil {
		return
	}
	state.generation++
	if state.cancel != nil {
		state.cancel()
	}
	ui.dashboardTab = nil
	specs := make([]dashboardCardSpec, 0, len(state.cards))
	for _, card := range state.cards {
		specs = append(specs, card.spec)
	}
	ui.dashboard = newDashboardUI(specs)
}

func (ui *mainUI) loadDashboard() {
	state := ui.dashboard
	if state == nil || state.info == nil || len(state.cards) == 0 {
		return
	}
	if state.cancel != nil {
		state.cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	state.cancel = cancel
	state.generation++
	generation := state.generation
	state.refresh.SetEnabled(false)
	state.info.SetText("正在读取线上待办数量……")
	for _, card := range state.cards {
		card.count.SetText("…")
		card.status.SetText("正在加载")
		card.open.SetEnabled(false)
	}

	guardedGo(func() {
		results := make(chan dashboardLoadResult, len(state.cards))
		sem := make(chan struct{}, 4)
		var wg sync.WaitGroup
		for index, card := range state.cards {
			wg.Add(1)
			guardedGo2(index, card.spec, func(index int, spec dashboardCardSpec) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				var total int64
				var err error
				switch spec.Module {
				case "inbound":
					var page api.InboundPage
					page, err = ui.session.Client.InboundReceipts(ctx, 1, 5, api.InboundFilters{Status: spec.Status})
					total = page.Total
				case "outbound":
					var page api.OutboundPage
					page, err = ui.session.Client.OutboundOrders(ctx, 1, 10, api.OutboundFilters{
						Status: spec.Status, IsPack: -1, IsWeigh: -1,
					})
					total = page.Total
				default:
					err = fmt.Errorf("未知待办模块：%s", spec.Module)
				}
				results <- dashboardLoadResult{index: index, total: total, err: err}
			})
		}
		guardedGo(func() {
			wg.Wait()
			close(results)
		})

		failed := 0
		for result := range results {
			if ctx.Err() != nil {
				return
			}
			ui.window.Synchronize(func() {
				if state != ui.dashboard || generation != state.generation || result.index >= len(state.cards) {
					return
				}
				card := state.cards[result.index]
				if result.err != nil {
					failed++
					card.count.SetText("—")
					card.status.SetText("加载失败：" + result.err.Error())
					card.open.SetEnabled(true)
					return
				}
				card.count.SetText(fmt.Sprint(result.total))
				card.status.SetText(card.spec.Hint)
				card.open.SetEnabled(true)
			})
		}
		if ctx.Err() != nil {
			return
		}
		ui.window.Synchronize(func() {
			if state != ui.dashboard || generation != state.generation {
				return
			}
			state.refresh.SetEnabled(true)
			if failed > 0 {
				state.info.SetText(fmt.Sprintf("待办刷新完成，%d 个状态加载失败；可单独打开队列重试。", failed))
			} else {
				state.info.SetText("待办已从线上接口刷新。按 F5 可重新读取。")
			}
		})
	})
}

func (ui *mainUI) openDashboardCard(spec dashboardCardSpec) {
	switch spec.Module {
	case "inbound":
		ui.openTab("inbound")
		if ui.inboundStatus == nil {
			return
		}
		ui.inboundSearch.SetText("")
		ui.inboundType.SetCurrentIndex(0)
		ui.inboundSupplier.SetCurrentIndex(0)
		ui.inboundCustomer.SetCurrentIndex(0)
		if index := stringIndex(inboundStatuses, spec.Status); index >= 0 {
			ui.inboundStatus.SetCurrentIndex(index)
		}
		ui.inboundPage = 1
		ui.loadInbound()
	case "outbound":
		ui.openTab("outbound")
		state := ui.outbound
		if state == nil || state.stage == nil {
			return
		}
		state.search.SetText("")
		state.orderType.SetCurrentIndex(0)
		state.supplier.SetCurrentIndex(0)
		state.customer.SetCurrentIndex(0)
		state.startDate.SetText("")
		state.endDate.SetText("")
		for index, stage := range outboundStages {
			if stage.Status == spec.Status {
				state.stage.SetCurrentIndex(index)
				break
			}
		}
		state.page = 1
		ui.loadOutbound()
	}
}
