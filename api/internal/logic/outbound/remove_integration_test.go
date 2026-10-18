package outbound

import (
	customerlogic "api/internal/logic/customer"
	customerfinance "api/internal/logic/customer/finance"
	"api/internal/svc"
	"api/internal/types"
	"api/model"
	code "api/pkg/code"
	"context"
	"errors"
	"net/http"
	"net/url"
	"os"
	"strings"
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

// These tests can only create their own disposable database on a loopback
// replica set. They never load service configuration or production credentials.
func newRemovalIntegrationService(t *testing.T, monitors ...*event.CommandMonitor) (*svc.ServiceContext, context.Context) {
	t.Helper()
	uri := os.Getenv("WMS_TEST_MONGO_URI")
	if uri == "" {
		t.Skip("set WMS_TEST_MONGO_URI to a disposable loopback MongoDB replica set")
	}
	parsed, err := url.Parse(uri)
	if err != nil || parsed.Scheme != "mongodb" || parsed.User != nil || (parsed.Hostname() != "127.0.0.1" && parsed.Hostname() != "localhost" && parsed.Hostname() != "::1") || strings.Contains(parsed.Host, ",") {
		t.Fatal("integration tests require a credential-free loopback MongoDB URI")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	ctx = context.WithValue(ctx, "uid", primitive.NewObjectID().Hex())
	ctx = context.WithValue(ctx, "name", "删除集成测试")
	databaseName := "wms_delete_test_" + primitive.NewObjectID().Hex()
	clientOptions := options.Client().ApplyURI(uri).SetAppName(databaseName).SetServerSelectionTimeout(5 * time.Second)
	if len(monitors) > 0 {
		clientOptions.SetMonitor(monitors[0])
	}
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	if err = client.Ping(ctx, nil); err != nil {
		cancel()
		_ = client.Disconnect(context.Background())
		t.Fatal(err)
	}
	db := client.Database(databaseName)
	collections := []string{"outbound_order", "outbound_material", "inventory", "customer", "customer_transaction", "customer_material_delivery", "material", "material_quote", "material_price", "inbound_receipt", "inbound_receipt_receive"}
	for _, name := range collections {
		if err = db.CreateCollection(ctx, name); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		cleanupCtx, done := context.WithTimeout(context.Background(), 10*time.Second)
		defer done()
		if e := db.Drop(cleanupCtx); e != nil {
			t.Errorf("drop isolated database: %v", e)
		}
		_ = client.Disconnect(cleanupCtx)
		cancel()
	})
	s := &svc.ServiceContext{DBClient: client,
		OutboundOrderModel: db.Collection("outbound_order"), OutboundMaterialModel: db.Collection("outbound_material"), InventoryModel: db.Collection("inventory"),
		CustomerModel: db.Collection("customer"), CustomerTransactionModel: db.Collection("customer_transaction"),
		CustomerMaterialDeliveryModel: db.Collection("customer_material_delivery"), MaterialModel: db.Collection("material"), MaterialQuoteModel: db.Collection("material_quote"), MaterialPriceModel: db.Collection("material_price"),
		InboundReceiptModel: db.Collection("inbound_receipt"), InboundReceiptReceiveModel: db.Collection("inbound_receipt_receive"),
	}
	_, err = s.CustomerTransactionModel.Indexes().CreateOne(ctx, mongo.IndexModel{Keys: bson.D{{Key: "idempotency_key", Value: 1}}, Options: options.Index().SetUnique(true).SetPartialFilterExpression(bson.M{"idempotency_key": bson.M{"$type": "string", "$gt": ""}})})
	if err != nil {
		t.Fatal(err)
	}
	return s, ctx
}

type removalIntegrationFixture struct {
	Order     model.OutboundOrder
	Customer  model.Customer
	Material  model.OutboundOrderMaterial
	Inventory model.Inventory
}

func seedRemovalIntegrationFixture(t *testing.T, s *svc.ServiceContext, status string) removalIntegrationFixture {
	t.Helper()
	snapshot := removalUnitSnapshot()
	snapshot.Order.Code = "O-TEST-" + snapshot.Order.Id.Hex()
	snapshot.Order.Status = status
	snapshot.Order.Type = "销售出库"
	snapshot.Order.TypeCode = code.OutboundTypeCodeSales
	snapshot.Order.DepartureTime = time.Now().Unix() - 100
	snapshot.Order.ReceiptTime = time.Now().Unix() - 50
	snapshot.Order.PriceStatus = code.OutboundPriceStatusFullyPriced
	snapshot.Order.ArStatus = code.OutboundARStatusPosted
	snapshot.Customer.ReceivableBalance = 0
	snapshot.Materials[0].OrderCode = snapshot.Order.Code
	snapshot.Inventories[0].WarehouseId = primitive.NewObjectID().Hex()
	snapshot.Inventories[0].WarehouseName = "隔离测试仓"
	if status == "预发货" {
		snapshot.Materials[0].Inventorys = nil
		snapshot.Inventories[0].AvailableQuantity = 10
	}
	if status == "待拣货" {
		snapshot.Inventories[0].LockedQuantity = 2
		snapshot.Inventories[0].AvailableQuantity = 10
	}
	ctx := context.Background()
	if status == "已签收" {
		snapshot.Customer.ReceivableBalance = 100
		snapshot.Transactions[0].SourceCode = snapshot.Order.Code
		snapshot.Transactions[0].OrderCode = snapshot.Order.Code
		snapshot.Transactions[0].IdempotencyKey = code.IdempotencyKey(code.TransactionTypeOutboundAR, snapshot.Order.Id.Hex())
		if _, err := s.CustomerTransactionModel.InsertOne(ctx, snapshot.Transactions[0]); err != nil {
			t.Fatal(err)
		}
	}
	for _, pair := range []struct {
		collection *mongo.Collection
		value      interface{}
	}{{s.OutboundOrderModel, snapshot.Order}, {s.OutboundMaterialModel, snapshot.Materials[0]}, {s.InventoryModel, snapshot.Inventories[0]}, {s.CustomerModel, snapshot.Customer},
		{s.MaterialModel, bson.M{"_id": mustRemovalObjectID(t, snapshot.Materials[0].MaterialId), "name": snapshot.Materials[0].Name, "unit": snapshot.Materials[0].Unit}}} {
		if _, err := pair.collection.InsertOne(ctx, pair.value); err != nil {
			t.Fatal(err)
		}
	}
	return removalIntegrationFixture{Order: snapshot.Order, Customer: *snapshot.Customer, Material: snapshot.Materials[0], Inventory: snapshot.Inventories[0]}
}

func mustRemovalObjectID(t *testing.T, value string) primitive.ObjectID {
	t.Helper()
	id, err := primitive.ObjectIDFromHex(value)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func removalPreviewForTest(t *testing.T, ctx context.Context, s *svc.ServiceContext, id primitive.ObjectID) types.OutboundRemovePreview {
	t.Helper()
	response, err := NewRemovePreviewLogic(ctx, s).RemovePreview(&types.OutboundOrderIdRequest{Id: id.Hex()})
	if err != nil || response.Code != http.StatusOK || !response.Data.Allowed {
		t.Fatalf("preview response=%+v err=%v", response, err)
	}
	return response.Data
}

func removeForTest(t *testing.T, ctx context.Context, s *svc.ServiceContext, id primitive.ObjectID, token string) *types.BaseResponse {
	t.Helper()
	response, err := NewRemoveLogic(ctx, s).Remove(&types.OutboundOrderRemoveRequest{Id: id.Hex(), PreviewToken: token})
	if err != nil {
		t.Fatal(err)
	}
	return response
}

func TestRemovalIntegrationStateInventoryMatrixAndRepeat(t *testing.T) {
	for _, status := range []string{"预发货", "待拣货", "已拣货", "已打包", "已称重", "已出库", "已签收"} {
		t.Run(status, func(t *testing.T) {
			s, ctx := newRemovalIntegrationService(t)
			f := seedRemovalIntegrationFixture(t, s, status)
			preview := removalPreviewForTest(t, ctx, s, f.Order.Id)
			if result := removeForTest(t, ctx, s, f.Order.Id, preview.PreviewToken); result.Code != http.StatusOK {
				t.Fatalf("delete: %+v", result)
			}
			if result := removeForTest(t, ctx, s, f.Order.Id, preview.PreviewToken); result.Code != http.StatusOK {
				t.Fatalf("repeat: %+v", result)
			}
			var inventory model.Inventory
			if err := s.InventoryModel.FindOne(ctx, bson.M{"_id": f.Inventory.Id}).Decode(&inventory); err != nil {
				t.Fatal(err)
			}
			if inventory.AvailableQuantity != 10 || inventory.LockedQuantity != 0 {
				t.Fatalf("stock restored more than once or wrong: %+v", inventory)
			}
			for _, collection := range []*mongo.Collection{s.OutboundOrderModel, s.OutboundMaterialModel, s.CustomerTransactionModel} {
				count, err := collection.CountDocuments(ctx, bson.M{})
				if err != nil || count != 0 {
					t.Fatalf("remaining %s=%d err=%v", collection.Name(), count, err)
				}
			}
		})
	}
}

func TestRemovalIntegrationDeletesAllAdjustmentsKeepsPaymentAndRebuildsCache(t *testing.T) {
	s, ctx := newRemovalIntegrationService(t)
	f := seedRemovalIntegrationFixture(t, s, "已签收")
	rows := []interface{}{
		model.CustomerTransaction{Id: primitive.NewObjectID(), CustomerId: f.Customer.Id.Hex(), TransactionType: code.TransactionTypeARAdjustment, SourceType: code.SourceTypeOutboundOrder, SourceId: f.Order.Id.Hex(), OrderCode: f.Order.Code, Amount: 20, Direction: code.TransactionDirectionIncrease},
		model.CustomerTransaction{Id: primitive.NewObjectID(), CustomerId: f.Customer.Id.Hex(), TransactionType: code.TransactionTypeARAdjustment, SourceType: code.SourceTypeSystemRecount, SourceId: f.Order.Id.Hex(), OrderCode: f.Order.Code, Amount: 10, Direction: code.TransactionDirectionDecrease},
		model.CustomerTransaction{Id: primitive.NewObjectID(), CustomerId: f.Customer.Id.Hex(), TransactionType: code.TransactionTypePayment, SourceType: code.SourceTypePayment, OrderCode: f.Order.Code, Amount: 110},
		model.CustomerTransaction{Id: primitive.NewObjectID(), CustomerId: f.Customer.Id.Hex(), Type: "应收账款", OrderCode: f.Order.Code, Amount: 99, Status: code.TransactionStatusVoided},
	}
	if _, err := s.CustomerTransactionModel.InsertMany(ctx, rows); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CustomerModel.UpdateByID(ctx, f.Customer.Id, bson.M{"$set": bson.M{"receivable_balance": 777}}); err != nil {
		t.Fatal(err)
	}
	preview := removalPreviewForTest(t, ctx, s, f.Order.Id)
	if preview.TransactionCount != 4 || preview.ReceivableReduction != 110 || preview.ReceivableBefore != 0 || preview.CreditAfter != 110 {
		t.Fatalf("preview=%+v", preview)
	}
	if result := removeForTest(t, ctx, s, f.Order.Id, preview.PreviewToken); result.Code != http.StatusOK {
		t.Fatalf("delete=%+v", result)
	}
	var customer model.Customer
	if err := s.CustomerModel.FindOne(ctx, bson.M{"_id": f.Customer.Id}).Decode(&customer); err != nil {
		t.Fatal(err)
	}
	if customer.ReceivableBalance != 0 || customer.CreditBalance != 110 {
		t.Fatalf("cache=%+v", customer)
	}
	var remaining []model.CustomerTransaction
	if err := readRemovalDocuments(ctx, s.CustomerTransactionModel, bson.M{}, &remaining); err != nil {
		t.Fatal(err)
	}
	if len(remaining) != 1 || remaining[0].TransactionType != code.TransactionTypePayment {
		t.Fatalf("must retain only payment, no reversal: %+v", remaining)
	}
}

func TestRemovalIntegrationLegacyOwnershipAndAmbiguity(t *testing.T) {
	for _, ambiguous := range []bool{false, true} {
		t.Run(map[bool]string{false: "legacy", true: "conflicting-source"}[ambiguous], func(t *testing.T) {
			s, ctx := newRemovalIntegrationService(t)
			f := seedRemovalIntegrationFixture(t, s, "已签收")
			update := bson.M{"source_id": "", "source_type": "", "transaction_type": "", "type": "应收账款"}
			if ambiguous {
				update["source_id"] = primitive.NewObjectID().Hex()
				update["source_type"] = code.SourceTypeOutboundOrder
			}
			if _, err := s.CustomerTransactionModel.UpdateMany(ctx, bson.M{}, bson.M{"$set": update}); err != nil {
				t.Fatal(err)
			}
			response, err := NewRemovePreviewLogic(ctx, s).RemovePreview(&types.OutboundOrderIdRequest{Id: f.Order.Id.Hex()})
			if err != nil || response.Code != http.StatusOK {
				t.Fatalf("response=%+v err=%v", response, err)
			}
			if response.Data.Allowed == ambiguous {
				t.Fatalf("unexpected allowed=%v", response.Data.Allowed)
			}
			if ambiguous {
				if result := removeForTest(t, ctx, s, f.Order.Id, "bad"); result.Code == http.StatusOK {
					t.Fatal("ambiguous delete succeeded")
				}
			} else {
				if result := removeForTest(t, ctx, s, f.Order.Id, response.Data.PreviewToken); result.Code != http.StatusOK {
					t.Fatalf("legacy delete=%+v", result)
				}
			}
		})
	}
}

func TestRemovalIntegrationRequiresPreviewAndRejectsStalePaymentSnapshot(t *testing.T) {
	s, ctx := newRemovalIntegrationService(t)
	f := seedRemovalIntegrationFixture(t, s, "已签收")
	if result := removeForTest(t, ctx, s, f.Order.Id, ""); result.Code != http.StatusBadRequest {
		t.Fatalf("missing token=%+v", result)
	}
	preview := removalPreviewForTest(t, ctx, s, f.Order.Id)
	postRemovalTestPayment(t, ctx, s, f.Customer, 30)
	if result := removeForTest(t, ctx, s, f.Order.Id, preview.PreviewToken); result.Code != http.StatusConflict {
		t.Fatalf("stale token=%+v", result)
	}
	var inventory model.Inventory
	if err := s.InventoryModel.FindOne(ctx, bson.M{"_id": f.Inventory.Id}).Decode(&inventory); err != nil {
		t.Fatal(err)
	}
	if inventory.AvailableQuantity != 8 {
		t.Fatal("stale preview changed stock")
	}
	preview = removalPreviewForTest(t, ctx, s, f.Order.Id)
	if result := removeForTest(t, ctx, s, f.Order.Id, preview.PreviewToken); result.Code != http.StatusOK {
		t.Fatalf("fresh token=%+v", result)
	}
}

func postRemovalTestPayment(t *testing.T, ctx context.Context, s *svc.ServiceContext, customer model.Customer, amount float64) {
	t.Helper()
	session, err := s.DBClient.StartSession()
	if err != nil {
		t.Fatal(err)
	}
	defer session.EndSession(ctx)
	_, err = session.WithTransaction(ctx, func(sc mongo.SessionContext) (interface{}, error) {
		return customerfinance.PostTransaction(sc, s, customerfinance.PostTransactionRequest{TransactionType: code.TransactionTypePayment, CustomerId: customer.Id.Hex(), Amount: amount, SourceType: code.SourceTypePayment, IdempotencyKey: "test-payment:" + primitive.NewObjectID().Hex()})
	}, removalTransactionOptions())
	if err != nil {
		t.Fatal(err)
	}
}

func TestRemovalIntegrationFailureRollsBackOrderInventoryAndLedger(t *testing.T) {
	s, ctx := newRemovalIntegrationService(t)
	f := seedRemovalIntegrationFixture(t, s, "已签收")
	session, err := s.DBClient.StartSession()
	if err != nil {
		t.Fatal(err)
	}
	defer session.EndSession(ctx)
	wantErr := errors.New("injected failure after all removal writes")
	_, err = session.WithTransaction(ctx, func(sc mongo.SessionContext) (interface{}, error) {
		plan, e := readRemovalPlan(sc, s, f.Order.Id)
		if e != nil {
			return nil, e
		}
		if e = applyRemovalPlan(sc, s, plan); e != nil {
			return nil, e
		}
		return nil, wantErr
	}, removalTransactionOptions())
	if !errors.Is(err, wantErr) {
		t.Fatalf("err=%v", err)
	}
	for _, collection := range []*mongo.Collection{s.OutboundOrderModel, s.OutboundMaterialModel, s.CustomerTransactionModel} {
		count, e := collection.CountDocuments(ctx, bson.M{})
		if e != nil || count != 1 {
			t.Fatalf("rollback %s=%d err=%v", collection.Name(), count, e)
		}
	}
	var inventory model.Inventory
	if err = s.InventoryModel.FindOne(ctx, bson.M{"_id": f.Inventory.Id}).Decode(&inventory); err != nil {
		t.Fatal(err)
	}
	if inventory.AvailableQuantity != 8 {
		t.Fatal("inventory rollback failed")
	}
	var customer model.Customer
	if err = s.CustomerModel.FindOne(ctx, bson.M{"_id": f.Customer.Id}).Decode(&customer); err != nil {
		t.Fatal(err)
	}
	if customer.ReceivableBalance != 100 {
		t.Fatal("customer rollback failed")
	}
}

func TestRemovalIntegrationSignedPendingAndGiftWithoutLedger(t *testing.T) {
	for _, kind := range []string{"pending", "gift"} {
		t.Run(kind, func(t *testing.T) {
			s, ctx := newRemovalIntegrationService(t)
			f := seedRemovalIntegrationFixture(t, s, "已签收")
			if _, err := s.CustomerTransactionModel.DeleteMany(ctx, bson.M{}); err != nil {
				t.Fatal(err)
			}
			set := bson.M{"ar_status": code.OutboundARStatusPending, "price_status": code.OutboundPriceStatusUnpriced}
			if kind == "gift" {
				set = bson.M{"type": "赠品出库", "type_code": code.OutboundTypeCodeGift, "ar_status": code.OutboundARStatusNotApplicable}
			}
			if _, err := s.OutboundOrderModel.UpdateByID(ctx, f.Order.Id, bson.M{"$set": set}); err != nil {
				t.Fatal(err)
			}
			preview := removalPreviewForTest(t, ctx, s, f.Order.Id)
			if preview.TransactionCount != 0 || preview.ReceivableReduction != 0 || preview.ReceivableAfter != 0 {
				t.Fatalf("unexpected finance effect: %+v", preview)
			}
			if result := removeForTest(t, ctx, s, f.Order.Id, preview.PreviewToken); result.Code != http.StatusOK {
				t.Fatalf("delete=%+v", result)
			}
		})
	}
}

func TestRemovalIntegrationMissingInventoryBlocksWithoutMutation(t *testing.T) {
	s, ctx := newRemovalIntegrationService(t)
	f := seedRemovalIntegrationFixture(t, s, "已签收")
	if _, err := s.InventoryModel.DeleteOne(ctx, bson.M{"_id": f.Inventory.Id}); err != nil {
		t.Fatal(err)
	}
	preview, err := NewRemovePreviewLogic(ctx, s).RemovePreview(&types.OutboundOrderIdRequest{Id: f.Order.Id.Hex()})
	if err != nil || preview.Code != http.StatusOK || preview.Data.Allowed || preview.Data.BlockReason == "" {
		t.Fatalf("blocked preview=%+v err=%v", preview, err)
	}
	if result := removeForTest(t, ctx, s, f.Order.Id, "not-valid"); result.Code == http.StatusOK {
		t.Fatal("deleted with missing original stock")
	}
	count, err := s.CustomerTransactionModel.CountDocuments(ctx, bson.M{})
	if err != nil || count != 1 {
		t.Fatal("blocked removal mutated ledger")
	}
}

func TestRemovalIntegrationConcurrentDeleteRestoresOnce(t *testing.T) {
	s, ctx := newRemovalIntegrationService(t)
	f := seedRemovalIntegrationFixture(t, s, "已签收")
	preview := removalPreviewForTest(t, ctx, s, f.Order.Id)
	var wg sync.WaitGroup
	responses := make(chan *types.BaseResponse, 2)
	start := make(chan struct{})
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			response, err := NewRemoveLogic(ctx, s).Remove(&types.OutboundOrderRemoveRequest{Id: f.Order.Id.Hex(), PreviewToken: preview.PreviewToken})
			if err != nil {
				responses <- &types.BaseResponse{Code: 500, Msg: err.Error()}
				return
			}
			responses <- response
		}()
	}
	close(start)
	wg.Wait()
	close(responses)
	for result := range responses {
		if result.Code != http.StatusOK {
			t.Fatalf("concurrent delete=%+v", result)
		}
	}
	var inventory model.Inventory
	if err := s.InventoryModel.FindOne(ctx, bson.M{"_id": f.Inventory.Id}).Decode(&inventory); err != nil {
		t.Fatal(err)
	}
	if inventory.AvailableQuantity != 10 {
		t.Fatalf("double stock restore: %g", inventory.AvailableQuantity)
	}
}

