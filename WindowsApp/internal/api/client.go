package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Client struct {
	baseURL string
	token   string
	http    *http.Client
	logger  Logger
}

type Logger interface {
	Printf(format string, args ...any)
}

type envelope struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

type LoginData struct {
	Name           string `json:"name"`
	Mobile         string `json:"mobile"`
	DepartmentName string `json:"department_name"`
	Token          string `json:"token"`
	Exp            int64  `json:"exp"`
}

type Profile struct {
	Name           string `json:"name"`
	DepartmentName string `json:"department_name"`
	Mobile         string `json:"mobile"`
	Status         string `json:"status"`
}

type Menu struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Path     string `json:"path"`
	Type     int64  `json:"type"`
	Children []Menu `json:"children"`
}

type Perms struct {
	Menus   []Menu   `json:"menus"`
	Buttons []Button `json:"buttons"`
}

type Button struct {
	Name  string `json:"name"`
	Perms string `json:"perms"`
}

type Material struct {
	ID               string  `json:"id"`
	Image            string  `json:"image"`
	CategoryName     string  `json:"category_name"`
	Name             string  `json:"name"`
	Model            string  `json:"model"`
	Material         string  `json:"material"`
	Specification    string  `json:"specification"`
	SurfaceTreatment string  `json:"surface_treatment"`
	StrengthGrade    string  `json:"strength_grade"`
	Quantity         float64 `json:"quantity"`
	Unit             string  `json:"unit"`
	Remark           string  `json:"remark"`
}

type Inventory struct {
	ID                string  `json:"id"`
	Type              string  `json:"type"`
	ReceiptCode       string  `json:"receipt_code"`
	ReceiveCode       string  `json:"receive_code"`
	WarehouseName     string  `json:"warehouse_name"`
	WarehouseZoneName string  `json:"warehouse_zone_name"`
	WarehouseRackName string  `json:"warehouse_rack_name"`
	WarehouseBinName  string  `json:"warehouse_bin_name"`
	Name              string  `json:"name"`
	Model             string  `json:"model"`
	Unit              string  `json:"unit"`
	Quantity          float64 `json:"quantity"`
	AvailableQuantity float64 `json:"available_quantity"`
	LockedQuantity    float64 `json:"locked_quantity"`
	FrozenQuantity    float64 `json:"frozen_quantity"`
}

type MaterialPage struct {
	Total int64      `json:"total"`
	List  []Material `json:"list"`
}

type MaterialFilters struct {
	Name             string
	CategoryID       string
	Material         string
	Specification    string
	Model            string
	SurfaceTreatment string
	StrengthGrade    string
}

type InventoryPage struct {
	Total    int64       `json:"total"`
	Quantity float64     `json:"quantity"`
	List     []Inventory `json:"list"`
}

type InventoryFilters struct {
	Type            string
	MaterialName    string
	MaterialModel   string
	WarehouseID     string
	WarehouseZoneID string
	WarehouseRackID string
	WarehouseBinID  string
}

type InboundMaterial struct {
	Index             int     `json:"index"`
	ID                string  `json:"id"`
	Name              string  `json:"name"`
	Model             string  `json:"model"`
	Unit              string  `json:"unit"`
	Price             float64 `json:"price"`
	EstimatedQuantity float64 `json:"estimated_quantity"`
	ActualQuantity    float64 `json:"actual_quantity"`
	Status            string  `json:"status"`
}

type InboundReceipt struct {
	ID            string            `json:"id"`
	Code          string            `json:"code"`
	Status        string            `json:"status"`
	Type          string            `json:"type"`
	SupplierName  string            `json:"supplier_name"`
	CustomerName  string            `json:"customer_name"`
	ReceivingDate int64             `json:"receiving_date"`
	TotalAmount   float64           `json:"total_amount"`
	Materials     []InboundMaterial `json:"materials"`
	Remark        string            `json:"remark"`
}

type InboundPage struct {
	Total int64            `json:"total"`
	List  []InboundReceipt `json:"list"`
}

