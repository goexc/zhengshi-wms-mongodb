package documents

import (
	finance "api/internal/logic/customer/finance"
	"api/internal/svc"
	"api/internal/types"
	"api/model"
	code "api/pkg/code"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func acceptanceView(row model.CustomerAcceptance) types.CustomerAcceptance {
	return types.CustomerAcceptance{Id: row.Id.Hex(), CustomerId: row.CustomerId, CustomerName: row.CustomerName, BillNumber: row.BillNumber, Amount: row.Amount, ReceivedDate: row.ReceivedDate, IssueDate: row.IssueDate, DueDate: row.DueDate, RedemptionDate: row.RedemptionDate, Status: row.Status, Annex: annexList(row.Annex), RedemptionAnnex: annexList(row.RedemptionAnnex), Remark: row.Remark, Version: row.Version, TransactionId: row.TransactionId, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt, CreatorName: row.CreatorName, History: historyView(row.History)}
}
func acceptanceResponse(ctx context.Context, row model.CustomerAcceptance, err error) *types.CustomerAcceptanceResponse {
	status, message := responseStatus(ctx, err)
	return &types.CustomerAcceptanceResponse{Code: status, Msg: message, Data: acceptanceView(row)}
}

func prepareAcceptance(req *types.CustomerAcceptanceSaveRequest) (model.CustomerAcceptance, error) {
	var row model.CustomerAcceptance
	id, err := objectID(req.CustomerId)
	if err != nil {
		return row, err
	}
	row.CustomerId = id.Hex()
	row.BillNumber, row.NumberKey, err = number(req.BillNumber)
	if err != nil {
		return row, err
	}
	row.AmountMinor, err = moneyMinor(req.Amount, false)
	if err != nil {
		return row, err
	}
	row.Amount = major(row.AmountMinor)
	if !validDate(req.ReceivedDate, false) || !validDate(req.IssueDate, false) || !validDate(req.DueDate, false) || beforeShanghaiDate(req.DueDate, req.IssueDate) || beforeShanghaiDate(req.ReceivedDate, req.IssueDate) {
		return row, invalid("出票、收票和到期日期必填；收票和到期日期不得早于出票日期")
	}
	row.ReceivedDate, row.IssueDate, row.DueDate = req.ReceivedDate, req.IssueDate, req.DueDate
	row.Annex, err = annex(req.Annex)
	if err != nil {
		return row, err
	}
	row.Remark, err = remark(req.Remark)
	if err != nil {
		return row, err
	}
	return row, nil
}

func SaveAcceptance(ctx context.Context, s *svc.ServiceContext, req *types.CustomerAcceptanceSaveRequest, create bool) *types.CustomerAcceptanceResponse {
	prepared, err := prepareAcceptance(req)
	if err != nil {
		return acceptanceResponse(ctx, prepared, err)
	}
	var result model.CustomerAcceptance
	if create {
		if strings.TrimSpace(req.Id) != "" {
			return acceptanceResponse(ctx, result, invalid("新增记录不能指定ID"))
		}
		requestKey, e := key(req.IdempotencyKey)
		if e != nil {
			return acceptanceResponse(ctx, result, e)
		}
		requestHash := hash(prepared)
		err = transaction(ctx, s, func(dbCtx mongo.SessionContext) error {
			var existing model.CustomerAcceptance
			e := s.CustomerAcceptanceModel.FindOne(dbCtx, bson.M{"idempotency_key": requestKey}).Decode(&existing)
			if e == nil {
				if existing.RequestHash != requestHash {
					return conflict("幂等键已用于不同登记内容")
				}
				result = existing
				return nil
			}
			if !errors.Is(e, mongo.ErrNoDocuments) {
				return e
			}
			customer, e := touchCustomer(dbCtx, s, prepared.CustomerId)
			if e != nil {
				return e
			}
			if e = bindAttachments(dbCtx, s, annexList(prepared.Annex), "acceptance", nil); e != nil {
				return e
			}
			result = prepared
			result.CustomerDocumentBase = base(ctx, customer, prepared.NumberKey, "pending", prepared.Annex, prepared.Remark, requestKey, requestHash)
			result.AttachmentHistory = mergeReferences(nil, result.Annex)
			_, e = s.CustomerAcceptanceModel.InsertOne(dbCtx, result)
			return e
		})
		if mongo.IsDuplicateKeyError(err) {
			var existing model.CustomerAcceptance
			if e := s.CustomerAcceptanceModel.FindOne(ctx, bson.M{"idempotency_key": requestKey}).Decode(&existing); e == nil {
				if existing.RequestHash == requestHash {
					result = existing
					err = nil
				} else {
					err = conflict("幂等键已用于不同登记内容")
				}
			}
		}
	} else {
		id, e := objectID(req.Id)
		if e != nil {
			return acceptanceResponse(ctx, result, e)
		}
		requestKey, requestHash := updateReceipt(id, req.Version, prepared)
		err = transaction(ctx, s, func(dbCtx mongo.SessionContext) error {
			var current model.CustomerAcceptance
			if e := s.CustomerAcceptanceModel.FindOne(dbCtx, bson.M{"_id": id}).Decode(&current); e != nil {
				return e
			}
			if repeated, e := replay(current.History, requestKey, requestHash); e != nil {
				return e
			} else if repeated {
				result = current
				return nil
			}
			if current.Status != "pending" {
				return conflict("仅待兑付承兑可以修改；已兑付或已作废记录不可修改")
			}
			if e := checkVersion(current.Version, req.Version); e != nil {
				return e
			}
			customer, e := touchCustomer(dbCtx, s, prepared.CustomerId)
			if e != nil {
				return e
			}
			if current.CustomerId != prepared.CustomerId {
				if _, e = touchCustomer(dbCtx, s, current.CustomerId); e != nil {
					return e
				}
			}
			if e = bindAttachments(dbCtx, s, annexList(prepared.Annex), "acceptance", mergeReferences(current.AttachmentHistory, current.Annex, current.RedemptionAnnex)); e != nil {
				return e
			}
			event := history(ctx, "update", "修改承兑登记信息", requestKey, requestHash)
			event.Before = beforeSnapshot(current)
			update := bson.M{"$set": bson.M{"customer_id": customer.Id.Hex(), "customer_name": customer.Name, "number_key": prepared.NumberKey, "bill_number": prepared.BillNumber, "amount": prepared.Amount, "amount_minor": prepared.AmountMinor, "received_date": prepared.ReceivedDate, "issue_date": prepared.IssueDate, "due_date": prepared.DueDate, "annex": prepared.Annex, "remark": prepared.Remark, "updated_at": event.Time}, "$inc": bson.M{"version": 1}, "$push": bson.M{"history": event}, "$addToSet": bson.M{"attachment_history": bson.M{"$each": annexList(prepared.Annex)}}}
			if e = modified(s.CustomerAcceptanceModel.UpdateOne(dbCtx, stateFilter(id, req.Version, "pending"), update)); e != nil {
				return e
			}
			return s.CustomerAcceptanceModel.FindOne(dbCtx, bson.M{"_id": id}).Decode(&result)
		})
	}
	return acceptanceResponse(ctx, result, err)
}

func DetailAcceptance(ctx context.Context, s *svc.ServiceContext, req *types.CustomerDocumentDetailRequest) *types.CustomerAcceptanceResponse {
	var row model.CustomerAcceptance
	id, err := objectID(req.Id)
	if err == nil {
		err = s.CustomerAcceptanceModel.FindOne(ctx, bson.M{"_id": id}).Decode(&row)
	}
	return acceptanceResponse(ctx, row, err)
}

// validatePayment permits only an effective, independent manual payment of the
// same customer and exact amount. Binding it must never change balance again.
func validatePayment(payment model.CustomerTransaction, row model.CustomerAcceptance) error {
	if payment.CustomerId != row.CustomerId || finance.TransactionType(payment) != code.TransactionTypePayment || !code.TransactionStatusCountsInBalance(payment.Status) {
		return invalid("所选流水不是此客户的有效回款")
	}
	if payment.SourceType != "" && payment.SourceType != code.SourceTypeManualAdjustment && payment.SourceType != code.SourceTypePayment {
		return invalid("所选回款已有其他业务来源，不能绑定承兑")
	}
	if payment.AcceptanceId != "" {
		return conflict("该回款已被其他承兑占用")
	}
	net, err := finance.TransactionNet(payment)
	if err != nil {
		return invalid("所选回款金额或方向异常")
	}
	if !net.Equal(decimal.NewFromInt(row.AmountMinor).Shift(-2).Neg()) {
		return invalid("回款金额必须与票面金额完全一致")
	}
	return nil
}

func RedeemAcceptance(ctx context.Context, s *svc.ServiceContext, req *types.CustomerAcceptanceRedeemRequest) *types.CustomerAcceptanceResponse {
	var result model.CustomerAcceptance
	id, err := objectID(req.Id)
	if err != nil {
		return acceptanceResponse(ctx, result, err)
	}
	requestKey, err := key(req.IdempotencyKey)
	if err != nil {
		return acceptanceResponse(ctx, result, err)
	}
	refs, err := annex(req.RedemptionAnnex)
	if err != nil {
		return acceptanceResponse(ctx, result, err)
	}
	note, err := remark(req.Remark)
	if err != nil {
		return acceptanceResponse(ctx, result, err)
	}
	if !validDate(req.RedemptionDate, false) {
		return acceptanceResponse(ctx, result, invalid("到账日期不合法"))
	}
	requestHash := hash([]any{"redeem", id.Hex(), req.Version, req.RedemptionDate, refs, note, strings.TrimSpace(req.TransactionId)})
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
		if current.Status != "pending" {
			return conflict("该承兑已兑付或已作废，请刷新核对")
		}
		if e = checkVersion(current.Version, req.Version); e != nil {
			return e
		}
		if beforeShanghaiDate(req.RedemptionDate, current.ReceivedDate) {
			return invalid("到账日期不能早于收票日期")
		}
		if _, e = touchCustomer(dbCtx, s, current.CustomerId); e != nil {
			return e
		}
		if e = bindAttachments(dbCtx, s, annexList(refs), "acceptance_redemption", mergeReferences(current.AttachmentHistory, current.Annex, current.RedemptionAnnex)); e != nil {
			return e
		}
		transactionID := strings.TrimSpace(req.TransactionId)
		if transactionID != "" {
			paymentID, e := objectID(transactionID)
			if e != nil {
				return e
			}
			var payment model.CustomerTransaction
			if e = s.CustomerTransactionModel.FindOne(dbCtx, bson.M{"_id": paymentID}).Decode(&payment); e != nil {
				return e
			}
			if e = validatePayment(payment, current); e != nil {
				return e
			}
			// The write also conflicts with concurrent ledger mutation. The unique
			// acceptance.transaction_id index is a second exclusive-binding guard.
			claimFilter := bson.M{"_id": paymentID, "$or": bson.A{bson.M{"acceptance_id": bson.M{"$exists": false}}, bson.M{"acceptance_id": ""}}}
			if e = modified(s.CustomerTransactionModel.UpdateOne(dbCtx, claimFilter, bson.M{"$set": bson.M{"acceptance_id": id.Hex()}})); e != nil {
				return e
			}
			transactionID = paymentID.Hex()
		} else {
			actorID, actorName := actor(ctx)
			posted, e := finance.PostTransaction(dbCtx, s, finance.PostTransactionRequest{TransactionType: code.TransactionTypePayment, Status: code.TransactionStatusConfirmed, SourceType: acceptanceSource, SourceId: id.Hex(), SourceCode: current.BillNumber, IdempotencyKey: fmt.Sprintf("%s:%s:redeem:%d", acceptanceSource, id.Hex(), current.Version), CustomerId: current.CustomerId, CustomerName: current.CustomerName, Amount: current.Amount, Annex: refs, Remark: "承兑兑付到账：" + current.BillNumber + " " + note, Time: req.RedemptionDate, Creator: actorID, CreatorName: actorName})
			if e != nil {
				return e
			}
			transactionID = posted.Transaction.Id.Hex()
			if e = modified(s.CustomerTransactionModel.UpdateOne(dbCtx, bson.M{"_id": posted.Transaction.Id}, bson.M{"$set": bson.M{"acceptance_id": id.Hex()}})); e != nil {
				return e
			}
		}
		event := history(ctx, "redeem", note, requestKey, requestHash)
		event.Before = beforeSnapshot(current)
		update := bson.M{"$set": bson.M{"status": "redeemed", "redemption_date": req.RedemptionDate, "redemption_annex": refs, "transaction_id": transactionID, "updated_at": time.Now().Unix()}, "$inc": bson.M{"version": 1}, "$push": bson.M{"history": event}, "$addToSet": bson.M{"attachment_history": bson.M{"$each": annexList(refs)}}}
		if e = modified(s.CustomerAcceptanceModel.UpdateOne(dbCtx, stateFilter(id, req.Version, "pending"), update)); e != nil {
			return e
		}
		return s.CustomerAcceptanceModel.FindOne(dbCtx, bson.M{"_id": id}).Decode(&result)
	})
	return acceptanceResponse(ctx, result, err)
}

