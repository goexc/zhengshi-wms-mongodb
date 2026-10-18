package api

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type DocumentHistory struct {
	Action       string `json:"action"`
	Remark       string `json:"remark"`
	Time         int64  `json:"time"`
	OperatorName string `json:"operator_name"`
}

// CustomerDocument contains the shared and type-specific server fields. Each kind
// has its own endpoint; unused fields are omitted from write requests.
type CustomerDocument struct {
	ID                 string            `json:"id,omitempty"`
	CustomerID         string            `json:"customer_id"`
	CustomerName       string            `json:"customer_name,omitempty"`
	BillNumber         string            `json:"bill_number,omitempty"`
	InvoiceNumber      string            `json:"invoice_number,omitempty"`
	Amount             float64           `json:"amount,omitempty"`
	ReceivedDate       int64             `json:"received_date,omitempty"`
	IssueDate          int64             `json:"issue_date,omitempty"`
	DueDate            int64             `json:"due_date,omitempty"`
	RedemptionDate     int64             `json:"redemption_date,omitempty"`
	InvoiceDate        int64             `json:"invoice_date,omitempty"`
	TaxRate            float64           `json:"tax_rate,omitempty"`
	AmountExcludingTax float64           `json:"amount_excluding_tax,omitempty"`
	TaxAmount          float64           `json:"tax_amount,omitempty"`
	TotalAmount        float64           `json:"total_amount,omitempty"`
	Annex              []string          `json:"annex"`
	RedemptionAnnex    []string          `json:"redemption_annex,omitempty"`
	Remark             string            `json:"remark"`
	Status             string            `json:"status,omitempty"`
	Version            int64             `json:"version,omitempty"`
	TransactionID      string            `json:"transaction_id,omitempty"`
	CreatedAt          int64             `json:"created_at,omitempty"`
	UpdatedAt          int64             `json:"updated_at,omitempty"`
	CreatorName        string            `json:"creator_name,omitempty"`
	History            []DocumentHistory `json:"history,omitempty"`
	IdempotencyKey     string            `json:"idempotency_key,omitempty"`
}
type DocumentSummary struct {
	Count              int64   `json:"count"`
	Amount             float64 `json:"amount"`
	PendingAmount      float64 `json:"pending_amount"`
	OverdueAmount      float64 `json:"overdue_amount"`
	RedeemedAmount     float64 `json:"redeemed_amount"`
	VoidedAmount       float64 `json:"voided_amount"`
	AmountExcludingTax float64 `json:"amount_excluding_tax"`
	TaxAmount          float64 `json:"tax_amount"`
	TotalAmount        float64 `json:"total_amount"`
}
type CustomerDocumentPage struct {
	Total   int64              `json:"total"`
	List    []CustomerDocument `json:"list"`
	Summary DocumentSummary    `json:"summary"`
}

// Write DTOs deliberately retain zero values. A zero-rate invoice still carries
// tax_rate and tax_amount, and response-only audit/status fields never get sent.
type CustomerInvoiceWriteRequest struct {
	ID                 string   `json:"id,omitempty"`
	Version            int64    `json:"version,omitempty"`
	CustomerID         string   `json:"customer_id"`
	InvoiceNumber      string   `json:"invoice_number"`
	InvoiceDate        int64    `json:"invoice_date"`
	AmountExcludingTax float64  `json:"amount_excluding_tax"`
	TaxRate            float64  `json:"tax_rate"`
	TaxAmount          float64  `json:"tax_amount"`
	TotalAmount        float64  `json:"total_amount"`
	Annex              []string `json:"annex"`
	Remark             string   `json:"remark"`
	IdempotencyKey     string   `json:"idempotency_key,omitempty"`
}
type CustomerAcceptanceWriteRequest struct {
	ID             string   `json:"id,omitempty"`
	Version        int64    `json:"version,omitempty"`
	CustomerID     string   `json:"customer_id"`
	BillNumber     string   `json:"bill_number"`
	Amount         float64  `json:"amount"`
	ReceivedDate   int64    `json:"received_date"`
	IssueDate      int64    `json:"issue_date"`
	DueDate        int64    `json:"due_date"`
	Annex          []string `json:"annex"`
	Remark         string   `json:"remark"`
	IdempotencyKey string   `json:"idempotency_key,omitempty"`
}
type DocumentFilters struct {
	DueState, TaxRate                    string
	RedemptionStart, RedemptionEnd       int64
	CustomerID, Keyword, Status          string
	DateStart, DateEnd, DueStart, DueEnd int64
}
type DocumentActionRequest struct {
	ID              string   `json:"id"`
	Version         int64    `json:"version"`
	RedemptionDate  int64    `json:"redemption_date,omitempty"`
	RedemptionAnnex []string `json:"redemption_annex"`
	Remark          string   `json:"remark,omitempty"`
	Reason          string   `json:"reason,omitempty"`
	TransactionID   string   `json:"transaction_id,omitempty"`
	IdempotencyKey  string   `json:"idempotency_key"`
}
type AcceptancePayment struct {
	ID         string   `json:"id"`
	Code       string   `json:"code"`
	CustomerID string   `json:"customer_id"`
	Amount     float64  `json:"amount"`
	Time       int64    `json:"time"`
	Remark     string   `json:"remark"`
	Annex      []string `json:"annex"`
}
type AcceptancePaymentPage struct {
	Total int64               `json:"total"`
	List  []AcceptancePayment `json:"list"`
}

