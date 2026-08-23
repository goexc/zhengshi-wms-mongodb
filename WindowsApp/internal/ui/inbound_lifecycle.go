package ui

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"

	"zhengshi-wms-windowsapp/internal/api"
)

func canEditInbound(status string) bool {
	return status == "待审核" || status == "审核不通过"
}

func canCheckInbound(status string) bool {
	return status == "待审核"
}

func canDeleteInbound(status string) bool {
	return status == "待审核" || status == "审核不通过"
}

func canReceiveInbound(status string) bool {
	switch status {
	case "待审核", "审核不通过", "作废", "入库完成":
		return false
	default:
		return true
	}
}

func (ui *mainUI) updateInboundActionButtons() {
	if ui.inboundAdd == nil {
		return
	}
	busy := ui.inboundOperationBusy
	ui.inboundAdd.SetEnabled(!busy && hasButton(ui.session.Perms.Buttons, "inbound:receipt:add"))
	ui.inboundEdit.SetEnabled(false)
	ui.inboundCheck.SetEnabled(false)
	ui.inboundDelete.SetEnabled(false)
	ui.inboundReceive.SetEnabled(false)
	if ui.inboundTable == nil {
		return
	}
	index := ui.inboundTable.CurrentIndex()
	if index < 0 || index >= len(ui.inboundRows) {
		if ui.inboundActionHint != nil {
			ui.inboundActionHint.SetText("选择一张入库单后，可按权限和当前状态执行操作。")
		}
		return
	}
	receipt := ui.inboundRows[index]
	ui.inboundEdit.SetEnabled(!busy && canEditInbound(receipt.Status) && hasButton(ui.session.Perms.Buttons, "inbound:receipt:edit"))
	ui.inboundCheck.SetEnabled(!busy && canCheckInbound(receipt.Status) && hasButton(ui.session.Perms.Buttons, "inbound:receipt:check"))
	ui.inboundDelete.SetEnabled(!busy && canDeleteInbound(receipt.Status) && hasButton(ui.session.Perms.Buttons, "inbound:receipt:delete"))
	ui.inboundReceive.SetEnabled(!busy && canReceiveInbound(receipt.Status) && hasButton(ui.session.Perms.Buttons, "inbound:receipt:receive"))
	if ui.inboundActionHint != nil {
		if busy {
			ui.inboundActionHint.SetText("正在重新读取线上状态或提交操作，请等待完成。")
		} else {
			ui.inboundActionHint.SetText(fmt.Sprintf("当前选择：%s · %s。按钮同时受账号权限和服务端状态约束。", receipt.Code, receipt.Status))
		}
	}
}

func (ui *mainUI) setInboundOperationBusy(busy bool) {
	ui.inboundOperationBusy = busy
	ui.updateInboundActionButtons()
}

func (ui *mainUI) checkSelectedInbound() {
	receipt, ok := ui.selectedInbound()
	if !ok {
		return
	}
	if !hasButton(ui.session.Perms.Buttons, "inbound:receipt:check") || !canCheckInbound(receipt.Status) {
		walk.MsgBox(ui.window, "当前不可审核", "只有待审核状态且具备审核权限的入库单可以审核。", walk.MsgBoxIconWarning)
		return
	}
	ui.setInboundOperationBusy(true)
	ui.inboundActionHint.SetText("正在重新读取线上入库单状态……")
	guardedGo(func() {
		current, found, err := ui.session.Client.FindInboundReceipt(context.Background(), receipt.ID, receipt.Code)
		ui.window.Synchronize(func() {
			ui.setInboundOperationBusy(false)
			switch {
			case err != nil:
				walk.MsgBox(ui.window, "审核前读取失败", err.Error(), walk.MsgBoxIconError)
			case !found:
				walk.MsgBox(ui.window, "入库单不存在", "该入库单可能已被其他用户删除，请刷新列表。", walk.MsgBoxIconWarning)
				ui.loadInbound()
			case !canCheckInbound(current.Status):
				walk.MsgBox(ui.window, "状态已变化", fmt.Sprintf("入库单当前状态为“%s”，不能继续审核。", current.Status), walk.MsgBoxIconWarning)
				ui.loadInbound()
			default:
				if showInboundCheckDialog(ui.window, ui.session.Client, current) {
					ui.loadInbound()
					ui.loadDashboard()
				}
			}
		})
	})
}