func VoidAcceptance(ctx context.Context, s *svc.ServiceContext, req *types.CustomerDocumentVoidRequest) *types.CustomerAcceptanceResponse {
	var result model.CustomerAcceptance
	id, requestKey, reason, err := voidInput(req)
	if err != nil {
		return acceptanceResponse(ctx, result, err)
	}
	requestHash := hash([]any{"void", id.Hex(), req.Version, reason})
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
		if current.Status != "pending" {
			return conflict("仅待兑付承兑可以作废；已兑付记录不得普通作废")
		}
		if e = checkVersion(current.Version, req.Version); e != nil {
			return e
		}
		if _, e = touchCustomer(dbCtx, s, current.CustomerId); e != nil {
			return e
		}
		event := history(ctx, "void", reason, requestKey, requestHash)
		if e = modified(s.CustomerAcceptanceModel.UpdateOne(dbCtx, stateFilter(id, req.Version, "pending"), bson.M{"$set": bson.M{"status": "voided", "updated_at": event.Time}, "$inc": bson.M{"version": 1}, "$push": bson.M{"history": event}})); e != nil {
			return e
		}
		return s.CustomerAcceptanceModel.FindOne(dbCtx, bson.M{"_id": id}).Decode(&result)
	})
	return acceptanceResponse(ctx, result, err)
}
