package documents

import (
	finance "api/internal/logic/customer/finance"
	transactionlogic "api/internal/logic/customer/transaction"
	"api/internal/types"
	"api/model"
	code "api/pkg/code"
	"context"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestMongoUpdateReceiptsRecoverConcurrentAndLaterStateRetries(t *testing.T) {
	ctx, s, c := documentFixture(t, 1000)
	req := acceptanceRequest(c, "UPDATE-RETRY", 100)
	created := requireAcceptance(t, SaveAcceptance(ctx, s, &req, true))
	req.Id, req.Version, req.Remark = created.Id, created.Version, "更正备注"
	var wg sync.WaitGroup
	results := make(chan *types.CustomerAcceptanceResponse, 4)
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); results <- SaveAcceptance(ctx, s, &req, false) }()
	}
	wg.Wait()
	close(results)
	for result := range results {
		row := requireAcceptance(t, result)
		if row.Version != 2 || len(row.History) != 2 {
			t.Fatalf("update applied more than once: %+v", row)
		}
	}
	paid := requireAcceptance(t, RedeemAcceptance(ctx, s, &types.CustomerAcceptanceRedeemRequest{Id: req.Id, Version: 2, RedemptionDate: 1700000200, IdempotencyKey: "update-paid"}))
	if row := requireAcceptance(t, SaveAcceptance(ctx, s, &req, false)); row.Status != "redeemed" || row.Version != paid.Version {
		t.Fatalf("old update rewrote later state: %+v", row)
	}
	req.Remark = "另一项修改"
	if result := SaveAcceptance(ctx, s, &req, false); result.Code != 409 {
		t.Fatalf("different stale update accepted: %+v", result)
	}
	inv := types.CustomerInvoiceSaveRequest{IdempotencyKey: "inv-update", CustomerId: c.Id.Hex(), InvoiceNumber: "INV-UPDATE", InvoiceDate: 1700000000, AmountExcludingTax: 100, TotalAmount: 100}
	createdInvoice := SaveInvoice(ctx, s, &inv, true)
	if createdInvoice.Code != 200 {
		t.Fatal(createdInvoice)
	}
	inv.Id, inv.Version, inv.Remark = createdInvoice.Data.Id, 1, "票面备注"
	for i := 0; i < 2; i++ {
		if result := SaveInvoice(ctx, s, &inv, false); result.Code != 200 || result.Data.Version != 2 || len(result.Data.History) != 2 {
			t.Fatalf("invoice retry: %+v", result)
		}
	}
	if result := VoidInvoice(ctx, s, &types.CustomerDocumentVoidRequest{Id: inv.Id, Version: 2, Reason: "重开", IdempotencyKey: "void-updated-invoice"}); result.Code != 200 {
		t.Fatal(result)
	}
	if result := SaveInvoice(ctx, s, &inv, false); result.Code != 200 || result.Data.Status != "voided" || result.Data.Version != 3 {
		t.Fatalf("invoice retry lost latest state: %+v", result)
	}
	inv.Remark = "旧版本另一修改"
	if result := SaveInvoice(ctx, s, &inv, false); result.Code != 409 {
		t.Fatalf("different invoice update accepted: %+v", result)
	}
	if got := customerBalance(t, ctx, s, c.Id); got != 900 {
		t.Fatalf("update retry affected ledger: %v", got)
	}
}

