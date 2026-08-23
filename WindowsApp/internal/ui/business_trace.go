package ui

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"

	"zhengshi-wms-windowsapp/internal/api"
)

type outboundDetailMaterialRow struct {
	Index         string
	Name          string
	Model         string
	Specification string
	Quantity      string
	Price         string
	Amount        string
}

func (ui *mainUI) beginBusinessTrace() (context.Context, int) {
	if ui.businessTraceCancel != nil {
		ui.businessTraceCancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	ui.businessTraceCancel = cancel
	ui.businessTraceGeneration++
	return ctx, ui.businessTraceGeneration
}

func (ui *mainUI) showSelectedOutboundDetail() {
	order, ok := ui.selectedOutbound()
	if !ok {
		return
	}
	ui.showOutboundDetailByCode(order.Code)
}

func (ui *mainUI) showOutboundDetailByCode(code string) {
	code = strings.TrimSpace(code)
	if code == "" {
		walk.MsgBox(ui.window, "无法追溯", "当前记录没有出库单号。", walk.MsgBoxIconInformation)
		return
	}
	if ui.status != nil {
		ui.status.SetText("正在追溯出库单 " + code + "……")
	}
	ctx, generation := ui.beginBusinessTrace()
	guardedGo(func() {
		order, err := ui.session.Client.FindOutboundByCode(ctx, code)
		var materials []api.OutboundMaterial
		if err == nil {
			materials, err = ui.session.Client.OutboundMaterials(ctx, code)
		}
		if ctx.Err() != nil {
			return
		}
		ui.window.Synchronize(func() {
			if generation != ui.businessTraceGeneration {
				return
			}
			if err != nil {
				walk.MsgBox(ui.window, "出库追溯失败", err.Error()+"。请检查网络或刷新来源页面后重试。", walk.MsgBoxIconError)
				return
			}
			ui.openOutboundTraceTab(order, materials)
		})
	})
}

func (ui *mainUI) updateOutboundReportDetailButton() {
	state := ui.outboundReport
	if state == nil || state.detail == nil || state.table == nil {
		return
	}
	index := state.table.CurrentIndex()
	state.detail.SetEnabled(index >= 0 && index < len(state.rows) && reportOrderCode(state.rows[index]) != "")
}

func (ui *mainUI) showSelectedOutboundReportOrder() {
	state := ui.outboundReport
	if state == nil || state.table == nil {
		return
	}
	index := state.table.CurrentIndex()
	if index < 0 || index >= len(state.rows) {
		walk.MsgBox(ui.window, "请选择报表行", "请先选择一条出库报表明细。", walk.MsgBoxIconInformation)
		return
	}
	ui.showOutboundDetailByCode(reportOrderCode(state.rows[index]))
}

func (ui *mainUI) updateInventoryTraceButton() {
	if ui.inventoryTrace == nil || ui.inventoryTable == nil {
		return
	}
	index := ui.inventoryTable.CurrentIndex()
	ui.inventoryTrace.SetEnabled(index >= 0 && index < len(ui.inventoryRows) && strings.TrimSpace(ui.inventoryRows[index].Detail.ReceiptCode) != "")
}

func (ui *mainUI) showSelectedInventorySource() {
	if ui.inventoryTable == nil {
		return
	}
	index := ui.inventoryTable.CurrentIndex()
	if index < 0 || index >= len(ui.inventoryRows) {
		walk.MsgBox(ui.window, "请选择库存", "请先选择一条库存记录。", walk.MsgBoxIconInformation)
		return
	}
	item := ui.inventoryRows[index].Detail
	code := strings.TrimSpace(item.ReceiptCode)
	if code == "" {
		walk.MsgBox(ui.window, "没有来源单号", "该库存记录没有返回来源入库单号，客户端无法继续追溯。", walk.MsgBoxIconInformation)
		return
	}
	ui.inventoryInfo.SetText("正在追溯入库单 " + code + "……")
	ctx, generation := ui.beginBusinessTrace()
	guardedGo(func() {
		page, err := ui.session.Client.InboundReceipts(ctx, 1, 100, api.InboundFilters{Code: code})
		var receipt api.InboundReceipt
		var records []api.InboundRecord
		found := false
		if err == nil {
			for _, candidate := range page.List {
				if strings.EqualFold(strings.TrimSpace(candidate.Code), code) {
					receipt = candidate
					found = true
					break
				}
			}
			if found {
				records, err = ui.session.Client.InboundRecords(ctx, receipt.ID)
			}
		}
		if ctx.Err() != nil {
			return
		}
		ui.window.Synchronize(func() {
			if generation != ui.businessTraceGeneration || ui.inventoryInfo == nil {
				return
			}
			if err != nil {
				ui.inventoryInfo.SetText("入库追溯失败：" + err.Error())
				return
			}
			if !found {
				ui.inventoryInfo.SetText("未查询到来源入库单 " + code + "。")
				return
			}
			ui.openInboundTraceTab(receipt, records, item.ReceiveCode)
			if strings.TrimSpace(item.ReceiveCode) != "" {
				ui.inventoryInfo.SetText("已打开来源入库单；批次参考：" + item.ReceiveCode)
			}
		})
	})
}

type materialInventoryRow struct {
	Type      string
	Receipt   string
	Batch     string
	Warehouse string
	Location  string
	Quantity  string
	Available string
	Locked    string
	Frozen    string
}

func ShowMaterialInventoryPositions(owner walk.Form, client *api.Client, material api.Material) {
	var dlg *walk.Dialog
	var table *walk.TableView
	var info *walk.Label
	var refresh, closeButton *walk.PushButton
	var closed atomic.Bool
	ctx, cancel := context.WithCancel(context.Background())
	var load func()
	err := Dialog{
		AssignTo: &dlg,
		Title:    "库存位置 - " + displayMaterialValue(material.Model),
		MinSize:  Size{Width: 820, Height: 460},
		Size:     Size{Width: 1040, Height: 600},
		Layout:   VBox{Margins: Margins{Left: 16, Top: 16, Right: 16, Bottom: 14}, Spacing: 10},
		Children: []Widget{
			Label{Text: material.Name, Font: Font{Family: "Microsoft YaHei UI", PointSize: 15, Bold: true}},
			Label{Text: "只读展示现有 /inventory/list 接口返回的库存位置。", TextColor: secondaryTextColor()},
			TableView{
				AssignTo: &table, Model: []materialInventoryRow{}, AlternatingRowBG: true, ColumnsOrderable: true, StretchFactor: 1,
				Accessibility: Accessibility{Name: "物料库存位置列表"},
				Columns: []TableViewColumn{
					{Title: "类型", DataMember: "Type", Width: 90},
					{Title: "入库单", DataMember: "Receipt", Width: 125},
					{Title: "入库批次", DataMember: "Batch", Width: 125},
					{Title: "仓库", DataMember: "Warehouse", Width: 110},
					{Title: "库位", DataMember: "Location", Width: 220},
					{Title: "库存", DataMember: "Quantity", Width: 90},
					{Title: "可用", DataMember: "Available", Width: 80},
					{Title: "锁定", DataMember: "Locked", Width: 80},
					{Title: "冻结", DataMember: "Frozen", Width: 80},
				},
			},
			Composite{Layout: HBox{Spacing: 8}, Children: []Widget{
				Label{AssignTo: &info, Text: "尚未加载", TextColor: secondaryTextColor()},
				HSpacer{},
				PushButton{AssignTo: &refresh, Text: "刷新", MinSize: Size{Width: 88, Height: 30}},
				PushButton{AssignTo: &closeButton, Text: "关闭", MinSize: Size{Width: 88, Height: 30}, OnClicked: func() { dlg.Accept() }},
			}},
		},
	}.Create(owner)
	if err != nil {
		cancel()
		walk.MsgBox(owner, "库存位置窗口错误", err.Error(), walk.MsgBoxIconError)
		return
	}
	dlg.Disposing().Attach(func() { closed.Store(true); cancel() })
	load = func() {
		info.SetText("正在加载库存位置……")
		refresh.SetEnabled(false)
		guardedGo(func() {
			items, requestErr := client.InventoryByMaterial(ctx, material.ID)
			if closed.Load() || ctx.Err() != nil {
				return
			}
			dlg.Synchronize(func() {
				if closed.Load() {
					return
				}
				refresh.SetEnabled(true)
				if requestErr != nil {
					info.SetText("加载失败：" + requestErr.Error() + "。请检查网络后重试。")
					return
				}
				rows := make([]materialInventoryRow, 0, len(items))
				for _, item := range items {
					location := strings.Trim(strings.Join([]string{item.WarehouseZoneName, item.WarehouseRackName, item.WarehouseBinName}, " / "), " /")
					rows = append(rows, materialInventoryRow{
						Type: item.Type, Receipt: item.ReceiptCode, Batch: item.ReceiveCode, Warehouse: item.WarehouseName, Location: location,
						Quantity: fmt.Sprintf("%g %s", item.Quantity, item.Unit), Available: fmt.Sprintf("%g", item.AvailableQuantity),
						Locked: fmt.Sprintf("%g", item.LockedQuantity), Frozen: fmt.Sprintf("%g", item.FrozenQuantity),
					})
				}
				_ = table.SetModel(rows)
				if len(rows) == 0 {
					info.SetText("该物料当前没有库存位置。")
				} else {
					info.SetText(fmt.Sprintf("共 %d 条库存位置。", len(rows)))
				}
			})
		})
	}
	refresh.Clicked().Attach(load)
	load()
	dlg.Run()
}
