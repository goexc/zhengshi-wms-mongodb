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
	"sync"
	"time"
)

type Client struct {
	baseURL        string
	token          string
	http           *http.Client
	logger         Logger
	onUnauthorized func()
	operationMu    sync.RWMutex
	onOperation    OperationObserver
}

type Logger interface {
	Printf(format string, args ...any)
}

// OperationEvent describes one non-query request made by the authenticated
// client. It intentionally contains no token, request body or response body.
type OperationEvent struct {
	ID          string
	Method      string
	Path        string
	KeyField    string
	BusinessKey string
	StartedAt   time.Time
	FinishedAt  time.Time
	Outcome     string
	Message     string
}

type OperationObserver func(OperationEvent)

const (
	OperationPending   = "pending"
	OperationSucceeded = "success"
	OperationFailed    = "failed"
	OperationUnknown   = "unknown"
)

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
	Sex            string `json:"sex"`
	DepartmentID   string `json:"department_id"`
	DepartmentName string `json:"department_name"`
	Mobile         string `json:"mobile"`
	Email          string `json:"email"`
	Status         string `json:"status"`
	Avatar         string `json:"avatar"`
	Remark         string `json:"remark"`
	CreatedAt      int64  `json:"created_at"`
	UpdatedAt      int64  `json:"updated_at"`
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
	CategoryID       string  `json:"category_id"`
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
	Creator          string  `json:"creator"`
	CreatorName      string  `json:"creator_name"`
	CreatedAt        int64   `json:"created_at"`
	UpdatedAt        int64   `json:"updated_at"`
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
	SupplierID    string            `json:"supplier_id"`
	SupplierName  string            `json:"supplier_name"`
	CustomerID    string            `json:"customer_id"`
	CustomerName  string            `json:"customer_name"`
	ReceivingDate int64             `json:"receiving_date"`
	TotalAmount   float64           `json:"total_amount"`
	Materials     []InboundMaterial `json:"materials"`
	Annex         []string          `json:"annex"`
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
	CustomerID string
}

type InboundMaterialRequest struct {
	Index             int      `json:"index"`
	ID                string   `json:"id"`
	Price             float64  `json:"price"`
	EstimatedQuantity float64  `json:"estimated_quantity"`
	Position          []string `json:"position,omitempty"`
}

type InboundReceiptRequest struct {
	ID            string                   `json:"id,omitempty"`
	Code          string                   `json:"code"`
	Type          string                   `json:"type"`
	SupplierID    string                   `json:"supplier_id,omitempty"`
	CustomerID    string                   `json:"customer_id,omitempty"`
	TotalAmount   float64                  `json:"total_amount"`
	ReceivingDate int64                    `json:"receiving_date"`
	Materials     []InboundMaterialRequest `json:"materials"`
	Annex         []string                 `json:"annex,omitempty"`
	Remark        string                   `json:"remark,omitempty"`
}

type InboundCheckRequest struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type InboundRecordMaterial struct {
	ID                string  `json:"id"`
	Index             int     `json:"index"`
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
	ID               string                  `json:"id"`
	InboundReceiptID string                  `json:"inbound_receipt_id"`
	Code             string                  `json:"code"`
	CarrierName      string                  `json:"carrier_name"`
	CarrierCost      float64                 `json:"carrier_cost"`
	OtherCost        float64                 `json:"other_cost"`
	TotalAmount      float64                 `json:"total_amount"`
	ReceivingDate    int64                   `json:"receiving_date"`
	Materials        []InboundRecordMaterial `json:"materials"`
	Annex            []string                `json:"annex"`
	Remark           string                  `json:"remark"`
	CreatorName      string                  `json:"creator_name"`
	CreatedAt        int64                   `json:"created_at"`
}

type WarehouseNode struct {
	ID       string          `json:"id"`
	Name     string          `json:"name"`
	Children []WarehouseNode `json:"children"`
}

