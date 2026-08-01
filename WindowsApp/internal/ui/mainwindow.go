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
	"zhengshi-wms-windowsapp/internal/securestore"
)

type materialRow struct {
	Category      string
	Name          string
	Model         string
	HasDrawing    string
	Material      string
	Specification string
	Surface       string
	Strength      string
	SafeQuantity  string
	Detail        api.Material
}

type inventoryRow struct {
	Type        string
	ReceiptCode string
	ReceiveCode string
	Warehouse   string
	Location    string
	Name        string
	Model       string
	Quantity    string
	Available   string
	Locked      string
	Frozen      string
}

type inboundRow struct {
	Code         string
	Type         string
	Status       string
	BusinessName string
	PlanDate     string
	MaterialInfo string
	Amount       string
	Remark       string
}

type selectOption struct {
	ID    string
	Label string
}

type mainUI struct {
	session *Session
	cfg     config.Config
	window  *walk.MainWindow
	status  *walk.Label

	sideMenu          *walk.ListBox
	tabs              *walk.TabWidget
	closeTabButton    *walk.PushButton
	menuKeys          []string
	syncingMenu       bool
	materialTab       *walk.TabPage
	inventoryTab      *walk.TabPage
	inboundTab        *walk.TabPage
	outboundTab       *walk.TabPage
	outboundReportTab *walk.TabPage
	partnerTab        *walk.TabPage
	warehouseTab      *walk.TabPage
	systemTab         *walk.TabPage
	outbound          *outboundUI
	outboundReport    *outboundReportUI
	partner           *partnerUI
	warehouse         *warehouseUI
	categoryReady     bool
	categoryFailed    bool
	warehouseReady    bool
	warehouseFailed   bool
	supplierReady     bool
	supplierFailed    bool
	customerReady     bool
	customerFailed    bool

	materialCategory         *walk.ComboBox
	materialName             *walk.LineEdit
	materialModel            *walk.LineEdit
	materialSpecification    *walk.LineEdit
	materialMaterial         *walk.LineEdit
	materialSurfaceTreatment *walk.LineEdit
	materialStrengthGrade    *walk.LineEdit
	materialTable            *walk.TableView
	materialInfo             *walk.Label
	materialQuery            *walk.PushButton
	materialReset            *walk.PushButton
	materialPrev             *walk.PushButton
	materialNext             *walk.PushButton
	materialSize             *walk.ComboBox
	materialCategoryOptions  []selectOption
	materialRows             []materialRow
	materialPage             int
	materialTotal            int64
	materialGeneration       int
	materialCancel           context.CancelFunc

	inventoryMode           *walk.ComboBox
	inventoryType           *walk.ComboBox
	inventoryName           *walk.LineEdit
	inventoryModel          *walk.LineEdit
	inventoryWarehouse      *walk.ComboBox
	inventoryZone           *walk.ComboBox
	inventoryRack           *walk.ComboBox
	inventoryBin            *walk.ComboBox
	inventoryTable          *walk.TableView
	inventoryInfo           *walk.Label
	inventoryQuery          *walk.PushButton
	inventoryReset          *walk.PushButton
	inventoryPrev           *walk.PushButton
	inventoryNext           *walk.PushButton
	inventorySize           *walk.ComboBox
	inventoryWarehouseNodes []api.WarehouseNode
	inventoryZoneNodes      []api.WarehouseNode
	inventoryRackNodes      []api.WarehouseNode
	inventoryBinNodes       []api.WarehouseNode
	inventoryPage           int
	inventoryTotal          int64
	inventoryGeneration     int
	inventoryCancel         context.CancelFunc

	inboundSearch          *walk.LineEdit
	inboundStatus          *walk.ComboBox
	inboundType            *walk.ComboBox
	inboundSupplier        *walk.ComboBox
	inboundCustomer        *walk.ComboBox
	inboundTable           *walk.TableView
	inboundInfo            *walk.Label
	inboundQuery           *walk.PushButton
	inboundReset           *walk.PushButton
	inboundPrev            *walk.PushButton
	inboundNext            *walk.PushButton
	inboundReceive         *walk.PushButton
	inboundSize            *walk.ComboBox
	inboundSupplierOptions []selectOption
	inboundCustomerOptions []selectOption
	inboundPage            int
	inboundRows            []api.InboundReceipt
	inboundGeneration      int
	inboundCancel          context.CancelFunc
}

var pageSizes = []int{10, 20, 50, 100}

var pageSizeLabels = []string{"10 条/页", "20 条/页", "50 条/页", "100 条/页"}

var clientVersion = "0.6.0"
var buildTime = "development"
var gitCommit = "unknown"

type MainResult struct {
	LoggedOut bool
}

