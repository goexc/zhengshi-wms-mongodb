package ui

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"

	"zhengshi-wms-windowsapp/internal/api"
)

type outboundReviseRow struct {
	Index        string
	Name         string
	Model        string
	Quantity     string
	CurrentPrice string
	NewPrice     string
	Amount       string
	Changed      string
	Material     api.OutboundMaterial
	NewValue     float64
}

func (ui *mainUI) reviseSelectedOutbound() {
	order, ok := ui.selectedOutbound()
	if !ok {
		return
	}
	if strings.TrimSpace(order.CustomerID) == "" {
		walk.MsgBox(ui.window, "无法核价", "该出库单没有客户，现有核价接口要求客户编号。", walk.MsgBoxIconWarning)
		return
	}
	if ShowOutboundRevise(ui.window, ui.session.Client, order) {
		ui.loadOutbound()
	}
}

func ShowOutboundRevise(owner walk.Form, client *api.Client, snapshot api.OutboundOrder) bool {
	var dlg *walk.Dialog
	var table *walk.TableView
	var statusLabel, selectedLabel, historyStatus, totalLabel *walk.Label
	var priceEdit *walk.LineEdit
	var historyCombo *walk.ComboBox
	var applyButton, reloadButton, submitButton, cancelButton *walk.PushButton
	var rows []outboundReviseRow
	var baselineOrder api.OutboundOrder
	var baselineMaterials []api.OutboundMaterial
	var historyPrices []api.MaterialPrice
	var historyLoading bool
	var loadGeneration, historyGeneration int
	var historyCancel context.CancelFunc
	var operationBusy bool
	var allowClose bool
	var success bool
	var closed atomic.Bool
	ctx, cancel := context.WithCancel(context.Background())

	var loadOrder func()
	var updateSelection func()
	var updateTotals func()
	var applySelected func(bool) bool

	err := Dialog{
		AssignTo: &dlg,
		Title:    "出库核价/调价 - " + snapshot.Code,
		MinSize:  Size{Width: 900, Height: 600},
		Size:     Size{Width: 1080, Height: 720},
		Layout:   VBox{Margins: Margins{Left: 16, Top: 16, Right: 16, Bottom: 14}, Spacing: 10},
		Children: []Widget{
			Label{Text: "出库核价/调价", Font: Font{Family: "Microsoft YaHei UI", PointSize: 15, Bold: true}},
			Label{
				Text:      fmt.Sprintf("出库单：%s    客户：%s    当前状态：%s", snapshot.Code, displayMaterialValue(snapshot.CustomerName), snapshot.Status),
				TextColor: walk.RGB(70, 70, 70),
			},
			Label{
				AssignTo: &statusLabel, Text: "正在重新读取线上订单和物料……",
				TextColor: walk.RGB(80, 80, 80), Accessibility: Accessibility{Name: "出库核价加载和提交状态"},
			},
			TableView{
				AssignTo: &table, Model: []outboundReviseRow{}, AlternatingRowBG: true,
				ColumnsOrderable: true, StretchFactor: 1,
				Accessibility: Accessibility{Name: "出库物料核价列表", Description: "选择物料后在下方编辑新单价"},
				OnCurrentIndexChanged: func() {
					if updateSelection != nil {
						updateSelection()
					}
				},
				OnItemActivated: func() {
					if priceEdit != nil && priceEdit.Enabled() {
						priceEdit.SetFocus()
					}
				},
				Columns: []TableViewColumn{
					{Title: "序号", DataMember: "Index", Width: 55},
					{Title: "物料", DataMember: "Name", Width: 190},
					{Title: "型号", DataMember: "Model", Width: 110},
					{Title: "数量", DataMember: "Quantity", Width: 95},
					{Title: "当前单价", DataMember: "CurrentPrice", Width: 95},
					{Title: "新单价", DataMember: "NewPrice", Width: 95},
					{Title: "新金额", DataMember: "Amount", Width: 105},
					{Title: "变更", DataMember: "Changed", Width: 65},
				},
			},
			GroupBox{
				Title:  "编辑当前物料",
				Layout: Grid{Columns: 4, Margins: Margins{Left: 12, Top: 10, Right: 12, Bottom: 12}, Spacing: 8},
				Children: []Widget{
					Label{AssignTo: &selectedLabel, Text: "请选择一条物料", ColumnSpan: 4, Font: Font{Bold: true}},
					Label{Text: "新单价"},
					LineEdit{
						AssignTo: &priceEdit, Enabled: false, CueBanner: "非负数字，最多保留三位显示",
						Accessibility: Accessibility{Name: "当前物料新单价"},
					},
					Label{Text: "历史参考价"},
					ComboBox{
						AssignTo: &historyCombo, Model: []string{"选择物料后加载"}, CurrentIndex: 0, Enabled: false,
						Accessibility: Accessibility{Name: "当前物料历史参考价"},
					},
					Label{AssignTo: &historyStatus, Text: "历史价格按当前客户读取，仅作填写参考。", ColumnSpan: 3, TextColor: walk.RGB(85, 85, 85)},
					PushButton{
						AssignTo: &applyButton, Text: "应用到当前物料", Enabled: false, MinSize: Size{Width: 130, Height: 30},
						OnClicked: func() {
							if applySelected != nil {
								applySelected(true)
							}
						},
					},
				},
			},
			Composite{Layout: HBox{Spacing: 8}, Children: []Widget{
				Label{AssignTo: &totalLabel, Text: "物料金额：—"},
				HSpacer{},
				PushButton{AssignTo: &reloadButton, Text: "重新读取", MinSize: Size{Width: 92, Height: 30}},
				PushButton{AssignTo: &cancelButton, Text: "取消", MinSize: Size{Width: 80, Height: 30}, OnClicked: func() { dlg.Cancel() }},
				PushButton{AssignTo: &submitButton, Text: "核对并提交", Enabled: false, MinSize: Size{Width: 110, Height: 30}},
			}},
		},
	}.Create(owner)
	if err != nil {
		cancel()
		walk.MsgBox(owner, "核价窗口错误", err.Error(), walk.MsgBoxIconError)
		return false
	}
	dlg.Disposing().Attach(func() {
		closed.Store(true)
		cancel()
		if historyCancel != nil {
			historyCancel()
		}
	})
	dlg.Closing().Attach(func(canceled *bool, _ walk.CloseReason) {
		if allowClose {
			return
		}
		if operationBusy {
			*canceled = true
			walk.MsgBox(dlg, "正在提交", "核价请求正在提交和复核，请等待结果，避免产生未知状态。", walk.MsgBoxIconInformation)
			return
		}
		if outboundRevisionDirty(rows) && walk.MsgBox(
			dlg, "放弃未提交修改", "当前价格修改尚未提交，是否放弃？", walk.MsgBoxYesNo|walk.MsgBoxIconWarning,
		) != walk.DlgCmdYes {
			*canceled = true
		}
	})

	setEditorEnabled := func(enabled bool) {
		priceEdit.SetEnabled(enabled && !operationBusy)
		applyButton.SetEnabled(enabled && !operationBusy)
		historyCombo.SetEnabled(enabled && !operationBusy && len(historyPrices) > 0)
	}
	updateTotals = func() {
		oldTotal, newTotal := outboundRevisionTotals(rows)
		totalLabel.SetText(fmt.Sprintf("物料金额：原 %.3f    新 %.3f    差额 %+.3f", oldTotal, newTotal, newTotal-oldTotal))
		submitButton.SetEnabled(len(rows) > 0 && outboundRevisionDirty(rows) && !operationBusy)
	}

	loadPriceHistory := func(materialID, customerID string) {
		if historyCancel != nil {
			historyCancel()
		}
		historyCtx, historyCtxCancel := context.WithCancel(ctx)
		historyCancel = historyCtxCancel
		historyGeneration++
		generation := historyGeneration
		historyLoading = true
		historyPrices = nil
		_ = historyCombo.SetModel([]string{"正在加载历史价格……"})
		_ = historyCombo.SetCurrentIndex(0)
		historyCombo.SetEnabled(false)
		historyStatus.SetText("正在读取当前客户的有效历史价格……")
		go func() {
			prices, requestErr := client.MaterialPrices(historyCtx, materialID, customerID)
			if historyCtx.Err() != nil || closed.Load() {
				return
			}
			dlg.Synchronize(func() {
				if closed.Load() || generation != historyGeneration {
					return
				}
				historyLoading = true
				if requestErr != nil {
					historyPrices = nil
					_ = historyCombo.SetModel([]string{"历史价格加载失败"})
					_ = historyCombo.SetCurrentIndex(0)
					historyCombo.SetEnabled(false)
					historyStatus.SetText("历史价格加载失败：" + requestErr.Error() + "。仍可人工输入新单价。")
					historyLoading = false
					return
				}
				validPrices := make([]api.MaterialPrice, 0, len(prices))
				for _, price := range prices {
					if price.SourceValid {
						validPrices = append(validPrices, price)
					}
				}
				sort.SliceStable(validPrices, func(i, j int) bool { return validPrices[i].Since > validPrices[j].Since })
				historyPrices = validPrices
				labels := []string{"选择后填入新单价"}
				for _, price := range historyPrices {
					labels = append(labels, fmt.Sprintf("%.3f · %s · %s · %s",
						price.Price, displayMaterialValue(price.CustomerName), formatUnixMinute(price.Since), materialPriceSourceLabel(price.SourceType)))
				}
				if len(historyPrices) == 0 {
					labels = []string{"暂无有效历史价格"}
				}
				_ = historyCombo.SetModel(labels)
				_ = historyCombo.SetCurrentIndex(0)
				historyCombo.SetEnabled(len(historyPrices) > 0 && !operationBusy)
				historyStatus.SetText(fmt.Sprintf("已加载 %d 条有效历史价格；选择后仍需点击“应用到当前物料”。", len(historyPrices)))
				historyLoading = false
			})
		}()
	}

	updateSelection = func() {
		index := table.CurrentIndex()
		if index < 0 || index >= len(rows) {
			selectedLabel.SetText("请选择一条物料")
			priceEdit.SetText("")
			historyPrices = nil
			_ = historyCombo.SetModel([]string{"选择物料后加载"})
			_ = historyCombo.SetCurrentIndex(0)
			setEditorEnabled(false)
			return
		}
		row := rows[index]
		selectedLabel.SetText(fmt.Sprintf("%s · %s · 数量 %s", row.Name, displayMaterialValue(row.Model), row.Quantity))
		priceEdit.SetText(formatOutboundRevisionPrice(row.NewValue))
		setEditorEnabled(true)
		loadPriceHistory(row.Material.MaterialID, baselineOrder.CustomerID)
	}
	applySelected = func(showFeedback bool) bool {
		index := table.CurrentIndex()
		if index < 0 || index >= len(rows) {
			if showFeedback {
				walk.MsgBox(dlg, "请选择物料", "请先在列表中选择需要核价的物料。", walk.MsgBoxIconInformation)
			}
			return false
		}
		value, parseErr := parseOutboundRevisionPrice(priceEdit.Text())
		if parseErr != nil {
			statusLabel.SetText("新单价格式错误：" + parseErr.Error())
			priceEdit.SetFocus()
			return false
		}
		rows[index].NewValue = value
		refreshOutboundRevisionRow(&rows[index])
		if modelErr := table.SetModel(rows); modelErr != nil {
			statusLabel.SetText("价格列表刷新失败：" + modelErr.Error())
			return false
		}
		_ = table.SetCurrentIndex(index)
		updateTotals()
		if showFeedback {
			statusLabel.SetText(fmt.Sprintf("已将 %s 的新单价更新为 %.3f；尚未提交到服务端。", rows[index].Name, value))
		}
		return true
	}
	historyCombo.CurrentIndexChanged().Attach(func() {
		if historyLoading {
			return
		}
		index := historyCombo.CurrentIndex() - 1
		if index >= 0 && index < len(historyPrices) {
			priceEdit.SetText(formatOutboundRevisionPrice(historyPrices[index].Price))
			priceEdit.SetFocus()
		}
	})
	priceEdit.KeyDown().Attach(func(key walk.Key) {
		if key == walk.KeyReturn {
			applySelected(true)
		}
	})

	loadOrder = func() {
		if operationBusy || closed.Load() {
			return
		}
		if outboundRevisionDirty(rows) && walk.MsgBox(
			dlg, "重新读取订单", "重新读取会放弃当前未提交的价格修改，是否继续？", walk.MsgBoxYesNo|walk.MsgBoxIconWarning,
		) != walk.DlgCmdYes {
			return
		}
		loadGeneration++
		generation := loadGeneration
		statusLabel.SetText("正在重新读取线上订单和物料……")
		reloadButton.SetEnabled(false)
		submitButton.SetEnabled(false)
		setEditorEnabled(false)
		go func() {
			order, orderErr := client.FindOutboundByCode(ctx, snapshot.Code)
			var materials []api.OutboundMaterial
			if orderErr == nil {
				materials, orderErr = client.OutboundMaterials(ctx, snapshot.Code)
			}
			if ctx.Err() != nil || closed.Load() {
				return
			}
			dlg.Synchronize(func() {
				if closed.Load() || generation != loadGeneration {
					return
				}
				reloadButton.SetEnabled(true)
				if orderErr != nil {
					statusLabel.SetText("订单读取失败：" + orderErr.Error() + "。请检查网络后重新读取。")
					return
				}
				if strings.TrimSpace(order.CustomerID) == "" {
					statusLabel.SetText("该出库单没有客户，现有核价接口无法提交。")
					return
				}
				baselineOrder = order
				baselineMaterials = append([]api.OutboundMaterial(nil), materials...)
				rows = buildOutboundReviseRows(materials)
				if modelErr := table.SetModel(rows); modelErr != nil {
					statusLabel.SetText("物料列表展示失败：" + modelErr.Error())
					return
				}
				statusLabel.SetText(fmt.Sprintf("已读取 %d 条物料。选择物料编辑价格；所有业务校验仍由服务端执行。", len(rows)))
				updateTotals()
				if len(rows) > 0 {
					_ = table.SetCurrentIndex(0)
					updateSelection()
				}
			})
		}()
	}
	reloadButton.Clicked().Attach(loadOrder)

	submitButton.Clicked().Attach(func() {
		if !applySelected(false) {
			return
		}
		request, summary, requestErr := buildOutboundReviseRequest(baselineOrder, rows)
		if requestErr != nil {
			statusLabel.SetText("提交数据不完整：" + requestErr.Error())
			return
		}
		warning := summary + "\r\n\r\n服务端将重新校验客户、物料和财务状态。"
		if baselineOrder.Status == "已签收" {
			warning += "\r\n该订单已签收，服务端可能生成应收调整流水。"
		}
		warning += "\r\n\r\n是否继续？"
		if walk.MsgBox(dlg, "确认线上核价/调价", warning, walk.MsgBoxYesNo|walk.MsgBoxIconWarning) != walk.DlgCmdYes {
			return
		}
		operationBusy = true
		setEditorEnabled(false)
		submitButton.SetEnabled(false)
		reloadButton.SetEnabled(false)
		cancelButton.SetEnabled(false)
		submitButton.SetText("正在提交，请勿关闭……")
		statusLabel.SetText("正在重新核对服务端订单快照……")
		go func() {
			currentOrder, submitErr := client.FindOutboundByCode(context.Background(), baselineOrder.Code)
			var currentMaterials []api.OutboundMaterial
			if submitErr == nil {
				currentMaterials, submitErr = client.OutboundMaterials(context.Background(), baselineOrder.Code)
			}
			if submitErr == nil {
				submitErr = validateOutboundRevisionSnapshot(baselineOrder, currentOrder, baselineMaterials, currentMaterials)
			}
			if submitErr == nil {
				submitErr = client.ReviseOutbound(context.Background(), request)
			}
			var verifyErr error
			if submitErr == nil {
				latestOrder, queryErr := client.FindOutboundByCode(context.Background(), request.Code)
				var latestMaterials []api.OutboundMaterial
				if queryErr == nil {
					latestMaterials, queryErr = client.OutboundMaterials(context.Background(), request.Code)
				}
				if queryErr != nil {
					verifyErr = fmt.Errorf("核价已返回成功，但重新查询订单结果失败：%w", queryErr)
				} else {
					verifyErr = verifyOutboundRevision(latestOrder, latestMaterials, request)
				}
			}
			dlg.Synchronize(func() {
				operationBusy = false
				submitButton.SetText("核对并提交")
				reloadButton.SetEnabled(true)
				cancelButton.SetEnabled(true)
				setEditorEnabled(true)
				updateTotals()
				if submitErr != nil {
					statusLabel.SetText("提交未完成：" + submitErr.Error() + "。客户端不会自动重试，请重新读取后确认。")
					return
				}
				if verifyErr != nil {
					walk.MsgBox(dlg, "提交成功但复核异常", verifyErr.Error()+"\r\n\r\n请人工复核，禁止直接重复提交。", walk.MsgBoxIconWarning)
				} else {
					walk.MsgBox(dlg, "核价完成", "服务端已确认核价成功，物料单价和订单总额复核一致。", walk.MsgBoxIconInformation)
				}
				success = true
				allowClose = true
				dlg.Accept()
			})
		}()
	})
	loadOrder()
	dlg.Run()
	return success
}

