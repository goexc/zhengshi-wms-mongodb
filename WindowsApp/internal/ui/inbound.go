package ui

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
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

type inboundBatchRow struct {
	BatchCode   string
	Time        string
	Carrier     string
	CarrierCost string
	OtherCost   string
	Total       string
	Attachments string
	Operator    string
	Remark      string
}

type inboundBatchMaterialRow struct {
	Index    string
	Material string
	Model    string
	Quantity string
	Status   string
	Location string
}

func ShowInboundDetail(owner walk.Form, client *api.Client, imageBaseURL string, receipt api.InboundReceipt) {
	var dlg *walk.Dialog
	var batchesTable, batchMaterialsTable *walk.TableView
	var receiptAttachmentButton, batchAttachmentButton *walk.PushButton
	var recordInfo *walk.Label
	var records []api.InboundRecord
	var updateSelectedBatch func()
	var closed atomic.Bool
	ctx, cancel := context.WithCancel(context.Background())
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
		"单号：%s    类型：%s    状态：%s\r\n供应商/客户：%s    附件：%d 张\r\n备注：%s",
		receipt.Code, receipt.Type, receipt.Status, business, len(receipt.Annex), receipt.Remark,
	)
	err := Dialog{
		AssignTo: &dlg,
		Title:    "入库详情 - " + receipt.Code,
		MinSize:  Size{Width: 900, Height: 680},
		Size:     Size{Width: 1080, Height: 780},
		Layout:   VBox{Margins: Margins{Left: 14, Top: 14, Right: 14, Bottom: 14}},
		Children: []Widget{
			TextEdit{Text: header, ReadOnly: true, MinSize: Size{Height: 70}, MaxSize: Size{Height: 90}},
			Label{Text: "物料进度", Font: Font{Bold: true}},
			TableView{
				Model:            materialRows,
				AlternatingRowBG: true,
				MinSize:          Size{Height: 160},
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
			Label{AssignTo: &recordInfo, Text: "正在加载收货批次……", TextColor: walk.RGB(80, 80, 80)},
			VSplitter{
				StretchFactor: 1,
				Children: []Widget{
					GroupBox{
						Title:         "收货批次",
						StretchFactor: 1,
						Layout:        VBox{Margins: Margins{Left: 8, Top: 8, Right: 8, Bottom: 8}},
						Children: []Widget{TableView{
							AssignTo: &batchesTable, Model: []inboundBatchRow{}, AlternatingRowBG: true,
							ColumnsOrderable: true, StretchFactor: 1,
							Accessibility: Accessibility{Name: "入库收货批次列表", Description: "选择批次后查看物料和附件"},
							OnCurrentIndexChanged: func() {
								if updateSelectedBatch != nil {
									updateSelectedBatch()
								}
							},
							Columns: []TableViewColumn{
								{Title: "批次编号", DataMember: "BatchCode", Width: 130},
								{Title: "收货时间", DataMember: "Time", Width: 125},
								{Title: "承运商", DataMember: "Carrier", Width: 110},
								{Title: "运费", DataMember: "CarrierCost", Width: 75},
								{Title: "其他费用", DataMember: "OtherCost", Width: 75},
								{Title: "批次总额", DataMember: "Total", Width: 85},
								{Title: "附件", DataMember: "Attachments", Width: 60},
								{Title: "操作人", DataMember: "Operator", Width: 85},
								{Title: "备注", DataMember: "Remark", Width: 150},
							},
						}},
					},
					GroupBox{
						Title:         "当前批次物料及库位",
						StretchFactor: 1,
						Layout:        VBox{Margins: Margins{Left: 8, Top: 8, Right: 8, Bottom: 8}},
						Children: []Widget{TableView{
							AssignTo: &batchMaterialsTable, Model: []inboundBatchMaterialRow{}, AlternatingRowBG: true,
							ColumnsOrderable: true, StretchFactor: 1,
							Accessibility: Accessibility{Name: "当前收货批次物料列表"},
							Columns: []TableViewColumn{
								{Title: "序号", DataMember: "Index", Width: 55},
								{Title: "物料", DataMember: "Material", Width: 190},
								{Title: "型号", DataMember: "Model", Width: 110},
								{Title: "数量", DataMember: "Quantity", Width: 95},
								{Title: "状态", DataMember: "Status", Width: 90},
								{Title: "仓储位置", DataMember: "Location", Width: 260},
							},
						}},
					},
				},
			},
			Composite{Layout: HBox{Spacing: 8}, Children: []Widget{
				PushButton{
					AssignTo: &batchAttachmentButton, Text: "当前批次附件", Enabled: false,
					MinSize: Size{Width: 120, Height: 30}, Accessibility: Accessibility{Name: "查看当前收货批次图片附件"},
					OnClicked: func() {
						index := batchesTable.CurrentIndex()
						if index >= 0 && index < len(records) {
							ShowOrderAttachments(dlg, client, imageBaseURL, "收货批次 "+records[index].Code, records[index].Annex)
						}
					},
				},
				HSpacer{},
				PushButton{
					AssignTo: &receiptAttachmentButton, Text: fmt.Sprintf("入库单附件 (%d)", len(receipt.Annex)),
					Enabled: len(receipt.Annex) > 0, MinSize: Size{Width: 110, Height: 30},
					Accessibility: Accessibility{Name: "查看入库单图片附件"},
					OnClicked: func() {
						ShowOrderAttachments(dlg, client, imageBaseURL, "入库单 "+receipt.Code, receipt.Annex)
					},
				},
				PushButton{Text: "关闭", MinSize: Size{Width: 88, Height: 30}, OnClicked: func() { dlg.Accept() }},
			}},
		},
	}.Create(owner)
	if err != nil {
		cancel()
		walk.MsgBox(owner, "详情窗口错误", err.Error(), walk.MsgBoxIconError)
		return
	}
	dlg.Disposing().Attach(func() {
		closed.Store(true)
		cancel()
	})
	updateSelectedBatch = func() {
		index := batchesTable.CurrentIndex()
		if index < 0 || index >= len(records) {
			_ = batchMaterialsTable.SetModel([]inboundBatchMaterialRow{})
			batchAttachmentButton.SetEnabled(false)
			batchAttachmentButton.SetText("当前批次附件")
			return
		}
		record := records[index]
		_ = batchMaterialsTable.SetModel(inboundBatchMaterialRows(record))
		batchAttachmentButton.SetEnabled(len(record.Annex) > 0)
		batchAttachmentButton.SetText(fmt.Sprintf("当前批次附件 (%d)", len(record.Annex)))
	}
	go func() {
		loaded, requestErr := client.InboundRecords(ctx, receipt.ID)
		if closed.Load() || ctx.Err() != nil {
			return
		}
		dlg.Synchronize(func() {
			if closed.Load() {
				return
			}
			if requestErr != nil {
				recordInfo.SetText("收货批次加载失败：" + requestErr.Error() + "。请关闭后重试。")
				return
			}
			records = loaded
			if err := batchesTable.SetModel(inboundBatchRows(records)); err != nil {
				recordInfo.SetText("收货批次展示失败：" + err.Error())
				return
			}
			if len(records) == 0 {
				recordInfo.SetText("暂无收货批次。")
				updateSelectedBatch()
				return
			}
			recordInfo.SetText(fmt.Sprintf("共 %d 个收货批次；选择批次可查看物料、库位和附件。", len(records)))
			_ = batchesTable.SetCurrentIndex(0)
			updateSelectedBatch()
		})
	}()
	dlg.Run()
}