type MaterialCategory struct {
	ID          string             `json:"id"`
	ParentID    string             `json:"parent_id"`
	SortID      int                `json:"sort_id"`
	Name        string             `json:"name"`
	Image       string             `json:"image"`
	Status      string             `json:"status"`
	Remark      string             `json:"remark"`
	CreatorName string             `json:"creator_name"`
	CreatedAt   int64              `json:"created_at"`
	UpdatedAt   int64              `json:"updated_at"`
	Children    []MaterialCategory `json:"children"`
}

type Supplier struct {
	ID                            string `json:"id"`
	Type                          string `json:"type"`
	Name                          string `json:"name"`
	Code                          string `json:"code"`
	Image                         string `json:"image"`
	LegalRepresentative           string `json:"legal_representative"`
	UnifiedSocialCreditIdentifier string `json:"unified_social_credit_identifier"`
	Address                       string `json:"address"`
	Contact                       string `json:"contact"`
	Manager                       string `json:"manager"`
	Level                         int    `json:"level"`
	Email                         string `json:"email"`
	Remark                        string `json:"remark"`
	Status                        string `json:"status"`
	CreateBy                      string `json:"create_by"`
	CreatedAt                     int64  `json:"created_at"`
	UpdatedAt                     int64  `json:"updated_at"`
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
	ID                            string  `json:"id"`
	Type                          string  `json:"type"`
	Name                          string  `json:"name"`
	Code                          string  `json:"code"`
	Image                         string  `json:"image"`
	LegalRepresentative           string  `json:"legal_representative"`
	UnifiedSocialCreditIdentifier string  `json:"unified_social_credit_identifier"`
	Address                       string  `json:"address"`
	Contact                       string  `json:"contact"`
	Manager                       string  `json:"manager"`
	Email                         string  `json:"email"`
	Remark                        string  `json:"remark"`
	Status                        string  `json:"status"`
	ReceivableBalance             float64 `json:"receivable_balance"`
	CreditBalance                 float64 `json:"credit_balance"`
	CreateBy                      string  `json:"create_by"`
	CreatedAt                     int64   `json:"created_at"`
	UpdatedAt                     int64   `json:"updated_at"`
}

type CustomerPage struct {
	Total int64      `json:"total"`
	List  []Customer `json:"list"`
}

type Carrier struct {
	ID                            string `json:"id"`
	Type                          string `json:"type"`
	Name                          string `json:"name"`
	Code                          string `json:"code"`
	Image                         string `json:"image"`
	LegalRepresentative           string `json:"legal_representative"`
	UnifiedSocialCreditIdentifier string `json:"unified_social_credit_identifier"`
	Address                       string `json:"address"`
	Contact                       string `json:"contact"`
	Manager                       string `json:"manager"`
	Email                         string `json:"email"`
	Remark                        string `json:"remark"`
	Status                        string `json:"status"`
	CreateBy                      string `json:"create_by"`
	CreatedAt                     int64  `json:"created_at"`
	UpdatedAt                     int64  `json:"updated_at"`
}

type CarrierPage struct {
	Total int64     `json:"total"`
	List  []Carrier `json:"list"`
}

type PartnerFilters struct {
	Name    string
	Code    string
	Manager string
	Contact string
	Email   string
	Level   int
}

type WarehouseFilters struct {
	Type            string
	Name            string
	Code            string
	Status          string
	WarehouseID     string
	WarehouseZoneID string
	WarehouseRackID string
}

type Warehouse struct {
	ID           string  `json:"id"`
	Type         string  `json:"type"`
	Name         string  `json:"name"`
	Code         string  `json:"code"`
	Address      string  `json:"address"`
	Capacity     float64 `json:"capacity"`
	CapacityUnit string  `json:"capacity_unit"`
	Status       string  `json:"status"`
	Manager      string  `json:"manager"`
	Contact      string  `json:"contact"`
	Image        string  `json:"image"`
	Remark       string  `json:"remark"`
	CreateBy     string  `json:"create_by"`
	CreatedAt    int64   `json:"created_at"`
	UpdatedAt    int64   `json:"updated_at"`
}

