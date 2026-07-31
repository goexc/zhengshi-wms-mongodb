package ui

import (
	"bytes"
	"context"
	"fmt"
	"image"
	stddraw "image/draw"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"strings"
	"sync/atomic"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"github.com/lxn/win"
	xdraw "golang.org/x/image/draw"

	"zhengshi-wms-windowsapp/internal/api"
)

func hasMaterialDrawing(material api.Material) bool {
	return strings.TrimSpace(material.Image) != ""
}

func materialDrawingStatus(material api.Material) string {
	if hasMaterialDrawing(material) {
		return "有图纸"
	}
	return "无图纸"
}

func materialInfoText(material api.Material) string {
	value := func(text string) string {
		if text = strings.TrimSpace(text); text != "" {
			return text
		}
		return "—"
	}
	return fmt.Sprintf(
		"物料名称\r\n%s\r\n\r\n型号\r\n%s\r\n\r\n物料分类\r\n%s\r\n\r\n材质\r\n%s\r\n\r\n规格\r\n%s\r\n\r\n表面处理\r\n%s\r\n\r\n强度等级\r\n%s\r\n\r\n安全库存\r\n%s\r\n\r\n备注\r\n%s",
		value(material.Name),
		value(material.Model),
		value(material.CategoryName),
		value(material.Material),
		value(material.Specification),
		value(material.SurfaceTreatment),
		value(material.StrengthGrade),
		materialQuantityText(material),
		value(material.Remark),
	)
}

