package documents

import (
	"api/internal/svc"
	"api/internal/types"
	"api/model"
	"context"
	"errors"
	"math"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func invoiceView(row model.CustomerInvoice) types.CustomerInvoice {
	return types.CustomerInvoice{Id: row.Id.Hex(), CustomerId: row.CustomerId, CustomerName: row.CustomerName, InvoiceNumber: row.InvoiceNumber, InvoiceDate: row.InvoiceDate, TaxRate: row.TaxRate, AmountExcludingTax: row.AmountExcludingTax, TaxAmount: row.TaxAmount, TotalAmount: row.TotalAmount, Annex: annexList(row.Annex), Remark: row.Remark, Status: row.Status, Version: row.Version, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt, CreatorName: row.CreatorName, History: historyView(row.History)}
}
func invoiceResponse(ctx context.Context, row model.CustomerInvoice, err error) *types.CustomerInvoiceResponse {
	status, message := responseStatus(ctx, err)
	return &types.CustomerInvoiceResponse{Code: status, Msg: message, Data: invoiceView(row)}
}

func prepareInvoice(req *types.CustomerInvoiceSaveRequest) (model.CustomerInvoice, error) {
	var row model.CustomerInvoice
	id, err := objectID(req.CustomerId)
	if err != nil {
		return row, err
	}
	row.CustomerId = id.Hex()
	row.InvoiceNumber, row.NumberKey, err = number(req.InvoiceNumber)
	if err != nil {
		return row, err
	}
	if !validDate(req.InvoiceDate, false) {
		return row, invalid("开票日期不合法")
	}
	row.InvoiceDate = req.InvoiceDate
	if math.IsNaN(req.TaxRate) || math.IsInf(req.TaxRate, 0) || req.TaxRate < 0 || req.TaxRate > 100 {
		return row, invalid("税率须在0到100之间，例如13表示13%")
	}
	row.TaxRate = req.TaxRate
	row.AmountExcludingTaxMinor, err = moneyMinor(req.AmountExcludingTax, false)
	if err != nil {
		return row, err
	}
	row.TaxAmountMinor, err = moneyMinor(req.TaxAmount, true)
	if err != nil {
		return row, err
	}
	row.TotalAmountMinor, err = moneyMinor(req.TotalAmount, false)
	if err != nil {
		return row, err
	}
	// The invoice face amount is authoritative; mixed-line rounding may differ
	// from a single multiplication of the displayed rate and net amount.
	if row.AmountExcludingTaxMinor+row.TaxAmountMinor != row.TotalAmountMinor {
		return row, invalid("未税金额加税额必须等于价税合计")
	}
	row.AmountExcludingTax, row.TaxAmount, row.TotalAmount = major(row.AmountExcludingTaxMinor), major(row.TaxAmountMinor), major(row.TotalAmountMinor)
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

func SaveInvoice(ctx context.Context, s *svc.ServiceContext, req *types.CustomerInvoiceSaveRequest, create bool) *types.CustomerInvoiceResponse {
	prepared, err := prepareInvoice(req)
	if err != nil {
		return invoiceResponse(ctx, prepared, err)
	}
	var result model.CustomerInvoice
	if create {
		if strings.TrimSpace(req.Id) != "" {
			return invoiceResponse(ctx, result, invalid("新增记录不能指定ID"))
		}
		requestKey, e := key(req.IdempotencyKey)
		if e != nil {
			return invoiceResponse(ctx, result, e)
		}
		requestHash := hash(prepared)
		err = transaction(ctx, s, func(dbCtx mongo.SessionContext) error {
			var existing model.CustomerInvoice
			e := s.CustomerInvoiceModel.FindOne(dbCtx, bson.M{"idempotency_key": requestKey}).Decode(&existing)
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
			if e = bindAttachments(dbCtx, s, annexList(prepared.Annex), "invoice", nil); e != nil {
				return e
			}
			result = prepared
			result.CustomerDocumentBase = base(ctx, customer, prepared.NumberKey, "active", prepared.Annex, prepared.Remark, requestKey, requestHash)
			result.AttachmentHistory = mergeReferences(nil, result.Annex)
			_, e = s.CustomerInvoiceModel.InsertOne(dbCtx, result)
			return e
		})
		if mongo.IsDuplicateKeyError(err) {
			var existing model.CustomerInvoice
			if e := s.CustomerInvoiceModel.FindOne(ctx, bson.M{"idempotency_key": requestKey}).Decode(&existing); e == nil {
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
			return invoiceResponse(ctx, result, e)
		}
		requestKey, requestHash := updateReceipt(id, req.Version, prepared)
		err = transaction(ctx, s, func(dbCtx mongo.SessionContext) error {
			var current model.CustomerInvoice
			if e := s.CustomerInvoiceModel.FindOne(dbCtx, bson.M{"_id": id}).Decode(&current); e != nil {
				return e
			}
			if repeated, e := replay(current.History, requestKey, requestHash); e != nil {
				return e
			} else if repeated {
				result = current
				return nil
			}
			if current.Status != "active" {
				return conflict("已作废发票不能修改")
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
			if e = bindAttachments(dbCtx, s, annexList(prepared.Annex), "invoice", mergeReferences(current.AttachmentHistory, current.Annex)); e != nil {
				return e
			}
			event := history(ctx, "update", "修改开票登记信息", requestKey, requestHash)
			event.Before = beforeSnapshot(current)
			update := bson.M{"$set": bson.M{"customer_id": customer.Id.Hex(), "customer_name": customer.Name, "number_key": prepared.NumberKey, "invoice_number": prepared.InvoiceNumber, "invoice_date": prepared.InvoiceDate, "tax_rate": prepared.TaxRate, "amount_excluding_tax": prepared.AmountExcludingTax, "tax_amount": prepared.TaxAmount, "total_amount": prepared.TotalAmount, "amount_excluding_tax_minor": prepared.AmountExcludingTaxMinor, "tax_amount_minor": prepared.TaxAmountMinor, "total_amount_minor": prepared.TotalAmountMinor, "annex": prepared.Annex, "remark": prepared.Remark, "updated_at": event.Time}, "$inc": bson.M{"version": 1}, "$push": bson.M{"history": event}, "$addToSet": bson.M{"attachment_history": bson.M{"$each": annexList(prepared.Annex)}}}
			if e = modified(s.CustomerInvoiceModel.UpdateOne(dbCtx, stateFilter(id, req.Version, "active"), update)); e != nil {
				return e
			}
			return s.CustomerInvoiceModel.FindOne(dbCtx, bson.M{"_id": id}).Decode(&result)
		})
	}
	return invoiceResponse(ctx, result, err)
}

func DetailInvoice(ctx context.Context, s *svc.ServiceContext, req *types.CustomerDocumentDetailRequest) *types.CustomerInvoiceResponse {
	var row model.CustomerInvoice
	id, err := objectID(req.Id)
	if err == nil {
		err = s.CustomerInvoiceModel.FindOne(ctx, bson.M{"_id": id}).Decode(&row)
	}
	return invoiceResponse(ctx, row, err)
}

func VoidInvoice(ctx context.Context, s *svc.ServiceContext, req *types.CustomerDocumentVoidRequest) *types.CustomerInvoiceResponse {
	var result model.CustomerInvoice
	id, requestKey, reason, err := voidInput(req)
	if err != nil {
		return invoiceResponse(ctx, result, err)
	}
	requestHash := hash([]any{"void", id.Hex(), req.Version, reason})
	err = transaction(ctx, s, func(dbCtx mongo.SessionContext) error {
		var current model.CustomerInvoice
		if e := s.CustomerInvoiceModel.FindOne(dbCtx, bson.M{"_id": id}).Decode(&current); e != nil {
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
		if current.Status != "active" {
			return conflict("该发票已作废，请刷新核对")
		}
		if e = checkVersion(current.Version, req.Version); e != nil {
			return e
		}
		if _, e = touchCustomer(dbCtx, s, current.CustomerId); e != nil {
			return e
		}
		event := history(ctx, "void", reason, requestKey, requestHash)
		if e = modified(s.CustomerInvoiceModel.UpdateOne(dbCtx, stateFilter(id, req.Version, "active"), bson.M{"$set": bson.M{"status": "voided", "updated_at": event.Time}, "$inc": bson.M{"version": 1}, "$push": bson.M{"history": event}})); e != nil {
			return e
		}
		return s.CustomerInvoiceModel.FindOne(dbCtx, bson.M{"_id": id}).Decode(&result)
	})
	return invoiceResponse(ctx, result, err)
}
