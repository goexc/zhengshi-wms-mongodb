package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/lxn/walk"

	"zhengshi-wms-windowsapp/internal/api"
)

func (ui *mainUI) confirmLargeQueryExport(total int64) bool {
	if total <= 1000 {
		return true
	}
	message := fmt.Sprintf("当前查询结果约 %d 条，导出需要分批读取线上接口，可能耗时较长。\r\n\r\n单次最多导出 %d 条，是否继续？", total, queryExportLimit)
	return walk.MsgBox(ui.window, "确认大批量导出", message, walk.MsgBoxYesNo|walk.MsgBoxIconQuestion) == walk.DlgCmdYes
}

func (ui *mainUI) exportInventoryQuery() {
	if ui.inventoryExport == nil || !ui.confirmLargeQueryExport(ui.inventoryTotal) {
		return
	}
	history := ui.inventoryMode.CurrentIndex() == 1
	mode := "当前库存"
	if history {
		mode = "库存批次历史"
	}
	target, accepted, err := chooseQueryExportTarget(ui.window, "导出"+mode, fmt.Sprintf("%s-%s.xlsx", mode, time.Now().Format("20060102-150405")))
	if err != nil {
		walk.MsgBox(ui.window, "无法选择导出位置", err.Error(), walk.MsgBoxIconError)
		return
	}
	if !accepted {
		return
	}
	filters := api.InventoryFilters{
		MaterialName: ui.inventoryName.Text(), MaterialModel: ui.inventoryModel.Text(),
		WarehouseID:     selectedNodeID(ui.inventoryWarehouse, ui.inventoryWarehouseNodes),
		WarehouseZoneID: selectedNodeID(ui.inventoryZone, ui.inventoryZoneNodes),
		WarehouseRackID: selectedNodeID(ui.inventoryRack, ui.inventoryRackNodes),
		WarehouseBinID:  selectedNodeID(ui.inventoryBin, ui.inventoryBinNodes),
	}
	if ui.inventoryType.CurrentIndex() > 0 {
		filters.Type = ui.inventoryType.Text()
	}
	description := strings.Join([]string{
		"视图=" + mode, "入库类型=" + ui.inventoryType.Text(), "物料名称=" + displayMaterialValue(ui.inventoryName.Text()),
		"物料型号=" + displayMaterialValue(ui.inventoryModel.Text()), "仓库=" + ui.inventoryWarehouse.Text(),
		"库区=" + ui.inventoryZone.Text(), "货架=" + ui.inventoryRack.Text(), "货位=" + ui.inventoryBin.Text(),
	}, "；")
	if ui.inventoryExportCancel != nil {
		ui.inventoryExportCancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	ui.inventoryExportCancel = cancel
	ui.inventoryExportGeneration++
	generation := ui.inventoryExportGeneration
	pageGeneration := ui.inventoryGeneration
	ui.inventoryExportBusy = true
	ui.inventoryExport.SetEnabled(false)
	ui.inventoryExportStop.SetEnabled(true)
	ui.inventoryInfo.SetText("正在分批读取线上库存并生成 Excel……")
	finishTask := ui.startBackgroundTask("库存查询导出", func() {
		if generation == ui.inventoryExportGeneration {
			ui.cancelInventoryExport()
		}
	})
	guardedGo(func() {
		defer finishTask()
		items, _, requestErr := fetchAllInventoryWithProgress(ctx, ui.session.Client, history, filters, func(done int, total int64) {
			ui.reportInventoryExportProgress(generation, pageGeneration, done, total, mode)
		})
		if requestErr == nil {
			requestErr = writeQueryExportXLSXAtomic(ctx, target, mode+"查询结果", description, "现有库存查询接口", inventoryExportColumns(), inventoryExportRows(items), time.Now())
		}
		if ctx.Err() != nil {
			return
		}
		ui.window.Synchronize(func() {
			if ui.inventoryTab == nil || ui.inventoryExport == nil || generation != ui.inventoryExportGeneration || pageGeneration != ui.inventoryGeneration {
				return
			}
			ui.inventoryExportBusy = false
			ui.inventoryExportCancel = nil
			ui.inventoryExport.SetEnabled(true)
			ui.inventoryExportStop.SetEnabled(false)
			if requestErr != nil {
				ui.inventoryInfo.SetText("导出失败：" + requestErr.Error())
				walk.MsgBox(ui.window, "导出失败", requestErr.Error(), walk.MsgBoxIconError)
				return
			}
			ui.inventoryInfo.SetText(fmt.Sprintf("已导出 %d 条%s记录。", len(items), mode))
		})
	})
}

func (ui *mainUI) exportInboundQuery() {
	if ui.inboundExport == nil || !ui.confirmLargeQueryExport(ui.inboundTotal) {
		return
	}
	target, accepted, err := chooseQueryExportTarget(ui.window, "导出入库查询结果", fmt.Sprintf("入库查询-%s.xlsx", time.Now().Format("20060102-150405")))
	if err != nil {
		walk.MsgBox(ui.window, "无法选择导出位置", err.Error(), walk.MsgBoxIconError)
		return
	}
	if !accepted {
		return
	}
	filters := api.InboundFilters{Code: ui.inboundSearch.Text()}
	if ui.inboundStatus.CurrentIndex() > 0 {
		filters.Status = ui.inboundStatus.Text()
	}
	if ui.inboundType.CurrentIndex() > 0 {
		filters.Type = ui.inboundType.Text()
	}
	filters.SupplierID = selectedOptionID(ui.inboundSupplier, ui.inboundSupplierOptions)
	filters.CustomerID = selectedOptionID(ui.inboundCustomer, ui.inboundCustomerOptions)
	description := strings.Join([]string{
		"单号=" + displayMaterialValue(ui.inboundSearch.Text()), "状态=" + ui.inboundStatus.Text(), "类型=" + ui.inboundType.Text(),
		"供应商=" + ui.inboundSupplier.Text(), "客户=" + ui.inboundCustomer.Text(),
	}, "；")
	if ui.inboundExportCancel != nil {
		ui.inboundExportCancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	ui.inboundExportCancel = cancel
	ui.inboundExportGeneration++
	generation := ui.inboundExportGeneration
	pageGeneration := ui.inboundGeneration
	ui.inboundExportBusy = true
	ui.inboundExport.SetEnabled(false)
	ui.inboundExportStop.SetEnabled(true)
	ui.inboundInfo.SetText("正在分批读取线上入库单并生成 Excel……")
	finishTask := ui.startBackgroundTask("入库查询导出", func() {
		if generation == ui.inboundExportGeneration {
			ui.cancelInboundExport()
		}
	})
	guardedGo(func() {
		defer finishTask()
		items, _, requestErr := fetchAllInboundWithProgress(ctx, ui.session.Client, filters, func(done int, total int64) {
			ui.reportInboundExportProgress(generation, pageGeneration, done, total)
		})
		if requestErr == nil {
			requestErr = writeQueryExportXLSXAtomic(ctx, target, "入库查询结果", description, "现有入库单查询接口", inboundExportColumns(), inboundExportRows(items), time.Now())
		}
		if ctx.Err() != nil {
			return
		}
		ui.window.Synchronize(func() {
			if ui.inboundTab == nil || ui.inboundExport == nil || generation != ui.inboundExportGeneration || pageGeneration != ui.inboundGeneration {
				return
			}
			ui.inboundExportBusy = false
			ui.inboundExportCancel = nil
			ui.inboundExport.SetEnabled(true)
			ui.inboundExportStop.SetEnabled(false)
			if requestErr != nil {
				ui.inboundInfo.SetText("导出失败：" + requestErr.Error())
				walk.MsgBox(ui.window, "导出失败", requestErr.Error(), walk.MsgBoxIconError)
				return
			}
			ui.inboundInfo.SetText(fmt.Sprintf("已导出 %d 张入库单。", len(items)))
		})
	})
}

func (ui *mainUI) exportOutboundQuery() {
	state := ui.outbound
	if state == nil || state.export == nil || !ui.confirmLargeQueryExport(state.total) {
		return
	}
	startTime, err := parseOutboundFilterDate(state.startDate.Text(), false)
	if err != nil {
		state.info.SetText("起始日期格式错误：" + err.Error())
		return
	}
	endTime, err := parseOutboundFilterDate(state.endDate.Text(), true)
	if err != nil {
		state.info.SetText("截止日期格式错误：" + err.Error())
		return
	}
	if startTime > 0 && endTime > 0 && startTime > endTime {
		state.info.SetText("签收起始日期不能晚于截止日期。")
		return
	}
	target, accepted, err := chooseQueryExportTarget(ui.window, "导出出库查询结果", fmt.Sprintf("出库查询-%s.xlsx", time.Now().Format("20060102-150405")))
	if err != nil {
		walk.MsgBox(ui.window, "无法选择导出位置", err.Error(), walk.MsgBoxIconError)
		return
	}
	if !accepted {
		return
	}
	stage := outboundStages[0]
	if index := state.stage.CurrentIndex(); index >= 0 && index < len(outboundStages) {
		stage = outboundStages[index]
	}
	filters := api.OutboundFilters{
		Code: state.search.Text(), Status: stage.Status, IsPack: stage.IsPack, IsWeigh: stage.IsWeigh,
		SupplierID: selectedOptionID(state.supplier, state.supplierOptions), CustomerID: selectedOptionID(state.customer, state.customerOptions),
		StartTime: startTime, EndTime: endTime,
	}
	if state.orderType.CurrentIndex() > 0 {
		filters.Type = state.orderType.Text()
	}
	description := strings.Join([]string{
		"单号=" + displayMaterialValue(state.search.Text()), "状态=" + state.stage.Text(), "类型=" + state.orderType.Text(),
		"供应商=" + state.supplier.Text(), "客户=" + state.customer.Text(),
		"签收起始=" + displayMaterialValue(state.startDate.Text()), "签收截止=" + displayMaterialValue(state.endDate.Text()),
	}, "；")
	if state.exportCancel != nil {
		state.exportCancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	state.exportCancel = cancel
	state.exportGeneration++
	generation := state.exportGeneration
	pageGeneration := state.generation
	state.exportBusy = true
	ui.setOutboundActionButtons(nil)
	state.exportStop.SetEnabled(true)
	state.info.SetText("正在分批读取线上出库单并生成 Excel……")
	finishTask := ui.startBackgroundTask("出库查询导出", func() {
		if state == ui.outbound && generation == state.exportGeneration {
			ui.cancelOutboundExport()
		}
	})
	guardedGo(func() {
		defer finishTask()
		items, _, requestErr := fetchAllOutboundWithProgress(ctx, ui.session.Client, filters, func(done int, total int64) {
			ui.reportOutboundExportProgress(state, generation, pageGeneration, done, total)
		})
		if requestErr == nil {
			requestErr = writeQueryExportXLSXAtomic(ctx, target, "出库查询结果", description, "现有出库分页查询接口", outboundExportColumns(), outboundExportRows(items), time.Now())
		}
		if ctx.Err() != nil {
			return
		}
		ui.window.Synchronize(func() {
			if state != ui.outbound || state.export == nil || generation != state.exportGeneration || pageGeneration != state.generation {
				return
			}
			state.exportBusy = false
			state.exportCancel = nil
			state.exportStop.SetEnabled(false)
			order, selected := ui.selectedOutboundWithoutMessage()
			if selected {
				ui.setOutboundActionButtons(&order)
			} else {
				ui.setOutboundActionButtons(nil)
			}
			if requestErr != nil {
				state.info.SetText("导出失败：" + requestErr.Error())
				walk.MsgBox(ui.window, "导出失败", requestErr.Error(), walk.MsgBoxIconError)
				return
			}
			state.info.SetText(fmt.Sprintf("已导出 %d 张出库单。", len(items)))
		})
	})
}

func exportProgressText(done int, total int64, noun string) string {
	if total <= 0 {
		return fmt.Sprintf("正在读取线上%s：已读取 %d 条……", noun, done)
	}
	percent := int64(done) * 100 / total
	if percent > 100 {
		percent = 100
	}
	return fmt.Sprintf("正在读取线上%s：%d/%d（%d%%）……", noun, done, total, percent)
}

func (ui *mainUI) reportInventoryExportProgress(generation, pageGeneration, done int, total int64, mode string) {
	ui.window.Synchronize(func() {
		if ui.inventoryTab == nil || ui.inventoryInfo == nil || generation != ui.inventoryExportGeneration || pageGeneration != ui.inventoryGeneration {
			return
		}
		ui.inventoryInfo.SetText(exportProgressText(done, total, mode))
	})
}

func (ui *mainUI) reportInboundExportProgress(generation, pageGeneration, done int, total int64) {
	ui.window.Synchronize(func() {
		if ui.inboundTab == nil || ui.inboundInfo == nil || generation != ui.inboundExportGeneration || pageGeneration != ui.inboundGeneration {
			return
		}
		ui.inboundInfo.SetText(exportProgressText(done, total, "入库单"))
	})
}

func (ui *mainUI) reportOutboundExportProgress(state *outboundUI, generation, pageGeneration, done int, total int64) {
	ui.window.Synchronize(func() {
		if state != ui.outbound || state.info == nil || generation != state.exportGeneration || pageGeneration != state.generation {
			return
		}
		state.info.SetText(exportProgressText(done, total, "出库单"))
	})
}

func (ui *mainUI) cancelInventoryExport() {
	if !ui.inventoryExportBusy {
		return
	}
	if ui.inventoryExportCancel != nil {
		ui.inventoryExportCancel()
	}
	ui.inventoryExportGeneration++
	ui.inventoryExportCancel = nil
	ui.inventoryExportBusy = false
	if ui.inventoryExport != nil {
		ui.inventoryExport.SetEnabled(true)
	}
	if ui.inventoryExportStop != nil {
		ui.inventoryExportStop.SetEnabled(false)
	}
	if ui.inventoryInfo != nil {
		ui.inventoryInfo.SetText("库存导出已取消；目标文件不会被半成品覆盖，临时文件正在清理。")
	}
}

func (ui *mainUI) cancelInboundExport() {
	if !ui.inboundExportBusy {
		return
	}
	if ui.inboundExportCancel != nil {
		ui.inboundExportCancel()
	}
	ui.inboundExportGeneration++
	ui.inboundExportCancel = nil
	ui.inboundExportBusy = false
	if ui.inboundExport != nil {
		ui.inboundExport.SetEnabled(true)
	}
	if ui.inboundExportStop != nil {
		ui.inboundExportStop.SetEnabled(false)
	}
	if ui.inboundInfo != nil {
		ui.inboundInfo.SetText("入库导出已取消；目标文件不会被半成品覆盖，临时文件正在清理。")
	}
}

func (ui *mainUI) cancelOutboundExport() {
	state := ui.outbound
	if state == nil || !state.exportBusy {
		return
	}
	if state.exportCancel != nil {
		state.exportCancel()
	}
	state.exportGeneration++
	state.exportCancel = nil
	state.exportBusy = false
	if state.exportStop != nil {
		state.exportStop.SetEnabled(false)
	}
	order, selected := ui.selectedOutboundWithoutMessage()
	if selected {
		ui.setOutboundActionButtons(&order)
	} else {
		ui.setOutboundActionButtons(nil)
	}
	if state.info != nil {
		state.info.SetText("出库导出已取消；目标文件不会被半成品覆盖，临时文件正在清理。")
	}
}