type InboundFilters struct {
	Code       string
	Status     string
	Type       string
	SupplierID string
}

type InboundRecordMaterial struct {
	ID                string  `json:"id"`
	Name              string  `json:"name"`
	Model             string  `json:"model"`
	Unit              string  `json:"unit"`
	ActualQuantity    float64 `json:"actual_quantity"`
	Status            string  `json:"status"`
	WarehouseName     string  `json:"warehouse_name"`
	WarehouseZoneName string  `json:"warehouse_zone_name"`
	WarehouseRackName string  `json:"warehouse_rack_name"`
	WarehouseBinName  string  `json:"warehouse_bin_name"`
}

type InboundRecord struct {
	ID            string                  `json:"id"`
	Code          string                  `json:"code"`
	ReceivingDate int64                   `json:"receiving_date"`
	Materials     []InboundRecordMaterial `json:"materials"`
	Remark        string                  `json:"remark"`
	CreatorName   string                  `json:"creator_name"`
}

type WarehouseNode struct {
	ID       string          `json:"id"`
	Name     string          `json:"name"`
	Children []WarehouseNode `json:"children"`
}

type MaterialCategory struct {
	ID       string             `json:"id"`
	Name     string             `json:"name"`
	Children []MaterialCategory `json:"children"`
}

type Supplier struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Code string `json:"code"`
}

type SupplierPage struct {
	Total int64      `json:"total"`
	List  []Supplier `json:"list"`
}

type ReceiveMaterial struct {
	Index          int      `json:"index"`
	ID             string   `json:"id"`
	Price          float64  `json:"price"`
	ActualQuantity float64  `json:"actual_quantity"`
	Position       []string `json:"position"`
	Status         string   `json:"status"`
}

type ReceiveRequest struct {
	ID            string            `json:"id"`
	Code          string            `json:"code"`
	ReceivingDate int64             `json:"receiving_date"`
	CarrierID     string            `json:"carrier_id,omitempty"`
	CarrierCost   float64           `json:"carrier_cost"`
	OtherCost     float64           `json:"other_cost"`
	Materials     []ReceiveMaterial `json:"materials"`
	Remark        string            `json:"remark,omitempty"`
}

type Customer struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Code string `json:"code"`
}

type CustomerPage struct {
	Total int64      `json:"total"`
	List  []Customer `json:"list"`
}

type Carrier struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Code   string `json:"code"`
	Status string `json:"status"`
}

type CarrierPage struct {
	Total int64     `json:"total"`
	List  []Carrier `json:"list"`
}

type OutboundFilters struct {
	Code       string
	Status     string
	IsPack     int
	IsWeigh    int
	Type       string
	SupplierID string
	CustomerID string
	StartTime  int64
	EndTime    int64
}

type OutboundOrder struct {
	ID            string             `json:"id"`
	Code          string             `json:"code"`
	Status        string             `json:"status"`
	IsPack        int                `json:"is_pack"`
	IsWeigh       int                `json:"is_weigh"`
	Type          string             `json:"type"`
	SupplierID    string             `json:"supplier_id"`
	SupplierName  string             `json:"supplier_name"`
	CustomerID    string             `json:"customer_id"`
	CustomerName  string             `json:"customer_name"`
	CarrierID     string             `json:"carrier_id"`
	CarrierName   string             `json:"carrier_name"`
	CarrierCost   float64            `json:"carrier_cost"`
	OtherCost     float64            `json:"other_cost"`
	ConfirmTime   int64              `json:"confirm_time"`
	PickingTime   int64              `json:"picking_time"`
	PackingTime   int64              `json:"packing_time"`
	WeighingTime  int64              `json:"weighing_time"`
	DepartureTime int64              `json:"departure_time"`
	ReceiptTime   int64              `json:"receipt_time"`
	TotalAmount   float64            `json:"total_amount"`
	Annex         []string           `json:"annex"`
	Remark        string             `json:"remark"`
	Materials     []OutboundMaterial `json:"materials"`
}

