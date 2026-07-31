package ui

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"

	"zhengshi-wms-windowsapp/internal/api"
)

type outboundAllocationData struct {
	Material    api.OutboundMaterial
	Inventories []api.Inventory
}

func (ui *mainUI) confirmSelectedOutbound() {
	order, ok := ui.selectedOutbound()
	if !ok {
		return
	}
	materials := append([]api.OutboundMaterial(nil), ui.outbound.selectedMaterials...)
	ui.setOutboundOperationBusy(true, "正在重新查询订单物料及可用库存……")
	go func() {
		var err error
		if len(materials) == 0 {
			materials, err = ui.session.Client.OutboundMaterials(context.Background(), order.Code)
		}
		var allocations []outboundAllocationData
		if err == nil {
			allocations, err = loadOutboundAllocationData(context.Background(), ui.session.Client, materials)
		}
		ui.window.Synchronize(func() {
			ui.setOutboundOperationBusy(false, "")
			if err != nil {
				walk.MsgBox(ui.window, "库存加载失败", err.Error(), walk.MsgBoxIconError)
				ui.loadOutbound()
				return
			}
			ShowOutboundConfirm(ui.window, ui.session.Client, order, allocations)
			ui.loadOutbound()
		})
	}()
}

func loadOutboundAllocationData(ctx context.Context, client *api.Client, materials []api.OutboundMaterial) ([]outboundAllocationData, error) {
	result := make([]outboundAllocationData, 0, len(materials))
	for _, material := range materials {
		inventories, err := client.InventoryByMaterial(ctx, material.MaterialID)
		if err != nil {
			return nil, fmt.Errorf("查询物料“%s”库存失败：%w", material.Name, err)
		}
		result = append(result, outboundAllocationData{Material: material, Inventories: inventories})
	}
	return result, nil
}

