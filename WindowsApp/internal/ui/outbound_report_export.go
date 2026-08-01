package ui

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/lxn/walk"
	"github.com/xuri/excelize/v2"

	"zhengshi-wms-windowsapp/internal/api"
)

type outboundReportGroup struct {
	key  string
	rows []api.OutboundSummaryRecord
}

var outboundReportExportColumns = []struct {
	header string
	width  float64
}{
	{"序号", 8},
	{"产品型号", 24},
	{"名称", 18},
	{"规格", 24},
	{"签收日期", 14},
	{"出库单编号", 20},
	{"单价", 12},
	{"数量", 12},
	{"总数量", 12},
	{"金额", 14},
	{"总金额", 14},
}

func exportOutboundReportXLSX(
	owner walk.Form,
	mode string,
	records []api.OutboundSummaryRecord,
	customerName string,
	startDate, endDate int64,
) (bool, error) {
	label := "按单据"
	if mode == "material" {
		label = "按物料"
	} else if mode != "order" {
		return false, fmt.Errorf("不支持的导出方式：%s", mode)
	}
	defaultName := fmt.Sprintf(
		"出库报表-%s-%s-%s.xlsx",
		label, formatReportDate(startDate), formatReportDate(endDate),
	)
	dialog := new(walk.FileDialog)
	dialog.Title = "导出出库报表"
	dialog.Filter = "Excel 工作簿 (*.xlsx)|*.xlsx"
	dialog.FilePath = defaultName
	accepted, err := dialog.ShowSave(owner)
	if err != nil {
		return false, err
	}
	if !accepted {
		return false, nil
	}
	target := strings.TrimSpace(dialog.FilePath)
	if filepath.Ext(target) == "" {
		target += ".xlsx"
	}
	if err := writeOutboundReportXLSX(target, mode, records, customerName, startDate, endDate); err != nil {
		return false, err
	}
	return true, nil
}

func writeOutboundReportXLSX(
	target string,
	mode string,
	records []api.OutboundSummaryRecord,
	customerName string,
	startDate, endDate int64,
) error {
	if mode != "order" && mode != "material" {
		return fmt.Errorf("不支持的导出方式：%s", mode)
	}
	workbook := excelize.NewFile()
	defer workbook.Close()
	sheet := "按单据导出"
	if mode == "material" {
		sheet = "按物料导出"
	}
	defaultSheet := workbook.GetSheetName(0)
	if err := workbook.SetSheetName(defaultSheet, sheet); err != nil {
		return err
	}
	if err := workbook.SetDocProps(&excelize.DocProperties{Creator: "zhengshi-wms"}); err != nil {
		return err
	}
	for index, column := range outboundReportExportColumns {
		name, _ := excelize.ColumnNumberToName(index + 1)
		if err := workbook.SetColWidth(sheet, name, name, column.width); err != nil {
			return err
		}
	}
	if err := workbook.MergeCell(sheet, "A1", "K1"); err != nil {
		return err
	}
	if err := workbook.MergeCell(sheet, "A2", "K2"); err != nil {
		return err
	}
	_ = workbook.SetCellValue(sheet, "A1", fmt.Sprintf(
		"诸城市双喜机械有限公司（%s 至 %s）",
		formatReportDate(startDate), formatReportDate(endDate),
	))
	_ = workbook.SetCellValue(sheet, "A2", "客户："+displayMaterialValue(customerName))
	headers := make([]interface{}, len(outboundReportExportColumns))
	for index, column := range outboundReportExportColumns {
		headers[index] = column.header
	}
	if err := workbook.SetSheetRow(sheet, "A3", &headers); err != nil {
		return err
	}
	_ = workbook.SetRowHeight(sheet, 1, 28)
	_ = workbook.SetRowHeight(sheet, 2, 24)
	if err := workbook.SetPanes(sheet, &excelize.Panes{
		Freeze: true, YSplit: 3, TopLeftCell: "A4", ActivePane: "bottomLeft",
		Selection: []excelize.Selection{{SQRef: "A4", ActiveCell: "A4", Pane: "bottomLeft"}},
	}); err != nil {
		return err
	}

	styles, err := createOutboundReportStyles(workbook)
	if err != nil {
		return err
	}
	if err := workbook.SetCellStyle(sheet, "A1", "K1", styles.title); err != nil {
		return err
	}
	if err := workbook.SetCellStyle(sheet, "A2", "K2", styles.subtitle); err != nil {
		return err
	}
	if err := workbook.SetCellStyle(sheet, "A3", "K3", styles.header); err != nil {
		return err
	}

	groups := groupOutboundReportRecords(records, mode)
	rowNumber := 4
	for groupIndex, group := range groups {
		startRow := rowNumber
		groupQuantity := reportTotalQuantity(group.rows)
		groupAmount := reportTotalAmount(group.rows)
		for _, record := range group.rows {
			values := []interface{}{
				groupIndex + 1,
				record.Model,
				record.Name,
				record.Specification,
				formatReportDate(record.ReceiptDate),
				reportOrderCode(record),
				record.Price,
				record.Quantity,
				groupQuantity,
				reportRecordAmount(record),
				groupAmount,
			}
			cell, _ := excelize.CoordinatesToCellName(1, rowNumber)
			if err := workbook.SetSheetRow(sheet, cell, &values); err != nil {
				return err
			}
			rowNumber++
		}
		endRow := rowNumber - 1
		mergeColumns := []int{1, 5, 6, 9, 11}
		if mode == "material" {
			mergeColumns = []int{1, 2, 3, 4, 9, 11}
		}
		if endRow > startRow {
			for _, column := range mergeColumns {
				startCell, _ := excelize.CoordinatesToCellName(column, startRow)
				endCell, _ := excelize.CoordinatesToCellName(column, endRow)
				if err := workbook.MergeCell(sheet, startCell, endCell); err != nil {
					return err
				}
			}
		}
	}
	if rowNumber > 4 {
		endCell, _ := excelize.CoordinatesToCellName(11, rowNumber-1)
		if err := workbook.SetCellStyle(sheet, "A4", endCell, styles.data); err != nil {
			return err
		}
		for _, column := range []string{"G", "J", "K"} {
			if err := workbook.SetCellStyle(sheet, column+"4", fmt.Sprintf("%s%d", column, rowNumber-1), styles.decimal); err != nil {
				return err
			}
		}
		for _, column := range []string{"H", "I"} {
			if err := workbook.SetCellStyle(sheet, column+"4", fmt.Sprintf("%s%d", column, rowNumber-1), styles.integer); err != nil {
				return err
			}
		}
	}
	if err := workbook.SaveAs(target); err != nil {
		return err
	}
	return nil
}

