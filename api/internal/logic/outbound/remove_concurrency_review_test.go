package outbound

import (
	"api/internal/svc"
	"api/internal/types"
	"api/model"
	code "api/pkg/code"
	"context"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/mongo"
)

// Pauses the first N matching writes after their callers have read the source
// snapshot. Tests establish the interleaving explicitly instead of hoping that
// goroutines happen to execute simultaneously.
func reviewWriteBarrier(command, collection string, count int64) (*event.CommandMonitor, <-chan struct{}, func()) {
	ready := make(chan struct{}, count)
	released := make(chan struct{})
	var attempts atomic.Int64
	var once sync.Once
	monitor := &event.CommandMonitor{Started: func(ctx context.Context, e *event.CommandStartedEvent) {
		if e.CommandName != command {
			return
		}
		name, ok := e.Command.Lookup(command).StringValueOK()
		if !ok || name != collection || attempts.Add(1) > count {
			return
		}
		ready <- struct{}{}
		select {
		case <-released:
		case <-ctx.Done():
		}
	}}
	return monitor, ready, func() { once.Do(func() { close(released) }) }
}

func reviewAwaitWrite(t *testing.T, ready <-chan struct{}) {
	t.Helper()
	select {
	case <-ready:
	case <-time.After(10 * time.Second):
		t.Fatal("command did not reach the write barrier")
	}
}

type reviewMutationResult struct {
	id       primitive.ObjectID
	response *types.BaseResponse
	err      error
}

func reviewAwaitResult(t *testing.T, results <-chan reviewMutationResult) reviewMutationResult {
	t.Helper()
	select {
	case result := <-results:
		if result.err != nil || result.response == nil {
			t.Fatalf("mutation response=%+v err=%v", result.response, result.err)
		}
		return result
	case <-time.After(10 * time.Second):
		t.Fatal("mutation did not finish after releasing the write barrier")
		return reviewMutationResult{}
	}
}

func reviewStartRemoval(ctx context.Context, s *svc.ServiceContext, id primitive.ObjectID, token string, results chan<- reviewMutationResult) {
	go func() {
		response, err := NewRemoveLogic(ctx, s).Remove(&types.OutboundOrderRemoveRequest{Id: id.Hex(), PreviewToken: token})
		results <- reviewMutationResult{id: id, response: response, err: err}
	}()
}

func reviewBalance(t *testing.T, ctx context.Context, s *svc.ServiceContext, id primitive.ObjectID, receivable, credit float64) {
	t.Helper()
	var customer model.Customer
	if err := s.CustomerModel.FindOne(ctx, bson.M{"_id": id}).Decode(&customer); err != nil {
		t.Fatal(err)
	}
	if customer.ReceivableBalance != receivable || customer.CreditBalance != credit {
		t.Fatalf("expected AR=%g credit=%g, got AR=%g credit=%g", receivable, credit, customer.ReceivableBalance, customer.CreditBalance)
	}
}

func TestRemovalReviewPaidSignedConcurrentReplayPreservesPayment(t *testing.T) {
	monitor, ready, release := reviewWriteBarrier("delete", "outbound_order", 2)
	s, ctx := newRemovalIntegrationService(t, monitor)
	t.Cleanup(release)
	f := seedRemovalIntegrationFixture(t, s, "已签收")
	postRemovalTestPayment(t, ctx, s, f.Customer, 100)
	// Even an independent payment mentioning the deleted order must survive byte
	// for byte. Its type remains payment, regardless of its display/source code.
	var payment model.CustomerTransaction
	if err := s.CustomerTransactionModel.FindOne(ctx, bson.M{"transaction_type": code.TransactionTypePayment}).Decode(&payment); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CustomerTransactionModel.UpdateByID(ctx, payment.Id, bson.M{"$set": bson.M{"order_code": f.Order.Code, "source_code": f.Order.Code}}); err != nil {
		t.Fatal(err)
	}
	paymentBefore := lifecycleDocument(t, ctx, s.CustomerTransactionModel, payment.Id)
	preview := removalPreviewForTest(t, ctx, s, f.Order.Id)
	if preview.TransactionCount != 1 || preview.ReceivableBefore != 0 || preview.CreditAfter != 100 {
		t.Fatalf("paid-order preview=%+v", preview)
	}
	results := make(chan reviewMutationResult, 2)
	for i := 0; i < 2; i++ {
		reviewStartRemoval(ctx, s, f.Order.Id, preview.PreviewToken, results)
	}
	reviewAwaitWrite(t, ready)
	reviewAwaitWrite(t, ready)
	release()
	for i := 0; i < 2; i++ {
		result := reviewAwaitResult(t, results)
		lifecycleResponse(t, result.response, result.err, http.StatusOK)
	}
	lifecycleStock(t, ctx, s, f.Inventory.Id, 10, 0)
	reviewBalance(t, ctx, s, f.Customer.Id, 0, 100)
	lifecycleCount(t, ctx, s.CustomerTransactionModel, 1)
	lifecycleUnchanged(t, ctx, s.CustomerTransactionModel, payment.Id, paymentBefore)
	stockBefore := lifecycleDocument(t, ctx, s.InventoryModel, f.Inventory.Id)
	customerBefore := lifecycleDocument(t, ctx, s.CustomerModel, f.Customer.Id)
	for i := 0; i < 3; i++ {
		result := removeForTest(t, ctx, s, f.Order.Id, preview.PreviewToken)
		lifecycleResponse(t, result, nil, http.StatusOK)
	}
	lifecycleUnchanged(t, ctx, s.InventoryModel, f.Inventory.Id, stockBefore)
	lifecycleUnchanged(t, ctx, s.CustomerModel, f.Customer.Id, customerBefore)
}