func RunMain(session *Session, cfg config.Config) (MainResult, error) {
	var result MainResult
	ui := &mainUI{
		session: session, cfg: cfg, materialPage: 1, inventoryPage: 1, inboundPage: 1,
		outbound:       newOutboundUI(),
		outboundReport: newOutboundReportUI(),
		partner:        newPartnerUI(availablePartnerKinds(session.Perms)),
		warehouse:      newWarehouseUI(availableWarehouseKinds(session.Perms)),
	}
	title := fmt.Sprintf("正时 WMS · %s", session.Profile.Name)
	pages := make([]TabPage, 0, 8)
	menuLabels := make([]string, 0, 9)
	if hasMenuPath(session.Perms.Menus, "/material/list") {
		pages = append(pages, ui.materialPageWidget())
		menuLabels = append(menuLabels, "物料查询")
		ui.menuKeys = append(ui.menuKeys, "material")
	}
	if hasAnyMenuPath(session.Perms.Menus, "/inventory/index", "/inventory/record") {
		pages = append(pages, ui.inventoryPageWidget())
		menuLabels = append(menuLabels, "库存查询")
		ui.menuKeys = append(ui.menuKeys, "inventory")
	}
	if hasMenuPath(session.Perms.Menus, "/inbound/receipt") {
		pages = append(pages, ui.inboundPageWidget())
		menuLabels = append(menuLabels, "入库工作台")
		ui.menuKeys = append(ui.menuKeys, "inbound")
	}
	if hasAnyMenuPath(session.Perms.Menus, "/outbound/receipt", "/outbound/receipt2") {
		pages = append(pages, ui.outboundPageWidget())
		menuLabels = append(menuLabels, "出库执行")
		ui.menuKeys = append(ui.menuKeys, "outbound")
	}
	if hasMenuPath(session.Perms.Menus, "/outbound/report") {
		pages = append(pages, ui.outboundReportPageWidget())
		menuLabels = append(menuLabels, "出库报表")
		ui.menuKeys = append(ui.menuKeys, "outbound_report")
	}
	if len(ui.partner.kinds) > 0 {
		pages = append(pages, ui.partnerPageWidget())
		menuLabels = append(menuLabels, "合作伙伴")
		ui.menuKeys = append(ui.menuKeys, "partner")
	}
	if len(ui.warehouse.kinds) > 0 {
		pages = append(pages, ui.warehousePageWidget())
		menuLabels = append(menuLabels, "仓储结构")
		ui.menuKeys = append(ui.menuKeys, "warehouse")
	}
	pages = append(pages, TabPage{
		AssignTo: &ui.systemTab,
		Title:    "系统信息",
		Layout:   VBox{Margins: Margins{Left: 24, Top: 24, Right: 24, Bottom: 24}, Spacing: 10},
		Children: []Widget{
			Label{Text: "系统信息", Font: Font{Family: "Microsoft YaHei UI", PointSize: 15, Bold: true}},
			Label{Text: "Windows 原生仓库执行客户端，当前直接连接线上 API。", TextColor: walk.RGB(85, 85, 85)},
			HSpacer{Size: 8},
			Label{Text: fmt.Sprintf("当前账号：%s（%s）", session.Profile.Name, session.Profile.DepartmentName)},
			Label{Text: "服务地址：" + cfg.APIBaseURL},
			Label{Text: fmt.Sprintf("已加载 %d 个顶级权限菜单。", len(session.Perms.Menus))},
			Label{Text: "客户端按账号权限创建功能入口；所有入库和出库操作仍由服务端权限及状态规则最终校验。"},
			Label{Text: fmt.Sprintf("客户端版本：v%s    构建时间：%s    提交：%s", clientVersion, buildTime, gitCommit)},
			VSpacer{},
		},
	})
	menuLabels = append(menuLabels, "系统信息")
	ui.menuKeys = append(ui.menuKeys, "system")
	err := MainWindow{
		AssignTo: &ui.window,
		Title:    title,
		MinSize:  Size{Width: 1280, Height: 720},
		Size:     Size{Width: 1480, Height: 880},
		Font:     Font{Family: "Microsoft YaHei UI", PointSize: 9},
		Layout:   VBox{Margins: Margins{Left: 16, Top: 12, Right: 16, Bottom: 10}, Spacing: 10},
		MenuItems: []MenuItem{
			Menu{
				Text: "工作区",
				Items: []MenuItem{
					Action{
						Text: "刷新当前页", Shortcut: Shortcut{Key: walk.KeyF5},
						OnTriggered: ui.refreshCurrentPage,
					},
					Action{
						Text: "定位到筛选条件", Shortcut: Shortcut{Modifiers: walk.ModControl, Key: walk.KeyF},
						OnTriggered: ui.focusCurrentPageSearch,
					},
					Separator{},
					Action{
						Text: "关闭当前标签", Shortcut: Shortcut{Modifiers: walk.ModControl, Key: walk.KeyW},
						OnTriggered: ui.closeCurrentTab,
					},
				},
			},
		},
		Children: []Widget{
			Composite{
				Layout: HBox{Spacing: 10},
				Children: []Widget{
					Label{Text: "正时 WMS", Font: Font{Family: "Microsoft YaHei UI", PointSize: 16, Bold: true}},
					Label{Text: "线上生产环境", Font: Font{Family: "Microsoft YaHei UI", PointSize: 9, Bold: true}, TextColor: walk.RGB(183, 45, 33), ToolTipText: "当前所有业务操作直接访问生产数据"},
					Label{Text: cfg.APIBaseURL, TextColor: walk.RGB(95, 95, 95)},
					HSpacer{},
					Label{Text: session.Profile.Name, Font: Font{Family: "Microsoft YaHei UI", PointSize: 9, Bold: true}},
					Label{Text: session.Profile.DepartmentName, TextColor: walk.RGB(95, 95, 95)},
					PushButton{Text: "退出登录", OnClicked: func() {
						result.LoggedOut = true
						_ = securestore.Delete()
						go session.Client.Logout(context.Background())
						ui.window.Close()
					}},
				},
			},
			HSplitter{
				HandleWidth:   4,
				StretchFactor: 1,
				Children: []Widget{
					Composite{
						MinSize:       Size{Width: 200},
						MaxSize:       Size{Width: 224},
						StretchFactor: 2,
						Layout:        VBox{Margins: Margins{Left: 12, Top: 12, Right: 12, Bottom: 12}, Spacing: 8},
						Children: []Widget{
							Label{Text: "功能菜单", Font: Font{Family: "Microsoft YaHei UI", PointSize: 12, Bold: true}},
							Label{Text: "单击打开或切换工作页", TextColor: walk.RGB(90, 90, 90)},
							ListBox{
								AssignTo:              &ui.sideMenu,
								Model:                 menuLabels,
								MinSize:               Size{Width: 176, Height: 240},
								StretchFactor:         1,
								OnCurrentIndexChanged: ui.openSelectedMenu,
								OnItemActivated:       ui.openSelectedMenu,
							},
							Label{Text: "关闭的页面可从菜单重新打开。", TextColor: walk.RGB(100, 100, 100)},
						},
					},
					Composite{
						StretchFactor: 12,
						Layout:        VBox{Margins: Margins{Left: 8, Top: 6, Right: 2, Bottom: 2}, Spacing: 6},
						Children: []Widget{
							Composite{
								Layout: HBox{Spacing: 8},
								Children: []Widget{
									Label{Text: "工作区", Font: Font{Family: "Microsoft YaHei UI", PointSize: 10, Bold: true}},
									Label{Text: "业务页面以标签方式打开", TextColor: walk.RGB(90, 90, 90)},
									HSpacer{},
									PushButton{
										AssignTo:    &ui.closeTabButton,
										Text:        "关闭当前标签",
										MinSize:     Size{Width: 110, Height: 30},
										ToolTipText: "关闭当前业务标签；可从左侧菜单重新打开",
										OnClicked:   ui.closeCurrentTab,
									},
								},
							},
							TabWidget{
								AssignTo:              &ui.tabs,
								Pages:                 pages,
								StretchFactor:         1,
								OnCurrentIndexChanged: ui.syncNavigationFromTab,
							},
						},
					},
				},
			},
			Label{AssignTo: &ui.status, Text: "已连接线上 API · 正在加载筛选选项…", TextColor: walk.RGB(85, 85, 85)},
		},
	}.Create()
	if err != nil {
		return result, err
	}
	if err := ui.installTabCloseHandler(); err != nil {
		ui.window.Dispose()
		return result, err
	}
	ui.window.Disposing().Attach(ui.disposeDetachedTabs)
	ui.syncNavigationFromTab()
	ui.initializePage("material")
	ui.initializePage("inventory")
	ui.initializePage("inbound")
	ui.initializePage("outbound")
	ui.initializePage("outbound_report")
	ui.initializePage("partner")
	ui.initializePage("warehouse")
	ui.loadFilterOptions()
	ui.window.Run()
	return result, nil
}

func (ui *mainUI) openSelectedMenu() {
	if ui.syncingMenu || ui.sideMenu == nil {
		return
	}
	index := ui.sideMenu.CurrentIndex()
	if index < 0 || index >= len(ui.menuKeys) {
		return
	}
	ui.openTab(ui.menuKeys[index])
}

func (ui *mainUI) openTab(key string) {
	if ui.tabs == nil {
		return
	}
	page := ui.tabForKey(key)
	created := false
	if page == nil {
		var err error
		page, err = ui.createTab(key)
		if err != nil {
			walk.MsgBox(ui.window, "无法打开页面", err.Error(), walk.MsgBoxIconError)
			return
		}
		created = true
	} else if ui.tabs.Pages().Index(page) < 0 {
		if err := ui.tabs.Pages().Insert(ui.tabInsertionIndex(key), page); err != nil {
			walk.MsgBox(ui.window, "无法恢复页面", err.Error(), walk.MsgBoxIconError)
			return
		}
		created = true
	}
	targetIndex := ui.tabs.Pages().Index(page)
	if created && ui.tabs.CurrentIndex() == targetIndex && ui.tabs.Pages().Len() > 1 {
		otherIndex := targetIndex + 1
		if otherIndex >= ui.tabs.Pages().Len() {
			otherIndex = targetIndex - 1
		}
		_ = ui.tabs.SetCurrentIndex(otherIndex)
	}
	if err := ui.tabs.SetCurrentIndex(targetIndex); err != nil {
		walk.MsgBox(ui.window, "无法切换页面", err.Error(), walk.MsgBoxIconError)
	}
}