func buildOutboundReviseRows(materials []api.OutboundMaterial) []outboundReviseRow {
	rows := make([]outboundReviseRow, 0, len(materials))
	for _, material := range materials {
		row := outboundReviseRow{
			Index: fmt.Sprint(material.Index), Name: material.Name, Model: material.Model,
			Quantity: fmt.Sprintf("%g %s", material.Quantity, material.Unit),
			Material: material, NewValue: material.Price,
		}
		refreshOutboundRevisionRow(&row)
		rows = append(rows, row)
	}
	return rows
}

func refreshOutboundRevisionRow(row *outboundReviseRow) {
	row.CurrentPrice = formatOutboundRevisionPrice(row.Material.Price)
	row.NewPrice = formatOutboundRevisionPrice(row.NewValue)
	row.Amount = fmt.Sprintf("%.3f", row.Material.Quantity*row.NewValue)
	row.Changed = ""
	if !outboundRevisionPriceEqual(row.Material.Price, row.NewValue) {
		row.Changed = "已修改"
	}
}

func formatOutboundRevisionPrice(value float64) string {
	return strconv.FormatFloat(value, 'f', 3, 64)
}

func parseOutboundRevisionPrice(text string) (float64, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0, fmt.Errorf("新单价不能为空")
	}
	value, err := strconv.ParseFloat(text, 64)
	if err != nil || value < 0 || math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, fmt.Errorf("新单价必须是非负数字")
	}
	return value, nil
}