func TestMongoAcceptanceCalendarDayPreservesLegacyTimesAndReplay(t *testing.T) {
	ctx, s, c := documentFixture(t, 1000)
	day := time.Date(2026, 10, 17, 16, 0, 0, 0, time.UTC).Unix()
	req := acceptanceRequest(c, "LEGACY-DATE-TIMES", 100)
	req.IssueDate, req.ReceivedDate, req.DueDate = day+20*3600, day+12*3600, day
	created := requireAcceptance(t, SaveAcceptance(ctx, s, &req, true))
	redeem := types.CustomerAcceptanceRedeemRequest{Id: created.Id, Version: created.Version, RedemptionDate: day - 1, IdempotencyKey: "legacy-day-redeem"}
	if result := RedeemAcceptance(ctx, s, &redeem); result.Code != 400 {
		t.Fatalf("prior Beijing day redemption accepted: %+v", result)
	}
	if paymentCount(t, ctx, s) != 0 || customerBalance(t, ctx, s, c.Id) != 1000 {
		t.Fatal("invalid redemption day changed ledger")
	}
	redeem.RedemptionDate = day
	paid := requireAcceptance(t, RedeemAcceptance(ctx, s, &redeem))
	replayed := requireAcceptance(t, RedeemAcceptance(ctx, s, &redeem))
	createdReplay := requireAcceptance(t, SaveAcceptance(ctx, s, &req, true))
	if paid.TransactionId != replayed.TransactionId || replayed.Version != 2 || createdReplay.Status != "redeemed" {
		t.Fatal("same-day validation changed action receipts")
	}
	if paid.ReceivedDate != req.ReceivedDate || paid.IssueDate != req.IssueDate || paid.DueDate != req.DueDate || paid.RedemptionDate != day {
		t.Fatalf("calendar comparison normalized persisted values: %+v", paid)
	}
	var payment model.CustomerTransaction
	if err := s.CustomerTransactionModel.FindOne(ctx, bson.M{"acceptance_id": paid.Id}).Decode(&payment); err != nil {
		t.Fatal(err)
	}
	if payment.Time != day || paymentCount(t, ctx, s) != 1 || customerBalance(t, ctx, s, c.Id) != 900 {
		t.Fatal("same-day redemption payment differs from original request")
	}
}

func TestMongoNewAttachmentBindingsEnforceScopeAndRetainOldReferences(t *testing.T) {
	ctx, s, c := documentFixture(t, 1000)
	for _, ref := range []string{"invoice/invoice.pdf", "acceptance/bill.pdf", "acceptance/redemption/proof.pdf", "legacy.pdf"} {
		if _, err := s.ImageModel.InsertOne(ctx, bson.M{"object_key": ref}); err != nil {
			t.Fatal(err)
		}
	}
	req := acceptanceRequest(c, "SCOPE", 100)
	req.Annex = []string{"invoice/invoice.pdf"}
	if result := SaveAcceptance(ctx, s, &req, true); result.Code != 400 {
		t.Fatalf("cross-scope acceptance: %+v", result)
	}
	req.Annex = []string{"acceptance/bill.pdf", "legacy.pdf"}
	created := requireAcceptance(t, SaveAcceptance(ctx, s, &req, true))
	redemption := types.CustomerAcceptanceRedeemRequest{Id: created.Id, Version: 1, RedemptionDate: 1700000200, IdempotencyKey: "scope-redeem", RedemptionAnnex: []string{"invoice/invoice.pdf"}}
	if result := RedeemAcceptance(ctx, s, &redemption); result.Code != 400 {
		t.Fatalf("cross-scope redemption: %+v", result)
	}
	if paymentCount(t, ctx, s) != 0 || customerBalance(t, ctx, s, c.Id) != 1000 {
		t.Fatal("rejected scope posted a payment")
	}
	redemption.RedemptionAnnex = []string{"acceptance/redemption/proof.pdf"}
	requireAcceptance(t, RedeemAcceptance(ctx, s, &redemption))
	inv := types.CustomerInvoiceSaveRequest{IdempotencyKey: "inv-scope", CustomerId: c.Id.Hex(), InvoiceNumber: "INV-SCOPE", InvoiceDate: 1700000000, AmountExcludingTax: 100, TotalAmount: 100, Annex: []string{"acceptance/bill.pdf"}}
	if result := SaveInvoice(ctx, s, &inv, true); result.Code != 400 {
		t.Fatalf("cross-scope invoice: %+v", result)
	}
	inv.Annex = []string{"invoice/invoice.pdf"}
	result := SaveInvoice(ctx, s, &inv, true)
	if result.Code != 200 {
		t.Fatal(result)
	}
	// An existing imported/cross-scope attachment can remain unchanged. This
	// fixture models a record created before directory validation was introduced.
	id, _ := primitive.ObjectIDFromHex(result.Data.Id)
	if _, err := s.CustomerInvoiceModel.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{"annex": "acceptance/bill.pdf"}, "$unset": bson.M{"attachment_history": ""}}); err != nil {
		t.Fatal(err)
	}
	inv.Id, inv.Version, inv.Annex = result.Data.Id, 1, []string{"acceptance/bill.pdf"}
	if updated := SaveInvoice(ctx, s, &inv, false); updated.Code != 200 {
		t.Fatalf("existing reference compatibility broken: %+v", updated)
	}
}

