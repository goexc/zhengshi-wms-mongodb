package ui

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"

	"zhengshi-wms-windowsapp/internal/api"
)

type inboundMaterialPickerRow struct {
	Name          string
	Model         string
	Material      string
	Specification string
	Unit          string
	Remark        string
	Detail        api.Material
}

func selectInboundMaterial(owner walk.Form, client *api.Client) (api.Material, bool) {
	return selectBusinessMaterial(owner, client, "入库")
}

func selectOutboundMaterial(owner walk.Form, client *api.Client) (api.Material, bool) {
	return selectBusinessMaterial(owner, client, "出库")
}

func selectBusinessMaterial(owner walk.Form, client *api.Client, purpose string) (api.Material, bool) {
	var dlg *walk.Dialog
	var nameEdit, modelEdit, materialEdit, specificationEdit *walk.LineEdit
	var table *walk.TableView
	var info *walk.Label
	var queryButton, resetButton, prevButton, nextButton, chooseButton, cancelButton *walk.PushButton
	var rows []inboundMaterialPickerRow
	var result api.Material
	var selected bool
	var page = 1
	var total int64
	var generation int
	var requestCancel context.CancelFunc
	ctx, cancel := context.WithCancel(context.Background())
	var closed atomic.Bool
	var load func()

	err := Dialog{
		AssignTo: &dlg,
		Title:    "选择" + purpose + "物料",
		MinSize:  Size{Width: 850, Height: 560},
		Size:     Size{Width: 1040, Height: 680},
		Layout:   VBox{Margins: Margins{Left: 16, Top: 16, Right: 16, Bottom: 14}, Spacing: 10},
		Children: []Widget{
			Label{Text: "选择" + purpose + "物料", Font: Font{Family: "Microsoft YaHei UI", PointSize: 15, Bold: true}},
			Label{Text: "按现有物料查询接口筛选；双击一行可直接选择。", TextColor: secondaryTextColor()},
			GroupBox{
				Title:  "筛选条件",
				Layout: Grid{Columns: 8, Margins: Margins{Left: 12, Top: 10, Right: 12, Bottom: 12}, Spacing: 8},
				Children: []Widget{
					Label{Text: "名称"},
					LineEdit{AssignTo: &nameEdit, CueBanner: "物料名称", Accessibility: Accessibility{Name: "待选物料名称"}},
					Label{Text: "型号"},
					LineEdit{AssignTo: &modelEdit, CueBanner: "物料型号", Accessibility: Accessibility{Name: "待选物料型号"}},
					Label{Text: "材质"},
					LineEdit{AssignTo: &materialEdit, CueBanner: "物料材质", Accessibility: Accessibility{Name: "待选物料材质"}},
					Label{Text: "规格"},
					LineEdit{AssignTo: &specificationEdit, CueBanner: "物料规格", Accessibility: Accessibility{Name: "待选物料规格"}},
					HSpacer{ColumnSpan: 6},
					PushButton{AssignTo: &resetButton, Text: "重置", MinSize: Size{Width: 80, Height: 30}},
					PushButton{AssignTo: &queryButton, Text: "查询", MinSize: Size{Width: 88, Height: 30}},
				},
			},
			TableView{
				AssignTo: &table, Model: rows, AlternatingRowBG: true, ColumnsOrderable: true, StretchFactor: 1,
				Accessibility: Accessibility{Name: "待选" + purpose + "物料列表", Description: "双击当前行选择物料"},
				OnCurrentIndexChanged: func() {
					index := table.CurrentIndex()
					chooseButton.SetEnabled(index >= 0 && index < len(rows))
				},
				OnItemActivated: func() {
					index := table.CurrentIndex()
					if index >= 0 && index < len(rows) {
						result = rows[index].Detail
						selected = true
						dlg.Accept()
					}
				},
				Columns: []TableViewColumn{
					{Title: "名称", DataMember: "Name", Width: 190},
					{Title: "型号", DataMember: "Model", Width: 130},
					{Title: "材质", DataMember: "Material", Width: 120},
					{Title: "规格", DataMember: "Specification", Width: 150},
					{Title: "单位", DataMember: "Unit", Width: 70},
					{Title: "备注", DataMember: "Remark", Width: 180},
				},
			},
			Composite{Layout: HBox{Spacing: 8}, Children: []Widget{
				Label{AssignTo: &info, Text: "尚未加载"},
				HSpacer{},
				PushButton{AssignTo: &prevButton, Text: "上一页", Enabled: false, MinSize: Size{Width: 80, Height: 30}},
				PushButton{AssignTo: &nextButton, Text: "下一页", Enabled: false, MinSize: Size{Width: 80, Height: 30}},
				PushButton{AssignTo: &cancelButton, Text: "取消", MinSize: Size{Width: 80, Height: 30}, OnClicked: func() { dlg.Cancel() }},
				PushButton{AssignTo: &chooseButton, Text: "选择物料", Enabled: false, MinSize: Size{Width: 96, Height: 30}},
			}},
		},
	}.Create(owner)
	if err != nil {
		cancel()
		walk.MsgBox(owner, "无法打开物料选择器", err.Error(), walk.MsgBoxIconError)
		return api.Material{}, false
	}
	dlg.Disposing().Attach(func() {
		closed.Store(true)
		cancel()
		if requestCancel != nil {
			requestCancel()
		}
	})

	setLoading := func(loading bool) {
		queryButton.SetEnabled(!loading)
		resetButton.SetEnabled(!loading)
		prevButton.SetEnabled(!loading && page > 1)
		nextButton.SetEnabled(!loading && int64(page*10) < total)
		if loading {
			chooseButton.SetEnabled(false)
		}
	}
	load = func() {
		if requestCancel != nil {
			requestCancel()
		}
		requestCtx, requestCtxCancel := context.WithCancel(ctx)
		requestCancel = requestCtxCancel
		generation++
		currentGeneration := generation
		filters := api.MaterialFilters{
			Name: nameEdit.Text(), Model: modelEdit.Text(), Material: materialEdit.Text(), Specification: specificationEdit.Text(),
		}
		info.SetText("正在加载线上物料……")
		setLoading(true)
		guardedGo(func() {
			materials, requestErr := client.Materials(requestCtx, page, 10, filters)
			if requestCtx.Err() != nil || closed.Load() {
				return
			}
			dlg.Synchronize(func() {
				if closed.Load() || currentGeneration != generation {
					return
				}
				setLoading(false)
				if requestErr != nil {
					info.SetText("物料加载失败：" + requestErr.Error())
					return
				}
				total = materials.Total
				rows = make([]inboundMaterialPickerRow, 0, len(materials.List))
				for _, material := range materials.List {
					rows = append(rows, inboundMaterialPickerRow{
						Name: material.Name, Model: material.Model, Material: material.Material,
						Specification: material.Specification, Unit: material.Unit, Remark: material.Remark, Detail: material,
					})
				}
				_ = table.SetModel(rows)
				info.SetText(fmt.Sprintf("第 %d 页 | 本页 %d 条 | 共 %d 条", page, len(rows), total))
				prevButton.SetEnabled(page > 1)
				nextButton.SetEnabled(int64(page*10) < total)
			})
		})
	}
	queryButton.Clicked().Attach(func() { page = 1; load() })
	resetButton.Clicked().Attach(func() {
		nameEdit.SetText("")
		modelEdit.SetText("")
		materialEdit.SetText("")
		specificationEdit.SetText("")
		page = 1
		load()
	})
	prevButton.Clicked().Attach(func() {
		if page > 1 {
			page--
			load()
		}
	})
	nextButton.Clicked().Attach(func() {
		if int64(page*10) < total {
			page++
			load()
		}
	})
	chooseButton.Clicked().Attach(func() {
		index := table.CurrentIndex()
		if index < 0 || index >= len(rows) {
			return
		}
		result = rows[index].Detail
		selected = true
		dlg.Accept()
	})
	for _, edit := range []*walk.LineEdit{nameEdit, modelEdit, materialEdit, specificationEdit} {
		current := edit
		current.KeyDown().Attach(func(key walk.Key) {
			if key == walk.KeyReturn {
				page = 1
				load()
			}
		})
	}
	load()
	dlg.Run()
	return result, selected
}

