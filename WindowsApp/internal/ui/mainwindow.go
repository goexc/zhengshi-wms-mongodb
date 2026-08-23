package ui

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
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
	Detail      api.Inventory
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
	session           *Session
	cfg               config.Config
	window            *walk.MainWindow
	status            *walk.Label
	workspaceSplitter *walk.Splitter
	layoutTables      map[string]*walk.TableView
	layoutSplitters   map[string]*walk.Splitter
	backgroundTasks   *backgroundTaskRegistry
	taskButton        *walk.PushButton

	sideMenu                *walk.ListBox
	sideMenuPanel           *walk.Composite
	sideMenuToggle          *walk.PushButton
	sideMenuCollapsed       bool
	tabs                    *walk.TabWidget
	closeTabButton          *walk.PushButton
	menuKeys                []string
	syncingMenu             bool
	workspaceReady          bool
	initializedPages        map[string]bool
	dashboardTab            *walk.TabPage
	globalLookupTab         *walk.TabPage
	operationTab            *walk.TabPage
	documentTab             *walk.TabPage
	materialTab             *walk.TabPage
	materialEditorTab       *walk.TabPage
	materialCategoryTab     *walk.TabPage
	materialQuoteTab        *walk.TabPage
	materialQuoteEditorTab  *walk.TabPage
	imageAssetTab           *walk.TabPage
	inventoryTab            *walk.TabPage
	inboundTab              *walk.TabPage
	inboundEditorTab        *walk.TabPage
	outboundTab             *walk.TabPage
	outboundEditorTab       *walk.TabPage
	fastOutboundTab         *walk.TabPage
	outboundReportTab       *walk.TabPage
	partnerTab              *walk.TabPage
	partnerEditorTab        *walk.TabPage
	warehouseTab            *walk.TabPage
	warehouseEditorTab      *walk.TabPage
	adminTab                *walk.TabPage
	profileTab              *walk.TabPage
	systemTab               *walk.TabPage
	businessTraceTab        *walk.TabPage
	customerFinanceTab      *walk.TabPage
	businessTraceCancel     context.CancelFunc
	businessTraceGeneration int
	dashboard               *dashboardUI
	globalLookup            *globalLookupUI
	operations              *operationCenterUI
	documents               *documentCenterUI
	inboundEditor           *inboundEditorUI
	outbound                *outboundUI
	outboundEditor          *outboundEditorUI
	fastOutbound            *fastOutboundUI
	outboundReport          *outboundReportUI
	partner                 *partnerUI
	partnerEditor           *partnerEditorUI
	warehouse               *warehouseUI
	warehouseEditor         *warehouseEditorUI
	admin                   *adminUI
	materialEditor          *materialEditorUI
	materialCategories      *materialCategoryUI
	materialQuote           *materialQuoteUI
	materialQuoteEditor     *materialQuoteEditorUI
	imageAssets             *imageAssetUI
	customerFinance         *customerFinanceUI
	profileAvatar           *walk.LineEdit
	profileAvatarStatus     *walk.Label
	profileAvatarPreview    *walk.PushButton
	profileAvatarChange     *walk.PushButton
	profileAvatarCancel     context.CancelFunc
	profileAvatarBusy       bool
	profileAvatarCancelable bool
	loggedOut               *bool
	categoryReady           bool
	categoryFailed          bool
	categoryLoading         bool
	warehouseReady          bool
	warehouseFailed         bool
	warehouseLoading        bool
	supplierReady           bool
	supplierFailed          bool
	supplierLoading         bool
	customerReady           bool
	customerFailed          bool
	customerLoading         bool

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
	materialAdd              *walk.PushButton
	materialEdit             *walk.PushButton
	materialCategoryManage   *walk.PushButton
	materialPrev             *walk.PushButton
	materialNext             *walk.PushButton
	materialSize             *walk.ComboBox
	materialCategoryOptions  []selectOption
	materialRows             []materialRow
	materialPage             int
	materialTotal            int64
	materialGeneration       int
	materialCancel           context.CancelFunc

	inventoryMode             *walk.ComboBox
	inventoryType             *walk.ComboBox
	inventoryName             *walk.LineEdit
	inventoryModel            *walk.LineEdit
	inventoryWarehouse        *walk.ComboBox
	inventoryZone             *walk.ComboBox
	inventoryRack             *walk.ComboBox
	inventoryBin              *walk.ComboBox
	inventoryTable            *walk.TableView
	inventoryInfo             *walk.Label
	inventoryQuery            *walk.PushButton
	inventoryReset            *walk.PushButton
	inventoryPrev             *walk.PushButton
	inventoryNext             *walk.PushButton
	inventorySize             *walk.ComboBox
	inventoryTrace            *walk.PushButton
	inventoryExport           *walk.PushButton
	inventoryExportStop       *walk.PushButton
	inventoryWarehouseNodes   []api.WarehouseNode
	inventoryZoneNodes        []api.WarehouseNode
	inventoryRackNodes        []api.WarehouseNode
	inventoryBinNodes         []api.WarehouseNode
	inventoryPage             int
	inventoryTotal            int64
	inventoryRows             []inventoryRow
	inventoryGeneration       int
	inventoryCancel           context.CancelFunc
	inventoryExportCancel     context.CancelFunc
	inventoryExportGeneration int
	inventoryExportBusy       bool

	inboundSearch           *walk.LineEdit
	inboundStatus           *walk.ComboBox
	inboundType             *walk.ComboBox
	inboundSupplier         *walk.ComboBox
	inboundCustomer         *walk.ComboBox
	inboundTable            *walk.TableView
	inboundInfo             *walk.Label
	inboundQuery            *walk.PushButton
	inboundReset            *walk.PushButton
	inboundPrev             *walk.PushButton
	inboundNext             *walk.PushButton
	inboundAdd              *walk.PushButton
	inboundEdit             *walk.PushButton
	inboundCheck            *walk.PushButton
	inboundDelete           *walk.PushButton
	inboundReceive          *walk.PushButton
	inboundExport           *walk.PushButton
	inboundExportStop       *walk.PushButton
	inboundActionHint       *walk.Label
	inboundSize             *walk.ComboBox
	inboundSupplierOptions  []selectOption
	inboundCustomerOptions  []selectOption
	inboundPage             int
	inboundTotal            int64
	inboundRows             []api.InboundReceipt
	inboundGeneration       int
	inboundCancel           context.CancelFunc
	inboundExportCancel     context.CancelFunc
	inboundExportGeneration int
	inboundExportBusy       bool
	inboundOperationBusy    bool
	sessionInvalidated      atomic.Bool
}

var pageSizes = []int{10, 20, 50, 100}

var pageSizeLabels = []string{"10 条/页", "20 条/页", "50 条/页", "100 条/页"}

var clientVersion = "1.7.0"
var buildTime = "development"
var gitCommit = "unknown"

type MainResult struct {
	LoggedOut bool
}

type topLevelMenuSpec struct {
	Key   string
	Label string
}

