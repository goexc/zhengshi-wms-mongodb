package ui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync/atomic"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"

	"zhengshi-wms-windowsapp/internal/api"
)

type materialPriceRow struct {
	Price    string
	Customer string
	Since    string
	Source   string
	Valid    string
	Reason   string
}

func ShowMaterialPriceHistory(owner walk.Form, client *api.Client, material api.Material) {
	var dlg *walk.Dialog
	var table *walk.TableView
	var info *walk.Label
	var refresh, closeButton *walk.PushButton
	var closed atomic.Bool
	ctx, cancel := context.WithCancel(context.Background())

	err := Dialog{
		AssignTo:      &dlg,
		Title:         "历史价格 - " + displayMaterialValue(material.Model),
		DefaultButton: &closeButton,
		MinSize:       Size{Width: 760, Height: 460},
		Size:          Size{Width: 920, Height: 580},
		Layout:        VBox{Margins: Margins{Left: 16, Top: 16, Right: 16, Bottom: 14}, Spacing: 10},
		Children: []Widget{
			Label{Text: material.Name, Font: Font{Family: "Microsoft YaHei UI", PointSize: 15, Bold: true}},
			Label{
				Text:      "只读展示现有物料价格记录；价格来源和有效性均由服务端返回。",
				TextColor: walk.RGB(80, 80, 80),
			},
			TableView{
				AssignTo: &table, Model: []materialPriceRow{}, AlternatingRowBG: true,
				ColumnsOrderable: true, StretchFactor: 1,
				Accessibility: Accessibility{Name: "物料历史价格列表"},
				Columns: []TableViewColumn{
					{Title: "价格", DataMember: "Price", Width: 95},
					{Title: "客户", DataMember: "Customer", Width: 170},
					{Title: "应用时间", DataMember: "Since", Width: 135},
					{Title: "来源", DataMember: "Source", Width: 120},
					{Title: "有效性", DataMember: "Valid", Width: 80},
					{Title: "说明", DataMember: "Reason", Width: 260},
				},
			},
			Composite{Layout: HBox{Spacing: 8}, Children: []Widget{
				Label{AssignTo: &info, Text: "尚未加载", TextColor: walk.RGB(80, 80, 80)},
				HSpacer{},
				PushButton{AssignTo: &refresh, Text: "刷新", MinSize: Size{Width: 88, Height: 30}},
				PushButton{
					AssignTo: &closeButton, Text: "关闭", MinSize: Size{Width: 88, Height: 30},
					OnClicked: func() { dlg.Accept() },
				},
			}},
		},
	}.Create(owner)
	if err != nil {
		cancel()
		walk.MsgBox(owner, "历史价格窗口错误", err.Error(), walk.MsgBoxIconError)
		return
	}
	dlg.Disposing().Attach(func() {
		closed.Store(true)
		cancel()
	})
	load := func() {
		info.SetText("正在加载历史价格……")
		refresh.SetEnabled(false)
		go func() {
			prices, requestErr := client.MaterialPrices(ctx, material.ID, "")
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
				rows := materialPriceRows(prices)
				if modelErr := table.SetModel(rows); modelErr != nil {
					info.SetText("价格列表展示失败：" + modelErr.Error())
					return
				}
				if len(rows) == 0 {
					info.SetText("该物料暂无历史价格。")
				} else {
					info.SetText(fmt.Sprintf("共 %d 条历史价格。", len(rows)))
				}
			})
		}()
	}
	refresh.Clicked().Attach(load)
	load()
	dlg.Run()
}

func materialPriceRows(prices []api.MaterialPrice) []materialPriceRow {
	items := append([]api.MaterialPrice(nil), prices...)
	sort.SliceStable(items, func(i, j int) bool { return items[i].Since > items[j].Since })
	rows := make([]materialPriceRow, 0, len(items))
	for _, price := range items {
		valid := "有效"
		reason := strings.TrimSpace(price.SourceInvalidReason)
		if !price.SourceValid {
			valid = "无效"
			if reason == "" {
				reason = "来源已失效"
			}
		}
		rows = append(rows, materialPriceRow{
			Price: fmt.Sprintf("%.3f", price.Price), Customer: displayMaterialValue(price.CustomerName),
			Since: formatUnixMinute(price.Since), Source: materialPriceSourceLabel(price.SourceType),
			Valid: valid, Reason: reason,
		})
	}
	return rows
}