type outboundReportStyles struct {
	title    int
	subtitle int
	header   int
	data     int
	decimal  int
	integer  int
}

func createOutboundReportStyles(workbook *excelize.File) (outboundReportStyles, error) {
	border := []excelize.Border{
		{Type: "left", Color: "7A8793", Style: 1},
		{Type: "right", Color: "7A8793", Style: 1},
		{Type: "top", Color: "7A8793", Style: 1},
		{Type: "bottom", Color: "7A8793", Style: 1},
	}
	title, err := workbook.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 16, Family: "Microsoft YaHei"},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	if err != nil {
		return outboundReportStyles{}, err
	}
	subtitle, err := workbook.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 12, Family: "Microsoft YaHei"},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
	})
	if err != nil {
		return outboundReportStyles{}, err
	}
	header, err := workbook.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Family: "Microsoft YaHei"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"D9EAF7"}},
		Border:    border,
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
	})
	if err != nil {
		return outboundReportStyles{}, err
	}
	data, err := workbook.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Family: "Microsoft YaHei"},
		Border:    border,
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
	})
	if err != nil {
		return outboundReportStyles{}, err
	}
	decimalFormat := "#,##0.000"
	decimal, err := workbook.NewStyle(&excelize.Style{
		Font:         &excelize.Font{Family: "Microsoft YaHei"},
		Border:       border,
		Alignment:    &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		CustomNumFmt: &decimalFormat,
	})
	if err != nil {
		return outboundReportStyles{}, err
	}
	integerFormat := "#,##0"
	integer, err := workbook.NewStyle(&excelize.Style{
		Font:         &excelize.Font{Family: "Microsoft YaHei"},
		Border:       border,
		Alignment:    &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		CustomNumFmt: &integerFormat,
	})
	if err != nil {
		return outboundReportStyles{}, err
	}
	return outboundReportStyles{
		title: title, subtitle: subtitle, header: header,
		data: data, decimal: decimal, integer: integer,
	}, nil
}

func groupOutboundReportRecords(records []api.OutboundSummaryRecord, mode string) []outboundReportGroup {
	rows := append([]api.OutboundSummaryRecord(nil), records...)
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].ReceiptDate != rows[j].ReceiptDate {
			left := rows[i].ReceiptDate
			right := rows[j].ReceiptDate
			if left == 0 {
				left = int64(^uint64(0) >> 1)
			}
			if right == 0 {
				right = int64(^uint64(0) >> 1)
			}
			return left < right
		}
		if reportOrderCode(rows[i]) != reportOrderCode(rows[j]) {
			return reportOrderCode(rows[i]) < reportOrderCode(rows[j])
		}
		return rows[i].Index < rows[j].Index
	})
	groupsByKey := make(map[string][]api.OutboundSummaryRecord)
	order := make([]string, 0)
	for _, row := range rows {
		key := reportOrderCode(row)
		if mode == "material" {
			key = row.MaterialID
		}
		if _, exists := groupsByKey[key]; !exists {
			order = append(order, key)
		}
		groupsByKey[key] = append(groupsByKey[key], row)
	}
	if mode == "material" {
		sort.SliceStable(order, func(i, j int) bool {
			left := groupsByKey[order[i]][0]
			right := groupsByKey[order[j]][0]
			if left.Model != right.Model {
				return left.Model < right.Model
			}
			if left.Name != right.Name {
				return left.Name < right.Name
			}
			return left.Specification < right.Specification
		})
	}
	result := make([]outboundReportGroup, 0, len(order))
	for _, key := range order {
		result = append(result, outboundReportGroup{key: key, rows: groupsByKey[key]})
	}
	return result
}