func outboundRevisionPriceEqual(left, right float64) bool {
	return math.Abs(left-right) < 0.0000001
}

func outboundRevisionDirty(rows []outboundReviseRow) bool {
	for _, row := range rows {
		if !outboundRevisionPriceEqual(row.Material.Price, row.NewValue) {
			return true
		}
	}
	return false
}

func outboundRevisionTotals(rows []outboundReviseRow) (float64, float64) {
	var oldTotal, newTotal float64
	for _, row := range rows {
		oldTotal += row.Material.Quantity * row.Material.Price
		newTotal += row.Material.Quantity * row.NewValue
	}
	return oldTotal, newTotal
}

func buildOutboundReviseRequest(order api.OutboundOrder, rows []outboundReviseRow) (api.OutboundReviseRequest, string, error) {
	if strings.TrimSpace(order.Code) == "" || strings.TrimSpace(order.CustomerID) == "" {
		return api.OutboundReviseRequest{}, "", fmt.Errorf("出库单号和客户不能为空")
	}
	if len(rows) == 0 {
		return api.OutboundReviseRequest{}, "", fmt.Errorf("出库单没有可核价物料")
	}
	request := api.OutboundReviseRequest{
		Code: order.Code, CustomerID: order.CustomerID,
		MaterialsPrice: make([]api.OutboundMaterialPrice, 0, len(rows)),
	}
	changes := make([]string, 0)
	for _, row := range rows {
		if strings.TrimSpace(row.Material.MaterialID) == "" || row.NewValue < 0 || math.IsNaN(row.NewValue) || math.IsInf(row.NewValue, 0) {
			return api.OutboundReviseRequest{}, "", fmt.Errorf("物料“%s”的新单价无效", row.Name)
		}
		request.MaterialsPrice = append(request.MaterialsPrice, api.OutboundMaterialPrice{
			MaterialID: row.Material.MaterialID, Price: row.NewValue,
		})
		if !outboundRevisionPriceEqual(row.Material.Price, row.NewValue) {
			changes = append(changes, fmt.Sprintf("%s：%.3f → %.3f", row.Name, row.Material.Price, row.NewValue))
		}
	}
	if len(changes) == 0 {
		return api.OutboundReviseRequest{}, "", fmt.Errorf("没有价格发生变化")
	}
	oldTotal, newTotal := outboundRevisionTotals(rows)
	visibleChanges := changes
	if len(visibleChanges) > 12 {
		visibleChanges = append(append([]string(nil), visibleChanges[:12]...), fmt.Sprintf("另有 %d 项变更", len(changes)-12))
	}
	summary := fmt.Sprintf("出库单：%s\r\n客户：%s\r\n变更项：%d\r\n%s\r\n\r\n物料金额：%.3f → %.3f（差额 %+.3f）",
		order.Code, displayMaterialValue(order.CustomerName), len(changes), strings.Join(visibleChanges, "\r\n"),
		oldTotal, newTotal, newTotal-oldTotal)
	return request, summary, nil
}