func ShowOutboundConfirm(owner walk.Form, client *api.Client, order api.OutboundOrder, allocations []outboundAllocationData) bool {
	var dlg *walk.Dialog
	var timeEdit *walk.LineEdit
	var submit *walk.PushButton
	editRows := make([][]*walk.LineEdit, len(allocations))
	content := make([]Widget, 0, len(allocations))
	for materialIndex, allocation := range allocations {
		gridChildren := make([]Widget, 0, len(allocation.Inventories)*4)
		editRows[materialIndex] = make([]*walk.LineEdit, len(allocation.Inventories))
		for inventoryIndex, inventory := range allocation.Inventories {
			location := inventoryLocation(inventory)
			batch := inventory.ReceiveCode
			if batch == "" {
				batch = inventory.ReceiptCode
			}
			gridChildren = append(gridChildren,
				Label{Text: location, ToolTipText: location},
				Label{Text: batch},
				Label{Text: fmt.Sprintf("可用 %g %s", inventory.AvailableQuantity, inventory.Unit)},
				LineEdit{
					AssignTo: &editRows[materialIndex][inventoryIndex], Text: "0",
					CueBanner: "分配数量", MinSize: Size{Width: 100, Height: 28},
				},
			)
		}
		if len(gridChildren) == 0 {
			gridChildren = append(gridChildren, Label{Text: "当前接口未返回可用库存批次。", ColumnSpan: 4, TextColor: walk.RGB(150, 55, 45)})
		}
		content = append(content, GroupBox{
			Title: fmt.Sprintf("%d. %s（订单数量 %g %s）",
				allocation.Material.Index, allocation.Material.Name,
				allocation.Material.Quantity, allocation.Material.Unit),
			Layout:   Grid{Columns: 4, Margins: Margins{Left: 12, Top: 10, Right: 12, Bottom: 12}, Spacing: 8},
			Children: gridChildren,
		})
	}
	var success bool
	err := Dialog{
		AssignTo: &dlg,
		Title:    "确认并分配库存 - " + order.Code,
		MinSize:  Size{Width: 860, Height: 620},
		Size:     Size{Width: 980, Height: 760},
		Layout:   VBox{Margins: Margins{Left: 16, Top: 14, Right: 16, Bottom: 14}, Spacing: 10},
		Children: []Widget{
			Label{Text: "线上写操作", Font: Font{Bold: true}, TextColor: walk.RGB(190, 45, 35)},
			Label{Text: "只填写本次需要分配的库存批次；未分配的物料由现有后端规则处理。"},
			Composite{Layout: Grid{Columns: 2, Spacing: 8}, Children: []Widget{
				Label{Text: "确认时间"},
				LineEdit{
					AssignTo: &timeEdit, Text: time.Now().Format("2006-01-02 15:04:05"),
					CueBanner: "YYYY-MM-DD HH:mm:ss",
				},
			}},
			ScrollView{
				HorizontalFixed: true,
				Layout:          VBox{Margins: Margins{Left: 2, Top: 2, Right: 8, Bottom: 2}, Spacing: 8},
				Children:        content,
				StretchFactor:   1,
			},
			Composite{Layout: HBox{Spacing: 8}, Children: []Widget{
				HSpacer{},
				PushButton{Text: "取消", OnClicked: func() { dlg.Cancel() }},
				PushButton{AssignTo: &submit, Text: "核对并提交"},
			}},
		},
	}.Create(owner)
	if err != nil {
		walk.MsgBox(owner, "确认出库窗口错误", err.Error(), walk.MsgBoxIconError)
		return false
	}
	submit.Clicked().Attach(func() {
		operationTime, parseErr := parseOperationTime(timeEdit.Text())
		if parseErr != nil {
			walk.MsgBox(dlg, "确认时间格式错误", parseErr.Error(), walk.MsgBoxIconWarning)
			timeEdit.SetFocus()
			return
		}
		quantities := make([][]string, len(editRows))
		for materialIndex := range editRows {
			quantities[materialIndex] = make([]string, len(editRows[materialIndex]))
			for inventoryIndex, edit := range editRows[materialIndex] {
				quantities[materialIndex][inventoryIndex] = edit.Text()
			}
		}
		request, summary, buildErr := buildOutboundConfirmRequest(order.Code, operationTime, allocations, quantities)
		if buildErr != nil {
			walk.MsgBox(dlg, "分配数据不完整", buildErr.Error(), walk.MsgBoxIconWarning)
			return
		}
		if walk.MsgBox(
			dlg, "确认线上库存分配",
			summary+"\r\n\r\n提交后可能按现有服务端规则拆分剩余预发货订单，是否继续？",
			walk.MsgBoxYesNo|walk.MsgBoxIconWarning,
		) != walk.DlgCmdYes {
			return
		}
		submit.SetEnabled(false)
		submit.SetText("正在提交，请勿关闭……")
		go func() {
			current, preflightErr := client.FindOutboundByCode(context.Background(), order.Code)
			if preflightErr == nil && !canConfirmOutbound(current) {
				preflightErr = fmt.Errorf("服务端最新状态为“%s”，已停止提交", current.Status)
			}
			if preflightErr != nil {
				dlg.Synchronize(func() {
					submit.SetEnabled(true)
					submit.SetText("核对并提交")
					walk.MsgBox(dlg, "提交前刷新失败", preflightErr.Error(), walk.MsgBoxIconWarning)
				})
				return
			}
			if preflightErr = validateCurrentAllocationAvailability(client, request); preflightErr != nil {
				dlg.Synchronize(func() {
					submit.SetEnabled(true)
					submit.SetText("核对并提交")
					walk.MsgBox(dlg, "库存已变化", preflightErr.Error(), walk.MsgBoxIconWarning)
				})
				return
			}
			requestErr := client.ConfirmOutbound(context.Background(), request)
			var verifyErr error
			if requestErr == nil {
				verifyErr = verifyOutboundStatus(client, order.Code, "待拣货")
			}
			dlg.Synchronize(func() {
				if requestErr != nil {
					walk.MsgBox(dlg, "操作结果未知", unknownWriteResultMessage(requestErr), walk.MsgBoxIconError)
					dlg.Cancel()
					return
				}
				success = true
				if verifyErr != nil {
					walk.MsgBox(dlg, "服务端已受理但回读异常", verifyErr.Error(), walk.MsgBoxIconWarning)
				} else {
					walk.MsgBox(dlg, "确认成功", "订单已进入待拣货状态；列表将重新从服务端加载。", walk.MsgBoxIconInformation)
				}
				dlg.Accept()
			})
		}()
	})
	dlg.Run()
	return success
}

