package ui

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"

	"zhengshi-wms-windowsapp/internal/api"
)

type positionOption struct {
	Label string
	IDs   []string
}

type inboundMaterialDetailRow struct {
	Index     string
	Name      string
	Model     string
	Planned   string
	Received  string
	Remaining string
	Status    string
}

type inboundRecordRow struct {
	BatchCode string
	Time      string
	Material  string
	Quantity  string
	Location  string
	Operator  string
	Remark    string
}

func ShowInboundDetail(owner walk.Form, client *api.Client, receipt api.InboundReceipt) {
	var dlg *walk.Dialog
	var recordsTable *walk.TableView
	materialRows := make([]inboundMaterialDetailRow, 0, len(receipt.Materials))
	for _, material := range receipt.Materials {
		remaining := material.EstimatedQuantity - material.ActualQuantity
		if remaining < 0 {
			remaining = 0
		}
		materialRows = append(materialRows, inboundMaterialDetailRow{
			Index: fmt.Sprint(material.Index + 1), Name: material.Name, Model: material.Model,
			Planned:  fmt.Sprintf("%g %s", material.EstimatedQuantity, material.Unit),
			Received: fmt.Sprintf("%g", material.ActualQuantity), Remaining: fmt.Sprintf("%g", remaining),
			Status: material.Status,
		})
	}
	business := receipt.SupplierName
	if business == "" {
		business = receipt.CustomerName
	}
	header := fmt.Sprintf(
		"单号：%s    类型：%s    状态：%s\r\n供应商/客户：%s\r\n备注：%s",
		receipt.Code, receipt.Type, receipt.Status, business, receipt.Remark,
	)
	err := Dialog{
		AssignTo: &dlg,
		Title:    "入库详情 - " + receipt.Code,
		MinSize:  Size{Width: 760, Height: 580},
		Size:     Size{Width: 860, Height: 680},
		Layout:   VBox{Margins: Margins{Left: 14, Top: 14, Right: 14, Bottom: 14}},
		Children: []Widget{
			TextEdit{Text: header, ReadOnly: true, MinSize: Size{Height: 70}, MaxSize: Size{Height: 90}},
			Label{Text: "物料进度", Font: Font{Bold: true}},
			TableView{
				Model:            materialRows,
				AlternatingRowBG: true,
				MinSize:          Size{Height: 210},
				Columns: []TableViewColumn{
					{Title: "序号", DataMember: "Index", Width: 55},
					{Title: "物料", DataMember: "Name", Width: 185},
					{Title: "型号", DataMember: "Model", Width: 110},
					{Title: "计划", DataMember: "Planned", Width: 100},
					{Title: "已收", DataMember: "Received", Width: 80},
					{Title: "剩余", DataMember: "Remaining", Width: 80},
					{Title: "状态", DataMember: "Status", Width: 95},
				},
			},
			Label{Text: "批次收货记录", Font: Font{Bold: true}},
			TableView{
				AssignTo:         &recordsTable,
				Model:            []inboundRecordRow{},
				AlternatingRowBG: true,
				MinSize:          Size{Height: 220},
				Columns: []TableViewColumn{
					{Title: "批次编号", DataMember: "BatchCode", Width: 130},
					{Title: "时间", DataMember: "Time", Width: 125},
					{Title: "物料", DataMember: "Material", Width: 135},
					{Title: "数量", DataMember: "Quantity", Width: 75},
					{Title: "仓储位置", DataMember: "Location", Width: 180},
					{Title: "操作人", DataMember: "Operator", Width: 80},
					{Title: "备注", DataMember: "Remark", Width: 140},
				},
			},
			Composite{Layout: HBox{}, Children: []Widget{HSpacer{}, PushButton{Text: "关闭", OnClicked: func() { dlg.Accept() }}}},
		},
	}.Create(owner)
	if err != nil {
		walk.MsgBox(owner, "详情窗口错误", err.Error(), walk.MsgBoxIconError)
		return
	}
	go func() {
		records, requestErr := client.InboundRecords(context.Background(), receipt.ID)
		dlg.Synchronize(func() {
			if requestErr != nil {
				walk.MsgBox(dlg, "收货记录查询失败", requestErr.Error(), walk.MsgBoxIconError)
				return
			}
			rows := make([]inboundRecordRow, 0)
			for _, record := range records {
				for _, material := range record.Materials {
					location := strings.Trim(strings.Join([]string{
						material.WarehouseName, material.WarehouseZoneName,
						material.WarehouseRackName, material.WarehouseBinName,
					}, " / "), " /")
					rows = append(rows, inboundRecordRow{
						BatchCode: record.Code,
						Time:      time.Unix(record.ReceivingDate, 0).Format("2006-01-02 15:04"),
						Material:  material.Name,
						Quantity:  fmt.Sprintf("%g %s", material.ActualQuantity, material.Unit),
						Location:  location,
						Operator:  record.CreatorName,
						Remark:    record.Remark,
					})
				}
			}
			if err := recordsTable.SetModel(rows); err != nil {
				walk.MsgBox(dlg, "收货记录展示失败", err.Error(), walk.MsgBoxIconError)
			}
		})
	}()
	dlg.Run()
}

