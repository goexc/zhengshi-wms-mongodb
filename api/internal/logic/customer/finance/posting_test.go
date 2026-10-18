package finance

import (
	"context"
	"math"
	"testing"
)

func TestPostTransactionRejectsNonfiniteAmountBeforeDatabaseAccess(t *testing.T) {
	for _, amount := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		if _, err := PostTransaction(context.Background(), nil, PostTransactionRequest{Amount: amount}); err == nil {
			t.Fatalf("accepted nonfinite amount %v", amount)
		}
	}
}
