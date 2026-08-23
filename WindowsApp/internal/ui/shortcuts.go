package ui

import "github.com/lxn/walk"

func addWindowShortcut(window walk.Window, shortcuts []walk.Shortcut, handler func()) {
	if window == nil || handler == nil {
		return
	}
	for _, shortcut := range shortcuts {
		action := walk.NewAction()
		if err := action.SetShortcut(shortcut); err != nil {
			continue
		}
		action.Triggered().Attach(handler)
		if err := window.AsWindowBase().ShortcutActions().Add(action); err != nil {
			action.SetShortcut(walk.Shortcut{})
		}
	}
}

func drawingZoomInShortcuts() []walk.Shortcut {
	return []walk.Shortcut{
		{Modifiers: walk.ModControl, Key: walk.KeyAdd},
		{Modifiers: walk.ModControl, Key: walk.KeyOEMPlus},
		{Modifiers: walk.ModControl | walk.ModShift, Key: walk.KeyOEMPlus},
	}
}

func drawingZoomOutShortcuts() []walk.Shortcut {
	return []walk.Shortcut{
		{Modifiers: walk.ModControl, Key: walk.KeySubtract},
		{Modifiers: walk.ModControl, Key: walk.KeyOEMMinus},
	}
}