func availableTopLevelMenuSpecs(perms api.Perms, dashboard, partners, warehouses bool) []topLevelMenuSpec {
	specs := make([]topLevelMenuSpec, 0, 16)
	if dashboard {
		specs = append(specs, topLevelMenuSpec{Key: "dashboard", Label: "仓库作业台"})
	}
	specs = append(specs, topLevelMenuSpec{Key: "global_lookup", Label: "统一检索"})
	if hasMenuPath(perms.Menus, "/material/list") {
		specs = append(specs, topLevelMenuSpec{Key: "material", Label: "物料中心"})
	}
	if hasMenuPath(perms.Menus, "/material/quote") {
		specs = append(specs, topLevelMenuSpec{Key: "material_quote", Label: "新增物料报价"})
	}
	if imageAssetMenuAvailable(perms) {
		specs = append(specs, topLevelMenuSpec{Key: "image_assets", Label: "图片素材"})
	}
	if hasAnyMenuPath(perms.Menus, "/inventory/index", "/inventory/record") {
		specs = append(specs, topLevelMenuSpec{Key: "inventory", Label: "库存查询"})
	}
	if hasMenuPath(perms.Menus, "/inbound/receipt") {
		specs = append(specs, topLevelMenuSpec{Key: "inbound", Label: "入库工作台"})
	}
	if hasAnyMenuPath(perms.Menus, "/outbound/receipt", "/outbound/receipt2") {
		specs = append(specs, topLevelMenuSpec{Key: "outbound", Label: "出库执行"})
	}
	if hasMenuPath(perms.Menus, "/outbound/report") {
		specs = append(specs, topLevelMenuSpec{Key: "outbound_report", Label: "出库报表"})
	}
	if partners {
		specs = append(specs, topLevelMenuSpec{Key: "partner", Label: "合作伙伴"})
	}
	if warehouses {
		specs = append(specs, topLevelMenuSpec{Key: "warehouse", Label: "仓储结构"})
	}
	if adminMenuAvailable(perms) {
		specs = append(specs, topLevelMenuSpec{Key: "admin", Label: "系统管理"})
	}
	if documentMenuAvailable(perms) {
		specs = append(specs, topLevelMenuSpec{Key: "documents", Label: "单据预览与打印"})
	}
	return append(specs,
		topLevelMenuSpec{Key: "operations", Label: "操作复核"},
		topLevelMenuSpec{Key: "profile", Label: "个人中心"},
		topLevelMenuSpec{Key: "system", Label: "系统信息"},
	)
}

func (ui *mainUI) initialTopLevelPageKeys() []string {
	if ui.workspaceMatchesSession() && len(ui.cfg.Workspace.OpenPages) > 0 {
		desired := make(map[string]bool, len(ui.cfg.Workspace.OpenPages))
		for _, key := range ui.cfg.Workspace.OpenPages {
			desired[key] = true
		}
		result := make([]string, 0, len(desired))
		for _, key := range ui.menuKeys {
			if key != "system" && desired[key] {
				result = append(result, key)
			}
		}
		return result
	}
	if stringIndex(ui.menuKeys, "dashboard") >= 0 {
		return []string{"dashboard"}
	}
	return []string{"global_lookup"}
}

func (ui *mainUI) systemPageWidget() TabPage {
	return TabPage{
		AssignTo: &ui.systemTab,
		Title:    "系统信息",
		Layout:   VBox{Margins: Margins{Left: 24, Top: 24, Right: 24, Bottom: 24}, Spacing: 10},
		Children: []Widget{
			Label{Text: "系统信息", Font: Font{Family: "Microsoft YaHei UI", PointSize: 15, Bold: true}, Accessibility: Accessibility{Name: "系统信息"}},
			Label{Text: "Windows 原生仓库执行客户端，当前直接连接线上 API。", TextColor: secondaryTextColor()},
			HSpacer{Size: 8},
			Label{Text: fmt.Sprintf("当前账号：%s（%s）", ui.session.Profile.Name, ui.session.Profile.DepartmentName)},
			Label{Text: "服务地址：" + ui.cfg.APIBaseURL},
			Label{Text: fmt.Sprintf("已加载 %d 个顶级权限菜单。", len(ui.session.Perms.Menus))},
			Label{Text: "客户端按账号权限创建功能入口；业务页按需加载，所有写操作仍由服务端权限及状态规则最终校验。"},
			Label{Text: fmt.Sprintf("客户端版本：v%s    构建时间：%s    提交：%s", clientVersion, buildTime, gitCommit)},
			Composite{Layout: HBox{Spacing: 8}, Children: []Widget{
				PushButton{Text: "查看诊断信息", MinSize: Size{Width: 108, Height: 32}, Accessibility: Accessibility{Name: "查看只读诊断信息"}, OnClicked: ui.showDiagnosticsDialog},
				Label{Text: "诊断摘要不包含密码或身份 Token。", TextColor: secondaryTextColor()},
				HSpacer{},
			}},
			VSpacer{},
		},
	}
}

