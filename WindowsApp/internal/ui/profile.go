package ui

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"

	"zhengshi-wms-windowsapp/internal/api"
	"zhengshi-wms-windowsapp/internal/config"
	"zhengshi-wms-windowsapp/internal/securestore"
)

func (ui *mainUI) profilePageWidget() TabPage {
	profile := ui.session.Profile
	return TabPage{
		AssignTo: &ui.profileTab,
		Title:    closableTabTitle("个人中心"),
		Layout:   VBox{Margins: Margins{Left: 24, Top: 24, Right: 24, Bottom: 20}, Spacing: 12},
		Children: []Widget{
			Label{Text: "个人中心", Font: Font{Family: "Microsoft YaHei UI", PointSize: 15, Bold: true}},
			Label{Text: "个人资料以只读方式展示；头像和密码分别使用独立接口更新，不调用完整资料编辑接口。", TextColor: secondaryTextColor()},
			GroupBox{
				Title:  "账号资料",
				Layout: Grid{Columns: 4, Margins: Margins{Left: 16, Top: 14, Right: 16, Bottom: 16}, Spacing: 12},
				Children: []Widget{
					Label{Text: "账号名称", Font: Font{Bold: true}}, TextLabel{Text: displayMaterialValue(profile.Name)},
					Label{Text: "账号状态", Font: Font{Bold: true}}, TextLabel{Text: displayMaterialValue(profile.Status)},
					Label{Text: "手机号", Font: Font{Bold: true}}, TextLabel{Text: displayMaterialValue(profile.Mobile)},
					Label{Text: "Email", Font: Font{Bold: true}}, TextLabel{Text: displayMaterialValue(profile.Email)},
					Label{Text: "性别", Font: Font{Bold: true}}, TextLabel{Text: displayMaterialValue(profile.Sex)},
					Label{Text: "所属部门", Font: Font{Bold: true}}, TextLabel{Text: displayMaterialValue(profile.DepartmentName)},
					Label{Text: "备注", Font: Font{Bold: true}}, TextLabel{Text: displayMaterialValue(profile.Remark), ColumnSpan: 3},
				},
			},
			GroupBox{
				Title:  "个人头像",
				Layout: Grid{Columns: 4, Margins: Margins{Left: 16, Top: 14, Right: 16, Bottom: 16}, Spacing: 10},
				Children: []Widget{
					Label{Text: "当前头像"},
					LineEdit{AssignTo: &ui.profileAvatar, Text: profile.Avatar, ReadOnly: true, ColumnSpan: 3, MinSize: Size{Height: 30}, Accessibility: Accessibility{Name: "当前账号头像地址"}},
					Label{AssignTo: &ui.profileAvatarStatus, Text: "选择本地图片后会先上传，再调用独立头像接口并在线回读。", TextColor: secondaryTextColor(), ColumnSpan: 2, Accessibility: Accessibility{Name: "头像更新状态"}},
					Composite{ColumnSpan: 2, Layout: HBox{Spacing: 8}, Children: []Widget{
						HSpacer{},
						PushButton{AssignTo: &ui.profileAvatarPreview, Text: "查看当前头像", Enabled: strings.TrimSpace(profile.Avatar) != "", MinSize: Size{Width: 108, Height: 30}, OnClicked: ui.previewCurrentAvatar},
						PushButton{AssignTo: &ui.profileAvatarChange, Text: "选择并更新头像", MinSize: Size{Width: 126, Height: 30}, ToolTipText: "上传或更新期间再次点击可取消", Accessibility: Accessibility{Name: "选择并更新当前账号头像"}, OnClicked: ui.changeCurrentAvatar},
					}},
				},
			},
			GroupBox{
				Title:  "账号安全",
				Layout: VBox{Margins: Margins{Left: 16, Top: 14, Right: 16, Bottom: 16}, Spacing: 10},
				Children: []Widget{
					Label{Text: "修改成功后，客户端会清除本机加密登录缓存、调用退出接口并返回登录窗口。", TextColor: secondaryTextColor()},
					Composite{Layout: HBox{}, Children: []Widget{
						PushButton{Text: "修改密码", MinSize: Size{Width: 110, Height: 32}, OnClicked: ui.showChangePasswordDialog},
						HSpacer{},
					}},
				},
			},
			VSpacer{},
		},
	}
}

func (ui *mainUI) previewCurrentAvatar() {
	if ui.profileAvatar == nil || strings.TrimSpace(ui.profileAvatar.Text()) == "" {
		return
	}
	ShowOrderAttachments(ui.window, ui.session.Client, config.ImageBaseURL(), "账号头像 "+ui.session.Profile.Name, []string{ui.profileAvatar.Text()})
}

