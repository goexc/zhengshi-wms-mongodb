package ui

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/lxn/walk"
	"github.com/xuri/excelize/v2"

	"zhengshi-wms-windowsapp/internal/api"
)

const queryExportLimit = 5000

type queryExportColumn struct {
	Header string
	Width  float64
}

func chooseQueryExportTarget(owner walk.Form, title, defaultName string) (string, bool, error) {
	dialog := new(walk.FileDialog)
	dialog.Title = title
	dialog.Filter = "Excel 工作簿 (*.xlsx)|*.xlsx"
	dialog.FilePath = defaultName
	accepted, err := dialog.ShowSave(owner)
	if err != nil || !accepted {
		return "", accepted, err
	}
	target := strings.TrimSpace(dialog.FilePath)
	if filepath.Ext(target) == "" {
		target += ".xlsx"
	}
	return target, true, nil
}

func writeQueryExportXLSX(target, title, filters, source string, columns []queryExportColumn, rows [][]any, queriedAt time.Time) error {
	if len(columns) == 0 {
		return fmt.Errorf("导出列不能为空")
	}
	workbook := excelize.NewFile()
	defer workbook.Close()
	sheet := "查询结果"
	if err := workbook.SetSheetName(workbook.GetSheetName(0), sheet); err != nil {
		return err
	}
	if err := workbook.SetDocProps(&excelize.DocProperties{Creator: "zhengshi-wms WindowsApp"}); err != nil {
		return err
	}
	lastColumn, _ := excelize.ColumnNumberToName(len(columns))
	for row := 1; row <= 4; row++ {
		if err := workbook.MergeCell(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("%s%d", lastColumn, row)); err != nil {
			return err
		}
	}
	_ = workbook.SetCellValue(sheet, "A1", title)
	_ = workbook.SetCellValue(sheet, "A2", "查询时间："+queriedAt.Format("2006-01-02 15:04:05"))
	_ = workbook.SetCellValue(sheet, "A3", "筛选条件："+displayMaterialValue(filters))
	_ = workbook.SetCellValue(sheet, "A4", "数据说明："+source+"；文件由 Windows 客户端本地生成，未修改线上数据。")
	headers := make([]any, len(columns))
	for index, column := range columns {
		headers[index] = column.Header
		name, _ := excelize.ColumnNumberToName(index + 1)
		if err := workbook.SetColWidth(sheet, name, name, column.Width); err != nil {
			return err
		}
	}
	if err := workbook.SetSheetRow(sheet, "A6", &headers); err != nil {
		return err
	}
	for index, row := range rows {
		if len(row) != len(columns) {
			return fmt.Errorf("第 %d 行导出列数不一致", index+1)
		}
		cell, _ := excelize.CoordinatesToCellName(1, index+7)
		if err := workbook.SetSheetRow(sheet, cell, &row); err != nil {
			return err
		}
	}
	border := []excelize.Border{
		{Type: "left", Color: "A6B0BA", Style: 1}, {Type: "right", Color: "A6B0BA", Style: 1},
		{Type: "top", Color: "A6B0BA", Style: 1}, {Type: "bottom", Color: "A6B0BA", Style: 1},
	}
	titleStyle, err := workbook.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 15, Family: "Microsoft YaHei"},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	if err != nil {
		return err
	}
	metaStyle, err := workbook.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Family: "Microsoft YaHei", Color: "4B5563"},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center", WrapText: true},
	})
	if err != nil {
		return err
	}
	headerStyle, err := workbook.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Family: "Microsoft YaHei"}, Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"D9EAF7"}},
		Border: border, Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
	})
	if err != nil {
		return err
	}
	dataStyle, err := workbook.NewStyle(&excelize.Style{
		Font: &excelize.Font{Family: "Microsoft YaHei"}, Border: border,
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
	})
	if err != nil {
		return err
	}
	if err := workbook.SetCellStyle(sheet, "A1", lastColumn+"1", titleStyle); err != nil {
		return err
	}
	if err := workbook.SetCellStyle(sheet, "A2", lastColumn+"4", metaStyle); err != nil {
		return err
	}
	if err := workbook.SetCellStyle(sheet, "A6", lastColumn+"6", headerStyle); err != nil {
		return err
	}
	if len(rows) > 0 {
		if err := workbook.SetCellStyle(sheet, "A7", fmt.Sprintf("%s%d", lastColumn, len(rows)+6), dataStyle); err != nil {
			return err
		}
	}
	_ = workbook.SetRowHeight(sheet, 1, 28)
	_ = workbook.SetRowHeight(sheet, 4, 32)
	if err := workbook.SetPanes(sheet, &excelize.Panes{Freeze: true, YSplit: 6, TopLeftCell: "A7", ActivePane: "bottomLeft"}); err != nil {
		return err
	}
	return workbook.SaveAs(target)
}

func inventoryExportColumns() []queryExportColumn {
	return []queryExportColumn{
		{"类型", 14}, {"入库单", 20}, {"入库批次", 20}, {"仓库", 16}, {"库区", 16}, {"货架", 14}, {"货位", 14},
		{"物料", 24}, {"型号", 18}, {"单位", 10}, {"库存", 12}, {"可用", 12}, {"锁定", 12}, {"冻结", 12},
	}
}

func inventoryExportRows(items []api.Inventory) [][]any {
	rows := make([][]any, 0, len(items))
	for _, item := range items {
		rows = append(rows, []any{
			item.Type, item.ReceiptCode, item.ReceiveCode, item.WarehouseName, item.WarehouseZoneName, item.WarehouseRackName, item.WarehouseBinName,
			item.Name, item.Model, item.Unit, item.Quantity, item.AvailableQuantity, item.LockedQuantity, item.FrozenQuantity,
		})
	}
	return rows
}