func ShowMaterialDetail(owner walk.Form, client *api.Client, imageBaseURL string, material api.Material) {
	if !hasMaterialDrawing(material) {
		showMaterialWithoutDrawing(owner, material)
		return
	}

	imageURL, err := api.ResolveImageURL(imageBaseURL, material.Image)
	if err != nil {
		walk.MsgBox(owner, "无法打开图纸", err.Error(), walk.MsgBoxIconError)
		return
	}

	var dlg *walk.Dialog
	var imageView *walk.ImageView
	var scrollView *walk.ScrollView
	var statusLabel *walk.Label
	var retryButton *walk.PushButton
	var fitButton *walk.PushButton
	var actualSizeButton *walk.PushButton
	var rotateLeftButton *walk.PushButton
	var rotateRightButton *walk.PushButton
	var zoomLabel *walk.Label
	var closeButton *walk.PushButton
	var currentBitmap *walk.Bitmap
	var closed atomic.Bool
	ctx, cancel := context.WithCancel(context.Background())

	titlePart := strings.TrimSpace(material.Model)
	if titlePart == "" {
		titlePart = strings.TrimSpace(material.Name)
	}
	err = Dialog{
		AssignTo:      &dlg,
		Title:         "物料图纸 - " + titlePart,
		DefaultButton: &closeButton,
		MinSize:       Size{Width: 900, Height: 580},
		Size:          Size{Width: 1120, Height: 700},
		Layout:        VBox{Margins: Margins{Left: 16, Top: 16, Right: 16, Bottom: 14}, Spacing: 10},
		Children: []Widget{
			Label{
				Text:          strings.TrimSpace(material.Name),
				Font:          Font{Family: "Microsoft YaHei UI", PointSize: 15, Bold: true},
				EllipsisMode:  EllipsisEnd,
				ToolTipText:   material.Name,
				Accessibility: Accessibility{Name: "当前物料名称"},
			},
			Label{
				Text:          fmt.Sprintf("型号：%s    图纸状态：有图纸", displayMaterialValue(material.Model)),
				TextColor:     walk.RGB(70, 70, 70),
				Accessibility: Accessibility{Name: "物料型号及图纸状态"},
			},
			HSplitter{
				HandleWidth:   6,
				StretchFactor: 1,
				Children: []Widget{
					GroupBox{
						Title:         "图纸预览",
						StretchFactor: 5,
						MinSize:       Size{Width: 620, Height: 450},
						Layout:        VBox{Margins: Margins{Left: 12, Top: 12, Right: 12, Bottom: 12}, Spacing: 8},
						Children: []Widget{
							Label{
								AssignTo:      &statusLabel,
								Text:          "正在加载图纸原图……",
								TextColor:     walk.RGB(70, 70, 70),
								Accessibility: Accessibility{Name: "图纸加载状态"},
							},
							Composite{
								Layout: HBox{Spacing: 8},
								Children: []Widget{
									PushButton{
										AssignTo:      &fitButton,
										Text:          "适应宽度",
										Enabled:       false,
										MinSize:       Size{Width: 84, Height: 30},
										ToolTipText:   "将图纸宽度缩放到预览区域宽度",
										Accessibility: Accessibility{Name: "图纸适应宽度"},
									},
									PushButton{
										AssignTo:      &actualSizeButton,
										Text:          "100%",
										Enabled:       false,
										MinSize:       Size{Width: 64, Height: 30},
										ToolTipText:   "按原始像素显示图纸",
										Accessibility: Accessibility{Name: "图纸原始大小"},
									},
									Label{
										Text:          "Ctrl + 鼠标滚轮缩放",
										TextColor:     walk.RGB(80, 80, 80),
										Accessibility: Accessibility{Name: "按住 Ctrl 并滚动鼠标滚轮可缩放图纸"},
									},
									HSpacer{},
									Label{
										AssignTo:      &zoomLabel,
										Text:          "—",
										MinSize:       Size{Width: 48},
										TextAlignment: AlignFar,
										Accessibility: Accessibility{Name: "当前缩放比例"},
									},
								},
							},
							ScrollView{
								AssignTo:      &scrollView,
								StretchFactor: 1,
								MinSize:       Size{Width: 480, Height: 350},
								Layout:        HBox{Margins: Margins{Left: 8, Top: 8, Right: 8, Bottom: 8}},
								Accessibility: Accessibility{Name: "可滚动图纸预览区域"},
								Children: []Widget{
									ImageView{
										AssignTo:    &imageView,
										Mode:        ImageViewModeIdeal,
										Alignment:   AlignHCenterVCenter,
										MinSize:     Size{Width: 560, Height: 320},
										Margin:      8,
										ToolTipText: "按住 Ctrl 并滚动鼠标滚轮缩放；放大后可使用滚动条查看细节",
										Accessibility: Accessibility{
											Name:        "物料图纸预览",
											Description: fmt.Sprintf("%s，型号 %s 的图纸", material.Name, material.Model),
										},
									},
								},
							},
							Composite{
								Layout: HBox{Spacing: 8},
								Children: []Widget{
									PushButton{
										AssignTo:      &rotateLeftButton,
										Text:          "向左旋转",
										Enabled:       false,
										MinSize:       Size{Width: 92, Height: 30},
										ToolTipText:   "将图纸向左旋转 90°，并重新适应预览宽度",
										Accessibility: Accessibility{Name: "图纸向左旋转九十度"},
									},
									PushButton{
										AssignTo:      &rotateRightButton,
										Text:          "向右旋转",
										Enabled:       false,
										MinSize:       Size{Width: 92, Height: 30},
										ToolTipText:   "将图纸向右旋转 90°，并重新适应预览宽度",
										Accessibility: Accessibility{Name: "图纸向右旋转九十度"},
									},
									HSpacer{},
									PushButton{
										AssignTo:    &retryButton,
										Text:        "重试加载",
										Visible:     false,
										MinSize:     Size{Width: 96, Height: 30},
										ToolTipText: "重新从图片服务加载图纸",
									},
								},
							},
						},
					},
					GroupBox{
						Title:         "物料信息",
						StretchFactor: 2,
						MinSize:       Size{Width: 240, Height: 450},
						Layout:        Grid{Columns: 2, Margins: Margins{Left: 12, Top: 12, Right: 12, Bottom: 12}, Spacing: 8},
						Children:      materialInformationWidgets(material),
					},
				},
			},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					Label{Text: "按住 Ctrl 滚动鼠标滚轮缩放；普通滚轮和滚动条用于浏览图纸。", TextColor: walk.RGB(90, 90, 90)},
					HSpacer{},
					PushButton{AssignTo: &closeButton, Text: "关闭", MinSize: Size{Width: 88, Height: 30}, OnClicked: func() { dlg.Accept() }},
				},
			},
		},
	}.Create(owner)
	if err != nil {
		cancel()
		walk.MsgBox(owner, "详情窗口错误", err.Error(), walk.MsgBoxIconError)
		return
	}
	if err := enableMaterialDialogSystemButtons(dlg); err != nil {
		cancel()
		dlg.Dispose()
		walk.MsgBox(owner, "详情窗口错误", "无法启用窗口最小化和最大化功能："+err.Error(), walk.MsgBoxIconError)
		return
	}
	closeButton.SetFocus()

	dlg.Disposing().Attach(func() {
		closed.Store(true)
		cancel()
		if currentBitmap != nil {
			currentBitmap.Dispose()
			currentBitmap = nil
		}
	})

	var loadGeneration int
	var renderGeneration int
	var drawingImage image.Image
	var rotationQuarterTurns int
	var zoomScale float64
	var maxZoomScale = 1.0
	var rendering bool
	setZoomControlsEnabled := func(enabled bool) {
		ready := enabled && drawingImage != nil
		fitButton.SetEnabled(ready)
		actualSizeButton.SetEnabled(ready)
		rotateLeftButton.SetEnabled(ready)
		rotateRightButton.SetEnabled(ready)
	}

	applyRenderedDrawing := func(rendered image.Image, targetScale float64, generation int, onSuccess func()) {
		dlg.Synchronize(func() {
			if closed.Load() || generation != renderGeneration {
				return
			}
			bitmap, bitmapErr := walk.NewBitmapFromImageForDPI(rendered, dlg.DPI())
			if bitmapErr != nil {
				rendering = false
				statusLabel.SetText("图纸显示失败：" + bitmapErr.Error() + "。请重试。")
				retryButton.SetVisible(true)
				setZoomControlsEnabled(true)
				return
			}
			if setErr := imageView.SetImage(bitmap); setErr != nil {
				bitmap.Dispose()
				rendering = false
				statusLabel.SetText("图纸显示失败：" + setErr.Error() + "。请重试。")
				retryButton.SetVisible(true)
				setZoomControlsEnabled(true)
				return
			}
			if currentBitmap != nil {
				currentBitmap.Dispose()
			}
			currentBitmap = bitmap
			if onSuccess != nil {
				onSuccess()
			}
			zoomScale = targetScale
			rendering = false
			zoomLabel.SetText(formatMaterialZoom(zoomScale))
			statusLabel.SetText(materialDrawingReadyStatus(rotationQuarterTurns))
			setZoomControlsEnabled(true)
		})
	}

	var renderZoom func(float64)
	renderZoom = func(targetScale float64) {
		if drawingImage == nil || rendering {
			return
		}
		targetScale = clampMaterialZoom(targetScale, maxZoomScale)
		if math.Abs(targetScale-zoomScale) < 0.001 {
			return
		}
		rendering = true
		renderGeneration++
		generation := renderGeneration
		source := drawingImage
		setZoomControlsEnabled(false)
		statusLabel.SetText(fmt.Sprintf("正在生成 %s 预览……", formatMaterialZoom(targetScale)))
		go func() {
			rendered := scaleMaterialDrawing(source, targetScale)
			if closed.Load() || ctx.Err() != nil {
				return
			}
			applyRenderedDrawing(rendered, targetScale, generation, nil)
		}()
	}

	fitDrawingWidth := func() {
		if drawingImage == nil {
			return
		}
		viewport := scrollView.ClientBoundsPixels()
		renderZoom(materialFitWidthZoom(drawingImage.Bounds(), viewport.Width-24))
	}
	fitButton.Clicked().Attach(fitDrawingWidth)
	actualSizeButton.Clicked().Attach(func() {
		renderZoom(1)
	})

	rotateDrawing := func(direction int) {
		if drawingImage == nil || rendering {
			return
		}
		rendering = true
		renderGeneration++
		generation := renderGeneration
		source := drawingImage
		viewportWidth := scrollView.ClientBoundsPixels().Width - 24
		nextRotation := normalizeMaterialRotation(rotationQuarterTurns + direction)
		action := "向右旋转"
		if direction < 0 {
			action = "向左旋转"
		}
		setZoomControlsEnabled(false)
		statusLabel.SetText("正在" + action + " 90°并适应宽度……")
		go func() {
			rotated := rotateMaterialDrawing(source, direction)
			rotatedMaxZoom := materialMaxZoom(rotated.Bounds())
			targetScale := clampMaterialZoom(materialFitWidthZoom(rotated.Bounds(), viewportWidth), rotatedMaxZoom)
			rendered := scaleMaterialDrawing(rotated, targetScale)
			if closed.Load() || ctx.Err() != nil {
				return
			}
			applyRenderedDrawing(rendered, targetScale, generation, func() {
				drawingImage = rotated
				rotationQuarterTurns = nextRotation
				maxZoomScale = rotatedMaxZoom
			})
		}()
	}
	rotateLeftButton.Clicked().Attach(func() {
		rotateDrawing(-1)
	})
	rotateRightButton.Clicked().Attach(func() {
		rotateDrawing(1)
	})

	imageView.MouseWheel().Attach(func(_, _ int, button walk.MouseButton) {
		const mouseWheelControlKey = 0x0008
		if walk.MouseWheelEventKeyState(button)&mouseWheelControlKey == 0 || drawingImage == nil {
			return
		}
		direction := -1
		if walk.MouseWheelEventDelta(button) > 0 {
			direction = 1
		}
		renderZoom(nextMaterialZoom(zoomScale, direction, maxZoomScale))
	})

	var loadImage func()
	loadImage = func() {
		loadGeneration++
		generation := loadGeneration
		renderGeneration++
		drawingImage = nil
		rotationQuarterTurns = 0
		zoomScale = 0
		rendering = false
		zoomLabel.SetText("—")
		setZoomControlsEnabled(false)
		statusLabel.SetText("正在加载图纸原图……")
		retryButton.SetVisible(false)
		go func() {
			data, requestErr := client.DownloadImage(ctx, imageURL)
			var decoded image.Image
			if requestErr == nil {
				decoded, _, requestErr = image.Decode(bytes.NewReader(data))
				if requestErr != nil {
					requestErr = fmt.Errorf("无法识别图纸图片格式: %w", requestErr)
				}
			}
			if closed.Load() || ctx.Err() != nil {
				return
			}
			dlg.Synchronize(func() {
				if closed.Load() || generation != loadGeneration {
					return
				}
				if requestErr != nil {
					statusLabel.SetText("图纸加载失败：" + requestErr.Error() + "。请检查网络后重试。")
					retryButton.SetVisible(true)
					return
				}
				drawingImage = decoded
				maxZoomScale = materialMaxZoom(decoded.Bounds())
				fitDrawingWidth()
			})
		}()
	}
	retryButton.Clicked().Attach(loadImage)
	dlg.Show()
	loadImage()
}