func ReceiveInbound(owner walk.Form, client *api.Client, receipt api.InboundReceipt) bool {
	tree, err := client.WarehouseTree(context.Background())
	if err != nil {
		walk.MsgBox(owner, "无法加载仓储位置", err.Error(), walk.MsgBoxIconError)
		return false
	}
	carriers, carrierErr := client.Carriers(context.Background())
	if carrierErr != nil {
		walk.MsgBox(owner, "承运商加载失败", "仍可继续收货，但不能选择承运商：\r\n"+carrierErr.Error(), walk.MsgBoxIconWarning)
		carriers = nil
	}
	options := flattenPositions(tree)
	if len(options) == 0 {
		walk.MsgBox(owner, "没有可用库位", "线上仓库树没有可选择的仓储位置。", walk.MsgBoxIconWarning)
		return false
	}
	labels := make([]string, len(options))
	for index := range options {
		labels[index] = options[index].Label
	}
	carrierLabels := make([]string, 1, len(carriers)+1)
	carrierLabels[0] = "不选择承运商"
	for _, carrier := range carriers {
		label := carrier.Name
		if carrier.Code != "" {
			label += "（" + carrier.Code + "）"
		}
		carrierLabels = append(carrierLabels, label)
	}

	var dlg *walk.Dialog
	var batchEdit, remarkEdit, carrierCostEdit, otherCostEdit *walk.LineEdit
	var positionCombo, carrierCombo *walk.ComboBox
	var submitButton *walk.PushButton
	quantityEdits := make([]*walk.LineEdit, len(receipt.Materials))
	children := []Widget{
		Label{Text: "线上生产环境写操作", Font: Font{Bold: true}, TextColor: walk.RGB(190, 45, 35)},
		Label{Text: "入库单：" + receipt.Code + "。提交后会实际增加线上库存。"},
		Composite{Layout: Grid{Columns: 2, Spacing: 8}, Children: []Widget{
			Label{Text: "批次编号"},
			LineEdit{AssignTo: &batchEdit, Text: "WIN-" + time.Now().Format("20060102-150405")},
			Label{Text: "统一仓储位置"},
			ComboBox{AssignTo: &positionCombo, Model: labels, CurrentIndex: 0},
			Label{Text: "承运商"},
			ComboBox{AssignTo: &carrierCombo, Model: carrierLabels, CurrentIndex: 0},
			Label{Text: "运费"},
			LineEdit{AssignTo: &carrierCostEdit, Text: "0", CueBanner: "非负数字"},
			Label{Text: "其他费用"},
			LineEdit{AssignTo: &otherCostEdit, Text: "0", CueBanner: "非负数字"},
			Label{Text: "备注"},
			LineEdit{AssignTo: &remarkEdit, Text: "Windows 客户端批次收货"},
		}},
		Label{Text: "本次收货数量", Font: Font{Bold: true}},
	}
	materialGrid := []Widget{}
	for index, material := range receipt.Materials {
		remaining := material.EstimatedQuantity - material.ActualQuantity
		if remaining < 0 {
			remaining = 0
		}
		materialGrid = append(materialGrid,
			Label{Text: fmt.Sprintf("%s（剩余 %g %s）", material.Name, remaining, material.Unit)},
			LineEdit{AssignTo: &quantityEdits[index], Text: "0"},
		)
	}
	children = append(children, Composite{
		Layout:   Grid{Columns: 2, Spacing: 8},
		Children: materialGrid,
	})
	children = append(children, Composite{Layout: HBox{}, Children: []Widget{
		HSpacer{},
		PushButton{Text: "取消", OnClicked: func() { dlg.Cancel() }},
		PushButton{AssignTo: &submitButton, Text: "核对并提交"},
	}})

	var success bool
	err = Dialog{
		AssignTo: &dlg,
		Title:    "批次收货 - " + receipt.Code,
		MinSize:  Size{Width: 680, Height: 500},
		Size:     Size{Width: 760, Height: 620},
		Layout:   VBox{Margins: Margins{Left: 16, Top: 16, Right: 16, Bottom: 16}, Spacing: 10},
		Children: children,
	}.Create(owner)
	if err != nil {
		walk.MsgBox(owner, "收货窗口错误", err.Error(), walk.MsgBoxIconError)
		return false
	}

	submitButton.Clicked().Attach(func() {
		carrierID := ""
		if carrierCombo.CurrentIndex() > 0 && carrierCombo.CurrentIndex()-1 < len(carriers) {
			carrierID = carriers[carrierCombo.CurrentIndex()-1].ID
		}
		request, summary, validationErr := buildReceiveRequest(
			receipt, batchEdit.Text(), remarkEdit.Text(), carrierID,
			carrierCostEdit.Text(), otherCostEdit.Text(), options[positionCombo.CurrentIndex()], quantityEdits,
		)
		if validationErr != nil {
			walk.MsgBox(dlg, "数据不完整", validationErr.Error(), walk.MsgBoxIconWarning)
			return
		}
		if walk.MsgBox(dlg, "确认线上批次收货", summary+"\r\n\r\n该操作将实际写入线上库存，是否继续？", walk.MsgBoxYesNo|walk.MsgBoxIconWarning) != walk.DlgCmdYes {
			return
		}
		submitButton.SetEnabled(false)
		submitButton.SetText("正在提交，请勿关闭……")
		go func() {
			requestErr := client.ReceiveInbound(context.Background(), request)
			var verifyErr error
			if requestErr == nil {
				verifyErr = verifyInbound(client, request)
			}
			dlg.Synchronize(func() {
				submitButton.SetEnabled(true)
				submitButton.SetText("核对并提交")
				if requestErr != nil {
					walk.MsgBox(dlg, "提交未完成", "服务端未确认收货成功：\r\n"+requestErr.Error()+"\r\n\r\n客户端不会自动重试，请按批次编号查询后再决定。", walk.MsgBoxIconError)
					return
				}
				if verifyErr != nil {
					walk.MsgBox(dlg, "提交成功但复核异常", verifyErr.Error(), walk.MsgBoxIconWarning)
					success = true
					dlg.Accept()
					return
				}
				walk.MsgBox(dlg, "收货并复核成功", "批次记录和相关库存记录均已在线上查询到。", walk.MsgBoxIconInformation)
				success = true
				dlg.Accept()
			})
		}()
	})
	dlg.Run()
	return success
}