type OutboundPage struct {
	Total int64           `json:"total"`
	List  []OutboundOrder `json:"list"`
}

type OutboundMaterial struct {
	ID            string  `json:"id"`
	OrderCode     string  `json:"order_code"`
	MaterialID    string  `json:"material_id"`
	Index         int     `json:"index"`
	Name          string  `json:"name"`
	Model         string  `json:"model"`
	Specification string  `json:"specification"`
	Price         float64 `json:"price"`
	Quantity      float64 `json:"quantity"`
	Weight        float64 `json:"weight"`
	Unit          string  `json:"unit"`
}

type OutboundConfirmInventory struct {
	InventoryID      string  `json:"inventory_id"`
	ShipmentQuantity float64 `json:"shipment_quantity"`
}

type OutboundConfirmMaterial struct {
	MaterialID string                     `json:"material_id"`
	Index      int                        `json:"index"`
	Inventorys []OutboundConfirmInventory `json:"inventorys"`
}

type OutboundConfirmRequest struct {
	Code        string                    `json:"code"`
	ConfirmTime int64                     `json:"confirm_time"`
	Materials   []OutboundConfirmMaterial `json:"materials"`
}

type OutboundPickRequest struct {
	Code        string `json:"code"`
	PickingTime int64  `json:"picking_time"`
}

type OutboundPackRequest struct {
	Code        string `json:"code"`
	PackingTime int64  `json:"packing_time"`
}

type OutboundWeighMaterial struct {
	MaterialID string  `json:"material_id"`
	Weight     float64 `json:"weight"`
}

type OutboundWeighRequest struct {
	Code         string                  `json:"code"`
	WeighingTime int64                   `json:"weighing_time"`
	Materials    []OutboundWeighMaterial `json:"materials"`
}

type OutboundDepartureRequest struct {
	Code          string  `json:"code"`
	DepartureTime int64   `json:"departure_time"`
	CarrierID     string  `json:"carrier_id,omitempty"`
	CarrierCost   float64 `json:"carrier_cost"`
	OtherCost     float64 `json:"other_cost"`
}

type OutboundReceiptRequest struct {
	Code        string   `json:"code"`
	ReceiptTime int64    `json:"receipt_time"`
	Annex       []string `json:"annex"`
}

type ImageURL struct {
	URL string `json:"url"`
}

type BusinessError struct {
	Code int
	Msg  string
}

func (e *BusinessError) Error() string { return e.Msg }

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		http:    &http.Client{Timeout: 20 * time.Second},
	}
}

func (c *Client) SetToken(token string)   { c.token = token }
func (c *Client) SetLogger(logger Logger) { c.logger = logger }

func (c *Client) Login(ctx context.Context, mobile, password string) (LoginData, error) {
	var result LoginData
	err := c.do(ctx, http.MethodPost, "/auth/login", nil, map[string]string{
		"mobile": mobile, "password": password, "device_type": "windows",
	}, &result)
	if err == nil && result.Token == "" {
		err = errors.New("登录成功响应中没有 Token")
	}
	return result, err
}

func (c *Client) Profile(ctx context.Context) (Profile, error) {
	var result Profile
	return result, c.do(ctx, http.MethodGet, "/account/profile", nil, nil, &result)
}

func (c *Client) Permissions(ctx context.Context) (Perms, error) {
	var result Perms
	return result, c.do(ctx, http.MethodGet, "/account/menu", nil, nil, &result)
}

func (c *Client) Materials(ctx context.Context, page, size int, filters MaterialFilters) (MaterialPage, error) {
	var result MaterialPage
	query := url.Values{"page": {fmt.Sprint(page)}, "size": {fmt.Sprint(size)}}
	setTrimmedQuery(query, "name", filters.Name)
	setTrimmedQuery(query, "category_id", filters.CategoryID)
	setTrimmedQuery(query, "material", filters.Material)
	setTrimmedQuery(query, "specification", filters.Specification)
	setTrimmedQuery(query, "model", filters.Model)
	setTrimmedQuery(query, "surface_treatment", filters.SurfaceTreatment)
	setTrimmedQuery(query, "strength_grade", filters.StrengthGrade)
	return result, c.do(ctx, http.MethodGet, "/material", query, nil, &result)
}