func TestRemovalIntegrationConcurrentPaymentRetriesWholeSnapshot(t *testing.T) {
	s, ctx := newRemovalIntegrationService(t)
	f := seedRemovalIntegrationFixture(t, s, "已签收")
	session, err := s.DBClient.StartSession()
	if err != nil {
		t.Fatal(err)
	}
	defer session.EndSession(ctx)
	attempts := 0
	_, err = session.WithTransaction(ctx, func(sc mongo.SessionContext) (interface{}, error) {
		attempts++
		plan, e := readRemovalPlan(sc, s, f.Order.Id)
		if e != nil {
			return nil, e
		}
		// A different transaction commits its payment after this transaction's
		// snapshot read, before the customer balance overwrite is attempted.
		if attempts == 1 {
			postRemovalTestPayment(t, ctx, s, f.Customer, 30)
		}
		return nil, applyRemovalPlan(sc, s, plan)
	}, removalTransactionOptions())
	if err != nil {
		t.Fatal(err)
	}
	if attempts < 2 {
		t.Fatal("expected a real MongoDB write conflict and full retry")
	}
	var customer model.Customer
	if err = s.CustomerModel.FindOne(ctx, bson.M{"_id": f.Customer.Id}).Decode(&customer); err != nil {
		t.Fatal(err)
	}
	if customer.ReceivableBalance != 0 || customer.CreditBalance != 30 {
		t.Fatalf("concurrent payment overwritten: %+v", customer)
	}
	var inventory model.Inventory
	if err = s.InventoryModel.FindOne(ctx, bson.M{"_id": f.Inventory.Id}).Decode(&inventory); err != nil {
		t.Fatal(err)
	}
	if inventory.AvailableQuantity != 10 {
		t.Fatalf("retry repeated stock restoration: %+v", inventory)
	}
}

