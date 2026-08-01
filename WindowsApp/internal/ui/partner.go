package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"

	"zhengshi-wms-windowsapp/internal/api"
	"zhengshi-wms-windowsapp/internal/config"
)

type partnerKind struct {
	Key        string
	Label      string
	MenuPath   string
	Permission string
}

type partnerDetail struct {
	ID               string
	Category         string
	Type             string
	Code             string
	Name             string
	Status           string
	Manager          string
	Contact          string
	Email            string
	LegalPerson      string
	CreditIdentifier string
	Address          string
	Level            string
	Receivable       string
	CreditBalance    string
	Remark           string
	CreateBy         string
	CreatedAt        int64
	UpdatedAt        int64
}

type partnerRow struct {
	Category string
	Type     string
	Code     string
	Name     string
	Status   string
	Manager  string
	Contact  string
	Email    string
	Address  string
	Detail   partnerDetail
}

type partnerUI struct {
	kinds   []partnerKind
	kind    *walk.ComboBox
	name    *walk.LineEdit
	code    *walk.LineEdit
	manager *walk.LineEdit
	contact *walk.LineEdit
	email   *walk.LineEdit
	level   *walk.ComboBox
	table   *walk.TableView
	info    *walk.Label
	query   *walk.PushButton
	reset   *walk.PushButton
	prev    *walk.PushButton
	next    *walk.PushButton
	size    *walk.ComboBox

	page       int
	total      int64
	rows       []partnerRow
	generation int
	cancel     context.CancelFunc
}

type partnerLoadResult struct {
	total int64
	rows  []partnerRow
}

func availablePartnerKinds(perms api.Perms) []partnerKind {
	candidates := []partnerKind{
		{Key: "supplier", Label: "供应商", MenuPath: "/business_partner/supplier", Permission: "business_partner:supplier:list"},
		{Key: "customer", Label: "客户", MenuPath: "/business_partner/customer", Permission: "business_partner:customer:list"},
		{Key: "carrier", Label: "承运商", MenuPath: "/business_partner/carrier", Permission: "business_partner:carrier:list"},
	}
	result := make([]partnerKind, 0, len(candidates))
	for _, candidate := range candidates {
		if hasMenuPath(perms.Menus, candidate.MenuPath) && hasButton(perms.Buttons, candidate.Permission) {
			result = append(result, candidate)
		}
	}
	return result
}

func newPartnerUI(kinds []partnerKind) *partnerUI {
	return &partnerUI{kinds: append([]partnerKind(nil), kinds...), page: 1}
}

