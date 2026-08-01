package ui

import (
	"fmt"
	"sync"
	"syscall"
	"unsafe"

	"github.com/lxn/walk"
	"github.com/lxn/win"
)

const tabCloseTargetWidth96DPI = 26

type tabCloseHandlerState struct {
	ui                 *mainUI
	originalWindowProc uintptr
}

var (
	tabCloseHandlersMu sync.RWMutex
	tabCloseHandlers   = make(map[win.HWND]tabCloseHandlerState)
	tabCloseWindowProc = syscall.NewCallback(handleTabCloseWindowMessage)
)

func closableTabTitle(title string) string {
	return title + "  ×"
}

func isTabClosePoint(rect win.RECT, x, y int, closeTargetWidth int32) bool {
	return int32(x) >= rect.Right-closeTargetWidth &&
		int32(x) < rect.Right &&
		int32(y) >= rect.Top &&
		int32(y) < rect.Bottom
}

func (ui *mainUI) installTabCloseHandler() error {
	if ui.tabs == nil {
		return fmt.Errorf("标签控件尚未创建")
	}
	tabHandle := nativeTabHandle(ui.tabs.Handle())
	if tabHandle == 0 {
		return fmt.Errorf("无法取得原生标签栏")
	}
	originalWindowProc := win.SetWindowLongPtr(tabHandle, win.GWLP_WNDPROC, tabCloseWindowProc)
	if originalWindowProc == 0 {
		return fmt.Errorf("无法安装标签关闭事件")
	}
	tabCloseHandlersMu.Lock()
	tabCloseHandlers[tabHandle] = tabCloseHandlerState{
		ui:                 ui,
		originalWindowProc: originalWindowProc,
	}
	tabCloseHandlersMu.Unlock()
	return nil
}

func nativeTabHandle(tabWidgetHandle win.HWND) win.HWND {
	for child := win.GetWindow(tabWidgetHandle, win.GW_CHILD); child != 0; child = win.GetWindow(child, win.GW_HWNDNEXT) {
		className := make([]uint16, 64)
		length, err := win.GetClassName(child, &className[0], len(className))
		if err == nil && syscall.UTF16ToString(className[:length]) == "SysTabControl32" {
			return child
		}
	}
	return 0
}

func handleTabCloseWindowMessage(hwnd win.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	tabCloseHandlersMu.RLock()
	state, exists := tabCloseHandlers[hwnd]
	tabCloseHandlersMu.RUnlock()
	if !exists {
		return win.DefWindowProc(hwnd, msg, wParam, lParam)
	}

	result := win.CallWindowProc(state.originalWindowProc, hwnd, msg, wParam, lParam)
	if msg == win.WM_LBUTTONUP {
		state.ui.handleTabMouseUp(
			hwnd,
			int(win.GET_X_LPARAM(lParam)),
			int(win.GET_Y_LPARAM(lParam)),
		)
	}
	if msg == win.WM_NCDESTROY {
		tabCloseHandlersMu.Lock()
		delete(tabCloseHandlers, hwnd)
		tabCloseHandlersMu.Unlock()
	}
	return result
}

func (ui *mainUI) handleTabMouseUp(tabHandle win.HWND, x, y int) {
	if ui.tabs == nil {
		return
	}
	dpi := int(win.GetDpiForWindow(tabHandle))
	closeTargetWidth := int32(walk.IntFrom96DPI(tabCloseTargetWidth96DPI, dpi))
	for index := 0; index < ui.tabs.Pages().Len(); index++ {
		page := ui.tabs.Pages().At(index)
		if page == ui.systemTab {
			continue
		}
		var rect win.RECT
		if win.SendMessage(
			tabHandle,
			win.TCM_GETITEMRECT,
			uintptr(index),
			uintptr(unsafe.Pointer(&rect)),
		) == 0 {
			continue
		}
		if isTabClosePoint(rect, x, y, closeTargetWidth) {
			ui.closeTabAt(index)
			return
		}
	}
}