func TestRemovalIntegrationConcurrentRevisionAndRecountInvalidatePreview(t *testing.T) {
	for _, operation := range []string{"revise", "recount"} {
		t.Run(operation, func(t *testing.T) {
			s, ctx := newRemovalIntegrationService(t)
			f := seedRemovalIntegrationFixture(t, s, "已签收")
			if operation == "recount" {
				if _, err := s.CustomerModel.UpdateByID(ctx, f.Customer.Id, bson.M{"$set": bson.M{"receivable_balance": 777}}); err != nil {
					t.Fatal(err)
				}
			}
			preview := removalPreviewForTest(t, ctx, s, f.Order.Id)
			session, err := s.DBClient.StartSession()
			if err != nil {
				t.Fatal(err)
			}
			defer session.EndSession(ctx)
			attempts := 0
			_, err = session.WithTransaction(ctx, func(sc mongo.SessionContext) (interface{}, error) {
				attempts++
				plan, e := readRemovalPlan(sc, s, f.Order.Id)
				if e != nil {
					return nil, e
				}
				if plan.Preview.PreviewToken != preview.PreviewToken {
					return nil, errRemovalPreviewChanged
				}
				if attempts == 1 {
					var response *types.BaseResponse
					if operation == "revise" {
						response, e = NewReviseLogic(ctx, s).Revise(&types.OutboundOrderReviseRequest{Code: f.Order.Code, CustomerId: f.Customer.Id.Hex(), MaterialsPrice: []types.OutboundMaterialPrice{{MaterialId: f.Material.MaterialId, Price: 75}}})
					} else {
						response, e = customerlogic.NewRecountLogic(ctx, s).Recount()
					}
					if e != nil {
						return nil, e
					}
					if response.Code != http.StatusOK {
						return nil, errors.New(response.Msg)
					}
				}
				return nil, applyRemovalPlan(sc, s, plan)
			}, removalTransactionOptions())
			if !errors.Is(err, errRemovalPreviewChanged) || attempts < 2 {
				t.Fatalf("attempts=%d err=%v", attempts, err)
			}
			var inventory model.Inventory
			if err = s.InventoryModel.FindOne(ctx, bson.M{"_id": f.Inventory.Id}).Decode(&inventory); err != nil {
				t.Fatal(err)
			}
			if inventory.AvailableQuantity != 8 {
				t.Fatal("stale snapshot changed inventory")
			}
			count, e := s.OutboundOrderModel.CountDocuments(ctx, bson.M{"_id": f.Order.Id})
			if e != nil || count != 1 {
				t.Fatal("stale snapshot removed order")
			}
			fresh := removalPreviewForTest(t, ctx, s, f.Order.Id)
			if result := removeForTest(t, ctx, s, f.Order.Id, fresh.PreviewToken); result.Code != http.StatusOK {
				t.Fatalf("fresh delete=%+v", result)
			}
		})
	}
}