func materialPriceSourceLabel(source string) string {
	switch strings.TrimSpace(source) {
	case "manual":
		return "人工维护"
	case "outbound":
		return "出库核价"
	case "material_quote":
		return "物料报价"
	case "":
		return "—"
	default:
		return source
	}
}

type customerTransactionRow struct {
	Type        string
	Direction   string
	Status      string
	Source      string
	Time        string
	Amount      string
	Remark      string
	Attachments string
	Detail      api.CustomerTransaction
}

func ShowCustomerTransactions(owner walk.Form, client *api.Client, imageBaseURL, customerID, customerName string) {
	var dlg *walk.Dialog
	var table *walk.TableView
	var info *walk.Label
	var refreshButton, prevButton, nextButton, attachment *walk.PushButton
	var pageSize *walk.ComboBox
	var closeAction *walk.PushButton
	var rows []customerTransactionRow
	var page = 1
	var total int64
	var generation int
	var requestCancel context.CancelFunc
	var closed atomic.Bool
	pageLabels := []string{"20 条/页", "50 条/页", "100 条/页"}
	pageValues := []int{20, 50, 100}
	var load func()
	var updateAttachment func()

	err := Dialog{
		AssignTo:      &dlg,
		Title:         "客户交易流水 - " + customerName,
		DefaultButton: &closeAction,
		MinSize:       Size{Width: 820, Height: 500},
		Size:          Size{Width: 1040, Height: 660},
		Layout:        VBox{Margins: Margins{Left: 16, Top: 16, Right: 16, Bottom: 14}, Spacing: 10},
		Children: []Widget{
			Label{Text: customerName, Font: Font{Family: "Microsoft YaHei UI", PointSize: 15, Bold: true}},
			Label{Text: "只读查询客户交易流水，不在 Windows 客户端新增或修改财务记录。", TextColor: walk.RGB(80, 80, 80)},
			TableView{
				AssignTo: &table, Model: []customerTransactionRow{}, AlternatingRowBG: true,
				ColumnsOrderable: true, StretchFactor: 1,
				Accessibility: Accessibility{Name: "客户交易流水列表"},
				OnCurrentIndexChanged: func() {
					if updateAttachment != nil {
						updateAttachment()
					}
				},
				Columns: []TableViewColumn{
					{Title: "交易类型", DataMember: "Type", Width: 115},
					{Title: "方向", DataMember: "Direction", Width: 90},
					{Title: "状态", DataMember: "Status", Width: 85},
					{Title: "来源单据", DataMember: "Source", Width: 150},
					{Title: "交易时间", DataMember: "Time", Width: 135},
					{Title: "金额", DataMember: "Amount", Width: 100},
					{Title: "附件", DataMember: "Attachments", Width: 60},
					{Title: "备注", DataMember: "Remark", Width: 240},
				},
			},
			Composite{Layout: HBox{Spacing: 8}, Children: []Widget{
				Label{AssignTo: &info, Text: "尚未加载", TextColor: walk.RGB(80, 80, 80)},
				HSpacer{},
				PushButton{AssignTo: &attachment, Text: "查看附件", Enabled: false, MinSize: Size{Width: 92, Height: 30}},
				PushButton{AssignTo: &refreshButton, Text: "刷新", MinSize: Size{Width: 80, Height: 30}},
				Label{Text: "每页"},
				ComboBox{AssignTo: &pageSize, Model: pageLabels, CurrentIndex: 0, MinSize: Size{Width: 92}},
				PushButton{AssignTo: &prevButton, Text: "上一页"},
				PushButton{AssignTo: &nextButton, Text: "下一页"},
				PushButton{AssignTo: &closeAction, Text: "关闭", MinSize: Size{Width: 80, Height: 30}, OnClicked: func() { dlg.Accept() }},
			}},
		},
	}.Create(owner)
	if err != nil {
		walk.MsgBox(owner, "交易流水窗口错误", err.Error(), walk.MsgBoxIconError)
		return
	}
	dlg.Disposing().Attach(func() {
		closed.Store(true)
		if requestCancel != nil {
			requestCancel()
		}
	})
	updateAttachment = func() {
		index := table.CurrentIndex()
		if index < 0 || index >= len(rows) {
			attachment.SetEnabled(false)
			attachment.SetText("查看附件")
			return
		}
		attachments := customerTransactionAttachments(rows[index].Detail.Annex)
		attachment.SetEnabled(len(attachments) > 0)
		attachment.SetText(fmt.Sprintf("查看附件 (%d)", len(attachments)))
	}
	attachment.Clicked().Attach(func() {
		index := table.CurrentIndex()
		if index < 0 || index >= len(rows) {
			return
		}
		ShowOrderAttachments(dlg, client, imageBaseURL, "客户流水 "+rows[index].Source, customerTransactionAttachments(rows[index].Detail.Annex))
	})
	load = func() {
		if requestCancel != nil {
			requestCancel()
		}
		ctx, cancel := context.WithCancel(context.Background())
		requestCancel = cancel
		generation++
		currentGeneration := generation
		currentPage := page
		sizeIndex := pageSize.CurrentIndex()
		if sizeIndex < 0 || sizeIndex >= len(pageValues) {
			sizeIndex = 0
		}
		size := pageValues[sizeIndex]
		info.SetText("正在加载客户交易流水……")
		refreshButton.SetEnabled(false)
		prevButton.SetEnabled(false)
		nextButton.SetEnabled(false)
		go func() {
			result, requestErr := client.CustomerTransactions(ctx, customerID, currentPage, size)
			if ctx.Err() != nil || closed.Load() {
				return
			}
			dlg.Synchronize(func() {
				if closed.Load() || currentGeneration != generation {
					return
				}
				refreshButton.SetEnabled(true)
				if requestErr != nil {
					info.SetText("加载失败：" + requestErr.Error() + "。请检查网络后重试。")
					return
				}
				rows = customerTransactionRows(result.List)
				total = result.Total
				if modelErr := table.SetModel(rows); modelErr != nil {
					info.SetText("流水列表展示失败：" + modelErr.Error())
					return
				}
				info.SetText(fmt.Sprintf("第 %d 页 | 本页 %d 条 | 共 %d 条", currentPage, len(rows), total))
				prevButton.SetEnabled(currentPage > 1)
				nextButton.SetEnabled(int64(currentPage*size) < total)
				updateAttachment()
			})
		}()
	}
	refreshButton.Clicked().Attach(load)
	pageSize.CurrentIndexChanged().Attach(func() {
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
		page++
		load()
	})
	load()
	dlg.Run()
}

