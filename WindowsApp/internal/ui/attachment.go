package ui

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"strings"
	"sync/atomic"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"

	"zhengshi-wms-windowsapp/internal/api"
)

type attachmentPreviewImageSetter interface {
	SetImage(walk.Image) error
}

type attachmentPreviewResource interface {
	Dispose()
}

func clearAttachmentPreview(setter attachmentPreviewImageSetter, current attachmentPreviewResource) error {
	err := setter.SetImage(nil)
	if current != nil {
		current.Dispose()
	}
	return err
}

func ShowOrderAttachments(
	owner walk.Form,
	client *api.Client,
	imageBaseURL, orderTitle string,
	attachments []string,
) {
	references := make([]string, 0, len(attachments))
	for _, reference := range attachments {
		if reference = strings.TrimSpace(reference); reference != "" {
			references = append(references, reference)
		}
	}
	if len(references) == 0 {
		walk.MsgBox(owner, "没有附件", orderTitle+"没有可预览的图片附件。", walk.MsgBoxIconInformation)
		return
	}

	var dlg *walk.Dialog
	var attachmentCombo *walk.ComboBox
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
	var currentBitmap attachmentPreviewResource
	var closed atomic.Bool
	ctx, cancel := context.WithCancel(context.Background())
	var loadCancel context.CancelFunc

	labels := make([]string, len(references))
	for index := range references {
		labels[index] = fmt.Sprintf("附件 %d / %d", index+1, len(references))
	}
	err := Dialog{
		AssignTo:      &dlg,
		Title:         "单据附件 - " + orderTitle,
		DefaultButton: &closeButton,
		MinSize:       Size{Width: 860, Height: 560},
		Size:          Size{Width: 1080, Height: 700},
		Layout:        VBox{Margins: Margins{Left: 16, Top: 16, Right: 16, Bottom: 14}, Spacing: 10},
		Children: []Widget{
			Label{
				Text: orderTitle, Font: Font{Family: "Microsoft YaHei UI", PointSize: 15, Bold: true},
				EllipsisMode: EllipsisEnd, ToolTipText: orderTitle,
			},
			Composite{
				Layout: HBox{Spacing: 8},
				Children: []Widget{
					Label{Text: "选择附件"},
					ComboBox{
						AssignTo: &attachmentCombo, Model: labels, CurrentIndex: 0,
						MinSize:       Size{Width: 140, Height: 28},
						Accessibility: Accessibility{Name: "选择需要预览的单据附件"},
					},
					Label{
						Text:      "附件来自当前线上单据；切换选择后重新加载原图。",
						TextColor: walk.RGB(80, 80, 80),
					},
					HSpacer{},
					Label{AssignTo: &zoomLabel, Text: "—", MinSize: Size{Width: 52}, TextAlignment: AlignFar},
				},
			},
			GroupBox{
				Title:         "图片预览",
				StretchFactor: 1,
				Layout:        VBox{Margins: Margins{Left: 12, Top: 12, Right: 12, Bottom: 12}, Spacing: 8},
				Children: []Widget{
					Label{
						AssignTo: &statusLabel, Text: "正在加载附件原图……",
						TextColor:     walk.RGB(70, 70, 70),
						Accessibility: Accessibility{Name: "附件图片加载状态"},
					},
					Composite{
						Layout: HBox{Spacing: 8},
						Children: []Widget{
							PushButton{
								AssignTo: &fitButton, Text: "适应宽度", Enabled: false,
								MinSize: Size{Width: 84, Height: 30}, ToolTipText: "将图片宽度适应预览区域",
								Accessibility: Accessibility{Name: "附件图片适应宽度"},
							},
							PushButton{
								AssignTo: &actualSizeButton, Text: "100%", Enabled: false,
								MinSize: Size{Width: 64, Height: 30}, ToolTipText: "按原始像素显示图片",
								Accessibility: Accessibility{Name: "附件图片原始大小"},
							},
							Label{Text: "Ctrl + 鼠标滚轮按 10% 缩放", TextColor: walk.RGB(80, 80, 80)},
							HSpacer{},
							PushButton{
								AssignTo: &retryButton, Text: "重试加载", Visible: false,
								MinSize:       Size{Width: 96, Height: 30},
								Accessibility: Accessibility{Name: "重新加载当前附件"},
							},
						},
					},
					ScrollView{
						AssignTo: &scrollView, StretchFactor: 1,
						MinSize:       Size{Width: 700, Height: 390},
						Layout:        HBox{Margins: Margins{Left: 8, Top: 8, Right: 8, Bottom: 8}},
						Accessibility: Accessibility{Name: "可滚动单据附件预览区域"},
						Children: []Widget{
							ImageView{
								AssignTo: &imageView, Mode: ImageViewModeIdeal, Alignment: AlignHCenterVCenter,
								MinSize: Size{Width: 680, Height: 360}, Margin: 8,
								ToolTipText:   "按住 Ctrl 滚动鼠标滚轮缩放；放大后可使用滚动条查看细节",
								Accessibility: Accessibility{Name: "单据附件图片预览"},
							},
						},
					},
					Composite{
						Layout: HBox{Spacing: 8},
						Children: []Widget{
							PushButton{
								AssignTo: &rotateLeftButton, Text: "向左旋转", Enabled: false,
								MinSize:       Size{Width: 92, Height: 30},
								Accessibility: Accessibility{Name: "附件图片向左旋转九十度"},
							},
							PushButton{
								AssignTo: &rotateRightButton, Text: "向右旋转", Enabled: false,
								MinSize:       Size{Width: 92, Height: 30},
								Accessibility: Accessibility{Name: "附件图片向右旋转九十度"},
							},
							HSpacer{},
						},
					},
				},
			},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					Label{Text: "普通滚轮和滚动条用于浏览；旋转后自动重新适应宽度。", TextColor: walk.RGB(90, 90, 90)},
					HSpacer{},
					PushButton{
						AssignTo: &closeButton, Text: "关闭", MinSize: Size{Width: 88, Height: 30},
						OnClicked: func() { dlg.Accept() },
					},
				},
			},
		},
	}.Create(owner)
	if err != nil {
		cancel()
		walk.MsgBox(owner, "附件窗口错误", err.Error(), walk.MsgBoxIconError)
		return
	}
	if err := enableMaterialDialogSystemButtons(dlg); err != nil {
		cancel()
		dlg.Dispose()
		walk.MsgBox(owner, "附件窗口错误", "无法启用窗口最小化和最大化功能："+err.Error(), walk.MsgBoxIconError)
		return
	}

	dlg.Disposing().Attach(func() {
		closed.Store(true)
		if loadCancel != nil {
			loadCancel()
		}
		cancel()
		_ = clearAttachmentPreview(imageView, currentBitmap)
		currentBitmap = nil
	})

	var loadGeneration int
	var renderGeneration int
	var sourceImage image.Image
	var rotationQuarterTurns int
	var zoomScale float64
	var maxZoomScale = 1.0
	var rendering bool
	setControlsEnabled := func(enabled bool) {
		ready := enabled && sourceImage != nil
		fitButton.SetEnabled(ready)
		actualSizeButton.SetEnabled(ready)
		rotateLeftButton.SetEnabled(ready)
		rotateRightButton.SetEnabled(ready)
	}

	applyRendered := func(rendered image.Image, targetScale float64, generation int, onSuccess func()) {
		dlg.Synchronize(func() {
			if closed.Load() || generation != renderGeneration {
				return
			}
			bitmap, bitmapErr := walk.NewBitmapFromImageForDPI(rendered, dlg.DPI())
			if bitmapErr != nil {
				rendering = false
				statusLabel.SetText("附件显示失败：" + bitmapErr.Error() + "。")
				retryButton.SetVisible(true)
				setControlsEnabled(true)
				return
			}
			if setErr := imageView.SetImage(bitmap); setErr != nil {
				_ = clearAttachmentPreview(imageView, bitmap)
				if currentBitmap != nil {
					currentBitmap.Dispose()
					currentBitmap = nil
				}
				rendering = false
				statusLabel.SetText("附件显示失败：" + setErr.Error() + "。")
				retryButton.SetVisible(true)
				setControlsEnabled(true)
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
			statusLabel.SetText(attachmentReadyStatus(rotationQuarterTurns))
			setControlsEnabled(true)
		})
	}

	var renderZoom func(float64)
	renderZoom = func(targetScale float64) {
		if sourceImage == nil || rendering {
			return
		}
		targetScale = clampMaterialZoom(targetScale, maxZoomScale)
		if math.Abs(targetScale-zoomScale) < 0.001 {
			return
		}
		rendering = true
		renderGeneration++
		generation := renderGeneration
		source := sourceImage
		setControlsEnabled(false)
		statusLabel.SetText("正在生成 " + formatMaterialZoom(targetScale) + " 预览……")
		go func() {
			rendered := scaleMaterialDrawing(source, targetScale)
			if closed.Load() || ctx.Err() != nil {
				return
			}
			applyRendered(rendered, targetScale, generation, nil)
		}()
	}

	fitWidth := func() {
		if sourceImage == nil {
			return
		}
		viewport := scrollView.ClientBoundsPixels()
		renderZoom(materialFitWidthZoom(sourceImage.Bounds(), viewport.Width-24))
	}
	fitButton.Clicked().Attach(fitWidth)
	actualSizeButton.Clicked().Attach(func() { renderZoom(1) })

	rotate := func(direction int) {
		if sourceImage == nil || rendering {
			return
		}
		rendering = true
		renderGeneration++
		generation := renderGeneration
		source := sourceImage
		viewportWidth := scrollView.ClientBoundsPixels().Width - 24
		nextRotation := normalizeMaterialRotation(rotationQuarterTurns + direction)
		setControlsEnabled(false)
		statusLabel.SetText("正在旋转 90°并适应宽度……")
		go func() {
			rotated := rotateMaterialDrawing(source, direction)
			rotatedMaxZoom := materialMaxZoom(rotated.Bounds())
			targetScale := clampMaterialZoom(materialFitWidthZoom(rotated.Bounds(), viewportWidth), rotatedMaxZoom)
			rendered := scaleMaterialDrawing(rotated, targetScale)
			if closed.Load() || ctx.Err() != nil {
				return
			}
			applyRendered(rendered, targetScale, generation, func() {
				sourceImage = rotated
				rotationQuarterTurns = nextRotation
				maxZoomScale = rotatedMaxZoom
			})
		}()
	}
	rotateLeftButton.Clicked().Attach(func() { rotate(-1) })
	rotateRightButton.Clicked().Attach(func() { rotate(1) })
	imageView.MouseWheel().Attach(func(_, _ int, button walk.MouseButton) {
		const mouseWheelControlKey = 0x0008
		if walk.MouseWheelEventKeyState(button)&mouseWheelControlKey == 0 || sourceImage == nil {
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
		index := attachmentCombo.CurrentIndex()
		if index < 0 || index >= len(references) {
			return
		}
		if loadCancel != nil {
			loadCancel()
		}
		loadCtx, cancelLoad := context.WithCancel(ctx)
		loadCancel = cancelLoad
		imageURL, resolveErr := api.ResolveImageURL(imageBaseURL, references[index])
		loadGeneration++
		generation := loadGeneration
		renderGeneration++
		_ = clearAttachmentPreview(imageView, currentBitmap)
		currentBitmap = nil
		sourceImage = nil
		rotationQuarterTurns = 0
		zoomScale = 0
		rendering = false
		zoomLabel.SetText("—")
		setControlsEnabled(false)
		retryButton.SetVisible(false)
		statusLabel.SetText(fmt.Sprintf("正在加载附件 %d / %d 原图……", index+1, len(references)))
		if resolveErr != nil {
			cancelLoad()
			statusLabel.SetText(fmt.Sprintf("附件 %d / %d 地址无效：%s", index+1, len(references), resolveErr.Error()))
			retryButton.SetVisible(true)
			return
		}
		go func() {
			defer cancelLoad()
			data, requestErr := client.DownloadImage(loadCtx, imageURL)
			var decoded image.Image
			if requestErr == nil {
				decoded, _, requestErr = image.Decode(bytes.NewReader(data))
				if requestErr != nil {
					requestErr = fmt.Errorf("无法识别附件图片格式: %w", requestErr)
				}
			}
			if closed.Load() || loadCtx.Err() != nil {
				return
			}
			dlg.Synchronize(func() {
				if closed.Load() || generation != loadGeneration {
					return
				}
				if requestErr != nil {
					statusLabel.SetText(fmt.Sprintf("附件 %d / %d 加载失败：%s。请检查网络或文件格式后重试。",
						index+1, len(references), requestErr.Error()))
					retryButton.SetVisible(true)
					return
				}
				sourceImage = decoded
				maxZoomScale = materialMaxZoom(decoded.Bounds())
				fitWidth()
			})
		}()
	}
	retryButton.Clicked().Attach(loadImage)
	attachmentCombo.CurrentIndexChanged().Attach(loadImage)
	loadImage()
	dlg.Run()
}

func attachmentReadyStatus(rotationQuarterTurns int) string {
	switch normalizeMaterialRotation(rotationQuarterTurns) {
	case 1:
		return "附件已加载 · 已向右旋转 90°"
	case 2:
		return "附件已加载 · 已旋转 180°"
	case 3:
		return "附件已加载 · 已向左旋转 90°"
	default:
		return "附件已加载"
	}
}
