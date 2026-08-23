package ui

import (
	"context"
	"fmt"
	"strings"

	"github.com/lxn/walk"

	"zhengshi-wms-windowsapp/internal/api"
)

func canDeleteOutbound(order api.OutboundOrder) bool {
	return strings.TrimSpace(order.ID) != "" && strings.TrimSpace(order.Status) == "预发货"
}

func (ui *mainUI) deleteSelectedOutbound() {
	order, ok := ui.selectedOutbound()
	if !ok {
		return
	}
	if !hasButton(ui.session.Perms.Buttons, "outbound:order:delete") || !canDeleteOutbound(order) {
		walk.MsgBox(ui.window, "当前不可删除", "Windows 客户端只允许删除仍处于“预发货”的出库单。", walk.MsgBoxIconWarning)
		return
	}
	message := fmt.Sprintf("即将删除线上预发货单：\r\n\r\n单号：%s\r\n类型：%s\r\n金额：%.3f\r\n\r\n删除后不可在客户端撤销。是否继续？", order.Code, order.Type, order.TotalAmount)
	if walk.MsgBox(ui.window, "确认删除预发货单", message, walk.MsgBoxYesNo|walk.MsgBoxIconWarning) != walk.DlgCmdYes {
		return
	}
	ui.setOutboundOperationBusy(true, "正在重新读取出库单状态……")
	guardedGo(func() {
		current, found, err := ui.session.Client.FindOutbound(context.Background(), order.ID, order.Code)
		if err == nil {
			switch {
			case !found:
				err = fmt.Errorf("出库单已不存在，请刷新列表")
			case !canDeleteOutbound(current):
				err = fmt.Errorf("出库单状态已变为“%s”，客户端已停止删除", current.Status)
			}
		}
		if err == nil {
			err = ui.session.Client.DeleteOutbound(context.Background(), current.ID)
		}
		serverAccepted := err == nil
		var verifyErr error
		var stillExists bool
		if serverAccepted {
			_, stillExists, verifyErr = ui.session.Client.FindOutbound(context.Background(), order.ID, order.Code)
			if verifyErr == nil && stillExists {
				verifyErr = fmt.Errorf("删除后仍查询到该出库单")
			}
		}
		ui.window.Synchronize(func() {
			if ui.outbound == nil {
				return
			}
			ui.setOutboundOperationBusy(false, "")
			if err != nil {
				ui.outbound.actionHint.SetText("删除失败：" + requestFailureText(err) + "。写操作不会自动重试。")
				walk.MsgBox(ui.window, "删除失败", ui.outbound.actionHint.Text(), walk.MsgBoxIconError)
				return
			}
			ui.loadOutbound()
			ui.loadDashboard()
			if verifyErr != nil {
				message := "服务端已返回删除成功，但在线复核失败：" + verifyErr.Error() + "。请勿重复删除，刷新列表后人工核对。"
				ui.outbound.actionHint.SetText(message)
				walk.MsgBox(ui.window, "删除后复核失败", message, walk.MsgBoxIconWarning)
				return
			}
			ui.notifyStatusSuccess("预发货单 " + order.Code + " 已删除，并完成线上回读复核。")
		})
	})
}