const materialZoomStep = 0.10

func materialDialogWindowStyle(style uint32) uint32 {
	return style | win.WS_MINIMIZEBOX | win.WS_MAXIMIZEBOX
}

func enableMaterialDialogSystemButtons(dlg *walk.Dialog) error {
	if dlg == nil || dlg.Handle() == 0 {
		return fmt.Errorf("图纸预览窗口尚未创建")
	}

	hwnd := dlg.Handle()
	win.SetLastError(0)
	style := uint32(win.GetWindowLong(hwnd, win.GWL_STYLE))
	if style == 0 && win.GetLastError() != win.ERROR_SUCCESS {
		return fmt.Errorf("读取窗口样式失败，错误代码 %d", win.GetLastError())
	}

	updatedStyle := materialDialogWindowStyle(style)
	if updatedStyle != style {
		win.SetLastError(0)
		if win.SetWindowLong(hwnd, win.GWL_STYLE, int32(updatedStyle)) == 0 && win.GetLastError() != win.ERROR_SUCCESS {
			return fmt.Errorf("更新窗口样式失败，错误代码 %d", win.GetLastError())
		}
	}

	if !win.SetWindowPos(
		hwnd,
		0,
		0,
		0,
		0,
		0,
		win.SWP_FRAMECHANGED|win.SWP_NOMOVE|win.SWP_NOSIZE|win.SWP_NOZORDER|win.SWP_NOOWNERZORDER,
	) {
		return fmt.Errorf("刷新窗口标题栏失败，错误代码 %d", win.GetLastError())
	}
	return nil
}

