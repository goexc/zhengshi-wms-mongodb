package documents

import (
	finance "api/internal/logic/customer/finance"
	"api/internal/svc"
	"api/internal/types"
	"api/model"
	code "api/pkg/code"
	"context"

	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// ReverseAcceptance deliberately distinguishes a generated payment from an
// independently entered payment. Only the former is reversed in the ledger.
func ReverseAcceptance(ctx context.Context, s *svc.ServiceContext, req *types.CustomerDocumentVoidRequest) *types.CustomerAcceptanceResponse {
	var result model.CustomerAcceptance
	id, requestKey, reason, err := voidInput(req)
	if err != nil {
		return acceptanceResponse(ctx, result, err)
	}
	requestHash := hash([]any{"reverse", id.Hex(), req.Version, reason})
	err = transaction(ctx, s, func(dbCtx mongo.SessionContext) error {
		var current model.CustomerAcceptance
		if e := s.CustomerAcceptanceModel.FindOne(dbCtx, bson.M{"_id": id}).Decode(&current); e != nil {
			return e
		}
		repeated, e := replay(current.History, requestKey, requestHash)
		if e != nil {
			return e
		}
		if repeated {
			result = current
			return nil
		}
		if current.Status != "redeemed" {
			return conflict("仅已兑付承兑可以撤销兑付")
		}
		if e = checkVersion(current.Version, req.Version); e != nil {
			return e
		}
		customer, e := touchCustomer(dbCtx, s, current.CustomerId)
		if e != nil {
			return e
		}
		paymentID, e := objectID(current.TransactionId)
		if e != nil {
			return invalid("承兑关联回款ID异常，请核对")
		}
		var payment model.CustomerTransaction
		if e = s.CustomerTransactionModel.FindOne(dbCtx, bson.M{"_id": paymentID}).Decode(&payment); e != nil {
			return e
		}
		if payment.CustomerId != current.CustomerId || payment.AcceptanceId != current.Id.Hex() || finance.TransactionType(payment) != code.TransactionTypePayment || !code.TransactionStatusCountsInBalance(payment.Status) {
			return conflict("关联回款状态或归属已变化，不能撤销兑付")
		}
		net, e := finance.TransactionNet(payment)
		if e != nil || !net.Equal(decimal.NewFromInt(current.AmountMinor).Shift(-2).Neg()) {
			return invalid("关联回款金额或方向异常，不能撤销兑付")
		}
		generated := payment.SourceType == acceptanceSource
		if generated && payment.SourceId != id.Hex() {
			return invalid("自动回款来源与承兑不一致")
		}
		if !generated && payment.SourceType != "" && payment.SourceType != code.SourceTypeManualAdjustment && payment.SourceType != code.SourceTypePayment {
			return invalid("关联回款存在其他业务来源，不能解除")
		}
		event := history(ctx, "reverse", reason, requestKey, requestHash)
		event.Before = beforeSnapshot(current)
		paymentUpdate := bson.M{"$unset": bson.M{"acceptance_id": ""}}
		if generated {
			paymentUpdate["$set"] = bson.M{"status": code.TransactionStatusReversed, "updated_at": event.Time, "editor": event.Operator, "editor_name": event.OperatorName}
		}
		if e = modified(s.CustomerTransactionModel.UpdateOne(dbCtx, bson.M{"_id": paymentID, "acceptance_id": id.Hex(), "status": bson.M{"$in": bson.A{code.TransactionStatusConfirmed, "", nil}}}, paymentUpdate)); e != nil {
			return e
		}
		if generated {
			if e = finance.RebuildCustomerBalance(dbCtx, s, customer.Id); e != nil {
				return e
			}
		}
		update := bson.M{"$set": bson.M{"status": "pending", "redemption_date": int64(0), "redemption_annex": "", "updated_at": event.Time}, "$unset": bson.M{"transaction_id": ""}, "$inc": bson.M{"version": 1}, "$push": bson.M{"history": event}}
		if e = modified(s.CustomerAcceptanceModel.UpdateOne(dbCtx, stateFilter(id, req.Version, "redeemed"), update)); e != nil {
			return e
		}
		return s.CustomerAcceptanceModel.FindOne(dbCtx, bson.M{"_id": id}).Decode(&result)
	})
	return acceptanceResponse(ctx, result, err)
}
