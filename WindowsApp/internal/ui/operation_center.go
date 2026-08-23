package ui

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"

	"zhengshi-wms-windowsapp/internal/api"
	"zhengshi-wms-windowsapp/internal/operationstore"
)

type operationCenterRow struct {
	Time    string
	Module  string
	Action  string
	Key     string
	Result  string
	Message string
}

type operationCenterUI struct {
	mu        sync.Mutex
	journalMu sync.Mutex
	events    []api.OperationEvent
	rows      []operationCenterRow
	table     *walk.TableView
	info      *walk.Label
	verify    *walk.PushButton
	openPage  *walk.PushButton
	ack       *walk.PushButton
	busy      bool
	warning   string
}

func newOperationCenterUI() *operationCenterUI { return &operationCenterUI{} }

func (ui *mainUI) operationCenterPageWidget() TabPage {
	if ui.operations == nil {
		ui.operations = newOperationCenterUI()
	}
	state := ui.operations
	return TabPage{
		AssignTo: &ui.operationTab,
		Title:    closableTabTitle("操作复核"),
		Layout:   VBox{Margins: Margins{Left: 16, Top: 16, Right: 16, Bottom: 14}, Spacing: 10},
		Children: []Widget{
			Label{Text: "近期线上操作复核", Font: Font{Family: "Microsoft YaHei UI", PointSize: 15, Bold: true}},
			Label{Text: "自动记录非查询请求；异常退出后仍保留未完成或结果待确认的操作。这里只在线回读，不会重新提交写操作。", TextColor: secondaryTextColor()},
			TableView{
				AssignTo: &state.table, Model: []operationCenterRow{}, AlternatingRowBG: true,
				ColumnsOrderable: true, StretchFactor: 1,
				Accessibility:         Accessibility{Name: "近期线上操作列表", Description: "选择一项后可在线回读，不会重新提交"},
				OnCurrentIndexChanged: ui.updateOperationCenterActions,
				OnItemActivated:       ui.verifySelectedOperation,
				Columns: []TableViewColumn{
					{Title: "时间", DataMember: "Time", Width: 135},
					{Title: "模块", DataMember: "Module", Width: 105},
					{Title: "动作", DataMember: "Action", Width: 120},
					{Title: "业务标识", DataMember: "Key", Width: 160},
					{Title: "请求结果", DataMember: "Result", Width: 100},
					{Title: "说明", DataMember: "Message", Width: 320},
				},
			},
			Label{AssignTo: &state.info, Text: "本次运行尚无线上写操作。", TextColor: secondaryTextColor(), Accessibility: Accessibility{Name: "操作复核状态"}},
			Composite{Layout: HBox{Spacing: 8}, Children: []Widget{
				PushButton{AssignTo: &state.openPage, Text: "打开对应模块", Enabled: false, MinSize: Size{Width: 112, Height: 30}, OnClicked: ui.openSelectedOperationModule},
				PushButton{AssignTo: &state.ack, Text: "标记已人工核对", Enabled: false, MinSize: Size{Width: 126, Height: 30}, OnClicked: ui.acknowledgeSelectedOperation},
				HSpacer{},
				PushButton{Text: "刷新记录", MinSize: Size{Width: 92, Height: 30}, OnClicked: ui.refreshOperationCenter},
				PushButton{AssignTo: &state.verify, Text: "在线回读所选项", Enabled: false, MinSize: Size{Width: 126, Height: 30}, OnClicked: ui.verifySelectedOperation},
			}},
		},
	}
}

func (ui *mainUI) loadOperationJournal() {
	state := ui.operations
	if state == nil || ui.session == nil {
		return
	}
	events, err := operationstore.Load(ui.cfg.APIBaseURL, ui.session.Login.Mobile)
	state.mu.Lock()
	defer state.mu.Unlock()
	if err != nil {
		state.warning = "未能读取加密操作复核日志：" + err.Error()
		return
	}
	state.events = events
}