func materialFitWidthZoom(bounds image.Rectangle, viewportWidth int) float64 {
	if bounds.Dx() <= 0 || viewportWidth <= 0 {
		return 1
	}
	scale := float64(viewportWidth) / float64(bounds.Dx())
	if scale > 1 {
		scale = 1
	}
	return math.Max(0.05, scale)
}

func materialMaxZoom(bounds image.Rectangle) float64 {
	pixels := float64(bounds.Dx()) * float64(bounds.Dy())
	if pixels <= 0 {
		return 1
	}
	scale := math.Sqrt(16_000_000 / pixels)
	if scale < 1 {
		return 1
	}
	if scale > 4 {
		return 4
	}
	return scale
}

func clampMaterialZoom(scale, maxScale float64) float64 {
	if maxScale < 1 {
		maxScale = 1
	}
	if scale < 0.05 {
		return 0.05
	}
	if scale > maxScale {
		return maxScale
	}
	return scale
}

func nextMaterialZoom(current float64, direction int, maxScale float64) float64 {
	if direction < 0 {
		return clampMaterialZoom(current-materialZoomStep, maxScale)
	}
	return clampMaterialZoom(current+materialZoomStep, maxScale)
}

func formatMaterialZoom(scale float64) string {
	return fmt.Sprintf("%.0f%%", scale*100)
}