func (ui *mainUI) changeCurrentAvatar() {
	if ui.profileAvatarChange == nil {
		return
	}
	if ui.profileAvatarBusy {
		if ui.profileAvatarCancelable && ui.profileAvatarCancel != nil {
			ui.profileAvatarCancel()
			ui.profileAvatarChange.SetText("正在取消")
			ui.profileAvatarChange.SetEnabled(false)
			ui.profileAvatarStatus.SetText("正在取消头像上传或更新……")
		}
		return
	}
	dialog := newImageOpenDialog()
	accepted, err := dialog.ShowOpen(ui.window)
	if err != nil {
		ui.profileAvatarStatus.SetText("选择图片失败：" + err.Error())
		return
	}
	if !accepted {
		return
	}
	if walk.MsgBox(ui.window, "确认更新头像", "将上传所选图片并更新当前账号头像。若头像更新失败，已上传的图片素材不会自动删除。是否继续？", walk.MsgBoxYesNo|walk.MsgBoxIconQuestion) != walk.DlgCmdYes {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	ui.profileAvatarCancel = cancel
	ui.profileAvatarBusy = true
	ui.profileAvatarCancelable = true
	ui.profileAvatarChange.SetText("取消更新")
	ui.profileAvatarChange.SetEnabled(true)
	ui.profileAvatarPreview.SetEnabled(false)
	ui.profileAvatarStatus.SetText("正在上传头像图片……")
	filePath := dialog.FilePath
	guardedGo(func() {
		defer cancel()
		reference, requestErr := ui.session.Client.UploadImage(ctx, filePath)
		var avatarURL string
		var profile api.Profile
		verifiedProfile := false
		if requestErr == nil {
			avatarURL, requestErr = api.ResolveImageURL(config.ImageBaseURL(), reference)
		}
		if requestErr == nil {
			ui.window.Synchronize(func() {
				ui.profileAvatarCancelable = false
				if ui.profileAvatarChange != nil && ui.profileAvatarStatus != nil {
					ui.profileAvatarChange.SetText("正在更新")
					ui.profileAvatarChange.SetEnabled(false)
					ui.profileAvatarStatus.SetText("图片已上传，正在提交头像并进行线上回读；此阶段不可取消。")
				}
			})
		}
		if requestErr == nil && ctx.Err() == nil {
			writeErr := ui.session.Client.ChangeAvatar(ctx, avatarURL)
			profile, requestErr = ui.session.Client.Profile(ctx)
			verifiedProfile = requestErr == nil
			if requestErr == nil {
				if strings.TrimSpace(profile.Avatar) == strings.TrimSpace(avatarURL) {
					requestErr = nil
				} else if writeErr != nil {
					requestErr = writeErr
				} else {
					requestErr = fmt.Errorf("服务端回读头像与本次提交不一致")
				}
			} else if writeErr != nil {
				requestErr = fmt.Errorf("头像提交失败且回读失败：%v；%w", writeErr, requestErr)
			}
		}
		canceled := ctx.Err() != nil
		ui.window.Synchronize(func() {
			ui.profileAvatarCancel = nil
			ui.profileAvatarBusy = false
			ui.profileAvatarCancelable = false
			if ui.profileAvatarChange == nil || ui.profileAvatarStatus == nil {
				return
			}
			ui.profileAvatarChange.SetText("选择并更新头像")
			ui.profileAvatarChange.SetEnabled(true)
			if canceled {
				ui.profileAvatarPreview.SetEnabled(strings.TrimSpace(ui.profileAvatar.Text()) != "")
				ui.profileAvatarStatus.SetText("头像上传或更新已取消，当前头像未由本次操作改变。")
				return
			}
			if requestErr != nil {
				if verifiedProfile {
					ui.session.Profile = profile
					ui.profileAvatar.SetText(profile.Avatar)
				}
				ui.profileAvatarPreview.SetEnabled(strings.TrimSpace(ui.profileAvatar.Text()) != "")
				ui.profileAvatarStatus.SetText("头像更新未通过回读复核：" + requestFailureText(requestErr) + "。当前显示以最近一次成功回读为准。")
				return
			}
			ui.session.Profile = profile
			ui.profileAvatar.SetText(profile.Avatar)
			ui.profileAvatarPreview.SetEnabled(true)
			ui.profileAvatarStatus.SetText("头像已更新并完成线上回读复核。")
		})
	})
}

func (ui *mainUI) showChangePasswordDialog() {
	var dlg *walk.Dialog
	var password, confirmation *walk.LineEdit
	var showPassword *walk.CheckBox
	var status *walk.Label
	var submit *walk.PushButton
	busy := false

	doSubmit := func() {
		if busy {
			return
		}
		value := password.Text()
		if utf8.RuneCountInString(value) < 6 {
			status.SetText("新密码至少需要 6 个字符。")
			password.SetFocus()
			return
		}
		if value != confirmation.Text() {
			status.SetText("两次输入的密码不一致。")
			confirmation.SetFocus()
			return
		}
		if !ui.confirmSessionExit("修改密码") {
			return
		}
		if walk.MsgBox(dlg, "确认修改密码", "密码修改成功后将立即退出当前 Windows 客户端会话，是否继续？", walk.MsgBoxYesNo|walk.MsgBoxIconWarning) != walk.DlgCmdYes {
			return
		}
		busy = true
		submit.SetEnabled(false)
		password.SetEnabled(false)
		confirmation.SetEnabled(false)
		showPassword.SetEnabled(false)
		status.SetText("正在提交新密码……")
		guardedGo(func() {
			err := ui.session.Client.ChangePassword(context.Background(), value)
			dlg.Synchronize(func() {
				if err != nil {
					busy = false
					submit.SetEnabled(true)
					password.SetEnabled(true)
					confirmation.SetEnabled(true)
					showPassword.SetEnabled(true)
					status.SetText("修改失败：" + err.Error() + "。请核对后重试。")
					password.SetFocus()
					return
				}
				_ = securestore.Delete()
				if ui.loggedOut != nil {
					*ui.loggedOut = true
				}
				guardedGo(func() { _ = ui.session.Client.Logout(context.Background()) })
				walk.MsgBox(dlg, "密码修改成功", "新密码已保存，本机登录缓存已清除。请使用新密码重新登录。", walk.MsgBoxIconInformation)
				dlg.Accept()
				ui.window.Close()
			})
		})
	}

	err := Dialog{
		AssignTo: &dlg, Title: "修改密码", DefaultButton: &submit,
		MinSize: Size{Width: 520, Height: 330}, Size: Size{Width: 560, Height: 380},
		Layout: VBox{Margins: Margins{Left: 24, Top: 22, Right: 24, Bottom: 20}, Spacing: 12},
		Children: []Widget{
			Label{Text: "修改登录密码", Font: Font{Family: "Microsoft YaHei UI", PointSize: 15, Bold: true}},
			Label{Text: "接口仅接收新密码，不要求填写旧密码；提交前请确认当前登录账号。", TextColor: secondaryTextColor()},
			GroupBox{Title: "新密码", Layout: Grid{Columns: 2, Margins: Margins{Left: 14, Top: 12, Right: 14, Bottom: 14}, Spacing: 10}, Children: []Widget{
				Label{Text: "新密码 *"}, LineEdit{AssignTo: &password, PasswordMode: true, MinSize: Size{Height: 30}},
				Label{Text: "确认密码 *"}, LineEdit{AssignTo: &confirmation, PasswordMode: true, MinSize: Size{Height: 30}},
			}},
			CheckBox{AssignTo: &showPassword, Text: "显示密码", OnClicked: func() {
				password.SetPasswordMode(!showPassword.Checked())
				confirmation.SetPasswordMode(!showPassword.Checked())
			}},
			Label{AssignTo: &status, Text: "密码至少 6 个字符。", TextColor: secondaryTextColor()},
			Composite{Layout: HBox{Spacing: 8}, Children: []Widget{
				HSpacer{},
				PushButton{Text: "取消", MinSize: Size{Width: 82, Height: 30}, OnClicked: func() {
					if !busy {
						dlg.Cancel()
					}
				}},
				PushButton{AssignTo: &submit, Text: "修改并重新登录", MinSize: Size{Width: 138, Height: 32}, OnClicked: doSubmit},
			}},
		},
	}.Create(ui.window)
	if err != nil {
		walk.MsgBox(ui.window, "无法打开密码窗口", err.Error(), walk.MsgBoxIconError)
		return
	}
	password.KeyDown().Attach(func(key walk.Key) {
		if key == walk.KeyReturn {
			confirmation.SetFocus()
		}
	})
	confirmation.KeyDown().Attach(func(key walk.Key) {
		if key == walk.KeyReturn {
			doSubmit()
		}
	})
	dlg.Run()
}

func (ui *mainUI) confirmSessionExit(action string) bool {
	busyNames := make([]string, 0, 3)
	dirtyNames := make([]string, 0, 3)
	if state := ui.inboundEditor; state != nil {
		if state.busy {
			busyNames = append(busyNames, "入库单")
		}
		if state.dirty {
			dirtyNames = append(dirtyNames, "入库单")
		}
	}
	if state := ui.outboundEditor; state != nil {
		if state.busy {
			busyNames = append(busyNames, "出库单")
		}
		if state.dirty {
			dirtyNames = append(dirtyNames, "出库单")
		}
	}
	if state := ui.partnerEditor; state != nil {
		if state.busy {
			busyNames = append(busyNames, "合作伙伴")
		}
		if state.dirty {
			dirtyNames = append(dirtyNames, "合作伙伴")
		}
	}
	if state := ui.warehouseEditor; state != nil {
		if state.busy {
			busyNames = append(busyNames, "仓储结构")
		}
		if state.dirty {
			dirtyNames = append(dirtyNames, "仓储结构")
		}
	}
	if len(busyNames) > 0 {
		walk.MsgBox(ui.window, "正在提交", fmt.Sprintf("以下编辑页正在提交或复核：%s。请等待完成后再%s。", strings.Join(busyNames, "、"), action), walk.MsgBoxIconInformation)
		return false
	}
	if len(dirtyNames) > 0 {
		return walk.MsgBox(ui.window, "放弃未提交修改", fmt.Sprintf("以下编辑页存在未提交修改：%s。继续%s将放弃这些修改，是否继续？", strings.Join(dirtyNames, "、"), action), walk.MsgBoxYesNo|walk.MsgBoxIconWarning) == walk.DlgCmdYes
	}
	return true
}
