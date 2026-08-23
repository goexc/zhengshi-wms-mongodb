package ui

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"

	"zhengshi-wms-windowsapp/internal/api"
)

var adminMenuPaths = []string{"/acl/user", "/acl/department", "/acl/role", "/acl/menu", "/acl/api"}

type adminUserRow struct {
	Name       string
	Mobile     string
	Email      string
	Department string
	Roles      string
	Status     string
	Remark     string
	Updated    string
}

type adminDepartmentRow struct {
	Name   string
	Code   string
	Parent string
	Sort   string
	Remark string
}

type adminRoleRow struct {
	Name    string
	Status  string
	Parent  string
	Remark  string
	Updated string
}

type adminMenuRow struct {
	Title string
	Kind  string
	Path  string
	Perms string
	Sort  string
}

type adminAPIRow struct {
	Name   string
	Kind   string
	Method string
	URI    string
	Sort   string
}

type adminCatalogOption struct {
	ID    string
	Label string
}

type adminUI struct {
	tab       *walk.TabPage
	innerTabs *walk.TabWidget
	info      *walk.Label
	refresh   *walk.PushButton

	usersPage  *walk.TabPage
	userName   *walk.LineEdit
	userMobile *walk.LineEdit
	userTable  *walk.TableView
	userQuery  *walk.PushButton
	userReset  *walk.PushButton
	userAdd    *walk.PushButton
	userEdit   *walk.PushButton
	userStatus *walk.PushButton
	userRoles  *walk.PushButton
	userPass   *walk.PushButton
	userPrev   *walk.PushButton
	userNext   *walk.PushButton
	userSize   *walk.ComboBox
	users      []api.AdminUser
	userPageNo int
	userTotal  int64
	userCancel context.CancelFunc

	departmentsPage *walk.TabPage
	departmentTable *walk.TableView
	departmentAdd   *walk.PushButton
	departmentEdit  *walk.PushButton
	departments     []api.AdminDepartment
	departmentRows  []adminDepartmentRow

	rolesPage  *walk.TabPage
	roleName   *walk.LineEdit
	roleTable  *walk.TableView
	roleQuery  *walk.PushButton
	roleReset  *walk.PushButton
	roleAdd    *walk.PushButton
	roleEdit   *walk.PushButton
	roleState  *walk.PushButton
	roleMenus  *walk.PushButton
	roleAPIs   *walk.PushButton
	rolePrev   *walk.PushButton
	roleNext   *walk.PushButton
	roleSize   *walk.ComboBox
	roles      []api.AdminRole
	roleList   []api.AdminRole
	rolePageNo int
	roleTotal  int64
	roleCancel context.CancelFunc

	catalogPage *walk.TabPage
	menuTable   *walk.TableView
	apiTable    *walk.TableView
	menus       []api.AdminMenu
	apis        []api.AdminAPI

	generation          int
	referenceGeneration int
	busy                bool
	initialized         bool
	ctx                 context.Context
	cancel              context.CancelFunc
	closed              atomic.Bool
}

func newAdminUI() *adminUI {
	ctx, cancel := context.WithCancel(context.Background())
	return &adminUI{userPageNo: 1, rolePageNo: 1, ctx: ctx, cancel: cancel}
}

func (state *adminUI) dispose() {
	if state == nil || state.closed.Swap(true) {
		return
	}
	state.generation++
	state.referenceGeneration++
	if state.userCancel != nil {
		state.userCancel()
	}
	if state.roleCancel != nil {
		state.roleCancel()
	}
	state.cancel()
}

func adminMenuAvailable(perms api.Perms) bool {
	return hasAnyMenuPath(perms.Menus, adminMenuPaths...)
}

func (ui *mainUI) adminPageWidget() TabPage {
	if ui.admin == nil {
		ui.admin = newAdminUI()
	}
	state := ui.admin
	pages := make([]TabPage, 0, 4)
	if hasMenuPath(ui.session.Perms.Menus, "/acl/user") {
		pages = append(pages, ui.adminUsersPageWidget(state))
	}
	if hasMenuPath(ui.session.Perms.Menus, "/acl/department") {
		pages = append(pages, ui.adminDepartmentsPageWidget(state))
	}
	if hasMenuPath(ui.session.Perms.Menus, "/acl/role") {
		pages = append(pages, ui.adminRolesPageWidget(state))
	}
	if hasAnyMenuPath(ui.session.Perms.Menus, "/acl/menu", "/acl/api") {
		pages = append(pages, ui.adminCatalogPageWidget(state))
	}
	return TabPage{
		AssignTo: &ui.adminTab,
		Title:    closableTabTitle("系统管理"),
		Layout:   VBox{Margins: Margins{Left: 14, Top: 14, Right: 14, Bottom: 12}, Spacing: 8},
		Children: []Widget{
			Composite{Layout: HBox{Spacing: 8}, Children: []Widget{
				Label{Text: "系统管理", Font: Font{Family: "Microsoft YaHei UI", PointSize: 15, Bold: true}},
				Label{Text: "页面入口由账号菜单权限决定，所有读写仍由服务端 API 权限最终校验。", TextColor: secondaryTextColor()},
				HSpacer{},
				PushButton{AssignTo: &state.refresh, Text: "刷新全部", MinSize: Size{Width: 92, Height: 30}, OnClicked: ui.refreshAdminPage, Accessibility: Accessibility{Name: "刷新系统管理全部数据"}},
			}},
			TabWidget{AssignTo: &state.innerTabs, Pages: pages, StretchFactor: 1},
			Label{AssignTo: &state.info, Text: "尚未加载系统管理数据。", TextColor: secondaryTextColor(), Accessibility: Accessibility{Name: "系统管理状态"}},
		},
	}
}

func (ui *mainUI) adminUsersPageWidget(state *adminUI) TabPage {
	return TabPage{
		AssignTo: &state.usersPage,
		Title:    "用户",
		Layout:   VBox{Margins: Margins{Left: 12, Top: 12, Right: 12, Bottom: 10}, Spacing: 8},
		Children: []Widget{
			GroupBox{Title: "筛选条件", Layout: Grid{Columns: 8, Margins: Margins{Left: 12, Top: 8, Right: 12, Bottom: 10}, Spacing: 8}, Children: []Widget{
				Label{Text: "用户名称"},
				LineEdit{AssignTo: &state.userName, CueBanner: "按名称查询", MinSize: Size{Width: 180, Height: 28}, Accessibility: Accessibility{Name: "用户名称筛选"}},
				Label{Text: "手机号码"},
				LineEdit{AssignTo: &state.userMobile, CueBanner: "按手机号码查询", MinSize: Size{Width: 180, Height: 28}, Accessibility: Accessibility{Name: "用户手机号码筛选"}},
				HSpacer{ColumnSpan: 2},
				PushButton{AssignTo: &state.userReset, Text: "重置", MinSize: Size{Width: 82, Height: 30}, OnClicked: ui.resetAdminUsers},
				PushButton{AssignTo: &state.userQuery, Text: "查询", MinSize: Size{Width: 88, Height: 30}, OnClicked: func() { state.userPageNo = 1; ui.loadAdminUsers() }},
			}},
			Composite{Layout: HBox{Spacing: 8}, Children: []Widget{
				PushButton{AssignTo: &state.userAdd, Text: "新增用户", MinSize: Size{Width: 92, Height: 30}, OnClicked: ui.addAdminUser},
				PushButton{AssignTo: &state.userEdit, Text: "编辑用户", Enabled: false, MinSize: Size{Width: 92, Height: 30}, OnClicked: ui.editSelectedAdminUser},
				PushButton{AssignTo: &state.userRoles, Text: "分配角色", Enabled: false, MinSize: Size{Width: 92, Height: 30}, OnClicked: ui.assignSelectedAdminUserRoles},
				PushButton{AssignTo: &state.userStatus, Text: "启用/禁用", Enabled: false, MinSize: Size{Width: 92, Height: 30}, OnClicked: ui.toggleSelectedAdminUserStatus},
				PushButton{AssignTo: &state.userPass, Text: "重置密码", Enabled: false, MinSize: Size{Width: 92, Height: 30}, OnClicked: ui.resetSelectedAdminUserPassword},
				Label{Text: "客户端不提供删除用户。", TextColor: secondaryTextColor()},
				HSpacer{},
			}},
			TableView{
				AssignTo: &state.userTable, AlternatingRowBG: true, ColumnsOrderable: true, StretchFactor: 1,
				Accessibility: Accessibility{Name: "系统用户列表"}, OnCurrentIndexChanged: ui.updateAdminUserActions, OnItemActivated: ui.editSelectedAdminUser,
				Columns: []TableViewColumn{
					{Title: "用户名称", DataMember: "Name", Width: 125}, {Title: "手机号码", DataMember: "Mobile", Width: 130},
					{Title: "Email", DataMember: "Email", Width: 180}, {Title: "部门", DataMember: "Department", Width: 130},
					{Title: "角色", DataMember: "Roles", Width: 180}, {Title: "状态", DataMember: "Status", Width: 75},
					{Title: "备注", DataMember: "Remark", Width: 160}, {Title: "更新时间", DataMember: "Updated", Width: 145},
				},
			},
			Composite{Layout: HBox{Spacing: 8}, Children: []Widget{
				Label{Text: "每页"},
				ComboBox{AssignTo: &state.userSize, Model: pageSizeLabels, CurrentIndex: 1, MinSize: Size{Width: 92}, OnCurrentIndexChanged: func() {
					if state.userSize != nil && state.initialized {
						state.userPageNo = 1
						ui.loadAdminUsers()
					}
				}},
				PushButton{AssignTo: &state.userPrev, Text: "上一页", Enabled: false, OnClicked: func() {
					if state.userPageNo > 1 {
						state.userPageNo--
						ui.loadAdminUsers()
					}
				}},
				PushButton{AssignTo: &state.userNext, Text: "下一页", Enabled: false, OnClicked: func() { state.userPageNo++; ui.loadAdminUsers() }},
				HSpacer{},
			}},
		},
	}
}

