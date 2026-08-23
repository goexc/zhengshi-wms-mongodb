package ui

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"

	"zhengshi-wms-windowsapp/internal/diagnostics"
)

func (ui *mainUI) showDiagnosticsDialog() {
	logPath, pathErr := diagnostics.Path()
	logTail, logErr := diagnostics.ReadTail(64 << 10)
	if pathErr != nil {
		logPath = "不可用：" + pathErr.Error()
	}
	if logErr != nil {
		logTail = "读取日志失败：" + logErr.Error()
	}
	if strings.TrimSpace(logTail) == "" {
		logTail = "当前没有请求日志。"
	}
	summary := fmt.Sprintf(
		"客户端版本：v%s\r\n构建时间：%s\r\n提交：%s\r\n运行环境：%s/%s\r\nAPI：%s\r\n账号：%s（%s）\r\n生成时间：%s\r\n日志文件：%s\r\n\r\n最近请求日志\r\n%s",
		clientVersion, buildTime, gitCommit, runtime.GOOS, runtime.GOARCH,
		ui.cfg.APIBaseURL, ui.session.Profile.Name, ui.session.Profile.DepartmentName,
		time.Now().Format("2006-01-02 15:04:05"), logPath, logTail,
	)
	var dlg *walk.Dialog
	var content *walk.TextEdit
	var info *walk.Label
	err := Dialog{
		AssignTo: &dlg, Title: "客户端诊断信息",
		MinSize: Size{Width: 760, Height: 560}, Size: Size{Width: 900, Height: 680},
		Layout: VBox{Margins: Margins{Left: 16, Top: 16, Right: 16, Bottom: 14}, Spacing: 10},
		Children: []Widget{
			Label{Text: "客户端诊断信息", Font: Font{Family: "Microsoft YaHei UI", PointSize: 15, Bold: true}},
			Label{Text: "摘要不包含登录密码或身份 Token；请求日志只记录方法、路径、状态码和耗时。", TextColor: secondaryTextColor()},
			TextEdit{AssignTo: &content, Text: summary, ReadOnly: true, StretchFactor: 1, Accessibility: Accessibility{Name: "可复制的客户端诊断摘要"}},
			Label{AssignTo: &info, Text: "可复制摘要提供给技术支持。", TextColor: secondaryTextColor()},
			Composite{Layout: HBox{Spacing: 8}, Children: []Widget{
				PushButton{Text: "打开日志目录", Enabled: pathErr == nil, MinSize: Size{Width: 104, Height: 30}, OnClicked: func() {
					command := exec.Command("explorer.exe", "/select,"+logPath)
					if err := command.Start(); err != nil {
						info.SetText("打开日志目录失败：" + err.Error())
					}
				}},
				PushButton{Text: "保存诊断包", MinSize: Size{Width: 104, Height: 30}, Accessibility: Accessibility{Name: "保存脱敏诊断摘要和本地日志压缩包"}, OnClicked: func() {
					dialog := new(walk.FileDialog)
					dialog.Title = "保存客户端诊断包"
					dialog.Filter = "ZIP 压缩包 (*.zip)|*.zip"
					dialog.FilePath = fmt.Sprintf("ZhengshiWMS-diagnostics-%s.zip", time.Now().Format("20060102-150405"))
					accepted, saveErr := dialog.ShowSave(dlg)
					if saveErr != nil {
						info.SetText("选择保存位置失败：" + saveErr.Error())
						return
					}
					if !accepted {
						return
					}
					target := strings.TrimSpace(dialog.FilePath)
					if filepath.Ext(target) == "" {
						target += ".zip"
					}
					if saveErr := diagnostics.ExportSupportBundle(target, content.Text()); saveErr != nil {
						info.SetText("诊断包保存失败：" + saveErr.Error())
						return
					}
					info.SetText("诊断包已保存；发送前仍建议人工确认其中内容。")
				}},
				HSpacer{},
				PushButton{Text: "复制诊断摘要", MinSize: Size{Width: 112, Height: 30}, OnClicked: func() {
					if err := walk.Clipboard().SetText(content.Text()); err != nil {
						info.SetText("复制失败：" + err.Error())
						return
					}
					info.SetText("诊断摘要已复制。")
				}},
				PushButton{Text: "关闭", MinSize: Size{Width: 84, Height: 30}, OnClicked: func() { dlg.Accept() }},
			}},
		},
	}.Create(ui.window)
	if err != nil {
		walk.MsgBox(ui.window, "无法打开诊断窗口", err.Error(), walk.MsgBoxIconError)
		return
	}
	dlg.Run()
}