func (ui *mainUI) closeCurrentTab() {
	if ui.tabs == nil {
		return
	}
	ui.closeTabAt(ui.tabs.CurrentIndex())
}

func (ui *mainUI) closeTabAt(index int) {
	if ui.tabs == nil {
		return
	}
	if index < 0 || index >= ui.tabs.Pages().Len() {
		return
	}
	page := ui.tabs.Pages().At(index)
	if page == ui.systemTab {
		return
	}
	if err := ui.tabs.Pages().RemoveAt(index); err != nil {
		walk.MsgBox(ui.window, "无法关闭页面", err.Error(), walk.MsgBoxIconError)
		return
	}
	ui.syncNavigationFromTab()
}

func (ui *mainUI) disposeDetachedTabs() {
	if ui.tabs == nil {
		return
	}
	for _, page := range []*walk.TabPage{
		ui.materialTab, ui.inventoryTab, ui.inboundTab, ui.outboundTab,
		ui.outboundReportTab, ui.partnerTab, ui.warehouseTab,
	} {
		if page != nil && ui.tabs.Pages().Index(page) < 0 {
			page.Dispose()
		}
	}
}

func (ui *mainUI) createTab(key string) (*walk.TabPage, error) {
	var pageDecl TabPage
	switch key {
	case "material":
		pageDecl = ui.materialPageWidget()
	case "inventory":
		pageDecl = ui.inventoryPageWidget()
	case "inbound":
		pageDecl = ui.inboundPageWidget()
	case "outbound":
		pageDecl = ui.outboundPageWidget()
	case "outbound_report":
		pageDecl = ui.outboundReportPageWidget()
	case "partner":
		pageDecl = ui.partnerPageWidget()
	case "warehouse":
		pageDecl = ui.warehousePageWidget()
	default:
		return nil, fmt.Errorf("未知工作页：%s", key)
	}
	if err := pageDecl.Create(NewBuilder(nil)); err != nil {
		return nil, err
	}
	page := ui.tabForKey(key)
	if page == nil {
		return nil, fmt.Errorf("工作页创建失败：%s", key)
	}
	if err := ui.tabs.Pages().Insert(ui.tabInsertionIndex(key), page); err != nil {
		page.Dispose()
		ui.releasePage(key)
		return nil, err
	}
	ui.initializePage(key)
	return page, nil
}

func (ui *mainUI) tabInsertionIndex(key string) int {
	targetOrder := stringIndex(ui.menuKeys, key)
	if targetOrder < 0 || ui.tabs == nil {
		return 0
	}
	index := 0
	for pageIndex := 0; pageIndex < ui.tabs.Pages().Len(); pageIndex++ {
		pageKey := ui.keyForTab(ui.tabs.Pages().At(pageIndex))
		if stringIndex(ui.menuKeys, pageKey) < targetOrder {
			index++
		}
	}
	return index
}

func (ui *mainUI) syncNavigationFromTab() {
	if ui.tabs == nil || ui.sideMenu == nil || ui.closeTabButton == nil {
		return
	}
	index := ui.tabs.CurrentIndex()
	if index < 0 || index >= ui.tabs.Pages().Len() {
		ui.closeTabButton.SetEnabled(false)
		return
	}
	page := ui.tabs.Pages().At(index)
	key := ui.keyForTab(page)
	ui.closeTabButton.SetEnabled(page != ui.systemTab)

	menuIndex := stringIndex(ui.menuKeys, key)
	if menuIndex < 0 || ui.sideMenu.CurrentIndex() == menuIndex {
		return
	}
	ui.syncingMenu = true
	_ = ui.sideMenu.SetCurrentIndex(menuIndex)
	ui.syncingMenu = false
}

func (ui *mainUI) tabForKey(key string) *walk.TabPage {
	switch key {
	case "material":
		return ui.materialTab
	case "inventory":
		return ui.inventoryTab
	case "inbound":
		return ui.inboundTab
	case "outbound":
		return ui.outboundTab
	case "outbound_report":
		return ui.outboundReportTab
	case "partner":
		return ui.partnerTab
	case "warehouse":
		return ui.warehouseTab
	case "system":
		return ui.systemTab
	default:
		return nil
	}
}

func (ui *mainUI) keyForTab(page *walk.TabPage) string {
	switch page {
	case ui.materialTab:
		return "material"
	case ui.inventoryTab:
		return "inventory"
	case ui.inboundTab:
		return "inbound"
	case ui.outboundTab:
		return "outbound"
	case ui.outboundReportTab:
		return "outbound_report"
	case ui.partnerTab:
		return "partner"
	case ui.warehouseTab:
		return "warehouse"
	case ui.systemTab:
		return "system"
	default:
		return ""
	}
}

func stringIndex(values []string, target string) int {
	for index, value := range values {
		if value == target {
			return index
		}
	}
	return -1
}

func (ui *mainUI) initializePage(key string) {
	switch key {
	case "material":
		if ui.materialTab == nil || ui.materialName == nil {
			return
		}
		ui.materialGeneration++
		ui.applyMaterialCategoryOptions()
		ui.materialName.KeyDown().Attach(func(key walk.Key) {
			if key == walk.KeyReturn {
				ui.materialPage = 1
				ui.loadMaterials()
			}
		})
		ui.loadMaterials()
	case "inventory":
		if ui.inventoryTab == nil || ui.inventoryName == nil {
			return
		}
		ui.inventoryGeneration++
		ui.applyWarehouseOptions()
		ui.inventoryName.KeyDown().Attach(func(key walk.Key) {
			if key == walk.KeyReturn {
				ui.inventoryPage = 1
				ui.loadInventory()
			}
		})
		ui.loadInventory()
	case "inbound":
		if ui.inboundTab == nil || ui.inboundSearch == nil {
			return
		}
		ui.inboundGeneration++
		ui.applySupplierOptions()
		ui.applyCustomerOptions()
		ui.inboundSearch.KeyDown().Attach(func(key walk.Key) {
			if key == walk.KeyReturn {
				ui.inboundPage = 1
				ui.loadInbound()
			}
		})
		ui.loadInbound()
	case "outbound":
		ui.initializeOutboundPage()
	case "outbound_report":
		ui.initializeOutboundReportPage()
	case "partner":
		ui.initializePartnerPage()
	case "warehouse":
		ui.initializeWarehousePage()
	}
}

func (ui *mainUI) releasePage(key string) {
	switch key {
	case "material":
		ui.materialGeneration++
		if ui.materialCancel != nil {
			ui.materialCancel()
		}
		ui.materialTab = nil
		ui.materialCategory = nil
		ui.materialName = nil
		ui.materialModel = nil
		ui.materialSpecification = nil
		ui.materialMaterial = nil
		ui.materialSurfaceTreatment = nil
		ui.materialStrengthGrade = nil
		ui.materialTable = nil
		ui.materialInfo = nil
		ui.materialQuery = nil
		ui.materialReset = nil
		ui.materialPrev = nil
		ui.materialNext = nil
		ui.materialSize = nil
		ui.materialRows = nil
		ui.materialPage = 1
		ui.materialTotal = 0
	case "inventory":
		ui.inventoryGeneration++
		if ui.inventoryCancel != nil {
			ui.inventoryCancel()
		}
		ui.inventoryTab = nil
		ui.inventoryMode = nil
		ui.inventoryType = nil
		ui.inventoryName = nil
		ui.inventoryModel = nil
		ui.inventoryWarehouse = nil
		ui.inventoryZone = nil
		ui.inventoryRack = nil
		ui.inventoryBin = nil
		ui.inventoryTable = nil
		ui.inventoryInfo = nil
		ui.inventoryQuery = nil
		ui.inventoryReset = nil
		ui.inventoryPrev = nil
		ui.inventoryNext = nil
		ui.inventorySize = nil
		ui.inventoryZoneNodes = nil
		ui.inventoryRackNodes = nil
		ui.inventoryBinNodes = nil
		ui.inventoryPage = 1
		ui.inventoryTotal = 0
	case "inbound":
		ui.inboundGeneration++
		if ui.inboundCancel != nil {
			ui.inboundCancel()
		}
		ui.inboundTab = nil
		ui.inboundSearch = nil
		ui.inboundStatus = nil
		ui.inboundType = nil
		ui.inboundSupplier = nil
		ui.inboundCustomer = nil
		ui.inboundTable = nil
		ui.inboundInfo = nil
		ui.inboundQuery = nil
		ui.inboundReset = nil
		ui.inboundPrev = nil
		ui.inboundNext = nil
		ui.inboundReceive = nil
		ui.inboundSize = nil
		ui.inboundPage = 1
		ui.inboundRows = nil
	case "outbound":
		ui.releaseOutboundPage()
	case "outbound_report":
		ui.releaseOutboundReportPage()
	case "partner":
		ui.releasePartnerPage()
	case "warehouse":
		ui.releaseWarehousePage()
	}
}