func (ui *mainUI) adminDepartmentsPageWidget(state *adminUI) TabPage {
	return TabPage{
		AssignTo: &state.departmentsPage,
		Title:    "部门",
		Layout:   VBox{Margins: Margins{Left: 12, Top: 12, Right: 12, Bottom: 10}, Spacing: 8},
		Children: []Widget{
			Composite{Layout: HBox{Spacing: 8}, Children: []Widget{
				Label{Text: "部门层级", Font: Font{Bold: true}},
				Label{Text: "仅新增和编辑，不提供删除。", TextColor: secondaryTextColor()},
				HSpacer{},
				PushButton{AssignTo: &state.departmentAdd, Text: "新增部门", MinSize: Size{Width: 92, Height: 30}, OnClicked: ui.addAdminDepartment},
				PushButton{AssignTo: &state.departmentEdit, Text: "编辑部门", Enabled: false, MinSize: Size{Width: 92, Height: 30}, OnClicked: ui.editSelectedAdminDepartment},
			}},
			TableView{
				AssignTo: &state.departmentTable, AlternatingRowBG: true, ColumnsOrderable: true, StretchFactor: 1,
				Accessibility: Accessibility{Name: "部门层级列表"}, OnCurrentIndexChanged: ui.updateAdminDepartmentActions, OnItemActivated: ui.editSelectedAdminDepartment,
				Columns: []TableViewColumn{
					{Title: "部门名称", DataMember: "Name", Width: 260}, {Title: "部门编码", DataMember: "Code", Width: 160},
					{Title: "上级部门", DataMember: "Parent", Width: 180}, {Title: "排序", DataMember: "Sort", Width: 75},
					{Title: "备注", DataMember: "Remark", Width: 260},
				},
			},
		},
	}
}

func (ui *mainUI) adminRolesPageWidget(state *adminUI) TabPage {
	return TabPage{
		AssignTo: &state.rolesPage,
		Title:    "角色与权限",
		Layout:   VBox{Margins: Margins{Left: 12, Top: 12, Right: 12, Bottom: 10}, Spacing: 8},
		Children: []Widget{
			GroupBox{Title: "筛选条件", Layout: Grid{Columns: 8, Margins: Margins{Left: 12, Top: 8, Right: 12, Bottom: 10}, Spacing: 8}, Children: []Widget{
				Label{Text: "角色名称"},
				LineEdit{AssignTo: &state.roleName, CueBanner: "按角色名称查询", MinSize: Size{Width: 180, Height: 28}, Accessibility: Accessibility{Name: "角色名称筛选"}},
				HSpacer{ColumnSpan: 4},
				PushButton{AssignTo: &state.roleReset, Text: "重置", MinSize: Size{Width: 82, Height: 30}, OnClicked: ui.resetAdminRoles},
				PushButton{AssignTo: &state.roleQuery, Text: "查询", MinSize: Size{Width: 88, Height: 30}, OnClicked: func() { state.rolePageNo = 1; ui.loadAdminRoles() }},
			}},
			Composite{Layout: HBox{Spacing: 8}, Children: []Widget{
				PushButton{AssignTo: &state.roleAdd, Text: "新增角色", MinSize: Size{Width: 92, Height: 30}, OnClicked: ui.addAdminRole},
				PushButton{AssignTo: &state.roleEdit, Text: "编辑角色", Enabled: false, MinSize: Size{Width: 92, Height: 30}, OnClicked: ui.editSelectedAdminRole},
				PushButton{AssignTo: &state.roleState, Text: "启用/禁用", Enabled: false, MinSize: Size{Width: 92, Height: 30}, OnClicked: ui.toggleSelectedAdminRoleStatus},
				PushButton{AssignTo: &state.roleMenus, Text: "分配菜单", Enabled: false, MinSize: Size{Width: 92, Height: 30}, OnClicked: ui.assignSelectedAdminRoleMenus},
				PushButton{AssignTo: &state.roleAPIs, Text: "分配 API", Enabled: false, MinSize: Size{Width: 92, Height: 30}, OnClicked: ui.assignSelectedAdminRoleAPIs},
				Label{Text: "客户端不提供删除角色。", TextColor: secondaryTextColor()},
				HSpacer{},
			}},
			TableView{
				AssignTo: &state.roleTable, AlternatingRowBG: true, ColumnsOrderable: true, StretchFactor: 1,
				Accessibility: Accessibility{Name: "系统角色列表"}, OnCurrentIndexChanged: ui.updateAdminRoleActions, OnItemActivated: ui.editSelectedAdminRole,
				Columns: []TableViewColumn{
					{Title: "角色名称", DataMember: "Name", Width: 180}, {Title: "状态", DataMember: "Status", Width: 80},
					{Title: "上级角色", DataMember: "Parent", Width: 180}, {Title: "备注", DataMember: "Remark", Width: 280},
					{Title: "更新时间", DataMember: "Updated", Width: 145},
				},
			},
			Composite{Layout: HBox{Spacing: 8}, Children: []Widget{
				Label{Text: "每页"},
				ComboBox{AssignTo: &state.roleSize, Model: pageSizeLabels, CurrentIndex: 1, MinSize: Size{Width: 92}, OnCurrentIndexChanged: func() {
					if state.roleSize != nil && state.initialized {
						state.rolePageNo = 1
						ui.loadAdminRoles()
					}
				}},
				PushButton{AssignTo: &state.rolePrev, Text: "上一页", Enabled: false, OnClicked: func() {
					if state.rolePageNo > 1 {
						state.rolePageNo--
						ui.loadAdminRoles()
					}
				}},
				PushButton{AssignTo: &state.roleNext, Text: "下一页", Enabled: false, OnClicked: func() { state.rolePageNo++; ui.loadAdminRoles() }},
				HSpacer{},
			}},
		},
	}
}

func (ui *mainUI) adminCatalogPageWidget(state *adminUI) TabPage {
	catalogs := make([]Widget, 0, 2)
	if hasMenuPath(ui.session.Perms.Menus, "/acl/menu") {
		catalogs = append(catalogs, GroupBox{Title: "菜单目录", StretchFactor: 1, Layout: VBox{Margins: Margins{Left: 8, Top: 8, Right: 8, Bottom: 8}}, Children: []Widget{
			TableView{AssignTo: &state.menuTable, AlternatingRowBG: true, ColumnsOrderable: true, StretchFactor: 1, Accessibility: Accessibility{Name: "只读菜单目录"}, Columns: []TableViewColumn{
				{Title: "名称", DataMember: "Title", Width: 240}, {Title: "类型", DataMember: "Kind", Width: 75}, {Title: "路径", DataMember: "Path", Width: 220}, {Title: "权限标识", DataMember: "Perms", Width: 240}, {Title: "排序", DataMember: "Sort", Width: 70},
			}},
		}})
	}
	if hasMenuPath(ui.session.Perms.Menus, "/acl/api") {
		catalogs = append(catalogs, GroupBox{Title: "API 目录", StretchFactor: 1, Layout: VBox{Margins: Margins{Left: 8, Top: 8, Right: 8, Bottom: 8}}, Children: []Widget{
			TableView{AssignTo: &state.apiTable, AlternatingRowBG: true, ColumnsOrderable: true, StretchFactor: 1, Accessibility: Accessibility{Name: "只读 API 目录"}, Columns: []TableViewColumn{
				{Title: "名称", DataMember: "Name", Width: 240}, {Title: "类型", DataMember: "Kind", Width: 75}, {Title: "方法", DataMember: "Method", Width: 80}, {Title: "URI", DataMember: "URI", Width: 340}, {Title: "排序", DataMember: "Sort", Width: 70},
			}},
		}})
	}
	return TabPage{
		AssignTo: &state.catalogPage,
		Title:    "权限目录（只读）",
		Layout:   VBox{Margins: Margins{Left: 12, Top: 12, Right: 12, Bottom: 10}, Spacing: 8},
		Children: []Widget{
			Label{Text: "Windows 客户端只读取菜单与 API 目录，用于角色授权；不提供目录增删改。", TextColor: secondaryTextColor()},
			VSplitter{HandleWidth: 4, StretchFactor: 1, Children: catalogs},
		},
	}
}