func inboundBatchRows(records []api.InboundRecord) []inboundBatchRow {
	rows := make([]inboundBatchRow, 0, len(records))
	for _, record := range records {
		rows = append(rows, inboundBatchRow{
			BatchCode: record.Code, Time: formatUnixMinute(record.ReceivingDate),
			Carrier:     displayMaterialValue(record.CarrierName),
			CarrierCost: fmt.Sprintf("%.2f", record.CarrierCost), OtherCost: fmt.Sprintf("%.2f", record.OtherCost),
			Total: fmt.Sprintf("%.2f", record.TotalAmount), Attachments: fmt.Sprint(len(record.Annex)),
			Operator: record.CreatorName, Remark: record.Remark,
		})
	}
	return rows
}

func inboundBatchMaterialRows(record api.InboundRecord) []inboundBatchMaterialRow {
	rows := make([]inboundBatchMaterialRow, 0, len(record.Materials))
	for _, material := range record.Materials {
		location := strings.Trim(strings.Join([]string{
			material.WarehouseName, material.WarehouseZoneName,
			material.WarehouseRackName, material.WarehouseBinName,
		}, " / "), " /")
		rows = append(rows, inboundBatchMaterialRow{
			Index: fmt.Sprint(material.Index + 1), Material: material.Name, Model: material.Model,
			Quantity: fmt.Sprintf("%g %s", material.ActualQuantity, material.Unit),
			Status:   material.Status, Location: location,
		})
	}
	return rows
}