func (ui *mainUI) partnerPageWidget() TabPage {
	state := ui.partner
	return TabPage{
		AssignTo: &ui.partnerTab,
		Title:    closableTabTitle("合作伙伴"),
		Layout:   VBox{Margins: Margins{Left: 14, Top: 14, Right: 14, Bottom: 12}, Spacing: 10},
		Children: []Widget{
			Label{Text: "合作伙伴", Font: Font{Family: "Microsoft YaHei UI", PointSize: 15, Bold: true}},
			Label{
				Text:          "查询服务端已有的供应商、客户和承运商资料。双击一行可查看完整只读信息。",
				TextColor:     walk.RGB(85, 85, 85),
				Accessibility: Accessibility{Name: "合作伙伴页面说明"},
			},
			GroupBox{
				Title:  "筛选条件",
				Layout: Grid{Columns: 8, Margins: Margins{Left: 14, Top: 10, Right: 14, Bottom: 12}, Spacing: 8},
				Children: []Widget{
					Label{Text: "伙伴类型"},
					ComboBox{
						AssignTo: &state.kind, Model: partnerKindLabels(state.kinds), CurrentIndex: 0,
						MinSize: Size{Width: 150, Height: 28}, OnCurrentIndexChanged: ui.partnerKindChanged,
					},
					Label{Text: "名称"},
					LineEdit{AssignTo: &state.name, MinSize: Size{Width: 170, Height: 28}, CueBanner: "伙伴名称"},
					Label{Text: "编号"},
					LineEdit{AssignTo: &state.code, MinSize: Size{Width: 150, Height: 28}, CueBanner: "伙伴编号"},
					Label{Text: "负责人"},
					LineEdit{AssignTo: &state.manager, MinSize: Size{Width: 150, Height: 28}, CueBanner: "负责人"},
					Label{Text: "联系电话"},
					LineEdit{AssignTo: &state.contact, MinSize: Size{Width: 170, Height: 28}, CueBanner: "完整手机号"},
					Label{Text: "Email"},
					LineEdit{AssignTo: &state.email, MinSize: Size{Width: 180, Height: 28}, CueBanner: "完整邮箱地址"},
					Label{Text: "供应商等级"},
					ComboBox{
						AssignTo: &state.level, Model: supplierLevelLabels(), CurrentIndex: 0,
						MinSize:       Size{Width: 130, Height: 28},
						ToolTipText:   "仅查询供应商时可用",
						Accessibility: Accessibility{Name: "供应商等级筛选"},
					},
					HSpacer{ColumnSpan: 2},
					Composite{
						ColumnSpan: 8,
						Layout:     HBox{Spacing: 8},
						Children: []Widget{
							HSpacer{},
							PushButton{
								AssignTo: &state.reset, Text: "重置", MinSize: Size{Width: 88, Height: 30},
								Accessibility: Accessibility{Name: "重置合作伙伴筛选条件"},
								OnClicked:     ui.resetPartnerFilters,
							},
							PushButton{
								AssignTo: &state.query, Text: "查询", MinSize: Size{Width: 96, Height: 30},
								Accessibility: Accessibility{Name: "查询合作伙伴"},
								OnClicked: func() {
									state.page = 1
									ui.loadPartners()
								},
							},
						},
					},
				},
			},
			TableView{
				AssignTo:         &state.table,
				AlternatingRowBG: true,
				ColumnsOrderable: true,
				StretchFactor:    1,
				Accessibility: Accessibility{
					Name:        "合作伙伴查询结果",
					Description: "双击当前行查看合作伙伴完整资料",
				},
				OnItemActivated: ui.showSelectedPartner,
				Columns: []TableViewColumn{
					{Title: "类别", DataMember: "Category", Width: 75},
					{Title: "类型", DataMember: "Type", Width: 75},
					{Title: "编号", DataMember: "Code", Width: 120},
					{Title: "名称", DataMember: "Name", Width: 180},
					{Title: "状态", DataMember: "Status", Width: 85},
					{Title: "负责人", DataMember: "Manager", Width: 95},
					{Title: "联系电话", DataMember: "Contact", Width: 120},
					{Title: "邮箱", DataMember: "Email", Width: 170},
					{Title: "地址", DataMember: "Address", Width: 220},
				},
			},
			Composite{
				Layout: HBox{Spacing: 8},
				Children: []Widget{
					Label{AssignTo: &state.info, Text: "尚未加载"},
					HSpacer{},
					PushButton{
						Text: "查看详情", MinSize: Size{Width: 88, Height: 30},
						Accessibility: Accessibility{Name: "查看选中的合作伙伴详情"},
						OnClicked:     ui.showSelectedPartner,
					},
					Label{Text: "每页"},
					ComboBox{
						AssignTo: &state.size, Model: pageSizeLabels, CurrentIndex: 1, MinSize: Size{Width: 92},
						OnCurrentIndexChanged: func() {
							if ui.window != nil && state.size != nil && state.size.CurrentIndex() >= 0 {
								state.page = 1
								ui.loadPartners()
							}
						},
					},
					PushButton{AssignTo: &state.prev, Text: "上一页", OnClicked: func() {
						if state.page > 1 {
							state.page--
							ui.loadPartners()
						}
					}},
					PushButton{AssignTo: &state.next, Text: "下一页", OnClicked: func() {
						state.page++
						ui.loadPartners()
					}},
				},
			},
		},
	}
}

func partnerKindLabels(kinds []partnerKind) []string {
	labels := make([]string, len(kinds))
	for index, kind := range kinds {
		labels[index] = kind.Label
	}
	return labels
}