func (ui *mainUI) initializeAdminPage() {
	state := ui.admin
	if state == nil || ui.adminTab == nil || state.initialized {
		return
	}
	state.initialized = true
	state.generation++
	if state.userName != nil {
		state.userName.KeyDown().Attach(func(key walk.Key) {
			if key == walk.KeyReturn {
				state.userPageNo = 1
				ui.loadAdminUsers()
			}
		})
		state.userMobile.KeyDown().Attach(func(key walk.Key) {
			if key == walk.KeyReturn {
				state.userPageNo = 1
				ui.loadAdminUsers()
			}
		})
	}
	if state.roleName != nil {
		state.roleName.KeyDown().Attach(func(key walk.Key) {
			if key == walk.KeyReturn {
				state.rolePageNo = 1
				ui.loadAdminRoles()
			}
		})
	}
	ui.refreshAdminPage()
}

func (ui *mainUI) releaseAdminPage() {
	if ui.admin != nil {
		ui.admin.dispose()
	}
	ui.admin = nil
	ui.adminTab = nil
}

func (ui *mainUI) refreshAdminPage() {
	state := ui.admin
	if state == nil || state.closed.Load() || state.busy {
		return
	}
	ui.loadAdminReferences()
	if state.usersPage != nil {
		ui.loadAdminUsers()
	}
	if state.rolesPage != nil {
		ui.loadAdminRoles()
	}
}

func (ui *mainUI) loadAdminReferences() {
	state := ui.admin
	if state == nil || state.closed.Load() {
		return
	}
	state.referenceGeneration++
	generation := state.referenceGeneration
	state.info.SetText("正在加载部门、角色和权限目录……")
	guardedGo(func() {
		var departments []api.AdminDepartment
		var roleList []api.AdminRole
		var menus []api.AdminMenu
		var apis []api.AdminAPI
		var failures []string
		if state.usersPage != nil || state.departmentsPage != nil {
			var err error
			departments, err = ui.session.Client.AdminDepartments(state.ctx)
			if err != nil {
				failures = append(failures, "部门："+requestFailureText(err))
			}
		}
		if state.usersPage != nil || state.rolesPage != nil {
			var err error
			roleList, err = ui.session.Client.AdminRoleList(state.ctx, "")
			if err != nil {
				failures = append(failures, "角色："+requestFailureText(err))
			}
		}
		if state.rolesPage != nil || hasMenuPath(ui.session.Perms.Menus, "/acl/menu") {
			var err error
			menus, err = ui.session.Client.AdminMenus(state.ctx)
			if err != nil {
				failures = append(failures, "菜单："+requestFailureText(err))
			}
		}
		if state.rolesPage != nil || hasMenuPath(ui.session.Perms.Menus, "/acl/api") {
			var err error
			apis, err = ui.session.Client.AdminAPIs(state.ctx)
			if err != nil {
				failures = append(failures, "API："+requestFailureText(err))
			}
		}
		if state.ctx.Err() != nil || state.closed.Load() {
			return
		}
		ui.window.Synchronize(func() {
			if state != ui.admin || state.closed.Load() || generation != state.referenceGeneration {
				return
			}
			state.departments = departments
			state.roleList = roleList
			state.menus = menus
			state.apis = apis
			state.departmentRows = flattenAdminDepartmentRows(departments, "", "")
			if state.departmentTable != nil {
				_ = state.departmentTable.SetModel(state.departmentRows)
			}
			if state.menuTable != nil {
				_ = state.menuTable.SetModel(flattenAdminMenuRows(menus, ""))
			}
			if state.apiTable != nil {
				_ = state.apiTable.SetModel(flattenAdminAPIRows(apis, ""))
			}
			ui.updateAdminDepartmentActions()
			ui.updateAdminUserActions()
			ui.updateAdminRoleActions()
			if len(failures) > 0 {
				state.info.SetText("部分系统管理数据加载失败：" + strings.Join(failures, "；"))
			} else {
				state.info.SetText(fmt.Sprintf("系统管理数据已刷新：%d 个部门、%d 个角色、%d 个菜单项、%d 个 API 项。", len(state.departmentRows), len(roleList), len(flattenAdminMenuRows(menus, "")), len(flattenAdminAPIRows(apis, ""))))
			}
		})
	})
}

func (ui *mainUI) loadAdminUsers() {
	state := ui.admin
	if state == nil || state.userTable == nil || state.busy {
		return
	}
	if state.userCancel != nil {
		state.userCancel()
	}
	ctx, cancel := context.WithCancel(state.ctx)
	state.userCancel = cancel
	page := state.userPageNo
	size := selectedPageSize(state.userSize)
	name := state.userName.Text()
	mobile := state.userMobile.Text()
	generation := state.generation
	state.userQuery.SetEnabled(false)
	state.userReset.SetEnabled(false)
	state.info.SetText("正在加载系统用户……")
	guardedGo(func() {
		result, err := ui.session.Client.AdminUsers(ctx, page, size, name, mobile)
		if ctx.Err() != nil || state.closed.Load() {
			return
		}
		ui.window.Synchronize(func() {
			if state != ui.admin || generation != state.generation || state.userTable == nil {
				return
			}
			state.userQuery.SetEnabled(true)
			state.userReset.SetEnabled(true)
			if err != nil {
				state.info.SetText("用户加载失败：" + requestFailureText(err))
				return
			}
			state.users = result.List
			state.userTotal = result.Total
			rows := make([]adminUserRow, 0, len(result.List))
			for _, user := range result.List {
				rows = append(rows, adminUserRow{Name: user.Name, Mobile: user.Mobile, Email: user.Email, Department: user.DepartmentName, Roles: strings.Join(user.RoleNames, "、"), Status: user.Status, Remark: user.Remark, Updated: formatAdminTime(user.UpdatedAt)})
			}
			_ = state.userTable.SetModel(rows)
			state.userPrev.SetEnabled(page > 1)
			state.userNext.SetEnabled(int64(page*size) < result.Total)
			ui.updateAdminUserActions()
			state.info.SetText(fmt.Sprintf("用户：第 %d 页，本页 %d 条，共 %d 条。", page, len(rows), result.Total))
		})
	})
}

func (ui *mainUI) loadAdminRoles() {
	state := ui.admin
	if state == nil || state.roleTable == nil || state.busy {
		return
	}
	if state.roleCancel != nil {
		state.roleCancel()
	}
	ctx, cancel := context.WithCancel(state.ctx)
	state.roleCancel = cancel
	page := state.rolePageNo
	size := selectedPageSize(state.roleSize)
	name := state.roleName.Text()
	generation := state.generation
	state.roleQuery.SetEnabled(false)
	state.roleReset.SetEnabled(false)
	state.info.SetText("正在加载系统角色……")
	guardedGo(func() {
		result, err := ui.session.Client.AdminRoles(ctx, page, size, name)
		if ctx.Err() != nil || state.closed.Load() {
			return
		}
		ui.window.Synchronize(func() {
			if state != ui.admin || generation != state.generation || state.roleTable == nil {
				return
			}
			state.roleQuery.SetEnabled(true)
			state.roleReset.SetEnabled(true)
			if err != nil {
				state.info.SetText("角色加载失败：" + requestFailureText(err))
				return
			}
			state.roles = result.List
			state.roleTotal = result.Total
			parentNames := make(map[string]string, len(state.roleList))
			for _, role := range state.roleList {
				parentNames[role.ID] = role.Name
			}
			rows := make([]adminRoleRow, 0, len(result.List))
			for _, role := range result.List {
				rows = append(rows, adminRoleRow{Name: role.Name, Status: role.Status, Parent: parentNames[role.ParentID], Remark: role.Remark, Updated: formatAdminTime(role.UpdatedAt)})
			}
			_ = state.roleTable.SetModel(rows)
			state.rolePrev.SetEnabled(page > 1)
			state.roleNext.SetEnabled(int64(page*size) < result.Total)
			ui.updateAdminRoleActions()
			state.info.SetText(fmt.Sprintf("角色：第 %d 页，本页 %d 条，共 %d 条。", page, len(rows), result.Total))
		})
	})
}

func formatAdminTime(value int64) string {
	if value <= 0 {
		return ""
	}
	return time.Unix(value, 0).Format("2006-01-02 15:04")
}