func TestMongoManualPaymentRetriesAndBindingKeepOneLedgerEntry(t *testing.T) {
	ctx, s, c := documentFixture(t, 1000)
	req := types.CustomerTransactionAddRequest{CustomerId: c.Id.Hex(), TransactionType: code.TransactionTypePayment, Amount: 100, Time: 1700000200, Annex: []string{"legacy.pdf"}, Remark: "到账", IdempotencyKey: "manual-audit"}
	var wg sync.WaitGroup
	responses := make(chan *types.BaseResponse, 6)
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			response, _ := transactionlogic.NewAddLogic(ctx, s).Add(&req)
			responses <- response
		}()
	}
	wg.Wait()
	close(responses)
	for response := range responses {
		if response.Code != 200 {
			t.Fatalf("concurrent payment retry: %+v", response)
		}
	}
	if got := customerBalance(t, ctx, s, c.Id); got != 900 || paymentCount(t, ctx, s) != 1 {
		t.Fatalf("manual payment duplicated: %v", got)
	}
	var payment model.CustomerTransaction
	if err := s.CustomerTransactionModel.FindOne(ctx, bson.M{"idempotency_key": req.IdempotencyKey}).Decode(&payment); err != nil {
		t.Fatal(err)
	}
	if payment.RequestHash == "" {
		t.Fatal("manual request receipt missing")
	}
	for _, mutate := range []func(*types.CustomerTransactionAddRequest){
		func(r *types.CustomerTransactionAddRequest) { r.Amount += 0.0000001 },
		func(r *types.CustomerTransactionAddRequest) { r.Time++ },
		func(r *types.CustomerTransactionAddRequest) { r.Remark = "其他内容" },
		func(r *types.CustomerTransactionAddRequest) { r.Annex = []string{"other.pdf"} },
	} {
		changed := req
		mutate(&changed)
		if response, _ := transactionlogic.NewAddLogic(ctx, s).Add(&changed); response.Code != 400 {
			t.Fatalf("changed payment request accepted: %+v", response)
		}
	}
	acceptance := acceptanceRequest(c, "MANUAL-AUDIT", 100)
	created := requireAcceptance(t, SaveAcceptance(ctx, s, &acceptance, true))
	paid := requireAcceptance(t, RedeemAcceptance(ctx, s, &types.CustomerAcceptanceRedeemRequest{Id: created.Id, Version: 1, RedemptionDate: req.Time, IdempotencyKey: "manual-bind-audit", TransactionId: payment.Id.Hex()}))
	if response, _ := transactionlogic.NewAddLogic(ctx, s).Add(&req); response.Code != 200 {
		t.Fatalf("bound payment retry: %+v", response)
	}
	requireAcceptance(t, ReverseAcceptance(ctx, s, &types.CustomerDocumentVoidRequest{Id: paid.Id, Version: paid.Version, Reason: "解除误关联", IdempotencyKey: "manual-unbind-audit"}))
	if err := transaction(ctx, s, func(dbCtx mongo.SessionContext) error { return finance.RebuildCustomerBalance(dbCtx, s, c.Id) }); err != nil {
		t.Fatal(err)
	}
	if got := customerBalance(t, ctx, s, c.Id); got != 900 || paymentCount(t, ctx, s) != 1 {
		t.Fatalf("manual binding/retry changed balance: %v", got)
	}
}

