package code

import "strings"

const (
	QuoteStatusUnquoted = "unquoted"
	QuoteStatusQuoting  = "quoting"
	QuoteStatusQuoted   = "quoted"
	QuoteStatusPriced   = "priced"

	QuoteModeDetailed = "detailed"
	QuoteModeSimple   = "simple"

	MaterialQuoteStatusDraft     = "draft"
	MaterialQuoteStatusSubmitted = "submitted"
	MaterialQuoteStatusQuoted    = "quoted"
	MaterialQuoteStatusPriced    = "priced"
	MaterialQuoteStatusVoid      = "void"

	MaterialDeliveryRebuildTaskStatusQueued  = "queued"
	MaterialDeliveryRebuildTaskStatusRunning = "running"
	MaterialDeliveryRebuildTaskStatusSuccess = "success"
	MaterialDeliveryRebuildTaskStatusFailed  = "failed"
)

// IsCustomerDeliveryOutbound 判断出库类型是否属于客户侧交付，客户新增物料只统计这类出库。
func IsCustomerDeliveryOutbound(typeCode, legacyType string) bool {
	switch strings.TrimSpace(typeCode) {
	case OutboundTypeCodeSales, OutboundTypeCodeSample, OutboundTypeCodeGift:
		return true
	}

	switch strings.TrimSpace(legacyType) {
	case "销售出库", "样品出库", "赠品出库":
		return true
	default:
		return false
	}
}

// ValidQuoteMode 判断报价方式是否有效。
func ValidQuoteMode(mode string) bool {
	switch strings.TrimSpace(mode) {
	case QuoteModeDetailed, QuoteModeSimple:
		return true
	default:
		return false
	}
}

// ValidMaterialDeliveryRebuildTaskStatus reports whether a rebuild task status is supported.
func ValidMaterialDeliveryRebuildTaskStatus(status string) bool {
	switch strings.TrimSpace(status) {
	case MaterialDeliveryRebuildTaskStatusQueued,
		MaterialDeliveryRebuildTaskStatusRunning,
		MaterialDeliveryRebuildTaskStatusSuccess,
		MaterialDeliveryRebuildTaskStatusFailed:
		return true
	default:
		return false
	}
}