func normalizeMaterialRotation(quarterTurns int) int {
	quarterTurns %= 4
	if quarterTurns < 0 {
		quarterTurns += 4
	}
	return quarterTurns
}

func materialDrawingReadyStatus(rotationQuarterTurns int) string {
	switch normalizeMaterialRotation(rotationQuarterTurns) {
	case 1:
		return "图纸已加载 · 已向右旋转 90°"
	case 2:
		return "图纸已加载 · 已旋转 180°"
	case 3:
		return "图纸已加载 · 已向左旋转 90°"
	default:
		return "图纸已加载"
	}
}

func rotateMaterialDrawing(source image.Image, direction int) image.Image {
	if source == nil || direction == 0 {
		return source
	}

	bounds := source.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width <= 0 || height <= 0 {
		return source
	}

	var normalized *image.NRGBA
	if nrgba, ok := source.(*image.NRGBA); ok && bounds.Min == (image.Point{}) {
		normalized = nrgba
	} else {
		normalized = image.NewNRGBA(image.Rect(0, 0, width, height))
		stddraw.Draw(normalized, normalized.Bounds(), source, bounds.Min, stddraw.Src)
	}

	target := image.NewNRGBA(image.Rect(0, 0, height, width))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			targetX, targetY := height-1-y, x
			if direction < 0 {
				targetX, targetY = y, width-1-x
			}
			sourceOffset := normalized.PixOffset(x, y)
			targetOffset := target.PixOffset(targetX, targetY)
			copy(target.Pix[targetOffset:targetOffset+4], normalized.Pix[sourceOffset:sourceOffset+4])
		}
	}
	return target
}

func scaleMaterialDrawing(source image.Image, scale float64) image.Image {
	if math.Abs(scale-1) < 0.001 {
		return source
	}
	width := int(math.Round(float64(source.Bounds().Dx()) * scale))
	height := int(math.Round(float64(source.Bounds().Dy()) * scale))
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	target := image.NewNRGBA(image.Rect(0, 0, width, height))
	xdraw.CatmullRom.Scale(target, target.Bounds(), source, source.Bounds(), xdraw.Src, nil)
	return target
}