func TestMongoPostingReplayPreservesOutboundCrossPathContract(t *testing.T) {
	ctx, s, c := documentFixture(t, 0)
	req := finance.PostTransactionRequest{CustomerId: c.Id.Hex(), TransactionType: code.TransactionTypeOutboundAR, SourceType: code.SourceTypeOutboundOrder, SourceId: "outbound-audit", SourceCode: "CK-AUDIT", IdempotencyKey: "outbound_ar:outbound-audit", Amount: 100, Time: 1700000000, Remark: "极速出库自动签收"}
	for i := 0; i < 2; i++ {
		if err := transaction(ctx, s, func(dbCtx mongo.SessionContext) error {
			result, err := finance.PostTransaction(dbCtx, s, req)
			if err == nil && result.Inserted != (i == 0) {
				t.Errorf("outbound inserted=%v on attempt %d", result.Inserted, i)
			}
			return err
		}); err != nil {
			t.Fatal(err)
		}
		req.Time++
		req.Remark = "正常签收"
	}
	if got := customerBalance(t, ctx, s, c.Id); got != 100 {
		t.Fatalf("outbound retry duplicated ledger: %v", got)
	}
}

func TestMongoDocumentNumberIndexesSurviveConcurrentCreateAndVoid(t *testing.T) {
	ctx, s, c := documentFixture(t, 0)
	results := make(chan *types.CustomerAcceptanceResponse, 2)
	var wg sync.WaitGroup
	for _, bill := range []string{"  Unique-Bill  ", "unique-bill"} {
		wg.Add(1)
		go func(number string) {
			defer wg.Done()
			req := acceptanceRequest(c, number, 100)
			results <- SaveAcceptance(ctx, s, &req, true)
		}(bill)
	}
	wg.Wait()
	close(results)
	wins, conflicts := 0, 0
	var row types.CustomerAcceptance
	for response := range results {
		switch response.Code {
		case 200:
			wins++
			row = response.Data
		case 409:
			conflicts++
		default:
			t.Fatalf("unexpected number race result: %+v", response)
		}
	}
	if wins != 1 || conflicts != 1 {
		t.Fatalf("unique index did not arbitrate: wins=%d conflicts=%d", wins, conflicts)
	}
	requireAcceptance(t, VoidAcceptance(ctx, s, &types.CustomerDocumentVoidRequest{Id: row.Id, Version: row.Version, Reason: "录错", IdempotencyKey: "void-unique"}))
	req := acceptanceRequest(c, "UNIQUE-BILL", 100)
	if result := SaveAcceptance(ctx, s, &req, true); result.Code != 409 {
		t.Fatalf("void released bill number: %+v", result)
	}
	first := types.CustomerInvoiceSaveRequest{IdempotencyKey: "inv-unique-first", CustomerId: c.Id.Hex(), InvoiceNumber: "Unique-Invoice", InvoiceDate: 1700000000, AmountExcludingTax: 100, TotalAmount: 100}
	created := SaveInvoice(ctx, s, &first, true)
	if created.Code != 200 {
		t.Fatal(created)
	}
	if result := VoidInvoice(ctx, s, &types.CustomerDocumentVoidRequest{Id: created.Data.Id, Version: 1, Reason: "重开", IdempotencyKey: "void-unique-invoice"}); result.Code != 200 {
		t.Fatal(result)
	}
	first.IdempotencyKey, first.InvoiceNumber = "inv-unique-other", " UNIQUE-INVOICE "
	if result := SaveInvoice(ctx, s, &first, true); result.Code != 409 {
		t.Fatalf("void released invoice number: %+v", result)
	}
}