func (ui *mainUI) applyMaterialCategoryOptions() {
	if ui.materialCategory == nil || !ui.categoryReady {
		return
	}
	label := "全部分类"
	if ui.categoryFailed {
		label = "分类加载失败"
	}
	_ = ui.materialCategory.SetModel(optionLabels(label, ui.materialCategoryOptions))
	ui.materialCategory.SetCurrentIndex(0)
}

func (ui *mainUI) applyWarehouseOptions() {
	if ui.inventoryWarehouse == nil || !ui.warehouseReady {
		return
	}
	label := "全部仓库"
	if ui.warehouseFailed {
		label = "仓库加载失败"
	}
	_ = ui.inventoryWarehouse.SetModel(nodeLabels(label, ui.inventoryWarehouseNodes))
	ui.inventoryWarehouse.SetCurrentIndex(0)
	ui.updateInventoryZones()
}

func (ui *mainUI) applySupplierOptions() {
	if ui.inboundSupplier == nil || !ui.supplierReady {
		return
	}
	label := "全部供应商"
	if ui.supplierFailed {
		label = "供应商加载失败"
	}
	_ = ui.inboundSupplier.SetModel(optionLabels(label, ui.inboundSupplierOptions))
	ui.inboundSupplier.SetCurrentIndex(0)
}

func (ui *mainUI) applyCustomerOptions() {
	if ui.inboundCustomer == nil || !ui.customerReady {
		return
	}
	label := "全部客户"
	if ui.customerFailed {
		label = "客户加载失败"
	}
	_ = ui.inboundCustomer.SetModel(optionLabels(label, ui.inboundCustomerOptions))
	ui.inboundCustomer.SetCurrentIndex(0)
}

func (ui *mainUI) inboundPageWidget() TabPage {
	statuses := []string{"全部状态", "待审核", "审核不通过", "审核通过", "未发货", "在途", "部分入库", "作废", "入库完成"}
	types := []string{"全部类型", "采购入库", "外协入库", "生产入库", "退货入库"}
	return TabPage{
		AssignTo: &ui.inboundTab,
		Title:    closableTabTitle("入库工作台"),
		Layout:   VBox{Margins: Margins{Left: 14, Top: 14, Right: 14, Bottom: 12}, Spacing: 10},
		Children: []Widget{
			Label{Text: "入库工作台", Font: Font{Family: "Microsoft YaHei UI", PointSize: 15, Bold: true}},
			Label{Text: "按 Web 端单据筛选条件查询，选中入库单后可查看详情、收货记录或执行批次收货。", TextColor: walk.RGB(85, 85, 85)},
			GroupBox{
				Title:  "筛选条件",
				Layout: Grid{Columns: 8, Margins: Margins{Left: 14, Top: 12, Right: 14, Bottom: 14}, Spacing: 8},
				Children: []Widget{
					Label{Text: "入库单号"},
					LineEdit{AssignTo: &ui.inboundSearch, MinSize: Size{Width: 180, Height: 28}, ToolTipText: "支持按完整或部分单号查询"},
					Label{Text: "状态"},
					ComboBox{AssignTo: &ui.inboundStatus, Model: statuses, CurrentIndex: 0, MinSize: Size{Width: 130, Height: 28}},
					Label{Text: "类型"},
					ComboBox{AssignTo: &ui.inboundType, Model: types, CurrentIndex: 0, MinSize: Size{Width: 130, Height: 28}},
					Label{Text: "供应商"},
					ComboBox{AssignTo: &ui.inboundSupplier, Model: []string{"全部供应商（正在加载）"}, CurrentIndex: 0, MinSize: Size{Width: 190, Height: 28}},
					Label{Text: "客户"},
					ComboBox{
						AssignTo: &ui.inboundCustomer, Model: []string{"全部客户（正在加载）"}, CurrentIndex: 0,
						MinSize: Size{Width: 190, Height: 28}, ToolTipText: "退货入库按客户筛选；全部类型时也可单独使用",
					},
					HSpacer{ColumnSpan: 6},
					Composite{
						ColumnSpan: 8,
						Layout:     HBox{Spacing: 8},
						Children: []Widget{
							HSpacer{},
							PushButton{AssignTo: &ui.inboundReset, Text: "重置", MinSize: Size{Width: 88, Height: 30}, OnClicked: ui.resetInboundFilters},
							PushButton{AssignTo: &ui.inboundQuery, Text: "查询", MinSize: Size{Width: 96, Height: 30}, OnClicked: func() {
								ui.inboundPage = 1
								ui.loadInbound()
							}},
						},
					},
				},
			},
			TableView{
				AssignTo:         &ui.inboundTable,
				AlternatingRowBG: true,
				StretchFactor:    1,
				Columns: []TableViewColumn{
					{Title: "入库单号", DataMember: "Code", Width: 145},
					{Title: "类型", DataMember: "Type", Width: 90},
					{Title: "状态", DataMember: "Status", Width: 90},
					{Title: "供应商/客户", DataMember: "BusinessName", Width: 150},
					{Title: "计划日期", DataMember: "PlanDate", Width: 100},
					{Title: "物料进度", DataMember: "MaterialInfo", Width: 150},
					{Title: "金额", DataMember: "Amount", Width: 90},
					{Title: "备注", DataMember: "Remark", Width: 180},
				},
				OnItemActivated: ui.showInboundDetail,
			},
			Composite{Layout: HBox{}, Children: []Widget{
				Label{AssignTo: &ui.inboundInfo, Text: "尚未加载"},
				HSpacer{},
				PushButton{Text: "详情/收货记录", OnClicked: ui.showInboundDetail},
				PushButton{AssignTo: &ui.inboundReceive, Text: "批次收货", Enabled: hasButton(ui.session.Perms.Buttons, "inbound:receipt:receive"), OnClicked: ui.receiveSelectedInbound},
				Label{Text: "每页"},
				ComboBox{AssignTo: &ui.inboundSize, Model: pageSizeLabels, CurrentIndex: 1, MinSize: Size{Width: 92}, OnCurrentIndexChanged: func() {
					if ui.window != nil && ui.inboundSize != nil && ui.inboundSize.CurrentIndex() >= 0 {
						ui.inboundPage = 1
						ui.loadInbound()
					}
				}},
				PushButton{AssignTo: &ui.inboundPrev, Text: "上一页", OnClicked: func() {
					if ui.inboundPage > 1 {
						ui.inboundPage--
						ui.loadInbound()
					}
				}},
				PushButton{AssignTo: &ui.inboundNext, Text: "下一页", OnClicked: func() { ui.inboundPage++; ui.loadInbound() }},
			}},
		},
	}
}

