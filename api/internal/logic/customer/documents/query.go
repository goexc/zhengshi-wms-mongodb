package documents

import (
	"api/internal/svc"
	"api/internal/types"
	"api/model"
	code "api/pkg/code"
	"context"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func pagination(page, size int64) (int64, int64, error) {
	if page == 0 {
		page = 1
	}
	if size == 0 {
		size = 20
	}
	if page < 1 || page > 1000000 || size < 1 || size > 100 {
		return 0, 0, invalid("页码或每页条数错误（每页1至100条）")
	}
	return (page - 1) * size, size, nil
}
func dateRange(start, end int64) (bson.M, error) {
	result := bson.M{}
	if !validDate(start, true) || !validDate(end, true) || (end > 0 && start > end) {
		return nil, invalid("日期范围不合法")
	}
	if start > 0 {
		result["$gte"] = start
	}
	if end > 0 {
		result["$lte"] = end
	}
	return result, nil
}
func documentFilter(req *types.CustomerDocumentPageRequest, acceptance bool) (bson.M, error) {
	filter := bson.M{}
	if strings.TrimSpace(req.CustomerId) != "" {
		id, err := objectID(req.CustomerId)
		if err != nil {
			return nil, err
		}
		filter["customer_id"] = id.Hex()
	}
	status := strings.TrimSpace(req.Status)
	if status != "" {
		valid := status == "voided" || (acceptance && (status == "pending" || status == "redeemed")) || (!acceptance && status == "active")
		if !valid {
			return nil, invalid("状态筛选不合法")
		}
		filter["status"] = status
	}
	keyword := strings.TrimSpace(req.Keyword)
	if utf8.RuneCountInString(keyword) > 128 {
		return nil, invalid("搜索内容最多128字")
	}
	numberField, dateField := "invoice_number", "invoice_date"
	if acceptance {
		numberField, dateField = "bill_number", "received_date"
	}
	if keyword != "" {
		pattern := primitive.Regex{Pattern: regexp.QuoteMeta(keyword), Options: "i"}
		filter["$or"] = bson.A{bson.M{numberField: pattern}, bson.M{"customer_name": pattern}}
	}
	rangeFilter, err := dateRange(req.DateStart, req.DateEnd)
	if err != nil {
		return nil, err
	}
	if len(rangeFilter) > 0 {
		filter[dateField] = rangeFilter
	}
	if acceptance {
		rangeFilter, err = dateRange(req.DueStart, req.DueEnd)
		if err != nil {
			return nil, err
		}
		if len(rangeFilter) > 0 {
			filter["due_date"] = rangeFilter
		}
		rangeFilter, err = dateRange(req.RedemptionStart, req.RedemptionEnd)
		if err != nil {
			return nil, err
		}
		if len(rangeFilter) > 0 {
			filter["redemption_date"] = rangeFilter
		}
		dueState := strings.TrimSpace(req.DueState)
		if dueState != "" {
			if dueState != "overdue" && dueState != "soon" {
				return nil, invalid("到期筛选不合法")
			}
			today := shanghaiToday(time.Now())
			dueRange := bson.M{"$lt": today}
			if dueState == "soon" {
				dueRange = bson.M{"$gte": today, "$lt": today + 7*86400}
			}
			filter["$and"] = bson.A{bson.M{"status": "pending"}, bson.M{"due_date": dueRange}}
		}
	} else if strings.TrimSpace(req.TaxRate) != "" {
		rate, e := strconv.ParseFloat(strings.TrimSpace(req.TaxRate), 64)
		if e != nil || math.IsNaN(rate) || math.IsInf(rate, 0) || rate < 0 || rate > 100 {
			return nil, invalid("税率筛选不合法")
		}
		filter["tax_rate"] = rate
	}
	return filter, nil
}

func shanghaiToday(now time.Time) int64 {
	local := now.In(time.FixedZone("Asia/Shanghai", 8*3600))
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, local.Location()).Unix()
}
func sumStatus(field, status string) bson.M {
	return bson.M{"$sum": bson.M{"$cond": bson.A{bson.M{"$eq": bson.A{"$status", status}}, "$" + field, 0}}}
}
func sumEffective(field string) bson.M {
	return bson.M{"$sum": bson.M{"$cond": bson.A{bson.M{"$ne": bson.A{"$status", "voided"}}, "$" + field, 0}}}
}
func pagePipeline(filter bson.M, group bson.M, dateField string, skip, size int64) mongo.Pipeline {
	return mongo.Pipeline{
		bson.D{{Key: "$match", Value: filter}},
		bson.D{{Key: "$facet", Value: bson.M{
			"list":    bson.A{bson.M{"$sort": bson.D{{Key: dateField, Value: -1}, {Key: "_id", Value: -1}}}, bson.M{"$skip": skip}, bson.M{"$limit": size}, bson.M{"$project": bson.M{"history.before": 0}}},
			"summary": bson.A{bson.M{"$group": group}},
		}}},
	}
}
func PageAcceptance(ctx context.Context, s *svc.ServiceContext, req *types.CustomerDocumentPageRequest) *types.CustomerAcceptancePageResponse {
	resp := &types.CustomerAcceptancePageResponse{Data: types.CustomerAcceptancePage{List: []types.CustomerAcceptance{}}}
	skip, size, err := pagination(req.Page, req.Size)
	var filter bson.M
	if err == nil {
		filter, err = documentFilter(req, true)
	}
	if err == nil {
		group := bson.M{"_id": nil, "count": bson.M{"$sum": 1}, "amount": sumEffective("amount_minor"), "pending": sumStatus("amount_minor", "pending"), "redeemed": sumStatus("amount_minor", "redeemed"), "voided": sumStatus("amount_minor", "voided")}
		group["overdue"] = bson.M{"$sum": bson.M{"$cond": bson.A{bson.M{"$and": bson.A{bson.M{"$eq": bson.A{"$status", "pending"}}, bson.M{"$lt": bson.A{"$due_date", shanghaiToday(time.Now())}}}}, "$amount_minor", 0}}}
		cur, e := s.CustomerAcceptanceModel.Aggregate(ctx, pagePipeline(filter, group, "received_date", skip, size))
		err = e
		if err == nil {
			defer cur.Close(ctx)
			var results []struct {
				List    []model.CustomerAcceptance `bson:"list"`
				Summary []struct {
					Count    int64 `bson:"count"`
					Amount   int64 `bson:"amount"`
					Pending  int64 `bson:"pending"`
					Redeemed int64 `bson:"redeemed"`
					Voided   int64 `bson:"voided"`
					Overdue  int64 `bson:"overdue"`
				} `bson:"summary"`
			}
			err = cur.All(ctx, &results)
			if err == nil && len(results) > 0 {
				for _, r := range results[0].List {
					resp.Data.List = append(resp.Data.List, acceptanceView(r))
				}
				if len(results[0].Summary) > 0 {
					sum := results[0].Summary[0]
					resp.Data.Total = sum.Count
					resp.Data.Summary = types.CustomerAcceptanceSummary{Count: sum.Count, Amount: major(sum.Amount), PendingAmount: major(sum.Pending), RedeemedAmount: major(sum.Redeemed), VoidedAmount: major(sum.Voided), OverdueAmount: major(sum.Overdue)}
				}
			}
		}
	}
	resp.Code, resp.Msg = responseStatus(ctx, err)
	return resp
}
func PageInvoice(ctx context.Context, s *svc.ServiceContext, req *types.CustomerDocumentPageRequest) *types.CustomerInvoicePageResponse {
	resp := &types.CustomerInvoicePageResponse{Data: types.CustomerInvoicePage{List: []types.CustomerInvoice{}}}
	skip, size, err := pagination(req.Page, req.Size)
	var filter bson.M
	if err == nil {
		filter, err = documentFilter(req, false)
	}
	if err == nil {
		group := bson.M{"_id": nil, "count": bson.M{"$sum": 1}, "net": sumEffective("amount_excluding_tax_minor"), "tax": sumEffective("tax_amount_minor"), "total": sumEffective("total_amount_minor"), "voided": sumStatus("total_amount_minor", "voided")}
		cur, e := s.CustomerInvoiceModel.Aggregate(ctx, pagePipeline(filter, group, "invoice_date", skip, size))
		err = e
		if err == nil {
			defer cur.Close(ctx)
			var results []struct {
				List    []model.CustomerInvoice `bson:"list"`
				Summary []struct {
					Count  int64 `bson:"count"`
					Net    int64 `bson:"net"`
					Tax    int64 `bson:"tax"`
					Total  int64 `bson:"total"`
					Voided int64 `bson:"voided"`
				} `bson:"summary"`
			}
			err = cur.All(ctx, &results)
			if err == nil && len(results) > 0 {
				for _, r := range results[0].List {
					resp.Data.List = append(resp.Data.List, invoiceView(r))
				}
				if len(results[0].Summary) > 0 {
					sum := results[0].Summary[0]
					resp.Data.Total = sum.Count
					resp.Data.Summary = types.CustomerInvoiceSummary{Count: sum.Count, AmountExcludingTax: major(sum.Net), TaxAmount: major(sum.Tax), TotalAmount: major(sum.Total), VoidedAmount: major(sum.Voided)}
				}
			}
		}
	}
	resp.Code, resp.Msg = responseStatus(ctx, err)
	return resp
}