func buildReceiveRequest(
	receipt api.InboundReceipt,
	batchCode, remark, carrierID, carrierCostText, otherCostText string,
	position positionOption,
	edits []*walk.LineEdit,
) (api.ReceiveRequest, string, error) {
	batchCode = strings.TrimSpace(batchCode)
	if batchCode == "" {
		return api.ReceiveRequest{}, "", fmt.Errorf("批次编号不能为空")
	}
	carrierCost, err := parseNonNegativeNumber(carrierCostText, "运费")
	if err != nil {
		return api.ReceiveRequest{}, "", err
	}
	otherCost, err := parseNonNegativeNumber(otherCostText, "其他费用")
	if err != nil {
		return api.ReceiveRequest{}, "", err
	}
	request := api.ReceiveRequest{
		ID: receipt.ID, Code: batchCode, ReceivingDate: time.Now().Unix(), Remark: strings.TrimSpace(remark),
		CarrierID: carrierID, CarrierCost: carrierCost, OtherCost: otherCost,
		Materials: make([]api.ReceiveMaterial, 0, len(receipt.Materials)),
	}
	total := 0.0
	lines := []string{
		"入库单：" + receipt.Code, "批次：" + batchCode, "库位：" + position.Label,
		fmt.Sprintf("运费：%.2f    其他费用：%.2f", carrierCost, otherCost),
	}
	for index, material := range receipt.Materials {
		quantity, err := strconv.ParseFloat(strings.TrimSpace(edits[index].Text()), 64)
		if err != nil || quantity < 0 {
			return api.ReceiveRequest{}, "", fmt.Errorf("物料“%s”的数量格式不正确", material.Name)
		}
		remaining := material.EstimatedQuantity - material.ActualQuantity
		if quantity > remaining+0.0000001 {
			return api.ReceiveRequest{}, "", fmt.Errorf("物料“%s”本次数量 %g 超过剩余数量 %g", material.Name, quantity, remaining)
		}
		status := material.Status
		switch status {
		case "未发货", "在途", "部分入库", "作废", "入库完成":
		default:
			status = "未发货"
		}
		if quantity > 0 {
			status = "部分入库"
			if material.ActualQuantity+quantity >= material.EstimatedQuantity {
				status = "入库完成"
			}
			total += quantity
			lines = append(lines, fmt.Sprintf("%s：%g %s", material.Name, quantity, material.Unit))
		}
		var positionIDs []string
		if quantity > 0 {
			positionIDs = append([]string(nil), position.IDs...)
		}
		request.Materials = append(request.Materials, api.ReceiveMaterial{
			Index: material.Index, ID: material.ID, Price: material.Price,
			ActualQuantity: quantity, Position: positionIDs, Status: status,
		})
	}
	if total <= 0 {
		return api.ReceiveRequest{}, "", fmt.Errorf("本次收货总数量必须大于 0")
	}
	lines = append(lines, fmt.Sprintf("本次合计：%g", total))
	return request, strings.Join(lines, "\r\n"), nil
}

