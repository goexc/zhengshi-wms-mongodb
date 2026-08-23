package ui

import (
	"context"
	"errors"
	"fmt"

	"zhengshi-wms-windowsapp/internal/api"
)

func (ui *mainUI) notifyStatusSuccess(message string) {
	if ui == nil || ui.status == nil {
		return
	}
	ui.status.SetText(message)
	ui.status.SetTextColor(successTextColor())
}

func requestFailureText(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.Canceled) {
		return "操作已取消"
	}
	var transportErr *api.TransportError
	if errors.As(err, &transportErr) {
		if transportErr.Timeout() {
			return "请求超时，请检查网络后手动重试"
		}
		return "网络连接失败，请检查网络后手动重试"
	}
	var businessErr *api.BusinessError
	if errors.As(err, &businessErr) {
		switch businessErr.Code {
		case 401:
			return "登录状态已失效，客户端将返回登录页"
		case 403:
			return "权限不足：" + businessErr.Msg
		default:
			if businessErr.Code >= 500 {
				return "线上服务暂时不可用：" + businessErr.Msg
			}
			return "业务校验未通过：" + businessErr.Msg
		}
	}
	return fmt.Sprintf("请求失败：%v", err)
}
