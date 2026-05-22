package code

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

const (
	// CustomerStatusCodeDeleted is the ASCII status code used by indexes and
	// backend predicates. The legacy Chinese status is kept only for display.
	CustomerStatusCodeDeleted = "deleted"

	TransactionTypeOpeningAR        = "opening_ar"
	TransactionTypeOutboundAR       = "outbound_ar"
	TransactionTypeARAdjustment     = "ar_adjustment"
	TransactionTypePayment          = "payment"
	TransactionTypeReturnCredit     = "return_credit"
	TransactionTypeManualAdjustment = "manual_adjustment"

	TransactionDirectionIncrease = "receivable_increase"
	TransactionDirectionDecrease = "receivable_decrease"

	TransactionStatusDraft     = "draft"
	TransactionStatusConfirmed = "confirmed"
	TransactionStatusReversed  = "reversed"
	TransactionStatusVoided    = "voided"

	SourceTypeOpening          = "opening"
	SourceTypeOutboundOrder    = "outbound_order"
	SourceTypeInboundReturn    = "inbound_return"
	SourceTypePayment          = "payment"
	SourceTypeManualAdjustment = "manual_adjustment"
	SourceTypeSystemRecount    = "system_recount"

	OutboundTypeCodeSales  = "sales_outbound"
	OutboundTypeCodeSample = "sample_outbound"
	OutboundTypeCodeGift   = "gift_outbound"

	OutboundPriceStatusUnpriced      = "unpriced"
	OutboundPriceStatusPartialPriced = "partial_priced"
	OutboundPriceStatusFullyPriced   = "fully_priced"

	OutboundARStatusPending       = "pending"
	OutboundARStatusPosted        = "posted"
	OutboundARStatusAdjusted      = "adjusted"
	OutboundARStatusNotApplicable = "not_applicable"
)

// CustomerNameFingerprint stores an ASCII hash of the normalized customer name.
// It lets MongoDB enforce uniqueness without placing Chinese names in an index.
func CustomerNameFingerprint(name string) string {
	normalized := strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(name)), " "))
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:])
}

func CustomerStatusCodeFromLabel(status string) string {
	switch strings.TrimSpace(status) {
	case "删除":
		return CustomerStatusCodeDeleted
	default:
		return ""
	}
}

// TransactionTypeCode normalizes new ASCII transaction codes and old Chinese
// labels into the canonical code stored in indexed fields.
func TransactionTypeCode(value string) (string, bool) {
	switch strings.TrimSpace(value) {
	case TransactionTypeOpeningAR, TransactionTypeOutboundAR, TransactionTypeARAdjustment,
		TransactionTypePayment, TransactionTypeReturnCredit, TransactionTypeManualAdjustment:
		return strings.TrimSpace(value), true
	case "期初应收":
		return TransactionTypeOpeningAR, true
	case "应收账款":
		return TransactionTypeOutboundAR, true
	case "回款":
		return TransactionTypePayment, true
	case "退货":
		return TransactionTypeReturnCredit, true
	default:
		return "", false
	}
}

func ManualTransactionTypeCode(value string) (string, bool) {
	switch strings.TrimSpace(value) {
	case TransactionTypePayment, "回款":
		return TransactionTypePayment, true
	case TransactionTypeReturnCredit, "退货":
		return TransactionTypeReturnCredit, true
	case TransactionTypeManualAdjustment, TransactionTypeARAdjustment, "应收账款":
		return TransactionTypeManualAdjustment, true
	default:
		return "", false
	}
}

func TransactionTypeLabel(typeCode, legacyType string) string {
	switch strings.TrimSpace(typeCode) {
	case TransactionTypeOpeningAR:
		return "期初应收"
	case TransactionTypeOutboundAR:
		return "应收账款"
	case TransactionTypeARAdjustment:
		return "应收调整"
	case TransactionTypePayment:
		return "回款"
	case TransactionTypeReturnCredit:
		return "退货冲减"
	case TransactionTypeManualAdjustment:
		return "手工调整"
	default:
		return strings.TrimSpace(legacyType)
	}
}

func TransactionDefaultDirection(typeCode string) string {
	switch strings.TrimSpace(typeCode) {
	case TransactionTypePayment, TransactionTypeReturnCredit:
		return TransactionDirectionDecrease
	default:
		return TransactionDirectionIncrease
	}
}

func TransactionBalanceDelta(typeCode, direction string, amount float64) (float64, bool) {
	if amount < 0 {
		return 0, false
	}
	typeCode = strings.TrimSpace(typeCode)
	direction = strings.TrimSpace(direction)
	switch strings.TrimSpace(typeCode) {
	case TransactionTypeOpeningAR, TransactionTypeOutboundAR:
		if direction == "" {
			direction = TransactionDirectionIncrease
		}
		if direction != TransactionDirectionIncrease {
			return 0, false
		}
	case TransactionTypePayment, TransactionTypeReturnCredit:
		if direction == "" {
			direction = TransactionDirectionDecrease
		}
		if direction != TransactionDirectionDecrease {
			return 0, false
		}
	case TransactionTypeARAdjustment, TransactionTypeManualAdjustment:
		if direction == "" {
			return 0, false
		}
	default:
		return 0, false
	}

	switch direction {
	case TransactionDirectionIncrease:
		return amount, true
	case TransactionDirectionDecrease:
		return -amount, true
	default:
		return 0, false
	}
}

func TransactionStatusCountsInBalance(status string) bool {
	switch strings.TrimSpace(status) {
	case "", TransactionStatusConfirmed:
		return true
	default:
		return false
	}
}

func IdempotencyKey(parts ...string) string {
	clean := make([]string, 0, len(parts))
	for _, part := range parts {
		clean = append(clean, strings.TrimSpace(part))
	}
	return strings.Join(clean, ":")
}

func OutboundTypeCodeFromLabel(label string) string {
	switch strings.TrimSpace(label) {
	case "销售出库", OutboundTypeCodeSales:
		return OutboundTypeCodeSales
	case "样品出库", OutboundTypeCodeSample:
		return OutboundTypeCodeSample
	case "赠品出库", OutboundTypeCodeGift:
		return OutboundTypeCodeGift
	default:
		return ""
	}
}

func OutboundTypeLabel(typeCode, legacyType string) string {
	switch strings.TrimSpace(typeCode) {
	case OutboundTypeCodeSales:
		return "销售出库"
	case OutboundTypeCodeSample:
		return "样品出库"
	case OutboundTypeCodeGift:
		return "赠品出库"
	default:
		return strings.TrimSpace(legacyType)
	}
}

// ShouldCreateReceivable centralizes the finance policy for customer outbound
// orders. Gifts are treated as non-receivable by default.
func ShouldCreateReceivable(outboundTypeCode string) bool {
	switch strings.TrimSpace(outboundTypeCode) {
	case OutboundTypeCodeSales, OutboundTypeCodeSample:
		return true
	default:
		return false
	}
}

func TransactionCode(prefix string, unixMilli int64) string {
	return fmt.Sprintf("%s-%d", strings.TrimSpace(prefix), unixMilli)
}