func buildOutboundConfirmRequest(
	code string,
	operationTime int64,
	allocations []outboundAllocationData,
	quantities [][]string,
) (api.OutboundConfirmRequest, string, error) {
	request := api.OutboundConfirmRequest{Code: strings.TrimSpace(code), ConfirmTime: operationTime}
	lines := []string{"出库单：" + strings.TrimSpace(code)}
	if len(allocations) != len(quantities) {
		return request, "", fmt.Errorf("库存分配行与物料不匹配")
	}
	for materialIndex, allocation := range allocations {
		if len(allocation.Inventories) != len(quantities[materialIndex]) {
			return request, "", fmt.Errorf("物料“%s”的库存分配行不匹配", allocation.Material.Name)
		}
		item := api.OutboundConfirmMaterial{
			MaterialID: allocation.Material.MaterialID,
			Index:      allocation.Material.Index,
		}
		total := 0.0
		for inventoryIndex, inventory := range allocation.Inventories {
			quantity, err := parseNonNegativeNumber(quantities[materialIndex][inventoryIndex], "分配数量")
			if err != nil {
				return request, "", fmt.Errorf("物料“%s”：%w", allocation.Material.Name, err)
			}
			if quantity <= 0 {
				continue
			}
			if quantity > inventory.AvailableQuantity+0.0000001 {
				return request, "", fmt.Errorf(
					"物料“%s”在批次“%s”的分配数量 %g 超过接口返回的可用数量 %g",
					allocation.Material.Name, inventory.ReceiveCode, quantity, inventory.AvailableQuantity,
				)
			}
			total += quantity
			item.Inventorys = append(item.Inventorys, api.OutboundConfirmInventory{
				InventoryID: inventory.ID, ShipmentQuantity: quantity,
			})
		}
		if total <= 0 {
			continue
		}
		if total > allocation.Material.Quantity+0.0000001 {
			return request, "", fmt.Errorf(
				"物料“%s”的分配合计 %g 超过订单数量 %g",
				allocation.Material.Name, total, allocation.Material.Quantity,
			)
		}
		request.Materials = append(request.Materials, item)
		lines = append(lines, fmt.Sprintf("%s：%g %s", allocation.Material.Name, total, allocation.Material.Unit))
	}
	if len(request.Materials) == 0 {
		return request, "", fmt.Errorf("请至少为一项物料填写大于 0 的分配数量")
	}
	return request, strings.Join(lines, "\r\n"), nil
}

func validateCurrentAllocationAvailability(client *api.Client, request api.OutboundConfirmRequest) error {
	for _, material := range request.Materials {
		inventories, err := client.InventoryByMaterial(context.Background(), material.MaterialID)
		if err != nil {
			return fmt.Errorf("重新查询物料库存失败：%w", err)
		}
		available := make(map[string]float64, len(inventories))
		for _, inventory := range inventories {
			available[inventory.ID] = inventory.AvailableQuantity
		}
		for _, allocation := range material.Inventorys {
			current, ok := available[allocation.InventoryID]
			if !ok {
				return fmt.Errorf("库存批次 %s 已不可用，请关闭窗口后刷新", allocation.InventoryID)
			}
			if allocation.ShipmentQuantity > current+0.0000001 {
				return fmt.Errorf(
					"库存批次 %s 的最新可用数量为 %g，小于本次分配数量 %g；请关闭窗口后刷新",
					allocation.InventoryID, current, allocation.ShipmentQuantity,
				)
			}
		}
	}
	return nil
}

func (ui *mainUI) pickSelectedOutbound() {
	order, ok := ui.selectedOutbound()
	if !ok {
		return
	}
	showOutboundTimeAction(
		ui.window, ui.session.Client, order, "确认拣货", "拣货时间", "待拣货", "已拣货",
		canPickOutbound,
		func(ctx context.Context, operationTime int64) error {
			return ui.session.Client.PickOutbound(ctx, api.OutboundPickRequest{Code: order.Code, PickingTime: operationTime})
		},
	)
	ui.loadOutbound()
}

func (ui *mainUI) packSelectedOutbound() {
	order, ok := ui.selectedOutbound()
	if !ok {
		return
	}
	showOutboundTimeAction(
		ui.window, ui.session.Client, order, "确认打包", "打包时间", order.Status, "已打包",
		canPackOutbound,
		func(ctx context.Context, operationTime int64) error {
			return ui.session.Client.PackOutbound(ctx, api.OutboundPackRequest{Code: order.Code, PackingTime: operationTime})
		},
	)
	ui.loadOutbound()
}

