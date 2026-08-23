//go:build windows

package ui

import (
	"unsafe"

	"github.com/lxn/walk"
	"github.com/lxn/win"
)

func accessibleStatusColor(preferred walk.Color) walk.Color {
	var contrast win.HIGHCONTRAST
	contrast.CbSize = uint32(unsafe.Sizeof(contrast))
	if win.SystemParametersInfo(win.SPI_GETHIGHCONTRAST, contrast.CbSize, unsafe.Pointer(&contrast), 0) && contrast.DwFlags&win.HCF_HIGHCONTRASTON != 0 {
		return walk.Color(win.GetSysColor(win.COLOR_WINDOWTEXT))
	}
	return preferred
}

func secondaryTextColor() walk.Color {
	return accessibleStatusColor(walk.RGB(85, 85, 85))
}

func successTextColor() walk.Color {
	return successTextColor()
}

func warningTextColor() walk.Color {
	return warningTextColor()
}

func dangerTextColor() walk.Color {
	return dangerTextColor()
}

func infoTextColor() walk.Color {
	return infoTextColor()
}