func (ui *mainUI) initializePartnerPage() {
	state := ui.partner
	if ui.partnerTab == nil || state == nil || state.name == nil || len(state.kinds) == 0 {
		return
	}
	state.generation++
	ui.updatePartnerLevelFilter()
	for _, edit := range []*walk.LineEdit{state.name, state.code, state.manager, state.contact, state.email} {
		edit.KeyDown().Attach(func(key walk.Key) {
			if key == walk.KeyReturn {
				state.page = 1
				ui.loadPartners()
			}
		})
	}
	ui.loadPartners()
}

func (ui *mainUI) releasePartnerPage() {
	if ui.partner == nil {
		return
	}
	kinds := append([]partnerKind(nil), ui.partner.kinds...)
	ui.partner.generation++
	if ui.partner.cancel != nil {
		ui.partner.cancel()
	}
	ui.partnerTab = nil
	ui.partner = newPartnerUI(kinds)
}

func (ui *mainUI) partnerKindChanged() {
	state := ui.partner
	if ui.window == nil || state == nil || state.kind == nil || state.kind.CurrentIndex() < 0 {
		return
	}
	ui.updatePartnerLevelFilter()
	state.page = 1
	ui.loadPartners()
}

func supplierLevelLabels() []string {
	return []string{"全部等级", "一级", "二级", "三级"}
}

func (ui *mainUI) updatePartnerLevelFilter() {
	state := ui.partner
	if state == nil || state.kind == nil || state.level == nil {
		return
	}
	index := state.kind.CurrentIndex()
	isSupplier := index >= 0 && index < len(state.kinds) && state.kinds[index].Key == "supplier"
	labels := []string{"不适用"}
	if isSupplier {
		labels = supplierLevelLabels()
	}
	_ = state.level.SetModel(labels)
	_ = state.level.SetCurrentIndex(0)
	state.level.SetEnabled(isSupplier)
}

func (ui *mainUI) resetPartnerFilters() {
	state := ui.partner
	if state == nil {
		return
	}
	state.name.SetText("")
	state.code.SetText("")
	state.manager.SetText("")
	state.contact.SetText("")
	state.email.SetText("")
	_ = state.level.SetCurrentIndex(0)
	state.page = 1
	ui.loadPartners()
}

