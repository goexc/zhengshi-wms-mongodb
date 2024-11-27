// Code generated by goctl. DO NOT EDIT.
// goctl 1.7.2

package handler

import (
	"net/http"

	account "api/internal/handler/account"
	api "api/internal/handler/api"
	auth "api/internal/handler/auth"
	carrier "api/internal/handler/carrier"
	company "api/internal/handler/company"
	customer "api/internal/handler/customer"
	customertransaction "api/internal/handler/customer/transaction"
	department "api/internal/handler/department"
	images "api/internal/handler/images"
	inboundreceipt "api/internal/handler/inbound/receipt"
	inventory "api/internal/handler/inventory"
	material "api/internal/handler/material"
	materialprice "api/internal/handler/material/price"
	menu "api/internal/handler/menu"
	outbound "api/internal/handler/outbound"
	plan "api/internal/handler/plan"
	role "api/internal/handler/role"
	supplier "api/internal/handler/supplier"
	user "api/internal/handler/user"
	warehouse "api/internal/handler/warehouse"
	warehouse_bin "api/internal/handler/warehouse_bin"
	warehouse_rack "api/internal/handler/warehouse_rack"
	warehouse_zone "api/internal/handler/warehouse_zone"
	"api/internal/svc"

	"github.com/zeromicro/go-zero/rest"
)