type WarehousePage struct {
	Total int64       `json:"total"`
	List  []Warehouse `json:"list"`
}

type WarehouseZone struct {
	ID            string  `json:"id"`
	WarehouseID   string  `json:"warehouse_id"`
	WarehouseName string  `json:"warehouse_name"`
	Name          string  `json:"name"`
	Code          string  `json:"code"`
	Image         string  `json:"image"`
	Capacity      float64 `json:"capacity"`
	CapacityUnit  string  `json:"capacity_unit"`
	Status        string  `json:"status"`
	Manager       string  `json:"manager"`
	Contact       string  `json:"contact"`
	Remark        string  `json:"remark"`
	CreateBy      string  `json:"create_by"`
	CreatedAt     int64   `json:"created_at"`
	UpdatedAt     int64   `json:"updated_at"`
}

type WarehouseZonePage struct {
	Total int64           `json:"total"`
	List  []WarehouseZone `json:"list"`
}

type WarehouseRack struct {
	ID                string  `json:"id"`
	WarehouseID       string  `json:"warehouse_id"`
	WarehouseName     string  `json:"warehouse_name"`
	WarehouseZoneID   string  `json:"warehouse_zone_id"`
	WarehouseZoneName string  `json:"warehouse_zone_name"`
	Type              string  `json:"type"`
	Name              string  `json:"name"`
	Code              string  `json:"code"`
	Image             string  `json:"image"`
	Capacity          float64 `json:"capacity"`
	CapacityUnit      string  `json:"capacity_unit"`
	Status            string  `json:"status"`
	Manager           string  `json:"manager"`
	Contact           string  `json:"contact"`
	Remark            string  `json:"remark"`
	CreateBy          string  `json:"create_by"`
	CreatedAt         int64   `json:"created_at"`
	UpdatedAt         int64   `json:"updated_at"`
}

type WarehouseRackPage struct {
	Total int64           `json:"total"`
	List  []WarehouseRack `json:"list"`
}

type WarehouseBin struct {
	ID                string  `json:"id"`
	WarehouseID       string  `json:"warehouse_id"`
	WarehouseName     string  `json:"warehouse_name"`
	WarehouseZoneID   string  `json:"warehouse_zone_id"`
	WarehouseZoneName string  `json:"warehouse_zone_name"`
	WarehouseRackID   string  `json:"warehouse_rack_id"`
	WarehouseRackName string  `json:"warehouse_rack_name"`
	Name              string  `json:"name"`
	Code              string  `json:"code"`
	Image             string  `json:"image"`
	Capacity          float64 `json:"capacity"`
	CapacityUnit      string  `json:"capacity_unit"`
	Status            string  `json:"status"`
	Manager           string  `json:"manager"`
	Contact           string  `json:"contact"`
	Remark            string  `json:"remark"`
	CreateBy          string  `json:"create_by"`
	CreatedAt         int64   `json:"created_at"`
	UpdatedAt         int64   `json:"updated_at"`
}

type WarehouseBinPage struct {
	Total int64          `json:"total"`
	List  []WarehouseBin `json:"list"`
}

