//go:build windows

package ui

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"syscall"
	"unsafe"

	"github.com/lxn/win"
)

type nativeDocumentPrinter struct {
	hdc          win.HDC
	pageWidth    int32
	pageHeight   int32
	left         int32
	right        int32
	top          int32
	bottom       int32
	y            int32
	pageStarted  bool
	pageFinished bool
}

func printNativeDocument(owner win.HWND, document *printDocument) (bool, error) {
	if document == nil {
		return false, fmt.Errorf("没有可打印的文档")
	}
	dialog := win.PRINTDLGEX{
		LStructSize: uint32(unsafe.Sizeof(win.PRINTDLGEX{})),
		HwndOwner:   owner,
		Flags:       win.PD_RETURNDC | win.PD_USEDEVMODECOPIESANDCOLLATE | win.PD_NOPAGENUMS | win.PD_NOSELECTION,
		NMinPage:    1, NMaxPage: 1, NCopies: 1, NStartPage: win.START_PAGE_GENERAL,
	}
	hresult := win.PrintDlgEx(&dialog)
	defer func() {
		if dialog.HDevMode != 0 {
			win.GlobalFree(dialog.HDevMode)
		}
		if dialog.HDevNames != 0 {
			win.GlobalFree(dialog.HDevNames)
		}
	}()
	if win.FAILED(hresult) {
		return false, fmt.Errorf("Windows 打印对话框错误：0x%08X", uint32(hresult))
	}
	if dialog.DwResultAction != win.PD_RESULT_PRINT {
		if dialog.HDC != 0 {
			win.DeleteDC(dialog.HDC)
		}
		return false, nil
	}
	if dialog.HDC == 0 {
		return false, fmt.Errorf("Windows 没有返回打印机设备")
	}
	defer win.DeleteDC(dialog.HDC)

	name, err := syscall.UTF16PtrFromString(document.Title)
	if err != nil {
		return false, err
	}
	docInfo := win.DOCINFO{CbSize: int32(unsafe.Sizeof(win.DOCINFO{})), LpszDocName: name}
	if win.StartDoc(dialog.HDC, &docInfo) <= 0 {
		return false, fmt.Errorf("无法创建打印任务")
	}
	completed := false
	defer func() {
		if !completed {
			win.AbortDoc(dialog.HDC)
		}
	}()

	printer := newNativeDocumentPrinter(dialog.HDC)
	if err := printer.startPage(); err != nil {
		return false, err
	}
	if err := printer.drawText(document.Title, 18, win.FW_BOLD, 12); err != nil {
		return false, err
	}
	if document.LabelCode != "" {
		if err := printer.drawText(document.LabelCode, 28, win.FW_BOLD, 22); err != nil {
			return false, err
		}
	}
	for _, line := range document.Lines {
		if line == "" {
			printer.y += printer.pointsToPixels(8)
			continue
		}
		weight := int32(win.FW_NORMAL)
		if line == "物料核对" || line == "出库物料" {
			weight = win.FW_BOLD
		}
		if err := printer.drawText(line, 10, weight, 5); err != nil {
			return false, err
		}
	}
	if len(document.ImageData) > 0 {
		decoded, _, decodeErr := image.Decode(bytes.NewReader(document.ImageData))
		if decodeErr != nil {
			return false, fmt.Errorf("打印图纸解码失败：%w", decodeErr)
		}
		if err := printer.drawImage(decoded); err != nil {
			return false, err
		}
	}
	if err := printer.endPage(); err != nil {
		return false, err
	}
	if win.EndDoc(dialog.HDC) <= 0 {
		return false, fmt.Errorf("打印任务提交失败")
	}
	completed = true
	return true, nil
}

func newNativeDocumentPrinter(hdc win.HDC) *nativeDocumentPrinter {
	width := win.GetDeviceCaps(hdc, win.HORZRES)
	height := win.GetDeviceCaps(hdc, win.VERTRES)
	dpiX := win.GetDeviceCaps(hdc, win.LOGPIXELSX)
	dpiY := win.GetDeviceCaps(hdc, win.LOGPIXELSY)
	marginX := dpiX * 6 / 10
	marginY := dpiY * 6 / 10
	return &nativeDocumentPrinter{
		hdc: hdc, pageWidth: width, pageHeight: height,
		left: marginX, right: width - marginX, top: marginY, bottom: height - marginY, y: marginY,
	}
}

func (printer *nativeDocumentPrinter) startPage() error {
	if win.StartPage(printer.hdc) <= 0 {
		return fmt.Errorf("无法开始打印页面")
	}
	printer.pageStarted = true
	printer.pageFinished = false
	printer.y = printer.top
	return nil
}

func (printer *nativeDocumentPrinter) endPage() error {
	if !printer.pageStarted || printer.pageFinished {
		return nil
	}
	if win.EndPage(printer.hdc) <= 0 {
		return fmt.Errorf("无法结束打印页面")
	}
	printer.pageFinished = true
	return nil
}

func (printer *nativeDocumentPrinter) newPage() error {
	if err := printer.endPage(); err != nil {
		return err
	}
	return printer.startPage()
}

func (printer *nativeDocumentPrinter) pointsToPixels(points int32) int32 {
	return win.MulDiv(points, win.GetDeviceCaps(printer.hdc, win.LOGPIXELSY), 72)
}