func RunMain(session *Session, cfg config.Config) (MainResult, error) {
	var result MainResult
	dashboardSpecs := dashboardCardSpecs(session.Perms)
	ui := &mainUI{
		session: session, cfg: cfg, materialPage: 1, inventoryPage: 1, inboundPage: 1,
		initializedPages: make(map[string]bool),
		layoutTables:     make(map[string]*walk.TableView),
		layoutSplitters:  make(map[string]*walk.Splitter),
		backgroundTasks:  newBackgroundTaskRegistry(),
		loggedOut:        &result.LoggedOut,
		dashboard:        newDashboardUI(dashboardSpecs),
		globalLookup:     newGlobalLookupUI(),
		operations:       newOperationCenterUI(),
		documents:        newDocumentCenterUI(),
		outbound:         newOutboundUI(),
		outboundReport:   newOutboundReportUI(),
		partner:          newPartnerUI(availablePartnerKinds(session.Perms)),
		warehouse:        newWarehouseUI(availableWarehouseKinds(session.Perms)),
		materialQuote:    newMaterialQuoteUI(),
		imageAssets:      newImageAssetUI(),
	}
	if !ui.workspaceMatchesSession() || ui.cfg.Workspace.LayoutVersion != workspaceLayoutVersion {
		ui.cfg.Workspace.LayoutVersion = workspaceLayoutVersion
		ui.cfg.Workspace.TableLayouts = nil
		ui.cfg.Workspace.SplitterLayouts = nil
	}
	ui.sideMenuCollapsed = ui.workspaceMatchesSession() && cfg.Workspace.SideMenuCollapsed
	ui.loadOperationJournal()
	title := fmt.Sprintf("正时 WMS · %s", session.Profile.Name)
	pages := make([]TabPage, 0, 8)
	menuLabels := make([]string, 0, 9)
	menuSpecs := availableTopLevelMenuSpecs(session.Perms, len(dashboardSpecs) > 0, len(ui.partner.kinds) > 0, len(ui.warehouse.kinds) > 0)
	for _, spec := range menuSpecs {
		menuLabels = append(menuLabels, spec.Label)
		ui.menuKeys = append(ui.menuKeys, spec.Key)
	}
	for _, key := range ui.initialTopLevelPageKeys() {
		page, err := ui.pageDeclaration(key)
		if err != nil {
			return result, err
		}
		pages = append(pages, page)
	}
	pages = append(pages, ui.systemPageWidget())
	initialBounds := ui.initialMainWindowBounds()
	err := MainWindow{
		AssignTo: &ui.window,
		Title:    title,
		MinSize:  Size{Width: mainWindowMinWidth, Height: mainWindowMinHeight},
		Bounds:   Rectangle{X: initialBounds.X, Y: initialBounds.Y, Width: initialBounds.Width, Height: initialBounds.Height},
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
					Action{
						Text: "统一检索", Shortcut: Shortcut{Modifiers: walk.ModControl, Key: walk.KeyK},
						OnTriggered: func() { ui.openTab("global_lookup") },
					},
					Action{
						Text: "新建当前模块单据", Shortcut: Shortcut{Modifiers: walk.ModControl, Key: walk.KeyN},
						OnTriggered: ui.newCurrentRecord,
					},
					Action{
						Text: "保存当前编辑页", Shortcut: Shortcut{Modifiers: walk.ModControl, Key: walk.KeyS},
						OnTriggered: ui.saveCurrentEditor,
					},
					Separator{},
					Action{
						Text: "关闭当前标签", Shortcut: Shortcut{Modifiers: walk.ModControl, Key: walk.KeyW},
						OnTriggered: ui.closeCurrentTab,
					},
					Action{
						Text: "展开或收起功能菜单", Shortcut: Shortcut{Modifiers: walk.ModControl, Key: walk.KeyM},
						OnTriggered: ui.toggleSideMenu,
					},
				},
			},
		},
		Children: []Widget{
			Composite{
				Layout: HBox{Spacing: 10},
				Children: []Widget{
					Label{Text: "正时 WMS", Font: Font{Family: "Microsoft YaHei UI", PointSize: 16, Bold: true}},
					Label{Text: "线上生产环境", Font: Font{Family: "Microsoft YaHei UI", PointSize: 9, Bold: true}, TextColor: dangerTextColor(), ToolTipText: "当前所有业务操作直接访问生产数据"},
					Label{Text: cfg.APIBaseURL, TextColor: secondaryTextColor()},
					HSpacer{},
					Label{Text: session.Profile.Name, Font: Font{Family: "Microsoft YaHei UI", PointSize: 9, Bold: true}},
					Label{Text: session.Profile.DepartmentName, TextColor: secondaryTextColor()},
					PushButton{Text: "个人中心", MinSize: Size{Width: 92, Height: 30}, Accessibility: Accessibility{Name: "打开个人中心"}, OnClicked: func() { ui.openTab("profile") }},
					PushButton{Text: "退出登录", Accessibility: Accessibility{Name: "退出当前账号"}, OnClicked: func() {
						if !ui.confirmSessionExit("退出登录") {
							return
						}
						ui.saveWorkspaceState()
						result.LoggedOut = true
						_ = securestore.Delete()
						guardedGo(func() { _ = session.Client.Logout(context.Background()) })
						ui.window.Close()
					}},
				},
			},
			HSplitter{
				AssignTo:      &ui.workspaceSplitter,
				HandleWidth:   4,
				StretchFactor: 1,
				Children: []Widget{
					Composite{
						AssignTo:      &ui.sideMenuPanel,
						Visible:       !ui.sideMenuCollapsed,
						MinSize:       Size{Width: 176},
						MaxSize:       Size{Width: 204},
						StretchFactor: 2,
						Layout:        VBox{Margins: Margins{Left: 12, Top: 12, Right: 12, Bottom: 12}, Spacing: 8},
						Children: []Widget{
							Label{Text: "功能菜单", Font: Font{Family: "Microsoft YaHei UI", PointSize: 12, Bold: true}},
							Label{Text: "单击打开或切换工作页", TextColor: secondaryTextColor()},
							ListBox{
								AssignTo:              &ui.sideMenu,
								Model:                 menuLabels,
								MinSize:               Size{Width: 176, Height: 240},
								StretchFactor:         1,
								Accessibility:         Accessibility{Name: "功能菜单", Description: "选择后打开或切换工作页"},
								OnCurrentIndexChanged: ui.openSelectedMenu,
								OnItemActivated:       ui.openSelectedMenu,
							},
							Label{Text: "关闭的页面可从菜单重新打开。", TextColor: secondaryTextColor()},
						},
					},
					Composite{
						StretchFactor: 12,
						Layout:        VBox{Margins: Margins{Left: 8, Top: 6, Right: 2, Bottom: 2}, Spacing: 6},
						Children: []Widget{
							Composite{
								Layout: HBox{Spacing: 8},
								Children: []Widget{
									PushButton{
										AssignTo:      &ui.sideMenuToggle,
										Text:          ui.sideMenuToggleText(),
										MinSize:       Size{Width: 92, Height: 30},
										ToolTipText:   "展开或收起左侧功能菜单（Ctrl+M）",
										Accessibility: Accessibility{Name: ui.sideMenuToggleAccessibilityName()},
										OnClicked:     ui.toggleSideMenu,
									},
									Label{Text: "工作区", Font: Font{Family: "Microsoft YaHei UI", PointSize: 10, Bold: true}},
									Label{Text: "业务页面以标签方式打开", TextColor: secondaryTextColor()},
									HSpacer{},
									PushButton{
										AssignTo:      &ui.closeTabButton,
										Text:          "关闭当前标签",
										MinSize:       Size{Width: 110, Height: 30},
										ToolTipText:   "关闭当前业务标签；可从左侧菜单重新打开",
										Accessibility: Accessibility{Name: "关闭当前业务标签"},
										OnClicked:     ui.closeCurrentTab,
									},
								},
							},
							TabWidget{
								AssignTo:              &ui.tabs,
								Pages:                 pages,
								StretchFactor:         1,
								Accessibility:         Accessibility{Name: "业务工作区标签"},
								OnCurrentIndexChanged: ui.handleTabChanged,
							},
						},
					},
				},
			},
			Composite{Layout: HBox{Spacing: 8}, Children: []Widget{
				Label{AssignTo: &ui.status, Text: "已连接线上 API · 页面将按需加载", TextColor: secondaryTextColor(), Accessibility: Accessibility{Name: "客户端状态", Description: "显示加载、提交和错误恢复状态"}},
				HSpacer{},
				PushButton{AssignTo: &ui.taskButton, Text: "后台任务（0）", Enabled: false, MinSize: Size{Width: 112, Height: 28}, ToolTipText: "查看可安全取消的查询导出任务", Accessibility: Accessibility{Name: "查看后台任务，当前 0 项"}, OnClicked: ui.showBackgroundTasks},
			}},
		},
	}.Create()
	if err != nil {
		return result, err
	}
	ui.registerSplitterLayout("main.workspace", ui.workspaceSplitter)
	if err := ui.installTabCloseHandler(); err != nil {
		ui.window.Dispose()
		return result, err
	}
	setAsyncIncidentHandler(func(incidentID string) {
		defer func() { _ = recover() }()
		if ui.window == nil || ui.window.IsDisposed() {
			return
		}
		ui.window.Synchronize(func() {
			if ui.status == nil || ui.window.IsDisposed() {
				return
			}
			ui.status.SetText("后台任务已安全停止，故障编号：" + incidentID + "。如涉及写操作，请在“操作复核”中确认线上结果。")
			ui.status.SetTextColor(dangerTextColor())
		})
	})
	ui.restoreWorkspaceTabs()
	session.Client.SetOperationObserver(ui.recordOperation)
	session.Client.SetUnauthorizedHandler(func() {
		if !ui.sessionInvalidated.CompareAndSwap(false, true) {
			return
		}
		_ = securestore.Delete()
		guardedGo(func() {
			ui.window.Synchronize(func() {
				result.LoggedOut = true
				walk.MsgBox(ui.window, "登录状态已失效", "线上服务已拒绝当前身份凭据，客户端已清除本地登录缓存。关闭主窗口后将返回登录页。", walk.MsgBoxIconWarning)
				ui.window.Close()
			})
		})
	})
	ui.window.Closing().Attach(func(canceled *bool, _ walk.CloseReason) {
		if result.LoggedOut {
			return
		}
		if !ui.confirmSessionExit("退出客户端") {
			*canceled = true
			return
		}
		ui.saveWorkspaceState()
	})
	ui.window.Disposing().Attach(func() {
		setAsyncIncidentHandler(nil)
		session.Client.SetUnauthorizedHandler(nil)
		session.Client.SetOperationObserver(nil)
		ui.disposeDetachedTabs()
	})
	ui.handleTabChanged()
	ui.showRestoredOperationSummary()
	ui.window.Run()
	return result, nil
}