func customerTransactionRows(items []api.CustomerTransaction) []customerTransactionRow {
	rows := make([]customerTransactionRow, 0, len(items))
	for _, item := range items {
		source := strings.TrimSpace(item.SourceCode)
		if source == "" {
			source = displayMaterialValue(item.SourceType)
		}
		rows = append(rows, customerTransactionRow{
			Type: customerTransactionTypeLabel(item), Direction: customerTransactionDirectionLabel(item.Direction),
			Status: displayMaterialValue(item.Status), Source: source,
			Time: formatUnixMinute(item.Time), Amount: fmt.Sprintf("%.4f", item.Amount),
			Remark: item.Remark, Attachments: fmt.Sprint(len(customerTransactionAttachments(item.Annex))), Detail: item,
		})
	}
	return rows
}

func customerTransactionTypeLabel(item api.CustomerTransaction) string {
	if value := strings.TrimSpace(item.Type); value != "" {
		return value
	}
	switch strings.TrimSpace(item.TransactionType) {
	case "outbound_ar":
		return "出库应收"
	case "opening_ar":
		return "期初应收"
	case "payment":
		return "回款"
	case "return_credit":
		return "退货贷项"
	case "ar_adjustment":
		return "应收调整"
	case "manual_adjustment":
		return "人工调整"
	case "":
		return "—"
	default:
		return item.TransactionType
	}
}

func customerTransactionDirectionLabel(direction string) string {
	switch strings.TrimSpace(direction) {
	case "receivable_increase":
		return "应收增加"
	case "receivable_decrease":
		return "应收减少"
	case "":
		return "—"
	default:
		return direction
	}
}

func customerTransactionAttachments(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			result = append(result, part)
		}
	}
	return result
}