func (ui *mainUI) loadPartners() {
	state := ui.partner
	if state == nil || state.table == nil || state.kind == nil || len(state.kinds) == 0 {
		return
	}
	index := state.kind.CurrentIndex()
	if index < 0 || index >= len(state.kinds) {
		return
	}
	if state.cancel != nil {
		state.cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	state.cancel = cancel
	state.generation++
	generation := state.generation
	page := state.page
	size := selectedPageSize(state.size)
	kind := state.kinds[index]
	filters := api.PartnerFilters{
		Name: state.name.Text(), Code: state.code.Text(),
		Manager: state.manager.Text(), Contact: state.contact.Text(), Email: state.email.Text(),
	}
	if kind.Key == "supplier" && state.level.CurrentIndex() > 0 {
		filters.Level = state.level.CurrentIndex()
	}
	state.info.SetText("正在加载线上" + kind.Label + "资料……")
	state.query.SetEnabled(false)
	state.reset.SetEnabled(false)
	state.prev.SetEnabled(false)
	state.next.SetEnabled(false)

	go func() {
		result, requestErr := ui.loadPartnerRows(ctx, kind, page, size, filters)
		if ctx.Err() != nil {
			return
		}
		ui.window.Synchronize(func() {
			if state != ui.partner || generation != state.generation || state.table == nil {
				return
			}
			state.query.SetEnabled(true)
			state.reset.SetEnabled(true)
			if requestErr != nil {
				state.info.SetText("加载失败：" + requestErr.Error() + "。请检查筛选条件后重试。")
				return
			}
			if modelErr := state.table.SetModel(result.rows); modelErr != nil {
				state.info.SetText("表格数据错误：" + modelErr.Error())
				return
			}
			state.rows = result.rows
			state.total = result.total
			state.info.SetText(fmt.Sprintf("%s | 第 %d 页 | 本页 %d 条 | 共 %d 条", kind.Label, page, len(result.rows), result.total))
			state.prev.SetEnabled(page > 1)
			state.next.SetEnabled(int64(page*size) < result.total)
		})
	}()
}

func (ui *mainUI) loadPartnerRows(
	ctx context.Context,
	kind partnerKind,
	page, size int,
	filters api.PartnerFilters,
) (partnerLoadResult, error) {
	switch kind.Key {
	case "supplier":
		result, err := ui.session.Client.SupplierDirectory(ctx, page, size, filters)
		rows := make([]partnerRow, 0, len(result.List))
		for _, item := range result.List {
			rows = append(rows, supplierPartnerRow(item))
		}
		return partnerLoadResult{total: result.Total, rows: rows}, err
	case "customer":
		result, err := ui.session.Client.CustomerDirectory(ctx, page, size, filters)
		rows := make([]partnerRow, 0, len(result.List))
		for _, item := range result.List {
			rows = append(rows, customerPartnerRow(item))
		}
		return partnerLoadResult{total: result.Total, rows: rows}, err
	case "carrier":
		result, err := ui.session.Client.CarrierDirectory(ctx, page, size, filters)
		rows := make([]partnerRow, 0, len(result.List))
		for _, item := range result.List {
			rows = append(rows, carrierPartnerRow(item))
		}
		return partnerLoadResult{total: result.Total, rows: rows}, err
	default:
		return partnerLoadResult{}, fmt.Errorf("不支持的合作伙伴类型：%s", kind.Label)
	}
}

func supplierPartnerRow(item api.Supplier) partnerRow {
	level := ""
	if item.Level > 0 {
		level = fmt.Sprintf("%d", item.Level)
	}
	detail := partnerDetail{
		ID: item.ID, Category: "供应商", Type: item.Type, Code: item.Code, Name: item.Name, Status: item.Status,
		Manager: item.Manager, Contact: item.Contact, Email: item.Email,
		LegalPerson: item.LegalRepresentative, CreditIdentifier: item.UnifiedSocialCreditIdentifier,
		Address: item.Address, Level: level, Remark: item.Remark, CreateBy: item.CreateBy,
		CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
	}
	return partnerRowFromDetail(detail)
}

func customerPartnerRow(item api.Customer) partnerRow {
	detail := partnerDetail{
		ID: item.ID, Category: "客户", Type: item.Type, Code: item.Code, Name: item.Name, Status: item.Status,
		Manager: item.Manager, Contact: item.Contact, Email: item.Email,
		LegalPerson: item.LegalRepresentative, CreditIdentifier: item.UnifiedSocialCreditIdentifier,
		Address: item.Address, Receivable: fmt.Sprintf("%.2f", item.ReceivableBalance),
		CreditBalance: fmt.Sprintf("%.2f", item.CreditBalance), Remark: item.Remark,
		CreateBy: item.CreateBy, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
	}
	return partnerRowFromDetail(detail)
}

func carrierPartnerRow(item api.Carrier) partnerRow {
	detail := partnerDetail{
		ID: item.ID, Category: "承运商", Type: item.Type, Code: item.Code, Name: item.Name, Status: item.Status,
		Manager: item.Manager, Contact: item.Contact, Email: item.Email,
		LegalPerson: item.LegalRepresentative, CreditIdentifier: item.UnifiedSocialCreditIdentifier,
		Address: item.Address, Remark: item.Remark, CreateBy: item.CreateBy,
		CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
	}
	return partnerRowFromDetail(detail)
}

func partnerRowFromDetail(detail partnerDetail) partnerRow {
	return partnerRow{
		Category: detail.Category, Type: detail.Type, Code: detail.Code, Name: detail.Name,
		Status: detail.Status, Manager: detail.Manager, Contact: detail.Contact,
		Email: detail.Email, Address: detail.Address, Detail: detail,
	}
}

func (ui *mainUI) showSelectedPartner() {
	state := ui.partner
	if state == nil || state.table == nil {
		return
	}
	index := state.table.CurrentIndex()
	if index < 0 || index >= len(state.rows) {
		walk.MsgBox(ui.window, "请选择合作伙伴", "请先在列表中选择一条合作伙伴资料。", walk.MsgBoxIconInformation)
		return
	}
	showPartnerDetail(ui.window, ui.session.Client, state.rows[index].Detail)
}

func showPartnerDetail(owner walk.Form, client *api.Client, detail partnerDetail) {
	var dlg *walk.Dialog
	var transactionButton *walk.PushButton
	var closeButton *walk.PushButton
	err := Dialog{
		AssignTo:      &dlg,
		Title:         detail.Category + "详情 - " + detail.Name,
		DefaultButton: &closeButton,
		MinSize:       Size{Width: 620, Height: 500},
		Size:          Size{Width: 720, Height: 620},
		Layout:        VBox{Margins: Margins{Left: 18, Top: 18, Right: 18, Bottom: 16}, Spacing: 10},
		Children: []Widget{
			Label{
				Text: detail.Name, Font: Font{Family: "Microsoft YaHei UI", PointSize: 15, Bold: true},
				EllipsisMode: EllipsisEnd, ToolTipText: detail.Name,
			},
			Label{Text: detail.Category + " · " + displayMaterialValue(detail.Status), TextColor: walk.RGB(70, 70, 70)},
			GroupBox{
				Title:         "只读资料",
				StretchFactor: 1,
				Layout:        Grid{Columns: 2, Margins: Margins{Left: 14, Top: 14, Right: 14, Bottom: 14}, Spacing: 9},
				Children:      partnerDetailWidgets(detail),
			},
			Composite{Layout: HBox{}, Children: []Widget{
				Label{Text: "资料来自当前线上接口。", TextColor: walk.RGB(90, 90, 90)},
				HSpacer{},
				PushButton{
					AssignTo: &transactionButton, Text: "交易流水", Visible: detail.Category == "客户",
					Enabled: detail.Category == "客户" && strings.TrimSpace(detail.ID) != "", MinSize: Size{Width: 92, Height: 30},
					Accessibility: Accessibility{Name: "查看客户交易流水"},
					OnClicked: func() {
						ShowCustomerTransactions(dlg, client, config.ImageBaseURL(), detail.ID, detail.Name)
					},
				},
				PushButton{
					AssignTo: &closeButton, Text: "关闭", MinSize: Size{Width: 88, Height: 30},
					OnClicked: func() { dlg.Accept() },
				},
			}},
		},
	}.Create(owner)
	if err != nil {
		walk.MsgBox(owner, "详情窗口错误", err.Error(), walk.MsgBoxIconError)
		return
	}
	dlg.Run()
}

func partnerDetailWidgets(detail partnerDetail) []Widget {
	fields := []struct {
		label string
		value string
	}{
		{"伙伴类别", detail.Category},
		{"主体类型", detail.Type},
		{"编号", detail.Code},
		{"状态", detail.Status},
		{"负责人", detail.Manager},
		{"联系电话", detail.Contact},
		{"邮箱", detail.Email},
		{"法定代表人", detail.LegalPerson},
		{"统一社会信用代码", detail.CreditIdentifier},
		{"地址", detail.Address},
	}
	if strings.TrimSpace(detail.Level) != "" {
		fields = append(fields, struct {
			label string
			value string
		}{"供应商等级", detail.Level})
	}
	if detail.Category == "客户" {
		fields = append(fields,
			struct {
				label string
				value string
			}{"应收账款", detail.Receivable},
			struct {
				label string
				value string
			}{"贷项余额", detail.CreditBalance},
		)
	}
	fields = append(fields,
		struct {
			label string
			value string
		}{"备注", detail.Remark},
		struct {
			label string
			value string
		}{"创建人", detail.CreateBy},
		struct {
			label string
			value string
		}{"创建时间", formatPartnerTime(detail.CreatedAt)},
		struct {
			label string
			value string
		}{"更新时间", formatPartnerTime(detail.UpdatedAt)},
	)

	widgets := make([]Widget, 0, len(fields)*2)
	for _, field := range fields {
		value := displayMaterialValue(field.value)
		widgets = append(widgets,
			Label{Text: field.label, Font: Font{Family: "Microsoft YaHei UI", Bold: true}, Alignment: AlignHNearVNear},
			TextLabel{
				Text: value, TextAlignment: AlignHNearVNear, ToolTipText: value,
				Accessibility: Accessibility{Name: field.label + "：" + value},
			},
		)
	}
	return widgets
}

func formatPartnerTime(value int64) string {
	if value <= 0 {
		return ""
	}
	return time.Unix(value, 0).Format("2006-01-02 15:04")
}