func ReceiveInbound(owner walk.Form, client *api.Client, receipt api.InboundReceipt) bool {
	var dlg *walk.Dialog
	var batchEdit, remarkEdit, carrierCostEdit, otherCostEdit *walk.LineEdit
	var positionCombo, carrierCombo *walk.ComboBox
	var dependencyLabel *walk.Label
	var retryButton, submitButton, cancelButton *walk.PushButton
	var options []positionOption
	var carriers []api.Carrier
	var warehouseReady bool
	var operationBusy bool
	var dependencyGeneration int
	var dependencyCancel context.CancelFunc
	var closed atomic.Bool
	quantityEdits := make([]*walk.LineEdit, len(receipt.Materials))
	children := []Widget{
		Label{Text: "线上生产环境写操作", Font: Font{Bold: true}, TextColor: walk.RGB(190, 45, 35)},
		Label{Text: "入库单：" + receipt.Code + "。提交后会实际增加线上库存。"},
		Composite{Layout: HBox{Spacing: 8}, Children: []Widget{
			Label{
				AssignTo: &dependencyLabel, Text: "正在加载仓储位置和承运商……",
				TextColor: walk.RGB(75, 75, 75), Accessibility: Accessibility{Name: "收货依赖数据加载状态"},
			},
			HSpacer{},
			PushButton{
				AssignTo: &retryButton, Text: "重新加载", Visible: false, MinSize: Size{Width: 88, Height: 30},
				Accessibility: Accessibility{Name: "重新加载仓储位置和承运商"},
			},
		}},
		Composite{Layout: Grid{Columns: 2, Spacing: 8}, Children: []Widget{
			Label{Text: "批次编号"},
			LineEdit{AssignTo: &batchEdit, Text: "WIN-" + time.Now().Format("20060102-150405")},
			Label{Text: "统一仓储位置"},
			ComboBox{AssignTo: &positionCombo, Model: []string{"正在加载仓储位置……"}, CurrentIndex: 0, Enabled: false},
			Label{Text: "承运商"},
			ComboBox{AssignTo: &carrierCombo, Model: []string{"正在加载承运商……"}, CurrentIndex: 0, Enabled: false},
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
		PushButton{AssignTo: &cancelButton, Text: "取消", OnClicked: func() { dlg.Cancel() }},
		PushButton{AssignTo: &submitButton, Text: "核对并提交", Enabled: false},
	}})

	var success bool
	err := Dialog{
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
	setSubmitAvailability := func() {
		submitButton.SetEnabled(warehouseReady && !operationBusy)
	}

	var loadDependencies func()
	loadDependencies = func() {
		if operationBusy || closed.Load() {
			return
		}
		if dependencyCancel != nil {
			dependencyCancel()
		}
		ctx, cancel := context.WithCancel(context.Background())
		dependencyCancel = cancel
		dependencyGeneration++
		generation := dependencyGeneration
		warehouseReady = false
		setSubmitAvailability()
		retryButton.SetVisible(false)
		positionCombo.SetEnabled(false)
		carrierCombo.SetEnabled(false)
		_ = positionCombo.SetModel([]string{"正在加载仓储位置……"})
		_ = positionCombo.SetCurrentIndex(0)
		_ = carrierCombo.SetModel([]string{"正在加载承运商……"})
		_ = carrierCombo.SetCurrentIndex(0)
		dependencyLabel.SetText("正在加载仓储位置和承运商……")

		go func() {
			var tree []api.WarehouseNode
			var loadedCarriers []api.Carrier
			var treeErr, carrierErr error
			var wait sync.WaitGroup
			wait.Add(2)
			go func() {
				defer wait.Done()
				tree, treeErr = client.WarehouseTree(ctx)
			}()
			go func() {
				defer wait.Done()
				loadedCarriers, carrierErr = client.Carriers(ctx)
			}()
			wait.Wait()
			if ctx.Err() != nil || closed.Load() {
				return
			}
			loadedOptions := flattenPositions(tree)
			if treeErr == nil && len(loadedOptions) == 0 {
				treeErr = fmt.Errorf("线上仓库树没有可选择的仓储位置")
			}
			dlg.Synchronize(func() {
				if closed.Load() || generation != dependencyGeneration {
					return
				}
				carriers = loadedCarriers
				carrierLabels := make([]string, 1, len(carriers)+1)
				carrierLabels[0] = "不选择承运商"
				if carrierErr != nil {
					carriers = nil
					carrierLabels = []string{"不选择承运商（加载失败）"}
				}
				for _, carrier := range carriers {
					carrierLabels = append(carrierLabels, businessOptionLabel(carrier.Name, carrier.Code))
				}
				_ = carrierCombo.SetModel(carrierLabels)
				_ = carrierCombo.SetCurrentIndex(0)
				carrierCombo.SetEnabled(carrierErr == nil && len(carriers) > 0)

				if treeErr != nil {
					options = nil
					_ = positionCombo.SetModel([]string{"仓储位置加载失败"})
					_ = positionCombo.SetCurrentIndex(0)
					positionCombo.SetEnabled(false)
					dependencyLabel.SetText("仓储位置加载失败：" + treeErr.Error() + "。请重新加载后再提交。")
					retryButton.SetVisible(true)
					setSubmitAvailability()
					return
				}

				options = loadedOptions
				labels := make([]string, len(options))
				for index := range options {
					labels[index] = options[index].Label
				}
				_ = positionCombo.SetModel(labels)
				_ = positionCombo.SetCurrentIndex(0)
				positionCombo.SetEnabled(true)
				warehouseReady = true
				if carrierErr != nil {
					dependencyLabel.SetText("仓储位置已加载；承运商加载失败，可不选承运商继续，或点击重新加载。")
					retryButton.SetVisible(true)
				} else {
					dependencyLabel.SetText(fmt.Sprintf("已加载 %d 个仓储位置、%d 个承运商。", len(options), len(carriers)))
					retryButton.SetVisible(false)
				}
				setSubmitAvailability()
			})
		}()
	}
	retryButton.Clicked().Attach(loadDependencies)
	dlg.Disposing().Attach(func() {
		closed.Store(true)
		if dependencyCancel != nil {
			dependencyCancel()
		}
	})
	dlg.Closing().Attach(func(canceled *bool, _ walk.CloseReason) {
		if operationBusy {
			*canceled = true
			walk.MsgBox(dlg, "正在提交", "收货请求正在提交和复核，请等待结果，避免产生未知状态。", walk.MsgBoxIconInformation)
		}
	})

	submitButton.Clicked().Attach(func() {
		positionIndex := positionCombo.CurrentIndex()
		if !warehouseReady || positionIndex < 0 || positionIndex >= len(options) {
			walk.MsgBox(dlg, "仓储位置不可用", "请先成功加载并选择仓储位置。", walk.MsgBoxIconWarning)
			return
		}
		carrierID := ""
		if carrierCombo.CurrentIndex() > 0 && carrierCombo.CurrentIndex()-1 < len(carriers) {
			carrierID = carriers[carrierCombo.CurrentIndex()-1].ID
		}
		request, summary, validationErr := buildReceiveRequest(
			receipt, batchEdit.Text(), remarkEdit.Text(), carrierID,
			carrierCostEdit.Text(), otherCostEdit.Text(), options[positionIndex], quantityEdits,
		)
		if validationErr != nil {
			walk.MsgBox(dlg, "数据不完整", validationErr.Error(), walk.MsgBoxIconWarning)
			return
		}
		if walk.MsgBox(dlg, "确认线上批次收货", summary+"\r\n\r\n该操作将实际写入线上库存，是否继续？", walk.MsgBoxYesNo|walk.MsgBoxIconWarning) != walk.DlgCmdYes {
			return
		}
		operationBusy = true
		setSubmitAvailability()
		cancelButton.SetEnabled(false)
		retryButton.SetEnabled(false)
		submitButton.SetText("正在提交，请勿关闭……")
		go func() {
			requestErr := client.ReceiveInbound(context.Background(), request)
			var verifyErr error
			if requestErr == nil {
				verifyErr = verifyInbound(client, request)
			}
			dlg.Synchronize(func() {
				operationBusy = false
				setSubmitAvailability()
				cancelButton.SetEnabled(true)
				retryButton.SetEnabled(true)
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
	loadDependencies()
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