type OutboundFilters struct {
	Code       string
	Status     string
	IsPack     int
	IsWeigh    int
	Type       string
	Model      string
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

type OutboundMaterialRequest struct {
	Index      int     `json:"index"`
	MaterialID string  `json:"material_id"`
	Price      float64 `json:"price"`
	Quantity   float64 `json:"quantity"`
}

type OutboundOrderRequest struct {
	Code        string                    `json:"code"`
	Type        string                    `json:"type"`
	SupplierID  string                    `json:"supplier_id,omitempty"`
	CustomerID  string                    `json:"customer_id,omitempty"`
	TotalAmount float64                   `json:"total_amount"`
	Materials   []OutboundMaterialRequest `json:"materials"`
	Annex       []string                  `json:"annex,omitempty"`
	Remark      string                    `json:"remark,omitempty"`
}

type OutboundSummaryRecord struct {
	Code          string  `json:"code"`
	DepartureDate int64   `json:"departure_date"`
	ReceiptDate   int64   `json:"receipt_date"`
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

type MaterialPrice struct {
	Price               float64 `json:"price"`
	Since               int64   `json:"since"`
	CustomerID          string  `json:"customer_id"`
	CustomerName        string  `json:"customer_name"`
	SourceType          string  `json:"source_type"`
	SourceQuoteID       string  `json:"source_quote_id"`
	SourceDeliveryID    string  `json:"source_delivery_id"`
	SourceValid         bool    `json:"source_valid"`
	SourceInvalidReason string  `json:"source_invalid_reason"`
}

type CustomerTransaction struct {
	Type            string  `json:"type"`
	TransactionType string  `json:"transaction_type"`
	Direction       string  `json:"direction"`
	Status          string  `json:"status"`
	SourceType      string  `json:"source_type"`
	SourceCode      string  `json:"source_code"`
	Time            int64   `json:"time"`
	Amount          float64 `json:"amount"`
	Remark          string  `json:"remark"`
	Annex           string  `json:"annex"`
}

type CustomerTransactionPage struct {
	Total int64                 `json:"total"`
	List  []CustomerTransaction `json:"list"`
}

type OutboundMaterialPrice struct {
	MaterialID string  `json:"material_id"`
	Price      float64 `json:"price"`
}

type OutboundReviseRequest struct {
	Code           string                  `json:"code"`
	CustomerID     string                  `json:"customer_id"`
	MaterialsPrice []OutboundMaterialPrice `json:"materials_price"`
}

type ImageURL struct {
	URL string `json:"url"`
}

type BusinessError struct {
	Code int
	Msg  string
}

func (e *BusinessError) Error() string { return e.Msg }

type TransportError struct {
	Err error
}

func (e *TransportError) Error() string {
	if e.Timeout() {
		return "请求线上服务超时"
	}
	return "无法连接线上服务"
}

func (e *TransportError) Unwrap() error { return e.Err }

func (e *TransportError) Timeout() bool {
	type timeoutError interface{ Timeout() bool }
	var target timeoutError
	return errors.As(e.Err, &target) && target.Timeout()
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		// Each operation applies its own deadline. Keeping the transport-level
		// timeout unset lets caller cancellation and longer file transfers work
		// without weakening the shorter deadline used by normal API requests.
		http: &http.Client{},
	}
}

const (
	readRequestTimeout   = 25 * time.Second
	writeRequestTimeout  = 35 * time.Second
	imageUploadTimeout   = 90 * time.Second
	imageDownloadTimeout = 60 * time.Second
	fileDownloadTimeout  = 90 * time.Second
)

func operationTimeout(method, path string) time.Duration {
	if method == http.MethodPost && path == "/images" {
		return imageUploadTimeout
	}
	if method == http.MethodGet || method == http.MethodHead {
		return readRequestTimeout
	}
	return writeRequestTimeout
}

func withOperationTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithTimeout(ctx, timeout)
}

func (c *Client) SetToken(token string)                 { c.token = token }
func (c *Client) SetLogger(logger Logger)               { c.logger = logger }
func (c *Client) SetUnauthorizedHandler(handler func()) { c.onUnauthorized = handler }
func (c *Client) SetOperationObserver(observer OperationObserver) {
	c.operationMu.Lock()
	c.onOperation = observer
	c.operationMu.Unlock()
}

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

func (c *Client) ChangeAvatar(ctx context.Context, avatarURL string) error {
	return c.do(ctx, http.MethodPatch, "/account/avatar", nil, map[string]string{
		"avatar": strings.TrimSpace(avatarURL),
	}, nil)
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
	setTrimmedQuery(query, "customer_id", filters.CustomerID)
	return result, c.do(ctx, http.MethodGet, "/inbound/receipt", query, nil, &result)
}

func (c *Client) InboundRecords(ctx context.Context, receiptID string) ([]InboundRecord, error) {
	var result []InboundRecord
	query := url.Values{"inbound_receipt_id": {receiptID}}
	return result, c.do(ctx, http.MethodGet, "/inbound/receipt/receive", query, nil, &result)
}

