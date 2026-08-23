package main

import (
	"errors"

	"github.com/lxn/walk"

	"zhengshi-wms-windowsapp/internal/config"
	"zhengshi-wms-windowsapp/internal/diagnostics"
	"zhengshi-wms-windowsapp/internal/securestore"
	"zhengshi-wms-windowsapp/internal/singleinstance"
	"zhengshi-wms-windowsapp/internal/ui"
)

func main() {
	instance, primary, err := singleinstance.Acquire(`Local\ZhengshiWMS.WindowsApp`)
	if err != nil {
		walk.MsgBox(nil, "无法启动客户端", "无法建立客户端单实例保护：\r\n"+err.Error(), walk.MsgBoxIconError)
		return
	}
	defer instance.Close()
	if !primary {
		if err := instance.ActivatePrimary(); err != nil {
			walk.MsgBox(nil, "正时 WMS", "客户端已经在当前 Windows 会话中运行，但未能自动唤醒已有窗口：\r\n"+err.Error(), walk.MsgBoxIconInformation)
		}
		return
	}

	logger, _ := diagnostics.New()
	if logger != nil {
		diagnostics.SetDefaultLogger(logger)
		defer logger.Close()
		defer diagnostics.SetDefaultLogger(nil)
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			incidentID := diagnostics.IncidentID()
			logger.RecordPanic(incidentID, recovered)
			message := "客户端发生未处理异常，已停止继续执行以避免重复业务操作。\r\n\r\n故障编号：" + incidentID
			if path, err := diagnostics.Path(); err == nil {
				message += "\r\n诊断日志：" + path
			}
			walk.MsgBox(nil, "客户端异常", message, walk.MsgBoxIconError)
		}
	}()
	cfg := config.Load()
	for {
		var session *ui.Session
		for {
			restored, err := ui.RestoreSession(cfg)
			if err == nil {
				session = restored
				break
			}
			var networkErr *ui.RestoreNetworkError
			if errors.As(err, &networkErr) {
				answer := walk.MsgBox(nil, "无法验证登录状态",
					networkErr.Error()+"\r\n\r\n选择“是”重试，选择“否”使用其他账号。",
					walk.MsgBoxYesNo|walk.MsgBoxIconWarning)
				if answer == walk.DlgCmdYes {
					continue
				}
				_ = securestore.Delete()
			}
			break
		}
		if session == nil {
			var ok bool
			session, ok = ui.Login(&cfg)
			if !ok {
				return
			}
		}
		session.Client.SetLogger(logger)
		result, err := ui.RunMain(session, cfg)
		if err != nil {
			walk.MsgBox(nil, "客户端错误", err.Error(), walk.MsgBoxIconError)
			return
		}
		if !result.LoggedOut {
			return
		}
	}
}