func hasButton(buttons []api.Button, permission string) bool {
	for _, button := range buttons {
		if button.Perms == permission {
			return true
		}
	}
	return false
}

func (ui *mainUI) selectedInbound() (api.InboundReceipt, bool) {
	index := ui.inboundTable.CurrentIndex()
	if index < 0 || index >= len(ui.inboundRows) {
		walk.MsgBox(ui.window, "请选择入库单", "请先在列表中选择一张入库单。", walk.MsgBoxIconInformation)
		return api.InboundReceipt{}, false
	}
	return ui.inboundRows[index], true
}

func (ui *mainUI) loadInbound() {
	if ui.inboundCancel != nil {
		ui.inboundCancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	ui.inboundCancel = cancel
	page := ui.inboundPage
	size := selectedPageSize(ui.inboundSize)
	generation := ui.inboundGeneration
	filters := api.InboundFilters{Code: ui.inboundSearch.Text()}
	if ui.inboundStatus.CurrentIndex() > 0 {
		filters.Status = ui.inboundStatus.Text()
	}
	if ui.inboundType.CurrentIndex() > 0 {
		filters.Type = ui.inboundType.Text()
	}
	filters.SupplierID = selectedOptionID(ui.inboundSupplier, ui.inboundSupplierOptions)
	filters.CustomerID = selectedOptionID(ui.inboundCustomer, ui.inboundCustomerOptions)
	ui.inboundInfo.SetText("正在加载线上入库单……")
	ui.inboundPrev.SetEnabled(false)
	ui.inboundNext.SetEnabled(false)
	ui.inboundQuery.SetEnabled(false)
	ui.inboundReset.SetEnabled(false)
	go func() {
		result, err := ui.session.Client.InboundReceipts(ctx, page, size, filters)
		if ctx.Err() != nil {
			return
		}
		ui.window.Synchronize(func() {
			if generation != ui.inboundGeneration || ui.inboundTable == nil {
				return
			}
			ui.inboundQuery.SetEnabled(true)
			ui.inboundReset.SetEnabled(true)
			if err != nil {
				ui.inboundInfo.SetText("加载失败：" + err.Error())
				return
			}
			rows := make([]inboundRow, 0, len(result.List))
			for _, receipt := range result.List {
				estimated, actual := 0.0, 0.0
				for _, material := range receipt.Materials {
					estimated += material.EstimatedQuantity
					actual += material.ActualQuantity
				}
				business := receipt.SupplierName
				if business == "" {
					business = receipt.CustomerName
				}
				planDate := ""
				if receipt.ReceivingDate > 0 {
					planDate = time.Unix(receipt.ReceivingDate, 0).Format("2006-01-02")
				}
				rows = append(rows, inboundRow{
					Code: receipt.Code, Type: receipt.Type, Status: receipt.Status,
					BusinessName: business, PlanDate: planDate,
					MaterialInfo: fmt.Sprintf("%g / %g", actual, estimated),
					Amount:       fmt.Sprintf("%.2f", receipt.TotalAmount), Remark: receipt.Remark,
				})
			}
			if modelErr := ui.inboundTable.SetModel(rows); modelErr != nil {
				ui.inboundInfo.SetText("表格数据错误：" + modelErr.Error())
				return
			}
			ui.inboundRows = result.List
			ui.inboundInfo.SetText(fmt.Sprintf("第 %d 页 | 本页 %d 条 | 共 %d 条", page, len(rows), result.Total))
			ui.inboundPrev.SetEnabled(page > 1)
			ui.inboundNext.SetEnabled(int64(page*size) < result.Total)
		})
	}()
}

func (ui *mainUI) resetInboundFilters() {
	ui.inboundSearch.SetText("")
	ui.inboundStatus.SetCurrentIndex(0)
	ui.inboundType.SetCurrentIndex(0)
	ui.inboundSupplier.SetCurrentIndex(0)
	ui.inboundCustomer.SetCurrentIndex(0)
	ui.inboundPage = 1
	ui.loadInbound()
}

func (ui *mainUI) showInboundDetail() {
	receipt, ok := ui.selectedInbound()
	if !ok {
		return
	}
	ShowInboundDetail(ui.window, ui.session.Client, config.ImageBaseURL(), receipt)
}

func (ui *mainUI) receiveSelectedInbound() {
	receipt, ok := ui.selectedInbound()
	if !ok {
		return
	}
	if receipt.Status == "待审核" || receipt.Status == "审核不通过" || receipt.Status == "作废" || receipt.Status == "入库完成" {
		walk.MsgBox(ui.window, "当前状态不可收货", fmt.Sprintf("入库单状态为“%s”，服务端不会允许批次收货。", receipt.Status), walk.MsgBoxIconWarning)
		return
	}
	if ReceiveInbound(ui.window, ui.session.Client, receipt) {
		ui.loadInbound()
	}
}

func hasMenu(menus []api.Menu, namePart, pathPart string) bool {
	for _, menu := range menus {
		if strings.Contains(menu.Name, namePart) || strings.Contains(menu.Path, pathPart) {
			return true
		}
		if hasMenu(menu.Children, namePart, pathPart) {
			return true
		}
	}
	return false
}

func hasMenuPath(menus []api.Menu, path string) bool {
	path = normalizeMenuPath(path)
	for _, menu := range menus {
		if normalizeMenuPath(menu.Path) == path || hasMenuPath(menu.Children, path) {
			return true
		}
	}
	return false
}

func hasAnyMenuPath(menus []api.Menu, paths ...string) bool {
	for _, path := range paths {
		if hasMenuPath(menus, path) {
			return true
		}
	}
	return false
}

func normalizeMenuPath(path string) string {
	path = strings.TrimSpace(path)
	if len(path) > 1 {
		path = strings.TrimRight(path, "/")
	}
	return strings.ToLower(path)
}

func (ui *mainUI) materialPageWidget() TabPage {
	return TabPage{
		AssignTo: &ui.materialTab,
		Title:    closableTabTitle("物料查询"),
		Layout:   VBox{Margins: Margins{Left: 14, Top: 14, Right: 14, Bottom: 12}, Spacing: 10},
		Children: []Widget{
			Label{Text: "物料查询", Font: Font{Family: "Microsoft YaHei UI", PointSize: 15, Bold: true}},
			Label{Text: "筛选条件与 Web 端物料列表保持一致，可组合查询分类及各项物料属性。", TextColor: walk.RGB(85, 85, 85)},
			GroupBox{
				Title:  "筛选条件",
				Layout: Grid{Columns: 8, Margins: Margins{Left: 14, Top: 12, Right: 14, Bottom: 14}, Spacing: 8},
				Children: []Widget{
					Label{Text: "物料分类"},
					ComboBox{AssignTo: &ui.materialCategory, Model: []string{"全部分类（正在加载）"}, CurrentIndex: 0, MinSize: Size{Width: 180, Height: 28}},
					Label{Text: "物料名称"},
					LineEdit{AssignTo: &ui.materialName, MinSize: Size{Width: 180, Height: 28}},
					Label{Text: "型号"},
					LineEdit{AssignTo: &ui.materialModel, MinSize: Size{Width: 180, Height: 28}},
					Label{Text: "规格"},
					LineEdit{AssignTo: &ui.materialSpecification, MinSize: Size{Width: 180, Height: 28}},
					Label{Text: "材质"},
					LineEdit{AssignTo: &ui.materialMaterial, MinSize: Size{Width: 180, Height: 28}},
					Label{Text: "表面处理"},
					LineEdit{AssignTo: &ui.materialSurfaceTreatment, MinSize: Size{Width: 180, Height: 28}},
					Label{Text: "强度等级"},
					LineEdit{AssignTo: &ui.materialStrengthGrade, MinSize: Size{Width: 180, Height: 28}},
					HSpacer{ColumnSpan: 2},
					Composite{
						ColumnSpan: 8,
						Layout:     HBox{Spacing: 8},
						Children: []Widget{
							HSpacer{},
							PushButton{AssignTo: &ui.materialReset, Text: "重置", MinSize: Size{Width: 88, Height: 30}, OnClicked: ui.resetMaterialFilters},
							PushButton{AssignTo: &ui.materialQuery, Text: "查询", MinSize: Size{Width: 96, Height: 30}, OnClicked: func() {
								ui.materialPage = 1
								ui.loadMaterials()
							}},
						},
					},
				},
			},
			Label{
				Text:          "提示：双击物料行，或选中后按 Enter，查看图纸与物料信息。",
				TextColor:     walk.RGB(75, 75, 75),
				Accessibility: Accessibility{Name: "物料详情操作提示"},
			},
			TableView{
				AssignTo:         &ui.materialTable,
				AlternatingRowBG: true,
				ColumnsOrderable: true,
				StretchFactor:    1,
				ToolTipText:      "双击物料行，或选中后按 Enter，查看图纸与物料信息",
				Accessibility:    Accessibility{Name: "物料查询结果表格", Description: "包含图纸状态；激活当前行可查看详情"},
				OnItemActivated:  ui.showMaterialDetail,
				Columns: []TableViewColumn{
					{Title: "分类", DataMember: "Category", Width: 100},
					{Title: "名称", DataMember: "Name", Width: 190},
					{Title: "型号", DataMember: "Model", Width: 130},
					{Title: "图纸", DataMember: "HasDrawing", Width: 56, Alignment: AlignCenter},
					{Title: "材质", DataMember: "Material", Width: 100},
					{Title: "规格", DataMember: "Specification", Width: 140},
					{Title: "表面处理", DataMember: "Surface", Width: 100},
					{Title: "强度", DataMember: "Strength", Width: 90},
					{Title: "安全库存", DataMember: "SafeQuantity", Width: 100},
				},
			},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					Label{AssignTo: &ui.materialInfo, Text: "尚未加载"},
					HSpacer{},
					Label{Text: "每页"},
					ComboBox{AssignTo: &ui.materialSize, Model: pageSizeLabels, CurrentIndex: 1, MinSize: Size{Width: 92}, OnCurrentIndexChanged: func() {
						if ui.window != nil && ui.materialSize != nil && ui.materialSize.CurrentIndex() >= 0 {
							ui.materialPage = 1
							ui.loadMaterials()
						}
					}},
					PushButton{AssignTo: &ui.materialPrev, Text: "上一页", OnClicked: func() {
						if ui.materialPage > 1 {
							ui.materialPage--
							ui.loadMaterials()
						}
					}},
					PushButton{AssignTo: &ui.materialNext, Text: "下一页", OnClicked: func() { ui.materialPage++; ui.loadMaterials() }},
				},
			},
		},
	}
}

