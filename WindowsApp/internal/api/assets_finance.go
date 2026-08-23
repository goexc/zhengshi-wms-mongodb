package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// ImageAsset is the exact item returned by GET /images.
type ImageAsset struct {
	URL string `json:"url"`
	Alt string `json:"alt"`
}

type ImageAssetPage struct {
	Total int64        `json:"total"`
	List  []ImageAsset `json:"list"`
}

func (c *Client) ImageAssets(ctx context.Context, page, size int, name string) (ImageAssetPage, error) {
	var result ImageAssetPage
	query := url.Values{
		"page": {fmt.Sprint(page)},
		"size": {fmt.Sprint(size)},
	}
	setTrimmedQuery(query, "name", name)
	return result, c.do(ctx, http.MethodGet, "/images", query, nil, &result)
}

const (
	CustomerTransactionPayment      = "payment"
	CustomerTransactionReturnCredit = "return_credit"
)

type CustomerTransactionAddRequest struct {
	CustomerID      string   `json:"customer_id"`
	Time            int64    `json:"time"`
	TransactionType string   `json:"transaction_type"`
	IdempotencyKey  string   `json:"idempotency_key"`
	Amount          float64  `json:"amount"`
	Annex           []string `json:"annex,omitempty"`
	Remark          string   `json:"remark,omitempty"`
}

func (request CustomerTransactionAddRequest) Normalized() CustomerTransactionAddRequest {
	request.CustomerID = strings.TrimSpace(request.CustomerID)
	request.TransactionType = strings.TrimSpace(request.TransactionType)
	request.IdempotencyKey = strings.TrimSpace(request.IdempotencyKey)
	request.Remark = strings.TrimSpace(request.Remark)
	cleaned := make([]string, 0, len(request.Annex))
	for _, reference := range request.Annex {
		if reference = strings.TrimSpace(reference); reference != "" {
			cleaned = append(cleaned, reference)
		}
	}
	request.Annex = cleaned
	return request
}

func (c *Client) AddCustomerTransaction(ctx context.Context, request CustomerTransactionAddRequest) error {
	request = request.Normalized()
	return c.do(ctx, http.MethodPost, "/customer/transaction", nil, request, nil)
}
