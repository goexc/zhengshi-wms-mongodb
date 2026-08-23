package api

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const (
	FastOutboundSale   = "销售出库"
	FastOutboundSample = "样品出库"
	FastOutboundGift   = "赠品出库"
)

type FastOutboundMaterialRequest struct {
	MaterialID string  `json:"material_id"`
	Price      float64 `json:"price"`
	Quantity   float64 `json:"quantity"`
}

type FastOutboundRequest struct {
	Code          string                        `json:"code"`
	Type          string                        `json:"type"`
	CustomerID    string                        `json:"customer_id"`
	DepartureTime int64                         `json:"departure_time"`
	PickingTime   int64                         `json:"picking_time,omitempty"`
	PackingTime   int64                         `json:"packing_time,omitempty"`
	WeighingTime  int64                         `json:"weighing_time,omitempty"`
	ReceiptTime   int64                         `json:"receipt_time,omitempty"`
	Materials     []FastOutboundMaterialRequest `json:"materials"`
}

func (request FastOutboundRequest) Normalized() FastOutboundRequest {
	request.Code = strings.TrimSpace(request.Code)
	request.Type = strings.TrimSpace(request.Type)
	request.CustomerID = strings.TrimSpace(request.CustomerID)
	request.Materials = append([]FastOutboundMaterialRequest(nil), request.Materials...)
	for index := range request.Materials {
		request.Materials[index].MaterialID = strings.TrimSpace(request.Materials[index].MaterialID)
	}
	return request
}

func (c *Client) FastOutbound(ctx context.Context, request FastOutboundRequest) error {
	return c.do(ctx, http.MethodPost, "/outbound/fast_departure", nil, request.Normalized(), nil)
}

type AdminUser struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Sex            string   `json:"sex"`
	DepartmentID   string   `json:"department_id"`
	DepartmentName string   `json:"department_name"`
	RoleIDs        []string `json:"roles_id"`
	RoleNames      []string `json:"roles_name"`
	Mobile         string   `json:"mobile"`
	Email          string   `json:"email"`
	Status         string   `json:"status"`
	Remark         string   `json:"remark"`
	CreatedAt      int64    `json:"created_at"`
	UpdatedAt      int64    `json:"updated_at"`
}

type AdminUserPage struct {
	Total int64       `json:"total"`
	List  []AdminUser `json:"list"`
}

type AdminUserCreateRequest struct {
	Name         string   `json:"name"`
	Password     string   `json:"password"`
	Sex          string   `json:"sex"`
	DepartmentID string   `json:"department_id"`
	RoleIDs      []string `json:"roles_id"`
	Mobile       string   `json:"mobile"`
	Email        string   `json:"email,omitempty"`
	Status       string   `json:"status,omitempty"`
	Remark       string   `json:"remark,omitempty"`
}

type AdminUserUpdateRequest struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Sex          string   `json:"sex"`
	DepartmentID string   `json:"department_id"`
	RoleIDs      []string `json:"roles_id"`
	Mobile       string   `json:"mobile"`
	Email        string   `json:"email,omitempty"`
	Status       string   `json:"status"`
	Remark       string   `json:"remark,omitempty"`
}

type AdminStatusRequest struct {
	IDs    []string `json:"id"`
	Status string   `json:"status"`
}

type AdminUserRolesRequest struct {
	ID      string   `json:"id"`
	RoleIDs []string `json:"roles_id"`
}

func (c *Client) AdminUsers(ctx context.Context, page, size int, name, mobile string) (AdminUserPage, error) {
	query := url.Values{}
	query.Set("page", strconv.Itoa(page))
	query.Set("size", strconv.Itoa(size))
	setTrimmedQuery(query, "name", name)
	setTrimmedQuery(query, "mobile", mobile)
	var result AdminUserPage
	err := c.do(ctx, http.MethodGet, "/user", query, nil, &result)
	return result, err
}

func (c *Client) CreateAdminUser(ctx context.Context, request AdminUserCreateRequest) error {
	request.Name = strings.TrimSpace(request.Name)
	request.Password = strings.TrimSpace(request.Password)
	request.Sex = strings.TrimSpace(request.Sex)
	request.DepartmentID = strings.TrimSpace(request.DepartmentID)
	request.RoleIDs = normalizedAdminIDs(request.RoleIDs)
	request.Mobile = strings.TrimSpace(request.Mobile)
	request.Email = strings.TrimSpace(request.Email)
	request.Status = strings.TrimSpace(request.Status)
	request.Remark = strings.TrimSpace(request.Remark)
	return c.do(ctx, http.MethodPost, "/user", nil, request, nil)
}