func (c *Client) CreateInboundReceipt(ctx context.Context, request InboundReceiptRequest) error {
	request.ID = ""
	return c.do(ctx, http.MethodPost, "/inbound/receipt", nil, request, nil)
}

func (c *Client) UpdateInboundReceipt(ctx context.Context, request InboundReceiptRequest) error {
	return c.do(ctx, http.MethodPut, "/inbound/receipt", nil, request, nil)
}

func (c *Client) CheckInboundReceipt(ctx context.Context, receiptID, status string) error {
	return c.do(ctx, http.MethodPatch, "/inbound/receipt/check", nil, InboundCheckRequest{
		ID: receiptID, Status: status,
	}, nil)
}

func (c *Client) DeleteInboundReceipt(ctx context.Context, receiptID string) error {
	query := url.Values{"id": {strings.TrimSpace(receiptID)}}
	return c.do(ctx, http.MethodDelete, "/inbound/receipt", query, nil, nil)
}

func (c *Client) FindInboundReceipt(ctx context.Context, receiptID, code string) (InboundReceipt, bool, error) {
	receiptID = strings.TrimSpace(receiptID)
	code = strings.TrimSpace(code)
	result, err := c.InboundReceipts(ctx, 1, 100, InboundFilters{Code: code})
	if err != nil {
		return InboundReceipt{}, false, err
	}
	for _, receipt := range result.List {
		idMatches := receiptID == "" || strings.TrimSpace(receipt.ID) == receiptID
		codeMatches := code == "" || strings.EqualFold(strings.TrimSpace(receipt.Code), code)
		if idMatches && codeMatches {
			return receipt, true, nil
		}
	}
	return InboundReceipt{}, false, nil
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

func (c *Client) SupplierDirectory(ctx context.Context, page, size int, filters PartnerFilters) (SupplierPage, error) {
	var result SupplierPage
	query := partnerQuery(page, size, filters, true)
	return result, c.do(ctx, http.MethodGet, "/supplier", query, nil, &result)
}

func (c *Client) CustomerDirectory(ctx context.Context, page, size int, filters PartnerFilters) (CustomerPage, error) {
	var result CustomerPage
	query := partnerQuery(page, size, filters, false)
	return result, c.do(ctx, http.MethodGet, "/customer", query, nil, &result)
}

func (c *Client) CarrierDirectory(ctx context.Context, page, size int, filters PartnerFilters) (CarrierPage, error) {
	var result CarrierPage
	query := partnerQuery(page, size, filters, false)
	return result, c.do(ctx, http.MethodGet, "/carrier", query, nil, &result)
}

func partnerQuery(page, size int, filters PartnerFilters, includeLevel bool) url.Values {
	query := url.Values{"page": {fmt.Sprint(page)}, "size": {fmt.Sprint(size)}}
	setTrimmedQuery(query, "name", filters.Name)
	setTrimmedQuery(query, "code", filters.Code)
	setTrimmedQuery(query, "manager", filters.Manager)
	setTrimmedQuery(query, "contact", filters.Contact)
	setTrimmedQuery(query, "email", filters.Email)
	if includeLevel && filters.Level > 0 {
		query.Set("level", fmt.Sprint(filters.Level))
	}
	return query
}

func (c *Client) Warehouses(ctx context.Context, page, size int, filters WarehouseFilters) (WarehousePage, error) {
	var result WarehousePage
	query := warehouseQuery(page, size, filters)
	return result, c.do(ctx, http.MethodGet, "/warehouse", query, nil, &result)
}

func (c *Client) WarehouseZones(ctx context.Context, page, size int, filters WarehouseFilters) (WarehouseZonePage, error) {
	var result WarehouseZonePage
	query := warehouseQuery(page, size, filters)
	return result, c.do(ctx, http.MethodGet, "/warehouse_zone", query, nil, &result)
}

func (c *Client) WarehouseRacks(ctx context.Context, page, size int, filters WarehouseFilters) (WarehouseRackPage, error) {
	var result WarehouseRackPage
	query := warehouseQuery(page, size, filters)
	return result, c.do(ctx, http.MethodGet, "/warehouse_rack", query, nil, &result)
}

func (c *Client) WarehouseBins(ctx context.Context, page, size int, filters WarehouseFilters) (WarehouseBinPage, error) {
	var result WarehouseBinPage
	query := warehouseQuery(page, size, filters)
	return result, c.do(ctx, http.MethodGet, "/warehouse_bin", query, nil, &result)
}

func warehouseQuery(page, size int, filters WarehouseFilters) url.Values {
	query := url.Values{"page": {fmt.Sprint(page)}, "size": {fmt.Sprint(size)}}
	setTrimmedQuery(query, "type", filters.Type)
	setTrimmedQuery(query, "name", filters.Name)
	setTrimmedQuery(query, "code", filters.Code)
	setTrimmedQuery(query, "status", filters.Status)
	setTrimmedQuery(query, "warehouse_id", filters.WarehouseID)
	setTrimmedQuery(query, "warehouse_zone_id", filters.WarehouseZoneID)
	setTrimmedQuery(query, "warehouse_rack_id", filters.WarehouseRackID)
	return query
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
	setTrimmedQuery(query, "model", filters.Model)
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

func (c *Client) CreateOutbound(ctx context.Context, request OutboundOrderRequest) error {
	return c.do(ctx, http.MethodPost, "/outbound", nil, request, nil)
}

func (c *Client) DeleteOutbound(ctx context.Context, orderID string) error {
	query := url.Values{"id": {strings.TrimSpace(orderID)}}
	return c.do(ctx, http.MethodDelete, "/outbound", query, nil, nil)
}

func (c *Client) MaterialPrices(ctx context.Context, materialID, customerID string) ([]MaterialPrice, error) {
	var result []MaterialPrice
	query := url.Values{"material_id": {strings.TrimSpace(materialID)}}
	setTrimmedQuery(query, "customer_id", customerID)
	return result, c.do(ctx, http.MethodGet, "/material/price", query, nil, &result)
}

func (c *Client) CustomerTransactions(ctx context.Context, customerID string, page, size int) (CustomerTransactionPage, error) {
	var result CustomerTransactionPage
	query := url.Values{
		"customer_id": {strings.TrimSpace(customerID)},
		"page":        {fmt.Sprint(page)},
		"size":        {fmt.Sprint(size)},
	}
	return result, c.do(ctx, http.MethodGet, "/customer/transaction", query, nil, &result)
}

func (c *Client) OutboundSummary(ctx context.Context, customerID string, startDate, endDate int64) ([]OutboundSummaryRecord, error) {
	var result []OutboundSummaryRecord
	query := url.Values{
		"customer_id": {strings.TrimSpace(customerID)},
		"start_date":  {fmt.Sprint(startDate)},
		"end_date":    {fmt.Sprint(endDate)},
	}
	return result, c.do(ctx, http.MethodGet, "/outbound/summary", query, nil, &result)
}

func (c *Client) FindOutboundByCode(ctx context.Context, code string) (OutboundOrder, error) {
	result, found, err := c.FindOutbound(ctx, "", code)
	if err != nil {
		return OutboundOrder{}, err
	}
	if !found {
		return OutboundOrder{}, fmt.Errorf("未查询到出库单 %s", strings.TrimSpace(code))
	}
	return result, nil
}

func (c *Client) FindOutbound(ctx context.Context, orderID, code string) (OutboundOrder, bool, error) {
	orderID = strings.TrimSpace(orderID)
	code = strings.TrimSpace(code)
	result, err := c.OutboundOrders(ctx, 1, 50, OutboundFilters{Code: code, IsPack: -1, IsWeigh: -1})
	if err != nil {
		return OutboundOrder{}, false, err
	}
	for _, order := range result.List {
		codeMatches := code == "" || strings.EqualFold(strings.TrimSpace(order.Code), code)
		idMatches := orderID == "" || strings.TrimSpace(order.ID) == orderID
		if codeMatches && idMatches {
			return order, true, nil
		}
	}
	return OutboundOrder{}, false, nil
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

func (c *Client) ReviseOutbound(ctx context.Context, request OutboundReviseRequest) error {
	return c.do(ctx, http.MethodPatch, "/outbound/revise", nil, request, nil)
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
	requestContext, cancel := withOperationTimeout(ctx, imageDownloadTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(requestContext, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "image/*")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("下载图片失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("下载图片失败: HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxImageDownloadBytes+1))
	if err != nil {
		return nil, fmt.Errorf("读取图片失败: %w", err)
	}
	if len(data) > maxImageDownloadBytes {
		return nil, errors.New("图片文件超过 25 MB，无法预览")
	}
	if len(data) == 0 {
		return nil, errors.New("图片文件为空")
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
	var bodyData []byte
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyData = data
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
	keyField, businessKey := operationBusinessKey(query, bodyData)
	return c.request(ctx, method, path, target, contentType, reader, out, keyField, businessKey)
}

func (c *Client) doWithContentType(ctx context.Context, method, path, contentType string, body io.Reader, out any) error {
	return c.request(ctx, method, path, c.baseURL+path, contentType, body, out, "", "")
}

func (c *Client) request(ctx context.Context, method, path, target, contentType string, body io.Reader, out any, keyField, businessKey string) (requestErr error) {
	requestContext, cancel := withOperationTimeout(ctx, operationTimeout(method, path))
	defer cancel()
	started := time.Now()
	requestID := fmt.Sprintf("%d", time.Now().UnixNano())
	observe := method != http.MethodGet && method != http.MethodHead && path != "/auth/login" && path != "/auth/logout"
	if observe {
		c.notifyOperation(OperationEvent{
			ID: requestID, Method: method, Path: path,
			KeyField: keyField, BusinessKey: businessKey,
			StartedAt: started, Outcome: OperationPending, Message: "请求正在提交",
		})
		defer func() {
			outcome := OperationSucceeded
			message := "服务端已返回成功"
			if requestErr != nil {
				outcome = OperationFailed
				message = requestErr.Error()
				var transportErr *TransportError
				var businessErr *BusinessError
				serverFailure := errors.As(requestErr, &businessErr) && businessErr.Code >= http.StatusInternalServerError
				if errors.As(requestErr, &transportErr) || serverFailure || strings.Contains(requestErr.Error(), "无法识别的响应") {
					outcome = OperationUnknown
					message = "请求结果无法确认，请在线回读业务状态"
				}
			}
			c.notifyOperation(OperationEvent{
				ID: requestID, Method: method, Path: path,
				KeyField: keyField, BusinessKey: businessKey,
				StartedAt: started, FinishedAt: time.Now(), Outcome: outcome, Message: message,
			})
		}()
	}
	req, err := http.NewRequestWithContext(requestContext, method, target, body)
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
		return &TransportError{Err: err}
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
		if env.Code == http.StatusUnauthorized && c.onUnauthorized != nil {
			c.onUnauthorized()
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

func operationBusinessKey(query url.Values, body []byte) (string, string) {
	for _, key := range []string{"code", "order_code", "receipt_code", "id", "name"} {
		if value := strings.TrimSpace(query.Get(key)); value != "" {
			return key, value
		}
	}
	if len(body) == 0 {
		return "", ""
	}
	var values map[string]any
	if json.Unmarshal(body, &values) != nil {
		return "", ""
	}
	for _, key := range []string{"code", "order_code", "receipt_code", "id", "name"} {
		if value, ok := values[key].(string); ok && strings.TrimSpace(value) != "" {
			return key, strings.TrimSpace(value)
		}
	}
	return "", ""
}

func (c *Client) notifyOperation(event OperationEvent) {
	c.operationMu.RLock()
	observer := c.onOperation
	c.operationMu.RUnlock()
	if observer != nil {
		observer(event)
	}
}