func (ui *mainUI) resetMaterialFilters() {
	ui.materialCategory.SetCurrentIndex(0)
	ui.materialName.SetText("")
	ui.materialModel.SetText("")
	ui.materialSpecification.SetText("")
	ui.materialMaterial.SetText("")
	ui.materialSurfaceTreatment.SetText("")
	ui.materialStrengthGrade.SetText("")
	ui.materialPage = 1
	ui.loadMaterials()
}

func (ui *mainUI) showMaterialDetail() {
	if ui.materialTable == nil {
		return
	}
	index := ui.materialTable.CurrentIndex()
	if index < 0 || index >= len(ui.materialRows) {
		return
	}
	ShowMaterialDetail(ui.window, ui.session.Client, config.ImageBaseURL(), ui.materialRows[index].Detail)
}

func (ui *mainUI) inventoryPageWidget() TabPage {
	types := []string{"全部类型", "采购入库", "外协入库", "生产入库", "退货入库"}
	modes := []string{"当前库存", "库存批次历史"}
	return TabPage{
		AssignTo: &ui.inventoryTab,
		Title:    closableTabTitle("库存查询"),
		Layout:   VBox{Margins: Margins{Left: 14, Top: 14, Right: 14, Bottom: 12}, Spacing: 10},
		Children: []Widget{
			Label{Text: "库存查询", Font: Font{Family: "Microsoft YaHei UI", PointSize: 15, Bold: true}},
			Label{Text: "可切换当前库存与库存批次历史；后者展示已有库存批次，不表示独立库存流水。", TextColor: walk.RGB(85, 85, 85)},
			GroupBox{
				Title:  "筛选条件",
				Layout: Grid{Columns: 8, Margins: Margins{Left: 14, Top: 12, Right: 14, Bottom: 14}, Spacing: 8},
				Children: []Widget{
					Label{Text: "库存视图"},
					ComboBox{AssignTo: &ui.inventoryMode, Model: modes, CurrentIndex: 0, MinSize: Size{Width: 135, Height: 28}},
					Label{Text: "入库类型"},
					ComboBox{AssignTo: &ui.inventoryType, Model: types, CurrentIndex: 0, MinSize: Size{Width: 135, Height: 28}},
					Label{Text: "物料名称"},
					LineEdit{AssignTo: &ui.inventoryName, MinSize: Size{Width: 155, Height: 28}},
					Label{Text: "物料型号"},
					LineEdit{AssignTo: &ui.inventoryModel, MinSize: Size{Width: 145, Height: 28}},
					Label{Text: "仓库"},
					ComboBox{
						AssignTo:     &ui.inventoryWarehouse,
						Model:        []string{"全部仓库（正在加载）"},
						CurrentIndex: 0,
						MinSize:      Size{Width: 155, Height: 28},
						OnCurrentIndexChanged: func() {
							ui.updateInventoryZones()
						},
					},
					Label{Text: "库区"},
					ComboBox{
						AssignTo:     &ui.inventoryZone,
						Model:        []string{"全部库区"},
						CurrentIndex: 0,
						MinSize:      Size{Width: 155, Height: 28},
						OnCurrentIndexChanged: func() {
							ui.updateInventoryRacks()
						},
					},
					Label{Text: "货架"},
					ComboBox{
						AssignTo:     &ui.inventoryRack,
						Model:        []string{"全部货架"},
						CurrentIndex: 0,
						MinSize:      Size{Width: 145, Height: 28},
						OnCurrentIndexChanged: func() {
							ui.updateInventoryBins()
						},
					},
					Label{Text: "货位"},
					ComboBox{AssignTo: &ui.inventoryBin, Model: []string{"全部货位"}, CurrentIndex: 0, MinSize: Size{Width: 145, Height: 28}},
					Composite{
						ColumnSpan: 8,
						Layout:     HBox{Spacing: 8},
						Children: []Widget{
							HSpacer{},
							PushButton{AssignTo: &ui.inventoryReset, Text: "重置", MinSize: Size{Width: 78, Height: 30}, OnClicked: ui.resetInventoryFilters},
							PushButton{AssignTo: &ui.inventoryQuery, Text: "查询", MinSize: Size{Width: 84, Height: 30}, OnClicked: func() {
								ui.inventoryPage = 1
								ui.loadInventory()
							}},
						},
					},
				},
			},
			TableView{
				AssignTo:         &ui.inventoryTable,
				AlternatingRowBG: true,
				ColumnsOrderable: true,
				StretchFactor:    1,
				Columns: []TableViewColumn{
					{Title: "类型", DataMember: "Type", Width: 90},
					{Title: "入库单", DataMember: "ReceiptCode", Width: 125},
					{Title: "批次入库", DataMember: "ReceiveCode", Width: 135},
					{Title: "仓库", DataMember: "Warehouse", Width: 100},
					{Title: "库位", DataMember: "Location", Width: 150},
					{Title: "物料", DataMember: "Name", Width: 210},
					{Title: "型号", DataMember: "Model", Width: 110},
					{Title: "库存", DataMember: "Quantity", Width: 90},
					{Title: "可用", DataMember: "Available", Width: 90},
					{Title: "锁定", DataMember: "Locked", Width: 90},
					{Title: "冻结", DataMember: "Frozen", Width: 90},
				},
			},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					Label{AssignTo: &ui.inventoryInfo, Text: "尚未加载"},
					HSpacer{},
					Label{Text: "每页"},
					ComboBox{AssignTo: &ui.inventorySize, Model: pageSizeLabels, CurrentIndex: 1, MinSize: Size{Width: 92}, OnCurrentIndexChanged: func() {
						if ui.window != nil && ui.inventorySize != nil && ui.inventorySize.CurrentIndex() >= 0 {
							ui.inventoryPage = 1
							ui.loadInventory()
						}
					}},
					PushButton{AssignTo: &ui.inventoryPrev, Text: "上一页", OnClicked: func() {
						if ui.inventoryPage > 1 {
							ui.inventoryPage--
							ui.loadInventory()
						}
					}},
					PushButton{AssignTo: &ui.inventoryNext, Text: "下一页", OnClicked: func() { ui.inventoryPage++; ui.loadInventory() }},
				},
			},
		},
	}
}