func showOutboundTimeAction(
	owner walk.Form,
	client *api.Client,
	order api.OutboundOrder,
	title, timeLabel, expectedDescription, targetStatus string,
	valid func(api.OutboundOrder) bool,
	execute func(context.Context, int64) error,
) bool {
	var dlg *walk.Dialog
	var timeEdit *walk.LineEdit
	var submit *walk.PushButton
	var success bool
	err := Dialog{
		AssignTo: &dlg,
		Title:    title + " - " + order.Code,
		MinSize:  Size{Width: 520, Height: 250},
		Size:     Size{Width: 560, Height: 280},
		Layout:   VBox{Margins: Margins{Left: 18, Top: 16, Right: 18, Bottom: 16}, Spacing: 10},
		Children: []Widget{
			Label{Text: "线上写操作", Font: Font{Bold: true}, TextColor: walk.RGB(190, 45, 35)},
			Label{Text: fmt.Sprintf("出库单：%s    当前状态：%s", order.Code, order.Status)},
			Composite{Layout: Grid{Columns: 2, Spacing: 8}, Children: []Widget{
				Label{Text: timeLabel},
				LineEdit{
					AssignTo: &timeEdit, Text: time.Now().Format("2006-01-02 15:04:05"),
					CueBanner: "YYYY-MM-DD HH:mm:ss",
				},
			}},
			Label{Text: "提交前会重新查询服务端状态；提交后不会自动重试。", TextColor: walk.RGB(85, 85, 85)},
			Composite{Layout: HBox{Spacing: 8}, Children: []Widget{
				HSpacer{},
				PushButton{Text: "取消", OnClicked: func() { dlg.Cancel() }},
				PushButton{AssignTo: &submit, Text: title},
			}},
		},
	}.Create(owner)
	if err != nil {
		walk.MsgBox(owner, title+"窗口错误", err.Error(), walk.MsgBoxIconError)
		return false
	}
	submit.Clicked().Attach(func() {
		operationTime, parseErr := parseOperationTime(timeEdit.Text())
		if parseErr != nil {
			walk.MsgBox(dlg, timeLabel+"格式错误", parseErr.Error(), walk.MsgBoxIconWarning)
			timeEdit.SetFocus()
			return
		}
		if walk.MsgBox(
			dlg, title,
			fmt.Sprintf("出库单：%s\r\n当前预期状态：%s\r\n\r\n是否提交？", order.Code, expectedDescription),
			walk.MsgBoxYesNo|walk.MsgBoxIconWarning,
		) != walk.DlgCmdYes {
			return
		}
		submit.SetEnabled(false)
		submit.SetText("正在提交，请勿关闭……")
		go func() {
			current, preflightErr := client.FindOutboundByCode(context.Background(), order.Code)
			if preflightErr == nil && !valid(current) {
				preflightErr = fmt.Errorf("服务端最新状态为“%s”，已停止提交", current.Status)
			}
			if preflightErr != nil {
				dlg.Synchronize(func() {
					submit.SetEnabled(true)
					submit.SetText(title)
					walk.MsgBox(dlg, "提交前刷新失败", preflightErr.Error(), walk.MsgBoxIconWarning)
				})
				return
			}
			requestErr := execute(context.Background(), operationTime)
			var verifyErr error
			if requestErr == nil {
				verifyErr = verifyOutboundStatus(client, order.Code, targetStatus)
			}
			dlg.Synchronize(func() {
				if requestErr != nil {
					walk.MsgBox(dlg, "操作结果未知", unknownWriteResultMessage(requestErr), walk.MsgBoxIconError)
					dlg.Cancel()
					return
				}
				success = true
				if verifyErr != nil {
					walk.MsgBox(dlg, "服务端已受理但回读异常", verifyErr.Error(), walk.MsgBoxIconWarning)
				} else {
					walk.MsgBox(dlg, title+"成功", "列表将重新从服务端加载。", walk.MsgBoxIconInformation)
				}
				dlg.Accept()
			})
		}()
	})
	dlg.Run()
	return success
}

func (ui *mainUI) weighSelectedOutbound() {
	order, ok := ui.selectedOutbound()
	if !ok {
		return
	}
	materials := append([]api.OutboundMaterial(nil), ui.outbound.selectedMaterials...)
	if len(materials) == 0 {
		walk.MsgBox(ui.window, "物料尚未加载", "请等待订单物料加载完成后再称重。", walk.MsgBoxIconInformation)
		return
	}
	ShowOutboundWeigh(ui.window, ui.session.Client, order, materials)
	ui.loadOutbound()
}

