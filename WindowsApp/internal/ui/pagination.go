package ui

import (
	"fmt"
	"math"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

func totalPages(total int64, pageSize int) int {
	if total <= 0 || pageSize <= 0 {
		return 1
	}
	return int((total + int64(pageSize) - 1) / int64(pageSize))
}

func promptPageNumber(owner walk.Form, current int, total int64, pageSize int) (int, bool) {
	pages := totalPages(total, pageSize)
	if current < 1 || current > pages {
		current = 1
	}
	var dlg *walk.Dialog
	var page *walk.NumberEdit
	var goButton *walk.PushButton
	err := Dialog{
		AssignTo: &dlg, Title: "跳转到指定页", DefaultButton: &goButton,
		MinSize: Size{Width: 380, Height: 200}, Size: Size{Width: 420, Height: 230},
		Layout: VBox{Margins: Margins{Left: 18, Top: 16, Right: 18, Bottom: 14}, Spacing: 10},
		Children: []Widget{
			Label{Text: fmt.Sprintf("共 %d 条，%d 页；请输入 1 至 %d。", total, pages, pages), Accessibility: Accessibility{Name: fmt.Sprintf("当前共有 %d 页", pages)}},
			Composite{Layout: Grid{Columns: 2, Spacing: 8}, Children: []Widget{
				Label{Text: "页码"},
				NumberEdit{AssignTo: &page, Value: float64(current), MinValue: 1, MaxValue: float64(pages), Increment: 1, Decimals: 0, SpinButtonsVisible: true, Accessibility: Accessibility{Name: "目标页码"}},
			}},
			VSpacer{},
			Composite{Layout: HBox{Spacing: 8}, Children: []Widget{
				HSpacer{},
				PushButton{Text: "取消", MinSize: Size{Width: 80, Height: 30}, OnClicked: func() { dlg.Cancel() }},
				PushButton{AssignTo: &goButton, Text: "跳转", MinSize: Size{Width: 88, Height: 30}, Accessibility: Accessibility{Name: "跳转到输入的页码"}, OnClicked: func() { dlg.Accept() }},
			}},
		},
	}.Create(owner)
	if err != nil {
		walk.MsgBox(owner, "无法打开页码跳转", err.Error(), walk.MsgBoxIconError)
		return current, false
	}
	if dlg.Run() != walk.DlgCmdOK {
		return current, false
	}
	return int(math.Round(page.Value())), true
}