func TestRemovalReviewSharedStockAndCustomerRequiresFreshSecondPreview(t *testing.T) {
	monitor, ready, release := reviewWriteBarrier("delete", "outbound_order", 2)
	s, ctx := newRemovalIntegrationService(t, monitor)
	t.Cleanup(release)
	f := seedRemovalIntegrationFixture(t, s, "已签收")
	second := f.Order
	second.Id, second.Code, second.TotalAmount = primitive.NewObjectID(), "O-SHARED-"+primitive.NewObjectID().Hex(), 150
	line := f.Material
	line.Id, line.OrderCode, line.Quantity = primitive.NewObjectID(), second.Code, 3
	line.Inventorys = []model.OutboundMaterialInventory{{InventoryId: f.Inventory.Id.Hex(), ShipmentQuantity: 3}}
	secondAR := model.CustomerTransaction{Id: primitive.NewObjectID(), CustomerId: f.Customer.Id.Hex(),
		TransactionType: code.TransactionTypeOutboundAR, SourceType: code.SourceTypeOutboundOrder,
		SourceId: second.Id.Hex(), SourceCode: second.Code, OrderCode: second.Code, Amount: 150}
	for _, pair := range []struct {
		collection *mongo.Collection
		document   interface{}
	}{{s.OutboundOrderModel, second}, {s.OutboundMaterialModel, line}, {s.CustomerTransactionModel, secondAR}} {
		if _, err := pair.collection.InsertOne(ctx, pair.document); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.InventoryModel.UpdateByID(ctx, f.Inventory.Id, bson.M{"$set": bson.M{"available_quantity": 5}}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CustomerModel.UpdateByID(ctx, f.Customer.Id, bson.M{"$set": bson.M{"receivable_balance": 250}}); err != nil {
		t.Fatal(err)
	}
	postRemovalTestPayment(t, ctx, s, f.Customer, 250)
	previews := []types.OutboundRemovePreview{removalPreviewForTest(t, ctx, s, f.Order.Id), removalPreviewForTest(t, ctx, s, second.Id)}
	results := make(chan reviewMutationResult, 2)
	for _, preview := range previews {
		reviewStartRemoval(ctx, s, mustRemovalObjectID(t, preview.Id), preview.PreviewToken, results)
	}
	reviewAwaitWrite(t, ready)
	reviewAwaitWrite(t, ready)
	release()
	var stale primitive.ObjectID
	for i := 0; i < 2; i++ {
		result := reviewAwaitResult(t, results)
		switch result.response.Code {
		case http.StatusOK:
		case http.StatusConflict:
			if !stale.IsZero() {
				t.Fatal("both deletions failed instead of one successful writer")
			}
			stale = result.id
		default:
			t.Fatalf("unexpected result: %+v", result.response)
		}
	}
	if stale.IsZero() {
		t.Fatal("second deletion committed using a stale shared-stock/customer preview")
	}
	if stale == second.Id {
		lifecycleStock(t, ctx, s, f.Inventory.Id, 7, 0)
		reviewBalance(t, ctx, s, f.Customer.Id, 0, 100)
	} else {
		lifecycleStock(t, ctx, s, f.Inventory.Id, 8, 0)
		reviewBalance(t, ctx, s, f.Customer.Id, 0, 150)
	}
	fresh := removalPreviewForTest(t, ctx, s, stale)
	lifecycleResponse(t, removeForTest(t, ctx, s, stale, fresh.PreviewToken), nil, http.StatusOK)
	lifecycleStock(t, ctx, s, f.Inventory.Id, 10, 0)
	reviewBalance(t, ctx, s, f.Customer.Id, 0, 250)
	lifecycleCount(t, ctx, s.CustomerTransactionModel, 1)
	for _, preview := range previews {
		lifecycleResponse(t, removeForTest(t, ctx, s, mustRemovalObjectID(t, preview.Id), preview.PreviewToken), nil, http.StatusOK)
	}
	lifecycleStock(t, ctx, s, f.Inventory.Id, 10, 0)
	reviewBalance(t, ctx, s, f.Customer.Id, 0, 250)
}

func TestRemovalReviewReceiptWinsInvalidatesDeletionAtPublicEntry(t *testing.T) {
	monitor, ready, release := reviewWriteBarrier("delete", "outbound_order", 1)
	s, ctx := newRemovalIntegrationService(t, monitor)
	t.Cleanup(release)
	f := seedRemovalIntegrationFixture(t, s, "已出库")
	postRemovalTestPayment(t, ctx, s, f.Customer, 100)
	preview := removalPreviewForTest(t, ctx, s, f.Order.Id)
	results := make(chan reviewMutationResult, 1)
	reviewStartRemoval(ctx, s, f.Order.Id, preview.PreviewToken, results)
	reviewAwaitWrite(t, ready)
	response, err := NewReceiptLogic(ctx, s).Receipt(&types.OutboundOrderReceiptRequest{Code: f.Order.Code, ReceiptTime: time.Now().Unix() - 1})
	lifecycleResponse(t, response, err, http.StatusOK)
	release()
	result := reviewAwaitResult(t, results)
	lifecycleResponse(t, result.response, result.err, http.StatusConflict)
	lifecycleStock(t, ctx, s, f.Inventory.Id, 8, 0)
	lifecycleCount(t, ctx, s.CustomerTransactionModel, 2)
	reviewBalance(t, ctx, s, f.Customer.Id, 0, 0)
	fresh := removalPreviewForTest(t, ctx, s, f.Order.Id)
	lifecycleResponse(t, removeForTest(t, ctx, s, f.Order.Id, fresh.PreviewToken), nil, http.StatusOK)
	lifecycleStock(t, ctx, s, f.Inventory.Id, 10, 0)
	reviewBalance(t, ctx, s, f.Customer.Id, 0, 100)
}

func TestRemovalReviewDeletionWinsPreventsStaleReceiptPosting(t *testing.T) {
	monitor, ready, release := reviewWriteBarrier("update", "outbound_order", 1)
	s, ctx := newRemovalIntegrationService(t, monitor)
	t.Cleanup(release)
	f := seedRemovalIntegrationFixture(t, s, "已出库")
	postRemovalTestPayment(t, ctx, s, f.Customer, 100)
	preview := removalPreviewForTest(t, ctx, s, f.Order.Id)
	results := make(chan reviewMutationResult, 1)
	go func() {
		response, err := NewReceiptLogic(ctx, s).Receipt(&types.OutboundOrderReceiptRequest{Code: f.Order.Code, ReceiptTime: time.Now().Unix() - 1})
		results <- reviewMutationResult{response: response, err: err}
	}()
	reviewAwaitWrite(t, ready)
	lifecycleResponse(t, removeForTest(t, ctx, s, f.Order.Id, preview.PreviewToken), nil, http.StatusOK)
	release()
	result := reviewAwaitResult(t, results)
	if result.response.Code == http.StatusOK {
		t.Fatal("stale receipt unexpectedly succeeded after deletion")
	}
	lifecycleStock(t, ctx, s, f.Inventory.Id, 10, 0)
	lifecycleCount(t, ctx, s.CustomerTransactionModel, 1)
	lifecycleCount(t, ctx, s.OutboundOrderModel, 0)
	lifecycleCount(t, ctx, s.OutboundMaterialModel, 0)
	reviewBalance(t, ctx, s, f.Customer.Id, 0, 100)
}

func TestRemovalReviewOldIDReplayCannotDeleteReusedOrderCode(t *testing.T) {
	s, ctx := newRemovalIntegrationService(t)
	f := seedRemovalIntegrationFixture(t, s, "已签收")
	postRemovalTestPayment(t, ctx, s, f.Customer, 100)
	preview := removalPreviewForTest(t, ctx, s, f.Order.Id)
	lifecycleResponse(t, removeForTest(t, ctx, s, f.Order.Id, preview.PreviewToken), nil, http.StatusOK)
	response, err := NewAddLogic(ctx, s).Add(lifecycleAddRequest(f, f.Order.Code))
	lifecycleResponse(t, response, err, http.StatusOK)
	var replacement model.OutboundOrder
	if err = s.OutboundOrderModel.FindOne(ctx, bson.M{"code": f.Order.Code}).Decode(&replacement); err != nil {
		t.Fatal(err)
	}
	if replacement.Id == f.Order.Id {
		t.Fatal("replacement reused the old order ID")
	}
	before := lifecycleDocument(t, ctx, s.OutboundOrderModel, replacement.Id)
	stock := lifecycleDocument(t, ctx, s.InventoryModel, f.Inventory.Id)
	customer := lifecycleDocument(t, ctx, s.CustomerModel, f.Customer.Id)
	for _, token := range []string{"", preview.PreviewToken} {
		lifecycleResponse(t, removeForTest(t, ctx, s, f.Order.Id, token), nil, http.StatusOK)
	}
	lifecycleUnchanged(t, ctx, s.OutboundOrderModel, replacement.Id, before)
	lifecycleUnchanged(t, ctx, s.InventoryModel, f.Inventory.Id, stock)
	lifecycleUnchanged(t, ctx, s.CustomerModel, f.Customer.Id, customer)
	lifecycleCount(t, ctx, s.OutboundMaterialModel, 1)
	lifecycleCount(t, ctx, s.CustomerTransactionModel, 1)
}

func TestRemovalReviewPreservesIndependentLedgerKindsWithOrderReferences(t *testing.T) {
	s, ctx := newRemovalIntegrationService(t)
	f := seedRemovalIntegrationFixture(t, s, "已签收")
	rows := []model.CustomerTransaction{
		{TransactionType: code.TransactionTypeOpeningAR, Amount: 30},
		{TransactionType: code.TransactionTypePayment, Amount: 100},
		{TransactionType: code.TransactionTypeManualAdjustment, Direction: code.TransactionDirectionIncrease, Amount: 10},
		{TransactionType: code.TransactionTypeManualAdjustment, Direction: code.TransactionDirectionDecrease, Amount: 5},
		{TransactionType: code.TransactionTypePayment, Status: code.TransactionStatusVoided, Amount: 999},
	}
	before := make(map[primitive.ObjectID]bson.M)
	for _, row := range rows {
		row.Id, row.CustomerId = primitive.NewObjectID(), f.Customer.Id.Hex()
		// Old/imported independent entries can mention both the order code and
		// ID. Their independent transaction kind must remain authoritative.
		row.OrderCode, row.SourceCode, row.SourceId = f.Order.Code, f.Order.Code, f.Order.Id.Hex()
		if _, err := s.CustomerTransactionModel.InsertOne(ctx, row); err != nil {
			t.Fatal(err)
		}
		before[row.Id] = lifecycleDocument(t, ctx, s.CustomerTransactionModel, row.Id)
	}
	other := model.CustomerTransaction{Id: primitive.NewObjectID(), CustomerId: f.Customer.Id.Hex(),
		TransactionType: code.TransactionTypeOutboundAR, SourceType: code.SourceTypeOutboundOrder,
		SourceId: primitive.NewObjectID().Hex(), OrderCode: "O-INDEPENDENT", Amount: 200}
	if _, err := s.CustomerTransactionModel.InsertOne(ctx, other); err != nil {
		t.Fatal(err)
	}
	before[other.Id] = lifecycleDocument(t, ctx, s.CustomerTransactionModel, other.Id)
	preview := removalPreviewForTest(t, ctx, s, f.Order.Id)
	if preview.TransactionCount != 1 || preview.ReceivableBefore != 235 || preview.ReceivableAfter != 135 || preview.CreditAfter != 0 {
		t.Fatalf("independent-ledger preview=%+v", preview)
	}
	lifecycleResponse(t, removeForTest(t, ctx, s, f.Order.Id, preview.PreviewToken), nil, http.StatusOK)
	lifecycleCount(t, ctx, s.CustomerTransactionModel, int64(len(before)))
	for id, document := range before {
		lifecycleUnchanged(t, ctx, s.CustomerTransactionModel, id, document)
	}
	reviewBalance(t, ctx, s, f.Customer.Id, 135, 0)
	lifecycleStock(t, ctx, s, f.Inventory.Id, 10, 0)
}