func (ui *mainUI) showRestoredOperationSummary() {
	state := ui.operations
	if state == nil || ui.status == nil {
		return
	}
	state.mu.Lock()
	count := 0
	for _, event := range state.events {
		if event.Outcome == api.OperationPending || event.Outcome == api.OperationUnknown {
			count++
		}
	}
	warning := state.warning
	state.mu.Unlock()
	if warning != "" {
		ui.status.SetText(warning + "；请打开“操作复核”查看本次运行记录。")
		ui.status.SetTextColor(dangerTextColor())
		return
	}
	if count > 0 {
		ui.status.SetText(fmt.Sprintf("已从加密日志恢复 %d 项待确认操作；请打开“操作复核”在线回读，客户端不会自动重提。", count))
		ui.status.SetTextColor(warningTextColor())
	}
}

func (ui *mainUI) recordOperation(event api.OperationEvent) {
	state := ui.operations
	if state == nil {
		return
	}
	state.mu.Lock()
	found := false
	for index := range state.events {
		if state.events[index].ID == event.ID {
			state.events[index] = event
			found = true
			break
		}
	}
	if !found {
		state.events = append([]api.OperationEvent{event}, state.events...)
	}
	if len(state.events) > 100 {
		state.events = state.events[:100]
	}
	state.mu.Unlock()
	ui.persistOperationJournal()
	if ui.window != nil {
		ui.window.Synchronize(func() {
			ui.refreshOperationCenter()
			ui.showOperationStatus(event)
		})
	}
}

func (ui *mainUI) persistOperationJournal() {
	state := ui.operations
	if state == nil || ui.session == nil {
		return
	}
	state.journalMu.Lock()
	defer state.journalMu.Unlock()
	state.mu.Lock()
	events := append([]api.OperationEvent(nil), state.events...)
	state.mu.Unlock()
	if err := operationstore.Save(ui.cfg.APIBaseURL, ui.session.Login.Mobile, events); err != nil {
		state.mu.Lock()
		state.warning = "加密操作复核日志保存失败：" + err.Error()
		state.mu.Unlock()
	} else {
		state.mu.Lock()
		state.warning = ""
		state.mu.Unlock()
	}
}

func (ui *mainUI) showOperationStatus(event api.OperationEvent) {
	if ui.status == nil {
		return
	}
	module := operationModule(event.Path)
	action := operationAction(event.Method, event.Path)
	switch event.Outcome {
	case api.OperationPending:
		ui.status.SetText(module + " · " + action + "正在提交，请勿重复操作…")
		ui.status.SetTextColor(infoTextColor())
	case api.OperationSucceeded:
		ui.status.SetText(module + " · " + action + "已由服务端确认成功。")
		ui.status.SetTextColor(successTextColor())
	case api.OperationUnknown:
		ui.status.SetText(module + " · " + action + "结果待确认；请打开“操作复核”在线回读，客户端不会自动重提。")
		ui.status.SetTextColor(warningTextColor())
	default:
		ui.status.SetText(module + " · " + action + "未被服务端接受：" + event.Message)
		ui.status.SetTextColor(dangerTextColor())
	}
}

func (ui *mainUI) refreshOperationCenter() {
	state := ui.operations
	if state == nil || state.table == nil {
		return
	}
	state.mu.Lock()
	events := append([]api.OperationEvent(nil), state.events...)
	warning := state.warning
	state.mu.Unlock()
	rows := make([]operationCenterRow, 0, len(events))
	for _, event := range events {
		displayTime := event.FinishedAt
		if displayTime.IsZero() {
			displayTime = event.StartedAt
		}
		rows = append(rows, operationCenterRow{
			Time:   displayTime.Local().Format("2006-01-02 15:04:05"),
			Module: operationModule(event.Path), Action: operationAction(event.Method, event.Path),
			Key: displayMaterialValue(event.BusinessKey), Result: operationOutcomeText(event.Outcome), Message: event.Message,
		})
	}
	state.rows = rows
	_ = state.table.SetModel(rows)
	if warning != "" {
		state.info.SetText(warning + "；本次运行仍会继续显示操作，但异常退出后可能无法恢复。")
	} else if len(rows) == 0 {
		state.info.SetText("本次运行尚无线上写操作。")
	} else if !state.busy {
		state.info.SetText(fmt.Sprintf("已记录 %d 项操作；未完成和结果待确认项会加密保留 7 天，最多 100 项。", len(rows)))
	}
	ui.updateOperationCenterActions()
}