func flattenAdminDepartmentRows(nodes []api.AdminDepartment, parentName, prefix string) []adminDepartmentRow {
	var rows []adminDepartmentRow
	for _, node := range nodes {
		rows = append(rows, adminDepartmentRow{Name: prefix + node.Name, Code: node.Code, Parent: parentName, Sort: fmt.Sprint(node.SortID), Remark: node.Remark})
		rows = append(rows, flattenAdminDepartmentRows(node.Children, node.Name, prefix+"    ")...)
	}
	return rows
}

func flattenAdminDepartmentOptions(nodes []api.AdminDepartment, prefix string, excluded map[string]bool) []selectOption {
	var result []selectOption
	for _, node := range nodes {
		if excluded[node.ID] {
			continue
		}
		result = append(result, selectOption{ID: node.ID, Label: prefix + node.Name})
		result = append(result, flattenAdminDepartmentOptions(node.Children, prefix+"    ", excluded)...)
	}
	return result
}

func flattenAdminMenuRows(nodes []api.AdminMenu, prefix string) []adminMenuRow {
	var rows []adminMenuRow
	for _, node := range nodes {
		title := strings.TrimSpace(node.Meta.Title)
		if title == "" {
			title = strings.TrimSpace(node.Name)
		}
		kind := "菜单"
		if node.Type == 2 {
			kind = "按钮"
		}
		rows = append(rows, adminMenuRow{Title: prefix + title, Kind: kind, Path: node.Path, Perms: node.Meta.Perms, Sort: fmt.Sprint(node.SortID)})
		rows = append(rows, flattenAdminMenuRows(node.Children, prefix+"    ")...)
	}
	return rows
}

func flattenAdminAPIRows(nodes []api.AdminAPI, prefix string) []adminAPIRow {
	var rows []adminAPIRow
	for _, node := range nodes {
		kind := "模块"
		if node.Type == 2 {
			kind = "API"
		}
		rows = append(rows, adminAPIRow{Name: prefix + node.Name, Kind: kind, Method: node.Method, URI: node.URI, Sort: fmt.Sprint(node.SortID)})
		rows = append(rows, flattenAdminAPIRows(node.Children, prefix+"    ")...)
	}
	return rows
}

func flattenAdminMenuOptions(nodes []api.AdminMenu, prefix string) []adminCatalogOption {
	var result []adminCatalogOption
	for _, node := range nodes {
		title := strings.TrimSpace(node.Meta.Title)
		if title == "" {
			title = strings.TrimSpace(node.Name)
		}
		label := prefix + title
		if node.Type == 2 && node.Meta.Perms != "" {
			label += "  [" + node.Meta.Perms + "]"
		}
		result = append(result, adminCatalogOption{ID: node.ID, Label: label})
		result = append(result, flattenAdminMenuOptions(node.Children, prefix+"    ")...)
	}
	return result
}

func flattenAdminAPIOptions(nodes []api.AdminAPI, prefix string) []adminCatalogOption {
	var result []adminCatalogOption
	for _, node := range nodes {
		label := prefix + node.Name
		if node.Type == 2 {
			label += "  [" + strings.TrimSpace(node.Method+" "+node.URI) + "]"
		}
		result = append(result, adminCatalogOption{ID: node.ID, Label: label})
		result = append(result, flattenAdminAPIOptions(node.Children, prefix+"    ")...)
	}
	return result
}

func resetAdminPageNumber(current *int) { *current = 1 }

func (ui *mainUI) resetAdminUsers() {
	state := ui.admin
	if state == nil {
		return
	}
	state.userName.SetText("")
	state.userMobile.SetText("")
	resetAdminPageNumber(&state.userPageNo)
	ui.loadAdminUsers()
}

func (ui *mainUI) resetAdminRoles() {
	state := ui.admin
	if state == nil {
		return
	}
	state.roleName.SetText("")
	resetAdminPageNumber(&state.rolePageNo)
	ui.loadAdminRoles()
}

func (ui *mainUI) selectedAdminUser() (api.AdminUser, bool) {
	state := ui.admin
	if state == nil || state.userTable == nil {
		return api.AdminUser{}, false
	}
	index := state.userTable.CurrentIndex()
	if index < 0 || index >= len(state.users) {
		walk.MsgBox(ui.window, "请选择用户", "请先选择一位系统用户。", walk.MsgBoxIconInformation)
		return api.AdminUser{}, false
	}
	return state.users[index], true
}

func (ui *mainUI) selectedAdminDepartment() (api.AdminDepartment, bool) {
	state := ui.admin
	if state == nil || state.departmentTable == nil {
		return api.AdminDepartment{}, false
	}
	index := state.departmentTable.CurrentIndex()
	flat := flattenAdminDepartments(state.departments)
	if index < 0 || index >= len(flat) {
		walk.MsgBox(ui.window, "请选择部门", "请先选择一个部门。", walk.MsgBoxIconInformation)
		return api.AdminDepartment{}, false
	}
	return flat[index], true
}

func flattenAdminDepartments(nodes []api.AdminDepartment) []api.AdminDepartment {
	var result []api.AdminDepartment
	for _, node := range nodes {
		result = append(result, node)
		result = append(result, flattenAdminDepartments(node.Children)...)
	}
	return result
}

func (ui *mainUI) selectedAdminRole() (api.AdminRole, bool) {
	state := ui.admin
	if state == nil || state.roleTable == nil {
		return api.AdminRole{}, false
	}
	index := state.roleTable.CurrentIndex()
	if index < 0 || index >= len(state.roles) {
		walk.MsgBox(ui.window, "请选择角色", "请先选择一个系统角色。", walk.MsgBoxIconInformation)
		return api.AdminRole{}, false
	}
	return state.roles[index], true
}

func (ui *mainUI) updateAdminUserActions() {
	state := ui.admin
	if state == nil || state.userEdit == nil {
		return
	}
	index := -1
	if state.userTable != nil {
		index = state.userTable.CurrentIndex()
	}
	enabled := !state.busy && index >= 0 && index < len(state.users) && state.users[index].Status != "删除"
	for _, button := range []*walk.PushButton{state.userEdit, state.userRoles, state.userStatus, state.userPass} {
		button.SetEnabled(enabled)
	}
}

func (ui *mainUI) updateAdminDepartmentActions() {
	state := ui.admin
	if state == nil || state.departmentEdit == nil {
		return
	}
	index := -1
	if state.departmentTable != nil {
		index = state.departmentTable.CurrentIndex()
	}
	state.departmentEdit.SetEnabled(!state.busy && index >= 0 && index < len(state.departmentRows))
}

func (ui *mainUI) updateAdminRoleActions() {
	state := ui.admin
	if state == nil || state.roleEdit == nil {
		return
	}
	index := -1
	if state.roleTable != nil {
		index = state.roleTable.CurrentIndex()
	}
	enabled := !state.busy && index >= 0 && index < len(state.roles) && state.roles[index].Status != "删除"
	for _, button := range []*walk.PushButton{state.roleEdit, state.roleState, state.roleMenus, state.roleAPIs} {
		button.SetEnabled(enabled)
	}
}

func (ui *mainUI) setAdminBusy(busy bool, message string) {
	state := ui.admin
	if state == nil {
		return
	}
	state.busy = busy
	if message != "" {
		state.info.SetText(message)
	}
	for _, button := range []*walk.PushButton{state.refresh, state.userQuery, state.userReset, state.userAdd, state.departmentAdd, state.roleQuery, state.roleReset, state.roleAdd} {
		if button != nil {
			button.SetEnabled(!busy)
		}
	}
	ui.updateAdminUserActions()
	ui.updateAdminDepartmentActions()
	ui.updateAdminRoleActions()
}

func (ui *mainUI) runAdminWrite(label string, write func(context.Context) error, after func()) {
	state := ui.admin
	if state == nil || state.busy {
		return
	}
	ui.setAdminBusy(true, label+"正在提交到线上服务……")
	guardedGo(func() {
		err := write(state.ctx)
		if state.ctx.Err() != nil || state.closed.Load() {
			return
		}
		ui.window.Synchronize(func() {
			if state != ui.admin || state.closed.Load() {
				return
			}
			ui.setAdminBusy(false, "")
			if err != nil {
				state.info.SetText(label + "失败：" + requestFailureText(err) + "。写操作不会自动重试。")
				return
			}
			state.info.SetText(label + "已提交成功，正在回读列表复核。")
			if after != nil {
				after()
			}
		})
	})
}

func adminOptionLabels(options []adminCatalogOption) []string {
	labels := make([]string, len(options))
	for index := range options {
		labels[index] = options[index].Label
	}
	return labels
}

func selectedAdminCatalogIDs(list *walk.ListBox, options []adminCatalogOption) []string {
	indexes := list.SelectedIndexes()
	result := make([]string, 0, len(indexes))
	for _, index := range indexes {
		if index >= 0 && index < len(options) {
			result = append(result, options[index].ID)
		}
	}
	return result
}

func selectedAdminOptionIDs(list *walk.ListBox, options []api.AdminRole) []string {
	indexes := list.SelectedIndexes()
	result := make([]string, 0, len(indexes))
	for _, index := range indexes {
		if index >= 0 && index < len(options) {
			result = append(result, options[index].ID)
		}
	}
	return result
}

