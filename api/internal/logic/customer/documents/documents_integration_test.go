package documents

import (
	customerlogic "api/internal/logic/customer"
	finance "api/internal/logic/customer/finance"
	"api/internal/svc"
	"api/internal/types"
	"api/model"
	code "api/pkg/code"
	"context"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// These tests never read application configuration. Each creates and drops only
// its own uniquely named database on an explicitly selected local replica set.
func documentFixture(t *testing.T, opening float64) (context.Context, *svc.ServiceContext, model.Customer) {
	t.Helper()
	uri := os.Getenv("WMS_TEST_MONGO_URI")
	if uri == "" {
		t.Skip("set WMS_TEST_MONGO_URI to a disposable loopback MongoDB replica set")
	}
	u, err := url.Parse(uri)
	if err != nil || u.Scheme != "mongodb" || u.User != nil || strings.Contains(u.Host, ",") || (u.Hostname() != "127.0.0.1" && u.Hostname() != "localhost" && u.Hostname() != "::1") {
		t.Fatal("only credential-free loopback MongoDB is allowed")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	ctx = context.WithValue(ctx, "uid", primitive.NewObjectID().Hex())
	ctx = context.WithValue(ctx, "name", "票据集成测试")
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri).SetServerSelectionTimeout(3*time.Second))
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	db := client.Database("wms_documents_test_" + primitive.NewObjectID().Hex())
	t.Cleanup(func() {
		cleanup, done := context.WithTimeout(context.Background(), 10*time.Second)
		defer done()
		if err := db.Drop(cleanup); err != nil {
			t.Errorf("drop isolated database: %v", err)
		}
		_ = client.Disconnect(cleanup)
		cancel()
	})
	for _, name := range []string{"customer", "customer_transaction", "customer_acceptance", "customer_invoice", "image"} {
		if err = db.CreateCollection(ctx, name); err != nil {
			t.Fatal(err)
		}
	}
	s := &svc.ServiceContext{DBClient: client, CustomerModel: db.Collection("customer"), CustomerTransactionModel: db.Collection("customer_transaction"), CustomerAcceptanceModel: db.Collection("customer_acceptance"), CustomerInvoiceModel: db.Collection("customer_invoice"), ImageModel: db.Collection("image")}
	if err = svc.EnsureCustomerDocumentIndexes(ctx, s); err != nil {
		t.Fatal(err)
	}
	_, err = s.CustomerTransactionModel.Indexes().CreateOne(ctx, mongo.IndexModel{Keys: bson.D{{Key: "idempotency_key", Value: 1}}, Options: options.Index().SetUnique(true).SetPartialFilterExpression(bson.M{"idempotency_key": bson.M{"$type": "string", "$gt": ""}})})
	if err != nil {
		t.Fatal(err)
	}
	customer := model.Customer{Id: primitive.NewObjectID(), Name: "测试客户", Status: "活动", ReceivableBalance: opening}
	if _, err = s.CustomerModel.InsertOne(ctx, customer); err != nil {
		t.Fatal(err)
	}
	if opening > 0 {
		_, err = s.CustomerTransactionModel.InsertOne(ctx, model.CustomerTransaction{Id: primitive.NewObjectID(), CustomerId: customer.Id.Hex(), TransactionType: code.TransactionTypeOpeningAR, Direction: code.TransactionDirectionIncrease, Status: code.TransactionStatusConfirmed, Amount: opening, IdempotencyKey: "opening"})
		if err != nil {
			t.Fatal(err)
		}
	}
	return ctx, s, customer
}
func acceptanceRequest(customer model.Customer, number string, amount float64) types.CustomerAcceptanceSaveRequest {
	return types.CustomerAcceptanceSaveRequest{IdempotencyKey: "create:" + number, CustomerId: customer.Id.Hex(), BillNumber: number, Amount: amount, IssueDate: 1700000000, ReceivedDate: 1700000100, DueDate: 1800000000}
}
func requireAcceptance(t *testing.T, response *types.CustomerAcceptanceResponse) types.CustomerAcceptance {
	t.Helper()
	if response.Code != 200 {
		t.Fatalf("acceptance: code=%d msg=%s", response.Code, response.Msg)
	}
	return response.Data
}
func customerBalance(t *testing.T, ctx context.Context, s *svc.ServiceContext, id primitive.ObjectID) float64 {
	t.Helper()
	var customer model.Customer
	if err := s.CustomerModel.FindOne(ctx, bson.M{"_id": id}).Decode(&customer); err != nil {
		t.Fatal(err)
	}
	return customer.ReceivableBalance - customer.CreditBalance
}
func paymentCount(t *testing.T, ctx context.Context, s *svc.ServiceContext) int64 {
	t.Helper()
	n, err := s.CustomerTransactionModel.CountDocuments(ctx, bson.M{"transaction_type": code.TransactionTypePayment})
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func TestMongoRegistrationEditVoidAndSummaryLeaveBalancesUntouched(t *testing.T) {
	ctx, s, c := documentFixture(t, 1000)
	req := acceptanceRequest(c, "B-1", 100.25)
	first := requireAcceptance(t, SaveAcceptance(ctx, s, &req, true))
	repeated := requireAcceptance(t, SaveAcceptance(ctx, s, &req, true))
	if first.Id != repeated.Id || first.Version != 1 {
		t.Fatal("create retry duplicated document")
	}
	changed := req
	changed.Amount = 200
	if resp := SaveAcceptance(ctx, s, &changed, true); resp.Code != 409 {
		t.Fatalf("key conflict=%+v", resp)
	}
	duplicate := req
	duplicate.IdempotencyKey = "other-key"
	if resp := SaveAcceptance(ctx, s, &duplicate, true); resp.Code != 409 {
		t.Fatalf("duplicate number=%+v", resp)
	}
	req.Id, req.Version = first.Id, first.Version
	req.Amount = 120.25
	updated := requireAcceptance(t, SaveAcceptance(ctx, s, &req, false))
	if updated.Version != 2 {
		t.Fatal("version not incremented")
	}
	if resp := SaveAcceptance(ctx, s, &req, false); resp.Code != 200 || resp.Data.Version != 2 {
		t.Fatalf("identical update retry not recovered: %+v", resp)
	}
	stale := req
	stale.Amount = 121
	if resp := SaveAcceptance(ctx, s, &stale, false); resp.Code != 409 {
		t.Fatalf("changed stale edit accepted: %+v", resp)
	}
	voidReq := types.CustomerDocumentVoidRequest{Id: first.Id, Version: updated.Version, Reason: "录入错误", IdempotencyKey: "void:b1"}
	voided := requireAcceptance(t, VoidAcceptance(ctx, s, &voidReq))
	requireAcceptance(t, VoidAcceptance(ctx, s, &voidReq))
	if voided.Status != "voided" {
		t.Fatal("void not persisted")
	}
	secondReq := acceptanceRequest(c, "B-2", 200)
	requireAcceptance(t, SaveAcceptance(ctx, s, &secondReq, true))
	page := PageAcceptance(ctx, s, &types.CustomerDocumentPageRequest{Page: 1, Size: 1})
	if page.Code != 200 || page.Data.Total != 2 || len(page.Data.List) != 1 || page.Data.Summary.Amount != 200 || page.Data.Summary.VoidedAmount != 120.25 {
		t.Fatalf("whole-filter summary: %+v", page)
	}
	filtered := PageAcceptance(ctx, s, &types.CustomerDocumentPageRequest{Status: "pending", Keyword: "B-2"})
	if filtered.Code != 200 || filtered.Data.Total != 1 {
		t.Fatalf("filtered summary: %+v", filtered)
	}
	if got := customerBalance(t, ctx, s, c.Id); got != 1000 || paymentCount(t, ctx, s) != 0 {
		t.Fatalf("registration/edit/void changed balance %v", got)
	}
}

func TestMongoRedeemConcurrentRetryPostsExactlyOnce(t *testing.T) {
	ctx, s, c := documentFixture(t, 1000)
	req := acceptanceRequest(c, "B-R", 100)
	row := requireAcceptance(t, SaveAcceptance(ctx, s, &req, true))
	redeem := types.CustomerAcceptanceRedeemRequest{Id: row.Id, Version: row.Version, RedemptionDate: 1700000200, IdempotencyKey: "redeem:r"}
	start := make(chan struct{})
	responses := make(chan *types.CustomerAcceptanceResponse, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); <-start; responses <- RedeemAcceptance(ctx, s, &redeem) }()
	}
	close(start)
	wg.Wait()
	close(responses)
	transactionID := ""
	for response := range responses {
		got := requireAcceptance(t, response)
		if got.Status != "redeemed" || got.TransactionId == "" {
			t.Fatalf("redeemed row: %+v", got)
		}
		if transactionID != "" && transactionID != got.TransactionId {
			t.Fatal("retry produced a second payment")
		}
		transactionID = got.TransactionId
	}
	if got := customerBalance(t, ctx, s, c.Id); got != 900 || paymentCount(t, ctx, s) != 1 {
		t.Fatalf("redeem balance=%v count=%d", got, paymentCount(t, ctx, s))
	}
	if resp := VoidAcceptance(ctx, s, &types.CustomerDocumentVoidRequest{Id: row.Id, Version: 2, Reason: "不能撤销已到账", IdempotencyKey: "void-r"}); resp.Code != 409 {
		t.Fatalf("redeemed void accepted: %+v", resp)
	}
	if resp := SaveAcceptance(ctx, s, &types.CustomerAcceptanceSaveRequest{Id: row.Id, Version: 2, CustomerId: c.Id.Hex(), BillNumber: row.BillNumber, Amount: 100, IssueDate: row.IssueDate, ReceivedDate: row.ReceivedDate, DueDate: row.DueDate}, false); resp.Code != 409 {
		t.Fatalf("redeemed edit accepted: %+v", resp)
	}
	var record model.CustomerTransaction
	paymentID, _ := primitive.ObjectIDFromHex(transactionID)
	if err := s.CustomerTransactionModel.FindOne(ctx, bson.M{"_id": paymentID}).Decode(&record); err != nil {
		t.Fatal(err)
	}
	if record.SourceType != acceptanceSource || record.SourceId != row.Id || record.AcceptanceId != row.Id {
		t.Fatalf("payment lost source link: %+v", record)
	}
	if err := transaction(ctx, s, func(dbCtx mongo.SessionContext) error { return finance.RebuildCustomerBalance(dbCtx, s, c.Id) }); err != nil {
		t.Fatal(err)
	}
	if got := customerBalance(t, ctx, s, c.Id); got != 900 {
		t.Fatalf("ledger rebuild changed redeem effect: %v", got)
	}
}