func ShowOutboundWeigh(owner walk.Form, client *api.Client, order api.OutboundOrder, materials []api.OutboundMaterial) bool {
	var dlg *walk.Dialog
	var timeEdit *walk.LineEdit
	var submit *walk.PushButton
	weightEdits := make([]*walk.LineEdit, len(materials))
	grid := make([]Widget, 0, len(materials)*2)
	for index, material := range materials {
		grid = append(grid,
			Label{Text: fmt.Sprintf("%d. %s（%g %s）", material.Index, material.Name, material.Quantity, material.Unit)},
			LineEdit{
				AssignTo: &weightEdits[index], Text: fmt.Sprintf("%g", material.Weight),
				CueBanner: "重量", MinSize: Size{Width: 130, Height: 28},
			},
		)
	}
	var success bool
	err := Dialog{
		AssignTo: &dlg,
		Title:    "确认称重 - " + order.Code,
		MinSize:  Size{Width: 620, Height: 420},
		Size:     Size{Width: 700, Height: 560},
		Layout:   VBox{Margins: Margins{Left: 16, Top: 14, Right: 16, Bottom: 14}, Spacing: 10},
		Children: []Widget{
			Label{Text: "线上写操作", Font: Font{Bold: true}, TextColor: walk.RGB(190, 45, 35)},
			Composite{Layout: Grid{Columns: 2, Spacing: 8}, Children: []Widget{
				Label{Text: "称重时间"},
				LineEdit{
					AssignTo: &timeEdit, Text: time.Now().Format("2006-01-02 15:04:05"),
					CueBanner: "YYYY-MM-DD HH:mm:ss",
				},
			}},
			ScrollView{
				HorizontalFixed: true, StretchFactor: 1,
				Layout:   Grid{Columns: 2, Margins: Margins{Left: 4, Top: 4, Right: 8, Bottom: 4}, Spacing: 8},
				Children: grid,
			},
			Composite{Layout: HBox{Spacing: 8}, Children: []Widget{
				HSpacer{},
				PushButton{Text: "取消", OnClicked: func() { dlg.Cancel() }},
				PushButton{AssignTo: &submit, Text: "核对并提交"},
			}},
		},
	}.Create(owner)
	if err != nil {
		walk.MsgBox(owner, "称重窗口错误", err.Error(), walk.MsgBoxIconError)
		return false
	}
	submit.Clicked().Attach(func() {
		operationTime, parseErr := parseOperationTime(timeEdit.Text())
		if parseErr != nil {
			walk.MsgBox(dlg, "称重时间格式错误", parseErr.Error(), walk.MsgBoxIconWarning)
			return
		}
		request := api.OutboundWeighRequest{Code: order.Code, WeighingTime: operationTime}
		for index, material := range materials {
			weight, numberErr := parseNonNegativeNumber(weightEdits[index].Text(), "重量")
			if numberErr != nil {
				walk.MsgBox(dlg, "重量格式错误", fmt.Sprintf("物料“%s”：%s", material.Name, numberErr), walk.MsgBoxIconWarning)
				weightEdits[index].SetFocus()
				return
			}
			request.Materials = append(request.Materials, api.OutboundWeighMaterial{
				MaterialID: material.MaterialID, Weight: weight,
			})
		}
		if walk.MsgBox(dlg, "确认线上称重", fmt.Sprintf("出库单：%s\r\n物料：%d 项\r\n\r\n是否提交？", order.Code, len(materials)), walk.MsgBoxYesNo|walk.MsgBoxIconWarning) != walk.DlgCmdYes {
			return
		}
		submit.SetEnabled(false)
		submit.SetText("正在提交，请勿关闭……")
		go func() {
			current, preflightErr := client.FindOutboundByCode(context.Background(), order.Code)
			if preflightErr == nil && !canWeighOutbound(current) {
				preflightErr = fmt.Errorf("服务端最新状态为“%s”，已停止提交", current.Status)
			}
			if preflightErr != nil {
				dlg.Synchronize(func() {
					submit.SetEnabled(true)
					submit.SetText("核对并提交")
					walk.MsgBox(dlg, "提交前刷新失败", preflightErr.Error(), walk.MsgBoxIconWarning)
				})
				return
			}
			requestErr := client.WeighOutbound(context.Background(), request)
			var verifyErr error
			if requestErr == nil {
				verifyErr = verifyOutboundStatus(client, order.Code, "已称重")
			}
			dlg.Synchronize(func() {
				if requestErr != nil {
					walk.MsgBox(dlg, "操作结果未知", unknownWriteResultMessage(requestErr), walk.MsgBoxIconError)
					dlg.Cancel()
					return
				}
				success = true
				if verifyErr != nil {
					walk.MsgBox(dlg, "服务端已受理但回读异常", verifyErr.Error(), walk.MsgBoxIconWarning)
				}
				dlg.Accept()
			})
		}()
	})
	dlg.Run()
	return success
}