func adminIndexesForIDs(ids []string, options []adminCatalogOption) []int {
	wanted := make(map[string]bool, len(ids))
	for _, id := range ids {
		wanted[strings.TrimSpace(id)] = true
	}
	var result []int
	for index, option := range options {
		if wanted[option.ID] {
			result = append(result, index)
		}
	}
	return result
}

func equalAdminIDSet(left, right []string) bool {
	leftSet := make(map[string]bool, len(left))
	for _, id := range left {
		if id = strings.TrimSpace(id); id != "" {
			leftSet[id] = true
		}
	}
	rightSet := make(map[string]bool, len(right))
	for _, id := range right {
		if id = strings.TrimSpace(id); id != "" {
			rightSet[id] = true
		}
	}
	if len(leftSet) != len(rightSet) {
		return false
	}
	for id := range leftSet {
		if !rightSet[id] {
			return false
		}
	}
	return true
}

func adminStatusTarget(current string) string {
	if strings.TrimSpace(current) == "启用" {
		return "禁用"
	}
	return "启用"
}

func parseAdminSort(text string) (int64, error) {
	value, err := strconv.ParseInt(strings.TrimSpace(text), 10, 64)
	if err != nil || value < 0 {
		return 0, fmt.Errorf("排序必须是大于或等于 0 的整数")
	}
	return value, nil
}

type adminUserFormResult struct {
	Create api.AdminUserCreateRequest
	Update api.AdminUserUpdateRequest
}

func (ui *mainUI) addAdminUser() {
	state := ui.admin
	if state == nil || state.busy {
		return
	}
	result, ok := editAdminUserDialog(ui.window, state, nil)
	if !ok {
		return
	}
	ui.runAdminWrite("新增用户", func(ctx context.Context) error { return ui.session.Client.CreateAdminUser(ctx, result.Create) }, func() {
		state.userPageNo = 1
		ui.loadAdminUsers()
		ui.loadAdminReferences()
	})
}

func (ui *mainUI) editSelectedAdminUser() {
	state := ui.admin
	if state == nil || state.busy {
		return
	}
	user, ok := ui.selectedAdminUser()
	if !ok {
		return
	}
	result, ok := editAdminUserDialog(ui.window, state, &user)
	if !ok {
		return
	}
	ui.runAdminWrite("编辑用户", func(ctx context.Context) error { return ui.session.Client.UpdateAdminUser(ctx, result.Update) }, func() {
		ui.loadAdminUsers()
		ui.loadAdminReferences()
	})
}

func editAdminUserDialog(owner walk.Form, state *adminUI, existing *api.AdminUser) (adminUserFormResult, bool) {
	var result adminUserFormResult
	if len(state.departments) == 0 || len(state.roleList) == 0 {
		walk.MsgBox(owner, "基础数据未就绪", "部门或角色列表尚未加载成功，请先刷新系统管理数据。", walk.MsgBoxIconWarning)
		return result, false
	}
	var dlg *walk.Dialog
	var nameEdit, passwordEdit, mobileEdit, emailEdit, remarkEdit *walk.LineEdit
	var sexCombo, departmentCombo, statusCombo *walk.ComboBox
	var roleList *walk.ListBox
	var info *walk.Label
	var save *walk.PushButton
	departments := flattenAdminDepartmentOptions(state.departments, "", map[string]bool{})
	departmentLabels := optionLabels("请选择部门", departments)
	var existingRoleIDs []string
	if existing != nil {
		existingRoleIDs = existing.RoleIDs
	}
	assignableRoles := assignableAdminRoles(state.roleList, existingRoleIDs)
	roleLabels := make([]string, len(assignableRoles))
	for index, role := range assignableRoles {
		roleLabels[index] = role.Name
		if role.Status != "启用" {
			roleLabels[index] += "（" + role.Status + "）"
		}
	}
	isCreate := existing == nil
	title := "新增用户"
	if !isCreate {
		title = "编辑用户 - " + existing.Name
	}
	children := []Widget{
		Label{Text: "账号名称 *"}, LineEdit{AssignTo: &nameEdit, CueBanner: "2-21 个字符", Accessibility: Accessibility{Name: "系统用户账号名称"}},
	}
	if isCreate {
		children = append(children, Label{Text: "初始密码 *"}, LineEdit{AssignTo: &passwordEdit, PasswordMode: true, CueBanner: "至少 6 位", Accessibility: Accessibility{Name: "系统用户初始密码"}})
	}
	children = append(children,
		Label{Text: "手机号码 *"}, LineEdit{AssignTo: &mobileEdit, CueBanner: "11 位手机号", Accessibility: Accessibility{Name: "系统用户手机号码"}},
		Label{Text: "Email"}, LineEdit{AssignTo: &emailEdit, CueBanner: "可选", Accessibility: Accessibility{Name: "系统用户 Email"}},
		Label{Text: "性别 *"}, ComboBox{AssignTo: &sexCombo, Model: []string{"男", "女"}, CurrentIndex: 0, Accessibility: Accessibility{Name: "系统用户性别"}},
		Label{Text: "部门 *"}, ComboBox{AssignTo: &departmentCombo, Model: departmentLabels, CurrentIndex: 0, Accessibility: Accessibility{Name: "系统用户部门"}},
		Label{Text: "状态 *"}, ComboBox{AssignTo: &statusCombo, Model: []string{"启用", "禁用"}, CurrentIndex: 0, Enabled: !isCreate, Accessibility: Accessibility{Name: "系统用户状态"}},
		Label{Text: "角色 *"}, ListBox{AssignTo: &roleList, Model: roleLabels, MultiSelection: true, MinSize: Size{Height: 120}, Accessibility: Accessibility{Name: "系统用户角色多选列表"}},
		Label{Text: "备注"}, LineEdit{AssignTo: &remarkEdit, CueBanner: "可选", Accessibility: Accessibility{Name: "系统用户备注"}},
	)
	err := Dialog{
		AssignTo: &dlg, Title: title, MinSize: Size{Width: 620, Height: 600}, Size: Size{Width: 700, Height: 680},
		Layout: VBox{Margins: Margins{Left: 16, Top: 16, Right: 16, Bottom: 14}, Spacing: 10},
		Children: []Widget{
			GroupBox{Title: "用户资料", Layout: Grid{Columns: 2, Margins: Margins{Left: 14, Top: 10, Right: 14, Bottom: 12}, Spacing: 8}, Children: children},
			Label{AssignTo: &info, Text: "带 * 的项目为必填；角色至少选择一个。", TextColor: secondaryTextColor()},
			Composite{Layout: HBox{Spacing: 8}, Children: []Widget{HSpacer{}, PushButton{Text: "取消", MinSize: Size{Width: 80, Height: 30}, OnClicked: func() { dlg.Cancel() }}, PushButton{AssignTo: &save, Text: "保存", MinSize: Size{Width: 88, Height: 30}}}},
		},
	}.Create(owner)
	if err != nil {
		walk.MsgBox(owner, "无法打开用户编辑", err.Error(), walk.MsgBoxIconError)
		return result, false
	}
	if existing != nil {
		nameEdit.SetText(existing.Name)
		mobileEdit.SetText(existing.Mobile)
		emailEdit.SetText(existing.Email)
		remarkEdit.SetText(existing.Remark)
		if existing.Sex == "女" {
			sexCombo.SetCurrentIndex(1)
		}
		statusCombo.SetCurrentIndex(0)
		if existing.Status == "禁用" {
			statusCombo.SetCurrentIndex(1)
		}
		departmentCombo.SetCurrentIndex(optionIndexByID(departments, existing.DepartmentID))
		roleList.SetSelectedIndexes(roleIndexesByIDs(existing.RoleIDs, assignableRoles))
	}
	accepted := false
	apply := func() {
		name := strings.TrimSpace(nameEdit.Text())
		if len([]rune(name)) < 2 || len([]rune(name)) > 21 {
			info.SetText("账号名称必须为 2-21 个字符。")
			nameEdit.SetFocus()
			return
		}
		password := ""
		if isCreate {
			password = strings.TrimSpace(passwordEdit.Text())
			if len([]rune(password)) < 6 {
				info.SetText("初始密码至少需要 6 位。")
				passwordEdit.SetFocus()
				return
			}
		}
		mobile := strings.TrimSpace(mobileEdit.Text())
		if !isSimpleMobile(mobile) {
			info.SetText("手机号码必须是 11 位数字。")
			mobileEdit.SetFocus()
			return
		}
		email := strings.TrimSpace(emailEdit.Text())
		if email != "" && (!strings.Contains(email, "@") || strings.HasPrefix(email, "@") || strings.HasSuffix(email, "@")) {
			info.SetText("Email 格式不正确。")
			emailEdit.SetFocus()
			return
		}
		departmentID := selectedOptionID(departmentCombo, departments)
		if departmentID == "" {
			info.SetText("请选择部门。")
			departmentCombo.SetFocus()
			return
		}
		roleIDs := selectedAdminOptionIDs(roleList, assignableRoles)
		if len(roleIDs) == 0 {
			info.SetText("至少需要选择一个角色。")
			roleList.SetFocus()
			return
		}
		sex := sexCombo.Text()
		status := statusCombo.Text()
		remark := strings.TrimSpace(remarkEdit.Text())
		if isCreate {
			result.Create = api.AdminUserCreateRequest{Name: name, Password: password, Sex: sex, DepartmentID: departmentID, RoleIDs: roleIDs, Mobile: mobile, Email: email, Status: "启用", Remark: remark}
		} else {
			result.Update = api.AdminUserUpdateRequest{ID: existing.ID, Name: name, Sex: sex, DepartmentID: departmentID, RoleIDs: roleIDs, Mobile: mobile, Email: email, Status: status, Remark: remark}
		}
		accepted = true
		dlg.Accept()
	}
	save.Clicked().Attach(apply)
	dlg.Run()
	return result, accepted
}