func TestRemovalIntegrationRecountDoesNotRecreateDeletedReceivable(t *testing.T) {
	s, ctx := newRemovalIntegrationService(t)
	f := seedRemovalIntegrationFixture(t, s, "已签收")
	postRemovalTestPayment(t, ctx, s, f.Customer, 40)
	preview := removalPreviewForTest(t, ctx, s, f.Order.Id)
	if result := removeForTest(t, ctx, s, f.Order.Id, preview.PreviewToken); result.Code != http.StatusOK {
		t.Fatalf("delete=%+v", result)
	}
	response, err := customerlogic.NewRecountLogic(ctx, s).Recount()
	if err != nil || response.Code != http.StatusOK {
		t.Fatalf("recount=%+v err=%v", response, err)
	}
	var rows []model.CustomerTransaction
	if err = readRemovalDocuments(ctx, s.CustomerTransactionModel, bson.M{}, &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].TransactionType != code.TransactionTypePayment {
		t.Fatalf("recount recreated deleted AR: %+v", rows)
	}
	var customer model.Customer
	if err = s.CustomerModel.FindOne(ctx, bson.M{"_id": f.Customer.Id}).Decode(&customer); err != nil {
		t.Fatal(err)
	}
	if customer.ReceivableBalance != 0 || customer.CreditBalance != 40 {
		t.Fatalf("recount balance=%+v", customer)
	}
}

