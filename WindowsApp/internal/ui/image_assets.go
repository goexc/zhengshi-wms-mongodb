package ui

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"strings"
	"sync/atomic"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"

	"zhengshi-wms-windowsapp/internal/api"
	"zhengshi-wms-windowsapp/internal/config"
)

type imageAssetRow struct {
	Name      string
	Reference string
	Detail    api.ImageAsset
}

type imageAssetUI struct {
	name          *walk.LineEdit
	table         *walk.TableView
	preview       *walk.ImageView
	previewStatus *walk.Label
	info          *walk.Label
	query         *walk.PushButton
	reset         *walk.PushButton
	upload        *walk.PushButton
	uploadStop    *walk.PushButton
	fullPreview   *walk.PushButton
	copyReference *walk.PushButton
	prev          *walk.PushButton
	next          *walk.PushButton
	size          *walk.ComboBox

	page              int
	total             int64
	rows              []imageAssetRow
	generation        int
	cancel            context.CancelFunc
	previewGeneration int
	previewCancel     context.CancelFunc
	previewBitmap     attachmentPreviewResource
	uploadCancel      context.CancelFunc
	busy              bool
}

func newImageAssetUI() *imageAssetUI {
	return &imageAssetUI{page: 1}
}

func imageAssetMenuAvailable(perms api.Perms) bool {
	return hasAnyMenuPath(perms.Menus, "/image", "/image/index", "/images") ||
		hasMenuName(perms.Menus, "素材") || hasMenuName(perms.Menus, "图片")
}

func hasMenuName(menus []api.Menu, namePart string) bool {
	namePart = strings.TrimSpace(namePart)
	if namePart == "" {
		return false
	}
	for _, menu := range menus {
		if strings.Contains(strings.TrimSpace(menu.Name), namePart) || hasMenuName(menu.Children, namePart) {
			return true
		}
	}
	return false
}

