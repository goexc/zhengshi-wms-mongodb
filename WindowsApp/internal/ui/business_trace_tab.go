package ui

import (
	"fmt"
	"strings"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"

	"zhengshi-wms-windowsapp/internal/api"
	"zhengshi-wms-windowsapp/internal/config"
)

type inboundTraceBatchRow struct {
	Source      string
	BatchCode   string
	Time        string
	Carrier     string
	Quantity    string
	Location    string
	Attachments string
	Operator    string
	Remark      string
}

func (ui *mainUI) replaceBusinessTraceTab(pageDecl TabPage) {
	if ui.tabs == nil {
		return
	}
	if ui.businessTraceTab != nil {
		if index := ui.tabs.Pages().Index(ui.businessTraceTab); index >= 0 {
			_ = ui.tabs.Pages().RemoveAt(index)
		}
		ui.businessTraceTab.Dispose()
		ui.businessTraceTab = nil
	}
	if err := pageDecl.Create(NewBuilder(nil)); err != nil {
		walk.MsgBox(ui.window, "无法打开追溯标签", err.Error(), walk.MsgBoxIconError)
		return
	}
	insertAt := ui.tabs.Pages().Len()
	if ui.systemTab != nil {
		if index := ui.tabs.Pages().Index(ui.systemTab); index >= 0 {
			insertAt = index
		}
	}
	if err := ui.tabs.Pages().Insert(insertAt, ui.businessTraceTab); err != nil {
		ui.businessTraceTab.Dispose()
		ui.businessTraceTab = nil
		walk.MsgBox(ui.window, "无法打开追溯标签", err.Error(), walk.MsgBoxIconError)
		return
	}
	_ = ui.tabs.SetCurrentIndex(ui.tabs.Pages().Index(ui.businessTraceTab))
}

func (ui *mainUI) openOutboundTraceTab(order api.OutboundOrder, materials []api.OutboundMaterial) {
	rows := make([]outboundDetailMaterialRow, 0, len(materials))
	for _, material := range materials {
		rows = append(rows, outboundDetailMaterialRow{
			Index: fmt.Sprint(material.Index), Name: material.Name, Model: material.Model,
			Specification: material.Specification, Quantity: fmt.Sprintf("%g %s", material.Quantity, material.Unit),
			Price: fmt.Sprintf("%.3f", material.Price), Amount: fmt.Sprintf("%.3f", material.Price*material.Quantity),
		})
	}
	business := strings.TrimSpace(order.CustomerName)
	if business == "" {
		business = strings.TrimSpace(order.SupplierName)
	}
	header := fmt.Sprintf(
		"单号：%s    类型：%s    状态：%s\r\n客户/供应商：%s    金额：%.3f 元    附件：%d 张\r\n出库时间：%s    签收时间：%s\r\n备注：%s",
		order.Code, order.Type, order.Status, displayMaterialValue(business), order.TotalAmount, len(order.Annex),
		formatUnixMinute(order.DepartureTime), formatUnixMinute(order.ReceiptTime), displayMaterialValue(order.Remark),
	)
	page := TabPage{
		AssignTo: &ui.businessTraceTab,
		Title:    closableTabTitle("出库追溯 · " + order.Code),
		Layout:   VBox{Margins: Margins{Left: 14, Top: 14, Right: 14, Bottom: 12}, Spacing: 10},
		Children: []Widget{
			Label{Text: "出库单追溯", Font: Font{Family: "Microsoft YaHei UI", PointSize: 15, Bold: true}},
			Label{Text: "从来源页面打开的只读追溯标签；关闭后来源筛选、分页和选中状态保持不变。", TextColor: secondaryTextColor()},
			TextEdit{Text: header, ReadOnly: true, MinSize: Size{Height: 88}, MaxSize: Size{Height: 105}, Accessibility: Accessibility{Name: "出库单追溯基本信息"}},
			GroupBox{
				Title: "物料明细", StretchFactor: 1,
				Layout: VBox{Margins: Margins{Left: 10, Top: 10, Right: 10, Bottom: 10}},
				Children: []Widget{TableView{
					Model: rows, AlternatingRowBG: true, ColumnsOrderable: true, StretchFactor: 1,
					Accessibility: Accessibility{Name: "出库追溯物料明细"},
					Columns: []TableViewColumn{
						{Title: "序号", DataMember: "Index", Width: 55}, {Title: "物料", DataMember: "Name", Width: 190},
						{Title: "型号", DataMember: "Model", Width: 120}, {Title: "规格", DataMember: "Specification", Width: 150},
						{Title: "数量", DataMember: "Quantity", Width: 100}, {Title: "单价", DataMember: "Price", Width: 90},
						{Title: "小计", DataMember: "Amount", Width: 100},
					},
				}},
			},
			Composite{Layout: HBox{Spacing: 8}, Children: []Widget{
				Label{Text: "数据来自现有出库分页和物料接口。", TextColor: secondaryTextColor()},
				HSpacer{},
				PushButton{
					Text: fmt.Sprintf("查看附件 (%d)", len(order.Annex)), Enabled: len(order.Annex) > 0,
					MinSize: Size{Width: 110, Height: 30}, OnClicked: func() {
						ShowOrderAttachments(ui.window, ui.session.Client, config.ImageBaseURL(), "出库单 "+order.Code, order.Annex)
					},
				},
				PushButton{Text: "关闭追溯标签", MinSize: Size{Width: 110, Height: 30}, OnClicked: ui.closeCurrentTab},
			}},
		},
	}
	ui.replaceBusinessTraceTab(page)
}