func (c *Client) Inventory(ctx context.Context, page, size int, filters InventoryFilters) (InventoryPage, error) {
	return c.inventoryPage(ctx, "/inventory", page, size, filters)
}

func (c *Client) InventoryHistory(ctx context.Context, page, size int, filters InventoryFilters) (InventoryPage, error) {
	return c.inventoryPage(ctx, "/inventory/record", page, size, filters)
}

func (c *Client) inventoryPage(ctx context.Context, path string, page, size int, filters InventoryFilters) (InventoryPage, error) {
	var result InventoryPage
	query := url.Values{"page": {fmt.Sprint(page)}, "size": {fmt.Sprint(size)}}
	setTrimmedQuery(query, "type", filters.Type)
	setTrimmedQuery(query, "material_name", filters.MaterialName)
	setTrimmedQuery(query, "material_model", filters.MaterialModel)
	setTrimmedQuery(query, "warehouse_id", filters.WarehouseID)
	setTrimmedQuery(query, "warehouse_zone_id", filters.WarehouseZoneID)
	setTrimmedQuery(query, "warehouse_rack_id", filters.WarehouseRackID)
	setTrimmedQuery(query, "warehouse_bin_id", filters.WarehouseBinID)
	return result, c.do(ctx, http.MethodGet, path, query, nil, &result)
}

func (c *Client) InventoryByMaterial(ctx context.Context, materialID string) ([]Inventory, error) {
	var result []Inventory
	query := url.Values{"material_id": {materialID}}
	return result, c.do(ctx, http.MethodGet, "/inventory/list", query, nil, &result)
}

func (c *Client) InboundReceipts(ctx context.Context, page, size int, filters InboundFilters) (InboundPage, error) {
	var result InboundPage
	query := url.Values{"page": {fmt.Sprint(page)}, "size": {fmt.Sprint(size)}}
	setTrimmedQuery(query, "code", filters.Code)
	setTrimmedQuery(query, "status", filters.Status)
	setTrimmedQuery(query, "type", filters.Type)
	setTrimmedQuery(query, "supplier_id", filters.SupplierID)
	return result, c.do(ctx, http.MethodGet, "/inbound/receipt", query, nil, &result)
}

func (c *Client) InboundRecords(ctx context.Context, receiptID string) ([]InboundRecord, error) {
	var result []InboundRecord
	query := url.Values{"inbound_receipt_id": {receiptID}}
	return result, c.do(ctx, http.MethodGet, "/inbound/receipt/receive", query, nil, &result)
}

func (c *Client) WarehouseTree(ctx context.Context) ([]WarehouseNode, error) {
	var result []WarehouseNode
	return result, c.do(ctx, http.MethodGet, "/warehouse/tree", nil, nil, &result)
}

func (c *Client) MaterialCategories(ctx context.Context) ([]MaterialCategory, error) {
	var result []MaterialCategory
	return result, c.do(ctx, http.MethodGet, "/material/category", nil, nil, &result)
}

func (c *Client) Suppliers(ctx context.Context) ([]Supplier, error) {
	var result SupplierPage
	err := c.do(ctx, http.MethodGet, "/supplier/list", nil, nil, &result)
	return result.List, err
}

func (c *Client) Customers(ctx context.Context) ([]Customer, error) {
	var result CustomerPage
	err := c.do(ctx, http.MethodGet, "/customer/list", nil, nil, &result)
	return result.List, err
}

func (c *Client) Carriers(ctx context.Context) ([]Carrier, error) {
	var result CarrierPage
	query := url.Values{"page": {"1"}, "size": {"100"}}
	err := c.do(ctx, http.MethodGet, "/carrier", query, nil, &result)
	return result.List, err
}

