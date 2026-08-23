package ui

import (
	"context"
	"crypto/rand"
	"encoding/hex"
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

type customerFinanceUI struct {
	customer      partnerDetail
	table         *walk.TableView
	balance       *walk.Label
	info          *walk.Label
	add           *walk.PushButton
	refresh       *walk.PushButton
	attachment    *walk.PushButton
	prev          *walk.PushButton
	next          *walk.PushButton
	size          *walk.ComboBox
	rows          []customerTransactionRow
	page          int
	total         int64
	generation    int
	cancel        context.CancelFunc
	balanceCancel context.CancelFunc
	busy          bool
	closed        atomic.Bool
}

func newCustomerFinanceUI(customer partnerDetail) *customerFinanceUI {
	return &customerFinanceUI{customer: customer, page: 1}
}

func (state *customerFinanceUI) dispose() {
	if state == nil || state.closed.Swap(true) {
		return
	}
	state.generation++
	if state.cancel != nil {
		state.cancel()
	}
	if state.balanceCancel != nil {
		state.balanceCancel()
	}
}

func (ui *mainUI) openSelectedCustomerFinance() {
	detail, kind, ok := ui.selectedPartner()
	if !ok || kind.Key != "customer" {
		walk.MsgBox(ui.window, "请选择客户", "请先切换到客户列表并选择一位客户。", walk.MsgBoxIconInformation)
		return
	}
	ui.openCustomerFinance(detail)
}

func (ui *mainUI) openCustomerFinance(customer partnerDetail) {
	if strings.TrimSpace(customer.ID) == "" {
		walk.MsgBox(ui.window, "客户信息不完整", "当前客户缺少有效 ID，无法查询交易流水。", walk.MsgBoxIconWarning)
		return
	}
	if ui.customerFinance != nil && ui.customerFinance.customer.ID == customer.ID && ui.customerFinanceTab != nil {
		if ui.tabs.Pages().Index(ui.customerFinanceTab) < 0 {
			insertAt := ui.dynamicTabInsertionIndex()
			if err := ui.tabs.Pages().Insert(insertAt, ui.customerFinanceTab); err != nil {
				walk.MsgBox(ui.window, "无法恢复客户应收页面", err.Error(), walk.MsgBoxIconError)
				return
			}
		}
		_ = ui.tabs.SetCurrentIndex(ui.tabs.Pages().Index(ui.customerFinanceTab))
		return
	}
	ui.closeCustomerFinanceTab()
	state := newCustomerFinanceUI(customer)
	ui.customerFinance = state
	pageDecl := ui.customerFinancePageWidget(state)
	if err := pageDecl.Create(NewBuilder(nil)); err != nil {
		state.dispose()
		ui.customerFinance = nil
		walk.MsgBox(ui.window, "无法打开客户应收页面", err.Error(), walk.MsgBoxIconError)
		return
	}
	if err := ui.tabs.Pages().Insert(ui.dynamicTabInsertionIndex(), ui.customerFinanceTab); err != nil {
		state.dispose()
		ui.customerFinanceTab.Dispose()
		ui.customerFinanceTab = nil
		ui.customerFinance = nil
		walk.MsgBox(ui.window, "无法打开客户应收页面", err.Error(), walk.MsgBoxIconError)
		return
	}
	_ = ui.tabs.SetCurrentIndex(ui.tabs.Pages().Index(ui.customerFinanceTab))
	ui.initializeCustomerFinancePage()
}

func (ui *mainUI) dynamicTabInsertionIndex() int {
	insertAt := ui.tabs.Pages().Len()
	if ui.systemTab != nil {
		if index := ui.tabs.Pages().Index(ui.systemTab); index >= 0 {
			insertAt = index
		}
	}
	return insertAt
}

func (ui *mainUI) closeCustomerFinanceTab() {
	if ui.customerFinance != nil {
		ui.customerFinance.dispose()
	}
	if ui.customerFinanceTab != nil {
		if ui.tabs != nil {
			if index := ui.tabs.Pages().Index(ui.customerFinanceTab); index >= 0 {
				_ = ui.tabs.Pages().RemoveAt(index)
			}
		}
		ui.customerFinanceTab.Dispose()
	}
	ui.customerFinanceTab = nil
	ui.customerFinance = nil
	ui.syncNavigationFromTab()
}

func (ui *mainUI) customerFinancePageWidget(state *customerFinanceUI) TabPage {
	return TabPage{
		AssignTo: &ui.customerFinanceTab,
		Title:    closableTabTitle("客户应收 · " + state.customer.Name),
		Layout:   VBox{Margins: Margins{Left: 14, Top: 14, Right: 14, Bottom: 12}, Spacing: 10},
		Children: []Widget{
			Label{Text: "客户应收与交易流水", Font: Font{Family: "Microsoft YaHei UI", PointSize: 15, Bold: true}},
			Label{
				Text:          "人工入账仅支持回款和退货冲减；交易方向和余额变化由服务端确定，写请求不会自动重试。",
				TextColor:     secondaryTextColor(),
				Accessibility: Accessibility{Name: "客户应收页面说明"},
			},
			GroupBox{
				Title:  "客户摘要",
				Layout: Grid{Columns: 8, Margins: Margins{Left: 14, Top: 10, Right: 14, Bottom: 12}, Spacing: 8},
				Children: []Widget{
					Label{Text: "客户名称"}, Label{Text: displayMaterialValue(state.customer.Name), Font: Font{Bold: true}},
					Label{Text: "客户编号"}, Label{Text: displayMaterialValue(state.customer.Code)},
					Label{Text: "客户状态"}, Label{Text: displayMaterialValue(state.customer.Status)},
					Label{Text: "当前应收"}, Label{AssignTo: &state.balance, Text: fmt.Sprintf("%.2f 元", state.customer.ReceivableValue), Font: Font{Bold: true}, Accessibility: Accessibility{Name: "客户当前应收余额"}},
				},
			},
			Composite{Layout: HBox{Spacing: 8}, Children: []Widget{
				PushButton{AssignTo: &state.add, Text: "新增交易", MinSize: Size{Width: 92, Height: 30}, Accessibility: Accessibility{Name: "新增客户回款或退货冲减"}, OnClicked: ui.addCustomerFinanceTransaction},
				PushButton{AssignTo: &state.attachment, Text: "查看附件", Enabled: false, MinSize: Size{Width: 92, Height: 30}, OnClicked: ui.previewCustomerFinanceAttachment},
				HSpacer{},
				PushButton{AssignTo: &state.refresh, Text: "刷新流水", MinSize: Size{Width: 92, Height: 30}, OnClicked: ui.loadCustomerFinanceTransactions},
			}},
			TableView{
				AssignTo: &state.table, Model: []customerTransactionRow{}, AlternatingRowBG: true,
				ColumnsOrderable: true, StretchFactor: 1,
				Accessibility:         Accessibility{Name: "客户交易流水列表", Description: "选择流水后可查看图片附件"},
				OnCurrentIndexChanged: ui.updateCustomerFinanceActions,
				Columns: []TableViewColumn{
					{Title: "交易类型", DataMember: "Type", Width: 115},
					{Title: "方向", DataMember: "Direction", Width: 90},
					{Title: "状态", DataMember: "Status", Width: 85},
					{Title: "来源单据", DataMember: "Source", Width: 150},
					{Title: "交易时间", DataMember: "Time", Width: 135},
					{Title: "金额", DataMember: "Amount", Width: 105},
					{Title: "附件", DataMember: "Attachments", Width: 60},
					{Title: "备注", DataMember: "Remark", Width: 260},
				},
			},
			Composite{Layout: HBox{Spacing: 8}, Children: []Widget{
				Label{AssignTo: &state.info, Text: "尚未加载", Accessibility: Accessibility{Name: "客户交易流水状态"}},
				HSpacer{},
				Label{Text: "每页"},
				ComboBox{
					AssignTo: &state.size, Model: pageSizeLabels, CurrentIndex: 1, MinSize: Size{Width: 92},
					OnCurrentIndexChanged: func() {
						if ui.window != nil && state.size != nil && state.size.CurrentIndex() >= 0 {
							state.page = 1
							ui.loadCustomerFinanceTransactions()
						}
					},
				},
				PushButton{AssignTo: &state.prev, Text: "上一页", OnClicked: func() {
					if state.page > 1 {
						state.page--
						ui.loadCustomerFinanceTransactions()
					}
				}},
				PushButton{AssignTo: &state.next, Text: "下一页", OnClicked: func() {
					state.page++
					ui.loadCustomerFinanceTransactions()
				}},
			}},
		},
	}
}

func (ui *mainUI) initializeCustomerFinancePage() {
	state := ui.customerFinance
	if state == nil || state.table == nil {
		return
	}
	state.generation++
	ui.updateCustomerFinanceActions()
	ui.loadCustomerFinanceTransactions()
}

func (ui *mainUI) loadCustomerFinanceTransactions() {
	state := ui.customerFinance
	if state == nil || state.table == nil || state.closed.Load() {
		return
	}
	if state.cancel != nil {
		state.cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	state.cancel = cancel
	state.generation++
	generation := state.generation
	page := state.page
	size := selectedPageSize(state.size)
	state.busy = true
	state.info.SetText("正在加载客户交易流水……")
	state.refresh.SetEnabled(false)
	state.add.SetEnabled(false)
	state.prev.SetEnabled(false)
	state.next.SetEnabled(false)
	ui.updateCustomerFinanceActions()

	guardedGo(func() {
		result, requestErr := ui.session.Client.CustomerTransactions(ctx, state.customer.ID, page, size)
		if ctx.Err() != nil || state.closed.Load() {
			return
		}
		ui.window.Synchronize(func() {
			if state != ui.customerFinance || generation != state.generation || state.table == nil {
				return
			}
			state.busy = false
			state.refresh.SetEnabled(true)
			state.add.SetEnabled(true)
			if requestErr != nil {
				state.info.SetText("流水加载失败：" + requestFailureText(requestErr))
				ui.updateCustomerFinanceActions()
				return
			}
			state.rows = customerTransactionRows(result.List)
			state.total = result.Total
			if modelErr := state.table.SetModel(state.rows); modelErr != nil {
				state.info.SetText("流水列表展示失败：" + modelErr.Error())
				return
			}
			state.info.SetText(fmt.Sprintf("第 %d 页 | 本页 %d 条 | 共 %d 条", page, len(state.rows), result.Total))
			state.prev.SetEnabled(page > 1)
			state.next.SetEnabled(int64(page*size) < result.Total)
			ui.updateCustomerFinanceActions()
		})
	})
}

func (ui *mainUI) updateCustomerFinanceActions() {
	state := ui.customerFinance
	if state == nil || state.attachment == nil || state.table == nil {
		return
	}
	index := state.table.CurrentIndex()
	count := 0
	if index >= 0 && index < len(state.rows) {
		count = len(customerTransactionAttachments(state.rows[index].Detail.Annex))
	}
	state.attachment.SetEnabled(count > 0 && !state.busy)
	if count > 0 {
		state.attachment.SetText(fmt.Sprintf("查看附件 (%d)", count))
	} else {
		state.attachment.SetText("查看附件")
	}
}

func (ui *mainUI) previewCustomerFinanceAttachment() {
	state := ui.customerFinance
	if state == nil || state.table == nil {
		return
	}
	index := state.table.CurrentIndex()
	if index < 0 || index >= len(state.rows) {
		return
	}
	ShowOrderAttachments(ui.window, ui.session.Client, config.ImageBaseURL(), "客户流水 "+state.rows[index].Source, customerTransactionAttachments(state.rows[index].Detail.Annex))
}

type customerTransactionDialogState struct {
	dlg              *walk.Dialog
	transactionType  *walk.ComboBox
	date             *walk.DateEdit
	amount           *walk.LineEdit
	remark           *walk.LineEdit
	attachment       *walk.ComboBox
	selectAttachment *walk.PushButton
	uploadAttachment *walk.PushButton
	preview          *walk.PushButton
	removeAttachment *walk.PushButton
	info             *walk.Label
	submit           *walk.PushButton
	cancelButton     *walk.PushButton
	annex            []string
	idempotencyKey   string
	busy             bool
	closed           atomic.Bool
	ctx              context.Context
	cancel           context.CancelFunc
	uploadCancel     context.CancelFunc
}

func (ui *mainUI) addCustomerFinanceTransaction() {
	finance := ui.customerFinance
	if finance == nil || finance.busy {
		return
	}
	dialogCtx, dialogCancel := context.WithCancel(context.Background())
	state := &customerTransactionDialogState{
		idempotencyKey: newManualTransactionIdempotencyKey(finance.customer.ID),
		ctx:            dialogCtx, cancel: dialogCancel,
	}
	err := Dialog{
		AssignTo: &state.dlg, Title: "新增客户交易 - " + finance.customer.Name,
		DefaultButton: &state.submit, CancelButton: &state.cancelButton,
		MinSize: Size{Width: 680, Height: 520}, Size: Size{Width: 760, Height: 600},
		Layout: VBox{Margins: Margins{Left: 18, Top: 18, Right: 18, Bottom: 16}, Spacing: 10},
		Children: []Widget{
			Label{Text: "新增客户交易", Font: Font{Family: "Microsoft YaHei UI", PointSize: 15, Bold: true}},
			Label{Text: "仅支持回款和退货冲减；两者都会减少应收余额，最终结果由服务端事务处理。", TextColor: secondaryTextColor()},
			GroupBox{Title: "交易信息", Layout: Grid{Columns: 4, Margins: Margins{Left: 14, Top: 12, Right: 14, Bottom: 14}, Spacing: 10}, Children: []Widget{
				Label{Text: "客户"}, Label{Text: finance.customer.Name, Font: Font{Bold: true}},
				Label{Text: "当前应收"}, Label{Text: fmt.Sprintf("%.2f 元", finance.customer.ReceivableValue), Font: Font{Bold: true}},
				Label{Text: "交易类型 *"}, ComboBox{AssignTo: &state.transactionType, Model: []string{"回款", "退货冲减"}, CurrentIndex: 0, MinSize: Size{Width: 220, Height: 28}, Accessibility: Accessibility{Name: "客户交易类型"}},
				Label{Text: "交易日期 *"}, DateEdit{AssignTo: &state.date, Format: "yyyy-MM-dd", MinSize: Size{Width: 220, Height: 28}, Accessibility: Accessibility{Name: "客户交易日期"}},
				Label{Text: "交易金额 *"}, LineEdit{AssignTo: &state.amount, CueBanner: "大于 0 的数字", MinSize: Size{Width: 220, Height: 28}, Accessibility: Accessibility{Name: "客户交易金额"}},
				Label{Text: "备注 *"}, LineEdit{AssignTo: &state.remark, ColumnSpan: 3, CueBanner: "填写回款或退货冲减说明", Accessibility: Accessibility{Name: "客户交易备注"}},
			}},
			GroupBox{Title: "图片附件（最多 10 张）", Layout: VBox{Margins: Margins{Left: 14, Top: 10, Right: 14, Bottom: 12}, Spacing: 8}, Children: []Widget{
				Composite{Layout: HBox{Spacing: 8}, Children: []Widget{
					ComboBox{AssignTo: &state.attachment, Model: []string{"暂无附件"}, CurrentIndex: 0, MinSize: Size{Width: 220, Height: 28}, Accessibility: Accessibility{Name: "客户交易附件列表"}},
					PushButton{AssignTo: &state.selectAttachment, Text: "选择素材", MinSize: Size{Width: 92, Height: 30}},
					PushButton{AssignTo: &state.uploadAttachment, Text: "上传图片", MinSize: Size{Width: 92, Height: 30}, ToolTipText: "上传期间再次点击可取消", Accessibility: Accessibility{Name: "上传客户交易图片附件"}},
					PushButton{AssignTo: &state.preview, Text: "预览", Enabled: false, MinSize: Size{Width: 72, Height: 30}},
					PushButton{AssignTo: &state.removeAttachment, Text: "移除", Enabled: false, MinSize: Size{Width: 72, Height: 30}},
					HSpacer{},
				}},
				Label{Text: "已上传图片不会因从本次表单移除而从服务端素材库删除。", TextColor: secondaryTextColor()},
			}},
			VSpacer{},
			Label{AssignTo: &state.info, Text: "请填写必填项；提交后将真实改变客户应收余额。", TextColor: secondaryTextColor(), Accessibility: Accessibility{Name: "客户交易提交状态"}},
			Composite{Layout: HBox{Spacing: 8}, Children: []Widget{
				Label{Text: "请求超时后请先刷新流水；再次提交会沿用同一幂等键。", TextColor: secondaryTextColor()},
				HSpacer{},
				PushButton{AssignTo: &state.cancelButton, Text: "取消", MinSize: Size{Width: 82, Height: 30}, OnClicked: func() { state.dlg.Cancel() }},
				PushButton{AssignTo: &state.submit, Text: "核对并提交", MinSize: Size{Width: 112, Height: 30}},
			}},
		},
	}.Create(ui.window)
	if err != nil {
		walk.MsgBox(ui.window, "交易窗口错误", err.Error(), walk.MsgBoxIconError)
		return
	}
	_ = state.date.SetDate(time.Now())
	state.dlg.Disposing().Attach(func() {
		state.closed.Store(true)
		state.cancel()
	})
	refreshTransactionDialogAttachments(state)

	state.selectAttachment.Clicked().Attach(func() {
		remaining := 10 - len(state.annex)
		if remaining <= 0 {
			state.info.SetText("最多只能添加 10 张附件。")
			return
		}
		references, ok := SelectImageAssets(state.dlg, ui.session.Client, config.ImageBaseURL(), remaining)
		if !ok {
			return
		}
		state.annex = appendUniqueReferences(state.annex, references, 10)
		refreshTransactionDialogAttachments(state)
		state.info.SetText(fmt.Sprintf("已选择 %d 张附件；提交交易后生效。", len(state.annex)))
	})
	state.uploadAttachment.Clicked().Attach(func() {
		if state.uploadCancel != nil {
			state.uploadCancel()
			state.uploadAttachment.SetText("正在取消")
			state.uploadAttachment.SetEnabled(false)
			state.info.SetText("正在取消图片附件上传……")
			return
		}
		if len(state.annex) >= 10 {
			state.info.SetText("最多只能添加 10 张附件。")
			return
		}
		if state.busy {
			return
		}
		dialog := newImageOpenDialog()
		accepted, chooseErr := dialog.ShowOpen(state.dlg)
		if chooseErr != nil {
			state.info.SetText("选择图片失败：" + chooseErr.Error())
			return
		}
		if !accepted {
			return
		}
		uploadCtx, uploadCancel := context.WithCancel(state.ctx)
		state.uploadCancel = uploadCancel
		setCustomerTransactionDialogEnabled(state, false)
		state.uploadAttachment.SetText("取消上传")
		state.uploadAttachment.SetEnabled(true)
		state.info.SetText("正在上传图片附件……")
		guardedGo1(dialog.FilePath, func(filePath string) {
			defer uploadCancel()
			reference, uploadErr := ui.session.Client.UploadImage(uploadCtx, filePath)
			if state.closed.Load() {
				return
			}
			canceled := uploadCtx.Err() != nil
			state.dlg.Synchronize(func() {
				if state.closed.Load() {
					return
				}
				state.uploadCancel = nil
				setCustomerTransactionDialogEnabled(state, true)
				state.uploadAttachment.SetText("上传图片")
				if canceled {
					state.info.SetText("附件上传已取消，当前交易未加入新附件。")
					return
				}
				if uploadErr != nil {
					state.info.SetText("附件上传失败：" + requestFailureText(uploadErr))
					return
				}
				state.annex = appendUniqueReferences(state.annex, []string{reference}, 10)
				refreshTransactionDialogAttachments(state)
				state.info.SetText("图片已上传并加入当前交易表单。")
			})
		})
	})
	state.preview.Clicked().Attach(func() {
		ShowOrderAttachments(state.dlg, ui.session.Client, config.ImageBaseURL(), "客户交易附件", state.annex)
	})
	state.removeAttachment.Clicked().Attach(func() {
		index := state.attachment.CurrentIndex()
		if index < 0 || index >= len(state.annex) || state.busy {
			return
		}
		state.annex = append(state.annex[:index], state.annex[index+1:]...)
		refreshTransactionDialogAttachments(state)
		state.info.SetText("附件已从当前交易表单移除。")
	})
	state.submit.Clicked().Attach(func() { ui.submitCustomerFinanceTransaction(state, finance) })
	if state.dlg.Run() == walk.DlgCmdOK {
		ui.loadCustomerFinanceTransactions()
		if ui.partnerTab != nil {
			ui.loadPartners()
		}
		ui.refreshCustomerFinanceBalance()
	}
}

func setCustomerTransactionDialogEnabled(state *customerTransactionDialogState, enabled bool) {
	state.busy = !enabled
	for _, widget := range []interface{ SetEnabled(bool) }{
		state.transactionType, state.date, state.amount, state.remark, state.attachment,
		state.selectAttachment, state.uploadAttachment, state.cancelButton, state.submit,
	} {
		widget.SetEnabled(enabled)
	}
	state.preview.SetEnabled(enabled && len(state.annex) > 0)
	state.removeAttachment.SetEnabled(enabled && len(state.annex) > 0)
}

func refreshTransactionDialogAttachments(state *customerTransactionDialogState) {
	labels := []string{"暂无附件"}
	if len(state.annex) > 0 {
		labels = make([]string, len(state.annex))
		for index := range state.annex {
			labels[index] = fmt.Sprintf("附件 %d / %d", index+1, len(state.annex))
		}
	}
	_ = state.attachment.SetModel(labels)
	_ = state.attachment.SetCurrentIndex(0)
	hasAttachments := len(state.annex) > 0 && !state.busy
	state.preview.SetEnabled(hasAttachments)
	state.removeAttachment.SetEnabled(hasAttachments)
}

func appendUniqueReferences(existing, additions []string, limit int) []string {
	seen := make(map[string]struct{}, len(existing)+len(additions))
	result := make([]string, 0, minInt(limit, len(existing)+len(additions)))
	for _, group := range [][]string{existing, additions} {
		for _, reference := range group {
			reference = strings.TrimSpace(reference)
			if reference == "" {
				continue
			}
			if _, ok := seen[reference]; ok {
				continue
			}
			seen[reference] = struct{}{}
			result = append(result, reference)
			if len(result) >= limit {
				return result
			}
		}
	}
	return result
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (ui *mainUI) submitCustomerFinanceTransaction(state *customerTransactionDialogState, finance *customerFinanceUI) {
	if state == nil || state.busy || finance == nil {
		return
	}
	typeCode := api.CustomerTransactionPayment
	if state.transactionType.CurrentIndex() == 1 {
		typeCode = api.CustomerTransactionReturnCredit
	}
	transactionDate := state.date.Date()
	if transactionDate.IsZero() {
		state.info.SetText("请选择交易日期。")
		_ = state.date.SetFocus()
		return
	}
	transactionTime := time.Date(transactionDate.Year(), transactionDate.Month(), transactionDate.Day(), 0, 0, 0, 0, time.Local)
	if transactionTime.After(time.Now()) {
		state.info.SetText("交易日期不能晚于今天。")
		_ = state.date.SetFocus()
		return
	}
	amount, err := strconv.ParseFloat(strings.TrimSpace(state.amount.Text()), 64)
	if err != nil || amount <= 0 {
		state.info.SetText("交易金额必须是大于 0 的数字。")
		_ = state.amount.SetFocus()
		return
	}
	remark := strings.TrimSpace(state.remark.Text())
	if remark == "" {
		state.info.SetText("请填写交易备注，说明回款或退货冲减原因。")
		_ = state.remark.SetFocus()
		return
	}
	request := api.CustomerTransactionAddRequest{
		CustomerID: finance.customer.ID, Time: transactionTime.Unix(), TransactionType: typeCode,
		IdempotencyKey: state.idempotencyKey, Amount: amount, Annex: append([]string(nil), state.annex...), Remark: remark,
	}
	if walk.MsgBox(state.dlg, "确认客户交易", fmt.Sprintf("客户：%s\r\n类型：%s\r\n金额：%.2f 元\r\n日期：%s\r\n附件：%d 张\r\n\r\n提交后将真实改变客户应收余额，是否继续？", finance.customer.Name, state.transactionType.Text(), amount, transactionTime.Format("2006-01-02"), len(state.annex)), walk.MsgBoxYesNo|walk.MsgBoxIconWarning) != walk.DlgCmdYes {
		return
	}
	setCustomerTransactionDialogEnabled(state, false)
	state.submit.SetText("正在提交……")
	state.info.SetText("正在提交线上交易；请求不会自动重试……")
	guardedGo(func() {
		requestErr := ui.session.Client.AddCustomerTransaction(state.ctx, request)
		if state.closed.Load() {
			return
		}
		state.dlg.Synchronize(func() {
			if state.closed.Load() {
				return
			}
			state.submit.SetText("核对并提交")
			setCustomerTransactionDialogEnabled(state, true)
			if requestErr != nil {
				state.info.SetText("交易提交未确认：" + requestFailureText(requestErr) + "。请先刷新流水；再次提交将沿用原幂等键。")
				return
			}
			state.info.SetText("交易提交成功，正在刷新流水和客户余额……")
			state.dlg.Accept()
		})
	})
}

func (ui *mainUI) refreshCustomerFinanceBalance() {
	state := ui.customerFinance
	if state == nil || state.closed.Load() {
		return
	}
	customerID := state.customer.ID
	customerCode := state.customer.Code
	if state.balanceCancel != nil {
		state.balanceCancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	state.balanceCancel = cancel
	guardedGo(func() {
		defer cancel()
		customer, found, requestErr := ui.session.Client.FindCustomer(ctx, customerID, customerCode)
		if ctx.Err() != nil || state.closed.Load() {
			return
		}
		ui.window.Synchronize(func() {
			if state != ui.customerFinance || state.balance == nil {
				return
			}
			state.balanceCancel = nil
			if requestErr != nil || !found {
				state.info.SetText(state.info.Text() + "；余额回读失败，请稍后刷新合作伙伴列表。")
				return
			}
			state.customer = customerPartnerRow(customer).Detail
			state.balance.SetText(fmt.Sprintf("%.2f 元", state.customer.ReceivableValue))
		})
	})
}

func newManualTransactionIdempotencyKey(customerID string) string {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err == nil {
		return "manual:" + strings.TrimSpace(customerID) + ":" + hex.EncodeToString(buffer)
	}
	return fmt.Sprintf("manual:%s:%d", strings.TrimSpace(customerID), time.Now().UnixNano())
}
