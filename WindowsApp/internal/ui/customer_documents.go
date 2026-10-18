package ui

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"github.com/lxn/win"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/webp"
	"zhengshi-wms-windowsapp/internal/api"
	"zhengshi-wms-windowsapp/internal/config"
)

type customerDocumentRow struct {
	Customer, Number, Date, Due, Amount, Status, Attachments string
	Record                                                   api.CustomerDocument
}
type customerDocumentsUI struct {
	kind                                                        string
	tab                                                         *walk.TabPage
	table                                                       *walk.TableView
	keyword                                                     *walk.LineEdit
	status, customer                                            *walk.ComboBox
	start, end                                                  *walk.DateEdit
	dates                                                       *walk.CheckBox
	info, summary                                               *walk.Label
	query, add, edit, detail, redeem, reverse, void, prev, next *walk.PushButton
	customers                                                   []api.Customer
	rows                                                        []customerDocumentRow
	page                                                        int
	generation                                                  int
	cancel                                                      context.CancelFunc
	busy                                                        bool
	closed                                                      atomic.Bool
}

func documentTitle(kind string) string {
	if kind == "acceptance" {
		return "客户承兑"
	}
	return "销售开票"
}
func documentIsOverdue(due, now int64) bool {
	return due > 0 && (due+8*3600)/86400 < (now+8*3600)/86400
}
func documentStatus(status string, due int64) string {
	switch status {
	case "pending":
		if due > 0 && documentIsOverdue(due, time.Now().Unix()) {
			return "待兑付 · 已到期"
		}
		return "待兑付"
	case "redeemed":
		return "已到账"
	case "active":
		return "有效"
	case "voided":
		return "已作废"
	}
	return status
}
func documentRecordAmount(kind string, r api.CustomerDocument) float64 {
	if kind == "invoice" {
		return r.TotalAmount
	}
	return r.Amount
}
func documentNumber(r api.CustomerDocument) string {
	if r.BillNumber != "" {
		return r.BillNumber
	}
	return r.InvoiceNumber
}
func documentDate(value int64) string {
	if value <= 0 {
		return "—"
	}
	return time.Unix(value, 0).In(documentLocation).Format("2006-01-02")
}
func (ui *mainUI) documentState(kind string) *customerDocumentsUI {
	if kind == "acceptance" {
		return ui.acceptanceDocuments
	}
	return ui.invoiceDocuments
}
func (ui *mainUI) customerDocumentsPage(kind string) TabPage {
	s := &customerDocumentsUI{kind: kind, page: 1}
	if kind == "acceptance" {
		ui.acceptanceDocuments = s
	} else {
		ui.invoiceDocuments = s
	}
	statuses := []string{"全部状态", "待兑付", "已到账", "已作废"}
	if kind == "invoice" {
		statuses = []string{"全部状态", "有效", "已作废"}
	}
	return TabPage{AssignTo: &s.tab, Title: closableTabTitle(documentTitle(kind)), Layout: VBox{Margins: Margins{Left: 14, Top: 14, Right: 14, Bottom: 12}, Spacing: 10}, Children: []Widget{
		Label{Text: documentTitle(kind), Font: Font{Family: "Microsoft YaHei UI", PointSize: 15, Bold: true}},
		Label{Text: map[string]string{"acceptance": "收票不改变应收；兑付到账后才计入回款。到期提示不代表已到账。", "invoice": "记录已开具发票及其附件；开票和作废均不改变客户应收。"}[kind], TextColor: secondaryTextColor()},
		Composite{Layout: HBox{Spacing: 8}, Children: []Widget{Label{Text: "客户"}, ComboBox{AssignTo: &s.customer, Model: []string{"全部客户"}, CurrentIndex: 0, MinSize: Size{Width: 200}}, Label{Text: "票据号码"}, LineEdit{AssignTo: &s.keyword, CueBanner: "输入号码或客户关键词", MinSize: Size{Width: 180}}, ComboBox{AssignTo: &s.status, Model: statuses, CurrentIndex: 0, MinSize: Size{Width: 125}}, PushButton{AssignTo: &s.query, Text: "查询", OnClicked: func() {
			s.page = 1
			ui.loadCustomerDocuments(s)
			if len(s.customers) == 0 {
				ui.loadDocumentCustomers(s)
			}
		}}}},
		Composite{Layout: HBox{Spacing: 8}, Children: []Widget{CheckBox{AssignTo: &s.dates, Text: map[string]string{"acceptance": "按收票日期", "invoice": "按开票日期"}[kind]}, DateEdit{AssignTo: &s.start, Date: time.Now().In(documentLocation).AddDate(0, -1, 0), Format: "yyyy-MM-dd"}, Label{Text: "至"}, DateEdit{AssignTo: &s.end, Date: time.Now().In(documentLocation), Format: "yyyy-MM-dd"}, HSpacer{}, PushButton{Text: "重置筛选", OnClicked: func() {
			s.keyword.SetText("")
			s.customer.SetCurrentIndex(0)
			s.status.SetCurrentIndex(0)
			s.dates.SetChecked(false)
			s.page = 1
			ui.loadCustomerDocuments(s)
		}}}},
		Label{AssignTo: &s.summary, Text: "汇总待加载", TextColor: secondaryTextColor()},
		Composite{Layout: HBox{Spacing: 8}, Children: []Widget{
			PushButton{AssignTo: &s.add, Text: "新增登记", OnClicked: func() { ui.editCustomerDocument(s, api.CustomerDocument{}) }}, PushButton{AssignTo: &s.edit, Text: "修改登记", OnClicked: func() { ui.loadSelectedCustomerDocument(s, "edit") }}, PushButton{AssignTo: &s.detail, Text: "查看详情及附件", OnClicked: func() { ui.loadSelectedCustomerDocument(s, "detail") }}, PushButton{AssignTo: &s.redeem, Text: "确认兑付到账", Visible: kind == "acceptance", OnClicked: func() { ui.loadSelectedCustomerDocument(s, "redeem") }}, PushButton{AssignTo: &s.reverse, Text: "撤销到账", Visible: kind == "acceptance", OnClicked: func() { ui.loadSelectedCustomerDocument(s, "reverse") }}, PushButton{AssignTo: &s.void, Text: "作废登记", OnClicked: func() { ui.loadSelectedCustomerDocument(s, "void") }}, HSpacer{}}},
		TableView{AssignTo: &s.table, Model: []customerDocumentRow{}, AlternatingRowBG: true, StretchFactor: 1, OnCurrentIndexChanged: func() { ui.updateDocumentActions(s) }, OnItemActivated: func() { ui.loadSelectedCustomerDocument(s, "detail") }, Columns: []TableViewColumn{{Title: "客户", DataMember: "Customer", Width: 180}, {Title: "票据号码", DataMember: "Number", Width: 210}, {Title: "登记日期", DataMember: "Date", Width: 100}, {Title: "到期日期", DataMember: "Due", Width: 100}, {Title: "金额", DataMember: "Amount", Width: 110}, {Title: "状态", DataMember: "Status", Width: 130}, {Title: "附件", DataMember: "Attachments", Width: 65}}},
		Composite{Layout: HBox{Spacing: 8}, Children: []Widget{Label{AssignTo: &s.info, Text: "尚未加载"}, HSpacer{}, PushButton{AssignTo: &s.prev, Text: "上一页", OnClicked: func() {
			if s.page > 1 {
				s.page--
				ui.loadCustomerDocuments(s)
			}
		}}, PushButton{AssignTo: &s.next, Text: "下一页", OnClicked: func() { s.page++; ui.loadCustomerDocuments(s) }}}},
	}}
}
func (ui *mainUI) initializeCustomerDocuments(s *customerDocumentsUI) {
	ui.updateDocumentActions(s)
	ui.loadCustomerDocuments(s)
	ui.loadDocumentCustomers(s)
}
func (ui *mainUI) loadDocumentCustomers(s *customerDocumentsUI) {
	guardedGo(func() {
		items, err := ui.session.Client.Customers(context.Background())
		if s.closed.Load() {
			return
		}
		ui.window.Synchronize(func() {
			if s.closed.Load() || s.customer.IsDisposed() {
				return
			}
			if err != nil {
				s.info.SetText(s.info.Text() + "；客户筛选加载失败，请点击查询重试")
				return
			}
			selectedID := ""
			selectedIndex := s.customer.CurrentIndex() - 1
			var selectedCustomer api.Customer
			if selectedIndex >= 0 && selectedIndex < len(s.customers) {
				selectedCustomer = s.customers[selectedIndex]
				selectedID = selectedCustomer.ID
			}
			found := false
			for _, c := range items {
				if c.ID == selectedID {
					found = true
				}
			}
			if selectedID != "" && !found {
				items = append(items, selectedCustomer)
			}
			s.customers = items
			labels := []string{"全部客户"}
			selectedIndex = 0
			for i, c := range items {
				labels = append(labels, c.Name)
				if c.ID == selectedID {
					selectedIndex = i + 1
				}
			}
			s.customer.SetModel(labels)
			s.customer.SetCurrentIndex(selectedIndex)
		})
	})
}
func (ui *mainUI) updateDocumentActions(s *customerDocumentsUI) {
	if s == nil || s.add == nil {
		return
	}
	index := s.table.CurrentIndex()
	selected := index >= 0 && index < len(s.rows)
	editable := false
	if selected {
		status := s.rows[index].Record.Status
		editable = status == "pending" || status == "active"
	}
	can := func(action string) bool {
		return !s.busy && hasButton(ui.session.Perms.Buttons, "finance:"+s.kind+":"+action)
	}
	s.add.SetEnabled(can("add"))
	s.edit.SetEnabled(editable && can("edit"))
	s.detail.SetEnabled(selected && !s.busy)
	s.redeem.SetEnabled(selected && editable && can("redeem"))
	s.void.SetEnabled(editable && can("void"))
	s.reverse.SetEnabled(selected && s.rows[index].Record.Status == "redeemed" && can("reverse"))
	s.query.SetEnabled(!s.busy)
}
func (ui *mainUI) loadCustomerDocuments(s *customerDocumentsUI) {
	if s == nil || s.closed.Load() || s.table == nil {
		return
	}
	if !hasButton(ui.session.Perms.Buttons, "finance:"+s.kind+":list") {
		return
	}
	if s.cancel != nil {
		s.cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.generation++
	generation := s.generation
	f := api.DocumentFilters{Keyword: strings.TrimSpace(s.keyword.Text())}
	idx := s.customer.CurrentIndex() - 1
	if idx >= 0 && idx < len(s.customers) {
		f.CustomerID = s.customers[idx].ID
	}
	status := s.status.CurrentIndex()
	if s.kind == "acceptance" {
		f.Status = []string{"", "pending", "redeemed", "voided"}[max(0, status)]
	} else {
		f.Status = []string{"", "active", "voided"}[max(0, status)]
	}
	if s.dates.Checked() {
		a, b := s.start.Date(), s.end.Date()
		f.DateStart = documentCalendarDate(a)
		f.DateEnd = documentCalendarDate(b) + 24*60*60 - 1
		if f.DateStart > f.DateEnd {
			s.info.SetText("开始日期不能晚于结束日期")
			return
		}
	}
	page := s.page
	s.busy = true
	s.prev.SetEnabled(false)
	s.next.SetEnabled(false)
	s.rows = nil
	s.table.SetModel([]customerDocumentRow{})
	s.info.SetText("正在加载台账……")
	ui.updateDocumentActions(s)
	guardedGo(func() {
		defer cancel()
		result, err := ui.session.Client.CustomerDocuments(ctx, s.kind, page, 20, f)
		if ctx.Err() != nil || s.closed.Load() {
			return
		}
		ui.window.Synchronize(func() {
			if s.closed.Load() || s.table.IsDisposed() || generation != s.generation {
				return
			}
			s.busy = false
			defer ui.updateDocumentActions(s)
			if err != nil {
				s.info.SetText(requestFailureText(err))
				s.summary.SetText("汇总未加载")
				return
			}
			if page > 1 && int64((page-1)*20) >= result.Total {
				s.page = max(1, int((result.Total+19)/20))
				ui.loadCustomerDocuments(s)
				return
			}
			rows := []customerDocumentRow{}
			for _, r := range result.List {
				amount, date := r.Amount, r.ReceivedDate
				if s.kind == "invoice" {
					amount, date = r.TotalAmount, r.InvoiceDate
				}
				rows = append(rows, customerDocumentRow{Customer: r.CustomerName, Number: documentNumber(r), Date: documentDate(date), Due: documentDate(r.DueDate), Amount: fmt.Sprintf("%.2f", amount), Status: documentStatus(r.Status, r.DueDate), Attachments: fmt.Sprint(len(r.Annex) + len(r.RedemptionAnnex)), Record: r})
			}
			s.rows = rows
			s.table.SetModel(rows)
			s.info.SetText(fmt.Sprintf("第 %d 页 · 共 %d 条 · 每页20条", page, result.Total))
			sum := result.Summary
			if s.kind == "acceptance" {
				s.summary.SetText(fmt.Sprintf("当前筛选：待兑付 %.2f · 其中逾期 %.2f · 已到账 %.2f · 已作废 %.2f", sum.PendingAmount, sum.OverdueAmount, sum.RedeemedAmount, sum.VoidedAmount))
			} else {
				s.summary.SetText(fmt.Sprintf("当前筛选有效发票：未税 %.2f · 税额 %.2f · 价税合计 %.2f · 作废 %.2f", sum.AmountExcludingTax, sum.TaxAmount, sum.TotalAmount, sum.VoidedAmount))
			}
			s.prev.SetEnabled(page > 1)
			s.next.SetEnabled(int64(page*20) < result.Total)
		})
	})
}
func (ui *mainUI) openSelectedCustomerDocuments(kind string) {
	customer, k, ok := ui.selectedPartner()
	if !ok || k.Key != "customer" {
		return
	}
	ui.openCustomerDocumentsFor(kind, customer)
}
func (ui *mainUI) openCustomerDocumentsFor(kind string, c partnerDetail) {
	if !hasButton(ui.session.Perms.Buttons, "finance:"+kind+":list") {
		return
	}
	ui.openTab("customer_" + kind)
	s := ui.documentState(kind)
	if s == nil {
		return
	}
	idx := -1
	for i, one := range s.customers {
		if one.ID == c.ID {
			idx = i
			break
		}
	}
	if idx < 0 {
		s.customers = append(s.customers, api.Customer{ID: c.ID, Name: c.Name})
		idx = len(s.customers) - 1
	}
	labels := []string{"全部客户"}
	for _, one := range s.customers {
		labels = append(labels, one.Name)
	}
	s.customer.SetModel(labels)
	s.customer.SetCurrentIndex(idx + 1)
	s.page = 1
	ui.loadCustomerDocuments(s)
}
func (ui *mainUI) loadSelectedCustomerDocument(s *customerDocumentsUI, action string) {
	index := s.table.CurrentIndex()
	if index < 0 || index >= len(s.rows) || s.busy {
		return
	}
	if action != "detail" && !hasButton(ui.session.Perms.Buttons, "finance:"+s.kind+":"+action) {
		return
	}
	id := s.rows[index].Record.ID
	s.busy = true
	ui.updateDocumentActions(s)
	guardedGo(func() {
		record, err := ui.session.Client.CustomerDocument(context.Background(), s.kind, id)
		if s.closed.Load() {
			return
		}
		ui.window.Synchronize(func() {
			if s.closed.Load() || s.table.IsDisposed() {
				return
			}
			s.busy = false
			ui.updateDocumentActions(s)
			if err != nil {
				walk.MsgBox(ui.window, "读取失败", requestFailureText(err), walk.MsgBoxIconError)
				return
			}
			switch action {
			case "detail":
				ui.showCustomerDocument(s, record)
			case "edit":
				ui.editCustomerDocument(s, record)
			default:
				ui.customerDocumentAction(s, record, action)
			}
		})
	})
}
func (ui *mainUI) showCustomerDocument(s *customerDocumentsUI, r api.CustomerDocument) {
	var dlg *walk.Dialog
	var close *walk.PushButton
	text := fmt.Sprintf("客户：%s\r\n票号：%s\r\n状态：%s\r\n登记人：%s\r\n备注：%s", r.CustomerName, documentNumber(r), documentStatus(r.Status, r.DueDate), r.CreatorName, r.Remark)
	if s.kind == "acceptance" {
		text += fmt.Sprintf("\r\n金额：%.2f\r\n出票：%s · 收票：%s · 到期：%s\r\n到账日期：%s\r\n关联回款：%s", r.Amount, documentDate(r.IssueDate), documentDate(r.ReceivedDate), documentDate(r.DueDate), documentDate(r.RedemptionDate), displayMaterialValue(r.TransactionID))
	} else {
		text += fmt.Sprintf("\r\n开票日期：%s\r\n未税金额：%.2f · 税率：%g%%\r\n税额：%.2f · 价税合计：%.2f", documentDate(r.InvoiceDate), r.AmountExcludingTax, r.TaxRate, r.TaxAmount, r.TotalAmount)
	}
	text += "\r\n\r\n操作记录"
	for _, h := range r.History {
		text += fmt.Sprintf("\r\n%s · %s · %s · %s", time.Unix(h.Time, 0).In(documentLocation).Format("2006-01-02 15:04"), documentHistoryAction(h.Action), h.OperatorName, h.Remark)
	}
	err := Dialog{AssignTo: &dlg, Title: documentTitle(s.kind) + "详情", MinSize: Size{Width: 740, Height: 560}, Layout: VBox{Margins: Margins{Left: 16, Top: 16, Right: 16, Bottom: 16}, Spacing: 10}, DefaultButton: &close, Children: []Widget{TextEdit{Text: text, ReadOnly: true, VScroll: true, StretchFactor: 1}, Composite{Layout: HBox{Spacing: 8}, Children: []Widget{PushButton{Text: fmt.Sprintf("票据附件 (%d)", len(r.Annex)), Enabled: len(r.Annex) > 0, OnClicked: func() { showFinancialAttachments(dlg, ui.session.Client, r.Annex) }}, PushButton{Text: fmt.Sprintf("到账凭证 (%d)", len(r.RedemptionAnnex)), Visible: s.kind == "acceptance", Enabled: len(r.RedemptionAnnex) > 0, OnClicked: func() { showFinancialAttachments(dlg, ui.session.Client, r.RedemptionAnnex) }}, HSpacer{}, PushButton{AssignTo: &close, Text: "关闭", OnClicked: func() { dlg.Accept() }}}}}}.Create(ui.window)
	if err != nil {
		walk.MsgBox(ui.window, "打开失败", err.Error(), walk.MsgBoxIconError)
		return
	}
	dlg.Run()
}
func documentHistoryAction(value string) string {
	switch value {
	case "create":
		return "新增"
	case "update":
		return "修改"
	case "redeem":
		return "兑付到账"
	case "void":
		return "作废"
	case "reverse":
		return "撤销到账"
	}
	return value
}
func showFinancialAttachments(owner walk.Form, client *api.Client, refs []string) {
	var dlg *walk.Dialog
	var list *walk.ListBox
	var close *walk.PushButton
	labels := []string{}
	for i, ref := range refs {
		labels = append(labels, fmt.Sprintf("%d. %s", i+1, path.Base(ref)))
	}
	open := func() {
		idx := list.CurrentIndex()
		if idx < 0 || idx >= len(refs) {
			return
		}
		ref := refs[idx]
		if financialAttachmentIsImage(ref) {
			ShowOrderAttachments(dlg, client, config.ImageBaseURL(), "票据附件", []string{ref})
			return
		}
		resolved, err := api.ResolveImageURL(config.ImageBaseURL(), ref)
		if err != nil {
			walk.MsgBox(dlg, "打开失败", err.Error(), walk.MsgBoxIconError)
			return
		}
		parsed, _ := url.Parse(resolved)
		if parsed == nil || parsed.Host == "" {
			return
		}
		verb, _ := syscall.UTF16PtrFromString("open")
		target, _ := syscall.UTF16PtrFromString(resolved)
		if result := win.ShellExecute(dlg.Handle(), verb, target, nil, nil, win.SW_SHOWNORMAL); !result {
			walk.MsgBox(dlg, "打开失败", "请检查系统浏览器或文件查看器。", walk.MsgBoxIconError)
		}
	}
	if err := (Dialog{AssignTo: &dlg, Title: "票据附件", MinSize: Size{Width: 650, Height: 380}, Layout: VBox{Margins: Margins{Left: 16, Top: 16, Right: 16, Bottom: 16}, Spacing: 10}, DefaultButton: &close, Children: []Widget{Label{Text: "图片在客户端预览；PDF 和其他文件由系统浏览器打开或下载。"}, ListBox{AssignTo: &list, Model: labels, CurrentIndex: 0, StretchFactor: 1, OnItemActivated: open}, Composite{Layout: HBox{Spacing: 8}, Children: []Widget{PushButton{Text: "打开 / 下载所选附件", OnClicked: open}, HSpacer{}, PushButton{AssignTo: &close, Text: "关闭", OnClicked: func() { dlg.Accept() }}}}}}).Create(owner); err == nil {
		dlg.Run()
	} else {
		walk.MsgBox(owner, "附件窗口错误", err.Error(), walk.MsgBoxIconError)
	}
}

func (ui *mainUI) editCustomerDocument(s *customerDocumentsUI, r api.CustomerDocument) {
	action := "add"
	if r.ID != "" {
		action = "edit"
		if r.Status != "pending" && r.Status != "active" {
			walk.MsgBox(ui.window, "无法修改", "仅待兑付承兑或有效发票可以修改。", walk.MsgBoxIconWarning)
			return
		}
	}
	if !hasButton(ui.session.Perms.Buttons, "finance:"+s.kind+":"+action) {
		return
	}
	var dlg *walk.Dialog
	var customer *walk.ComboBox
	var number, remark *walk.LineEdit
	var amount, taxRate, taxAmount *walk.NumberEdit
	var totalAmount *walk.Label
	var date, issue, due *walk.DateEdit
	var fileList *walk.ListBox
	var info *walk.Label
	var save, upload, remove, preview, close *walk.PushButton
	var closed atomic.Bool
	busy := false
	var submission documentSubmission[api.CustomerDocument]
	annex := append([]string{}, r.Annex...)
	customers := append([]api.Customer{}, s.customers...)
	labels := []string{"请选择客户"}
	idx := 0
	for i, c := range customers {
		labels = append(labels, c.Name)
		if c.ID == r.CustomerID {
			idx = i + 1
		}
	}
	if r.CustomerID != "" && idx == 0 {
		customers = append(customers, api.Customer{ID: r.CustomerID, Name: r.CustomerName})
		labels = append(labels, r.CustomerName)
		idx = len(customers)
	}
	if r.ID == "" && s.customer.CurrentIndex() > 0 {
		idx = s.customer.CurrentIndex()
	}
	value := r.Amount
	recordDate := r.ReceivedDate
	if s.kind == "invoice" {
		value = r.AmountExcludingTax
		recordDate = r.InvoiceDate
	}
	asTime := func(v int64) time.Time {
		if v > 0 {
			return time.Unix(v, 0).In(documentLocation)
		}
		return time.Now().In(documentLocation)
	}
	key := newManualTransactionIdempotencyKey("document-" + s.kind)
	setBusy := func(v bool) {
		busy = v
		if dlg != nil {
			for _, w := range []walk.Widget{customer, number, remark, amount, taxRate, taxAmount, date, issue, due, fileList, upload, remove} {
				if w != nil {
					w.SetEnabled(!v && !submission.Uncertain())
				}
			}
			for _, w := range []walk.Widget{save, close, preview} {
				if w != nil {
					w.SetEnabled(!v)
				}
			}
			if save != nil {
				if submission.Uncertain() {
					save.SetText("重试原操作")
				} else {
					save.SetText("保存登记")
				}
			}
		}
	}
	updateFiles := func() {
		names := []string{}
		for _, ref := range annex {
			names = append(names, path.Base(ref))
		}
		fileList.SetModel(names)
		if len(names) > 0 {
			fileList.SetCurrentIndex(len(names) - 1)
		}
	}
	err := (Dialog{AssignTo: &dlg, Title: map[bool]string{true: "修改", false: "新增"}[r.ID != ""] + documentTitle(s.kind), MinSize: Size{Width: 760, Height: 640}, Layout: VBox{Margins: Margins{Left: 16, Top: 16, Right: 16, Bottom: 16}, Spacing: 10}, DefaultButton: &save, CancelButton: &close, Children: []Widget{
		Label{Text: "登记票据不会改变应收；请核对客户、号码、金额和日期。"}, Composite{Layout: Grid{Columns: 4, Spacing: 8}, Children: []Widget{Label{Text: "客户 *"}, ComboBox{AssignTo: &customer, Model: labels, CurrentIndex: idx, ColumnSpan: 3}, Label{Text: "票据号码 *"}, LineEdit{AssignTo: &number, Text: documentNumber(r), ColumnSpan: 3}, Label{Text: map[string]string{"acceptance": "票面金额 *", "invoice": "未税金额 *"}[s.kind]}, NumberEdit{AssignTo: &amount, Value: value, Decimals: 2, MinValue: 0, MaxValue: documentMaxAmount}, Label{Text: "税率 (%)", Visible: s.kind == "invoice"}, NumberEdit{AssignTo: &taxRate, Visible: s.kind == "invoice", Value: r.TaxRate, Decimals: 4, MinValue: 0, MaxValue: 100}, Label{Text: "票面税额（可校正）", Visible: s.kind == "invoice"}, NumberEdit{AssignTo: &taxAmount, Value: r.TaxAmount, Decimals: 2, MinValue: 0, MaxValue: documentMaxAmount, Visible: s.kind == "invoice"}, Label{Text: "价税合计", Visible: s.kind == "invoice"}, Label{AssignTo: &totalAmount, Text: fmt.Sprintf("%.2f", r.TotalAmount), Visible: s.kind == "invoice"}, Label{Text: map[string]string{"acceptance": "收票日期 *", "invoice": "开票日期 *"}[s.kind]}, DateEdit{AssignTo: &date, Date: asTime(recordDate), Format: "yyyy-MM-dd"}, Label{Text: "出票日期 *", Visible: s.kind == "acceptance"}, DateEdit{AssignTo: &issue, Date: asTime(r.IssueDate), Format: "yyyy-MM-dd", Visible: s.kind == "acceptance"}, Label{Text: "到期日期 *", Visible: s.kind == "acceptance"}, DateEdit{AssignTo: &due, Date: asTime(r.DueDate), Format: "yyyy-MM-dd", Visible: s.kind == "acceptance"}, Label{Text: "备注"}, LineEdit{AssignTo: &remark, Text: r.Remark}}},
		GroupBox{Title: "票据附件（图片 / PDF / OFD / Word / Excel，最多 10 个）", Layout: VBox{Spacing: 8}, StretchFactor: 1, Children: []Widget{ListBox{AssignTo: &fileList, Model: []string{}, StretchFactor: 1}, Composite{Layout: HBox{Spacing: 8}, Children: []Widget{PushButton{AssignTo: &upload, Text: "上传附件"}, PushButton{AssignTo: &remove, Text: "移除所选", OnClicked: func() {
			if busy || submission.Uncertain() {
				return
			}
			i := fileList.CurrentIndex()
			if i >= 0 && i < len(annex) {
				annex = append(annex[:i], annex[i+1:]...)
				updateFiles()
			}
		}}, PushButton{AssignTo: &preview, Text: "查看附件", OnClicked: func() {
			if len(annex) > 0 {
				showFinancialAttachments(dlg, ui.session.Client, annex)
			}
		}}}}}}, Label{AssignTo: &info, Text: "附件从表单移除不会直接删除服务器文件。"}, Composite{Layout: HBox{Spacing: 8}, Children: []Widget{HSpacer{}, PushButton{AssignTo: &save, Text: "保存登记"}, PushButton{AssignTo: &close, Text: "取消", OnClicked: func() {
			if !busy {
				dlg.Cancel()
			}
		}}}}}}).Create(ui.window)
	if err != nil {
		walk.MsgBox(ui.window, "打开失败", err.Error(), walk.MsgBoxIconError)
		return
	}
	dlg.Disposing().Attach(func() { closed.Store(true) })
	dlg.Closing().Attach(func(c *bool, _ walk.CloseReason) {
		if busy {
			*c = true
		}
	})
	customer.SetEnabled(true)
	syncTotal := func() {
		totalAmount.SetText(fmt.Sprintf("%.2f", documentMoneySum(amount.Value(), taxAmount.Value())))
	}
	recalculate := func() {
		if s.kind == "invoice" {
			taxAmount.SetValue(documentTaxAmount(amount.Value(), taxRate.Value()))
			syncTotal()
		}
	}
	amount.ValueChanged().Attach(recalculate)
	taxRate.ValueChanged().Attach(recalculate)
	taxAmount.ValueChanged().Attach(syncTotal)
	updateFiles()
	upload.Clicked().Attach(func() {
		if busy || submission.Uncertain() {
			return
		}
		if len(annex) >= 10 {
			info.SetText("最多10个附件")
			return
		}
		fd := walk.FileDialog{Title: "选择票据图片或文件", Filter: "票据文件 (*.png;*.jpg;*.jpeg;*.gif;*.webp;*.bmp;*.pdf;*.ofd;*.doc;*.docx;*.xls;*.xlsx)|*.png;*.jpg;*.jpeg;*.gif;*.webp;*.bmp;*.pdf;*.ofd;*.doc;*.docx;*.xls;*.xlsx"}
		ok, err := fd.ShowOpen(dlg)
		if err != nil || !ok {
			return
		}
		setBusy(true)
		info.SetText("正在上传附件……")
		guardedGo(func() {
			ref, err := ui.session.Client.UploadDocumentAttachment(context.Background(), fd.FilePath, s.kind)
			if closed.Load() {
				return
			}
			dlg.Synchronize(func() {
				if closed.Load() {
					return
				}
				setBusy(false)
				if err != nil {
					info.SetText("上传失败：" + requestFailureText(err))
					return
				}
				annex = append(annex, ref)
				updateFiles()
				info.SetText("上传完成，保存登记后关联附件。")
			})
		})
	})
	save.Clicked().Attach(func() {
		if busy {
			return
		}
		i := customer.CurrentIndex() - 1
		if i < 0 || i >= len(customers) {
			info.SetText("请选择客户")
			return
		}
		if strings.TrimSpace(number.Text()) == "" || amount.Value() <= 0 {
			info.SetText("请填写号码及大于零的金额")
			return
		}
		record := api.CustomerDocument{ID: r.ID, Version: r.Version, CustomerID: customers[i].ID, Annex: append([]string{}, annex...), Remark: strings.TrimSpace(remark.Text()), IdempotencyKey: key}
		if s.kind == "acceptance" {
			record.BillNumber = strings.TrimSpace(number.Text())
			record.Amount = amount.Value()
			record.ReceivedDate = documentCalendarDate(date.Date())
			record.IssueDate = documentCalendarDate(issue.Date())
			record.DueDate = documentCalendarDate(due.Date())
			if record.IssueDate > record.ReceivedDate {
				info.SetText("出票日期不能晚于收票日期")
				return
			}
			if record.IssueDate > record.DueDate {
				info.SetText("到期日期不能早于出票日期")
				return
			}
		} else {
			record.InvoiceNumber = strings.TrimSpace(number.Text())
			record.InvoiceDate = documentCalendarDate(date.Date())
			record.AmountExcludingTax = amount.Value()
			record.TaxRate = taxRate.Value()
			record.TaxAmount = taxAmount.Value()
			total, totalErr := documentInvoiceTotal(record.AmountExcludingTax, record.TaxAmount)
			if totalErr != nil {
				info.SetText(totalErr.Error())
				return
			}
			record.TotalAmount = total
		}
		record, beginErr := submission.Begin(record)
		if beginErr != nil {
			info.SetText(beginErr.Error())
			return
		}
		setBusy(true)
		info.SetText("正在保存……")
		guardedGo(func() {
			_, err := ui.session.Client.SaveCustomerDocument(context.Background(), s.kind, record)
			submission.Finish(err)
			if closed.Load() {
				return
			}
			dlg.Synchronize(func() {
				if closed.Load() {
					return
				}
				setBusy(false)
				if err != nil {
					if submission.Uncertain() {
						info.SetText("结果未确认，表单已锁定。点击“重试原操作”会使用原请求核对；也可关闭后刷新台账。")
					} else {
						key = newManualTransactionIdempotencyKey("document-" + s.kind)
						info.SetText("保存失败：" + requestFailureText(err))
					}
					return
				}
				dlg.Accept()
				ui.loadCustomerDocuments(s)
			})
		})
	})
	dlg.Run()
}

func (ui *mainUI) customerDocumentAction(s *customerDocumentsUI, r api.CustomerDocument, action string) {
	if !hasButton(ui.session.Perms.Buttons, "finance:"+s.kind+":"+action) {
		return
	}
	if (action == "reverse" && r.Status != "redeemed") || (action != "reverse" && r.Status != "pending" && r.Status != "active") {
		walk.MsgBox(ui.window, "状态已改变", "该记录已到账或已作废，请刷新后查看。", walk.MsgBoxIconWarning)
		return
	}
	var dlg *walk.Dialog
	var date *walk.DateEdit
	var remark *walk.LineEdit
	var payments *walk.ComboBox
	var info *walk.Label
	var save, close, upload, removeAttachment, previewAttachment, loadPayments, previousPayment, nextPayment *walk.PushButton
	var files *walk.ListBox
	var closed atomic.Bool
	busy := false
	var submission documentSubmission[api.DocumentActionRequest]
	annex := []string{}
	choices := []api.AcceptancePayment{}
	paymentPage := 1
	var paymentTotal int64
	request := api.DocumentActionRequest{ID: r.ID, Version: r.Version, IdempotencyKey: newManualTransactionIdempotencyKey("document-" + action + "-" + r.ID), RedemptionAnnex: []string{}}
	setBusy := func(v bool) {
		busy = v
		for _, w := range []walk.Widget{date, remark, payments, upload, removeAttachment, loadPayments} {
			if w != nil {
				w.SetEnabled(!v && !submission.Uncertain())
			}
		}
		for _, w := range []walk.Widget{save, close, previewAttachment} {
			if w != nil {
				w.SetEnabled(!v)
			}
		}
		if save != nil && submission.Uncertain() {
			save.SetText("重试原操作")
		}
		if previousPayment != nil {
			previousPayment.SetEnabled(!v && !submission.Uncertain() && paymentPage > 1)
		}
		if nextPayment != nil {
			nextPayment.SetEnabled(!v && !submission.Uncertain() && int64(paymentPage*20) < paymentTotal)
		}
	}
	title := "作废登记"
	hint := "作废后保留记录和附件，不改变应收。请填写原因。"
	if action == "reverse" {
		title = "撤销到账"
		hint = "自动生成的回款将冲回并恢复应收；绑定的既有人工回款仅解除关联。操作留痕，请填写原因。"
	}
	if action == "redeem" {
		title = "确认兑付到账"
		hint = "新建回款会减少应收；若已手工入账，请加载并选择已有等额回款。到账凭证选填。"
	}
	err := (Dialog{AssignTo: &dlg, Title: title, MinSize: Size{Width: 760, Height: 490}, Layout: VBox{Margins: Margins{Left: 16, Top: 16, Right: 16, Bottom: 16}, Spacing: 10}, DefaultButton: &save, CancelButton: &close, Children: []Widget{Label{Text: fmt.Sprintf("%s · %s · 金额 %.2f", r.CustomerName, documentNumber(r), documentRecordAmount(s.kind, r))}, Label{Text: hint}, DateEdit{AssignTo: &date, Date: time.Now().In(documentLocation), Format: "yyyy-MM-dd", Visible: action == "redeem"}, ComboBox{AssignTo: &payments, Model: []string{"新建回款（本次实际到账，减少应收）"}, CurrentIndex: 0, Visible: action == "redeem"}, Composite{Visible: action == "redeem", Layout: HBox{Spacing: 8}, Children: []Widget{PushButton{AssignTo: &loadPayments, Text: "加载已有等额回款"}, PushButton{AssignTo: &previousPayment, Text: "上一页回款"}, PushButton{AssignTo: &nextPayment, Text: "下一页回款"}}}, LineEdit{AssignTo: &remark, CueBanner: map[bool]string{true: "到账备注（选填）", false: "操作原因（必填）"}[action == "redeem"]}, ListBox{AssignTo: &files, Model: []string{}, Visible: action == "redeem", StretchFactor: 1}, PushButton{AssignTo: &upload, Text: "上传到账凭证（选填）", Visible: action == "redeem"}, Composite{Visible: action == "redeem", Layout: HBox{Spacing: 8}, Children: []Widget{PushButton{AssignTo: &removeAttachment, Text: "移除所选凭证", OnClicked: func() {
		if busy || submission.Uncertain() {
			return
		}
		i := files.CurrentIndex()
		if i >= 0 && i < len(annex) {
			annex = append(annex[:i], annex[i+1:]...)
			names := []string{}
			for _, a := range annex {
				names = append(names, path.Base(a))
			}
			files.SetModel(names)
		}
	}}, PushButton{AssignTo: &previewAttachment, Text: "查看到账凭证", OnClicked: func() {
		if len(annex) > 0 {
			showFinancialAttachments(dlg, ui.session.Client, annex)
		}
	}}}}, Label{AssignTo: &info, Text: "请核对后提交，写入不会自动重试。"}, Composite{Layout: HBox{Spacing: 8}, Children: []Widget{HSpacer{}, PushButton{AssignTo: &save, Text: title}, PushButton{AssignTo: &close, Text: "取消", OnClicked: func() {
		if !busy {
			dlg.Cancel()
		}
	}}}}}}).Create(ui.window)
	if err != nil {
		walk.MsgBox(ui.window, "打开失败", err.Error(), walk.MsgBoxIconError)
		return
	}
	dlg.Disposing().Attach(func() { closed.Store(true) })
	dlg.Closing().Attach(func(c *bool, _ walk.CloseReason) {
		if busy {
			*c = true
		}
	})
	fetchPayments := func() {
		if busy || submission.Uncertain() {
			return
		}
		setBusy(true)
		info.SetText("正在查询尚未关联的等额回款……")
		page := paymentPage
		guardedGo(func() {
			data, err := ui.session.Client.AcceptancePayments(context.Background(), r.CustomerID, r.Amount, page, 20)
			if closed.Load() {
				return
			}
			dlg.Synchronize(func() {
				if closed.Load() {
					return
				}
				setBusy(false)
				if err != nil {
					info.SetText(requestFailureText(err))
					return
				}
				choices = data.List
				paymentTotal = data.Total
				names := []string{"新建回款（本次实际到账，减少应收）"}
				for _, p := range choices {
					names = append(names, fmt.Sprintf("%s · %s · %.2f · %s", p.Code, documentDate(p.Time), p.Amount, p.Remark))
				}
				payments.SetModel(names)
				payments.SetCurrentIndex(0)
				previousPayment.SetEnabled(page > 1)
				nextPayment.SetEnabled(int64(page*20) < data.Total)
				info.SetText(fmt.Sprintf("已有回款第 %d 页，共 %d 条；选择已有回款只绑定，不再次改变应收。", page, data.Total))
			})
		})
	}
	loadPayments.Clicked().Attach(func() {
		if busy || submission.Uncertain() {
			return
		}
		paymentPage = 1
		fetchPayments()
	})
	previousPayment.Clicked().Attach(func() {
		if !busy && !submission.Uncertain() && paymentPage > 1 {
			paymentPage--
			fetchPayments()
		}
	})
	nextPayment.Clicked().Attach(func() {
		if busy || submission.Uncertain() || int64(paymentPage*20) >= paymentTotal {
			return
		}
		paymentPage++
		fetchPayments()
	})
	previousPayment.SetEnabled(false)
	nextPayment.SetEnabled(false)
	upload.Clicked().Attach(func() {
		if busy || submission.Uncertain() {
			return
		}
		if len(annex) >= 10 {
			info.SetText("最多10个附件")
			return
		}
		fd := walk.FileDialog{Title: "选择到账凭证", Filter: "票据文件 (*.png;*.jpg;*.jpeg;*.gif;*.webp;*.bmp;*.pdf;*.ofd;*.doc;*.docx;*.xls;*.xlsx)|*.png;*.jpg;*.jpeg;*.gif;*.webp;*.bmp;*.pdf;*.ofd;*.doc;*.docx;*.xls;*.xlsx"}
		ok, err := fd.ShowOpen(dlg)
		if err != nil || !ok {
			return
		}
		setBusy(true)
		guardedGo(func() {
			ref, err := ui.session.Client.UploadDocumentAttachment(context.Background(), fd.FilePath, "acceptance_redemption")
			if closed.Load() {
				return
			}
			dlg.Synchronize(func() {
				if closed.Load() {
					return
				}
				setBusy(false)
				if err != nil {
					info.SetText(requestFailureText(err))
					return
				}
				annex = append(annex, ref)
				names := []string{}
				for _, a := range annex {
					names = append(names, path.Base(a))
				}
				files.SetModel(names)
				info.SetText("凭证已上传，提交到账后关联。")
			})
		})
	})
	save.Clicked().Attach(func() {
		if busy {
			return
		}
		request.Remark = strings.TrimSpace(remark.Text())
		request.Reason = request.Remark
		if action != "redeem" && request.Reason == "" {
			info.SetText("请填写操作原因")
			return
		}
		request.RedemptionDate = documentCalendarDate(date.Date())
		if action == "redeem" && request.RedemptionDate < documentCalendarDate(time.Unix(r.ReceivedDate, 0).In(documentLocation)) {
			info.SetText("到账日期不能早于收票日期")
			return
		}
		request.RedemptionAnnex = append([]string{}, annex...)
		request.TransactionID = ""
		if i := payments.CurrentIndex() - 1; i >= 0 && i < len(choices) {
			request.TransactionID = choices[i].ID
		}
		confirm := "确认作废该登记？"
		if action == "reverse" {
			confirm = "确认撤销到账？自动生成回款会冲回并恢复应收；既有人工回款只解除承兑关联，不改变原回款。"
		}
		if action == "redeem" {
			confirm = fmt.Sprintf("确认承兑 %.2f 元已兑付到账，并新建回款减少应收？", r.Amount)
			if request.TransactionID != "" {
				confirm = "确认绑定所选已入账回款？本次不会再次冲减应收。"
			}
		}
		if !submission.Uncertain() && walk.MsgBox(dlg, title, confirm, walk.MsgBoxYesNo|walk.MsgBoxIconQuestion) != walk.DlgCmdYes {
			return
		}
		req, beginErr := submission.Begin(request)
		if beginErr != nil {
			info.SetText(beginErr.Error())
			return
		}
		setBusy(true)
		info.SetText("正在提交……")
		guardedGo(func() {
			_, err := ui.session.Client.CustomerDocumentAction(context.Background(), s.kind, action, req)
			submission.Finish(err)
			if closed.Load() {
				return
			}
			dlg.Synchronize(func() {
				if closed.Load() {
					return
				}
				setBusy(false)
				if err != nil {
					if submission.Uncertain() {
						info.SetText("结果未确认，日期、回款和凭证已锁定。点击“重试原操作”使用原请求核对，请勿另行新建回款。")
					} else {
						request.IdempotencyKey = newManualTransactionIdempotencyKey("document-" + action + "-" + r.ID)
						info.SetText(requestFailureText(err))
					}
					return
				}
				dlg.Accept()
				ui.loadCustomerDocuments(s)
				if action == "redeem" || action == "reverse" {
					ui.refreshCustomerFinanceBalance()
				}
			})
		})
	})
	dlg.Run()
}