func (c *Client) UpdateAdminUser(ctx context.Context, request AdminUserUpdateRequest) error {
	request.ID = strings.TrimSpace(request.ID)
	request.Name = strings.TrimSpace(request.Name)
	request.Sex = strings.TrimSpace(request.Sex)
	request.DepartmentID = strings.TrimSpace(request.DepartmentID)
	request.RoleIDs = normalizedAdminIDs(request.RoleIDs)
	request.Mobile = strings.TrimSpace(request.Mobile)
	request.Email = strings.TrimSpace(request.Email)
	request.Status = strings.TrimSpace(request.Status)
	request.Remark = strings.TrimSpace(request.Remark)
	return c.do(ctx, http.MethodPut, "/user", nil, request, nil)
}

func (c *Client) UpdateAdminUserStatus(ctx context.Context, ids []string, status string) error {
	return c.do(ctx, http.MethodPatch, "/user/status", nil, AdminStatusRequest{
		IDs: normalizedAdminIDs(ids), Status: strings.TrimSpace(status),
	}, nil)
}

func (c *Client) ResetAdminUserPassword(ctx context.Context, id, password string) error {
	return c.do(ctx, http.MethodPatch, "/user/password", nil, map[string]string{
		"id": strings.TrimSpace(id), "password": strings.TrimSpace(password),
	}, nil)
}

func (c *Client) UpdateAdminUserRoles(ctx context.Context, id string, roleIDs []string) error {
	return c.do(ctx, http.MethodPatch, "/user/roles", nil, AdminUserRolesRequest{
		ID: strings.TrimSpace(id), RoleIDs: normalizedAdminIDs(roleIDs),
	}, nil)
}

type AdminDepartment struct {
	ID        string            `json:"id"`
	SortID    int64             `json:"sort_id"`
	ParentID  string            `json:"parent_id"`
	Name      string            `json:"name"`
	Code      string            `json:"code"`
	Remark    string            `json:"remark"`
	CreatedAt int64             `json:"created_at"`
	UpdatedAt int64             `json:"updated_at"`
	Children  []AdminDepartment `json:"children"`
}

type AdminDepartmentRequest struct {
	ID       string `json:"id,omitempty"`
	SortID   int64  `json:"sort_id"`
	ParentID string `json:"parent_id,omitempty"`
	Name     string `json:"name"`
	Code     string `json:"code"`
	Remark   string `json:"remark"`
}

func (c *Client) AdminDepartments(ctx context.Context) ([]AdminDepartment, error) {
	var result []AdminDepartment
	err := c.do(ctx, http.MethodGet, "/department", nil, nil, &result)
	return result, err
}

func (c *Client) CreateAdminDepartment(ctx context.Context, request AdminDepartmentRequest) error {
	request.ID = ""
	request = normalizeAdminDepartmentRequest(request)
	return c.do(ctx, http.MethodPost, "/department", nil, request, nil)
}

func (c *Client) UpdateAdminDepartment(ctx context.Context, request AdminDepartmentRequest) error {
	request = normalizeAdminDepartmentRequest(request)
	return c.do(ctx, http.MethodPut, "/department", nil, request, nil)
}

func normalizeAdminDepartmentRequest(request AdminDepartmentRequest) AdminDepartmentRequest {
	request.ID = strings.TrimSpace(request.ID)
	request.ParentID = strings.TrimSpace(request.ParentID)
	request.Name = strings.TrimSpace(request.Name)
	request.Code = strings.TrimSpace(request.Code)
	request.Remark = strings.TrimSpace(request.Remark)
	return request
}