func TestRemovalIntegrationUnknownCommitRetriesCommitWithoutReapplying(t *testing.T) {
	var commits, orderDeletes atomic.Int64
	monitor := &event.CommandMonitor{Started: func(_ context.Context, e *event.CommandStartedEvent) {
		if e.CommandName == "commitTransaction" {
			commits.Add(1)
		}
		if e.CommandName == "delete" {
			if value, ok := e.Command.Lookup("delete").StringValueOK(); ok && value == "outbound_order" {
				orderDeletes.Add(1)
			}
		}
	}}
	s, ctx := newRemovalIntegrationService(t, monitor)
	f := seedRemovalIntegrationFixture(t, s, "已签收")
	preview := removalPreviewForTest(t, ctx, s, f.Order.Id)
	commits.Store(0)
	orderDeletes.Store(0)
	// failCommand is scoped to this random appName, so parallel test clients
	// and their commits are unaffected. No production server is reachable here.
	command := bson.D{{Key: "configureFailPoint", Value: "failCommand"}, {Key: "mode", Value: bson.M{"times": 1}}, {Key: "data", Value: bson.M{
		"failCommands": bson.A{"commitTransaction"}, "closeConnection": true, "appName": s.OutboundOrderModel.Database().Name(),
	}}}
	if err := s.DBClient.Database("admin").RunCommand(ctx, command).Err(); err != nil {
		var commandErr mongo.CommandError
		if errors.As(err, &commandErr) && (commandErr.Code == 59 || commandErr.Code == 115) {
			t.Skip("isolated MongoDB must enable test commands for commit fault injection")
		}
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.DBClient.Database("admin").RunCommand(cleanupCtx, bson.D{{Key: "configureFailPoint", Value: "failCommand"}, {Key: "mode", Value: "off"}}).Err(); err != nil {
			t.Errorf("disable isolated failpoint: %v", err)
		}
	})
	if result := removeForTest(t, ctx, s, f.Order.Id, preview.PreviewToken); result.Code != http.StatusOK {
		t.Fatalf("unknown commit result=%+v", result)
	}
	if commits.Load() < 2 || orderDeletes.Load() != 1 {
		t.Fatalf("commit attempts=%d, order delete attempts=%d", commits.Load(), orderDeletes.Load())
	}
	var inventory model.Inventory
	if err := s.InventoryModel.FindOne(ctx, bson.M{"_id": f.Inventory.Id}).Decode(&inventory); err != nil {
		t.Fatal(err)
	}
	if inventory.AvailableQuantity != 10 {
		t.Fatalf("commit retry duplicated stock: %+v", inventory)
	}
	count, err := s.CustomerTransactionModel.CountDocuments(ctx, bson.M{})
	if err != nil || count != 0 {
		t.Fatalf("remaining ledger count=%d err=%v", count, err)
	}
}

