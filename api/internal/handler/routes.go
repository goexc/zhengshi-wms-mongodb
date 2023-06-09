// Code generated by goctl. DO NOT EDIT.
package handler

import (
	"net/http"

	account "api/internal/handler/account"
	api "api/internal/handler/api"
	auth "api/internal/handler/auth"
	company "api/internal/handler/company"
	customer "api/internal/handler/customer"
	department "api/internal/handler/department"
	images "api/internal/handler/images"
	inbound "api/internal/handler/inbound"
	material "api/internal/handler/material"
	menu "api/internal/handler/menu"
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
				Method:  http.MethodPost,
				Path:    "/register",
				Handler: auth.RegisterHandler(serverCtx),
			},
			{
				Method:  http.MethodPost,
				Path:    "/login",
				Handler: auth.LoginHandler(serverCtx),
			},
			{
				Method:  http.MethodPost,
				Path:    "/logout",
				Handler: auth.LogoutHandler(serverCtx),
			},
		},
		rest.WithPrefix("/auth"),
	)

	server.AddRoutes(
		[]rest.Route{
			{
				Method:  http.MethodGet,
				Path:    "/list",
				Handler: menu.ListHandler(serverCtx),
			},
			{
				Method:  http.MethodPost,
				Path:    "/",
				Handler: menu.AddHandler(serverCtx),
			},
			{
				Method:  http.MethodPut,
				Path:    "/",
				Handler: menu.UpdateHandler(serverCtx),
			},
			{
				Method:  http.MethodDelete,
				Path:    "/",
				Handler: menu.RemoveHandler(serverCtx),
			},
		},
		rest.WithJwt(serverCtx.Config.Auth.AccessSecret),
		rest.WithPrefix("/menu"),
	)

	server.AddRoutes(
		[]rest.Route{
			{
				Method:  http.MethodGet,
				Path:    "/",
				Handler: api.ListHandler(serverCtx),
			},
			{
				Method:  http.MethodPost,
				Path:    "/",
				Handler: api.AddHandler(serverCtx),
			},
			{
				Method:  http.MethodPut,
				Path:    "/",
				Handler: api.UpdateHandler(serverCtx),
			},
			{
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
				Method:  http.MethodGet,
				Path:    "/list",
				Handler: role.ListHandler(serverCtx),
			},
			{
				Method:  http.MethodGet,
				Path:    "/",
				Handler: role.PaginateHandler(serverCtx),
			},
			{
				Method:  http.MethodPost,
				Path:    "/",
				Handler: role.AddHandler(serverCtx),
			},
			{
				Method:  http.MethodPut,
				Path:    "/",
				Handler: role.UpdateHandler(serverCtx),
			},
			{
				Method:  http.MethodDelete,
				Path:    "/",
				Handler: role.RemoveHandler(serverCtx),
			},
			{
				Method:  http.MethodPatch,
				Path:    "/status",
				Handler: role.StatusHandler(serverCtx),
			},
			{
				Method:  http.MethodGet,
				Path:    "/menus",
				Handler: role.MenusHandler(serverCtx),
			},
			{
				Method:  http.MethodPost,
				Path:    "/menus",
				Handler: role.MenuDistributeHandler(serverCtx),
			},
			{
				Method:  http.MethodGet,
				Path:    "/apis",
				Handler: role.ApisHandler(serverCtx),
			},
			{
				Method:  http.MethodPost,
				Path:    "/apis",
				Handler: role.ApiDistributeHandler(serverCtx),
			},
		},
		rest.WithJwt(serverCtx.Config.Auth.AccessSecret),
		rest.WithPrefix("/role"),
	)

	server.AddRoutes(
		[]rest.Route{
			{
				Method:  http.MethodGet,
				Path:    "/",
				Handler: department.ListHandler(serverCtx),
			},
			{
				Method:  http.MethodPost,
				Path:    "/",
				Handler: department.AddHandler(serverCtx),
			},
			{
				Method:  http.MethodPut,
				Path:    "/",
				Handler: department.UpdateHandler(serverCtx),
			},
			{
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
				Method:  http.MethodGet,
				Path:    "/",
				Handler: user.PaginateHandler(serverCtx),
			},
			{
				Method:  http.MethodPost,
				Path:    "/",
				Handler: user.AddHandler(serverCtx),
			},
			{
				Method:  http.MethodPut,
				Path:    "/",
				Handler: user.UpdateHandler(serverCtx),
			},
			{
				Method:  http.MethodDelete,
				Path:    "/",
				Handler: user.RemoveHandler(serverCtx),
			},
			{
				Method:  http.MethodPatch,
				Path:    "/password",
				Handler: user.ChangePasswordHandler(serverCtx),
			},
			{
				Method:  http.MethodPatch,
				Path:    "/status",
				Handler: user.StatusHandler(serverCtx),
			},
			{
				Method:  http.MethodPatch,
				Path:    "/roles",
				Handler: user.RolesHandler(serverCtx),
			},
		},
		rest.WithJwt(serverCtx.Config.Auth.AccessSecret),
		rest.WithPrefix("/user"),
	)

	server.AddRoutes(
		[]rest.Route{
			{
				Method:  http.MethodGet,
				Path:    "/profile",
				Handler: account.ProfileHandler(serverCtx),
			},
			{
				Method:  http.MethodPut,
				Path:    "/profile",
				Handler: account.EditProfileHandler(serverCtx),
			},
			{
				Method:  http.MethodPatch,
				Path:    "/password",
				Handler: account.ChangePasswordHandler(serverCtx),
			},
			{
				Method:  http.MethodPatch,
				Path:    "/avatar",
				Handler: account.ChangeAvatarHandler(serverCtx),
			},
			{
				Method:  http.MethodGet,
				Path:    "/menu",
				Handler: account.MenuHandler(serverCtx),
			},
		},
		rest.WithJwt(serverCtx.Config.Auth.AccessSecret),
		rest.WithPrefix("/account"),
	)

	server.AddRoutes(
		[]rest.Route{
			{
				Method:  http.MethodGet,
				Path:    "/",
				Handler: company.GetHandler(serverCtx),
			},
			{
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
				Method:  http.MethodGet,
				Path:    "/",
				Handler: supplier.ListHandler(serverCtx),
			},
			{
				Method:  http.MethodPost,
				Path:    "/",
				Handler: supplier.AddHandler(serverCtx),
			},
			{
				Method:  http.MethodPut,
				Path:    "/",
				Handler: supplier.UpdateHandler(serverCtx),
			},
			{
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
				Method:  http.MethodGet,
				Path:    "/",
				Handler: customer.ListHandler(serverCtx),
			},
			{
				Method:  http.MethodPost,
				Path:    "/",
				Handler: customer.AddHandler(serverCtx),
			},
			{
				Method:  http.MethodPut,
				Path:    "/",
				Handler: customer.UpdateHandler(serverCtx),
			},
			{
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
				Method:  http.MethodGet,
				Path:    "/list",
				Handler: warehouse.ListHandler(serverCtx),
			},
			{
				Method:  http.MethodGet,
				Path:    "/",
				Handler: warehouse.PaginateHandler(serverCtx),
			},
			{
				Method:  http.MethodPost,
				Path:    "/",
				Handler: warehouse.AddHandler(serverCtx),
			},
			{
				Method:  http.MethodPut,
				Path:    "/",
				Handler: warehouse.UpdateHandler(serverCtx),
			},
			{
				Method:  http.MethodPatch,
				Path:    "/status",
				Handler: warehouse.StatusHandler(serverCtx),
			},
		},
		rest.WithJwt(serverCtx.Config.Auth.AccessSecret),
		rest.WithPrefix("/warehouse"),
	)

	server.AddRoutes(
		[]rest.Route{
			{
				Method:  http.MethodGet,
				Path:    "/list",
				Handler: warehouse_zone.ListHandler(serverCtx),
			},
			{
				Method:  http.MethodGet,
				Path:    "/",
				Handler: warehouse_zone.PaginateHandler(serverCtx),
			},
			{
				Method:  http.MethodPost,
				Path:    "/",
				Handler: warehouse_zone.AddHandler(serverCtx),
			},
			{
				Method:  http.MethodPut,
				Path:    "/",
				Handler: warehouse_zone.UpdateHandler(serverCtx),
			},
			{
				Method:  http.MethodPatch,
				Path:    "/status",
				Handler: warehouse_zone.StatusHandler(serverCtx),
			},
		},
		rest.WithJwt(serverCtx.Config.Auth.AccessSecret),
		rest.WithPrefix("/warehouse_zone"),
	)

	server.AddRoutes(
		[]rest.Route{
			{
				Method:  http.MethodGet,
				Path:    "/list",
				Handler: warehouse_rack.ListHandler(serverCtx),
			},
			{
				Method:  http.MethodGet,
				Path:    "/",
				Handler: warehouse_rack.PaginateHandler(serverCtx),
			},
			{
				Method:  http.MethodPost,
				Path:    "/",
				Handler: warehouse_rack.AddHandler(serverCtx),
			},
			{
				Method:  http.MethodPut,
				Path:    "/",
				Handler: warehouse_rack.UpdateHandler(serverCtx),
			},
			{
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
				Method:  http.MethodGet,
				Path:    "/list",
				Handler: warehouse_bin.ListHandler(serverCtx),
			},
			{
				Method:  http.MethodGet,
				Path:    "/",
				Handler: warehouse_bin.PaginateHandler(serverCtx),
			},
			{
				Method:  http.MethodPost,
				Path:    "/",
				Handler: warehouse_bin.AddHandler(serverCtx),
			},
			{
				Method:  http.MethodPut,
				Path:    "/",
				Handler: warehouse_bin.UpdateHandler(serverCtx),
			},
			{
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
				Method:  http.MethodPost,
				Path:    "/",
				Handler: material.AddHandler(serverCtx),
			},
			{
				Method:  http.MethodPut,
				Path:    "/",
				Handler: material.UpdateHandler(serverCtx),
			},
			{
				Method:  http.MethodDelete,
				Path:    "/",
				Handler: material.RemoveHandler(serverCtx),
			},
			{
				Method:  http.MethodGet,
				Path:    "/",
				Handler: material.ListHandler(serverCtx),
			},
		},
		rest.WithJwt(serverCtx.Config.Auth.AccessSecret),
		rest.WithPrefix("/material"),
	)

	server.AddRoutes(
		[]rest.Route{
			{
				Method:  http.MethodPost,
				Path:    "/procurement",
				Handler: inbound.ProcurementHandler(serverCtx),
			},
		},
		rest.WithJwt(serverCtx.Config.Auth.AccessSecret),
		rest.WithPrefix("/inbound"),
	)

	server.AddRoutes(
		[]rest.Route{
			{
				Method:  http.MethodPost,
				Path:    "/",
				Handler: images.AddHandler(serverCtx),
			},
		},
		rest.WithJwt(serverCtx.Config.Auth.AccessSecret),
		rest.WithPrefix("/images"),
	)
}