func (ui *mainUI) imageAssetPageWidget() TabPage {
	state := ui.imageAssets
	return TabPage{
		AssignTo: &ui.imageAssetTab,
		Title:    closableTabTitle("图片素材"),
		Layout:   VBox{Margins: Margins{Left: 14, Top: 14, Right: 14, Bottom: 12}, Spacing: 10},
		Children: []Widget{
			Label{Text: "图片素材", Font: Font{Family: "Microsoft YaHei UI", PointSize: 15, Bold: true}},
			Label{
				Text:          "查询、上传和复用线上图片；本客户端不提供素材删除，避免破坏历史业务引用。",
				TextColor:     secondaryTextColor(),
				Accessibility: Accessibility{Name: "图片素材页面说明"},
			},
			GroupBox{
				Title:  "筛选条件",
				Layout: Grid{Columns: 4, Margins: Margins{Left: 14, Top: 10, Right: 14, Bottom: 12}, Spacing: 8},
				Children: []Widget{
					Label{Text: "素材名称"},
					LineEdit{
						AssignTo: &state.name, MinSize: Size{Width: 240, Height: 28}, CueBanner: "最多 20 个字符",
						Accessibility: Accessibility{Name: "图片素材名称筛选"},
					},
					Composite{ColumnSpan: 2, Layout: HBox{Spacing: 8}, Children: []Widget{
						HSpacer{},
						PushButton{AssignTo: &state.reset, Text: "重置", MinSize: Size{Width: 88, Height: 30}, OnClicked: ui.resetImageAssetFilters},
						PushButton{AssignTo: &state.query, Text: "查询", MinSize: Size{Width: 96, Height: 30}, OnClicked: func() {
							state.page = 1
							ui.loadImageAssets()
						}},
					}},
				},
			},
			Composite{Layout: HBox{Spacing: 8}, Children: []Widget{
				PushButton{AssignTo: &state.upload, Text: "上传新图片", MinSize: Size{Width: 104, Height: 30}, OnClicked: ui.uploadImageAsset},
				PushButton{AssignTo: &state.uploadStop, Text: "取消上传", Enabled: false, MinSize: Size{Width: 92, Height: 30}, Accessibility: Accessibility{Name: "取消当前图片素材上传"}, OnClicked: ui.cancelImageAssetUpload},
				PushButton{AssignTo: &state.fullPreview, Text: "查看原图", Enabled: false, MinSize: Size{Width: 92, Height: 30}, OnClicked: ui.previewSelectedImageAsset},
				PushButton{AssignTo: &state.copyReference, Text: "复制引用", Enabled: false, MinSize: Size{Width: 92, Height: 30}, OnClicked: ui.copySelectedImageAssetReference},
				HSpacer{},
				Label{Text: "选择素材后可在右侧预览；双击列表查看原图。", TextColor: secondaryTextColor()},
			}},
			Composite{
				StretchFactor: 1,
				Layout:        HBox{Spacing: 10},
				Children: []Widget{
					TableView{
						AssignTo: &state.table, Model: []imageAssetRow{}, AlternatingRowBG: true,
						ColumnsOrderable: true, StretchFactor: 2,
						Accessibility:         Accessibility{Name: "图片素材查询结果", Description: "选择一行查看缩略预览，双击查看原图"},
						OnCurrentIndexChanged: ui.updateImageAssetSelection,
						OnItemActivated:       ui.previewSelectedImageAsset,
						Columns: []TableViewColumn{
							{Title: "素材名称", DataMember: "Name", Width: 240},
							{Title: "图片引用", DataMember: "Reference", Width: 420},
						},
					},
					GroupBox{
						Title: "选中素材预览", MinSize: Size{Width: 360, Height: 320},
						Layout: VBox{Margins: Margins{Left: 10, Top: 10, Right: 10, Bottom: 10}, Spacing: 8},
						Children: []Widget{
							Label{AssignTo: &state.previewStatus, Text: "请选择一张图片素材。", TextColor: secondaryTextColor(), Accessibility: Accessibility{Name: "素材缩略图加载状态"}},
							ImageView{
								AssignTo: &state.preview, Mode: ImageViewModeIdeal, Alignment: AlignHCenterVCenter,
								MinSize: Size{Width: 330, Height: 260}, StretchFactor: 1,
								Accessibility: Accessibility{Name: "选中图片素材缩略预览"},
							},
						},
					},
				},
			},
			Composite{Layout: HBox{Spacing: 8}, Children: []Widget{
				Label{AssignTo: &state.info, Text: "尚未加载", Accessibility: Accessibility{Name: "图片素材列表状态"}},
				HSpacer{},
				Label{Text: "每页"},
				ComboBox{
					AssignTo: &state.size, Model: pageSizeLabels, CurrentIndex: 1, MinSize: Size{Width: 92},
					OnCurrentIndexChanged: func() {
						if ui.window != nil && state.size != nil && state.size.CurrentIndex() >= 0 {
							state.page = 1
							ui.loadImageAssets()
						}
					},
				},
				PushButton{AssignTo: &state.prev, Text: "上一页", OnClicked: func() {
					if state.page > 1 {
						state.page--
						ui.loadImageAssets()
					}
				}},
				PushButton{AssignTo: &state.next, Text: "下一页", OnClicked: func() {
					state.page++
					ui.loadImageAssets()
				}},
				PushButton{Text: "跳转页", Accessibility: Accessibility{Name: "跳转到指定图片素材结果页"}, OnClicked: func() {
					if page, ok := promptPageNumber(ui.window, state.page, state.total, selectedPageSize(state.size)); ok {
						state.page = page
						ui.loadImageAssets()
					}
				}},
			}},
		},
	}
}

func (ui *mainUI) initializeImageAssetPage() {
	state := ui.imageAssets
	if ui.imageAssetTab == nil || state == nil || state.name == nil {
		return
	}
	state.generation++
	state.name.KeyDown().Attach(func(key walk.Key) {
		if key == walk.KeyReturn {
			state.page = 1
			ui.loadImageAssets()
		}
	})
	ui.updateImageAssetSelection()
	ui.loadImageAssets()
}

func (ui *mainUI) releaseImageAssetPage() {
	state := ui.imageAssets
	if state != nil {
		state.generation++
		state.previewGeneration++
		if state.cancel != nil {
			state.cancel()
		}
		if state.previewCancel != nil {
			state.previewCancel()
		}
		if state.uploadCancel != nil {
			state.uploadCancel()
		}
		if state.preview != nil {
			_ = clearAttachmentPreview(state.preview, state.previewBitmap)
		} else if state.previewBitmap != nil {
			state.previewBitmap.Dispose()
		}
		state.previewBitmap = nil
	}
	ui.imageAssetTab = nil
	ui.imageAssets = newImageAssetUI()
}