func RegisterHandlers(server *rest.Server, serverCtx *svc.ServiceContext) {
	server.AddRoutes(
		[]rest.Route{
			{
				// 修改个人头像
				Method:  http.MethodPatch,
				Path:    "/avatar",
				Handler: account.ChangeAvatarHandler(serverCtx),
			},
			{
				// 获取菜单列表
				Method:  http.MethodGet,
				Path:    "/menu",
				Handler: account.MenuHandler(serverCtx),
			},
			{
				// 修改个人密码
				Method:  http.MethodPatch,
				Path:    "/password",
				Handler: account.ChangePasswordHandler(serverCtx),
			},
			{
				// 个人信息
				Method:  http.MethodGet,
				Path:    "/profile",
				Handler: account.ProfileHandler(serverCtx),
			},
			{
				// 修改个人信息
				Method:  http.MethodPut,
				Path:    "/profile",
				Handler: account.EditProfileHandler(serverCtx),
			},
		},
		rest.WithJwt(serverCtx.Config.Auth.AccessSecret),
		rest.WithPrefix("/account"),
	)

	server.AddRoutes(
		[]rest.Route{
			{
				// api列表
				Method:  http.MethodGet,
				Path:    "/",
				Handler: api.ListHandler(serverCtx),
			},
			{
				// 添加api
				Method:  http.MethodPost,
				Path:    "/",
				Handler: api.AddHandler(serverCtx),
			},
			{
				// 修改api
				Method:  http.MethodPut,
				Path:    "/",
				Handler: api.UpdateHandler(serverCtx),
			},
			{
				// 删除api
				Method:  http.MethodDelete,
				Path:    "/",
				Handler: api.RemoveHandler(serverCtx),
			},
		},
		rest.WithJwt(serverCtx.Config.Auth.AccessSecret),
		rest.WithPrefix("/api"),
	)

	server.AddRoutes(
		[]rest.Route{
			{
				// 登录
				Method:  http.MethodPost,
				Path:    "/login",
				Handler: auth.LoginHandler(serverCtx),
			},
			{
				// 注册
				Method:  http.MethodPost,
				Path:    "/register",
				Handler: auth.RegisterHandler(serverCtx),
			},
		},
		rest.WithPrefix("/auth"),
	)

	server.AddRoutes(
		[]rest.Route{
			{
				// 退出登录
				Method:  http.MethodPost,
				Path:    "/logout",
				Handler: auth.LogoutHandler(serverCtx),
			},
		},
		rest.WithJwt(serverCtx.Config.Auth.AccessSecret),
		rest.WithPrefix("/auth"),
	)

	server.AddRoutes(
		[]rest.Route{
			{
				// 承运商分页
				Method:  http.MethodGet,
				Path:    "/",
				Handler: carrier.ListHandler(serverCtx),
			},
			{
				// 添加承运商
				Method:  http.MethodPost,
				Path:    "/",
				Handler: carrier.AddHandler(serverCtx),
			},
			{
				// 修改承运商
				Method:  http.MethodPut,
				Path:    "/",
				Handler: carrier.UpdateHandler(serverCtx),
			},
			{
				// 修改承运商状态/删除承运商
				Method:  http.MethodPatch,
				Path:    "/status",
				Handler: carrier.StatusHandler(serverCtx),
			},
		},
		rest.WithJwt(serverCtx.Config.Auth.AccessSecret),
		rest.WithPrefix("/carrier"),
	)

	server.AddRoutes(
		[]rest.Route{
			{
				// 获取企业信息
				Method:  http.MethodGet,
				Path:    "/",
				Handler: company.InfoHandler(serverCtx),
			},
			{
				// 修改企业信息
				Method:  http.MethodPut,
				Path:    "/",
				Handler: company.UpdateHandler(serverCtx),
			},
		},
		rest.WithJwt(serverCtx.Config.Auth.AccessSecret),
		rest.WithPrefix("/company"),
	)

	server.AddRoutes(
		[]rest.Route{
			{
				// 客户分页
				Method:  http.MethodGet,
				Path:    "/",
				Handler: customer.PageHandler(serverCtx),
			},
			{
				// 添加客户
				Method:  http.MethodPost,
				Path:    "/",
				Handler: customer.AddHandler(serverCtx),
			},
			{
				// 修改客户
				Method:  http.MethodPut,
				Path:    "/",
				Handler: customer.UpdateHandler(serverCtx),
			},
			{
				// 客户列表
				Method:  http.MethodGet,
				Path:    "/list",
				Handler: customer.ListHandler(serverCtx),
			},
			{
				// 重新统计应收账款
				Method:  http.MethodGet,
				Path:    "/recount",
				Handler: customer.RecountHandler(serverCtx),
			},
			{
				// 修改客户状态/删除客户
				Method:  http.MethodPatch,
				Path:    "/status",
				Handler: customer.StatusHandler(serverCtx),
			},
		},
		rest.WithJwt(serverCtx.Config.Auth.AccessSecret),
		rest.WithPrefix("/customer"),
	)

	server.AddRoutes(
		[]rest.Route{
			{
				// 客户交易流水分页
				Method:  http.MethodGet,
				Path:    "/",
				Handler: customertransaction.PageHandler(serverCtx),
			},
			{
				// 添加客户交易记录
				Method:  http.MethodPost,
				Path:    "/",
				Handler: customertransaction.AddHandler(serverCtx),
			},
		},
		rest.WithJwt(serverCtx.Config.Auth.AccessSecret),
		rest.WithPrefix("/customer/transaction"),
	)

	server.AddRoutes(
		[]rest.Route{
			{
				// 部门列表
				Method:  http.MethodGet,
				Path:    "/",
				Handler: department.ListHandler(serverCtx),
			},
			{
				// 添加部门
				Method:  http.MethodPost,
				Path:    "/",
				Handler: department.AddHandler(serverCtx),
			},
			{
				// 修改部门
				Method:  http.MethodPut,
				Path:    "/",
				Handler: department.UpdateHandler(serverCtx),
			},
			{
				// 删除部门
				Method:  http.MethodDelete,
				Path:    "/",
				Handler: department.RemoveHandler(serverCtx),
			},
		},
		rest.WithJwt(serverCtx.Config.Auth.AccessSecret),
		rest.WithPrefix("/department"),
	)

	server.AddRoutes(
		[]rest.Route{
			{
				// 上传图片
				Method:  http.MethodPost,
				Path:    "/",
				Handler: images.AddHandler(serverCtx),
			},
			{
				// 图片分页
				Method:  http.MethodGet,
				Path:    "/",
				Handler: images.PageHandler(serverCtx),
			},
			{
				// 删除图片
				Method:  http.MethodDelete,
				Path:    "/",
				Handler: images.RemoveHandler(serverCtx),
			},
		},
		rest.WithJwt(serverCtx.Config.Auth.AccessSecret),
		rest.WithPrefix("/images"),
	)

	server.AddRoutes(
		[]rest.Route{
			{
				// 入库单分页
				Method:  http.MethodGet,
				Path:    "/",
				Handler: inboundreceipt.PageHandler(serverCtx),
			},
			{
				// 创建入库单
				Method:  http.MethodPost,
				Path:    "/",
				Handler: inboundreceipt.AddHandler(serverCtx),
			},
			{
				// 修改入库单
				Method:  http.MethodPut,
				Path:    "/",
				Handler: inboundreceipt.UpdateHandler(serverCtx),
			},
			{
				// 删除入库单
				Method:  http.MethodDelete,
				Path:    "/",
				Handler: inboundreceipt.RemoveHandler(serverCtx),
			},
			{
				// 审核入库单
				Method:  http.MethodPatch,
				Path:    "/check",
				Handler: inboundreceipt.CheckHandler(serverCtx),
			},
			{
				// 批次入库
				Method:  http.MethodPost,
				Path:    "/receive",
				Handler: inboundreceipt.ReceiveHandler(serverCtx),
			},
			{
				// 入库记录
				Method:  http.MethodGet,
				Path:    "/receive",
				Handler: inboundreceipt.RecordHandler(serverCtx),
			},
		},
		rest.WithJwt(serverCtx.Config.Auth.AccessSecret),
		rest.WithPrefix("/inbound/receipt"),
	)

	server.AddRoutes(
		[]rest.Route{
			{
				// 库存管理
				Method:  http.MethodGet,
				Path:    "/",
				Handler: inventory.PageHandler(serverCtx),
			},
			{
				// 物料库存(用于确认出库单时，查询物料库存)
				Method:  http.MethodGet,
				Path:    "/list",
				Handler: inventory.ListHandler(serverCtx),
			},
			{
				// 库存记录
				Method:  http.MethodGet,
				Path:    "/record",
				Handler: inventory.RecordHandler(serverCtx),
			},
		},
		rest.WithJwt(serverCtx.Config.Auth.AccessSecret),
		rest.WithPrefix("/inventory"),
	)

	server.AddRoutes(
		[]rest.Route{
			{
				// 添加物料
				Method:  http.MethodPost,
				Path:    "/",
				Handler: material.AddHandler(serverCtx),
			},
			{
				// 修改物料
				Method:  http.MethodPut,
				Path:    "/",
				Handler: material.UpdateHandler(serverCtx),
			},
			{
				// 删除物料
				Method:  http.MethodDelete,
				Path:    "/",
				Handler: material.RemoveHandler(serverCtx),
			},
			{
				// 物料分页
				Method:  http.MethodGet,
				Path:    "/",
				Handler: material.ListHandler(serverCtx),
			},
			{
				// 物料分类列表
				Method:  http.MethodGet,
				Path:    "/category",
				Handler: material.CategoryListHandler(serverCtx),
			},
			{
				// 添加物料分类
				Method:  http.MethodPost,
				Path:    "/category",
				Handler: material.AddCategoryHandler(serverCtx),
			},
			{
				// 修改物料分类
				Method:  http.MethodPut,
				Path:    "/category",
				Handler: material.UpdateCategoryHandler(serverCtx),
			},
			{
				// 删除物料分类
				Method:  http.MethodDelete,
				Path:    "/category",
				Handler: material.RemoveCategoryHandler(serverCtx),
			},
			{
				// 物料信息
				Method:  http.MethodGet,
				Path:    "/info",
				Handler: material.InfoHandler(serverCtx),
			},
		},
		rest.WithJwt(serverCtx.Config.Auth.AccessSecret),
		rest.WithPrefix("/material"),
	)

	server.AddRoutes(
		[]rest.Route{
			{
				// 物料单价
				Method:  http.MethodGet,
				Path:    "/",
				Handler: materialprice.ListHandler(serverCtx),
			},
			{
				// 删除物料单价
				Method:  http.MethodDelete,
				Path:    "/",
				Handler: materialprice.RemoveHandler(serverCtx),
			},
		},
		rest.WithJwt(serverCtx.Config.Auth.AccessSecret),
		rest.WithPrefix("/material/price"),
	)

	server.AddRoutes(
		[]rest.Route{
			{
				// 添加菜单
				Method:  http.MethodPost,
				Path:    "/",
				Handler: menu.AddHandler(serverCtx),
			},
			{
				// 修改菜单
				Method:  http.MethodPut,
				Path:    "/",
				Handler: menu.UpdateHandler(serverCtx),
			},
			{
				// 删除菜单
				Method:  http.MethodDelete,
				Path:    "/",
				Handler: menu.RemoveHandler(serverCtx),
			},
			{
				// 菜单列表
				Method:  http.MethodGet,
				Path:    "/list",
				Handler: menu.ListHandler(serverCtx),
			},
		},
		rest.WithJwt(serverCtx.Config.Auth.AccessSecret),
		rest.WithPrefix("/menu"),
	)

	server.AddRoutes(
		[]rest.Route{
			{
				// 创建出库单
				Method:  http.MethodPost,
				Path:    "/",
				Handler: outbound.AddHandler(serverCtx),
			},
			{
				// 删除出库单
				Method:  http.MethodDelete,
				Path:    "/",
				Handler: outbound.RemoveHandler(serverCtx),
			},
			{
				// 确认出库单
				Method:  http.MethodPatch,
				Path:    "/confirm",
				Handler: outbound.ConfirmHandler(serverCtx),
			},
			{
				// 出库
				Method:  http.MethodPatch,
				Path:    "/departure",
				Handler: outbound.DepartureHandler(serverCtx),
			},
			{
				// 出库单物料列表
				Method:  http.MethodGet,
				Path:    "/materials",
				Handler: outbound.MaterialsHandler(serverCtx),
			},
			{
				// 确认打包
				Method:  http.MethodPatch,
				Path:    "/pack",
				Handler: outbound.PackHandler(serverCtx),
			},
			{
				// 出库单分页(不携带物料列表)
				Method:  http.MethodGet,
				Path:    "/page",
				Handler: outbound.PageHandler(serverCtx),
			},
			{
				// 出库单分页(携带物料列表)
				Method:  http.MethodGet,
				Path:    "/page2",
				Handler: outbound.Page2Handler(serverCtx),
			},
			{
				// 确认拣货
				Method:  http.MethodPatch,
				Path:    "/pick",
				Handler: outbound.PickHandler(serverCtx),
			},
			{
				// 签收
				Method:  http.MethodPatch,
				Path:    "/receipt",
				Handler: outbound.ReceiptHandler(serverCtx),
			},
			{
				// 修改出库单物料单价
				Method:  http.MethodPatch,
				Path:    "/revise",
				Handler: outbound.ReviseHandler(serverCtx),
			},
			{
				// 出库单汇总
				Method:  http.MethodGet,
				Path:    "/summary",
				Handler: outbound.SummaryHandler(serverCtx),
			},
			{
				// 确认称重
				Method:  http.MethodPatch,
				Path:    "/weigh",
				Handler: outbound.WeighHandler(serverCtx),
			},
		},
		rest.WithJwt(serverCtx.Config.Auth.AccessSecret),
		rest.WithPrefix("/outbound"),
	)

	server.AddRoutes(
		[]rest.Route{
			{
				// 添加计划
				Method:  http.MethodPost,
				Path:    "/",
				Handler: plan.AddHandler(serverCtx),
			},
			{
				// 修改计划
				Method:  http.MethodPut,
				Path:    "/",
				Handler: plan.UpdateHandler(serverCtx),
			},
			{
				// 计划分页
				Method:  http.MethodGet,
				Path:    "/",
				Handler: plan.PageHandler(serverCtx),
			},
			{
				// 删除计划
				Method:  http.MethodDelete,
				Path:    "/",
				Handler: plan.RemoveHandler(serverCtx),
			},
			{
				// 修改计划状态
				Method:  http.MethodPatch,
				Path:    "/status",
				Handler: plan.StatusHandler(serverCtx),
			},
		},
		rest.WithJwt(serverCtx.Config.Auth.AccessSecret),
		rest.WithPrefix("/plan"),
	)

	server.AddRoutes(
		[]rest.Route{
			{
				// 角色分页
				Method:  http.MethodGet,
				Path:    "/",
				Handler: role.PaginateHandler(serverCtx),
			},
			{
				// 添加角色
				Method:  http.MethodPost,
				Path:    "/",
				Handler: role.AddHandler(serverCtx),
			},
			{
				// 修改角色
				Method:  http.MethodPut,
				Path:    "/",
				Handler: role.UpdateHandler(serverCtx),
			},
			{
				// 删除角色
				Method:  http.MethodDelete,
				Path:    "/",
				Handler: role.RemoveHandler(serverCtx),
			},
			{
				// 角色的api列表
				Method:  http.MethodGet,
				Path:    "/apis",
				Handler: role.ApisHandler(serverCtx),
			},
			{
				// 分配角色api
				Method:  http.MethodPost,
				Path:    "/apis",
				Handler: role.ApiDistributeHandler(serverCtx),
			},
			{
				// 角色列表
				Method:  http.MethodGet,
				Path:    "/list",
				Handler: role.ListHandler(serverCtx),
			},
			{
				// 角色的菜单列表
				Method:  http.MethodGet,
				Path:    "/menus",
				Handler: role.MenusHandler(serverCtx),
			},
			{
				// 分配角色菜单
				Method:  http.MethodPost,
				Path:    "/menus",
				Handler: role.MenuDistributeHandler(serverCtx),
			},
			{
				// 更改角色状态
				Method:  http.MethodPatch,
				Path:    "/status",
				Handler: role.StatusHandler(serverCtx),
			},
		},
		rest.WithJwt(serverCtx.Config.Auth.AccessSecret),
		rest.WithPrefix("/role"),
	)

	server.AddRoutes(
		[]rest.Route{
			{
				// 供应商分页
				Method:  http.MethodGet,
				Path:    "/",
				Handler: supplier.PaginateHandler(serverCtx),
			},
			{
				// 添加供应商
				Method:  http.MethodPost,
				Path:    "/",
				Handler: supplier.AddHandler(serverCtx),
			},
			{
				// 修改供应商
				Method:  http.MethodPut,
				Path:    "/",
				Handler: supplier.UpdateHandler(serverCtx),
			},
			{
				// 供应商列表
				Method:  http.MethodGet,
				Path:    "/list",
				Handler: supplier.ListHandler(serverCtx),
			},
			{
				// 修改供应商状态/删除供应商
				Method:  http.MethodPatch,
				Path:    "/status",
				Handler: supplier.StatusHandler(serverCtx),
			},
		},
		rest.WithJwt(serverCtx.Config.Auth.AccessSecret),
		rest.WithPrefix("/supplier"),
	)

	server.AddRoutes(
		[]rest.Route{
			{
				// 用户分页
				Method:  http.MethodGet,
				Path:    "/",
				Handler: user.PaginateHandler(serverCtx),
			},
			{
				// 添加用户
				Method:  http.MethodPost,
				Path:    "/",
				Handler: user.AddHandler(serverCtx),
			},
			{
				// 修改用户
				Method:  http.MethodPut,
				Path:    "/",
				Handler: user.UpdateHandler(serverCtx),
			},
			{
				// 删除用户
				Method:  http.MethodDelete,
				Path:    "/",
				Handler: user.RemoveHandler(serverCtx),
			},
			{
				// 管理员修改用户密码
				Method:  http.MethodPatch,
				Path:    "/password",
				Handler: user.ChangePasswordHandler(serverCtx),
			},
			{
				// 管理员更改用户角色
				Method:  http.MethodPatch,
				Path:    "/roles",
				Handler: user.RolesHandler(serverCtx),
			},
			{
				// 管理员更改用户状态
				Method:  http.MethodPatch,
				Path:    "/status",
				Handler: user.StatusHandler(serverCtx),
			},
		},
		rest.WithJwt(serverCtx.Config.Auth.AccessSecret),
		rest.WithPrefix("/user"),
	)

	server.AddRoutes(
		[]rest.Route{
			{
				// 仓库分页
				Method:  http.MethodGet,
				Path:    "/",
				Handler: warehouse.PaginateHandler(serverCtx),
			},
			{
				// 添加仓库
				Method:  http.MethodPost,
				Path:    "/",
				Handler: warehouse.AddHandler(serverCtx),
			},
			{
				// 修改仓库
				Method:  http.MethodPut,
				Path:    "/",
				Handler: warehouse.UpdateHandler(serverCtx),
			},
			{
				// 仓库列表
				Method:  http.MethodGet,
				Path:    "/list",
				Handler: warehouse.ListHandler(serverCtx),
			},
			{
				// 修改仓库状态/删除仓库
				Method:  http.MethodPatch,
				Path:    "/status",
				Handler: warehouse.StatusHandler(serverCtx),
			},
			{
				// 仓库/库区/货架/货位树
				Method:  http.MethodGet,
				Path:    "/tree",
				Handler: warehouse.TreeHandler(serverCtx),
			},
		},
		rest.WithJwt(serverCtx.Config.Auth.AccessSecret),
		rest.WithPrefix("/warehouse"),
	)

	server.AddRoutes(
		[]rest.Route{
			{
				// 货位分页
				Method:  http.MethodGet,
				Path:    "/",
				Handler: warehouse_bin.PaginateHandler(serverCtx),
			},
			{
				// 添加货位
				Method:  http.MethodPost,
				Path:    "/",
				Handler: warehouse_bin.AddHandler(serverCtx),
			},
			{
				// 修改货位
				Method:  http.MethodPut,
				Path:    "/",
				Handler: warehouse_bin.UpdateHandler(serverCtx),
			},
			{
				// 货位列表
				Method:  http.MethodGet,
				Path:    "/list",
				Handler: warehouse_bin.ListHandler(serverCtx),
			},
			{
				// 修改货位状态/删除货位
				Method:  http.MethodPatch,
				Path:    "/status",
				Handler: warehouse_bin.StatusHandler(serverCtx),
			},
		},
		rest.WithJwt(serverCtx.Config.Auth.AccessSecret),
		rest.WithPrefix("/warehouse_bin"),
	)

	server.AddRoutes(
		[]rest.Route{
			{
				// 货架分页
				Method:  http.MethodGet,
				Path:    "/",
				Handler: warehouse_rack.PaginateHandler(serverCtx),
			},
			{
				// 添加货架
				Method:  http.MethodPost,
				Path:    "/",
				Handler: warehouse_rack.AddHandler(serverCtx),
			},
			{
				// 修改货架
				Method:  http.MethodPut,
				Path:    "/",
				Handler: warehouse_rack.UpdateHandler(serverCtx),
			},
			{
				// 货架列表
				Method:  http.MethodGet,
				Path:    "/list",
				Handler: warehouse_rack.ListHandler(serverCtx),
			},
			{
				// 修改货架状态/删除货架
				Method:  http.MethodPatch,
				Path:    "/status",
				Handler: warehouse_rack.StatusHandler(serverCtx),
			},
		},
		rest.WithJwt(serverCtx.Config.Auth.AccessSecret),
		rest.WithPrefix("/warehouse_rack"),
	)

	server.AddRoutes(
		[]rest.Route{
			{
				// 库区分页
				Method:  http.MethodGet,
				Path:    "/",
				Handler: warehouse_zone.PaginateHandler(serverCtx),
			},
			{
				// 添加库区
				Method:  http.MethodPost,
				Path:    "/",
				Handler: warehouse_zone.AddHandler(serverCtx),
			},
			{
				// 修改库区
				Method:  http.MethodPut,
				Path:    "/",
				Handler: warehouse_zone.UpdateHandler(serverCtx),
			},
			{
				// 库区列表
				Method:  http.MethodGet,
				Path:    "/list",
				Handler: warehouse_zone.ListHandler(serverCtx),
			},
			{
				// 修改库区状态/删除库区
				Method:  http.MethodPatch,
				Path:    "/status",
				Handler: warehouse_zone.StatusHandler(serverCtx),
			},
		},
		rest.WithJwt(serverCtx.Config.Auth.AccessSecret),
		rest.WithPrefix("/warehouse_zone"),
	)
}