func parseNonNegativeNumber(text, field string) (float64, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0, nil
	}
	value, err := strconv.ParseFloat(text, 64)
	if err != nil || value < 0 {
		return 0, fmt.Errorf("%s必须是非负数字", field)
	}
	return value, nil
}

func verifyInbound(client *api.Client, request api.ReceiveRequest) error {
	records, err := client.InboundRecords(context.Background(), request.ID)
	if err != nil {
		return fmt.Errorf("批次已提交，但查询收货记录失败：%w", err)
	}
	found := false
	for _, record := range records {
		if record.Code == request.Code {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("服务端返回成功，但收货记录中未找到批次 %s；请人工复核，禁止重复提交", request.Code)
	}
	for _, material := range request.Materials {
		if material.ActualQuantity <= 0 {
			continue
		}
		items, queryErr := client.InventoryByMaterial(context.Background(), material.ID)
		if queryErr != nil {
			return fmt.Errorf("批次记录存在，但物料库存复核失败：%w", queryErr)
		}
		inventoryFound := false
		for _, item := range items {
			if item.ReceiveCode == request.Code {
				inventoryFound = true
				break
			}
		}
		if !inventoryFound {
			return fmt.Errorf("批次记录存在，但物料 %s 未查询到批次 %s 对应的库存记录", material.ID, request.Code)
		}
	}
	return nil
}

func flattenPositions(nodes []api.WarehouseNode) []positionOption {
	var result []positionOption
	var walkNodes func([]api.WarehouseNode, []string, []string)
	walkNodes = func(items []api.WarehouseNode, names, ids []string) {
		for _, node := range items {
			nextNames := append(append([]string(nil), names...), node.Name)
			nextIDs := append(append([]string(nil), ids...), node.ID)
			if len(node.Children) == 0 {
				result = append(result, positionOption{Label: strings.Join(nextNames, " / "), IDs: nextIDs})
				continue
			}
			walkNodes(node.Children, nextNames, nextIDs)
		}
	}
	walkNodes(nodes, nil, nil)
	return result
}