func inboundTraceRows(records []api.InboundRecord, sourceBatch string) []inboundTraceBatchRow {
	rows := make([]inboundTraceBatchRow, 0, len(records))
	for _, record := range records {
		quantity := 0.0
		locations := make([]string, 0)
		seen := make(map[string]bool)
		for _, material := range record.Materials {
			quantity += material.ActualQuantity
			location := strings.Trim(strings.Join([]string{material.WarehouseName, material.WarehouseZoneName, material.WarehouseRackName, material.WarehouseBinName}, " / "), " /")
			if location != "" && !seen[location] {
				seen[location] = true
				locations = append(locations, location)
			}
		}
		source := ""
		if strings.EqualFold(strings.TrimSpace(record.Code), strings.TrimSpace(sourceBatch)) {
			source = "当前来源"
		}
		rows = append(rows, inboundTraceBatchRow{
			Source: source, BatchCode: record.Code, Time: formatUnixMinute(record.ReceivingDate), Carrier: record.CarrierName,
			Quantity: fmt.Sprintf("%g", quantity), Location: strings.Join(locations, "；"), Attachments: fmt.Sprint(len(record.Annex)),
			Operator: record.CreatorName, Remark: record.Remark,
		})
	}
	return rows
}

func (ui *mainUI) openInboundTraceTab(receipt api.InboundReceipt, records []api.InboundRecord, sourceBatch string) {
	materials := make([]inboundMaterialDetailRow, 0, len(receipt.Materials))
	for _, material := range receipt.Materials {
		remaining := material.EstimatedQuantity - material.ActualQuantity
		if remaining < 0 {
			remaining = 0
		}
		materials = append(materials, inboundMaterialDetailRow{
			Index: fmt.Sprint(material.Index + 1), Name: material.Name, Model: material.Model,
			Planned: fmt.Sprintf("%g %s", material.EstimatedQuantity, material.Unit), Received: fmt.Sprintf("%g", material.ActualQuantity),
			Remaining: fmt.Sprintf("%g", remaining), Status: material.Status,
		})
	}
	business := receipt.SupplierName
	if strings.TrimSpace(business) == "" {
		business = receipt.CustomerName
	}
	header := fmt.Sprintf("单号：%s    类型：%s    状态：%s\r\n供应商/客户：%s    来源批次：%s    附件：%d 张\r\n备注：%s",
		receipt.Code, receipt.Type, receipt.Status, displayMaterialValue(business), displayMaterialValue(sourceBatch), len(receipt.Annex), displayMaterialValue(receipt.Remark))
	page := TabPage{
		AssignTo: &ui.businessTraceTab,
		Title:    closableTabTitle("入库追溯 · " + receipt.Code),
		Layout:   VBox{Margins: Margins{Left: 14, Top: 14, Right: 14, Bottom: 12}, Spacing: 8},
		Children: []Widget{
			Label{Text: "入库来源追溯", Font: Font{Family: "Microsoft YaHei UI", PointSize: 15, Bold: true}},
			Label{Text: "从库存记录回查入库单及其收货批次；“当前来源”按库存记录返回的批次编号标记。", TextColor: secondaryTextColor()},
			TextEdit{Text: header, ReadOnly: true, MinSize: Size{Height: 70}, MaxSize: Size{Height: 88}, Accessibility: Accessibility{Name: "入库来源基本信息"}},
			VSplitter{StretchFactor: 1, Children: []Widget{
				GroupBox{Title: "入库单物料进度", StretchFactor: 1, Layout: VBox{Margins: Margins{Left: 8, Top: 8, Right: 8, Bottom: 8}}, Children: []Widget{TableView{
					Model: materials, AlternatingRowBG: true, ColumnsOrderable: true, StretchFactor: 1,
					Columns: []TableViewColumn{
						{Title: "序号", DataMember: "Index", Width: 55}, {Title: "物料", DataMember: "Name", Width: 185},
						{Title: "型号", DataMember: "Model", Width: 110}, {Title: "计划", DataMember: "Planned", Width: 100},
						{Title: "已收", DataMember: "Received", Width: 80}, {Title: "剩余", DataMember: "Remaining", Width: 80},
						{Title: "状态", DataMember: "Status", Width: 95},
					},
				}}},
				GroupBox{Title: "收货批次", StretchFactor: 1, Layout: VBox{Margins: Margins{Left: 8, Top: 8, Right: 8, Bottom: 8}}, Children: []Widget{TableView{
					Model: inboundTraceRows(records, sourceBatch), AlternatingRowBG: true, ColumnsOrderable: true, StretchFactor: 1,
					Columns: []TableViewColumn{
						{Title: "来源", DataMember: "Source", Width: 85}, {Title: "批次编号", DataMember: "BatchCode", Width: 135},
						{Title: "收货时间", DataMember: "Time", Width: 130}, {Title: "承运商", DataMember: "Carrier", Width: 110},
						{Title: "数量", DataMember: "Quantity", Width: 80}, {Title: "仓储位置", DataMember: "Location", Width: 280},
						{Title: "附件", DataMember: "Attachments", Width: 60}, {Title: "操作人", DataMember: "Operator", Width: 90},
						{Title: "备注", DataMember: "Remark", Width: 160},
					},
				}}},
			}},
			Composite{Layout: HBox{Spacing: 8}, Children: []Widget{
				Label{Text: fmt.Sprintf("共 %d 个收货批次。", len(records)), TextColor: secondaryTextColor()},
				HSpacer{},
				PushButton{
					Text: fmt.Sprintf("入库单附件 (%d)", len(receipt.Annex)), Enabled: len(receipt.Annex) > 0,
					MinSize: Size{Width: 110, Height: 30}, OnClicked: func() {
						ShowOrderAttachments(ui.window, ui.session.Client, config.ImageBaseURL(), "入库单 "+receipt.Code, receipt.Annex)
					},
				},
				PushButton{Text: "关闭追溯标签", MinSize: Size{Width: 110, Height: 30}, OnClicked: ui.closeCurrentTab},
			}},
		},
	}
	ui.replaceBusinessTraceTab(page)
}