func roleIndexesByIDs(ids []string, roles []api.AdminRole) []int {
	wanted := make(map[string]bool, len(ids))
	for _, id := range ids {
		wanted[strings.TrimSpace(id)] = true
	}
	var result []int
	for index, role := range roles {
		if wanted[role.ID] {
			result = append(result, index)
		}
	}
	return result
}

func assignableAdminRoles(roles []api.AdminRole, existingIDs []string) []api.AdminRole {
	existing := make(map[string]bool, len(existingIDs))
	for _, id := range existingIDs {
		existing[strings.TrimSpace(id)] = true
	}
	result := make([]api.AdminRole, 0, len(roles))
	for _, role := range roles {
		if role.Status == "启用" || existing[role.ID] {
			result = append(result, role)
		}
	}
	return result
}

func isSimpleMobile(value string) bool {
	if len(value) != 11 {
		return false
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}

func (ui *mainUI) assignSelectedAdminUserRoles() {
	state := ui.admin
	if state == nil || state.busy {
		return
	}
	user, ok := ui.selectedAdminUser()
	if !ok {
		return
	}
	if len(state.roleList) == 0 {
		walk.MsgBox(ui.window, "角色未加载", "请先刷新系统管理数据。", walk.MsgBoxIconWarning)
		return
	}
	assignableRoles := assignableAdminRoles(state.roleList, user.RoleIDs)
	var dlg *walk.Dialog
	var list *walk.ListBox
	var info *walk.Label
	var apply *walk.PushButton
	labels := make([]string, len(assignableRoles))
	for index, role := range assignableRoles {
		labels[index] = role.Name + "（" + role.Status + "）"
	}
	accepted := false
	err := Dialog{AssignTo: &dlg, Title: "分配角色 - " + user.Name, MinSize: Size{Width: 520, Height: 480}, Size: Size{Width: 600, Height: 560}, Layout: VBox{Margins: Margins{Left: 16, Top: 16, Right: 16, Bottom: 14}, Spacing: 10}, Children: []Widget{
		Label{Text: "用户：" + user.Name + "（" + user.Mobile + "）", Font: Font{Bold: true}},
		Label{Text: "选择一个或多个角色；保存后会重新读取用户列表。", TextColor: secondaryTextColor()},
		ListBox{AssignTo: &list, Model: labels, MultiSelection: true, StretchFactor: 1, Accessibility: Accessibility{Name: "为用户分配角色"}},
		Label{AssignTo: &info, Text: "至少选择一个角色。", TextColor: secondaryTextColor()},
		Composite{Layout: HBox{Spacing: 8}, Children: []Widget{HSpacer{}, PushButton{Text: "取消", OnClicked: func() { dlg.Cancel() }}, PushButton{AssignTo: &apply, Text: "保存分配"}}},
	}}.Create(ui.window)
	if err != nil {
		walk.MsgBox(ui.window, "无法分配角色", err.Error(), walk.MsgBoxIconError)
		return
	}
	list.SetSelectedIndexes(roleIndexesByIDs(user.RoleIDs, assignableRoles))
	var roleIDs []string
	apply.Clicked().Attach(func() {
		roleIDs = selectedAdminOptionIDs(list, assignableRoles)
		if len(roleIDs) == 0 {
			info.SetText("至少需要选择一个角色。")
			list.SetFocus()
			return
		}
		accepted = true
		dlg.Accept()
	})
	dlg.Run()
	if !accepted {
		return
	}
	if walk.MsgBox(ui.window, "确认分配角色", fmt.Sprintf("将为用户“%s”分配 %d 个角色，是否提交？", user.Name, len(roleIDs)), walk.MsgBoxYesNo|walk.MsgBoxIconQuestion) != walk.DlgCmdYes {
		return
	}
	ui.runAdminWrite("分配用户角色", func(ctx context.Context) error { return ui.session.Client.UpdateAdminUserRoles(ctx, user.ID, roleIDs) }, ui.loadAdminUsers)
}

func (ui *mainUI) toggleSelectedAdminUserStatus() {
	state := ui.admin
	if state == nil || state.busy {
		return
	}
	user, ok := ui.selectedAdminUser()
	if !ok {
		return
	}
	target := adminStatusTarget(user.Status)
	if target == "禁用" && strings.TrimSpace(user.Mobile) == strings.TrimSpace(ui.session.Profile.Mobile) {
		walk.MsgBox(ui.window, "不能禁用当前账号", "客户端不允许禁用当前登录账号，以免当前会话失去管理能力。", walk.MsgBoxIconWarning)
		return
	}
	if walk.MsgBox(ui.window, "确认变更用户状态", fmt.Sprintf("是否将用户“%s”从“%s”变更为“%s”？", user.Name, user.Status, target), walk.MsgBoxYesNo|walk.MsgBoxIconWarning) != walk.DlgCmdYes {
		return
	}
	ui.runAdminWrite("变更用户状态", func(ctx context.Context) error {
		return ui.session.Client.UpdateAdminUserStatus(ctx, []string{user.ID}, target)
	}, ui.loadAdminUsers)
}

func (ui *mainUI) resetSelectedAdminUserPassword() {
	state := ui.admin
	if state == nil || state.busy {
		return
	}
	user, ok := ui.selectedAdminUser()
	if !ok {
		return
	}
	var dlg *walk.Dialog
	var first, second *walk.LineEdit
	var info *walk.Label
	var apply *walk.PushButton
	password := ""
	err := Dialog{AssignTo: &dlg, Title: "重置密码 - " + user.Name, MinSize: Size{Width: 500, Height: 300}, Size: Size{Width: 580, Height: 360}, Layout: VBox{Margins: Margins{Left: 16, Top: 16, Right: 16, Bottom: 14}, Spacing: 10}, Children: []Widget{
		GroupBox{Title: "新密码", Layout: Grid{Columns: 2, Margins: Margins{Left: 12, Top: 10, Right: 12, Bottom: 12}, Spacing: 8}, Children: []Widget{
			Label{Text: "新密码 *"}, LineEdit{AssignTo: &first, PasswordMode: true, CueBanner: "至少 6 位", Accessibility: Accessibility{Name: "用户新密码"}},
			Label{Text: "再次输入 *"}, LineEdit{AssignTo: &second, PasswordMode: true, CueBanner: "再次输入", Accessibility: Accessibility{Name: "确认用户新密码"}},
		}},
		Label{AssignTo: &info, Text: "密码重置后，请通知用户使用新密码登录。", TextColor: secondaryTextColor()},
		Composite{Layout: HBox{Spacing: 8}, Children: []Widget{HSpacer{}, PushButton{Text: "取消", OnClicked: func() { dlg.Cancel() }}, PushButton{AssignTo: &apply, Text: "确认重置"}}},
	}}.Create(ui.window)
	if err != nil {
		walk.MsgBox(ui.window, "无法重置密码", err.Error(), walk.MsgBoxIconError)
		return
	}
	apply.Clicked().Attach(func() {
		password = strings.TrimSpace(first.Text())
		if len([]rune(password)) < 6 {
			info.SetText("新密码至少需要 6 位。")
			first.SetFocus()
			return
		}
		if password != strings.TrimSpace(second.Text()) {
			info.SetText("两次输入的密码不一致。")
			second.SetFocus()
			return
		}
		dlg.Accept()
	})
	dlg.Run()
	if password == "" {
		return
	}
	if walk.MsgBox(ui.window, "确认重置密码", "是否重置用户“"+user.Name+"”的密码？", walk.MsgBoxYesNo|walk.MsgBoxIconWarning) != walk.DlgCmdYes {
		return
	}
	ui.runAdminWrite("重置用户密码", func(ctx context.Context) error {
		return ui.session.Client.ResetAdminUserPassword(ctx, user.ID, password)
	}, nil)
}

func (ui *mainUI) addAdminDepartment() {
	state := ui.admin
	if state == nil || state.busy {
		return
	}
	request, ok := editAdminDepartmentDialog(ui.window, state.departments, nil)
	if !ok {
		return
	}
	ui.runAdminWrite("新增部门", func(ctx context.Context) error { return ui.session.Client.CreateAdminDepartment(ctx, request) }, ui.loadAdminReferences)
}

func (ui *mainUI) editSelectedAdminDepartment() {
	state := ui.admin
	if state == nil || state.busy {
		return
	}
	department, ok := ui.selectedAdminDepartment()
	if !ok {
		return
	}
	request, ok := editAdminDepartmentDialog(ui.window, state.departments, &department)
	if !ok {
		return
	}
	ui.runAdminWrite("编辑部门", func(ctx context.Context) error { return ui.session.Client.UpdateAdminDepartment(ctx, request) }, ui.loadAdminReferences)
}

func editAdminDepartmentDialog(owner walk.Form, departments []api.AdminDepartment, existing *api.AdminDepartment) (api.AdminDepartmentRequest, bool) {
	var result api.AdminDepartmentRequest
	excluded := map[string]bool{}
	if existing != nil {
		collectAdminDepartmentIDs(*existing, excluded)
	}
	parents := flattenAdminDepartmentOptions(departments, "", excluded)
	var dlg *walk.Dialog
	var nameEdit, codeEdit, sortEdit, remarkEdit *walk.LineEdit
	var parentCombo *walk.ComboBox
	var info *walk.Label
	var save *walk.PushButton
	title := "新增部门"
	if existing != nil {
		title = "编辑部门 - " + existing.Name
	}
	err := Dialog{AssignTo: &dlg, Title: title, MinSize: Size{Width: 540, Height: 400}, Size: Size{Width: 620, Height: 470}, Layout: VBox{Margins: Margins{Left: 16, Top: 16, Right: 16, Bottom: 14}, Spacing: 10}, Children: []Widget{
		GroupBox{Title: "部门资料", Layout: Grid{Columns: 2, Margins: Margins{Left: 12, Top: 10, Right: 12, Bottom: 12}, Spacing: 8}, Children: []Widget{
			Label{Text: "部门名称 *"}, LineEdit{AssignTo: &nameEdit, Accessibility: Accessibility{Name: "部门名称"}},
			Label{Text: "部门编码 *"}, LineEdit{AssignTo: &codeEdit, Accessibility: Accessibility{Name: "部门编码"}},
			Label{Text: "上级部门"}, ComboBox{AssignTo: &parentCombo, Model: optionLabels("无上级部门", parents), CurrentIndex: 0, Accessibility: Accessibility{Name: "上级部门"}},
			Label{Text: "排序 *"}, LineEdit{AssignTo: &sortEdit, Text: "0", CueBanner: "非负整数", Accessibility: Accessibility{Name: "部门排序"}},
			Label{Text: "备注"}, LineEdit{AssignTo: &remarkEdit, CueBanner: "可选", Accessibility: Accessibility{Name: "部门备注"}},
		}},
		Label{AssignTo: &info, Text: "编辑时不会提供当前部门及其下级作为上级选项。", TextColor: secondaryTextColor()},
		Composite{Layout: HBox{Spacing: 8}, Children: []Widget{HSpacer{}, PushButton{Text: "取消", OnClicked: func() { dlg.Cancel() }}, PushButton{AssignTo: &save, Text: "保存"}}},
	}}.Create(owner)
	if err != nil {
		walk.MsgBox(owner, "无法编辑部门", err.Error(), walk.MsgBoxIconError)
		return result, false
	}
	if existing != nil {
		nameEdit.SetText(existing.Name)
		codeEdit.SetText(existing.Code)
		sortEdit.SetText(fmt.Sprint(existing.SortID))
		remarkEdit.SetText(existing.Remark)
		parentCombo.SetCurrentIndex(optionIndexByID(parents, existing.ParentID))
	}
	accepted := false
	save.Clicked().Attach(func() {
		name := strings.TrimSpace(nameEdit.Text())
		if name == "" {
			info.SetText("部门名称不能为空。")
			nameEdit.SetFocus()
			return
		}
		code := strings.TrimSpace(codeEdit.Text())
		if code == "" {
			info.SetText("部门编码不能为空。")
			codeEdit.SetFocus()
			return
		}
		sortID, err := parseAdminSort(sortEdit.Text())
		if err != nil {
			info.SetText(err.Error())
			sortEdit.SetFocus()
			return
		}
		result = api.AdminDepartmentRequest{Name: name, Code: code, SortID: sortID, ParentID: selectedOptionID(parentCombo, parents), Remark: strings.TrimSpace(remarkEdit.Text())}
		if existing != nil {
			result.ID = existing.ID
		}
		accepted = true
		dlg.Accept()
	})
	dlg.Run()
	return result, accepted
}

func collectAdminDepartmentIDs(node api.AdminDepartment, target map[string]bool) {
	target[node.ID] = true
	for _, child := range node.Children {
		collectAdminDepartmentIDs(child, target)
	}
}

func (ui *mainUI) addAdminRole() {
	state := ui.admin
	if state == nil || state.busy {
		return
	}
	request, ok := editAdminRoleDialog(ui.window, state.roleList, nil)
	if !ok {
		return
	}
	ui.runAdminWrite("新增角色", func(ctx context.Context) error { return ui.session.Client.CreateAdminRole(ctx, request) }, func() { ui.loadAdminReferences(); state.rolePageNo = 1; ui.loadAdminRoles() })
}

func (ui *mainUI) editSelectedAdminRole() {
	state := ui.admin
	if state == nil || state.busy {
		return
	}
	role, ok := ui.selectedAdminRole()
	if !ok {
		return
	}
	request, ok := editAdminRoleDialog(ui.window, state.roleList, &role)
	if !ok {
		return
	}
	ui.runAdminWrite("编辑角色", func(ctx context.Context) error { return ui.session.Client.UpdateAdminRole(ctx, request) }, func() { ui.loadAdminReferences(); ui.loadAdminRoles() })
}

func editAdminRoleDialog(owner walk.Form, roles []api.AdminRole, existing *api.AdminRole) (api.AdminRoleRequest, bool) {
	var result api.AdminRoleRequest
	excluded := map[string]bool{}
	if existing != nil {
		collectAdminRoleDescendants(existing.ID, roles, excluded)
	}
	parents := make([]selectOption, 0, len(roles))
	for _, role := range roles {
		if !excluded[role.ID] {
			parents = append(parents, selectOption{ID: role.ID, Label: role.Name + "（" + role.Status + "）"})
		}
	}
	var dlg *walk.Dialog
	var nameEdit, remarkEdit *walk.LineEdit
	var parentCombo, statusCombo *walk.ComboBox
	var info *walk.Label
	var save *walk.PushButton
	title := "新增角色"
	if existing != nil {
		title = "编辑角色 - " + existing.Name
	}
	err := Dialog{AssignTo: &dlg, Title: title, MinSize: Size{Width: 540, Height: 350}, Size: Size{Width: 620, Height: 430}, Layout: VBox{Margins: Margins{Left: 16, Top: 16, Right: 16, Bottom: 14}, Spacing: 10}, Children: []Widget{
		GroupBox{Title: "角色资料", Layout: Grid{Columns: 2, Margins: Margins{Left: 12, Top: 10, Right: 12, Bottom: 12}, Spacing: 8}, Children: []Widget{
			Label{Text: "角色名称 *"}, LineEdit{AssignTo: &nameEdit, Accessibility: Accessibility{Name: "角色名称"}},
			Label{Text: "上级角色"}, ComboBox{AssignTo: &parentCombo, Model: optionLabels("无上级角色", parents), CurrentIndex: 0, Accessibility: Accessibility{Name: "上级角色"}},
			Label{Text: "状态 *"}, ComboBox{AssignTo: &statusCombo, Model: []string{"启用", "禁用"}, CurrentIndex: 0, Accessibility: Accessibility{Name: "角色状态"}},
			Label{Text: "备注"}, LineEdit{AssignTo: &remarkEdit, CueBanner: "可选", Accessibility: Accessibility{Name: "角色备注"}},
		}},
		Label{AssignTo: &info, Text: "保存角色资料不会同时修改菜单或 API 授权。", TextColor: secondaryTextColor()},
		Composite{Layout: HBox{Spacing: 8}, Children: []Widget{HSpacer{}, PushButton{Text: "取消", OnClicked: func() { dlg.Cancel() }}, PushButton{AssignTo: &save, Text: "保存"}}},
	}}.Create(owner)
	if err != nil {
		walk.MsgBox(owner, "无法编辑角色", err.Error(), walk.MsgBoxIconError)
		return result, false
	}
	if existing != nil {
		nameEdit.SetText(existing.Name)
		remarkEdit.SetText(existing.Remark)
		parentCombo.SetCurrentIndex(optionIndexByID(parents, existing.ParentID))
		if existing.Status == "禁用" {
			statusCombo.SetCurrentIndex(1)
		}
	}
	accepted := false
	save.Clicked().Attach(func() {
		name := strings.TrimSpace(nameEdit.Text())
		if name == "" {
			info.SetText("角色名称不能为空。")
			nameEdit.SetFocus()
			return
		}
		result = api.AdminRoleRequest{Name: name, ParentID: selectedOptionID(parentCombo, parents), Status: statusCombo.Text(), Remark: strings.TrimSpace(remarkEdit.Text())}
		if existing != nil {
			result.ID = existing.ID
		}
		accepted = true
		dlg.Accept()
	})
	dlg.Run()
	return result, accepted
}

func collectAdminRoleDescendants(rootID string, roles []api.AdminRole, target map[string]bool) {
	if target[rootID] {
		return
	}
	target[rootID] = true
	for _, role := range roles {
		if role.ParentID == rootID {
			collectAdminRoleDescendants(role.ID, roles, target)
		}
	}
}

func (ui *mainUI) toggleSelectedAdminRoleStatus() {
	state := ui.admin
	if state == nil || state.busy {
		return
	}
	role, ok := ui.selectedAdminRole()
	if !ok {
		return
	}
	target := adminStatusTarget(role.Status)
	if walk.MsgBox(ui.window, "确认变更角色状态", fmt.Sprintf("是否将角色“%s”从“%s”变更为“%s”？", role.Name, role.Status, target), walk.MsgBoxYesNo|walk.MsgBoxIconWarning) != walk.DlgCmdYes {
		return
	}
	ui.runAdminWrite("变更角色状态", func(ctx context.Context) error {
		return ui.session.Client.UpdateAdminRoleStatus(ctx, []string{role.ID}, target)
	}, func() { ui.loadAdminReferences(); ui.loadAdminRoles() })
}

func (ui *mainUI) assignSelectedAdminRoleMenus() { ui.assignSelectedAdminRoleCatalog("menu") }
func (ui *mainUI) assignSelectedAdminRoleAPIs()  { ui.assignSelectedAdminRoleCatalog("api") }

func (ui *mainUI) assignSelectedAdminRoleCatalog(kind string) {
	state := ui.admin
	if state == nil || state.busy {
		return
	}
	role, ok := ui.selectedAdminRole()
	if !ok {
		return
	}
	var options []adminCatalogOption
	label := "菜单"
	if kind == "menu" {
		options = flattenAdminMenuOptions(state.menus, "")
	} else {
		options = flattenAdminAPIOptions(state.apis, "")
		label = "API"
	}
	if len(options) == 0 {
		walk.MsgBox(ui.window, label+"目录未加载", "请先刷新系统管理数据。", walk.MsgBoxIconWarning)
		return
	}
	ui.setAdminBusy(true, "正在读取角色“"+role.Name+"”的"+label+"授权……")
	guardedGo(func() {
		var current []string
		var err error
		if kind == "menu" {
			current, err = ui.session.Client.AdminRoleMenuIDs(state.ctx, role.ID)
		} else {
			current, err = ui.session.Client.AdminRoleAPIIDs(state.ctx, role.ID)
		}
		if state.ctx.Err() != nil || state.closed.Load() {
			return
		}
		ui.window.Synchronize(func() {
			if state != ui.admin || state.closed.Load() {
				return
			}
			ui.setAdminBusy(false, "")
			if err != nil {
				state.info.SetText("读取角色" + label + "授权失败：" + requestFailureText(err))
				return
			}
			selected, accepted := editAdminRoleCatalogDialog(ui.window, role, label, options, current)
			if !accepted {
				return
			}
			ui.writeAdminRoleCatalog(role, kind, label, selected)
		})
	})
}

func editAdminRoleCatalogDialog(owner walk.Form, role api.AdminRole, label string, options []adminCatalogOption, current []string) ([]string, bool) {
	var dlg *walk.Dialog
	var list *walk.ListBox
	var info *walk.Label
	var apply *walk.PushButton
	var selected []string
	accepted := false
	err := Dialog{AssignTo: &dlg, Title: "分配" + label + " - " + role.Name, MinSize: Size{Width: 700, Height: 620}, Size: Size{Width: 800, Height: 720}, Layout: VBox{Margins: Margins{Left: 16, Top: 16, Right: 16, Bottom: 14}, Spacing: 10}, Children: []Widget{
		Label{Text: "角色：" + role.Name, Font: Font{Bold: true}},
		Label{Text: "目录按层级缩进展示；请同时保留访问路径所需的上级项目。", TextColor: secondaryTextColor()},
		ListBox{AssignTo: &list, Model: adminOptionLabels(options), MultiSelection: true, StretchFactor: 1, Accessibility: Accessibility{Name: "角色" + label + "授权多选列表"}},
		Label{AssignTo: &info, Text: fmt.Sprintf("当前已选 %d 项；服务端要求至少选择一项。", len(current)), TextColor: secondaryTextColor()},
		Composite{Layout: HBox{Spacing: 8}, Children: []Widget{HSpacer{}, PushButton{Text: "取消", OnClicked: func() { dlg.Cancel() }}, PushButton{AssignTo: &apply, Text: "核对并保存"}}},
	}}.Create(owner)
	if err != nil {
		walk.MsgBox(owner, "无法分配"+label, err.Error(), walk.MsgBoxIconError)
		return nil, false
	}
	list.SetSelectedIndexes(adminIndexesForIDs(current, options))
	apply.Clicked().Attach(func() {
		selected = selectedAdminCatalogIDs(list, options)
		if len(selected) == 0 {
			info.SetText("至少需要选择一项，现有接口不接受空授权列表。")
			list.SetFocus()
			return
		}
		added, removed := adminIDDiffCounts(current, selected)
		if walk.MsgBox(dlg, "核对"+label+"授权", fmt.Sprintf("角色：%s\r\n选择：%d 项\r\n新增：%d 项\r\n移除：%d 项\r\n\r\n是否提交到线上服务？", role.Name, len(selected), added, removed), walk.MsgBoxYesNo|walk.MsgBoxIconWarning) != walk.DlgCmdYes {
			return
		}
		accepted = true
		dlg.Accept()
	})
	dlg.Run()
	return selected, accepted
}

func adminIDDiffCounts(before, after []string) (added, removed int) {
	beforeSet := make(map[string]bool, len(before))
	afterSet := make(map[string]bool, len(after))
	for _, id := range before {
		beforeSet[strings.TrimSpace(id)] = true
	}
	for _, id := range after {
		afterSet[strings.TrimSpace(id)] = true
	}
	for id := range afterSet {
		if id != "" && !beforeSet[id] {
			added++
		}
	}
	for id := range beforeSet {
		if id != "" && !afterSet[id] {
			removed++
		}
	}
	return
}

func (ui *mainUI) writeAdminRoleCatalog(role api.AdminRole, kind, label string, selected []string) {
	state := ui.admin
	if state == nil || state.busy {
		return
	}
	ui.setAdminBusy(true, "正在保存角色“"+role.Name+"”的"+label+"授权并回读复核……")
	guardedGo(func() {
		var writeErr error
		if kind == "menu" {
			writeErr = ui.session.Client.SetAdminRoleMenuIDs(state.ctx, role.ID, selected)
		} else {
			writeErr = ui.session.Client.SetAdminRoleAPIIDs(state.ctx, role.ID, selected)
		}
		writeApplied := writeErr == nil
		var verified []string
		var verifyErr error
		if writeApplied {
			if kind == "menu" {
				verified, verifyErr = ui.session.Client.AdminRoleMenuIDs(state.ctx, role.ID)
			} else {
				verified, verifyErr = ui.session.Client.AdminRoleAPIIDs(state.ctx, role.ID)
			}
			if verifyErr == nil && !equalAdminIDSet(selected, verified) {
				verifyErr = fmt.Errorf("回读结果与提交内容不一致")
			}
		}
		if state.ctx.Err() != nil || state.closed.Load() {
			return
		}
		ui.window.Synchronize(func() {
			if state != ui.admin || state.closed.Load() {
				return
			}
			ui.setAdminBusy(false, "")
			if writeErr != nil {
				state.info.SetText("角色" + label + "授权失败：" + requestFailureText(writeErr) + "。写操作不会自动重试。")
				return
			}
			if verifyErr != nil {
				state.info.SetText("角色" + label + "授权已返回成功，但回读复核失败：" + verifyErr.Error() + "。请勿立即重复提交，先刷新后核对。")
				walk.MsgBox(ui.window, "授权回读复核失败", state.info.Text(), walk.MsgBoxIconWarning)
				return
			}
			state.info.SetText(fmt.Sprintf("角色“%s”的%s授权已保存并回读复核，共 %d 项。", role.Name, label, len(verified)))
		})
	})
}