func (ui *mainUI) selectedOperation() (api.OperationEvent, bool) {
	state := ui.operations
	if state == nil || state.table == nil {
		return api.OperationEvent{}, false
	}
	index := state.table.CurrentIndex()
	state.mu.Lock()
	defer state.mu.Unlock()
	if index < 0 || index >= len(state.events) {
		return api.OperationEvent{}, false
	}
	return state.events[index], true
}

func (ui *mainUI) updateOperationCenterActions() {
	state := ui.operations
	if state == nil || state.verify == nil || state.ack == nil {
		return
	}
	event, ok := ui.selectedOperation()
	state.verify.SetEnabled(ok && !state.busy)
	state.openPage.SetEnabled(ok && !state.busy)
	state.ack.SetEnabled(ok && !state.busy && (event.Outcome == api.OperationPending || event.Outcome == api.OperationUnknown))
}

func (ui *mainUI) acknowledgeSelectedOperation() {
	event, ok := ui.selectedOperation()
	state := ui.operations
	if !ok || state == nil || state.busy {
		return
	}
	if walk.MsgBox(ui.window, "确认已人工核对", "该操作将从本机待确认日志中移除，但不会向后端发送任何请求。请确认已经通过业务页面核对线上状态。", walk.MsgBoxYesNo|walk.MsgBoxIconWarning) != walk.DlgCmdYes {
		return
	}
	state.mu.Lock()
	for index := range state.events {
		if state.events[index].ID == event.ID {
			state.events = append(state.events[:index], state.events[index+1:]...)
			break
		}
	}
	state.mu.Unlock()
	ui.persistOperationJournal()
	ui.refreshOperationCenter()
}

func (ui *mainUI) verifySelectedOperation() {
	event, ok := ui.selectedOperation()
	state := ui.operations
	if !ok || state == nil || state.busy {
		return
	}
	state.busy = true
	state.info.SetText("正在通过现有查询接口回读线上状态……")
	ui.updateOperationCenterActions()
	guardedGo(func() {
		result, err := verifyOperationOnline(context.Background(), ui.session.Client, event)
		ui.window.Synchronize(func() {
			state.busy = false
			ui.updateOperationCenterActions()
			if err != nil {
				state.info.SetText("在线回读失败：" + err.Error() + "。未重新提交原操作。")
				return
			}
			state.info.SetText(result)
		})
	})
}

func verifyOperationOnline(ctx context.Context, client *api.Client, event api.OperationEvent) (string, error) {
	key := strings.TrimSpace(event.BusinessKey)
	switch {
	case event.Path == "/account/avatar":
		profile, err := client.Profile(ctx)
		if err != nil {
			return "", err
		}
		return "回读成功：当前头像地址为 " + displayMaterialValue(profile.Avatar) + "。", nil
	case strings.HasPrefix(event.Path, "/inbound/receipt") && key != "":
		id, code := "", ""
		if event.KeyField == "id" {
			id = key
		} else {
			code = key
		}
		receipt, found, err := client.FindInboundReceipt(ctx, id, code)
		if err != nil {
			return "", err
		}
		if !found {
			if event.Method == "DELETE" {
				return "回读完成：查询结果中未找到该入库单，与删除动作的预期一致。", nil
			}
			return "回读完成：当前查询窗口未找到该入库单，无法据此确认原操作结果；请打开入库工作台按单号复核。", nil
		}
		return fmt.Sprintf("回读成功：入库单 %s 当前状态为“%s”。", receipt.Code, receipt.Status), nil
	case strings.HasPrefix(event.Path, "/outbound") && key != "":
		id, code := "", ""
		if event.KeyField == "id" {
			id = key
		} else {
			code = key
		}
		order, found, err := client.FindOutbound(ctx, id, code)
		if err != nil {
			return "", err
		}
		if !found {
			if event.Method == "DELETE" {
				return "回读完成：查询结果中未找到该出库单，与删除动作的预期一致。", nil
			}
			return "回读完成：当前查询窗口未找到该出库单，无法据此确认原操作结果；请打开出库执行按单号复核。", nil
		}
		return fmt.Sprintf("回读成功：出库单 %s 当前状态为“%s”。", order.Code, order.Status), nil
	case event.Path == "/material" && key != "" && event.KeyField == "id":
		material, err := client.MaterialInfo(ctx, key)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("回读成功：物料 %s / %s 已在线读取。", material.Name, material.Model), nil
	case event.Path == "/material" && key != "":
		page, err := client.Materials(ctx, 1, 20, api.MaterialFilters{Name: key})
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("回读成功：按名称“%s”查询到 %d 条物料。", key, page.Total), nil
	default:
		return "该操作已记录，但现有查询接口没有稳定的通用业务键；请打开对应模块并按 F5 刷新核对。未重新提交原操作。", nil
	}
}

