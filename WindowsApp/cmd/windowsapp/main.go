package main

import (
	"errors"

	"github.com/lxn/walk"

	"zhengshi-wms-windowsapp/internal/config"
	"zhengshi-wms-windowsapp/internal/diagnostics"
	"zhengshi-wms-windowsapp/internal/securestore"
	"zhengshi-wms-windowsapp/internal/ui"
)

func main() {
	logger, _ := diagnostics.New()
	if logger != nil {
		defer logger.Close()
	}
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
