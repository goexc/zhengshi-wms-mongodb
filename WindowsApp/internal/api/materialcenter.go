package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type MaterialRequest struct {
	ID               string  `json:"id,omitempty"`
	CategoryID       string  `json:"category_id,omitempty"`
	Name             string  `json:"name"`
	Model            string  `json:"model"`
	Image            string  `json:"image,omitempty"`
	Material         string  `json:"material,omitempty"`
	Specification    string  `json:"specification,omitempty"`
	SurfaceTreatment string  `json:"surface_treatment,omitempty"`
	StrengthGrade    string  `json:"strength_grade,omitempty"`
	Quantity         float64 `json:"quantity"`
	Unit             string  `json:"unit,omitempty"`
	Remark           string  `json:"remark,omitempty"`
	Price            float64 `json:"price,omitempty"`
}

type MaterialCategoryRequest struct {
	ID       string `json:"id,omitempty"`
	ParentID string `json:"parent_id,omitempty"`
	SortID   int    `json:"sort_id"`
	Name     string `json:"name"`
	Status   string `json:"status"`
	Remark   string `json:"remark,omitempty"`
}

type NewCustomerMaterialFilters struct {
	CustomerID    string
	StartTime     int64
	EndTime       int64
	QuoteStatus   string
	MaterialName  string
	MaterialModel string
}

type NewCustomerMaterial struct {
	ID                     string  `json:"id"`
	CustomerID             string  `json:"customer_id"`
	CustomerName           string  `json:"customer_name"`
	MaterialID             string  `json:"material_id"`
	MaterialName           string  `json:"material_name"`
	MaterialModel          string  `json:"material_model"`
	MaterialSpecification  string  `json:"material_specification"`
	MaterialUnit           string  `json:"material_unit"`
	FirstDeliveryTime      int64   `json:"first_delivery_time"`
	FirstDeliveryOrderCode string  `json:"first_delivery_order_code"`
	FirstDeliveryQuantity  float64 `json:"first_delivery_quantity"`
	FirstDeliveryPrice     float64 `json:"first_delivery_price"`
	QuoteStatus            string  `json:"quote_status"`
	LatestQuoteID          string  `json:"latest_quote_id"`
	LatestQuoteNo          string  `json:"latest_quote_no"`
	LatestPrice            float64 `json:"latest_price"`
}

type NewCustomerMaterialPage struct {
	Total int64                 `json:"total"`
	List  []NewCustomerMaterial `json:"list"`
}

type NewCustomerMaterialExportRequest struct {
	CustomerID    string `json:"customer_id"`
	StartTime     int64  `json:"start_time"`
	EndTime       int64  `json:"end_time"`
	QuoteStatus   string `json:"quote_status,omitempty"`
	MaterialName  string `json:"material_name,omitempty"`
	MaterialModel string `json:"material_model,omitempty"`
}

type MaterialQuoteCostItem struct {
	Index        int     `json:"index"`
	CategoryCode string  `json:"category_code"`
	CategoryName string  `json:"category_name"`
	Name         string  `json:"name"`
	Enabled      bool    `json:"enabled"`
	Custom       bool    `json:"custom"`
	Amount       float64 `json:"amount"`
	Remark       string  `json:"remark"`
}

type MaterialQuote struct {
	ID                    string                  `json:"id"`
	QuoteNo               string                  `json:"quote_no"`
	CustomerID            string                  `json:"customer_id"`
	CustomerName          string                  `json:"customer_name"`
	MaterialID            string                  `json:"material_id"`
	MaterialName          string                  `json:"material_name"`
	MaterialModel         string                  `json:"material_model"`
	MaterialSpecification string                  `json:"material_specification"`
	MaterialUnit          string                  `json:"material_unit"`
	DeliveryID            string                  `json:"delivery_id"`
	SourceOrderCode       string                  `json:"source_order_code"`
	QuoteMode             string                  `json:"quote_mode"`
	Status                string                  `json:"status"`
	Currency              string                  `json:"currency"`
	CostItems             []MaterialQuoteCostItem `json:"cost_items"`
	SimplePrice           float64                 `json:"simple_price"`
	TotalCost             float64                 `json:"total_cost"`
	ProfitRate            float64                 `json:"profit_rate"`
	ProfitAmount          float64                 `json:"profit_amount"`
	TaxRate               float64                 `json:"tax_rate"`
	TaxAmount             float64                 `json:"tax_amount"`
	FinalPrice            float64                 `json:"final_price"`
	TotalAmount           float64                 `json:"total_amount"`
	ValidFrom             int64                   `json:"valid_from"`
	ValidTo               int64                   `json:"valid_to"`
	Remark                string                  `json:"remark"`
	SourceValid           bool                    `json:"source_valid"`
	SourceInvalidReason   string                  `json:"source_invalid_reason"`
	CreatorID             string                  `json:"creator_id"`
	CreatorName           string                  `json:"creator_name"`
	CreatedAt             int64                   `json:"created_at"`
	UpdatedAt             int64                   `json:"updated_at"`
}

