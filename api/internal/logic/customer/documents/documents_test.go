package documents

import (
	"api/internal/types"
	"api/model"
	code "api/pkg/code"
	"math"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestMoneyBoundaryUsesExactCents(t *testing.T) {
	for _, tt := range []struct {
		value float64
		zero  bool
		want  int64
		ok    bool
	}{
		{0.01, false, 1, true}, {100.25, false, 10025, true}, {0, true, 0, true}, {0, false, 0, false},
		{1.001, false, 0, false}, {-1, false, 0, false}, {math.NaN(), false, 0, false}, {math.Inf(1), false, 0, false}, {maxMoney + 1, false, 0, false},
	} {
		got, err := moneyMinor(tt.value, tt.zero)
		if (err == nil) != tt.ok || (tt.ok && got != tt.want) {
			t.Errorf("moneyMinor(%v)=%d,%v", tt.value, got, err)
		}
	}
}

func TestInvoiceFaceAmountsAllowRoundingButMustBalance(t *testing.T) {
	req := types.CustomerInvoiceSaveRequest{CustomerId: primitive.NewObjectID().Hex(), InvoiceNumber: "FP-1", InvoiceDate: 1700000000, TaxRate: 13, AmountExcludingTax: 100, TaxAmount: 13.01, TotalAmount: 113.01}
	if _, err := prepareInvoice(&req); err != nil {
		t.Fatalf("face rounding should be accepted: %v", err)
	}
	req.TotalAmount = 113
	if _, err := prepareInvoice(&req); err == nil {
		t.Fatal("unbalanced invoice accepted")
	}
	req.TotalAmount = 113.01
	req.TaxRate = 101
	if _, err := prepareInvoice(&req); err == nil {
		t.Fatal("invalid tax rate accepted")
	}
}

func TestAcceptanceDatesPermitOverdueReceipt(t *testing.T) {
	req := types.CustomerAcceptanceSaveRequest{CustomerId: primitive.NewObjectID().Hex(), BillNumber: "B1", Amount: 100, IssueDate: 1700000000, DueDate: 1700000100, ReceivedDate: 1700000200}
	if _, err := prepareAcceptance(&req); err != nil {
		t.Fatalf("overdue receipt rejected: %v", err)
	}
	req.IssueDate = 0
	if _, err := prepareAcceptance(&req); err == nil {
		t.Fatal("missing issue date accepted")
	}
	req.IssueDate = 1700000000 + 86400
	if _, err := prepareAcceptance(&req); err == nil {
		t.Fatal("issue after receipt accepted")
	}
}

func TestAcceptanceDatesCompareBeijingDaysWithoutChangingSeconds(t *testing.T) {
	day := time.Date(2026, 10, 17, 16, 0, 0, 0, time.UTC).Unix() // October 18 in Beijing.
	req := types.CustomerAcceptanceSaveRequest{CustomerId: primitive.NewObjectID().Hex(), BillNumber: "DAY", Amount: 100, IssueDate: day + 23*3600, ReceivedDate: day, DueDate: day + 12*3600}
	prepared, err := prepareAcceptance(&req)
	if err != nil {
		t.Fatalf("same Beijing day rejected because of hidden times: %v", err)
	}
	if prepared.IssueDate != req.IssueDate || prepared.ReceivedDate != req.ReceivedDate || prepared.DueDate != req.DueDate {
		t.Fatal("calendar validation rewrote stored dates or request fingerprint input")
	}
	for _, field := range []string{"received", "due"} {
		changed := req
		if field == "received" {
			changed.ReceivedDate = day - 1
		} else {
			changed.DueDate = day - 1
		}
		if _, err := prepareAcceptance(&changed); err == nil {
			t.Fatalf("%s on earlier Beijing day accepted", field)
		}
	}
	if beforeShanghaiDate(day, day+23*3600) || !beforeShanghaiDate(day-1, day) {
		t.Fatal("Beijing midnight comparison does not match calendar boundary")
	}
}

func TestIdempotentActionsRequireIdenticalPayload(t *testing.T) {
	rows := []model.DocumentHistory{{IdempotencyKey: "request-1", RequestHash: "hash-1"}}
	if replayed, err := replay(rows, "request-1", "hash-1"); err != nil || !replayed {
		t.Fatal("identical retry not recognized")
	}
	if _, err := replay(rows, "request-1", "hash-2"); err == nil {
		t.Fatal("changed payload reused key")
	}
	if err := checkVersion(2, 1); err == nil {
		t.Fatal("stale version accepted")
	}
}

func TestShanghaiCalendarDueBoundaryAndQueryValidation(t *testing.T) {
	instant := time.Date(2026, 10, 17, 16, 30, 0, 0, time.UTC)
	want := time.Date(2026, 10, 17, 16, 0, 0, 0, time.UTC).Unix()
	if got := shanghaiToday(instant); got != want {
		t.Fatalf("Shanghai day=%d want=%d", got, want)
	}
	if _, err := documentFilter(&types.CustomerDocumentPageRequest{DueState: "unknown"}, true); err == nil {
		t.Fatal("invalid due filter accepted")
	}
	if _, err := documentFilter(&types.CustomerDocumentPageRequest{TaxRate: "NaN"}, false); err == nil {
		t.Fatal("invalid rate filter accepted")
	}
	if _, err := documentFilter(&types.CustomerDocumentPageRequest{RedemptionStart: 200, RedemptionEnd: 100}, true); err == nil {
		t.Fatal("reversed redemption interval accepted")
	}
}

func TestManualPaymentBindingGuardsCustomerAmountStateAndOwnership(t *testing.T) {
	row := model.CustomerAcceptance{CustomerDocumentBase: model.CustomerDocumentBase{CustomerId: primitive.NewObjectID().Hex()}, AmountMinor: 10000, Amount: 100}
	valid := model.CustomerTransaction{CustomerId: row.CustomerId, TransactionType: code.TransactionTypePayment, Direction: code.TransactionDirectionDecrease, Status: code.TransactionStatusConfirmed, SourceType: code.SourceTypeManualAdjustment, Amount: 100}
	if err := validatePayment(valid, row); err != nil {
		t.Fatal(err)
	}
	mutations := []func(*model.CustomerTransaction){
		func(p *model.CustomerTransaction) { p.CustomerId = primitive.NewObjectID().Hex() },
		func(p *model.CustomerTransaction) { p.Amount = 100.001 },
		func(p *model.CustomerTransaction) { p.Status = code.TransactionStatusVoided },
		func(p *model.CustomerTransaction) { p.Direction = code.TransactionDirectionIncrease },
		func(p *model.CustomerTransaction) { p.AcceptanceId = primitive.NewObjectID().Hex() },
		func(p *model.CustomerTransaction) { p.SourceType = acceptanceSource },
	}
	for i, change := range mutations {
		p := valid
		change(&p)
		if err := validatePayment(p, row); err == nil {
			t.Errorf("invalid payment variant %d accepted", i)
		}
	}
	legacy := valid
	legacy.TransactionType = ""
	legacy.Type = "回款"
	legacy.Status = ""
	legacy.Direction = ""
	legacy.SourceType = ""
	if err := validatePayment(legacy, row); err != nil {
		t.Fatalf("effective legacy manual payment rejected: %v", err)
	}
}
