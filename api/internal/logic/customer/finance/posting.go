package finance

import (
	"api/internal/svc"
	"api/model"
	financeCode "api/pkg/code"
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

const amountTolerance = 0.000001

// BusinessError represents a finance rule violation that should be returned to
// the client as a business message rather than logged as an infrastructure fault.
type BusinessError struct {
	Message string
}

func (e *BusinessError) Error() string {
	return e.Message
}

func IsBusinessError(err error) (string, bool) {
	var businessErr *BusinessError
	if errors.As(err, &businessErr) {
		return businessErr.Message, true
	}
	return "", false
}

type PostTransactionRequest struct {
	TransactionType       string
	Direction             string
	Status                string
	Code                  string
	OrderCode             string
	SourceType            string
	SourceId              string
	SourceCode            string
	IdempotencyKey        string
	OriginalTransactionId string
	CustomerId            string
	CustomerName          string
	Amount                float64
	Annex                 string
	Remark                string
	Time                  int64
	Creator               string
	CreatorName           string
	CreatedAt             int64
}

type PostTransactionResult struct {
	Transaction  model.CustomerTransaction
	Inserted     bool
	BalanceDelta float64
}

// ResolveDirection keeps the accounting direction on the server side. Client
// requests describe a transaction type; they do not get to decide whether it
// increases or decreases receivables.
func ResolveDirection(transactionType, requestedDirection string) (string, error) {
	transactionType = strings.TrimSpace(transactionType)
	requestedDirection = strings.TrimSpace(requestedDirection)

	expected := ""
	switch transactionType {
	case financeCode.TransactionTypeOpeningAR, financeCode.TransactionTypeOutboundAR:
		expected = financeCode.TransactionDirectionIncrease
	case financeCode.TransactionTypePayment, financeCode.TransactionTypeReturnCredit:
		expected = financeCode.TransactionDirectionDecrease
	case financeCode.TransactionTypeARAdjustment, financeCode.TransactionTypeManualAdjustment:
		if requestedDirection == "" {
			return "", &BusinessError{Message: "调整类交易必须指定调整方向"}
		}
		if requestedDirection != financeCode.TransactionDirectionIncrease &&
			requestedDirection != financeCode.TransactionDirectionDecrease {
			return "", &BusinessError{Message: "交易方向错误"}
		}
		return requestedDirection, nil
	default:
		return "", &BusinessError{Message: "交易类型错误"}
	}

	if requestedDirection != "" && requestedDirection != expected {
		return "", &BusinessError{Message: "交易类型与余额方向不匹配"}
	}
	return expected, nil
}

// ApplyBalanceDelta projects the signed ledger delta into two cache fields.
// Positive net balance is stored as receivable_balance; negative net balance is
// stored as credit_balance so valid price decreases or overpayments are not
// rejected simply because the current receivable is already zero.
func ApplyBalanceDelta(ctx context.Context, svcCtx *svc.ServiceContext, customerId primitive.ObjectID, delta float64) error {
	if math.Abs(delta) <= amountTolerance {
		return nil
	}

	filter := bson.M{
		"_id":        customerId,
		"is_deleted": bson.M{"$ne": true},
		"status":     bson.M{"$ne": "删除"},
	}
	netBefore := bson.D{{"$subtract", bson.A{
		bson.D{{"$ifNull", bson.A{"$receivable_balance", 0}}},
		bson.D{{"$ifNull", bson.A{"$credit_balance", 0}}},
	}}}
	netAfter := bson.D{{"$add", bson.A{netBefore, delta}}}
	update := mongo.Pipeline{
		bson.D{{"$set", bson.D{
			{"receivable_balance", bson.D{{"$cond", bson.A{
				bson.D{{"$gt", bson.A{netAfter, amountTolerance}}},
				netAfter,
				0,
			}}}},
			{"credit_balance", bson.D{{"$cond", bson.A{
				bson.D{{"$lt", bson.A{netAfter, -amountTolerance}}},
				bson.D{{"$abs", netAfter}},
				0,
			}}}},
			{"updated_at", time.Now().Unix()},
		}}},
	}

	res, err := svcCtx.CustomerModel.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if res.MatchedCount != 1 {
		return &BusinessError{Message: "客户不存在或已删除"}
	}

	return nil
}

// PostTransaction inserts one customer finance transaction and applies the
// balance cache change in the same caller-provided transaction context.
func PostTransaction(ctx context.Context, svcCtx *svc.ServiceContext, req PostTransactionRequest) (*PostTransactionResult, error) {
	if req.Amount <= 0 {
		return nil, &BusinessError{Message: "交易金额必须大于0"}
	}
	if strings.TrimSpace(req.IdempotencyKey) == "" {
		return nil, &BusinessError{Message: "缺少幂等键"}
	}

	direction, err := ResolveDirection(req.TransactionType, req.Direction)
	if err != nil {
		return nil, err
	}

	status := strings.TrimSpace(req.Status)
	if status == "" {
		status = financeCode.TransactionStatusConfirmed
	}

	customerObjectId, err := primitive.ObjectIDFromHex(strings.TrimSpace(req.CustomerId))
	if err != nil {
		return nil, &BusinessError{Message: "客户参数错误"}
	}

	now := req.CreatedAt
	if now == 0 {
		now = time.Now().Unix()
	}
	transactionTime := req.Time
	if transactionTime == 0 {
		transactionTime = now
	}
	transactionId := primitive.NewObjectID()
	code := strings.TrimSpace(req.Code)
	if code == "" {
		code = fmt.Sprintf("CT-%d", time.Now().UnixMilli())
	}

	record := model.CustomerTransaction{
		Id:                    transactionId,
		Type:                  financeCode.TransactionTypeLabel(req.TransactionType, ""),
		TransactionType:       strings.TrimSpace(req.TransactionType),
		Direction:             direction,
		Status:                status,
		Code:                  code,
		OrderCode:             strings.TrimSpace(req.OrderCode),
		SourceType:            strings.TrimSpace(req.SourceType),
		SourceId:              strings.TrimSpace(req.SourceId),
		SourceCode:            strings.TrimSpace(req.SourceCode),
		IdempotencyKey:        strings.TrimSpace(req.IdempotencyKey),
		OriginalTransactionId: strings.TrimSpace(req.OriginalTransactionId),
		CustomerId:            strings.TrimSpace(req.CustomerId),
		CustomerName:          strings.TrimSpace(req.CustomerName),
		Amount:                req.Amount,
		Annex:                 strings.TrimSpace(req.Annex),
		Remark:                strings.TrimSpace(req.Remark),
		Time:                  transactionTime,
		Creator:               strings.TrimSpace(req.Creator),
		CreatorName:           strings.TrimSpace(req.CreatorName),
		CreatedAt:             now,
		UpdatedAt:             0,
	}

	if _, err = svcCtx.CustomerTransactionModel.InsertOne(ctx, &record); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			existing, findErr := findByIdempotencyKey(ctx, svcCtx, record.IdempotencyKey)
			if findErr != nil {
				return nil, findErr
			}
			if !sameTransactionPayload(existing, record) {
				return nil, &BusinessError{Message: "幂等键已被其他交易使用"}
			}
			return &PostTransactionResult{Transaction: existing, Inserted: false, BalanceDelta: 0}, nil
		}
		return nil, err
	}

	delta, ok := financeCode.TransactionBalanceDelta(record.TransactionType, record.Direction, record.Amount)
	if !ok {
		return nil, &BusinessError{Message: "交易方向错误"}
	}
	if !financeCode.TransactionStatusCountsInBalance(record.Status) {
		delta = 0
	}

	if err = ApplyBalanceDelta(ctx, svcCtx, customerObjectId, delta); err != nil {
		return nil, err
	}

	return &PostTransactionResult{Transaction: record, Inserted: true, BalanceDelta: delta}, nil
}

func findByIdempotencyKey(ctx context.Context, svcCtx *svc.ServiceContext, key string) (model.CustomerTransaction, error) {
	var existing model.CustomerTransaction
	err := svcCtx.CustomerTransactionModel.FindOne(ctx, bson.M{"idempotency_key": key}).Decode(&existing)
	return existing, err
}

func sameTransactionPayload(left model.CustomerTransaction, right model.CustomerTransaction) bool {
	return left.CustomerId == right.CustomerId &&
		left.TransactionType == right.TransactionType &&
		left.Direction == right.Direction &&
		left.Status == right.Status &&
		left.SourceType == right.SourceType &&
		left.SourceId == right.SourceId &&
		left.SourceCode == right.SourceCode &&
		math.Abs(left.Amount-right.Amount) <= amountTolerance
}