func (ui *mainUI) openSelectedOperationModule() {
	event, ok := ui.selectedOperation()
	if !ok {
		return
	}
	key := operationModuleKey(event.Path)
	if key == "" {
		ui.operations.info.SetText("当前操作没有可直接定位的业务模块，请使用左侧菜单手动打开。")
		return
	}
	ui.openTab(key)
}

func operationModule(path string) string {
	switch operationModuleKey(path) {
	case "material", "material_quote":
		return "物料"
	case "inbound":
		return "入库"
	case "outbound":
		return "出库"
	case "partner":
		return "合作伙伴"
	case "warehouse":
		return "仓储结构"
	case "admin":
		return "系统管理"
	case "profile":
		return "个人中心"
	case "image_assets":
		return "图片素材"
	default:
		return "其他"
	}
}

func operationModuleKey(path string) string {
	switch {
	case strings.HasPrefix(path, "/material/quote"):
		return "material_quote"
	case strings.HasPrefix(path, "/material"):
		return "material"
	case strings.HasPrefix(path, "/inbound"):
		return "inbound"
	case strings.HasPrefix(path, "/outbound"):
		return "outbound"
	case strings.HasPrefix(path, "/supplier"), strings.HasPrefix(path, "/customer"), strings.HasPrefix(path, "/carrier"):
		return "partner"
	case strings.HasPrefix(path, "/warehouse"):
		return "warehouse"
	case strings.HasPrefix(path, "/user"), strings.HasPrefix(path, "/department"), strings.HasPrefix(path, "/role"):
		return "admin"
	case strings.HasPrefix(path, "/account"):
		return "profile"
	case strings.HasPrefix(path, "/images"):
		return "image_assets"
	default:
		return ""
	}
}

func operationAction(method, path string) string {
	switch {
	case path == "/account/avatar":
		return "更新头像"
	case strings.Contains(path, "/check"):
		return "审核"
	case strings.Contains(path, "/receive"):
		return "签收/收货"
	case path == "/outbound/receipt":
		return "确认签收"
	case strings.Contains(path, "/confirm"):
		return "确认"
	case strings.Contains(path, "/pick"):
		return "拣货"
	case strings.Contains(path, "/pack"):
		return "打包"
	case strings.Contains(path, "/weigh"):
		return "称重"
	case strings.Contains(path, "/departure"):
		return "出库"
	case strings.Contains(path, "/status"):
		return "变更状态"
	case method == "POST":
		return "新增/提交"
	case method == "PUT":
		return "编辑"
	case method == "PATCH":
		return "变更"
	case method == "DELETE":
		return "删除"
	default:
		return method
	}
}

func operationOutcomeText(outcome string) string {
	switch outcome {
	case api.OperationPending:
		return "正在提交"
	case api.OperationSucceeded:
		return "服务端成功"
	case api.OperationUnknown:
		return "结果待确认"
	default:
		return "服务端拒绝"
	}
}