func (c *Client) ReceiveInbound(ctx context.Context, request ReceiveRequest) error {
	return c.do(ctx, http.MethodPost, "/inbound/receipt/receive", nil, request, nil)
}

func (c *Client) OutboundOrders(ctx context.Context, page, size int, filters OutboundFilters) (OutboundPage, error) {
	var result OutboundPage
	query := url.Values{
		"page":     {fmt.Sprint(page)},
		"size":     {fmt.Sprint(size)},
		"is_pack":  {fmt.Sprint(filters.IsPack)},
		"is_weigh": {fmt.Sprint(filters.IsWeigh)},
	}
	setTrimmedQuery(query, "code", filters.Code)
	setTrimmedQuery(query, "status", filters.Status)
	setTrimmedQuery(query, "type", filters.Type)
	setTrimmedQuery(query, "supplier_id", filters.SupplierID)
	setTrimmedQuery(query, "customer_id", filters.CustomerID)
	if filters.StartTime > 0 {
		query.Set("start_time", fmt.Sprint(filters.StartTime))
	}
	if filters.EndTime > 0 {
		query.Set("end_time", fmt.Sprint(filters.EndTime))
	}
	return result, c.do(ctx, http.MethodGet, "/outbound/page", query, nil, &result)
}

func (c *Client) OutboundMaterials(ctx context.Context, orderCode string) ([]OutboundMaterial, error) {
	var result []OutboundMaterial
	query := url.Values{"order_code": {strings.TrimSpace(orderCode)}}
	return result, c.do(ctx, http.MethodGet, "/outbound/materials", query, nil, &result)
}

func (c *Client) FindOutboundByCode(ctx context.Context, code string) (OutboundOrder, error) {
	code = strings.TrimSpace(code)
	result, err := c.OutboundOrders(ctx, 1, 50, OutboundFilters{Code: code, IsPack: -1, IsWeigh: -1})
	if err != nil {
		return OutboundOrder{}, err
	}
	for _, order := range result.List {
		if strings.EqualFold(strings.TrimSpace(order.Code), code) {
			return order, nil
		}
	}
	return OutboundOrder{}, fmt.Errorf("未查询到出库单 %s", code)
}

func (c *Client) ConfirmOutbound(ctx context.Context, request OutboundConfirmRequest) error {
	return c.do(ctx, http.MethodPatch, "/outbound/confirm", nil, request, nil)
}

func (c *Client) PickOutbound(ctx context.Context, request OutboundPickRequest) error {
	return c.do(ctx, http.MethodPatch, "/outbound/pick", nil, request, nil)
}

func (c *Client) PackOutbound(ctx context.Context, request OutboundPackRequest) error {
	return c.do(ctx, http.MethodPatch, "/outbound/pack", nil, request, nil)
}

func (c *Client) WeighOutbound(ctx context.Context, request OutboundWeighRequest) error {
	return c.do(ctx, http.MethodPatch, "/outbound/weigh", nil, request, nil)
}

func (c *Client) DepartOutbound(ctx context.Context, request OutboundDepartureRequest) error {
	return c.do(ctx, http.MethodPatch, "/outbound/departure", nil, request, nil)
}

func (c *Client) ReceiptOutbound(ctx context.Context, request OutboundReceiptRequest) error {
	return c.do(ctx, http.MethodPatch, "/outbound/receipt", nil, request, nil)
}

func (c *Client) UploadImage(ctx context.Context, filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("files", filepath.Base(filePath))
	if err != nil {
		return "", err
	}
	if _, err = io.Copy(part, file); err != nil {
		return "", err
	}
	if err = writer.Close(); err != nil {
		return "", err
	}

	var result ImageURL
	err = c.doWithContentType(ctx, http.MethodPost, "/images", writer.FormDataContentType(), &body, &result)
	return result.URL, err
}

const maxImageDownloadBytes = 25 << 20