func TestRemovalIntegrationMicroPaymentsAccumulateBeforeAndAfterRemoval(t *testing.T) {
	s, ctx := newRemovalIntegrationService(t)
	f := seedRemovalIntegrationFixture(t, s, "已签收")
	postRemovalTestPayment(t, ctx, s, f.Customer, 100)
	postRemovalTestPayment(t, ctx, s, f.Customer, 0.00000075)
	postRemovalTestPayment(t, ctx, s, f.Customer, 0.00000075)
	var customer model.Customer
	if err := s.CustomerModel.FindOne(ctx, bson.M{"_id": f.Customer.Id}).Decode(&customer); err != nil {
		t.Fatal(err)
	}
	if customer.ReceivableBalance != 0 || customer.CreditBalance != 0.0000015 {
		t.Fatalf("micro payments lost by per-entry rounding: receivable=%g credit=%g", customer.ReceivableBalance, customer.CreditBalance)
	}
	preview := removalPreviewForTest(t, ctx, s, f.Order.Id)
	if result := removeForTest(t, ctx, s, f.Order.Id, preview.PreviewToken); result.Code != http.StatusOK {
		t.Fatalf("delete=%+v", result)
	}
	if err := s.CustomerModel.FindOne(ctx, bson.M{"_id": f.Customer.Id}).Decode(&customer); err != nil {
		t.Fatal(err)
	}
	if customer.ReceivableBalance != 0 || customer.CreditBalance != 100.0000015 {
		t.Fatalf("post-delete credit=%g", customer.CreditBalance)
	}
}

func TestRemovalIntegrationConcurrentMicroPaymentsInvalidatePreview(t *testing.T) {
	s, ctx := newRemovalIntegrationService(t)
	f := seedRemovalIntegrationFixture(t, s, "已签收")
	preview := removalPreviewForTest(t, ctx, s, f.Order.Id)
	session, err := s.DBClient.StartSession()
	if err != nil {
		t.Fatal(err)
	}
	defer session.EndSession(ctx)
	attempts := 0
	_, err = session.WithTransaction(ctx, func(sc mongo.SessionContext) (interface{}, error) {
		attempts++
		plan, e := readRemovalPlan(sc, s, f.Order.Id)
		if e != nil {
			return nil, e
		}
		if plan.Preview.PreviewToken != preview.PreviewToken {
			return nil, errRemovalPreviewChanged
		}
		if attempts == 1 {
			postRemovalTestPayment(t, ctx, s, f.Customer, 0.00000075)
			postRemovalTestPayment(t, ctx, s, f.Customer, 0.00000075)
		}
		return nil, applyRemovalPlan(sc, s, plan)
	}, removalTransactionOptions())
	if !errors.Is(err, errRemovalPreviewChanged) || attempts < 2 {
		t.Fatalf("micro payments bypassed customer conflict gate: attempts=%d err=%v", attempts, err)
	}
	count, e := s.OutboundOrderModel.CountDocuments(ctx, bson.M{"_id": f.Order.Id})
	if e != nil || count != 1 {
		t.Fatal("stale preview deleted the order")
	}
	var inventory model.Inventory
	if e = s.InventoryModel.FindOne(ctx, bson.M{"_id": f.Inventory.Id}).Decode(&inventory); e != nil {
		t.Fatal(e)
	}
	if inventory.AvailableQuantity != 8 {
		t.Fatalf("stale preview changed stock: %g", inventory.AvailableQuantity)
	}
	count, e = s.CustomerTransactionModel.CountDocuments(ctx, bson.M{"customer_id": f.Customer.Id.Hex(), "transaction_type": code.TransactionTypePayment})
	if e != nil || count != 2 {
		t.Fatalf("concurrent payments not retained: count=%d err=%v", count, e)
	}
	fresh := removalPreviewForTest(t, ctx, s, f.Order.Id)
	if result := removeForTest(t, ctx, s, f.Order.Id, fresh.PreviewToken); result.Code != http.StatusOK {
		t.Fatalf("fresh delete=%+v", result)
	}
	if e = s.InventoryModel.FindOne(ctx, bson.M{"_id": f.Inventory.Id}).Decode(&inventory); e != nil {
		t.Fatal(e)
	}
	if inventory.AvailableQuantity != 10 {
		t.Fatalf("fresh deletion stock=%g", inventory.AvailableQuantity)
	}
	var customer model.Customer
	if e = s.CustomerModel.FindOne(ctx, bson.M{"_id": f.Customer.Id}).Decode(&customer); e != nil {
		t.Fatal(e)
	}
	if customer.ReceivableBalance != 0 || customer.CreditBalance != 0.0000015 {
		t.Fatalf("fresh deletion dropped micro payments: receivable=%g credit=%g", customer.ReceivableBalance, customer.CreditBalance)
	}
}

func TestRemovalIntegrationRecountKeepsLegacyNullStatusPayment(t *testing.T) {
	for _, test := range []struct {
		name   string
		status interface{}
	}{
		{"null", nil}, {"missing", nil}, {"empty", ""}, {"padded-confirmed", " confirmed "}, {"whitespace", "   "},
	} {
		t.Run(test.name, func(t *testing.T) {
			s, ctx := newRemovalIntegrationService(t)
			f := seedRemovalIntegrationFixture(t, s, "已签收")
			row := bson.M{"_id": primitive.NewObjectID(), "customer_id": f.Customer.Id.Hex(), "transaction_type": code.TransactionTypePayment, "amount": 30}
			if test.name != "missing" {
				row["status"] = test.status
			}
			if _, err := s.CustomerTransactionModel.InsertOne(ctx, row); err != nil {
				t.Fatal(err)
			}
			preview := removalPreviewForTest(t, ctx, s, f.Order.Id)
			if result := removeForTest(t, ctx, s, f.Order.Id, preview.PreviewToken); result.Code != http.StatusOK {
				t.Fatalf("delete=%+v", result)
			}
			response, err := customerlogic.NewRecountLogic(ctx, s).Recount()
			if err != nil || response.Code != http.StatusOK {
				t.Fatalf("recount=%+v err=%v", response, err)
			}
			var customer model.Customer
			if err = s.CustomerModel.FindOne(ctx, bson.M{"_id": f.Customer.Id}).Decode(&customer); err != nil {
				t.Fatal(err)
			}
			if customer.ReceivableBalance != 0 || customer.CreditBalance != 30 {
				t.Fatalf("legacy payment disappeared from recount: credit=%g", customer.CreditBalance)
			}
		})
	}
}