// This opt-in fault injection is restricted by documentFixture to a disposable
// loopback database. Run separately from other failCommand tests: MongoDB has
// one global failpoint slot even when each injection has a private appName.
func TestMongoCommitAcknowledgementFailurePreservesFinancialReceipts(t *testing.T) {
	if os.Getenv("WMS_TEST_MONGO_FAILPOINTS") != "1" {
		t.Skip("requires disposable MongoDB failpoint opt-in")
	}
	ctx, original, c := documentFixture(t, 1000)
	appName := "wms_document_commit_" + primitive.NewObjectID().Hex()
	var injected atomic.Int32
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(os.Getenv("WMS_TEST_MONGO_URI")).SetAppName(appName).SetMonitor(&event.CommandMonitor{Succeeded: func(_ context.Context, e *event.CommandSucceededEvent) {
		if e.CommandName == "commitTransaction" && e.Reply.Lookup("writeConcernError").Type != 0 {
			injected.Add(1)
		}
	}}))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Disconnect(context.Background())
	db := client.Database(original.CustomerModel.Database().Name())
	s := *original
	s.DBClient, s.CustomerModel, s.CustomerTransactionModel = client, db.Collection("customer"), db.Collection("customer_transaction")
	s.CustomerAcceptanceModel, s.CustomerInvoiceModel, s.ImageModel = db.Collection("customer_acceptance"), db.Collection("customer_invoice"), db.Collection("image")
	admin := original.DBClient.Database("admin")
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = admin.RunCommand(cleanup, bson.D{{Key: "configureFailPoint", Value: "failCommand"}, {Key: "mode", Value: "off"}}).Err()
	}()
	runUncertainCommit := func(action func()) {
		t.Helper()
		before := injected.Load()
		cmd := bson.D{{Key: "configureFailPoint", Value: "failCommand"}, {Key: "mode", Value: bson.M{"times": 1}}, {Key: "data", Value: bson.M{"failCommands": bson.A{"commitTransaction"}, "appName": appName, "writeConcernError": bson.M{"code": 64, "errmsg": "isolated commit acknowledgement failure"}}}}
		if err := admin.RunCommand(ctx, cmd).Err(); err != nil {
			t.Fatal(err)
		}
		action()
		if injected.Load() != before+1 {
			t.Fatal("commit acknowledgement failure was not exercised")
		}
	}
	manual := types.CustomerTransactionAddRequest{CustomerId: c.Id.Hex(), TransactionType: code.TransactionTypePayment, Amount: 100, IdempotencyKey: "uncertain-manual"}
	runUncertainCommit(func() {
		if response, _ := transactionlogic.NewAddLogic(ctx, &s).Add(&manual); response.Code != 200 {
			t.Fatalf("manual commit recovery: %+v", response)
		}
	})
	if response, _ := transactionlogic.NewAddLogic(ctx, &s).Add(&manual); response.Code != 200 {
		t.Fatalf("manual replay: %+v", response)
	}
	req := acceptanceRequest(c, "UNCERTAIN", 100)
	var row types.CustomerAcceptance
	runUncertainCommit(func() { row = requireAcceptance(t, SaveAcceptance(ctx, &s, &req, true)) })
	req.Id, req.Version, req.Remark = row.Id, row.Version, "修改后结果不确定"
	runUncertainCommit(func() { row = requireAcceptance(t, SaveAcceptance(ctx, &s, &req, false)) })
	if replay := requireAcceptance(t, SaveAcceptance(ctx, &s, &req, false)); replay.Version != row.Version {
		t.Fatal("update commit retry duplicated history")
	}
	redeem := types.CustomerAcceptanceRedeemRequest{Id: row.Id, Version: row.Version, RedemptionDate: 1700000200, IdempotencyKey: "uncertain-redeem"}
	runUncertainCommit(func() { row = requireAcceptance(t, RedeemAcceptance(ctx, &s, &redeem)) })
	requireAcceptance(t, RedeemAcceptance(ctx, &s, &redeem))
	reverse := types.CustomerDocumentVoidRequest{Id: row.Id, Version: row.Version, Reason: "撤销错误兑付", IdempotencyKey: "uncertain-reverse"}
	runUncertainCommit(func() { row = requireAcceptance(t, ReverseAcceptance(ctx, &s, &reverse)) })
	requireAcceptance(t, ReverseAcceptance(ctx, &s, &reverse))
	if row.Status != "pending" || row.Version != 4 || paymentCount(t, ctx, &s) != 2 || customerBalance(t, ctx, &s, c.Id) != 900 {
		t.Fatalf("uncertain commit lost consistency: %+v", row)
	}
	if err := transaction(ctx, &s, func(dbCtx mongo.SessionContext) error { return finance.RebuildCustomerBalance(dbCtx, &s, c.Id) }); err != nil {
		t.Fatal(err)
	}
	if customerBalance(t, ctx, &s, c.Id) != 900 {
		t.Fatal("uncertain commit cache differs from ledger")
	}
}