func ResolveImageURL(baseURL, reference string) (string, error) {
	reference = strings.TrimSpace(reference)
	if reference == "" {
		return "", errors.New("图片地址为空")
	}
	ref, err := url.Parse(reference)
	if err != nil {
		return "", fmt.Errorf("图片地址无效: %w", err)
	}
	if ref.IsAbs() {
		if ref.Scheme != "http" && ref.Scheme != "https" {
			return "", fmt.Errorf("不支持的图片地址协议: %s", ref.Scheme)
		}
		return ref.String(), nil
	}

	base, err := url.Parse(strings.TrimRight(strings.TrimSpace(baseURL), "/") + "/")
	if err != nil || (base.Scheme != "http" && base.Scheme != "https") || base.Host == "" {
		return "", errors.New("图片服务地址无效")
	}
	ref.Path = strings.TrimLeft(ref.Path, "/")
	return base.ResolveReference(ref).String(), nil
}

func (c *Client) DownloadImage(ctx context.Context, imageURL string) ([]byte, error) {
	parsed, err := url.Parse(strings.TrimSpace(imageURL))
	if err != nil || !parsed.IsAbs() || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, errors.New("图片下载地址无效")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "image/*")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("下载图纸失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("下载图纸失败: HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxImageDownloadBytes+1))
	if err != nil {
		return nil, fmt.Errorf("读取图纸失败: %w", err)
	}
	if len(data) > maxImageDownloadBytes {
		return nil, errors.New("图纸文件超过 25 MB，无法预览")
	}
	if len(data) == 0 {
		return nil, errors.New("图纸文件为空")
	}
	return data, nil
}

func (c *Client) Logout(ctx context.Context) error {
	return c.do(ctx, http.MethodPost, "/auth/logout", nil, nil, nil)
}

func setTrimmedQuery(query url.Values, key, value string) {
	if value = strings.TrimSpace(value); value != "" {
		query.Set(key, value)
	}
}

func (c *Client) do(ctx context.Context, method, path string, query url.Values, body any, out any) error {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(data)
	}
	target := c.baseURL + path
	if len(query) > 0 {
		target += "?" + query.Encode()
	}
	contentType := ""
	if body != nil {
		contentType = "application/json"
	}
	return c.request(ctx, method, path, target, contentType, reader, out)
}

func (c *Client) doWithContentType(ctx context.Context, method, path, contentType string, body io.Reader, out any) error {
	return c.request(ctx, method, path, c.baseURL+path, contentType, body, out)
}

func (c *Client) request(ctx context.Context, method, path, target, contentType string, body io.Reader, out any) error {
	started := time.Now()
	requestID := fmt.Sprintf("%d", time.Now().UnixNano())
	req, err := http.NewRequestWithContext(ctx, method, target, body)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if c.token != "" {
		// Existing Web client sends the raw JWT value, without a Bearer prefix.
		req.Header.Set("Authorization", c.token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		if c.logger != nil {
			c.logger.Printf("request_id=%s method=%s path=%s duration=%s result=network_error", requestID, method, path, time.Since(started))
		}
		return fmt.Errorf("无法连接线上服务：%w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return err
	}
	var env envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return fmt.Errorf("服务返回了无法识别的响应（HTTP %d）", resp.StatusCode)
	}
	if c.logger != nil {
		c.logger.Printf("request_id=%s method=%s path=%s http=%d code=%d duration=%s", requestID, method, path, resp.StatusCode, env.Code, time.Since(started))
	}
	if env.Code != http.StatusOK {
		if env.Msg == "" {
			env.Msg = http.StatusText(env.Code)
		}
		return &BusinessError{Code: env.Code, Msg: env.Msg}
	}
	if out != nil && len(env.Data) > 0 && string(env.Data) != "null" {
		if err := json.Unmarshal(env.Data, out); err != nil {
			return fmt.Errorf("响应数据解析失败：%w", err)
		}
	}
	return nil
}