type AdminRole struct {
	ID        string `json:"id"`
	ParentID  string `json:"parent_id"`
	Status    string `json:"status"`
	Name      string `json:"name"`
	Remark    string `json:"remark"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

type AdminRolePage struct {
	Total int64       `json:"total"`
	List  []AdminRole `json:"list"`
}

type AdminRoleRequest struct {
	ID       string `json:"id,omitempty"`
	ParentID string `json:"parent_id,omitempty"`
	Status   string `json:"status"`
	Name     string `json:"name"`
	Remark   string `json:"remark,omitempty"`
}

func (c *Client) AdminRoles(ctx context.Context, page, size int, name string) (AdminRolePage, error) {
	query := url.Values{}
	query.Set("page", strconv.Itoa(page))
	query.Set("size", strconv.Itoa(size))
	setTrimmedQuery(query, "name", name)
	var result AdminRolePage
	err := c.do(ctx, http.MethodGet, "/role", query, nil, &result)
	return result, err
}

func (c *Client) AdminRoleList(ctx context.Context, name string) ([]AdminRole, error) {
	query := url.Values{}
	setTrimmedQuery(query, "name", name)
	var result AdminRolePage
	err := c.do(ctx, http.MethodGet, "/role/list", query, nil, &result)
	return result.List, err
}

func (c *Client) CreateAdminRole(ctx context.Context, request AdminRoleRequest) error {
	request.ID = ""
	request = normalizeAdminRoleRequest(request)
	return c.do(ctx, http.MethodPost, "/role", nil, request, nil)
}

func (c *Client) UpdateAdminRole(ctx context.Context, request AdminRoleRequest) error {
	request = normalizeAdminRoleRequest(request)
	return c.do(ctx, http.MethodPut, "/role", nil, request, nil)
}

func (c *Client) UpdateAdminRoleStatus(ctx context.Context, ids []string, status string) error {
	return c.do(ctx, http.MethodPatch, "/role/status", nil, AdminStatusRequest{
		IDs: normalizedAdminIDs(ids), Status: strings.TrimSpace(status),
	}, nil)
}

func normalizeAdminRoleRequest(request AdminRoleRequest) AdminRoleRequest {
	request.ID = strings.TrimSpace(request.ID)
	request.ParentID = strings.TrimSpace(request.ParentID)
	request.Status = strings.TrimSpace(request.Status)
	request.Name = strings.TrimSpace(request.Name)
	request.Remark = strings.TrimSpace(request.Remark)
	return request
}

type AdminMenuMeta struct {
	Title      string `json:"title"`
	Icon       string `json:"icon"`
	Transition string `json:"transition"`
	Hidden     bool   `json:"hidden"`
	Fixed      bool   `json:"fixed"`
	IsFull     bool   `json:"is_full"`
	Perms      string `json:"perms"`
}

type AdminMenu struct {
	ID        string        `json:"id"`
	Type      int64         `json:"type"`
	SortID    int64         `json:"sort_id"`
	ParentID  string        `json:"parent_id"`
	Path      string        `json:"path"`
	Name      string        `json:"name"`
	Component string        `json:"component"`
	Meta      AdminMenuMeta `json:"meta"`
	Remark    string        `json:"remark"`
	CreatedAt int64         `json:"created_at"`
	UpdatedAt int64         `json:"updated_at"`
	Children  []AdminMenu   `json:"children"`
}

type AdminAPI struct {
	ID        string     `json:"id"`
	Type      int64      `json:"type"`
	SortID    int64      `json:"sort_id"`
	ParentID  string     `json:"parent_id"`
	URI       string     `json:"uri"`
	Method    string     `json:"method"`
	Name      string     `json:"name"`
	Remark    string     `json:"remark"`
	CreatedAt int64      `json:"created_at"`
	UpdatedAt int64      `json:"updated_at"`
	Children  []AdminAPI `json:"children"`
}

func (c *Client) AdminMenus(ctx context.Context) ([]AdminMenu, error) {
	var result []AdminMenu
	err := c.do(ctx, http.MethodGet, "/menu/list", nil, nil, &result)
	return result, err
}

func (c *Client) AdminAPIs(ctx context.Context) ([]AdminAPI, error) {
	var result []AdminAPI
	err := c.do(ctx, http.MethodGet, "/api", nil, nil, &result)
	return result, err
}

func (c *Client) AdminRoleMenuIDs(ctx context.Context, roleID string) ([]string, error) {
	query := url.Values{"id": []string{strings.TrimSpace(roleID)}}
	var result []string
	err := c.do(ctx, http.MethodGet, "/role/menus", query, nil, &result)
	return result, err
}

func (c *Client) SetAdminRoleMenuIDs(ctx context.Context, roleID string, menuIDs []string) error {
	return c.do(ctx, http.MethodPost, "/role/menus", nil, map[string]any{
		"id": strings.TrimSpace(roleID), "menus_id": normalizedAdminIDs(menuIDs),
	}, nil)
}

func (c *Client) AdminRoleAPIIDs(ctx context.Context, roleID string) ([]string, error) {
	query := url.Values{"id": []string{strings.TrimSpace(roleID)}}
	var result []string
	err := c.do(ctx, http.MethodGet, "/role/apis", query, nil, &result)
	return result, err
}

func (c *Client) SetAdminRoleAPIIDs(ctx context.Context, roleID string, apiIDs []string) error {
	return c.do(ctx, http.MethodPost, "/role/apis", nil, map[string]any{
		"id": strings.TrimSpace(roleID), "apis_id": normalizedAdminIDs(apiIDs),
	}, nil)
}

func normalizedAdminIDs(ids []string) []string {
	seen := make(map[string]struct{}, len(ids))
	result := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}