func (ui *mainUI) departSelectedOutbound() {
	order, ok := ui.selectedOutbound()
	if !ok {
		return
	}
	ui.setOutboundOperationBusy(true, "正在加载承运商选项……")
	go func() {
		carriers, err := ui.session.Client.Carriers(context.Background())
		ui.window.Synchronize(func() {
			ui.setOutboundOperationBusy(false, "")
			if err != nil {
				walk.MsgBox(ui.window, "承运商加载失败", "仍可不选择承运商继续出库：\r\n"+err.Error(), walk.MsgBoxIconWarning)
				carriers = nil
			}
			ShowOutboundDeparture(ui.window, ui.session.Client, order, carriers)
			ui.loadOutbound()
		})
	}()
}

func ShowOutboundDeparture(owner walk.Form, client *api.Client, order api.OutboundOrder, carriers []api.Carrier) bool {
	var dlg *walk.Dialog
	var timeEdit, carrierCostEdit, otherCostEdit *walk.LineEdit
	var carrierCombo *walk.ComboBox
	var submit *walk.PushButton
	labels := make([]string, 1, len(carriers)+1)
	labels[0] = "不选择承运商"
	for _, carrier := range carriers {
		labels = append(labels, businessOptionLabel(carrier.Name, carrier.Code))
	}
	var success bool
	err := Dialog{
		AssignTo: &dlg,
		Title:    "确认出库 - " + order.Code,
		MinSize:  Size{Width: 580, Height: 360},
		Size:     Size{Width: 640, Height: 410},
		Layout:   VBox{Margins: Margins{Left: 18, Top: 16, Right: 18, Bottom: 16}, Spacing: 10},
		Children: []Widget{
			Label{Text: "线上写操作", Font: Font{Bold: true}, TextColor: walk.RGB(190, 45, 35)},
			Composite{Layout: Grid{Columns: 2, Spacing: 8}, Children: []Widget{
				Label{Text: "出库时间"},
				LineEdit{AssignTo: &timeEdit, Text: time.Now().Format("2006-01-02 15:04:05")},
				Label{Text: "承运商"},
				ComboBox{AssignTo: &carrierCombo, Model: labels, CurrentIndex: 0},
				Label{Text: "运费"},
				LineEdit{AssignTo: &carrierCostEdit, Text: "0", CueBanner: "非负数字"},
				Label{Text: "其他费用"},
				LineEdit{AssignTo: &otherCostEdit, Text: "0", CueBanner: "非负数字"},
			}},
			Label{Text: "提交后不会自动重试；若网络中断，请返回队列刷新状态。", TextColor: walk.RGB(85, 85, 85)},
			Composite{Layout: HBox{Spacing: 8}, Children: []Widget{
				HSpacer{},
				PushButton{Text: "取消", OnClicked: func() { dlg.Cancel() }},
				PushButton{AssignTo: &submit, Text: "核对并提交"},
			}},
		},
	}.Create(owner)
	if err != nil {
		walk.MsgBox(owner, "出库窗口错误", err.Error(), walk.MsgBoxIconError)
		return false
	}
	submit.Clicked().Attach(func() {
		operationTime, parseErr := parseOperationTime(timeEdit.Text())
		if parseErr != nil {
			walk.MsgBox(dlg, "出库时间格式错误", parseErr.Error(), walk.MsgBoxIconWarning)
			return
		}
		carrierCost, numberErr := parseNonNegativeNumber(carrierCostEdit.Text(), "运费")
		if numberErr != nil {
			walk.MsgBox(dlg, "运费格式错误", numberErr.Error(), walk.MsgBoxIconWarning)
			return
		}
		otherCost, numberErr := parseNonNegativeNumber(otherCostEdit.Text(), "其他费用")
		if numberErr != nil {
			walk.MsgBox(dlg, "其他费用格式错误", numberErr.Error(), walk.MsgBoxIconWarning)
			return
		}
		carrierID := ""
		if index := carrierCombo.CurrentIndex() - 1; index >= 0 && index < len(carriers) {
			carrierID = carriers[index].ID
		}
		request := api.OutboundDepartureRequest{
			Code: order.Code, DepartureTime: operationTime, CarrierID: carrierID,
			CarrierCost: carrierCost, OtherCost: otherCost,
		}
		if walk.MsgBox(dlg, "确认线上出库", fmt.Sprintf(
			"出库单：%s\r\n运费：%.2f\r\n其他费用：%.2f\r\n\r\n是否提交？",
			order.Code, carrierCost, otherCost,
		), walk.MsgBoxYesNo|walk.MsgBoxIconWarning) != walk.DlgCmdYes {
			return
		}
		submit.SetEnabled(false)
		submit.SetText("正在提交，请勿关闭……")
		go func() {
			current, preflightErr := client.FindOutboundByCode(context.Background(), order.Code)
			if preflightErr == nil && !canDepartOutbound(current) {
				preflightErr = fmt.Errorf("服务端最新状态为“%s”，已停止提交", current.Status)
			}
			if preflightErr != nil {
				dlg.Synchronize(func() {
					submit.SetEnabled(true)
					submit.SetText("核对并提交")
					walk.MsgBox(dlg, "提交前刷新失败", preflightErr.Error(), walk.MsgBoxIconWarning)
				})
				return
			}
			requestErr := client.DepartOutbound(context.Background(), request)
			var verifyErr error
			if requestErr == nil {
				verifyErr = verifyOutboundStatus(client, order.Code, "已出库")
			}
			dlg.Synchronize(func() {
				if requestErr != nil {
					walk.MsgBox(dlg, "操作结果未知", unknownWriteResultMessage(requestErr), walk.MsgBoxIconError)
					dlg.Cancel()
					return
				}
				success = true
				if verifyErr != nil {
					walk.MsgBox(dlg, "服务端已受理但回读异常", verifyErr.Error(), walk.MsgBoxIconWarning)
				}
				dlg.Accept()
			})
		}()
	})
	dlg.Run()
	return success
}

