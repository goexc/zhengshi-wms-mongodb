package ui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"

	"zhengshi-wms-windowsapp/internal/api"
	"zhengshi-wms-windowsapp/internal/config"
)

type globalLookupResult struct {
	Kind     string
	Code     string
	Name     string
	Status   string
	Summary  string
	Material api.Material
	Inbound  api.InboundReceipt
	Outbound api.OutboundOrder
}

type globalLookupRow struct {
	Type    string
	Code    string
	Name    string
	Status  string
	Summary string
}

type globalLookupUI struct {
	splitter   *walk.Splitter
	kind       *walk.ComboBox
	query      *walk.LineEdit
	search     *walk.PushButton
	reset      *walk.PushButton
	table      *walk.TableView
	recentList *walk.ListBox
	info       *walk.Label
	detail     *walk.PushButton
	locate     *walk.PushButton
	results    []globalLookupResult
	recent     []string
	kindKeys   []string
	generation int
	cancel     context.CancelFunc
	busy       bool
}

func newGlobalLookupUI() *globalLookupUI { return &globalLookupUI{} }

func availableGlobalLookupKinds(perms api.Perms) ([]string, []string) {
	labels := []string{"自动识别"}
	keys := []string{"auto"}
	if hasMenuPath(perms.Menus, "/material/list") {
		labels = append(labels, "物料")
		keys = append(keys, "material")
	}
	if hasMenuPath(perms.Menus, "/inbound/receipt") {
		labels = append(labels, "入库单")
		keys = append(keys, "inbound")
	}
	if hasAnyMenuPath(perms.Menus, "/outbound/receipt", "/outbound/receipt2") {
		labels = append(labels, "出库单")
		keys = append(keys, "outbound")
	}
	return labels, keys
}

func (ui *mainUI) globalLookupPageWidget() TabPage {
	if ui.globalLookup == nil {
		ui.globalLookup = newGlobalLookupUI()
	}
	state := ui.globalLookup
	labels, keys := availableGlobalLookupKinds(ui.session.Perms)
	state.kindKeys = keys
	return TabPage{
		AssignTo: &ui.globalLookupTab,
		Title:    closableTabTitle("统一检索"),
		Layout:   VBox{Margins: Margins{Left: 16, Top: 16, Right: 16, Bottom: 14}, Spacing: 10},
		Children: []Widget{
			Label{Text: "统一扫码与业务检索", Font: Font{Family: "Microsoft YaHei UI", PointSize: 15, Bold: true}},
			Label{Text: "扫描枪可直接输入单号或物料关键字并发送 Enter；自动识别只调用当前账号已有的查询接口，不触发任何业务写入。", TextColor: secondaryTextColor()},
			GroupBox{Title: "检索条件", Layout: Grid{Columns: 4, Margins: Margins{Left: 14, Top: 12, Right: 14, Bottom: 14}, Spacing: 10}, Children: []Widget{
				Label{Text: "检索类型"},
				ComboBox{AssignTo: &state.kind, Model: labels, CurrentIndex: 0, MinSize: Size{Width: 160, Height: 30}, Accessibility: Accessibility{Name: "统一检索类型"}},
				Label{Text: "单号/物料关键字"},
				LineEdit{AssignTo: &state.query, MinSize: Size{Width: 320, Height: 30}, CueBanner: "扫码或输入后按 Enter", Accessibility: Accessibility{Name: "统一检索关键字"}},
				Composite{ColumnSpan: 4, Layout: HBox{Spacing: 8}, Children: []Widget{
					HSpacer{},
					PushButton{AssignTo: &state.reset, Text: "清空", MinSize: Size{Width: 84, Height: 30}, OnClicked: ui.resetGlobalLookup},
					PushButton{AssignTo: &state.search, Text: "在线检索", MinSize: Size{Width: 104, Height: 32}, OnClicked: ui.runGlobalLookup},
				}},
			}},
			HSplitter{AssignTo: &state.splitter, HandleWidth: 5, StretchFactor: 1, Children: []Widget{
				GroupBox{Title: "检索结果", StretchFactor: 5, MinSize: Size{Width: 700, Height: 380}, Layout: VBox{Margins: Margins{Left: 10, Top: 10, Right: 10, Bottom: 10}, Spacing: 8}, Children: []Widget{
					TableView{
						AssignTo: &state.table, Model: []globalLookupRow{}, AlternatingRowBG: true, ColumnsOrderable: true, StretchFactor: 1,
						Accessibility:         Accessibility{Name: "统一检索结果", Description: "激活当前行查看线上详情"},
						OnCurrentIndexChanged: ui.updateGlobalLookupActions,
						OnItemActivated:       ui.showSelectedGlobalLookupDetail,
						Columns: []TableViewColumn{
							{Title: "类型", DataMember: "Type", Width: 80},
							{Title: "单号/型号", DataMember: "Code", Width: 170},
							{Title: "名称/往来单位", DataMember: "Name", Width: 190},
							{Title: "状态", DataMember: "Status", Width: 100},
							{Title: "摘要", DataMember: "Summary", Width: 300},
						},
					},
				}},
				GroupBox{Title: "本次运行最近检索", StretchFactor: 2, MinSize: Size{Width: 250, Height: 380}, Layout: VBox{Margins: Margins{Left: 10, Top: 10, Right: 10, Bottom: 10}, Spacing: 8}, Children: []Widget{
					Label{Text: "仅保存在内存中，退出客户端后清空。", TextColor: secondaryTextColor()},
					ListBox{AssignTo: &state.recentList, Model: []string{}, StretchFactor: 1, Accessibility: Accessibility{Name: "本次运行最近检索"}, OnCurrentIndexChanged: ui.useSelectedRecentLookup, OnItemActivated: ui.runSelectedRecentLookup},
				}},
			}},
			Label{AssignTo: &state.info, Text: "请输入检索内容；也可按 Ctrl+K 随时打开本页。", TextColor: secondaryTextColor(), Accessibility: Accessibility{Name: "统一检索状态"}},
			Composite{Layout: HBox{Spacing: 8}, Children: []Widget{
				HSpacer{},
				PushButton{AssignTo: &state.locate, Text: "定位到业务模块", Enabled: false, MinSize: Size{Width: 126, Height: 30}, OnClicked: ui.locateSelectedGlobalLookupResult},
				PushButton{AssignTo: &state.detail, Text: "查看线上详情", Enabled: false, MinSize: Size{Width: 112, Height: 30}, OnClicked: ui.showSelectedGlobalLookupDetail},
			}},
		},
	}
}

