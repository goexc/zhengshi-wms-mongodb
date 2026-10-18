package finance

import (
	"api/model"
	code "api/pkg/code"
	"math"
	"testing"

	"github.com/shopspring/decimal"
)

func TestLedgerBalanceRetainsPaymentsAndSplitsOverpayment(t *testing.T) {
	records := []model.CustomerTransaction{
		{CustomerId: "c", TransactionType: code.TransactionTypeOpeningAR, Amount: 30},
		{CustomerId: "c", Type: "回款", Amount: 100},
		{CustomerId: "c", TransactionType: code.TransactionTypeOutboundAR, Status: code.TransactionStatusVoided, Amount: 900},
		{CustomerId: "other", TransactionType: code.TransactionTypeOutboundAR, Amount: 200},
	}
	balances, err := CustomerBalanceSnapshots(records)
	if err != nil {
		t.Fatal(err)
	}
	ar, credit := SplitCustomerNetBalance(balances["c"])
	if !ar.IsZero() || !credit.Equal(decimal.NewFromInt(70)) {
		t.Fatalf("AR=%s credit=%s", ar, credit)
	}
	if !balances["other"].Equal(decimal.NewFromInt(200)) {
		t.Fatal("mixed customer ledgers")
	}
}

func TestLedgerRejectsInvalidActiveAmountsDirectionsAndTypes(t *testing.T) {
	cases := []model.CustomerTransaction{
		{TransactionType: code.TransactionTypeOutboundAR, Amount: math.NaN()},
		{TransactionType: code.TransactionTypeOutboundAR, Amount: math.Inf(1)},
		{TransactionType: code.TransactionTypeOutboundAR, Amount: -1},
		{TransactionType: code.TransactionTypePayment, Direction: code.TransactionDirectionIncrease, Amount: 1},
		{TransactionType: "unrecognized", Type: "应收账款", Amount: 1},
		{TransactionType: code.TransactionTypeARAdjustment, Amount: 1},
	}
	for i, record := range cases {
		if _, err := TransactionNet(record); err == nil {
			t.Errorf("case %d accepted invalid record", i)
		}
	}
}