func documentPath(kind string) (string, error) {
	if kind != "acceptance" && kind != "invoice" {
		return "", fmt.Errorf("无效台账类型")
	}
	return "/customer/" + kind, nil
}
func (c *Client) CustomerDocuments(ctx context.Context, kind string, page, size int, f DocumentFilters) (CustomerDocumentPage, error) {
	var result CustomerDocumentPage
	path, err := documentPath(kind)
	if err != nil {
		return result, err
	}
	q := url.Values{"page": {fmt.Sprint(page)}, "size": {fmt.Sprint(size)}}
	setTrimmedQuery(q, "customer_id", f.CustomerID)
	setTrimmedQuery(q, "keyword", f.Keyword)
	setTrimmedQuery(q, "status", f.Status)
	setTrimmedQuery(q, "due_state", f.DueState)
	setTrimmedQuery(q, "tax_rate", f.TaxRate)
	for key, value := range map[string]int64{"date_start": f.DateStart, "date_end": f.DateEnd, "due_start": f.DueStart, "due_end": f.DueEnd, "redemption_start": f.RedemptionStart, "redemption_end": f.RedemptionEnd} {
		if value > 0 {
			q.Set(key, fmt.Sprint(value))
		}
	}
	err = c.do(ctx, http.MethodGet, path, q, nil, &result)
	return result, err
}
func (c *Client) CustomerDocument(ctx context.Context, kind, id string) (CustomerDocument, error) {
	var result CustomerDocument
	path, err := documentPath(kind)
	if err != nil {
		return result, err
	}
	err = c.do(ctx, http.MethodGet, path+"/detail", url.Values{"id": {id}}, nil, &result)
	return result, err
}
func (c *Client) SaveCustomerDocument(ctx context.Context, kind string, record CustomerDocument) (CustomerDocument, error) {
	var result CustomerDocument
	path, err := documentPath(kind)
	if err != nil {
		return result, err
	}
	method := http.MethodPost
	if record.ID != "" {
		method = http.MethodPut
	}
	annex := record.Annex
	if annex == nil {
		annex = []string{}
	}
	var request any
	if kind == "invoice" {
		request = CustomerInvoiceWriteRequest{ID: record.ID, Version: record.Version, CustomerID: record.CustomerID, InvoiceNumber: record.InvoiceNumber, InvoiceDate: record.InvoiceDate, AmountExcludingTax: record.AmountExcludingTax, TaxRate: record.TaxRate, TaxAmount: record.TaxAmount, TotalAmount: record.TotalAmount, Annex: annex, Remark: record.Remark, IdempotencyKey: record.IdempotencyKey}
	} else {
		request = CustomerAcceptanceWriteRequest{ID: record.ID, Version: record.Version, CustomerID: record.CustomerID, BillNumber: record.BillNumber, Amount: record.Amount, ReceivedDate: record.ReceivedDate, IssueDate: record.IssueDate, DueDate: record.DueDate, Annex: annex, Remark: record.Remark, IdempotencyKey: record.IdempotencyKey}
	}
	err = c.do(ctx, method, path, nil, request, &result)
	return result, err
}
func (c *Client) CustomerDocumentAction(ctx context.Context, kind, action string, req DocumentActionRequest) (CustomerDocument, error) {
	var result CustomerDocument
	path, err := documentPath(kind)
	if err != nil {
		return result, err
	}
	if action != "void" && !(kind == "acceptance" && (action == "redeem" || action == "reverse")) {
		return result, fmt.Errorf("无效台账操作")
	}
	err = c.do(ctx, http.MethodPost, path+"/"+action, nil, req, &result)
	return result, err
}
func (c *Client) AcceptancePayments(ctx context.Context, customerID string, amount float64, page, size int) (AcceptancePaymentPage, error) {
	var result AcceptancePaymentPage
	q := url.Values{"customer_id": {customerID}, "amount": {fmt.Sprint(amount)}, "page": {fmt.Sprint(page)}, "size": {fmt.Sprint(size)}}
	err := c.do(ctx, http.MethodGet, "/customer/acceptance/payments", q, nil, &result)
	return result, err
}
func (c *Client) UploadDocumentAttachment(ctx context.Context, filePath, scope string) (string, error) {
	if scope != "acceptance" && scope != "acceptance_redemption" && scope != "invoice" {
		return "", fmt.Errorf("无效附件用途")
	}
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() || info.Size() == 0 {
		return "", fmt.Errorf("请选择非空附件文件")
	}
	if info.Size() > 20<<20 {
		return "", fmt.Errorf("附件不能超过20MB")
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err = writer.WriteField("scope", scope); err != nil {
		return "", err
	}
	part, err := writer.CreateFormFile("files", filepath.Base(filePath))
	if err != nil {
		return "", err
	}
	n, err := io.Copy(part, io.LimitReader(file, (20<<20)+1))
	if err != nil {
		return "", err
	}
	if n > 20<<20 {
		return "", fmt.Errorf("附件不能超过20MB")
	}
	if err = writer.Close(); err != nil {
		return "", err
	}
	var result ImageURL
	err = c.doWithContentType(ctx, http.MethodPost, "/images", writer.FormDataContentType(), &body, &result)
	if err != nil {
		return "", err
	}
	if !validFinancialUploadReference(result.URL, scope) {
		return "", fmt.Errorf("上传响应的附件地址或用途不正确，请重新上传")
	}
	return result.URL, nil
}

func validFinancialUploadReference(reference, scope string) bool {
	if reference == "" || strings.TrimSpace(reference) != reference || strings.ContainsAny(reference, "\\,:?#%\x00\r\n") {
		return false
	}
	for _, part := range strings.Split(reference, "/") {
		if part == "" || part == "." || part == ".." {
			return false
		}
	}
	actual := ""
	switch {
	case strings.HasPrefix(reference, "acceptance/redemption/"):
		actual = "acceptance_redemption"
	case strings.HasPrefix(reference, "acceptance/"):
		actual = "acceptance"
	case strings.HasPrefix(reference, "invoice/"):
		actual = "invoice"
	}
	return actual == scope
}