type MaterialQuoteSaveRequest struct {
	ID           string                  `json:"id,omitempty"`
	DeliveryID   string                  `json:"delivery_id"`
	QuoteMode    string                  `json:"quote_mode"`
	Currency     string                  `json:"currency,omitempty"`
	CostItems    []MaterialQuoteCostItem `json:"cost_items,omitempty"`
	SimplePrice  float64                 `json:"simple_price,omitempty"`
	ProfitAmount float64                 `json:"profit_amount,omitempty"`
	TaxRate      float64                 `json:"tax_rate,omitempty"`
	FinalPrice   float64                 `json:"final_price,omitempty"`
	ValidFrom    int64                   `json:"valid_from,omitempty"`
	ValidTo      int64                   `json:"valid_to,omitempty"`
	Remark       string                  `json:"remark,omitempty"`
}

type MaterialQuoteFilters struct {
	CustomerID    string
	MaterialID    string
	DeliveryID    string
	Status        string
	QuoteMode     string
	MaterialName  string
	MaterialModel string
}

type MaterialQuotePage struct {
	Total int64           `json:"total"`
	List  []MaterialQuote `json:"list"`
}

type MaterialQuotePriceRequest struct {
	ID          string  `json:"id"`
	FinalPrice  float64 `json:"final_price"`
	EffectiveAt int64   `json:"effective_at,omitempty"`
	Remark      string  `json:"remark,omitempty"`
}

type MaterialDeliveryRebuildTask struct {
	ID            string `json:"id"`
	Status        string `json:"status"`
	OrderCount    int64  `json:"order_count"`
	DeliveryCount int64  `json:"delivery_count"`
	Message       string `json:"message"`
	ErrorMessage  string `json:"error_message"`
	CreatorID     string `json:"creator_id"`
	CreatorName   string `json:"creator_name"`
	CreatedAt     int64  `json:"created_at"`
	StartedAt     int64  `json:"started_at"`
	FinishedAt    int64  `json:"finished_at"`
	UpdatedAt     int64  `json:"updated_at"`
}

type MaterialDeliveryRebuildTaskPage struct {
	Total int64                         `json:"total"`
	List  []MaterialDeliveryRebuildTask `json:"list"`
}

func (c *Client) CreateMaterial(ctx context.Context, request MaterialRequest) error {
	request.ID = ""
	return c.do(ctx, http.MethodPost, "/material", nil, request, nil)
}

func (c *Client) UpdateMaterial(ctx context.Context, request MaterialRequest) error {
	return c.do(ctx, http.MethodPut, "/material", nil, request, nil)
}

func (c *Client) MaterialInfo(ctx context.Context, id string) (Material, error) {
	var result Material
	query := url.Values{"id": {strings.TrimSpace(id)}}
	return result, c.do(ctx, http.MethodGet, "/material/info", query, nil, &result)
}

func (c *Client) CreateMaterialCategory(ctx context.Context, request MaterialCategoryRequest) error {
	request.ID = ""
	return c.do(ctx, http.MethodPost, "/material/category", nil, request, nil)
}

func (c *Client) UpdateMaterialCategory(ctx context.Context, request MaterialCategoryRequest) error {
	return c.do(ctx, http.MethodPut, "/material/category", nil, request, nil)
}

func (c *Client) NewCustomerMaterials(ctx context.Context, page, size int, filters NewCustomerMaterialFilters) (NewCustomerMaterialPage, error) {
	var result NewCustomerMaterialPage
	query := url.Values{
		"page":        {fmt.Sprint(page)},
		"size":        {fmt.Sprint(size)},
		"customer_id": {strings.TrimSpace(filters.CustomerID)},
		"start_time":  {fmt.Sprint(filters.StartTime)},
		"end_time":    {fmt.Sprint(filters.EndTime)},
	}
	setTrimmedQuery(query, "quote_status", filters.QuoteStatus)
	setTrimmedQuery(query, "material_name", filters.MaterialName)
	setTrimmedQuery(query, "material_model", filters.MaterialModel)
	return result, c.do(ctx, http.MethodGet, "/material/new_delivery", query, nil, &result)
}

func (c *Client) MaterialQuotes(ctx context.Context, page, size int, filters MaterialQuoteFilters) (MaterialQuotePage, error) {
	var result MaterialQuotePage
	query := url.Values{"page": {fmt.Sprint(page)}, "size": {fmt.Sprint(size)}}
	setTrimmedQuery(query, "customer_id", filters.CustomerID)
	setTrimmedQuery(query, "material_id", filters.MaterialID)
	setTrimmedQuery(query, "delivery_id", filters.DeliveryID)
	setTrimmedQuery(query, "status", filters.Status)
	setTrimmedQuery(query, "quote_mode", filters.QuoteMode)
	setTrimmedQuery(query, "material_name", filters.MaterialName)
	setTrimmedQuery(query, "material_model", filters.MaterialModel)
	return result, c.do(ctx, http.MethodGet, "/material/quote", query, nil, &result)
}