func TestRemovalIntegrationRecountAndDeletionPreserveIndependentManualAdjustment(t *testing.T) {
	s, ctx := newRemovalIntegrationService(t)
	f := seedRemovalIntegrationFixture(t, s, "已签收")
	manual := model.CustomerTransaction{Id: primitive.NewObjectID(), CustomerId: f.Customer.Id.Hex(), TransactionType: code.TransactionTypeARAdjustment, Direction: code.TransactionDirectionDecrease, SourceType: code.SourceTypeManualAdjustment, SourceId: "manual:independent", OrderCode: f.Order.Code, Amount: 20}
	if _, err := s.CustomerTransactionModel.InsertOne(ctx, manual); err != nil {
		t.Fatal(err)
	}
	response, err := customerlogic.NewRecountLogic(ctx, s).Recount()
	if err != nil || response.Code != http.StatusOK {
		t.Fatalf("recount=%+v err=%v", response, err)
	}
	count, err := s.CustomerTransactionModel.CountDocuments(ctx, bson.M{})
	if err != nil || count != 2 {
		t.Fatalf("recount generated false AR adjustment: count=%d err=%v", count, err)
	}
	preview := removalPreviewForTest(t, ctx, s, f.Order.Id)
	if preview.TransactionCount != 1 || preview.ReceivableReduction != 100 || preview.CreditAfter != 20 {
		t.Fatalf("manual adjustment wrongly attributed: %+v", preview)
	}
	if result := removeForTest(t, ctx, s, f.Order.Id, preview.PreviewToken); result.Code != http.StatusOK {
		t.Fatalf("delete=%+v", result)
	}
	var customer model.Customer
	if err = s.CustomerModel.FindOne(ctx, bson.M{"_id": f.Customer.Id}).Decode(&customer); err != nil {
		t.Fatal(err)
	}
	if customer.CreditBalance != 20 {
		t.Fatalf("independent adjustment missing: credit=%g", customer.CreditBalance)
	}
	if err = s.CustomerTransactionModel.FindOne(ctx, bson.M{"_id": manual.Id}).Err(); err != nil {
		t.Fatalf("manual adjustment removed: %v", err)
	}
}

func TestRemovalIntegrationRevisionPaymentDeletionKeepsIndependentManualAdjustment(t *testing.T) {
	s, ctx := newRemovalIntegrationService(t)
	f := seedRemovalIntegrationFixture(t, s, "已签收")
	manual := model.CustomerTransaction{Id: primitive.NewObjectID(), CustomerId: f.Customer.Id.Hex(), TransactionType: code.TransactionTypeARAdjustment, Direction: code.TransactionDirectionDecrease, SourceType: code.SourceTypeManualAdjustment, SourceId: "manual:independent", OrderCode: f.Order.Code, Amount: 20}
	if _, err := s.CustomerTransactionModel.InsertOne(ctx, manual); err != nil {
		t.Fatal(err)
	}
	response, err := NewReviseLogic(ctx, s).Revise(&types.OutboundOrderReviseRequest{Code: f.Order.Code, CustomerId: f.Customer.Id.Hex(), MaterialsPrice: []types.OutboundMaterialPrice{{MaterialId: f.Material.MaterialId, Price: 50}}})
	if err != nil || response.Code != http.StatusOK {
		t.Fatalf("same-price revise=%+v err=%v", response, err)
	}
	count, err := s.CustomerTransactionModel.CountDocuments(ctx, bson.M{})
	if err != nil || count != 2 {
		t.Fatalf("same-price revision fabricated AR adjustment: count=%d err=%v", count, err)
	}
	response, err = NewReviseLogic(ctx, s).Revise(&types.OutboundOrderReviseRequest{Code: f.Order.Code, CustomerId: f.Customer.Id.Hex(), MaterialsPrice: []types.OutboundMaterialPrice{{MaterialId: f.Material.MaterialId, Price: 60}}})
	if err != nil || response.Code != http.StatusOK {
		t.Fatalf("price revise=%+v err=%v", response, err)
	}
	postRemovalTestPayment(t, ctx, s, f.Customer, 100)
	preview := removalPreviewForTest(t, ctx, s, f.Order.Id)
	if preview.TransactionCount != 2 || preview.ReceivableReduction != 120 || preview.ReceivableBefore != 0 || preview.CreditAfter != 120 {
		t.Fatalf("revision/payment attribution wrong: %+v", preview)
	}
	if result := removeForTest(t, ctx, s, f.Order.Id, preview.PreviewToken); result.Code != http.StatusOK {
		t.Fatalf("delete=%+v", result)
	}
	var remaining []model.CustomerTransaction
	if err = readRemovalDocuments(ctx, s.CustomerTransactionModel, bson.M{}, &remaining); err != nil {
		t.Fatal(err)
	}
	if len(remaining) != 2 {
		t.Fatalf("independent rows not preserved: %+v", remaining)
	}
	if err = s.CustomerTransactionModel.FindOne(ctx, bson.M{"_id": manual.Id}).Err(); err != nil {
		t.Fatal(err)
	}
}

func TestRemovalIntegrationRevisionRejectsConflictingAROwnershipAtomically(t *testing.T) {
	for _, kind := range []string{"other-order", "other-customer", "other-source-code"} {
		t.Run(kind, func(t *testing.T) {
			s, ctx := newRemovalIntegrationService(t)
			f := seedRemovalIntegrationFixture(t, s, "已签收")
			foreign := model.CustomerTransaction{Id: primitive.NewObjectID(), CustomerId: f.Customer.Id.Hex(), TransactionType: code.TransactionTypeOutboundAR, SourceType: code.SourceTypeOutboundOrder, SourceId: f.Order.Id.Hex(), OrderCode: f.Order.Code, SourceCode: f.Order.Code, Amount: 50}
			switch kind {
			case "other-order":
				foreign.SourceId = primitive.NewObjectID().Hex()
			case "other-customer":
				foreign.CustomerId = primitive.NewObjectID().Hex()
			case "other-source-code":
				foreign.SourceCode = "OTHER-ORDER"
			}
			if _, err := s.CustomerTransactionModel.InsertOne(ctx, foreign); err != nil {
				t.Fatal(err)
			}
			response, err := NewReviseLogic(ctx, s).Revise(&types.OutboundOrderReviseRequest{Code: f.Order.Code, CustomerId: f.Customer.Id.Hex(), MaterialsPrice: []types.OutboundMaterialPrice{{MaterialId: f.Material.MaterialId, Price: 60}}})
			if err != nil || response.Code != http.StatusBadRequest {
				t.Fatalf("conflicting ownership accepted: response=%+v err=%v", response, err)
			}
			var material model.OutboundOrderMaterial
			if err = s.OutboundMaterialModel.FindOne(ctx, bson.M{"_id": f.Material.Id}).Decode(&material); err != nil {
				t.Fatal(err)
			}
			if material.Price != 50 {
				t.Fatalf("failed revision changed price: %g", material.Price)
			}
			var order model.OutboundOrder
			if err = s.OutboundOrderModel.FindOne(ctx, bson.M{"_id": f.Order.Id}).Decode(&order); err != nil {
				t.Fatal(err)
			}
			if order.RevisionNo != 0 || order.TotalAmount != 100 {
				t.Fatalf("failed revision changed order: %+v", order)
			}
			count, err := s.CustomerTransactionModel.CountDocuments(ctx, bson.M{})
			if err != nil || count != 2 {
				t.Fatalf("failed revision changed ledger: count=%d err=%v", count, err)
			}
			count, err = s.MaterialPriceModel.CountDocuments(ctx, bson.M{})
			if err != nil || count != 0 {
				t.Fatalf("failed revision saved customer price: count=%d err=%v", count, err)
			}
			var customer model.Customer
			if err = s.CustomerModel.FindOne(ctx, bson.M{"_id": f.Customer.Id}).Decode(&customer); err != nil {
				t.Fatal(err)
			}
			if customer.ReceivableBalance != 100 || customer.CreditBalance != 0 {
				t.Fatalf("failed revision changed balance: %+v", customer)
			}
		})
	}
}