func inboundExportColumns() []queryExportColumn {
	return []queryExportColumn{
		{"入库单号", 22}, {"类型", 14}, {"状态", 14}, {"供应商", 20}, {"客户", 20}, {"计划日期", 14},
		{"物料种类", 12}, {"计划数量", 12}, {"已收数量", 12}, {"金额", 14}, {"附件数", 10}, {"备注", 28},
	}
}

func inboundExportRows(items []api.InboundReceipt) [][]any {
	rows := make([][]any, 0, len(items))
	for _, item := range items {
		estimated, actual := 0.0, 0.0
		for _, material := range item.Materials {
			estimated += material.EstimatedQuantity
			actual += material.ActualQuantity
		}
		planDate := ""
		if item.ReceivingDate > 0 {
			planDate = time.Unix(item.ReceivingDate, 0).Format("2006-01-02")
		}
		rows = append(rows, []any{
			item.Code, item.Type, item.Status, item.SupplierName, item.CustomerName, planDate,
			len(item.Materials), estimated, actual, item.TotalAmount, len(item.Annex), item.Remark,
		})
	}
	return rows
}

func outboundExportColumns() []queryExportColumn {
	return []queryExportColumn{
		{"出库单号", 22}, {"类型", 14}, {"状态", 14}, {"执行进度", 20}, {"供应商", 20}, {"客户", 20}, {"承运商", 18},
		{"金额", 14}, {"出库时间", 20}, {"签收时间", 20}, {"附件数", 10}, {"备注", 28},
	}
}

func outboundExportRows(items []api.OutboundOrder) [][]any {
	rows := make([][]any, 0, len(items))
	for _, item := range items {
		rows = append(rows, []any{
			item.Code, item.Type, item.Status, outboundProgress(item), item.SupplierName, item.CustomerName, item.CarrierName,
			item.TotalAmount, formatUnixMinute(item.DepartureTime), formatUnixMinute(item.ReceiptTime), len(item.Annex), item.Remark,
		})
	}
	return rows
}

func fetchAllInventory(ctx context.Context, client *api.Client, history bool, filters api.InventoryFilters) ([]api.Inventory, int64, error) {
	return fetchAllInventoryWithProgress(ctx, client, history, filters, nil)
}

func fetchAllInventoryWithProgress(ctx context.Context, client *api.Client, history bool, filters api.InventoryFilters, progress func(int, int64)) ([]api.Inventory, int64, error) {
	items := make([]api.Inventory, 0)
	for page := 1; ; page++ {
		if err := ctx.Err(); err != nil {
			return nil, 0, err
		}
		var result api.InventoryPage
		var err error
		if history {
			result, err = client.InventoryHistory(ctx, page, 50, filters)
		} else {
			result, err = client.Inventory(ctx, page, 50, filters)
		}
		if err != nil {
			return nil, 0, err
		}
		if result.Total > queryExportLimit {
			return nil, result.Total, fmt.Errorf("当前筛选共 %d 条，超过单次导出上限 %d 条，请缩小筛选范围", result.Total, queryExportLimit)
		}
		items = append(items, result.List...)
		if progress != nil {
			progress(len(items), result.Total)
		}
		if len(result.List) == 0 || int64(len(items)) >= result.Total {
			return items, result.Total, nil
		}
	}
}

func fetchAllInbound(ctx context.Context, client *api.Client, filters api.InboundFilters) ([]api.InboundReceipt, int64, error) {
	return fetchAllInboundWithProgress(ctx, client, filters, nil)
}

func fetchAllInboundWithProgress(ctx context.Context, client *api.Client, filters api.InboundFilters, progress func(int, int64)) ([]api.InboundReceipt, int64, error) {
	items := make([]api.InboundReceipt, 0)
	for page := 1; ; page++ {
		if err := ctx.Err(); err != nil {
			return nil, 0, err
		}
		result, err := client.InboundReceipts(ctx, page, 50, filters)
		if err != nil {
			return nil, 0, err
		}
		if result.Total > queryExportLimit {
			return nil, result.Total, fmt.Errorf("当前筛选共 %d 条，超过单次导出上限 %d 条，请缩小筛选范围", result.Total, queryExportLimit)
		}
		items = append(items, result.List...)
		if progress != nil {
			progress(len(items), result.Total)
		}
		if len(result.List) == 0 || int64(len(items)) >= result.Total {
			return items, result.Total, nil
		}
	}
}

func fetchAllOutbound(ctx context.Context, client *api.Client, filters api.OutboundFilters) ([]api.OutboundOrder, int64, error) {
	return fetchAllOutboundWithProgress(ctx, client, filters, nil)
}

func fetchAllOutboundWithProgress(ctx context.Context, client *api.Client, filters api.OutboundFilters, progress func(int, int64)) ([]api.OutboundOrder, int64, error) {
	items := make([]api.OutboundOrder, 0)
	for page := 1; ; page++ {
		if err := ctx.Err(); err != nil {
			return nil, 0, err
		}
		result, err := client.OutboundOrders(ctx, page, 50, filters)
		if err != nil {
			return nil, 0, err
		}
		if result.Total > queryExportLimit {
			return nil, result.Total, fmt.Errorf("当前筛选共 %d 条，超过单次导出上限 %d 条，请缩小筛选范围", result.Total, queryExportLimit)
		}
		items = append(items, result.List...)
		if progress != nil {
			progress(len(items), result.Total)
		}
		if len(result.List) == 0 || int64(len(items)) >= result.Total {
			return items, result.Total, nil
		}
	}
}