func (ui *mainUI) sideMenuToggleText() string {
	if ui.sideMenuCollapsed {
		return "展开菜单"
	}
	return "收起菜单"
}

func (ui *mainUI) sideMenuToggleAccessibilityName() string {
	if ui.sideMenuCollapsed {
		return "展开左侧功能菜单"
	}
	return "收起左侧功能菜单"
}

func (ui *mainUI) toggleSideMenu() {
	if ui.sideMenuPanel == nil {
		return
	}
	ui.sideMenuCollapsed = !ui.sideMenuCollapsed
	ui.sideMenuPanel.SetVisible(!ui.sideMenuCollapsed)
	if ui.sideMenuToggle != nil {
		ui.sideMenuToggle.SetText(ui.sideMenuToggleText())
		_ = ui.sideMenuToggle.Accessibility().SetName(ui.sideMenuToggleAccessibilityName())
	}
	if ui.status != nil {
		ui.status.SetText(map[bool]string{true: "左侧功能菜单已收起，可使用“展开菜单”恢复。", false: "左侧功能菜单已展开。"}[ui.sideMenuCollapsed])
	}
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
	if stringIndex(ui.menuKeys, key) < 0 {
		if ui.status != nil {
			ui.status.SetText("当前账号没有打开该工作页的菜单权限。")
			ui.status.SetTextColor(dangerTextColor())
		}
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
	ui.initializePage(key)
	ui.saveWorkspaceState()
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
	if page == ui.inboundEditorTab {
		ui.closeInboundEditor(false)
		return
	}
	if page == ui.materialEditorTab {
		ui.closeMaterialEditor(false)
		return
	}
	if page == ui.materialCategoryTab {
		ui.closeMaterialCategoryPage()
		return
	}
	if page == ui.materialQuoteEditorTab {
		ui.closeMaterialQuoteEditor(false)
		return
	}
	if page == ui.outboundEditorTab {
		ui.closeOutboundEditor(false)
		return
	}
	if page == ui.fastOutboundTab {
		ui.closeFastOutbound(false)
		return
	}
	if page == ui.partnerEditorTab {
		ui.closePartnerEditor(false)
		return
	}
	if page == ui.warehouseEditorTab {
		ui.closeWarehouseEditor(false)
		return
	}
	if page == ui.customerFinanceTab {
		ui.closeCustomerFinanceTab()
		return
	}
	if page == ui.profileTab && ui.profileAvatarBusy {
		walk.MsgBox(ui.window, "头像任务正在执行", "请先取消头像上传，或等待头像更新与线上回读完成后再关闭页面。", walk.MsgBoxIconInformation)
		return
	}
	if err := ui.tabs.Pages().RemoveAt(index); err != nil {
		walk.MsgBox(ui.window, "无法关闭页面", err.Error(), walk.MsgBoxIconError)
		return
	}
	ui.syncNavigationFromTab()
	ui.saveWorkspaceState()
}

func (ui *mainUI) disposeDetachedTabs() {
	if ui.tabs == nil {
		return
	}
	if ui.globalLookup != nil && ui.globalLookup.cancel != nil {
		ui.globalLookup.cancel()
	}
	if ui.documents != nil && ui.documents.cancel != nil {
		ui.documents.cancel()
	}
	if ui.inboundEditor != nil {
		ui.inboundEditor.dispose()
	}
	if ui.businessTraceCancel != nil {
		ui.businessTraceCancel()
	}
	if ui.outboundEditor != nil {
		ui.outboundEditor.dispose()
	}
	if ui.fastOutbound != nil {
		ui.fastOutbound.dispose()
	}
	if ui.partnerEditor != nil {
		ui.partnerEditor.dispose()
	}
	if ui.warehouseEditor != nil {
		ui.warehouseEditor.dispose()
	}
	if ui.materialEditor != nil {
		ui.materialEditor.dispose()
	}
	if ui.materialCategories != nil {
		ui.materialCategories.dispose()
	}
	if ui.materialQuote != nil {
		ui.materialQuote.dispose()
	}
	if ui.materialQuoteEditor != nil {
		ui.materialQuoteEditor.dispose()
	}
	if ui.customerFinance != nil {
		ui.customerFinance.dispose()
	}
	if ui.admin != nil {
		ui.admin.dispose()
	}
	if ui.profileAvatarCancel != nil {
		ui.profileAvatarCancel()
	}
	for _, page := range []*walk.TabPage{
		ui.dashboardTab, ui.globalLookupTab, ui.operationTab, ui.documentTab, ui.materialTab, ui.materialEditorTab, ui.materialCategoryTab, ui.materialQuoteTab, ui.materialQuoteEditorTab,
		ui.imageAssetTab,
		ui.inventoryTab, ui.inboundTab, ui.inboundEditorTab, ui.outboundTab, ui.outboundEditorTab, ui.fastOutboundTab,
		ui.outboundReportTab, ui.partnerTab, ui.partnerEditorTab, ui.warehouseTab, ui.warehouseEditorTab, ui.adminTab, ui.profileTab,
		ui.businessTraceTab,
		ui.customerFinanceTab,
	} {
		if page != nil && ui.tabs.Pages().Index(page) < 0 {
			page.Dispose()
		}
	}
}

func (ui *mainUI) createTab(key string) (*walk.TabPage, error) {
	pageDecl, err := ui.pageDeclaration(key)
	if err != nil {
		return nil, err
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
	ui.loadFilterOptions()
	return page, nil
}

func (ui *mainUI) pageDeclaration(key string) (TabPage, error) {
	switch key {
	case "dashboard":
		return ui.dashboardPageWidget(), nil
	case "global_lookup":
		return ui.globalLookupPageWidget(), nil
	case "operations":
		return ui.operationCenterPageWidget(), nil
	case "documents":
		return ui.documentCenterPageWidget(), nil
	case "material":
		return ui.materialPageWidget(), nil
	case "material_quote":
		return ui.materialQuotePageWidget(), nil
	case "image_assets":
		return ui.imageAssetPageWidget(), nil
	case "inventory":
		return ui.inventoryPageWidget(), nil
	case "inbound":
		return ui.inboundPageWidget(), nil
	case "outbound":
		return ui.outboundPageWidget(), nil
	case "outbound_report":
		return ui.outboundReportPageWidget(), nil
	case "partner":
		return ui.partnerPageWidget(), nil
	case "warehouse":
		return ui.warehousePageWidget(), nil
	case "admin":
		return ui.adminPageWidget(), nil
	case "profile":
		return ui.profilePageWidget(), nil
	default:
		return TabPage{}, fmt.Errorf("未知工作页：%s", key)
	}
}

func (ui *mainUI) handleTabChanged() {
	ui.syncNavigationFromTab()
	if !ui.workspaceReady || ui.tabs == nil {
		return
	}
	index := ui.tabs.CurrentIndex()
	if index < 0 || index >= ui.tabs.Pages().Len() {
		return
	}
	key := ui.keyForTab(ui.tabs.Pages().At(index))
	ui.initializePage(key)
	ui.loadFilterOptions()
	ui.saveWorkspaceState()
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
	case "dashboard":
		return ui.dashboardTab
	case "global_lookup":
		return ui.globalLookupTab
	case "operations":
		return ui.operationTab
	case "documents":
		return ui.documentTab
	case "material":
		return ui.materialTab
	case "material_editor":
		return ui.materialEditorTab
	case "material_category":
		return ui.materialCategoryTab
	case "material_quote":
		return ui.materialQuoteTab
	case "material_quote_editor":
		return ui.materialQuoteEditorTab
	case "image_assets":
		return ui.imageAssetTab
	case "inventory":
		return ui.inventoryTab
	case "inbound":
		return ui.inboundTab
	case "inbound_editor":
		return ui.inboundEditorTab
	case "outbound":
		return ui.outboundTab
	case "outbound_editor":
		return ui.outboundEditorTab
	case "fast_outbound":
		return ui.fastOutboundTab
	case "outbound_report":
		return ui.outboundReportTab
	case "business_trace":
		return ui.businessTraceTab
	case "customer_finance":
		return ui.customerFinanceTab
	case "partner":
		return ui.partnerTab
	case "partner_editor":
		return ui.partnerEditorTab
	case "warehouse":
		return ui.warehouseTab
	case "warehouse_editor":
		return ui.warehouseEditorTab
	case "admin":
		return ui.adminTab
	case "profile":
		return ui.profileTab
	case "system":
		return ui.systemTab
	default:
		return nil
	}
}

func (ui *mainUI) keyForTab(page *walk.TabPage) string {
	switch page {
	case ui.dashboardTab:
		return "dashboard"
	case ui.globalLookupTab:
		return "global_lookup"
	case ui.operationTab:
		return "operations"
	case ui.documentTab:
		return "documents"
	case ui.materialTab:
		return "material"
	case ui.materialEditorTab:
		return "material_editor"
	case ui.materialCategoryTab:
		return "material_category"
	case ui.materialQuoteTab:
		return "material_quote"
	case ui.materialQuoteEditorTab:
		return "material_quote_editor"
	case ui.imageAssetTab:
		return "image_assets"
	case ui.inventoryTab:
		return "inventory"
	case ui.inboundTab:
		return "inbound"
	case ui.inboundEditorTab:
		return "inbound_editor"
	case ui.outboundTab:
		return "outbound"
	case ui.outboundEditorTab:
		return "outbound_editor"
	case ui.fastOutboundTab:
		return "fast_outbound"
	case ui.outboundReportTab:
		return "outbound_report"
	case ui.businessTraceTab:
		return "business_trace"
	case ui.customerFinanceTab:
		return "customer_finance"
	case ui.partnerTab:
		return "partner"
	case ui.partnerEditorTab:
		return "partner_editor"
	case ui.warehouseTab:
		return "warehouse"
	case ui.warehouseEditorTab:
		return "warehouse_editor"
	case ui.adminTab:
		return "admin"
	case ui.profileTab:
		return "profile"
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
	if ui.tabs == nil || ui.initializedPages[key] {
		return
	}
	page := ui.tabForKey(key)
	if page == nil || ui.tabs.Pages().Index(page) < 0 {
		return
	}
	ui.initializedPages[key] = true
	ui.registerPageLayouts(key)
	switch key {
	case "dashboard":
		ui.initializeDashboardPage()
	case "global_lookup":
		ui.initializeGlobalLookupPage()
	case "operations":
		ui.refreshOperationCenter()
	case "documents":
		ui.initializeDocumentCenterPage()
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
		ui.updateMaterialActionButtons()
		ui.loadMaterials()
	case "material_quote":
		ui.initializeMaterialQuotePage()
	case "image_assets":
		ui.initializeImageAssetPage()
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
	case "admin":
		ui.initializeAdminPage()
	case "profile":
		// Profile data belongs to the authenticated session and is rendered read-only.
	}
}

func (ui *mainUI) releasePage(key string) {
	delete(ui.initializedPages, key)
	switch key {
	case "dashboard":
		ui.releaseDashboardPage()
	case "global_lookup":
		ui.releaseGlobalLookupPage()
	case "operations":
		ui.operationTab = nil
		ui.operations = newOperationCenterUI()
	case "documents":
		ui.releaseDocumentCenterPage()
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
		ui.materialAdd = nil
		ui.materialEdit = nil
		ui.materialCategoryManage = nil
		ui.materialPrev = nil
		ui.materialNext = nil
		ui.materialSize = nil
		ui.materialRows = nil
		ui.materialPage = 1
		ui.materialTotal = 0
	case "material_quote":
		ui.releaseMaterialQuotePage()
	case "image_assets":
		ui.releaseImageAssetPage()
	case "inventory":
		ui.inventoryGeneration++
		if ui.inventoryCancel != nil {
			ui.inventoryCancel()
		}
		if ui.inventoryExportCancel != nil {
			ui.inventoryExportCancel()
		}
		ui.inventoryExportGeneration++
		ui.inventoryExportBusy = false
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
		ui.inventoryTrace = nil
		ui.inventoryExport = nil
		ui.inventoryExportStop = nil
		ui.inventoryZoneNodes = nil
		ui.inventoryRackNodes = nil
		ui.inventoryBinNodes = nil
		ui.inventoryPage = 1
		ui.inventoryTotal = 0
		ui.inventoryRows = nil
	case "inbound":
		ui.inboundGeneration++
		if ui.inboundCancel != nil {
			ui.inboundCancel()
		}
		if ui.inboundExportCancel != nil {
			ui.inboundExportCancel()
		}
		ui.inboundExportGeneration++
		ui.inboundExportBusy = false
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
		ui.inboundAdd = nil
		ui.inboundEdit = nil
		ui.inboundCheck = nil
		ui.inboundDelete = nil
		ui.inboundReceive = nil
		ui.inboundExport = nil
		ui.inboundExportStop = nil
		ui.inboundActionHint = nil
		ui.inboundSize = nil
		ui.inboundPage = 1
		ui.inboundTotal = 0
		ui.inboundRows = nil
	case "outbound":
		ui.releaseOutboundPage()
	case "outbound_report":
		ui.releaseOutboundReportPage()
	case "partner":
		ui.releasePartnerPage()
	case "warehouse":
		ui.releaseWarehousePage()
	case "admin":
		ui.releaseAdminPage()
	case "profile":
		if ui.profileAvatarCancel != nil {
			ui.profileAvatarCancel()
		}
		ui.profileTab = nil
		ui.profileAvatar = nil
		ui.profileAvatarStatus = nil
		ui.profileAvatarPreview = nil
		ui.profileAvatarChange = nil
		ui.profileAvatarBusy = false
		ui.profileAvatarCancelable = false
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

var inboundStatuses = []string{"全部状态", "待审核", "审核不通过", "审核通过", "未发货", "在途", "部分入库", "作废", "入库完成"}
var inboundTypes = []string{"全部类型", "采购入库", "外协入库", "生产入库", "退货入库"}

func (ui *mainUI) inboundPageWidget() TabPage {
	return TabPage{
		AssignTo: &ui.inboundTab,
		Title:    closableTabTitle("入库工作台"),
		Layout:   VBox{Margins: Margins{Left: 14, Top: 14, Right: 14, Bottom: 12}, Spacing: 10},
		Children: []Widget{
			Label{Text: "入库工作台", Font: Font{Family: "Microsoft YaHei UI", PointSize: 15, Bold: true}},
			Label{Text: "按 Web 端单据筛选条件查询，选中入库单后可查看详情、收货记录或执行批次收货。", TextColor: secondaryTextColor()},
			GroupBox{
				Title:  "筛选条件",
				Layout: Grid{Columns: 8, Margins: Margins{Left: 14, Top: 12, Right: 14, Bottom: 14}, Spacing: 8},
				Children: []Widget{
					Label{Text: "入库单号"},
					LineEdit{AssignTo: &ui.inboundSearch, MinSize: Size{Width: 180, Height: 28}, ToolTipText: "现有接口按完整单号查询", Accessibility: Accessibility{Name: "入库单号筛选"}},
					Label{Text: "状态"},
					ComboBox{AssignTo: &ui.inboundStatus, Model: inboundStatuses, CurrentIndex: ui.initialWorkspaceFilterIndex("inbound_status", len(inboundStatuses)), MinSize: Size{Width: 130, Height: 28}, Accessibility: Accessibility{Name: "入库状态筛选"}},
					Label{Text: "类型"},
					ComboBox{AssignTo: &ui.inboundType, Model: inboundTypes, CurrentIndex: ui.initialWorkspaceFilterIndex("inbound_type", len(inboundTypes)), MinSize: Size{Width: 130, Height: 28}, Accessibility: Accessibility{Name: "入库类型筛选"}},
					Label{Text: "供应商"},
					ComboBox{AssignTo: &ui.inboundSupplier, Model: []string{"全部供应商（正在加载）"}, CurrentIndex: 0, MinSize: Size{Width: 190, Height: 28}, Accessibility: Accessibility{Name: "入库供应商筛选"}},
					Label{Text: "客户"},
					ComboBox{
						AssignTo: &ui.inboundCustomer, Model: []string{"全部客户（正在加载）"}, CurrentIndex: 0,
						MinSize: Size{Width: 190, Height: 28}, ToolTipText: "退货入库按客户筛选；全部类型时也可单独使用", Accessibility: Accessibility{Name: "入库客户筛选"},
					},
					HSpacer{ColumnSpan: 6},
					Composite{
						ColumnSpan: 8,
						Layout:     HBox{Spacing: 8},
						Children: []Widget{
							HSpacer{},
							PushButton{AssignTo: &ui.inboundReset, Text: "重置", MinSize: Size{Width: 88, Height: 30}, Accessibility: Accessibility{Name: "重置入库筛选条件"}, OnClicked: ui.resetInboundFilters},
							PushButton{AssignTo: &ui.inboundQuery, Text: "查询", MinSize: Size{Width: 96, Height: 30}, Accessibility: Accessibility{Name: "查询入库单"}, OnClicked: func() {
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
				ColumnsOrderable: true,
				StretchFactor:    1,
				Accessibility:    Accessibility{Name: "入库单查询结果表格", Description: "选择后可查看详情或执行有权限的入库操作"},
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
				OnCurrentIndexChanged: ui.updateInboundActionButtons,
				OnItemActivated:       ui.showInboundDetail,
			},
			Label{
				AssignTo: &ui.inboundActionHint, Text: "选择一张入库单后，可按权限和当前状态执行操作。",
				TextColor: secondaryTextColor(), Accessibility: Accessibility{Name: "入库单操作提示"},
			},
			Composite{Layout: VBox{Spacing: 6}, Children: []Widget{
				Composite{Layout: HBox{Spacing: 6}, Children: []Widget{
					Label{Text: "当前单据操作", Font: Font{Bold: true}},
					PushButton{AssignTo: &ui.inboundAdd, Text: "新建", Enabled: hasButton(ui.session.Perms.Buttons, "inbound:receipt:add"), Accessibility: Accessibility{Name: "新建入库单"}, OnClicked: ui.newInboundReceipt},
					PushButton{AssignTo: &ui.inboundEdit, Text: "编辑", Enabled: false, Accessibility: Accessibility{Name: "编辑选中的入库单"}, OnClicked: ui.editSelectedInbound},
					PushButton{AssignTo: &ui.inboundCheck, Text: "审核", Enabled: false, Accessibility: Accessibility{Name: "审核选中的入库单"}, OnClicked: ui.checkSelectedInbound},
					PushButton{AssignTo: &ui.inboundDelete, Text: "删除", Enabled: false, Accessibility: Accessibility{Name: "删除选中的入库单"}, OnClicked: ui.deleteSelectedInbound},
					PushButton{Text: "详情/收货记录", Accessibility: Accessibility{Name: "查看选中入库单详情和收货记录"}, OnClicked: ui.showInboundDetail},
					PushButton{AssignTo: &ui.inboundReceive, Text: "批次收货", Enabled: false, Accessibility: Accessibility{Name: "对选中的入库单执行批次收货"}, OnClicked: ui.receiveSelectedInbound},
					HSpacer{},
					PushButton{
						AssignTo: &ui.inboundExport, Text: "导出当前筛选", MinSize: Size{Width: 108, Height: 30},
						Accessibility: Accessibility{Name: "导出当前入库筛选的全部结果到 Excel"}, OnClicked: ui.exportInboundQuery,
					},
					PushButton{
						AssignTo: &ui.inboundExportStop, Text: "取消导出", Enabled: false, MinSize: Size{Width: 88, Height: 30},
						Accessibility: Accessibility{Name: "取消当前入库导出并清理未完成文件"}, OnClicked: ui.cancelInboundExport,
					},
				}},
				Composite{Layout: HBox{Spacing: 6}, Children: []Widget{
					Label{AssignTo: &ui.inboundInfo, Text: "尚未加载", Accessibility: Accessibility{Name: "入库查询分页状态"}},
					HSpacer{},
					Label{Text: "每页"},
					ComboBox{AssignTo: &ui.inboundSize, Model: pageSizeLabels, CurrentIndex: ui.initialPageSizeIndex("inbound", len(pageSizeLabels)), MinSize: Size{Width: 92}, Accessibility: Accessibility{Name: "入库结果每页数量"}, OnCurrentIndexChanged: func() {
						if ui.window != nil && ui.inboundSize != nil && ui.inboundSize.CurrentIndex() >= 0 {
							ui.inboundPage = 1
							ui.loadInbound()
						}
					}},
					PushButton{AssignTo: &ui.inboundPrev, Text: "上一页", Accessibility: Accessibility{Name: "上一页入库结果"}, OnClicked: func() {
						if ui.inboundPage > 1 {
							ui.inboundPage--
							ui.loadInbound()
						}
					}},
					PushButton{AssignTo: &ui.inboundNext, Text: "下一页", Accessibility: Accessibility{Name: "下一页入库结果"}, OnClicked: func() { ui.inboundPage++; ui.loadInbound() }},
					PushButton{Text: "跳转页", Accessibility: Accessibility{Name: "跳转到指定入库结果页"}, OnClicked: func() {
						if page, ok := promptPageNumber(ui.window, ui.inboundPage, ui.inboundTotal, selectedPageSize(ui.inboundSize)); ok {
							ui.inboundPage = page
							ui.loadInbound()
						}
					}},
				}},
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
	if ui.inboundExport != nil {
		ui.inboundExport.SetEnabled(false)
	}
	ui.setInboundOperationBusy(true)
	guardedGo(func() {
		result, err := ui.session.Client.InboundReceipts(ctx, page, size, filters)
		if ctx.Err() != nil {
			return
		}
		ui.window.Synchronize(func() {
			if generation != ui.inboundGeneration || ui.inboundTable == nil {
				return
			}
			ui.setInboundOperationBusy(false)
			ui.inboundQuery.SetEnabled(true)
			ui.inboundReset.SetEnabled(true)
			if ui.inboundExport != nil {
				ui.inboundExport.SetEnabled(!ui.inboundExportBusy)
			}
			if err != nil {
				ui.inboundInfo.SetText(requestFailureText(err))
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
			ui.inboundTotal = result.Total
			ui.updateInboundActionButtons()
			ui.inboundInfo.SetText(fmt.Sprintf("第 %d 页 | 本页 %d 条 | 共 %d 条", page, len(rows), result.Total))
			ui.inboundPrev.SetEnabled(page > 1)
			ui.inboundNext.SetEnabled(int64(page*size) < result.Total)
		})
	})
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
	if !canReceiveInbound(receipt.Status) {
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
		Title:    closableTabTitle("物料中心"),
		Layout:   VBox{Margins: Margins{Left: 14, Top: 14, Right: 14, Bottom: 12}, Spacing: 10},
		Children: []Widget{
			Label{Text: "物料中心", Font: Font{Family: "Microsoft YaHei UI", PointSize: 15, Bold: true}},
			Label{Text: "查询和维护物料资料；价格变更继续通过现有报价与核价流程完成。", TextColor: secondaryTextColor()},
			GroupBox{
				Title:  "筛选条件",
				Layout: Grid{Columns: 8, Margins: Margins{Left: 14, Top: 12, Right: 14, Bottom: 14}, Spacing: 8},
				Children: []Widget{
					Label{Text: "物料分类"},
					ComboBox{AssignTo: &ui.materialCategory, Model: []string{"全部分类（正在加载）"}, CurrentIndex: 0, MinSize: Size{Width: 180, Height: 28}, Accessibility: Accessibility{Name: "物料分类筛选"}},
					Label{Text: "物料名称"},
					LineEdit{AssignTo: &ui.materialName, MinSize: Size{Width: 180, Height: 28}, Accessibility: Accessibility{Name: "物料名称筛选"}},
					Label{Text: "型号"},
					LineEdit{AssignTo: &ui.materialModel, MinSize: Size{Width: 180, Height: 28}, Accessibility: Accessibility{Name: "物料型号筛选"}},
					Label{Text: "规格"},
					LineEdit{AssignTo: &ui.materialSpecification, MinSize: Size{Width: 180, Height: 28}, Accessibility: Accessibility{Name: "物料规格筛选"}},
					Label{Text: "材质"},
					LineEdit{AssignTo: &ui.materialMaterial, MinSize: Size{Width: 180, Height: 28}, Accessibility: Accessibility{Name: "物料材质筛选"}},
					Label{Text: "表面处理"},
					LineEdit{AssignTo: &ui.materialSurfaceTreatment, MinSize: Size{Width: 180, Height: 28}, Accessibility: Accessibility{Name: "物料表面处理筛选"}},
					Label{Text: "强度等级"},
					LineEdit{AssignTo: &ui.materialStrengthGrade, MinSize: Size{Width: 180, Height: 28}, Accessibility: Accessibility{Name: "物料强度等级筛选"}},
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
				TextColor:     secondaryTextColor(),
				Accessibility: Accessibility{Name: "物料详情操作提示"},
			},
			Composite{Layout: HBox{Spacing: 8}, Children: []Widget{
				PushButton{AssignTo: &ui.materialAdd, Text: "新增物料", Visible: hasButton(ui.session.Perms.Buttons, "material:material:add"), MinSize: Size{Width: 92, Height: 30}, OnClicked: ui.newMaterial},
				PushButton{AssignTo: &ui.materialEdit, Text: "编辑当前物料", Visible: hasButton(ui.session.Perms.Buttons, "material:material:edit"), Enabled: false, MinSize: Size{Width: 112, Height: 30}, OnClicked: ui.editSelectedMaterial},
				PushButton{AssignTo: &ui.materialCategoryManage, Text: "物料分类", Visible: hasMenuPath(ui.session.Perms.Menus, "/material/category") && hasButton(ui.session.Perms.Buttons, "material:category:list"), MinSize: Size{Width: 92, Height: 30}, OnClicked: ui.openMaterialCategoryPage},
				HSpacer{},
				Label{Text: "不提供物料、分类和价格删除。", TextColor: secondaryTextColor()},
			}},
			TableView{
				AssignTo:              &ui.materialTable,
				AlternatingRowBG:      true,
				ColumnsOrderable:      true,
				StretchFactor:         1,
				ToolTipText:           "双击物料行，或选中后按 Enter，查看图纸与物料信息",
				Accessibility:         Accessibility{Name: "物料查询结果表格", Description: "包含图纸状态；激活当前行可查看详情"},
				OnCurrentIndexChanged: ui.updateMaterialActionButtons,
				OnItemActivated:       ui.showMaterialDetail,
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
					ComboBox{AssignTo: &ui.materialSize, Model: pageSizeLabels, CurrentIndex: ui.initialPageSizeIndex("material", len(pageSizeLabels)), MinSize: Size{Width: 92}, OnCurrentIndexChanged: func() {
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
					PushButton{Text: "跳转页", Accessibility: Accessibility{Name: "跳转到指定物料结果页"}, OnClicked: func() {
						if page, ok := promptPageNumber(ui.window, ui.materialPage, ui.materialTotal, selectedPageSize(ui.materialSize)); ok {
							ui.materialPage = page
							ui.loadMaterials()
						}
					}},
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
			Label{Text: "可切换当前库存与库存批次历史；后者展示已有库存批次，不表示独立库存流水。", TextColor: secondaryTextColor()},
			GroupBox{
				Title:  "筛选条件",
				Layout: Grid{Columns: 8, Margins: Margins{Left: 14, Top: 12, Right: 14, Bottom: 14}, Spacing: 8},
				Children: []Widget{
					Label{Text: "库存视图"},
					ComboBox{AssignTo: &ui.inventoryMode, Model: modes, CurrentIndex: ui.initialWorkspaceFilterIndex("inventory_mode", len(modes)), MinSize: Size{Width: 135, Height: 28}, Accessibility: Accessibility{Name: "库存视图筛选"}},
					Label{Text: "入库类型"},
					ComboBox{AssignTo: &ui.inventoryType, Model: types, CurrentIndex: ui.initialWorkspaceFilterIndex("inventory_type", len(types)), MinSize: Size{Width: 135, Height: 28}, Accessibility: Accessibility{Name: "库存入库类型筛选"}},
					Label{Text: "物料名称"},
					LineEdit{AssignTo: &ui.inventoryName, MinSize: Size{Width: 155, Height: 28}, Accessibility: Accessibility{Name: "库存物料名称筛选"}},
					Label{Text: "物料型号"},
					LineEdit{AssignTo: &ui.inventoryModel, MinSize: Size{Width: 145, Height: 28}, Accessibility: Accessibility{Name: "库存物料型号筛选"}},
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
				AssignTo:              &ui.inventoryTable,
				AlternatingRowBG:      true,
				ColumnsOrderable:      true,
				StretchFactor:         1,
				OnCurrentIndexChanged: ui.updateInventoryTraceButton,
				OnItemActivated:       ui.showSelectedInventorySource,
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
					PushButton{
						AssignTo: &ui.inventoryTrace, Text: "追溯入库来源", Enabled: false,
						MinSize: Size{Width: 108, Height: 30}, Accessibility: Accessibility{Name: "查看选中库存的来源入库单和批次"},
						OnClicked: ui.showSelectedInventorySource,
					},
					PushButton{
						AssignTo: &ui.inventoryExport, Text: "导出当前筛选", MinSize: Size{Width: 108, Height: 30},
						Accessibility: Accessibility{Name: "导出当前库存筛选的全部结果到 Excel"}, OnClicked: ui.exportInventoryQuery,
					},
					PushButton{
						AssignTo: &ui.inventoryExportStop, Text: "取消导出", Enabled: false, MinSize: Size{Width: 88, Height: 30},
						Accessibility: Accessibility{Name: "取消当前库存导出并清理未完成文件"}, OnClicked: ui.cancelInventoryExport,
					},
					Label{Text: "每页"},
					ComboBox{AssignTo: &ui.inventorySize, Model: pageSizeLabels, CurrentIndex: ui.initialPageSizeIndex("inventory", len(pageSizeLabels)), MinSize: Size{Width: 92}, OnCurrentIndexChanged: func() {
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
					PushButton{Text: "跳转页", Accessibility: Accessibility{Name: "跳转到指定库存结果页"}, OnClicked: func() {
						if page, ok := promptPageNumber(ui.window, ui.inventoryPage, ui.inventoryTotal, selectedPageSize(ui.inventorySize)); ok {
							ui.inventoryPage = page
							ui.loadInventory()
						}
					}},
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
	guardedGo(func() {
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
				ui.materialInfo.SetText(requestFailureText(err))
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
	})
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
	ui.inventoryRows = nil
	ui.updateInventoryTraceButton()
	if ui.inventoryExport != nil {
		ui.inventoryExport.SetEnabled(false)
	}
	historyMode := ui.inventoryMode != nil && ui.inventoryMode.CurrentIndex() == 1
	guardedGo(func() {
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
			if ui.inventoryExport != nil {
				ui.inventoryExport.SetEnabled(!ui.inventoryExportBusy)
			}
			if err != nil {
				ui.inventoryInfo.SetText(requestFailureText(err))
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
					Detail:    item,
				})
			}
			if modelErr := ui.inventoryTable.SetModel(rows); modelErr != nil {
				ui.inventoryInfo.SetText("表格数据错误：" + modelErr.Error())
				return
			}
			ui.inventoryTotal = result.Total
			ui.inventoryRows = rows
			ui.updateInventoryTraceButton()
			modeLabel := "当前库存"
			if historyMode {
				modeLabel = "库存批次历史"
			}
			ui.inventoryInfo.SetText(fmt.Sprintf("%s | 第 %d 页 | 本页 %d 条 | 共 %d 条 | 可用合计 %g",
				modeLabel, page, len(rows), result.Total, result.Quantity))
			ui.inventoryPrev.SetEnabled(page > 1)
			ui.inventoryNext.SetEnabled(int64(page*size) < result.Total)
		})
	})
}

func (ui *mainUI) loadFilterOptions() {
	loadCategories := ui.materialCategory != nil && (!ui.categoryReady || ui.categoryFailed) && !ui.categoryLoading
	loadWarehouses := ui.inventoryWarehouse != nil && (!ui.warehouseReady || ui.warehouseFailed) && !ui.warehouseLoading
	loadSuppliers := ui.inboundSupplier != nil && (!ui.supplierReady || ui.supplierFailed) && !ui.supplierLoading
	loadCustomers := ui.inboundCustomer != nil && (!ui.customerReady || ui.customerFailed) && !ui.customerLoading
	if !loadCategories && !loadWarehouses && !loadSuppliers && !loadCustomers {
		return
	}
	ui.categoryLoading = ui.categoryLoading || loadCategories
	ui.warehouseLoading = ui.warehouseLoading || loadWarehouses
	ui.supplierLoading = ui.supplierLoading || loadSuppliers
	ui.customerLoading = ui.customerLoading || loadCustomers
	guardedGo(func() {
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
				ui.categoryLoading = false
				ui.categoryReady = true
				ui.categoryFailed = !categoryOK
				if categoryOK {
					ui.materialCategoryOptions = flattenCategoryOptions(categories, "")
				}
				ui.applyMaterialCategoryOptions()
			}
			if loadWarehouses {
				ui.warehouseLoading = false
				ui.warehouseReady = true
				ui.warehouseFailed = !warehouseOK
				if warehouseOK {
					ui.inventoryWarehouseNodes = warehouses
				}
				ui.applyWarehouseOptions()
			}
			if loadSuppliers {
				ui.supplierLoading = false
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
				ui.customerLoading = false
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
				ui.status.SetText(fmt.Sprintf("已连接线上 API · 当前页面筛选选项已就绪 · v%s", clientVersion))
				ui.status.SetTextColor(successTextColor())
			} else {
				ui.status.SetText("已连接线上 API · 以下筛选选项加载失败，可在当前页刷新重试：" + strings.Join(errorLabels, "、"))
				ui.status.SetTextColor(warningTextColor())
			}
		})
	})
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
