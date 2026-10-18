package finance

import (
	"api/internal/svc"
	"api/model"
	financeCode "api/pkg/code"
	"context"
	"math"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TransactionType preserves the legacy display-only type fallback without
// allowing a known source ID or a nonempty canonical type to be overwritten.
func TransactionType(record model.CustomerTransaction) string {
	if value := strings.TrimSpace(record.TransactionType); value != "" {
		return value
	}
	value, _ := financeCode.TransactionTypeCode(record.Type)
	return value
}

// IndependentManualARAdjustment is shared by deletion, repricing and recount.
// A manual source can mention an order code without becoming that order's AR;
// claiming the order's actual ID as its independent source is contradictory.
func IndependentManualARAdjustment(record model.CustomerTransaction, orderID string) (bool, error) {
	if TransactionType(record) != financeCode.TransactionTypeARAdjustment || strings.TrimSpace(record.SourceType) != financeCode.SourceTypeManualAdjustment {
		return false, nil
	}
	if orderID != "" && strings.TrimSpace(record.SourceId) == orderID {
		return false, &BusinessError{Message: "独立调整的来源ID与出库单冲突，请先核对"}
	}
	return true, nil
}

// TransactionNet is the balance contribution of one ledger row. All consumers
// use the same status, legacy-type, direction and finite-amount rules.
func TransactionNet(record model.CustomerTransaction) (decimal.Decimal, error) {
	if !financeCode.TransactionStatusCountsInBalance(record.Status) {
		return decimal.Zero, nil
	}
	if math.IsNaN(record.Amount) || math.IsInf(record.Amount, 0) {
		return decimal.Zero, &BusinessError{Message: "存在金额异常的交易流水"}
	}
	delta, ok := financeCode.TransactionBalanceDelta(TransactionType(record), record.Direction, record.Amount)
	if !ok {
		return decimal.Zero, &BusinessError{Message: "存在未知类型或方向异常的交易流水"}
	}
	return decimal.NewFromFloat(delta), nil
}

func CustomerBalanceSnapshots(records []model.CustomerTransaction) (map[string]decimal.Decimal, error) {
	balances := make(map[string]decimal.Decimal)
	for _, record := range records {
		delta, err := TransactionNet(record)
		if err != nil {
			return nil, err
		}
		balances[record.CustomerId] = balances[record.CustomerId].Add(delta)
	}
	return balances, nil
}

func SplitCustomerNetBalance(net decimal.Decimal) (receivable, credit decimal.Decimal) {
	tolerance := decimal.NewFromFloat(amountTolerance)
	switch {
	case net.GreaterThan(tolerance):
		return net, decimal.Zero
	case net.LessThan(tolerance.Neg()):
		return decimal.Zero, net.Abs()
	default:
		return decimal.Zero, decimal.Zero
	}
}

// RebuildCustomerBalance reads the caller's transaction snapshot, including its
// own inserts/deletes, and writes both caches from the complete effective ledger.
// The revision is a concurrency gate even if tolerance rounds both caches to 0.
func RebuildCustomerBalance(ctx context.Context, svcCtx *svc.ServiceContext, customerID primitive.ObjectID) error {
	cur, err := svcCtx.CustomerTransactionModel.Find(ctx, bson.M{"customer_id": customerID.Hex()})
	if err != nil {
		return err
	}
	defer cur.Close(ctx)
	var records []model.CustomerTransaction
	if err = cur.All(ctx, &records); err != nil {
		return err
	}
	balances, err := CustomerBalanceSnapshots(records)
	if err != nil {
		return err
	}
	receivable, credit := SplitCustomerNetBalance(balances[customerID.Hex()])
	result, err := svcCtx.CustomerModel.UpdateOne(ctx,
		bson.M{"_id": customerID, "is_deleted": bson.M{"$ne": true}, "status": bson.M{"$ne": "删除"}},
		bson.M{
			"$set": bson.M{"receivable_balance": receivable.InexactFloat64(), "credit_balance": credit.InexactFloat64(), "updated_at": time.Now().Unix()},
			"$inc": bson.M{"balance_revision": int64(1)},
		},
	)
	if err != nil {
		return err
	}
	if result.MatchedCount != 1 {
		return &BusinessError{Message: "客户不存在或已删除"}
	}
	return nil
}