func showInboundCheckDialog(owner walk.Form, client *api.Client, receipt api.InboundReceipt) bool {
	var dlg *walk.Dialog
	var statusCombo *walk.ComboBox
	var info *walk.Label
	var submitButton, cancelButton *walk.PushButton
	var changed bool
	var busy bool
	var closed atomic.Bool
	ctx, cancel := context.WithCancel(context.Background())
	summary := inboundReceiptSummary(receipt)
	err := Dialog{
		AssignTo: &dlg,
		Title:    "审核入库单 - " + receipt.Code,
		MinSize:  Size{Width: 620, Height: 480},
		Size:     Size{Width: 760, Height: 600},
		Layout:   VBox{Margins: Margins{Left: 16, Top: 16, Right: 16, Bottom: 14}, Spacing: 10},
		Children: []Widget{
			Label{Text: "审核入库单", Font: Font{Family: "Microsoft YaHei UI", PointSize: 15, Bold: true}},
			Label{Text: "以下内容来自审核前重新读取的线上单据。", TextColor: secondaryTextColor()},
			TextEdit{Text: summary, ReadOnly: true, StretchFactor: 1, Accessibility: Accessibility{Name: "待审核入库单摘要"}},
			GroupBox{
				Title:  "审核结论 *",
				Layout: Grid{Columns: 2, Margins: Margins{Left: 12, Top: 10, Right: 12, Bottom: 12}, Spacing: 8},
				Children: []Widget{
					Label{Text: "结论"},
					ComboBox{AssignTo: &statusCombo, Model: []string{"审核通过", "审核不通过"}, CurrentIndex: 0, MinSize: Size{Width: 180, Height: 28}, Accessibility: Accessibility{Name: "入库单审核结论"}},
				},
			},
			Label{AssignTo: &info, Text: "提交前会再次检查该单据仍为待审核状态。", TextColor: secondaryTextColor()},
			Composite{Layout: HBox{Spacing: 8}, Children: []Widget{
				HSpacer{},
				PushButton{AssignTo: &cancelButton, Text: "取消", MinSize: Size{Width: 80, Height: 30}, OnClicked: func() { dlg.Cancel() }},
				PushButton{AssignTo: &submitButton, Text: "确认审核", MinSize: Size{Width: 96, Height: 30}},
			}},
		},
	}.Create(owner)
	if err != nil {
		cancel()
		walk.MsgBox(owner, "无法打开审核窗口", err.Error(), walk.MsgBoxIconError)
		return false
	}
	dlg.Disposing().Attach(func() { closed.Store(true); cancel() })
	dlg.Closing().Attach(func(canceled *bool, _ walk.CloseReason) {
		if busy {
			*canceled = true
			walk.MsgBox(dlg, "正在审核", "审核请求正在提交和复核，请等待完成。", walk.MsgBoxIconInformation)
		}
	})
	submitButton.Clicked().Attach(func() {
		status := statusCombo.Text()
		if walk.MsgBox(dlg, "确认审核结论", fmt.Sprintf("入库单：%s\r\n审核结论：%s\r\n\r\n是否提交？", receipt.Code, status), walk.MsgBoxYesNo|walk.MsgBoxIconQuestion) != walk.DlgCmdYes {
			return
		}
		busy = true
		statusCombo.SetEnabled(false)
		submitButton.SetEnabled(false)
		cancelButton.SetEnabled(false)
		info.SetText("正在重新核对状态并提交审核……")
		guardedGo(func() {
			current, found, requestErr := client.FindInboundReceipt(ctx, receipt.ID, receipt.Code)
			if requestErr == nil && !found {
				requestErr = fmt.Errorf("入库单已不存在")
			}
			if requestErr == nil && !canCheckInbound(current.Status) {
				requestErr = fmt.Errorf("入库单状态已变为“%s”", current.Status)
			}
			if requestErr == nil {
				requestErr = client.CheckInboundReceipt(ctx, receipt.ID, status)
			}
			var verifyErr error
			if requestErr == nil {
				verified, exists, err := client.FindInboundReceipt(ctx, receipt.ID, receipt.Code)
				switch {
				case err != nil:
					verifyErr = err
				case !exists:
					verifyErr = fmt.Errorf("审核后未查询到该入库单")
				case verified.Status != status:
					verifyErr = fmt.Errorf("审核后状态为“%s”，预期为“%s”", verified.Status, status)
				}
			}
			if ctx.Err() != nil || closed.Load() {
				return
			}
			dlg.Synchronize(func() {
				if closed.Load() {
					return
				}
				busy = false
				if requestErr != nil {
					statusCombo.SetEnabled(true)
					submitButton.SetEnabled(true)
					cancelButton.SetEnabled(true)
					info.SetText("审核失败：" + requestErr.Error())
					return
				}
				changed = true
				if verifyErr != nil {
					info.SetText("服务端已返回成功，但复核失败：" + verifyErr.Error() + "。请勿重复提交。")
					cancelButton.SetEnabled(true)
					walk.MsgBox(dlg, "审核后复核失败", info.Text(), walk.MsgBoxIconWarning)
					return
				}
				walk.MsgBox(dlg, "审核成功", "审核结论已提交，并已在线重新读取到目标状态。", walk.MsgBoxIconInformation)
				dlg.Accept()
			})
		})
	})
	dlg.Run()
	return changed
}

