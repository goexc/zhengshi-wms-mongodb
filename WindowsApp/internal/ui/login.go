package ui

import (
	"context"
	"fmt"
	"strings"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"

	"zhengshi-wms-windowsapp/internal/api"
	"zhengshi-wms-windowsapp/internal/config"
	"zhengshi-wms-windowsapp/internal/securestore"
)

type Session struct {
	Client  *api.Client
	Login   api.LoginData
	Profile api.Profile
	Perms   api.Perms
}

func Login(cfg *config.Config) (*Session, bool) {
	var dlg *walk.Dialog
	var mobileEdit, passwordEdit *walk.LineEdit
	var rememberCB *walk.CheckBox
	var keepLoginCB *walk.CheckBox
	var showPasswordCB *walk.CheckBox
	var loginButton *walk.PushButton
	var statusLabel *walk.Label
	var session *Session
	var authenticated bool

	updateStatus := func(text string) {
		statusLabel.SetText(text)
	}
	submit := func() {
		baseURL := strings.TrimSpace(cfg.APIBaseURL)
		mobile := strings.TrimSpace(mobileEdit.Text())
		password := passwordEdit.Text()
		if baseURL == "" || mobile == "" || password == "" {
			updateStatus("请填写手机号和密码。")
			return
		}
		loginButton.SetEnabled(false)
		updateStatus("正在连接线上服务并验证账号……")
		go func() {
			client := api.NewClient(baseURL)
			loginData, err := client.Login(context.Background(), mobile, password)
			if err == nil {
				client.SetToken(loginData.Token)
			}
			var profile api.Profile
			var perms api.Perms
			if err == nil {
				profile, err = client.Profile(context.Background())
			}
			if err == nil {
				perms, err = client.Permissions(context.Background())
			}
			dlg.Synchronize(func() {
				loginButton.SetEnabled(true)
				if err != nil {
					updateStatus("登录失败：" + err.Error())
					passwordEdit.SetText("")
					passwordEdit.SetFocus()
					return
				}
				*cfg = config.Config{
					APIBaseURL:   baseURL,
					RememberUser: rememberCB.Checked(),
					KeepLoggedIn: keepLoginCB.Checked(),
				}
				if cfg.RememberUser {
					cfg.Mobile = mobile
				}
				_ = config.Save(*cfg)
				if cfg.KeepLoggedIn {
					if saveErr := securestore.Save(securestore.CachedSession{
						Token: loginData.Token, ExpiresAt: loginData.Exp,
						Mobile: mobile, APIBaseURL: baseURL,
					}); saveErr != nil {
						walk.MsgBox(dlg, "无法保持登录", "身份验证已成功，但 Windows 加密会话保存失败：\r\n"+saveErr.Error(), walk.MsgBoxIconWarning)
					}
				} else {
					_ = securestore.Delete()
				}
				session = &Session{Client: client, Login: loginData, Profile: profile, Perms: perms}
				authenticated = true
				dlg.Accept()
			})
		}()
	}

	err := Dialog{
		AssignTo:      &dlg,
		Title:         "登录 · 正时 WMS",
		DefaultButton: &loginButton,
		MinSize:       Size{Width: 520, Height: 390},
		Size:          Size{Width: 560, Height: 420},
		Font:          Font{Family: "Microsoft YaHei UI", PointSize: 9},
		Layout:        VBox{Margins: Margins{Left: 28, Top: 24, Right: 28, Bottom: 24}, Spacing: 12},
		Children: []Widget{
			Label{Text: "正时 WMS", Font: Font{Family: "Microsoft YaHei UI", PointSize: 18, Bold: true}},
			Label{Text: "仓库执行 Windows 客户端", TextColor: walk.RGB(90, 90, 90)},
			GroupBox{
				Title:  "连接环境",
				Layout: VBox{Margins: Margins{Left: 14, Top: 10, Right: 14, Bottom: 12}, Spacing: 6},
				Children: []Widget{
					Label{Text: "线上生产环境", Font: Font{Family: "Microsoft YaHei UI", PointSize: 9, Bold: true}, TextColor: walk.RGB(183, 45, 33)},
					Label{Text: "登录后将直接读取和操作生产数据，请确认账号与操作内容。", TextColor: walk.RGB(85, 85, 85)},
				},
			},
			GroupBox{
				Title:  "账号登录",
				Layout: Grid{Columns: 2, Margins: Margins{Left: 14, Top: 12, Right: 14, Bottom: 14}, Spacing: 10},
				Children: []Widget{
					Label{Text: "手机号 *"},
					LineEdit{AssignTo: &mobileEdit, Text: cfg.Mobile, MinSize: Size{Height: 30}, ToolTipText: "请输入登录手机号"},
					Label{Text: "密码 *"},
					LineEdit{AssignTo: &passwordEdit, PasswordMode: true, MinSize: Size{Height: 30}, ToolTipText: "请输入登录密码"},
				},
			},
			Composite{
				Layout: HBox{Spacing: 16},
				Children: []Widget{
					CheckBox{AssignTo: &rememberCB, Text: "记住手机号", Checked: cfg.RememberUser},
					CheckBox{AssignTo: &showPasswordCB, Text: "显示密码", OnClicked: func() {
						passwordEdit.SetPasswordMode(!showPasswordCB.Checked())
						passwordEdit.SetFocus()
					}},
					HSpacer{},
				},
			},
			CheckBox{AssignTo: &keepLoginCB, Text: "保持登录（使用 Windows 加密保护登录状态）", Checked: cfg.KeepLoggedIn},
			Label{AssignTo: &statusLabel, Text: "请输入手机号和密码。", TextColor: walk.RGB(80, 80, 80)},
			Composite{
				Layout: HBox{Spacing: 8},
				Children: []Widget{
					HSpacer{},
					PushButton{AssignTo: &loginButton, Text: "登录并进入系统", MinSize: Size{Width: 150, Height: 36}, OnClicked: submit},
				},
			},
		},
	}.Create(nil)
	if err != nil {
		walk.MsgBox(nil, "启动失败", fmt.Sprintf("无法创建登录窗口：%v", err), walk.MsgBoxIconError)
		return nil, false
	}
	passwordEdit.KeyDown().Attach(func(key walk.Key) {
		if key == walk.KeyReturn {
			submit()
		}
	})
	dlg.Run()
	return session, authenticated
}