func (ui *mainUI) receiptSelectedOutbound() {
	order, ok := ui.selectedOutbound()
	if !ok {
		return
	}
	ShowOutboundReceipt(ui.window, ui.session.Client, order)
	ui.loadOutbound()
}

func ShowOutboundReceipt(owner walk.Form, client *api.Client, order api.OutboundOrder) bool {
	var dlg *walk.Dialog
	var timeEdit, fileEdit *walk.LineEdit
	var submit *walk.PushButton
	var filePath string
	var success bool
	err := Dialog{
		AssignTo: &dlg,
		Title:    "确认签收 - " + order.Code,
		MinSize:  Size{Width: 620, Height: 310},
		Size:     Size{Width: 700, Height: 350},
		Layout:   VBox{Margins: Margins{Left: 18, Top: 16, Right: 18, Bottom: 16}, Spacing: 10},
		Children: []Widget{
			Label{Text: "线上写操作", Font: Font{Bold: true}, TextColor: walk.RGB(190, 45, 35)},
			Composite{Layout: Grid{Columns: 3, Spacing: 8}, Children: []Widget{
				Label{Text: "签收时间"},
				LineEdit{
					AssignTo: &timeEdit, Text: time.Now().Format("2006-01-02 15:04:05"),
					ColumnSpan: 2,
				},
				Label{Text: "签收附件"},
				LineEdit{AssignTo: &fileEdit, ReadOnly: true, CueBanner: "可选，支持常见图片格式"},
				PushButton{Text: "选择图片", OnClicked: func() {
					fileDialog := new(walk.FileDialog)
					fileDialog.Title = "选择签收附件"
					fileDialog.Filter = "图片文件 (*.png;*.jpg;*.jpeg;*.bmp;*.gif)|*.png;*.jpg;*.jpeg;*.bmp;*.gif"
					ok, openErr := fileDialog.ShowOpen(dlg)
					if openErr != nil {
						walk.MsgBox(dlg, "无法选择文件", openErr.Error(), walk.MsgBoxIconError)
						return
					}
					if !ok {
						return
					}
					filePath = fileDialog.FilePath
					fileEdit.SetText(filepath.Base(filePath))
				}},
			}},
			Label{Text: "若选择附件，将先调用现有图片上传接口，再提交签收。写请求不会自动重试。", TextColor: walk.RGB(85, 85, 85)},
			Composite{Layout: HBox{Spacing: 8}, Children: []Widget{
				HSpacer{},
				PushButton{Text: "取消", OnClicked: func() { dlg.Cancel() }},
				PushButton{AssignTo: &submit, Text: "核对并提交"},
			}},
		},
	}.Create(owner)
	if err != nil {
		walk.MsgBox(owner, "签收窗口错误", err.Error(), walk.MsgBoxIconError)
		return false
	}
	submit.Clicked().Attach(func() {
		operationTime, parseErr := parseOperationTime(timeEdit.Text())
		if parseErr != nil {
			walk.MsgBox(dlg, "签收时间格式错误", parseErr.Error(), walk.MsgBoxIconWarning)
			return
		}
		attachment := "无"
		if filePath != "" {
			attachment = filepath.Base(filePath)
		}
		if walk.MsgBox(dlg, "确认线上签收", fmt.Sprintf(
			"出库单：%s\r\n签收附件：%s\r\n\r\n是否提交？", order.Code, attachment,
		), walk.MsgBoxYesNo|walk.MsgBoxIconWarning) != walk.DlgCmdYes {
			return
		}
		submit.SetEnabled(false)
		submit.SetText("正在提交，请勿关闭……")
		go func() {
			current, preflightErr := client.FindOutboundByCode(context.Background(), order.Code)
			if preflightErr == nil && !canReceiptOutbound(current) {
				preflightErr = fmt.Errorf("服务端最新状态为“%s”，已停止提交", current.Status)
			}
			if preflightErr != nil {
				dlg.Synchronize(func() {
					submit.SetEnabled(true)
					submit.SetText("核对并提交")
					walk.MsgBox(dlg, "提交前刷新失败", preflightErr.Error(), walk.MsgBoxIconWarning)
				})
				return
			}
			annex := make([]string, 0, 1)
			var requestErr error
			if filePath != "" {
				var imageURL string
				imageURL, requestErr = client.UploadImage(context.Background(), filePath)
				if requestErr == nil {
					annex = append(annex, imageURL)
				}
			}
			if requestErr == nil {
				requestErr = client.ReceiptOutbound(context.Background(), api.OutboundReceiptRequest{
					Code: order.Code, ReceiptTime: operationTime, Annex: annex,
				})
			}
			var verifyErr error
			if requestErr == nil {
				verifyErr = verifyOutboundStatus(client, order.Code, "已签收")
			}
			dlg.Synchronize(func() {
				if requestErr != nil {
					walk.MsgBox(dlg, "操作结果未知", unknownWriteResultMessage(requestErr), walk.MsgBoxIconError)
					dlg.Cancel()
					return
				}
				success = true
				if verifyErr != nil {
					walk.MsgBox(dlg, "服务端已受理但回读异常", verifyErr.Error(), walk.MsgBoxIconWarning)
				}
				dlg.Accept()
			})
		}()
	})
	dlg.Run()
	return success
}