func (ui *mainUI) initializeGlobalLookupPage() {
	state := ui.globalLookup
	if state == nil || state.query == nil {
		return
	}
	state.query.KeyDown().Attach(func(key walk.Key) {
		if key == walk.KeyReturn {
			ui.runGlobalLookup()
		}
	})
	state.query.SetFocus()
	ui.updateGlobalLookupActions()
}

func (ui *mainUI) releaseGlobalLookupPage() {
	if ui.globalLookup != nil && ui.globalLookup.cancel != nil {
		ui.globalLookup.cancel()
	}
	ui.globalLookupTab = nil
	ui.globalLookup = newGlobalLookupUI()
}

func (ui *mainUI) resetGlobalLookup() {
	state := ui.globalLookup
	if state == nil || state.busy {
		return
	}
	state.query.SetText("")
	state.kind.SetCurrentIndex(0)
	state.results = nil
	_ = state.table.SetModel([]globalLookupRow{})
	state.info.SetText("检索条件已清空。")
	state.query.SetFocus()
	ui.updateGlobalLookupActions()
}

func (ui *mainUI) runGlobalLookup() {
	state := ui.globalLookup
	if state == nil || state.query == nil || state.busy {
		return
	}
	query := strings.TrimSpace(state.query.Text())
	if query == "" {
		state.info.SetText("请输入单号、型号或物料名称后再检索。")
		state.query.SetFocus()
		return
	}
	index := state.kind.CurrentIndex()
	if index < 0 || index >= len(state.kindKeys) {
		index = 0
	}
	kind := state.kindKeys[index]
	if state.cancel != nil {
		state.cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	state.cancel = cancel
	state.generation++
	generation := state.generation
	state.busy = true
	state.results = nil
	_ = state.table.SetModel([]globalLookupRow{})
	state.search.SetEnabled(false)
	state.reset.SetEnabled(false)
	state.info.SetText("正在通过现有线上查询接口检索……")
	ui.updateGlobalLookupActions()
	guardedGo(func() {
		results, warnings := searchGlobalLookup(ctx, ui.session.Client, ui.session.Perms, kind, query)
		if ctx.Err() != nil {
			return
		}
		ui.window.Synchronize(func() {
			if generation != state.generation || state != ui.globalLookup || state.table == nil {
				return
			}
			state.busy = false
			state.search.SetEnabled(true)
			state.reset.SetEnabled(true)
			state.results = results
			rows := make([]globalLookupRow, 0, len(results))
			for _, result := range results {
				rows = append(rows, globalLookupRow{Type: lookupKindText(result.Kind), Code: result.Code, Name: result.Name, Status: result.Status, Summary: result.Summary})
			}
			if err := state.table.SetModel(rows); err != nil {
				state.results = nil
				state.info.SetText("检索结果展示失败：" + err.Error())
				return
			}
			addRecentGlobalLookup(state, query)
			message := fmt.Sprintf("在线检索完成：找到 %d 项。", len(results))
			if len(warnings) > 0 {
				message += " 部分查询未完成：" + strings.Join(warnings, "；")
			}
			if len(results) == 0 && len(warnings) == 0 {
				message = "线上未找到匹配结果，请检查检索类型和输入内容。"
			}
			state.info.SetText(message)
			ui.updateGlobalLookupActions()
		})
	})
}

type lookupSearchResult struct {
	results []globalLookupResult
	warning string
}

func searchGlobalLookup(ctx context.Context, client *api.Client, perms api.Perms, kind, query string) ([]globalLookupResult, []string) {
	allowed := map[string]bool{}
	_, keys := availableGlobalLookupKinds(perms)
	for _, key := range keys {
		allowed[key] = true
	}
	targets := []string{kind}
	if kind == "auto" {
		targets = targets[:0]
		for _, candidate := range []string{"material", "inbound", "outbound"} {
			if allowed[candidate] {
				targets = append(targets, candidate)
			}
		}
	}
	channel := make(chan lookupSearchResult, len(targets))
	var wait sync.WaitGroup
	for _, target := range targets {
		if !allowed[target] {
			continue
		}
		wait.Add(1)
		guardedGo1(target, func(target string) {
			defer wait.Done()
			results, err := searchGlobalLookupKind(ctx, client, target, query)
			item := lookupSearchResult{results: results}
			if err != nil {
				item.warning = lookupKindText(target) + "查询失败：" + err.Error()
			}
			channel <- item
		})
	}
	wait.Wait()
	close(channel)
	var results []globalLookupResult
	var warnings []string
	for item := range channel {
		results = append(results, item.results...)
		if item.warning != "" {
			warnings = append(warnings, item.warning)
		}
	}
	sort.SliceStable(results, func(i, j int) bool {
		return lookupResultScore(results[i], query) > lookupResultScore(results[j], query)
	})
	return results, warnings
}

func searchGlobalLookupKind(ctx context.Context, client *api.Client, kind, query string) ([]globalLookupResult, error) {
	switch kind {
	case "material":
		byName, err := client.Materials(ctx, 1, 20, api.MaterialFilters{Name: query})
		if err != nil {
			return nil, err
		}
		byModel, err := client.Materials(ctx, 1, 20, api.MaterialFilters{Model: query})
		if err != nil {
			return nil, err
		}
		seen := map[string]bool{}
		materials := append(byName.List, byModel.List...)
		results := make([]globalLookupResult, 0, len(materials))
		for _, material := range materials {
			if seen[material.ID] {
				continue
			}
			seen[material.ID] = true
			results = append(results, globalLookupResult{
				Kind: "material", Code: displayMaterialValue(material.Model), Name: material.Name,
				Status:   "图纸 " + materialDrawingStatus(material),
				Summary:  fmt.Sprintf("%s / %s / 安全库存 %g %s", displayMaterialValue(material.CategoryName), displayMaterialValue(material.Specification), material.Quantity, material.Unit),
				Material: material,
			})
		}
		return results, nil
	case "inbound":
		receipt, found, err := client.FindInboundReceipt(ctx, "", query)
		if err != nil || !found {
			return nil, err
		}
		business := receipt.SupplierName
		if strings.TrimSpace(business) == "" {
			business = receipt.CustomerName
		}
		return []globalLookupResult{{Kind: "inbound", Code: receipt.Code, Name: business, Status: receipt.Status, Summary: fmt.Sprintf("%s / %d 项物料 / %.2f", receipt.Type, len(receipt.Materials), receipt.TotalAmount), Inbound: receipt}}, nil
	case "outbound":
		order, found, err := client.FindOutbound(ctx, "", query)
		if err != nil || !found {
			return nil, err
		}
		business := order.SupplierName
		if strings.TrimSpace(business) == "" {
			business = order.CustomerName
		}
		return []globalLookupResult{{Kind: "outbound", Code: order.Code, Name: business, Status: order.Status, Summary: fmt.Sprintf("%s / %.2f", order.Type, order.TotalAmount), Outbound: order}}, nil
	default:
		return nil, fmt.Errorf("不支持的检索类型")
	}
}

func lookupResultScore(result globalLookupResult, query string) int {
	query = strings.TrimSpace(query)
	for _, value := range []string{result.Code, result.Name} {
		if strings.EqualFold(strings.TrimSpace(value), query) {
			return 2
		}
	}
	return 1
}

func lookupKindText(kind string) string {
	switch kind {
	case "material":
		return "物料"
	case "inbound":
		return "入库单"
	case "outbound":
		return "出库单"
	default:
		return "自动识别"
	}
}

func addRecentGlobalLookup(state *globalLookupUI, query string) {
	values := []string{query}
	for _, value := range state.recent {
		if !strings.EqualFold(strings.TrimSpace(value), query) {
			values = append(values, value)
		}
		if len(values) >= 10 {
			break
		}
	}
	state.recent = values
	_ = state.recentList.SetModel(append([]string(nil), values...))
}

func (ui *mainUI) useSelectedRecentLookup() {
	state := ui.globalLookup
	if state == nil || state.recentList == nil || state.busy {
		return
	}
	index := state.recentList.CurrentIndex()
	if index >= 0 && index < len(state.recent) {
		state.query.SetText(state.recent[index])
	}
}

func (ui *mainUI) runSelectedRecentLookup() {
	ui.useSelectedRecentLookup()
	ui.runGlobalLookup()
}

func (ui *mainUI) selectedGlobalLookupResult() (globalLookupResult, bool) {
	state := ui.globalLookup
	if state == nil || state.table == nil {
		return globalLookupResult{}, false
	}
	index := state.table.CurrentIndex()
	if index < 0 || index >= len(state.results) {
		return globalLookupResult{}, false
	}
	return state.results[index], true
}

func (ui *mainUI) updateGlobalLookupActions() {
	state := ui.globalLookup
	if state == nil || state.detail == nil {
		return
	}
	_, ok := ui.selectedGlobalLookupResult()
	state.detail.SetEnabled(ok && !state.busy)
	state.locate.SetEnabled(ok && !state.busy)
}

func (ui *mainUI) showSelectedGlobalLookupDetail() {
	result, ok := ui.selectedGlobalLookupResult()
	if !ok {
		return
	}
	switch result.Kind {
	case "material":
		ShowMaterialDetail(ui.window, ui.session.Client, config.ImageBaseURL(), result.Material)
	case "inbound":
		ShowInboundDetail(ui.window, ui.session.Client, config.ImageBaseURL(), result.Inbound)
	case "outbound":
		ui.showOutboundDetailByCode(result.Outbound.Code)
	}
}

func (ui *mainUI) locateSelectedGlobalLookupResult() {
	result, ok := ui.selectedGlobalLookupResult()
	if !ok {
		return
	}
	switch result.Kind {
	case "material":
		ui.openTab("material")
		ui.materialName.SetText(result.Material.Name)
		ui.materialModel.SetText(result.Material.Model)
		ui.materialPage = 1
		ui.loadMaterials()
	case "inbound":
		ui.openTab("inbound")
		ui.inboundSearch.SetText(result.Inbound.Code)
		ui.inboundPage = 1
		ui.loadInbound()
	case "outbound":
		ui.openTab("outbound")
		ui.outbound.search.SetText(result.Outbound.Code)
		ui.outbound.page = 1
		ui.loadOutbound()
	}
}