func materialInformationWidgets(material api.Material) []Widget {
	fields := []struct {
		label string
		value string
	}{
		{"物料名称", material.Name},
		{"型号", material.Model},
		{"物料分类", material.CategoryName},
		{"材质", material.Material},
		{"规格", material.Specification},
		{"表面处理", material.SurfaceTreatment},
		{"强度等级", material.StrengthGrade},
		{"安全库存", materialQuantityText(material)},
		{"备注", material.Remark},
	}
	widgets := make([]Widget, 0, len(fields)*2)
	for _, field := range fields {
		value := displayMaterialValue(field.value)
		widgets = append(widgets,
			Label{
				Text:          field.label,
				Font:          Font{Family: "Microsoft YaHei UI", Bold: true},
				Alignment:     AlignHNearVNear,
				Accessibility: Accessibility{Name: field.label},
			},
			TextLabel{
				Text:          value,
				TextAlignment: AlignHNearVNear,
				ToolTipText:   value,
				Accessibility: Accessibility{Name: field.label + "：" + value},
			},
		)
	}
	return widgets
}

func materialQuantityText(material api.Material) string {
	if unit := strings.TrimSpace(material.Unit); unit != "" {
		return fmt.Sprintf("%g %s", material.Quantity, unit)
	}
	return fmt.Sprintf("%g", material.Quantity)
}

func showMaterialWithoutDrawing(owner walk.Form, material api.Material) {
	var dlg *walk.Dialog
	var closeButton *walk.PushButton
	titlePart := strings.TrimSpace(material.Model)
	if titlePart == "" {
		titlePart = strings.TrimSpace(material.Name)
	}
	err := Dialog{
		AssignTo:      &dlg,
		Title:         "物料图纸 - " + titlePart,
		DefaultButton: &closeButton,
		MinSize:       Size{Width: 520, Height: 280},
		Size:          Size{Width: 600, Height: 330},
		Layout:        VBox{Margins: Margins{Left: 24, Top: 24, Right: 24, Bottom: 20}, Spacing: 12},
		Children: []Widget{
			Label{
				Text:          strings.TrimSpace(material.Name),
				Font:          Font{Family: "Microsoft YaHei UI", PointSize: 15, Bold: true},
				EllipsisMode:  EllipsisEnd,
				ToolTipText:   material.Name,
				Accessibility: Accessibility{Name: "当前物料名称"},
			},
			Label{
				Text:          fmt.Sprintf("型号：%s    图纸状态：无图纸", displayMaterialValue(material.Model)),
				TextColor:     walk.RGB(70, 70, 70),
				Accessibility: Accessibility{Name: "物料型号及图纸状态"},
			},
			GroupBox{
				Title:         "图纸预览",
				StretchFactor: 1,
				Layout:        VBox{Margins: Margins{Left: 16, Top: 16, Right: 16, Bottom: 16}},
				Children: []Widget{
					TextLabel{
						Text:          "该物料没有图纸。",
						TextAlignment: AlignHCenterVCenter,
						Font:          Font{Family: "Microsoft YaHei UI", PointSize: 13, Bold: true},
						TextColor:     walk.RGB(80, 80, 80),
						StretchFactor: 1,
						Accessibility: Accessibility{Name: "该物料没有图纸"},
					},
				},
			},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					HSpacer{},
					PushButton{AssignTo: &closeButton, Text: "关闭", MinSize: Size{Width: 88, Height: 30}, OnClicked: func() { dlg.Accept() }},
				},
			},
		},
	}.Create(owner)
	if err != nil {
		walk.MsgBox(owner, "详情窗口错误", err.Error(), walk.MsgBoxIconError)
		return
	}
	dlg.Show()
}

func displayMaterialValue(value string) string {
	if value = strings.TrimSpace(value); value != "" {
		return value
	}
	return "—"
}