func eligiblePaymentFilter(customerID string, amount float64) bson.M {
	return bson.M{"customer_id": customerID, "amount": amount, "$and": bson.A{
		bson.M{"$or": bson.A{bson.M{"transaction_type": code.TransactionTypePayment}, bson.M{"transaction_type": bson.M{"$in": bson.A{"", nil}}, "type": "回款"}}},
		bson.M{"status": bson.M{"$in": bson.A{code.TransactionStatusConfirmed, "", nil}}},
		bson.M{"direction": bson.M{"$in": bson.A{code.TransactionDirectionDecrease, "", nil}}},
		bson.M{"source_type": bson.M{"$in": bson.A{code.SourceTypeManualAdjustment, code.SourceTypePayment, "", nil}}},
		bson.M{"acceptance_id": bson.M{"$in": bson.A{"", nil}}},
	}}
}
func Payments(ctx context.Context, s *svc.ServiceContext, req *types.CustomerAcceptancePaymentsRequest) *types.CustomerAcceptancePaymentsResponse {
	resp := &types.CustomerAcceptancePaymentsResponse{Data: types.CustomerAcceptancePaymentsPage{List: []types.CustomerAcceptancePayment{}}}
	id, err := objectID(req.CustomerId)
	var skip, size int64
	var minor int64
	if err == nil {
		minor, err = moneyMinor(req.Amount, false)
	}
	if err == nil {
		skip, size, err = pagination(req.Page, req.Size)
	}
	if err == nil {
		pipeline := mongo.Pipeline{
			bson.D{{Key: "$match", Value: eligiblePaymentFilter(id.Hex(), major(minor))}},
			bson.D{{Key: "$lookup", Value: bson.M{"from": "customer_acceptance", "let": bson.M{"payment": bson.M{"$toString": "$_id"}}, "pipeline": bson.A{bson.M{"$match": bson.M{"$expr": bson.M{"$eq": bson.A{"$transaction_id", "$$payment"}}}}, bson.M{"$limit": 1}}, "as": "bindings"}}},
			bson.D{{Key: "$match", Value: bson.M{"bindings.0": bson.M{"$exists": false}}}},
			bson.D{{Key: "$facet", Value: bson.M{"list": bson.A{bson.M{"$sort": bson.D{{Key: "time", Value: -1}, {Key: "_id", Value: -1}}}, bson.M{"$skip": skip}, bson.M{"$limit": size}}, "total": bson.A{bson.M{"$count": "count"}}}}},
		}
		cur, e := s.CustomerTransactionModel.Aggregate(ctx, pipeline)
		err = e
		if err == nil {
			defer cur.Close(ctx)
			var rows []struct {
				List  []model.CustomerTransaction `bson:"list"`
				Total []struct {
					Count int64 `bson:"count"`
				} `bson:"total"`
			}
			err = cur.All(ctx, &rows)
			if err == nil && len(rows) > 0 {
				for _, p := range rows[0].List {
					resp.Data.List = append(resp.Data.List, types.CustomerAcceptancePayment{Id: p.Id.Hex(), Code: p.Code, CustomerId: p.CustomerId, Amount: p.Amount, Time: p.Time, Remark: p.Remark, Annex: annexList(p.Annex)})
				}
				if len(rows[0].Total) > 0 {
					resp.Data.Total = rows[0].Total[0].Count
				}
			}
		}
	}
	resp.Code, resp.Msg = responseStatus(ctx, err)
	return resp
}