func (printer *nativeDocumentPrinter) drawText(text string, pointSize, weight, gapPoints int32) error {
	font := createPrinterFont(printer.hdc, pointSize, weight)
	if font == 0 {
		return fmt.Errorf("无法创建打印字体")
	}
	defer win.DeleteObject(win.HGDIOBJ(font))
	oldFont := win.SelectObject(printer.hdc, win.HGDIOBJ(font))
	if oldFont == 0 {
		return fmt.Errorf("无法选择打印字体")
	}
	defer win.SelectObject(printer.hdc, oldFont)
	win.SetBkMode(printer.hdc, win.TRANSPARENT)
	win.SetTextColor(printer.hdc, win.RGB(0, 0, 0))
	value, err := syscall.UTF16PtrFromString(text)
	if err != nil {
		return err
	}
	length := int32(len(syscall.StringToUTF16(text)) - 1)
	rect := win.RECT{Left: printer.left, Top: printer.y, Right: printer.right, Bottom: printer.bottom}
	format := uint32(win.DT_LEFT | win.DT_WORDBREAK | win.DT_NOPREFIX | win.DT_CALCRECT)
	if win.DrawTextEx(printer.hdc, value, length, &rect, format, nil) == 0 {
		return fmt.Errorf("无法计算打印文本布局")
	}
	height := rect.Bottom - rect.Top
	if height <= 0 {
		height = printer.pointsToPixels(pointSize + 3)
	}
	if printer.y+height > printer.bottom {
		if err := printer.newPage(); err != nil {
			return err
		}
		rect = win.RECT{Left: printer.left, Top: printer.y, Right: printer.right, Bottom: printer.bottom}
	}
	rect.Bottom = printer.y + height
	format = uint32(win.DT_LEFT | win.DT_WORDBREAK | win.DT_NOPREFIX)
	if win.DrawTextEx(printer.hdc, value, length, &rect, format, nil) == 0 {
		return fmt.Errorf("无法输出打印文本")
	}
	printer.y += height + printer.pointsToPixels(gapPoints)
	return nil
}

func createPrinterFont(hdc win.HDC, pointSize, weight int32) win.HFONT {
	font := win.LOGFONT{LfHeight: -win.MulDiv(pointSize, win.GetDeviceCaps(hdc, win.LOGPIXELSY), 72), LfWeight: weight, LfCharSet: 1}
	face := syscall.StringToUTF16("Microsoft YaHei UI")
	copy(font.LfFaceName[:], face)
	return win.CreateFontIndirect(&font)
}

func (printer *nativeDocumentPrinter) drawImage(source image.Image) error {
	bounds := source.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width <= 0 || height <= 0 {
		return fmt.Errorf("图纸图片尺寸无效")
	}
	if int64(width)*int64(height) > 50_000_000 {
		return fmt.Errorf("图纸像素超过 5000 万，无法安全打印")
	}
	availableWidth := printer.right - printer.left
	availableHeight := printer.bottom - printer.y
	minimumHeight := win.GetDeviceCaps(printer.hdc, win.LOGPIXELSY) * 2
	if availableHeight < minimumHeight {
		if err := printer.newPage(); err != nil {
			return err
		}
		availableHeight = printer.bottom - printer.y
	}
	scale := math.Min(float64(availableWidth)/float64(width), float64(availableHeight)/float64(height))
	if scale <= 0 {
		return fmt.Errorf("打印页面没有足够空间显示图纸")
	}
	targetWidth := int32(math.Round(float64(width) * scale))
	targetHeight := int32(math.Round(float64(height) * scale))
	targetX := printer.left + (availableWidth-targetWidth)/2

	header := win.BITMAPINFOHEADER{
		BiSize: uint32(unsafe.Sizeof(win.BITMAPINFOHEADER{})), BiWidth: int32(width), BiHeight: -int32(height),
		BiPlanes: 1, BiBitCount: 32, BiCompression: win.BI_RGB, BiSizeImage: uint32(width * height * 4),
	}
	var bits unsafe.Pointer
	bitmap := win.CreateDIBSection(printer.hdc, &header, win.DIB_RGB_COLORS, &bits, 0, 0)
	if bitmap == 0 || bits == nil {
		return fmt.Errorf("无法创建图纸打印位图")
	}
	defer win.DeleteObject(win.HGDIOBJ(bitmap))
	pixels := unsafe.Slice((*byte)(bits), width*height*4)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r, g, b, _ := source.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			offset := (y*width + x) * 4
			pixels[offset] = byte(b >> 8)
			pixels[offset+1] = byte(g >> 8)
			pixels[offset+2] = byte(r >> 8)
			pixels[offset+3] = 0
		}
	}
	memoryDC := win.CreateCompatibleDC(printer.hdc)
	if memoryDC == 0 {
		return fmt.Errorf("无法创建图纸打印上下文")
	}
	defer win.DeleteDC(memoryDC)
	oldBitmap := win.SelectObject(memoryDC, win.HGDIOBJ(bitmap))
	if oldBitmap == 0 {
		return fmt.Errorf("无法选择图纸打印位图")
	}
	defer win.SelectObject(memoryDC, oldBitmap)
	win.SetStretchBltMode(printer.hdc, win.HALFTONE)
	if !win.StretchBlt(printer.hdc, targetX, printer.y, targetWidth, targetHeight, memoryDC, 0, 0, int32(width), int32(height), win.SRCCOPY) {
		return fmt.Errorf("图纸输出到打印机失败")
	}
	printer.y += targetHeight
	return nil
}