func validateOutboundRevisionSnapshot(
	baselineOrder, currentOrder api.OutboundOrder,
	baselineMaterials, currentMaterials []api.OutboundMaterial,
) error {
	if currentOrder.CustomerID != baselineOrder.CustomerID {
		return fmt.Errorf("订单客户已发生变化，请重新读取后再提交")
	}
	if currentOrder.Status != baselineOrder.Status {
		return fmt.Errorf("订单状态已从“%s”变为“%s”，请重新读取后再提交", baselineOrder.Status, currentOrder.Status)
	}
	if len(currentMaterials) != len(baselineMaterials) {
		return fmt.Errorf("订单物料数量已发生变化，请重新读取后再提交")
	}
	currentByID := make(map[string]api.OutboundMaterial, len(currentMaterials))
	for _, material := range currentMaterials {
		currentByID[material.MaterialID] = material
	}
	for _, baseline := range baselineMaterials {
		current, ok := currentByID[baseline.MaterialID]
		if !ok {
			return fmt.Errorf("物料“%s”已不在订单中，请重新读取", baseline.Name)
		}
		if !outboundRevisionPriceEqual(current.Price, baseline.Price) || math.Abs(current.Quantity-baseline.Quantity) > 0.0000001 {
			return fmt.Errorf("物料“%s”的数量或价格已被其他操作修改，请重新读取", baseline.Name)
		}
	}
	return nil
}

func verifyOutboundRevision(order api.OutboundOrder, materials []api.OutboundMaterial, request api.OutboundReviseRequest) error {
	latest := make(map[string]float64, len(materials))
	materialAmount := 0.0
	for _, material := range materials {
		latest[material.MaterialID] = material.Price
		materialAmount += material.Quantity * material.Price
	}
	for _, expected := range request.MaterialsPrice {
		price, ok := latest[expected.MaterialID]
		if !ok {
			return fmt.Errorf("服务端返回成功，但物料 %s 已不在订单中", expected.MaterialID)
		}
		if !outboundRevisionPriceEqual(price, expected.Price) {
			return fmt.Errorf("服务端返回成功，但物料 %s 的最新价格为 %.3f，预期 %.3f", expected.MaterialID, price, expected.Price)
		}
	}
	expectedTotal := order.CarrierCost + order.OtherCost + materialAmount
	if math.Abs(order.TotalAmount-expectedTotal) > 0.0001 {
		return fmt.Errorf("服务端返回成功，但订单最新总额为 %.3f，按物料、运费和其他费用复算应为 %.3f", order.TotalAmount, expectedTotal)
	}
	return nil
}