func (ui *mainUI) deleteSelectedInbound() {
	receipt, ok := ui.selectedInbound()
	if !ok {
		return
	}
	if !hasButton(ui.session.Perms.Buttons, "inbound:receipt:delete") || !canDeleteInbound(receipt.Status) {
		walk.MsgBox(ui.window, "当前不可删除", "只有待审核或审核不通过状态且具备删除权限的入库单可以删除。", walk.MsgBoxIconWarning)
		return
	}
	ui.setInboundOperationBusy(true)
	ui.inboundActionHint.SetText("正在删除前重新读取线上入库单……")
	guardedGo(func() {
		current, found, err := ui.session.Client.FindInboundReceipt(context.Background(), receipt.ID, receipt.Code)
		ui.window.Synchronize(func() {
			ui.setInboundOperationBusy(false)
			switch {
			case err != nil:
				walk.MsgBox(ui.window, "删除前读取失败", err.Error(), walk.MsgBoxIconError)
			case !found:
				walk.MsgBox(ui.window, "入库单不存在", "该入库单可能已被其他用户删除，列表将重新刷新。", walk.MsgBoxIconWarning)
				ui.loadInbound()
			case !canDeleteInbound(current.Status):
				walk.MsgBox(ui.window, "状态已变化", fmt.Sprintf("入库单当前状态为“%s”，不能删除。", current.Status), walk.MsgBoxIconWarning)
				ui.loadInbound()
			default:
				message := fmt.Sprintf("入库单：%s\r\n状态：%s\r\n物料：%d 条\r\n\r\n删除后无法在客户端恢复，是否继续？", current.Code, current.Status, len(current.Materials))
				if walk.MsgBox(ui.window, "确认删除入库单", message, walk.MsgBoxYesNo|walk.MsgBoxIconWarning) != walk.DlgCmdYes {
					return
				}
				ui.executeInboundDelete(current)
			}
		})
	})
}

func (ui *mainUI) executeInboundDelete(receipt api.InboundReceipt) {
	ui.setInboundOperationBusy(true)
	ui.inboundActionHint.SetText("正在删除线上入库单并复核结果……")
	guardedGo(func() {
		err := ui.session.Client.DeleteInboundReceipt(context.Background(), receipt.ID)
		var verifyErr error
		if err == nil {
			_, found, findErr := ui.session.Client.FindInboundReceipt(context.Background(), receipt.ID, receipt.Code)
			if findErr != nil {
				verifyErr = findErr
			} else if found {
				verifyErr = fmt.Errorf("删除后仍能查询到该入库单")
			}
		}
		ui.window.Synchronize(func() {
			ui.setInboundOperationBusy(false)
			if err != nil {
				walk.MsgBox(ui.window, "删除失败", err.Error(), walk.MsgBoxIconError)
				return
			}
			if verifyErr != nil {
				walk.MsgBox(ui.window, "删除后复核失败", "服务端已返回成功，但"+verifyErr.Error()+"。请勿重复删除，刷新列表人工核对。", walk.MsgBoxIconWarning)
			} else {
				ui.notifyStatusSuccess("入库单已删除，并完成线上回读复核。")
			}
			ui.loadInbound()
			ui.loadDashboard()
		})
	})
}

func inboundReceiptSummary(receipt api.InboundReceipt) string {
	business := strings.TrimSpace(receipt.SupplierName)
	if business == "" {
		business = strings.TrimSpace(receipt.CustomerName)
	}
	if business == "" {
		business = "—"
	}
	lines := []string{
		"入库单号：" + receipt.Code,
		"入库类型：" + receipt.Type,
		"当前状态：" + receipt.Status,
		"往来单位：" + business,
		fmt.Sprintf("预计金额：%.3f", receipt.TotalAmount),
		fmt.Sprintf("物料条数：%d", len(receipt.Materials)),
	}
	if receipt.ReceivingDate > 0 {
		lines = append(lines, "预计入库日期："+formatUnixMinute(receipt.ReceivingDate))
	}
	lines = append(lines, "", "物料明细：")
	for _, material := range receipt.Materials {
		lines = append(lines, fmt.Sprintf("%d. %s / %s    %g %s × %.3f", material.Index, material.Name, displayMaterialValue(material.Model), material.EstimatedQuantity, material.Unit, material.Price))
	}
	if strings.TrimSpace(receipt.Remark) != "" {
		lines = append(lines, "", "备注："+receipt.Remark)
	}
	return strings.Join(lines, "\r\n")
}