func (ui *mainUI) resetImageAssetFilters() {
	state := ui.imageAssets
	if state == nil || state.name == nil {
		return
	}
	state.name.SetText("")
	state.page = 1
	ui.loadImageAssets()
}

func (ui *mainUI) loadImageAssets() {
	state := ui.imageAssets
	if state == nil || state.table == nil || state.name == nil {
		return
	}
	if state.cancel != nil {
		state.cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	state.cancel = cancel
	state.generation++
	generation := state.generation
	page := state.page
	size := selectedPageSize(state.size)
	name := state.name.Text()
	state.busy = true
	state.info.SetText("正在加载线上图片素材……")
	state.query.SetEnabled(false)
	state.reset.SetEnabled(false)
	state.upload.SetEnabled(false)
	state.prev.SetEnabled(false)
	state.next.SetEnabled(false)
	ui.updateImageAssetSelection()

	guardedGo(func() {
		result, requestErr := ui.session.Client.ImageAssets(ctx, page, size, name)
		if ctx.Err() != nil {
			return
		}
		ui.window.Synchronize(func() {
			if state != ui.imageAssets || generation != state.generation || state.table == nil {
				return
			}
			state.busy = false
			state.query.SetEnabled(true)
			state.reset.SetEnabled(true)
			state.upload.SetEnabled(true)
			if requestErr != nil {
				state.info.SetText("素材加载失败：" + requestFailureText(requestErr))
				ui.updateImageAssetSelection()
				return
			}
			rows := make([]imageAssetRow, 0, len(result.List))
			for _, item := range result.List {
				rows = append(rows, imageAssetRow{Name: displayMaterialValue(item.Alt), Reference: item.URL, Detail: item})
			}
			if err := state.table.SetModel(rows); err != nil {
				state.info.SetText("素材列表展示失败：" + err.Error())
				return
			}
			state.rows = rows
			state.total = result.Total
			state.info.SetText(fmt.Sprintf("第 %d 页 | 本页 %d 张 | 共 %d 张", page, len(rows), result.Total))
			state.prev.SetEnabled(page > 1)
			state.next.SetEnabled(int64(page*size) < result.Total)
			ui.updateImageAssetSelection()
		})
	})
}

func (ui *mainUI) selectedImageAsset() (api.ImageAsset, bool) {
	state := ui.imageAssets
	if state == nil || state.table == nil {
		return api.ImageAsset{}, false
	}
	index := state.table.CurrentIndex()
	if index < 0 || index >= len(state.rows) {
		return api.ImageAsset{}, false
	}
	return state.rows[index].Detail, true
}

func (ui *mainUI) updateImageAssetSelection() {
	state := ui.imageAssets
	if state == nil || state.fullPreview == nil {
		return
	}
	_, selected := ui.selectedImageAsset()
	state.fullPreview.SetEnabled(selected && !state.busy)
	state.copyReference.SetEnabled(selected && !state.busy)
	ui.loadSelectedImageAssetThumbnail()
}

func (ui *mainUI) loadSelectedImageAssetThumbnail() {
	state := ui.imageAssets
	if state == nil || state.preview == nil || state.previewStatus == nil {
		return
	}
	if state.previewCancel != nil {
		state.previewCancel()
	}
	state.previewGeneration++
	generation := state.previewGeneration
	_ = clearAttachmentPreview(state.preview, state.previewBitmap)
	state.previewBitmap = nil
	asset, ok := ui.selectedImageAsset()
	if !ok {
		state.previewStatus.SetText("请选择一张图片素材。")
		return
	}
	imageURL, err := api.ResolveImageURL(config.ImageBaseURL(), asset.URL)
	if err != nil {
		state.previewStatus.SetText("图片地址无效：" + err.Error())
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	state.previewCancel = cancel
	state.previewStatus.SetText("正在加载缩略预览……")
	guardedGo(func() {
		defer cancel()
		data, requestErr := ui.session.Client.DownloadImage(ctx, imageURL)
		var decoded image.Image
		if requestErr == nil {
			decoded, _, requestErr = image.Decode(bytes.NewReader(data))
		}
		if ctx.Err() != nil {
			return
		}
		ui.window.Synchronize(func() {
			if state != ui.imageAssets || generation != state.previewGeneration || state.preview == nil {
				return
			}
			if requestErr != nil {
				state.previewStatus.SetText("缩略预览加载失败：" + requestErr.Error())
				return
			}
			targetScale := materialFitWidthZoom(decoded.Bounds(), 320)
			rendered := scaleMaterialDrawing(decoded, targetScale)
			bitmap, bitmapErr := walk.NewBitmapFromImageForDPI(rendered, ui.window.DPI())
			if bitmapErr != nil {
				state.previewStatus.SetText("缩略预览生成失败：" + bitmapErr.Error())
				return
			}
			if setErr := state.preview.SetImage(bitmap); setErr != nil {
				bitmap.Dispose()
				state.previewStatus.SetText("缩略预览显示失败：" + setErr.Error())
				return
			}
			if state.previewBitmap != nil {
				state.previewBitmap.Dispose()
			}
			state.previewBitmap = bitmap
			state.previewStatus.SetText(displayMaterialValue(asset.Alt) + " · 点击“查看原图”检查细节")
		})
	})
}

func (ui *mainUI) previewSelectedImageAsset() {
	asset, ok := ui.selectedImageAsset()
	if !ok {
		walk.MsgBox(ui.window, "请选择素材", "请先在素材列表中选择一张图片。", walk.MsgBoxIconInformation)
		return
	}
	ShowOrderAttachments(ui.window, ui.session.Client, config.ImageBaseURL(), "图片素材 "+displayMaterialValue(asset.Alt), []string{asset.URL})
}

func (ui *mainUI) copySelectedImageAssetReference() {
	asset, ok := ui.selectedImageAsset()
	if !ok {
		return
	}
	if err := walk.Clipboard().SetText(strings.TrimSpace(asset.URL)); err != nil {
		walk.MsgBox(ui.window, "复制失败", err.Error(), walk.MsgBoxIconError)
		return
	}
	ui.imageAssets.info.SetText("已复制图片引用：" + asset.URL)
}

func (ui *mainUI) uploadImageAsset() {
	state := ui.imageAssets
	if state == nil || state.busy {
		return
	}
	dialog := newImageOpenDialog()
	accepted, err := dialog.ShowOpen(ui.window)
	if err != nil {
		state.info.SetText("选择图片失败：" + err.Error())
		return
	}
	if !accepted {
		return
	}
	state.busy = true
	state.upload.SetEnabled(false)
	state.uploadStop.SetEnabled(true)
	state.query.SetEnabled(false)
	state.info.SetText("正在上传图片素材……")
	filePath := dialog.FilePath
	ctx, cancel := context.WithCancel(context.Background())
	state.uploadCancel = cancel
	guardedGo(func() {
		defer cancel()
		reference, requestErr := ui.session.Client.UploadImage(ctx, filePath)
		if ctx.Err() != nil {
			return
		}
		ui.window.Synchronize(func() {
			if state != ui.imageAssets || state.info == nil {
				return
			}
			state.uploadCancel = nil
			state.busy = false
			state.upload.SetEnabled(true)
			state.uploadStop.SetEnabled(false)
			state.query.SetEnabled(true)
			if requestErr != nil {
				state.info.SetText("图片上传失败：" + requestFailureText(requestErr))
				return
			}
			state.info.SetText("图片上传成功：" + reference)
			state.page = 1
			ui.loadImageAssets()
		})
	})
}

func (ui *mainUI) cancelImageAssetUpload() {
	state := ui.imageAssets
	if state == nil || state.uploadCancel == nil {
		return
	}
	state.uploadCancel()
	state.uploadCancel = nil
	state.busy = false
	if state.upload != nil {
		state.upload.SetEnabled(true)
	}
	if state.uploadStop != nil {
		state.uploadStop.SetEnabled(false)
	}
	if state.query != nil {
		state.query.SetEnabled(true)
	}
	if state.info != nil {
		state.info.SetText("图片上传已取消；线上结果可能已产生，请刷新素材列表或在“操作复核”中确认。")
	}
}

func newImageOpenDialog() *walk.FileDialog {
	return &walk.FileDialog{
		Title:  "选择图片",
		Filter: "图片文件 (*.png;*.jpg;*.jpeg;*.gif)|*.png;*.jpg;*.jpeg;*.gif|所有文件 (*.*)|*.*",
	}
}

type imageAssetPickerRow struct {
	Name      string
	Reference string
	Detail    api.ImageAsset
}

func SelectImageAssets(owner walk.Form, client *api.Client, imageBaseURL string, limit int) ([]string, bool) {
	if limit <= 0 {
		limit = 1
	}
	var dlg *walk.Dialog
	var name *walk.LineEdit
	var table *walk.TableView
	var info *walk.Label
	var query, reset, preview, prev, next, confirm, cancelButton *walk.PushButton
	var size *walk.ComboBox
	var rows []imageAssetPickerRow
	var page = 1
	var total int64
	var generation int
	var requestCancel context.CancelFunc
	var closed atomic.Bool
	var selected []string

	updateActions := func() {}
	err := Dialog{
		AssignTo: &dlg, Title: "选择图片素材", DefaultButton: &confirm, CancelButton: &cancelButton,
		MinSize: Size{Width: 760, Height: 500}, Size: Size{Width: 940, Height: 640},
		Layout: VBox{Margins: Margins{Left: 16, Top: 16, Right: 16, Bottom: 14}, Spacing: 10},
		Children: []Widget{
			Label{Text: "选择已有图片素材", Font: Font{Family: "Microsoft YaHei UI", PointSize: 15, Bold: true}},
			Label{Text: fmt.Sprintf("可选择当前页中的图片，最多 %d 张；不会删除或修改服务端素材。", limit), TextColor: secondaryTextColor()},
			GroupBox{Title: "筛选条件", Layout: Grid{Columns: 4, Margins: Margins{Left: 12, Top: 10, Right: 12, Bottom: 12}, Spacing: 8}, Children: []Widget{
				Label{Text: "素材名称"},
				LineEdit{AssignTo: &name, MinSize: Size{Width: 240, Height: 28}, CueBanner: "最多 20 个字符", Accessibility: Accessibility{Name: "选择素材名称筛选"}},
				Composite{ColumnSpan: 2, Layout: HBox{Spacing: 8}, Children: []Widget{
					HSpacer{},
					PushButton{AssignTo: &reset, Text: "重置", MinSize: Size{Width: 82, Height: 30}},
					PushButton{AssignTo: &query, Text: "查询", MinSize: Size{Width: 92, Height: 30}},
				}},
			}},
			TableView{
				AssignTo: &table, Model: []imageAssetPickerRow{}, AlternatingRowBG: true, ColumnsOrderable: true,
				MultiSelection: limit > 1, StretchFactor: 1,
				Accessibility:            Accessibility{Name: "可选择的图片素材", Description: "选择一行或多行后确认"},
				OnCurrentIndexChanged:    func() { updateActions() },
				OnSelectedIndexesChanged: func() { updateActions() },
				Columns: []TableViewColumn{
					{Title: "素材名称", DataMember: "Name", Width: 250},
					{Title: "图片引用", DataMember: "Reference", Width: 520},
				},
			},
			Composite{Layout: HBox{Spacing: 8}, Children: []Widget{
				Label{AssignTo: &info, Text: "尚未加载", Accessibility: Accessibility{Name: "素材选择列表状态"}},
				HSpacer{},
				PushButton{AssignTo: &preview, Text: "预览当前项", Enabled: false, MinSize: Size{Width: 104, Height: 30}},
				Label{Text: "每页"},
				ComboBox{AssignTo: &size, Model: pageSizeLabels, CurrentIndex: 1, MinSize: Size{Width: 92}},
				PushButton{AssignTo: &prev, Text: "上一页"},
				PushButton{AssignTo: &next, Text: "下一页"},
			}},
			Composite{Layout: HBox{Spacing: 8}, Children: []Widget{
				Label{Text: "选择结果只写入当前表单，保存业务单据后才会生效。", TextColor: secondaryTextColor()},
				HSpacer{},
				PushButton{AssignTo: &cancelButton, Text: "取消", MinSize: Size{Width: 82, Height: 30}, OnClicked: func() { dlg.Cancel() }},
				PushButton{AssignTo: &confirm, Text: "使用所选素材", Enabled: false, MinSize: Size{Width: 112, Height: 30}},
			}},
		},
	}.Create(owner)
	if err != nil {
		walk.MsgBox(owner, "素材选择窗口错误", err.Error(), walk.MsgBoxIconError)
		return nil, false
	}
	dlg.Disposing().Attach(func() {
		closed.Store(true)
		if requestCancel != nil {
			requestCancel()
		}
	})

	selectedIndexes := func() []int {
		if limit > 1 {
			return table.SelectedIndexes()
		}
		if index := table.CurrentIndex(); index >= 0 {
			return []int{index}
		}
		return nil
	}
	updateActions = func() {
		indexes := selectedIndexes()
		valid := 0
		for _, index := range indexes {
			if index >= 0 && index < len(rows) {
				valid++
			}
		}
		preview.SetEnabled(table.CurrentIndex() >= 0 && table.CurrentIndex() < len(rows))
		confirm.SetEnabled(valid > 0 && valid <= limit)
		if valid > limit {
			info.SetText(fmt.Sprintf("已选择 %d 张，最多只能选择 %d 张。", valid, limit))
		}
	}

	load := func() {}
	load = func() {
		if requestCancel != nil {
			requestCancel()
		}
		ctx, cancel := context.WithCancel(context.Background())
		requestCancel = cancel
		generation++
		currentGeneration := generation
		currentPage := page
		currentSize := selectedPageSize(size)
		info.SetText("正在加载线上图片素材……")
		query.SetEnabled(false)
		reset.SetEnabled(false)
		prev.SetEnabled(false)
		next.SetEnabled(false)
		confirm.SetEnabled(false)
		guardedGo(func() {
			result, requestErr := client.ImageAssets(ctx, currentPage, currentSize, name.Text())
			if ctx.Err() != nil || closed.Load() {
				return
			}
			dlg.Synchronize(func() {
				if closed.Load() || currentGeneration != generation {
					return
				}
				query.SetEnabled(true)
				reset.SetEnabled(true)
				if requestErr != nil {
					info.SetText("素材加载失败：" + requestFailureText(requestErr))
					return
				}
				rows = make([]imageAssetPickerRow, 0, len(result.List))
				for _, item := range result.List {
					rows = append(rows, imageAssetPickerRow{Name: displayMaterialValue(item.Alt), Reference: item.URL, Detail: item})
				}
				if modelErr := table.SetModel(rows); modelErr != nil {
					info.SetText("素材列表展示失败：" + modelErr.Error())
					return
				}
				total = result.Total
				info.SetText(fmt.Sprintf("第 %d 页 | 本页 %d 张 | 共 %d 张", currentPage, len(rows), total))
				prev.SetEnabled(currentPage > 1)
				next.SetEnabled(int64(currentPage*currentSize) < total)
				updateActions()
			})
		})
	}

	query.Clicked().Attach(func() { page = 1; load() })
	reset.Clicked().Attach(func() { name.SetText(""); page = 1; load() })
	name.KeyDown().Attach(func(key walk.Key) {
		if key == walk.KeyReturn {
			page = 1
			load()
		}
	})
	size.CurrentIndexChanged().Attach(func() {
		if !closed.Load() && size.CurrentIndex() >= 0 {
			page = 1
			load()
		}
	})
	prev.Clicked().Attach(func() {
		if page > 1 {
			page--
			load()
		}
	})
	next.Clicked().Attach(func() { page++; load() })
	preview.Clicked().Attach(func() {
		index := table.CurrentIndex()
		if index >= 0 && index < len(rows) {
			ShowOrderAttachments(dlg, client, imageBaseURL, "图片素材 "+rows[index].Name, []string{rows[index].Reference})
		}
	})
	confirm.Clicked().Attach(func() {
		indexes := selectedIndexes()
		if len(indexes) == 0 || len(indexes) > limit {
			updateActions()
			return
		}
		selected = selected[:0]
		for _, index := range indexes {
			if index >= 0 && index < len(rows) {
				selected = append(selected, rows[index].Reference)
			}
		}
		if len(selected) > 0 {
			dlg.Accept()
		}
	})
	load()
	if dlg.Run() != walk.DlgCmdOK {
		return nil, false
	}
	return append([]string(nil), selected...), true
}