func TestMongoExistingPaymentCanBindOnlyOneAcceptanceWithoutSecondReduction(t *testing.T) {
	ctx, s, c := documentFixture(t, 1000)
	var paymentID string
	if err := transaction(ctx, s, func(dbCtx mongo.SessionContext) error {
		result, err := finance.PostTransaction(dbCtx, s, finance.PostTransactionRequest{TransactionType: code.TransactionTypePayment, SourceType: code.SourceTypeManualAdjustment, SourceId: "manual-1", IdempotencyKey: "manual-1", CustomerId: c.Id.Hex(), CustomerName: c.Name, Amount: 100, Time: 1700000200})
		if err == nil {
			paymentID = result.Transaction.Id.Hex()
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	payments := Payments(ctx, s, &types.CustomerAcceptancePaymentsRequest{CustomerId: c.Id.Hex(), Amount: 100})
	if payments.Code != 200 || payments.Data.Total != 1 || payments.Data.List[0].Id != paymentID {
		t.Fatalf("eligible payments: %+v", payments)
	}
	firstReq, secondReq := acceptanceRequest(c, "B-L1", 100), acceptanceRequest(c, "B-L2", 100)
	first, second := requireAcceptance(t, SaveAcceptance(ctx, s, &firstReq, true)), requireAcceptance(t, SaveAcceptance(ctx, s, &secondReq, true))
	start := make(chan struct{})
	responses := make(chan *types.CustomerAcceptanceResponse, 2)
	var wg sync.WaitGroup
	for _, row := range []types.CustomerAcceptance{first, second} {
		wg.Add(1)
		go func(row types.CustomerAcceptance) {
			defer wg.Done()
			<-start
			responses <- RedeemAcceptance(ctx, s, &types.CustomerAcceptanceRedeemRequest{Id: row.Id, Version: row.Version, RedemptionDate: 1700000200, IdempotencyKey: "bind:" + row.Id, TransactionId: paymentID})
		}(row)
	}
	close(start)
	wg.Wait()
	close(responses)
	successes, conflicts := 0, 0
	for resp := range responses {
		if resp.Code == 200 {
			successes++
		} else if resp.Code == 409 {
			conflicts++
		} else {
			t.Fatalf("unexpected binding result: %+v", resp)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("binding results success=%d conflict=%d", successes, conflicts)
	}
	if got := customerBalance(t, ctx, s, c.Id); got != 900 || paymentCount(t, ctx, s) != 1 {
		t.Fatalf("binding changed balance again: %v", got)
	}
	payments = Payments(ctx, s, &types.CustomerAcceptancePaymentsRequest{CustomerId: c.Id.Hex(), Amount: 100})
	if payments.Code != 200 || payments.Data.Total != 0 {
		t.Fatalf("occupied payment still offered: %+v", payments)
	}
}

func TestMongoInvoiceDoesNotPostAndProtectsCustomerAndAttachmentHistory(t *testing.T) {
	ctx, s, c := documentFixture(t, 0)
	for _, ref := range []string{"invoice/a.pdf", "invoice/b.pdf", "invoice/deleting.pdf"} {
		_, err := s.ImageModel.InsertOne(ctx, bson.M{"object_key": ref, "deleting": ref == "invoice/deleting.pdf"})
		if err != nil {
			t.Fatal(err)
		}
	}
	req := types.CustomerInvoiceSaveRequest{IdempotencyKey: "invoice-1", CustomerId: c.Id.Hex(), InvoiceNumber: "FP1", InvoiceDate: 1700000000, TaxRate: 13, AmountExcludingTax: 100, TaxAmount: 13.01, TotalAmount: 113.01, Annex: []string{"invoice/a.pdf"}}
	created := SaveInvoice(ctx, s, &req, true)
	if created.Code != 200 {
		t.Fatalf("invoice create: %+v", created)
	}
	if replay := SaveInvoice(ctx, s, &req, true); replay.Code != 200 || replay.Data.Id != created.Data.Id {
		t.Fatalf("invoice retry: %+v", replay)
	}
	deleted, err := customerlogic.NewStatusLogic(ctx, s).Status(&types.CustomerStatusRequest{Id: c.Id.Hex(), Status: "删除"})
	if err != nil || deleted.Code != 400 {
		t.Fatalf("customer with invoice deleted: %+v %v", deleted, err)
	}
	req.Id, req.Version = created.Data.Id, created.Data.Version
	req.Annex = []string{"invoice/b.pdf"}
	updated := SaveInvoice(ctx, s, &req, false)
	if updated.Code != 200 {
		t.Fatalf("invoice update: %+v", updated)
	}
	id, _ := primitive.ObjectIDFromHex(created.Data.Id)
	var stored model.CustomerInvoice
	if err = s.CustomerInvoiceModel.FindOne(ctx, bson.M{"_id": id}).Decode(&stored); err != nil {
		t.Fatal(err)
	}
	if len(stored.AttachmentHistory) != 2 || len(stored.History) != 2 || stored.History[1].Before["annex"] != "invoice/a.pdf" {
		t.Fatalf("attachment audit lost: %+v", stored)
	}
	req.Version = updated.Data.Version
	req.Annex = []string{"invoice/deleting.pdf"}
	if resp := SaveInvoice(ctx, s, &req, false); resp.Code != 400 {
		t.Fatalf("deleting attachment accepted: %+v", resp)
	}
	req.Annex = []string{"invoice/missing.pdf"}
	if resp := SaveInvoice(ctx, s, &req, false); resp.Code != 400 {
		t.Fatalf("missing attachment accepted: %+v", resp)
	}
	voidReq := types.CustomerDocumentVoidRequest{Id: created.Data.Id, Version: updated.Data.Version, Reason: "重新开具", IdempotencyKey: "void-fp1"}
	if resp := VoidInvoice(ctx, s, &voidReq); resp.Code != 200 {
		t.Fatalf("invoice void: %+v", resp)
	}
	if resp := VoidInvoice(ctx, s, &voidReq); resp.Code != 200 {
		t.Fatalf("invoice void retry: %+v", resp)
	}
	page := PageInvoice(ctx, s, &types.CustomerDocumentPageRequest{})
	if page.Code != 200 || page.Data.Total != 1 || page.Data.Summary.TotalAmount != 0 || page.Data.Summary.VoidedAmount != 113.01 {
		t.Fatalf("invoice summary: %+v", page)
	}
	if got := customerBalance(t, ctx, s, c.Id); got != 0 || paymentCount(t, ctx, s) != 0 {
		t.Fatalf("invoice altered balance: %v", got)
	}
}

func TestMongoReverseGeneratedPaymentRestoresBalanceAndAllowsNewRedemption(t *testing.T) {
	ctx, s, c := documentFixture(t, 1000)
	req := acceptanceRequest(c, "B-REV", 100)
	created := requireAcceptance(t, SaveAcceptance(ctx, s, &req, true))
	redeem := types.CustomerAcceptanceRedeemRequest{Id: created.Id, Version: created.Version, RedemptionDate: 1700000200, IdempotencyKey: "redeem-original"}
	paid := requireAcceptance(t, RedeemAcceptance(ctx, s, &redeem))
	reverse := types.CustomerDocumentVoidRequest{Id: paid.Id, Version: paid.Version, Reason: "到账登记有误", IdempotencyKey: "reverse-original"}
	start := make(chan struct{})
	results := make(chan *types.CustomerAcceptanceResponse, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); <-start; results <- ReverseAcceptance(ctx, s, &reverse) }()
	}
	close(start)
	wg.Wait()
	close(results)
	var pending types.CustomerAcceptance
	for response := range results {
		pending = requireAcceptance(t, response)
		if pending.Status != "pending" || pending.Version != 3 || pending.TransactionId != "" || pending.RedemptionDate != 0 {
			t.Fatalf("bad reversal: %+v", pending)
		}
	}
	if got := customerBalance(t, ctx, s, c.Id); got != 1000 {
		t.Fatalf("reversal balance=%v", got)
	}
	var oldPayment model.CustomerTransaction
	paymentID, _ := primitive.ObjectIDFromHex(paid.TransactionId)
	if err := s.CustomerTransactionModel.FindOne(ctx, bson.M{"_id": paymentID}).Decode(&oldPayment); err != nil {
		t.Fatal(err)
	}
	if oldPayment.Status != code.TransactionStatusReversed || oldPayment.AcceptanceId != "" {
		t.Fatalf("old payment not preserved and reversed: %+v", oldPayment)
	}
	if err := transaction(ctx, s, func(dbCtx mongo.SessionContext) error { return finance.RebuildCustomerBalance(dbCtx, s, c.Id) }); err != nil {
		t.Fatal(err)
	}
	if got := customerBalance(t, ctx, s, c.Id); got != 1000 {
		t.Fatalf("rebuild after reversal=%v", got)
	}
	redeem.Version = pending.Version
	redeem.IdempotencyKey = "redeem-corrected"
	redeem.RedemptionDate = 1700000300
	corrected := requireAcceptance(t, RedeemAcceptance(ctx, s, &redeem))
	requireAcceptance(t, RedeemAcceptance(ctx, s, &redeem))
	if corrected.TransactionId == paid.TransactionId {
		t.Fatal("redemption reused reversed payment")
	}
	if got := customerBalance(t, ctx, s, c.Id); got != 900 || paymentCount(t, ctx, s) != 2 {
		t.Fatalf("new redemption balance=%v payments=%d", got, paymentCount(t, ctx, s))
	}
	// Retrying the old reversal after further work returns current state without
	// reversing the new round of payment.
	requireAcceptance(t, ReverseAcceptance(ctx, s, &reverse))
	if got := customerBalance(t, ctx, s, c.Id); got != 900 {
		t.Fatalf("old reverse replay affected new payment: %v", got)
	}
}

func TestMongoReverseManualBindingOnlyUnlinksOriginalPayment(t *testing.T) {
	ctx, s, c := documentFixture(t, 1000)
	var payment model.CustomerTransaction
	if err := transaction(ctx, s, func(dbCtx mongo.SessionContext) error {
		p, e := finance.PostTransaction(dbCtx, s, finance.PostTransactionRequest{TransactionType: code.TransactionTypePayment, SourceType: code.SourceTypeManualAdjustment, SourceId: "manual-reverse", IdempotencyKey: "manual-reverse", CustomerId: c.Id.Hex(), Amount: 100})
		if e == nil {
			payment = p.Transaction
		}
		return e
	}); err != nil {
		t.Fatal(err)
	}
	req := acceptanceRequest(c, "B-MREV", 100)
	created := requireAcceptance(t, SaveAcceptance(ctx, s, &req, true))
	paid := requireAcceptance(t, RedeemAcceptance(ctx, s, &types.CustomerAcceptanceRedeemRequest{Id: created.Id, Version: created.Version, RedemptionDate: 1700000200, IdempotencyKey: "bind-m", TransactionId: payment.Id.Hex()}))
	pending := requireAcceptance(t, ReverseAcceptance(ctx, s, &types.CustomerDocumentVoidRequest{Id: paid.Id, Version: paid.Version, Reason: "关联错误", IdempotencyKey: "unlink-m"}))
	var saved model.CustomerTransaction
	if err := s.CustomerTransactionModel.FindOne(ctx, bson.M{"_id": payment.Id}).Decode(&saved); err != nil {
		t.Fatal(err)
	}
	if saved.Status != code.TransactionStatusConfirmed || saved.AcceptanceId != "" || saved.SourceType != code.SourceTypeManualAdjustment {
		t.Fatalf("original payment mutated: %+v", saved)
	}
	if got := customerBalance(t, ctx, s, c.Id); got != 900 || paymentCount(t, ctx, s) != 1 {
		t.Fatalf("manual unlink balance=%v", got)
	}
	available := Payments(ctx, s, &types.CustomerAcceptancePaymentsRequest{CustomerId: c.Id.Hex(), Amount: 100})
	if available.Code != 200 || available.Data.Total != 1 {
		t.Fatalf("unlinked payment unavailable: %+v", available)
	}
	requireAcceptance(t, RedeemAcceptance(ctx, s, &types.CustomerAcceptanceRedeemRequest{Id: pending.Id, Version: pending.Version, RedemptionDate: 1700000200, IdempotencyKey: "rebind-m", TransactionId: payment.Id.Hex()}))
	if got := customerBalance(t, ctx, s, c.Id); got != 900 || paymentCount(t, ctx, s) != 1 {
		t.Fatalf("manual rebind balance=%v", got)
	}
}

func TestMongoDueRedemptionAndTaxRateFilters(t *testing.T) {
	ctx, s, c := documentFixture(t, 1000)
	today := shanghaiToday(time.Now())
	for _, tt := range []struct {
		number string
		due    int64
		amount float64
	}{{"OLD", today - 86400, 100}, {"TODAY", today, 200}, {"SOON", today + 6*86400, 300}, {"LATER", today + 7*86400, 400}} {
		req := acceptanceRequest(c, tt.number, tt.amount)
		req.DueDate = tt.due
		requireAcceptance(t, SaveAcceptance(ctx, s, &req, true))
	}
	all := PageAcceptance(ctx, s, &types.CustomerDocumentPageRequest{})
	if all.Code != 200 || all.Data.Summary.OverdueAmount != 100 {
		t.Fatalf("overdue summary: %+v", all)
	}
	overdue := PageAcceptance(ctx, s, &types.CustomerDocumentPageRequest{DueState: "overdue"})
	if overdue.Code != 200 || overdue.Data.Total != 1 || overdue.Data.List[0].BillNumber != "OLD" {
		t.Fatalf("overdue filter: %+v", overdue)
	}
	soon := PageAcceptance(ctx, s, &types.CustomerDocumentPageRequest{DueState: "soon"})
	if soon.Code != 200 || soon.Data.Total != 2 || soon.Data.Summary.Amount != 500 {
		t.Fatalf("soon filter: %+v", soon)
	}
	old := overdue.Data.List[0]
	requireAcceptance(t, RedeemAcceptance(ctx, s, &types.CustomerAcceptanceRedeemRequest{Id: old.Id, Version: old.Version, RedemptionDate: 1700000500, IdempotencyKey: "pay-old"}))
	paid := PageAcceptance(ctx, s, &types.CustomerDocumentPageRequest{RedemptionStart: 1700000400, RedemptionEnd: 1700000600})
	if paid.Code != 200 || paid.Data.Total != 1 || paid.Data.Summary.RedeemedAmount != 100 || paid.Data.Summary.OverdueAmount != 0 {
		t.Fatalf("redemption filter: %+v", paid)
	}
	for _, rate := range []float64{0, 13} {
		req := types.CustomerInvoiceSaveRequest{CustomerId: c.Id.Hex(), InvoiceNumber: primitive.NewObjectID().Hex(), InvoiceDate: 1700000000, TaxRate: rate, AmountExcludingTax: 100, TaxAmount: rate, TotalAmount: 100 + rate, IdempotencyKey: primitive.NewObjectID().Hex()}
		if resp := SaveInvoice(ctx, s, &req, true); resp.Code != 200 {
			t.Fatalf("invoice: %+v", resp)
		}
	}
	zero := PageInvoice(ctx, s, &types.CustomerDocumentPageRequest{TaxRate: "0"})
	if zero.Code != 200 || zero.Data.Total != 1 || zero.Data.Summary.TotalAmount != 100 {
		t.Fatalf("zero-tax filter: %+v", zero)
	}
}

func TestMongoDocumentRegistrationAndCustomerDeletionCannotCreateOrphan(t *testing.T) {
	ctx, s, c := documentFixture(t, 0)
	start := make(chan struct{})
	var wg sync.WaitGroup
	req := acceptanceRequest(c, "RACE-DELETE", 100)
	wg.Add(2)
	go func() { defer wg.Done(); <-start; SaveAcceptance(ctx, s, &req, true) }()
	go func() {
		defer wg.Done()
		<-start
		_, _ = customerlogic.NewStatusLogic(ctx, s).Status(&types.CustomerStatusRequest{Id: c.Id.Hex(), Status: "删除"})
	}()
	close(start)
	wg.Wait()
	var customer model.Customer
	if err := s.CustomerModel.FindOne(ctx, bson.M{"_id": c.Id}).Decode(&customer); err != nil {
		t.Fatal(err)
	}
	n, err := s.CustomerAcceptanceModel.CountDocuments(ctx, bson.M{"customer_id": c.Id.Hex()})
	if err != nil {
		t.Fatal(err)
	}
	if customer.IsDeleted && n != 0 {
		t.Fatal("concurrent deletion orphaned acceptance")
	}
	if !customer.IsDeleted && n != 1 {
		t.Fatal("both independent operations failed unexpectedly")
	}
}
