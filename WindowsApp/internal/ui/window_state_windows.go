package ui

import (
	"github.com/lxn/walk"
	"github.com/lxn/win"
)

const (
	mainWindowMinWidth      = 1024
	mainWindowMinHeight     = 640
	mainWindowDefaultWidth  = 1440
	mainWindowDefaultHeight = 840
)

func virtualDesktopBounds() walk.Rectangle {
	bounds := walk.Rectangle{
		X:      int(win.GetSystemMetrics(win.SM_XVIRTUALSCREEN)),
		Y:      int(win.GetSystemMetrics(win.SM_YVIRTUALSCREEN)),
		Width:  int(win.GetSystemMetrics(win.SM_CXVIRTUALSCREEN)),
		Height: int(win.GetSystemMetrics(win.SM_CYVIRTUALSCREEN)),
	}
	if bounds.Width <= 0 || bounds.Height <= 0 {
		return walk.Rectangle{Width: 1920, Height: 1080}
	}
	return bounds
}

func (ui *mainUI) initialMainWindowBounds() walk.Rectangle {
	desktop := virtualDesktopBounds()
	requested := walk.Rectangle{
		Width:  mainWindowDefaultWidth,
		Height: mainWindowDefaultHeight,
	}
	requested.X = desktop.X + (desktop.Width-requested.Width)/2
	requested.Y = desktop.Y + (desktop.Height-requested.Height)/2
	if ui.workspaceMatchesSession() && ui.cfg.Workspace.WindowBoundsSet {
		requested = walk.Rectangle{
			X: ui.cfg.Workspace.WindowX, Y: ui.cfg.Workspace.WindowY,
			Width: ui.cfg.Workspace.WindowWidth, Height: ui.cfg.Workspace.WindowHeight,
		}
	}
	return clampWindowBounds(requested, desktop, mainWindowMinWidth, mainWindowMinHeight)
}

func clampWindowBounds(requested, desktop walk.Rectangle, minWidth, minHeight int) walk.Rectangle {
	if desktop.Width <= 0 || desktop.Height <= 0 {
		desktop = walk.Rectangle{Width: 1920, Height: 1080}
	}
	width := requested.Width
	height := requested.Height
	if width < minWidth {
		width = minWidth
	}
	if height < minHeight {
		height = minHeight
	}
	if width > desktop.Width {
		width = desktop.Width
	}
	if height > desktop.Height {
		height = desktop.Height
	}
	x := requested.X
	y := requested.Y
	if x < desktop.X {
		x = desktop.X
	}
	if y < desktop.Y {
		y = desktop.Y
	}
	if x+width > desktop.X+desktop.Width {
		x = desktop.X + desktop.Width - width
	}
	if y+height > desktop.Y+desktop.Height {
		y = desktop.Y + desktop.Height - height
	}
	return walk.Rectangle{X: x, Y: y, Width: width, Height: height}
}

func validSavedWindowBounds(bounds walk.Rectangle) bool {
	return bounds.Width >= mainWindowMinWidth/2 && bounds.Height >= mainWindowMinHeight/2
}