func parseOperationTime(text string) (int64, error) {
	value, err := time.ParseInLocation("2006-01-02 15:04:05", strings.TrimSpace(text), time.Local)
	if err != nil {
		return 0, fmt.Errorf("请使用 YYYY-MM-DD HH:mm:ss")
	}
	return value.Unix(), nil
}

func verifyOutboundStatus(client *api.Client, code, expected string) error {
	order, err := client.FindOutboundByCode(context.Background(), code)
	if err != nil {
		return fmt.Errorf("写入已返回成功，但重新查询订单失败：%w", err)
	}
	if order.Status != expected {
		return fmt.Errorf("写入已返回成功，但服务端最新状态为“%s”，预期为“%s”；请人工复核", order.Status, expected)
	}
	return nil
}

func unknownWriteResultMessage(err error) string {
	return "服务端未确认操作完成：\r\n" + err.Error() +
		"\r\n\r\n客户端不会自动重试。窗口关闭后请刷新队列，确认服务端状态后再决定下一步。"
}

func inventoryLocation(inventory api.Inventory) string {
	parts := []string{
		strings.TrimSpace(inventory.WarehouseName),
		strings.TrimSpace(inventory.WarehouseZoneName),
		strings.TrimSpace(inventory.WarehouseRackName),
		strings.TrimSpace(inventory.WarehouseBinName),
	}
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			filtered = append(filtered, part)
		}
	}
	return strings.Join(filtered, " / ")
}