func (ui *mainUI) resetInventoryFilters() {
	ui.inventoryMode.SetCurrentIndex(0)
	ui.inventoryType.SetCurrentIndex(0)
	ui.inventoryName.SetText("")
	ui.inventoryModel.SetText("")
	ui.inventoryWarehouse.SetCurrentIndex(0)
	ui.updateInventoryZones()
	ui.inventoryPage = 1
	ui.loadInventory()
}

func (ui *mainUI) loadMaterials() {
	if ui.materialCancel != nil {
		ui.materialCancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	ui.materialCancel = cancel
	page := ui.materialPage
	size := selectedPageSize(ui.materialSize)
	generation := ui.materialGeneration
	filters := api.MaterialFilters{
		CategoryID:       selectedOptionID(ui.materialCategory, ui.materialCategoryOptions),
		Name:             ui.materialName.Text(),
		Model:            ui.materialModel.Text(),
		Specification:    ui.materialSpecification.Text(),
		Material:         ui.materialMaterial.Text(),
		SurfaceTreatment: ui.materialSurfaceTreatment.Text(),
		StrengthGrade:    ui.materialStrengthGrade.Text(),
	}
	ui.materialInfo.SetText("正在加载线上物料数据……")
	ui.materialPrev.SetEnabled(false)
	ui.materialNext.SetEnabled(false)
	ui.materialQuery.SetEnabled(false)
	ui.materialReset.SetEnabled(false)
	go func() {
		result, err := ui.session.Client.Materials(ctx, page, size, filters)
		if ctx.Err() != nil {
			return
		}
		ui.window.Synchronize(func() {
			if generation != ui.materialGeneration || ui.materialTable == nil {
				return
			}
			ui.materialQuery.SetEnabled(true)
			ui.materialReset.SetEnabled(true)
			if err != nil {
				ui.materialInfo.SetText("加载失败：" + err.Error())
				return
			}
			rows := make([]materialRow, 0, len(result.List))
			for _, item := range result.List {
				rows = append(rows, materialRow{
					Category: item.CategoryName, Name: item.Name, Model: item.Model,
					HasDrawing: materialDrawingStatus(item),
					Material:   item.Material, Specification: item.Specification,
					Surface: item.SurfaceTreatment, Strength: item.StrengthGrade,
					SafeQuantity: fmt.Sprintf("%g %s", item.Quantity, item.Unit), Detail: item,
				})
			}
			ui.materialRows = rows
			if modelErr := ui.materialTable.SetModel(ui.materialRows); modelErr != nil {
				ui.materialRows = nil
				ui.materialInfo.SetText("表格数据错误：" + modelErr.Error())
				return
			}
			ui.materialTotal = result.Total
			ui.materialInfo.SetText(fmt.Sprintf("第 %d 页 | 本页 %d 条 | 共 %d 条", page, len(rows), result.Total))
			ui.materialPrev.SetEnabled(page > 1)
			ui.materialNext.SetEnabled(int64(page*size) < result.Total)
		})
	}()
}

func (ui *mainUI) loadInventory() {
	if ui.inventoryCancel != nil {
		ui.inventoryCancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	ui.inventoryCancel = cancel
	page := ui.inventoryPage
	size := selectedPageSize(ui.inventorySize)
	generation := ui.inventoryGeneration
	filters := api.InventoryFilters{
		MaterialName:    ui.inventoryName.Text(),
		MaterialModel:   ui.inventoryModel.Text(),
		WarehouseID:     selectedNodeID(ui.inventoryWarehouse, ui.inventoryWarehouseNodes),
		WarehouseZoneID: selectedNodeID(ui.inventoryZone, ui.inventoryZoneNodes),
		WarehouseRackID: selectedNodeID(ui.inventoryRack, ui.inventoryRackNodes),
		WarehouseBinID:  selectedNodeID(ui.inventoryBin, ui.inventoryBinNodes),
	}
	if ui.inventoryType.CurrentIndex() > 0 {
		filters.Type = ui.inventoryType.Text()
	}
	ui.inventoryInfo.SetText("正在加载线上库存数据……")
	ui.inventoryPrev.SetEnabled(false)
	ui.inventoryNext.SetEnabled(false)
	ui.inventoryQuery.SetEnabled(false)
	ui.inventoryReset.SetEnabled(false)
	historyMode := ui.inventoryMode != nil && ui.inventoryMode.CurrentIndex() == 1
	go func() {
		var result api.InventoryPage
		var err error
		if historyMode {
			result, err = ui.session.Client.InventoryHistory(ctx, page, size, filters)
		} else {
			result, err = ui.session.Client.Inventory(ctx, page, size, filters)
		}
		if ctx.Err() != nil {
			return
		}
		ui.window.Synchronize(func() {
			if generation != ui.inventoryGeneration || ui.inventoryTable == nil {
				return
			}
			ui.inventoryQuery.SetEnabled(true)
			ui.inventoryReset.SetEnabled(true)
			if err != nil {
				ui.inventoryInfo.SetText("加载失败：" + err.Error())
				return
			}
			rows := make([]inventoryRow, 0, len(result.List))
			for _, item := range result.List {
				location := fmt.Sprintf("%s / %s / %s", item.WarehouseZoneName, item.WarehouseRackName, item.WarehouseBinName)
				rows = append(rows, inventoryRow{
					Type: item.Type, ReceiptCode: item.ReceiptCode, ReceiveCode: item.ReceiveCode,
					Warehouse: item.WarehouseName, Location: location,
					Name: item.Name, Model: item.Model,
					Quantity:  fmt.Sprintf("%g %s", item.Quantity, item.Unit),
					Available: fmt.Sprintf("%g", item.AvailableQuantity),
					Locked:    fmt.Sprintf("%g", item.LockedQuantity),
					Frozen:    fmt.Sprintf("%g", item.FrozenQuantity),
				})
			}
			if modelErr := ui.inventoryTable.SetModel(rows); modelErr != nil {
				ui.inventoryInfo.SetText("表格数据错误：" + modelErr.Error())
				return
			}
			ui.inventoryTotal = result.Total
			modeLabel := "当前库存"
			if historyMode {
				modeLabel = "库存批次历史"
			}
			ui.inventoryInfo.SetText(fmt.Sprintf("%s | 第 %d 页 | 本页 %d 条 | 共 %d 条 | 可用合计 %g",
				modeLabel, page, len(rows), result.Total, result.Quantity))
			ui.inventoryPrev.SetEnabled(page > 1)
			ui.inventoryNext.SetEnabled(int64(page*size) < result.Total)
		})
	}()
}

func (ui *mainUI) loadFilterOptions() {
	loadCategories := ui.materialCategory != nil
	loadWarehouses := ui.inventoryWarehouse != nil
	loadSuppliers := ui.inboundSupplier != nil
	loadCustomers := ui.inboundCustomer != nil
	go func() {
		var (
			categories  []api.MaterialCategory
			warehouses  []api.WarehouseNode
			suppliers   []api.Supplier
			customers   []api.Customer
			errorLabels []string
			categoryOK  = !loadCategories
			warehouseOK = !loadWarehouses
			supplierOK  = !loadSuppliers
			customerOK  = !loadCustomers
		)

		if loadCategories {
			var err error
			categories, err = ui.session.Client.MaterialCategories(context.Background())
			categoryOK = err == nil
			if err != nil {
				errorLabels = append(errorLabels, "物料分类")
			}
		}
		if loadWarehouses {
			var err error
			warehouses, err = ui.session.Client.WarehouseTree(context.Background())
			warehouseOK = err == nil
			if err != nil {
				errorLabels = append(errorLabels, "仓库位置")
			}
		}
		if loadSuppliers {
			var err error
			suppliers, err = ui.session.Client.Suppliers(context.Background())
			supplierOK = err == nil
			if err != nil {
				errorLabels = append(errorLabels, "供应商")
			}
		}
		if loadCustomers {
			var err error
			customers, err = ui.session.Client.Customers(context.Background())
			customerOK = err == nil
			if err != nil {
				errorLabels = append(errorLabels, "客户")
			}
		}

		ui.window.Synchronize(func() {
			if loadCategories {
				ui.categoryReady = true
				ui.categoryFailed = !categoryOK
				if categoryOK {
					ui.materialCategoryOptions = flattenCategoryOptions(categories, "")
				}
				ui.applyMaterialCategoryOptions()
			}
			if loadWarehouses {
				ui.warehouseReady = true
				ui.warehouseFailed = !warehouseOK
				if warehouseOK {
					ui.inventoryWarehouseNodes = warehouses
				}
				ui.applyWarehouseOptions()
			}
			if loadSuppliers {
				ui.supplierReady = true
				ui.supplierFailed = !supplierOK
				if supplierOK {
					ui.inboundSupplierOptions = make([]selectOption, 0, len(suppliers))
					for _, supplier := range suppliers {
						optionLabel := supplier.Name
						if supplier.Code != "" {
							optionLabel += " · " + supplier.Code
						}
						ui.inboundSupplierOptions = append(ui.inboundSupplierOptions, selectOption{ID: supplier.ID, Label: optionLabel})
					}
				}
				ui.applySupplierOptions()
			}
			if loadCustomers {
				ui.customerReady = true
				ui.customerFailed = !customerOK
				if customerOK {
					ui.inboundCustomerOptions = make([]selectOption, 0, len(customers))
					for _, customer := range customers {
						ui.inboundCustomerOptions = append(ui.inboundCustomerOptions, selectOption{
							ID: customer.ID, Label: businessOptionLabel(customer.Name, customer.Code),
						})
					}
				}
				ui.applyCustomerOptions()
			}
			if len(errorLabels) == 0 {
				ui.status.SetText(fmt.Sprintf("已连接线上 API · 分类 %d · 仓库 %d · 供应商 %d · 客户 %d · v%s",
					len(ui.materialCategoryOptions), len(ui.inventoryWarehouseNodes), len(ui.inboundSupplierOptions),
					len(ui.inboundCustomerOptions), clientVersion))
			} else {
				ui.status.SetText("已连接线上 API · 以下筛选选项加载失败，可稍后重启重试：" + strings.Join(errorLabels, "、"))
			}
		})
	}()
}

func flattenCategoryOptions(categories []api.MaterialCategory, parent string) []selectOption {
	result := make([]selectOption, 0)
	for _, category := range categories {
		label := category.Name
		if parent != "" {
			label = parent + " / " + category.Name
		}
		result = append(result, selectOption{ID: category.ID, Label: label})
		result = append(result, flattenCategoryOptions(category.Children, label)...)
	}
	return result
}

func optionLabels(allLabel string, options []selectOption) []string {
	labels := make([]string, 1, len(options)+1)
	labels[0] = allLabel
	for _, option := range options {
		labels = append(labels, option.Label)
	}
	return labels
}

func nodeLabels(allLabel string, nodes []api.WarehouseNode) []string {
	labels := make([]string, 1, len(nodes)+1)
	labels[0] = allLabel
	for _, node := range nodes {
		labels = append(labels, node.Name)
	}
	return labels
}

func selectedOptionID(combo *walk.ComboBox, options []selectOption) string {
	if combo == nil {
		return ""
	}
	index := combo.CurrentIndex() - 1
	if index < 0 || index >= len(options) {
		return ""
	}
	return options[index].ID
}

func selectedNodeID(combo *walk.ComboBox, nodes []api.WarehouseNode) string {
	if combo == nil {
		return ""
	}
	index := combo.CurrentIndex() - 1
	if index < 0 || index >= len(nodes) {
		return ""
	}
	return nodes[index].ID
}

func selectedNodeChildren(combo *walk.ComboBox, nodes []api.WarehouseNode) []api.WarehouseNode {
	if combo == nil {
		return nil
	}
	index := combo.CurrentIndex() - 1
	if index < 0 || index >= len(nodes) {
		return nil
	}
	return nodes[index].Children
}

func selectedPageSize(combo *walk.ComboBox) int {
	if combo == nil {
		return 20
	}
	index := combo.CurrentIndex()
	if index < 0 || index >= len(pageSizes) {
		return 20
	}
	return pageSizes[index]
}

func (ui *mainUI) updateInventoryZones() {
	if ui.inventoryZone == nil {
		return
	}
	ui.inventoryZoneNodes = selectedNodeChildren(ui.inventoryWarehouse, ui.inventoryWarehouseNodes)
	_ = ui.inventoryZone.SetModel(nodeLabels("全部库区", ui.inventoryZoneNodes))
	ui.inventoryZone.SetCurrentIndex(0)
	ui.updateInventoryRacks()
}

func (ui *mainUI) updateInventoryRacks() {
	if ui.inventoryRack == nil {
		return
	}
	ui.inventoryRackNodes = selectedNodeChildren(ui.inventoryZone, ui.inventoryZoneNodes)
	_ = ui.inventoryRack.SetModel(nodeLabels("全部货架", ui.inventoryRackNodes))
	ui.inventoryRack.SetCurrentIndex(0)
	ui.updateInventoryBins()
}

func (ui *mainUI) updateInventoryBins() {
	if ui.inventoryBin == nil {
		return
	}
	ui.inventoryBinNodes = selectedNodeChildren(ui.inventoryRack, ui.inventoryRackNodes)
	_ = ui.inventoryBin.SetModel(nodeLabels("全部货位", ui.inventoryBinNodes))
	ui.inventoryBin.SetCurrentIndex(0)
}