func editInboundMaterialDetails(owner walk.Form, row *inboundEditorMaterial, positions []positionOption) bool {
	var dlg *walk.Dialog
	var priceEdit, quantityEdit *walk.LineEdit
	var positionCombo *walk.ComboBox
	var info *walk.Label
	var saveButton, cancelButton *walk.PushButton
	labels := []string{"不指定计划仓位"}
	for _, position := range positions {
		labels = append(labels, position.Label)
	}
	positionIndex := 0
	for index, position := range positions {
		if stringSlicesEqual(position.IDs, row.Request.Position) {
			positionIndex = index + 1
			break
		}
	}
	err := Dialog{
		AssignTo: &dlg,
		Title:    "编辑入库物料 - " + row.Name,
		MinSize:  Size{Width: 560, Height: 300},
		Size:     Size{Width: 680, Height: 360},
		Layout:   VBox{Margins: Margins{Left: 16, Top: 16, Right: 16, Bottom: 14}, Spacing: 10},
		Children: []Widget{
			Label{Text: row.Name, Font: Font{Family: "Microsoft YaHei UI", PointSize: 14, Bold: true}},
			Label{Text: fmt.Sprintf("型号：%s    单位：%s", displayMaterialValue(row.Model), displayMaterialValue(row.Unit)), TextColor: secondaryTextColor()},
			GroupBox{
				Title:  "本次入库计划",
				Layout: Grid{Columns: 2, Margins: Margins{Left: 12, Top: 10, Right: 12, Bottom: 12}, Spacing: 8},
				Children: []Widget{
					Label{Text: "单价 *"},
					LineEdit{AssignTo: &priceEdit, Text: fmt.Sprintf("%g", row.Request.Price), CueBanner: "非负数字", Accessibility: Accessibility{Name: "物料单价"}},
					Label{Text: "预计数量 *"},
					LineEdit{AssignTo: &quantityEdit, Text: fmt.Sprintf("%g", row.Request.EstimatedQuantity), CueBanner: "大于 0", Accessibility: Accessibility{Name: "物料预计入库数量"}},
					Label{Text: "计划仓位"},
					ComboBox{AssignTo: &positionCombo, Model: labels, CurrentIndex: positionIndex, MinSize: Size{Width: 380, Height: 28}, Accessibility: Accessibility{Name: "物料计划仓位"}},
				},
			},
			Label{AssignTo: &info, Text: "仓位是可选字段；最终合法性仍由服务端校验。", TextColor: secondaryTextColor()},
			VSpacer{},
			Composite{Layout: HBox{Spacing: 8}, Children: []Widget{
				HSpacer{},
				PushButton{AssignTo: &cancelButton, Text: "取消", MinSize: Size{Width: 80, Height: 30}, OnClicked: func() { dlg.Cancel() }},
				PushButton{AssignTo: &saveButton, Text: "应用", MinSize: Size{Width: 88, Height: 30}},
			}},
		},
	}.Create(owner)
	if err != nil {
		walk.MsgBox(owner, "无法编辑物料", err.Error(), walk.MsgBoxIconError)
		return false
	}
	accepted := false
	apply := func() {
		price, parseErr := parsePositiveInboundNumber(priceEdit.Text(), "单价", true)
		if parseErr != nil {
			info.SetText(parseErr.Error())
			priceEdit.SetFocus()
			return
		}
		quantity, parseErr := parsePositiveInboundNumber(quantityEdit.Text(), "预计数量", false)
		if parseErr != nil {
			info.SetText(parseErr.Error())
			quantityEdit.SetFocus()
			return
		}
		row.Request.Price = price
		row.Request.EstimatedQuantity = quantity
		row.Request.Position = nil
		if index := positionCombo.CurrentIndex() - 1; index >= 0 && index < len(positions) {
			row.Request.Position = append([]string(nil), positions[index].IDs...)
		}
		accepted = true
		dlg.Accept()
	}
	saveButton.Clicked().Attach(apply)
	priceEdit.KeyDown().Attach(func(key walk.Key) {
		if key == walk.KeyReturn {
			quantityEdit.SetFocus()
		}
	})
	quantityEdit.KeyDown().Attach(func(key walk.Key) {
		if key == walk.KeyReturn {
			apply()
		}
	})
	dlg.Run()
	return accepted
}