func TestRemovalIntegrationRecountRejectsSourceIDCodeConflict(t *testing.T) {
	s, ctx := newRemovalIntegrationService(t)
	f := seedRemovalIntegrationFixture(t, s, "已签收")
	other := f.Order
	other.Id = primitive.NewObjectID()
	other.Code = "OTHER-" + other.Id.Hex()
	if _, err := s.OutboundOrderModel.InsertOne(ctx, other); err != nil {
		t.Fatal(err)
	}
	otherMaterial := f.Material
	otherMaterial.Id = primitive.NewObjectID()
	otherMaterial.OrderCode = other.Code
	if _, err := s.OutboundMaterialModel.InsertOne(ctx, otherMaterial); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CustomerTransactionModel.UpdateMany(ctx, bson.M{"source_id": f.Order.Id.Hex()}, bson.M{"$set": bson.M{"source_code": other.Code}}); err != nil {
		t.Fatal(err)
	}
	response, err := customerlogic.NewRecountLogic(ctx, s).Recount()
	if err != nil || response.Code != http.StatusBadRequest {
		t.Fatalf("recount accepted contradictory source identity: response=%+v err=%v", response, err)
	}
	count, err := s.CustomerTransactionModel.CountDocuments(ctx, bson.M{})
	if err != nil || count != 1 {
		t.Fatalf("recount mutated ledger before blocking: count=%d err=%v", count, err)
	}
	var customer model.Customer
	if err = s.CustomerModel.FindOne(ctx, bson.M{"_id": f.Customer.Id}).Decode(&customer); err != nil {
		t.Fatal(err)
	}
	if customer.ReceivableBalance != 100 || customer.CreditBalance != 0 {
		t.Fatalf("recount mutated balance: %+v", customer)
	}
}

func TestRemovalIntegrationMissingPreviewIsReadOnly(t *testing.T) {
	s, ctx := newRemovalIntegrationService(t)
	f := seedRemovalIntegrationFixture(t, s, "已签收")
	postRemovalTestPayment(t, ctx, s, f.Customer, 100)

	assertPreviewError := func(id string, wantCode int) {
		t.Helper()
		response, err := NewRemovePreviewLogic(ctx, s).RemovePreview(&types.OutboundOrderIdRequest{Id: id})
		if err != nil || response == nil || response.Code != wantCode {
			t.Fatalf("preview id=%q response=%+v err=%v, want code=%d", id, response, err, wantCode)
		}
		if response.Data.Allowed || response.Data.PreviewToken != "" {
			t.Fatalf("missing or invalid order authorized deletion: %+v", response.Data)
		}
	}
	assertState := func(available, credit float64, orders, transactions int64) {
		t.Helper()
		var inventory model.Inventory
		if err := s.InventoryModel.FindOne(ctx, bson.M{"_id": f.Inventory.Id}).Decode(&inventory); err != nil {
			t.Fatal(err)
		}
		if inventory.AvailableQuantity != available || inventory.LockedQuantity != 0 {
			t.Fatalf("preview changed inventory: available=%g locked=%g", inventory.AvailableQuantity, inventory.LockedQuantity)
		}
		var customer model.Customer
		if err := s.CustomerModel.FindOne(ctx, bson.M{"_id": f.Customer.Id}).Decode(&customer); err != nil {
			t.Fatal(err)
		}
		if customer.ReceivableBalance != 0 || customer.CreditBalance != credit {
			t.Fatalf("preview changed balance: receivable=%g credit=%g", customer.ReceivableBalance, customer.CreditBalance)
		}
		for _, expected := range []struct {
			collection *mongo.Collection
			count      int64
		}{{s.OutboundOrderModel, orders}, {s.OutboundMaterialModel, orders}, {s.CustomerTransactionModel, transactions}} {
			count, err := expected.collection.CountDocuments(ctx, bson.M{})
			if err != nil || count != expected.count {
				t.Fatalf("preview changed %s: count=%d want=%d err=%v", expected.collection.Name(), count, expected.count, err)
			}
		}
	}

	assertPreviewError("not-an-object-id", http.StatusBadRequest)
	assertPreviewError(primitive.NilObjectID.Hex(), http.StatusBadRequest)
	missingID := primitive.NewObjectID().Hex()
	for i := 0; i < 2; i++ {
		assertPreviewError(missingID, http.StatusNotFound)
		assertState(8, 0, 1, 2)
	}

	preview := removalPreviewForTest(t, ctx, s, f.Order.Id)
	if result := removeForTest(t, ctx, s, f.Order.Id, preview.PreviewToken); result.Code != http.StatusOK {
		t.Fatalf("delete=%+v", result)
	}
	assertState(10, 100, 0, 1)
	for i := 0; i < 3; i++ {
		assertPreviewError(f.Order.Id.Hex(), http.StatusNotFound)
		assertState(10, 100, 0, 1)
	}
}