func (c *Client) MaterialQuoteInfo(ctx context.Context, id string) (MaterialQuote, error) {
	var result MaterialQuote
	query := url.Values{"id": {strings.TrimSpace(id)}}
	return result, c.do(ctx, http.MethodGet, "/material/quote/info", query, nil, &result)
}

func (c *Client) SaveMaterialQuote(ctx context.Context, request MaterialQuoteSaveRequest) (MaterialQuote, error) {
	var result MaterialQuote
	method := http.MethodPut
	if strings.TrimSpace(request.ID) == "" {
		method = http.MethodPost
	}
	return result, c.do(ctx, method, "/material/quote", nil, request, &result)
}

func (c *Client) SubmitMaterialQuote(ctx context.Context, id string) (MaterialQuote, error) {
	var result MaterialQuote
	return result, c.do(ctx, http.MethodPost, "/material/quote/submit", nil, map[string]string{"id": strings.TrimSpace(id)}, &result)
}

func (c *Client) PriceMaterialQuote(ctx context.Context, request MaterialQuotePriceRequest) (MaterialQuote, error) {
	var result MaterialQuote
	return result, c.do(ctx, http.MethodPost, "/material/quote/price", nil, request, &result)
}

func (c *Client) VoidMaterialQuote(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodPatch, "/material/quote/void", nil, map[string]string{"id": strings.TrimSpace(id)}, nil)
}

func (c *Client) ExportNewCustomerMaterialQuotes(ctx context.Context, request NewCustomerMaterialExportRequest) ([]byte, string, error) {
	return c.download(ctx, http.MethodPost, "/material/new_delivery/quote/export", request, "new-material-quotes.csv")
}

func (c *Client) ExportMaterialQuote(ctx context.Context, id string) ([]byte, string, error) {
	return c.download(ctx, http.MethodPost, "/material/quote/export", map[string]string{"id": strings.TrimSpace(id)}, "material-quote.csv")
}

func (c *Client) StartMaterialDeliveryRebuild(ctx context.Context) (MaterialDeliveryRebuildTask, error) {
	var result MaterialDeliveryRebuildTask
	return result, c.do(ctx, http.MethodPost, "/material/new_delivery/rebuild", nil, map[string]any{}, &result)
}

func (c *Client) LatestMaterialDeliveryRebuild(ctx context.Context) (MaterialDeliveryRebuildTask, error) {
	var result MaterialDeliveryRebuildTask
	return result, c.do(ctx, http.MethodGet, "/material/new_delivery/rebuild/latest", nil, nil, &result)
}

func (c *Client) MaterialDeliveryRebuildTasks(ctx context.Context, page, size int, status string) (MaterialDeliveryRebuildTaskPage, error) {
	var result MaterialDeliveryRebuildTaskPage
	query := url.Values{"page": {fmt.Sprint(page)}, "size": {fmt.Sprint(size)}}
	setTrimmedQuery(query, "status", status)
	return result, c.do(ctx, http.MethodGet, "/material/new_delivery/rebuild/tasks", query, nil, &result)
}

func (c *Client) download(ctx context.Context, method, path string, body any, fallbackName string) ([]byte, string, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, "", err
	}
	target := c.baseURL + path
	requestContext, cancel := withOperationTimeout(ctx, fileDownloadTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(requestContext, method, target, bytes.NewReader(data))
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Accept", "text/csv, application/json")
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", c.token)
	}
	started := time.Now()
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, "", &TransportError{Err: err}
	}
	defer resp.Body.Close()
	result, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return nil, "", err
	}
	if c.logger != nil {
		c.logger.Printf("method=%s path=%s http=%d duration=%s result=download", method, path, resp.StatusCode, time.Since(started))
	}
	contentType := strings.ToLower(resp.Header.Get("Content-Type"))
	if strings.Contains(contentType, "application/json") || resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var env envelope
		if jsonErr := json.Unmarshal(result, &env); jsonErr != nil {
			return nil, "", fmt.Errorf("导出接口返回异常（HTTP %d）", resp.StatusCode)
		}
		if env.Msg == "" {
			env.Msg = http.StatusText(env.Code)
		}
		if env.Code == http.StatusUnauthorized && c.onUnauthorized != nil {
			c.onUnauthorized()
		}
		return nil, "", &BusinessError{Code: env.Code, Msg: env.Msg}
	}
	if len(result) == 0 {
		return nil, "", fmt.Errorf("导出文件为空")
	}
	fileName := fallbackName
	if _, params, parseErr := mime.ParseMediaType(resp.Header.Get("Content-Disposition")); parseErr == nil {
		if candidate := strings.TrimSpace(params["filename"]); candidate != "" {
			fileName = candidate
		}
	}
	return result, fileName, nil
}
