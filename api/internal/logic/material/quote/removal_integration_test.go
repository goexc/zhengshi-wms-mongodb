package quote

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	materialdelivery "api/internal/logic/material/delivery"
	"api/internal/svc"
	"api/internal/types"
	"api/model"
	quoteCode "api/pkg/code"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

func newQuoteRemovalTestDB(t *testing.T, monitor *event.CommandMonitor) (context.Context, *svc.ServiceContext) {
	t.Helper()
	uri := os.Getenv("WMS_TEST_MONGO_URI")
	if uri == "" {
		t.Skip("WMS_TEST_MONGO_URI is not set; isolated MongoDB integration test skipped")
	}
	u, err := url.Parse(uri)
	if err != nil || u.Scheme != "mongodb" || u.User != nil {
		t.Fatal("integration tests require an unauthenticated mongodb:// loopback URI")
	}
	for _, address := range strings.Split(u.Host, ",") {
		host, _, err := net.SplitHostPort(address)
		if err != nil || (host != "localhost" && (net.ParseIP(host) == nil || !net.ParseIP(host).IsLoopback())) {
			t.Fatalf("integration tests refuse non-loopback MongoDB address %q", address)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri).SetMonitor(monitor))
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	if err = client.Ping(ctx, readpref.Primary()); err != nil {
		_ = client.Disconnect(ctx)
		cancel()
		t.Fatal(err)
	}
	db := client.Database("wms_delete_test_quote_" + primitive.NewObjectID().Hex())
	t.Cleanup(func() {
		cleanup, done := context.WithTimeout(context.Background(), 10*time.Second)
		defer done()
		if err := db.Drop(cleanup); err != nil {
			t.Errorf("drop isolated quote test database: %v", err)
		}
		_ = client.Disconnect(cleanup)
		cancel()
	})
	return ctx, &svc.ServiceContext{
		DBClient: client, OutboundOrderModel: db.Collection("outbound_order"),
		OutboundMaterialModel: db.Collection("outbound_material"), MaterialModel: db.Collection("material"),
		CustomerMaterialDeliveryModel: db.Collection("customer_material_delivery"),
		MaterialQuoteModel:            db.Collection("material_quote"), MaterialPriceModel: db.Collection("material_price"),
	}
}

func insertQuoteRemovalTestDoc(t *testing.T, ctx context.Context, collection *mongo.Collection, doc interface{}) {
	t.Helper()
	if _, err := collection.InsertOne(ctx, doc); err != nil {
		t.Fatal(err)
	}
}

func TestQuoteCreationChecksSourceInsideTransaction(t *testing.T) {
	for _, sourceExists := range []bool{false, true} {
		t.Run(map[bool]string{false: "deleted-source", true: "legacy-valid-source"}[sourceExists], func(t *testing.T) {
			ctx, s := newQuoteRemovalTestDB(t, nil)
			deliveryID, materialID, orderID := primitive.NewObjectID(), primitive.NewObjectID(), primitive.NewObjectID()
			// A legacy source flag without an invalid reason remains compatible.
			insertQuoteRemovalTestDoc(t, ctx, s.CustomerMaterialDeliveryModel, model.CustomerMaterialDelivery{
				Id: deliveryID, CustomerId: "A", MaterialId: materialID.Hex(), FirstDeliveryOrderCode: "SOURCE",
			})
			if sourceExists {
				insertQuoteRemovalTestDoc(t, ctx, s.OutboundOrderModel, model.OutboundOrder{Id: orderID, Code: "SOURCE", CustomerId: "A", Status: "已签收"})
				insertQuoteRemovalTestDoc(t, ctx, s.OutboundMaterialModel, model.OutboundOrderMaterial{Id: primitive.NewObjectID(), OrderCode: "SOURCE", MaterialId: materialID.Hex()})
			}
			service := quoteService{ctx: ctx, svcCtx: s}
			resp, err := service.create(&types.MaterialQuoteSaveRequest{DeliveryId: deliveryID.Hex(), QuoteMode: quoteCode.QuoteModeSimple, SimplePrice: 7})
			if err != nil {
				t.Fatal(err)
			}
			wantStatus, wantCount := http.StatusBadRequest, int64(0)
			if sourceExists {
				wantStatus, wantCount = http.StatusOK, 1
			}
			count, err := s.MaterialQuoteModel.CountDocuments(ctx, bson.M{})
			if err != nil || resp.Code != wantStatus || count != wantCount {
				t.Fatalf("sourceExists=%v response=%+v quote count=%d err=%v", sourceExists, resp, count, err)
			}
			if sourceExists {
				var created model.MaterialQuote
				if err := s.MaterialQuoteModel.FindOne(ctx, bson.M{}).Decode(&created); err != nil || created.SourceOrderId != orderID.Hex() {
					t.Fatalf("quote did not freeze source order identity: %+v err=%v", created, err)
				}
			}
		})
	}
}

func TestPricingRollsBackWhenSourceInvalidatesAfterPrecheck(t *testing.T) {
	var armed atomic.Bool
	var s *svc.ServiceContext
	quoteID := primitive.NewObjectID()
	invalidation := make(chan error, 1)
	monitor := &event.CommandMonitor{Started: func(ctx context.Context, e *event.CommandStartedEvent) {
		if e.CommandName != "find" || e.Command.Lookup("find").StringValue() != "customer_material_delivery" || !armed.CompareAndSwap(true, false) {
			return
		}
		// validateQuoteActive already accepted its previously loaded quote, but
		// deletion invalidates that quote before pricing starts its transaction.
		_, err := s.MaterialQuoteModel.UpdateByID(context.Background(), quoteID, bson.M{"$set": bson.M{
			"source_valid": false, "source_invalid_reason": "source_outbound_order_removed",
		}})
		invalidation <- err
	}}
	ctx, db := newQuoteRemovalTestDB(t, monitor)
	s = db
	deliveryID := primitive.NewObjectID()
	insertQuoteRemovalTestDoc(t, ctx, s.CustomerMaterialDeliveryModel, model.CustomerMaterialDelivery{Id: deliveryID, SourceValid: true})
	insertQuoteRemovalTestDoc(t, ctx, s.MaterialQuoteModel, model.MaterialQuote{
		Id: quoteID, DeliveryId: deliveryID.Hex(), SourceOrderCode: "DELETED", SourceValid: true, Status: quoteCode.MaterialQuoteStatusQuoted,
	})
	armed.Store(true)
	service := quoteService{ctx: ctx, svcCtx: s}
	resp, err := service.price(&types.MaterialQuotePriceRequest{Id: quoteID.Hex(), FinalPrice: 8})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case err = <-invalidation:
		if err != nil {
			t.Fatal(err)
		}
	default:
		t.Fatal("concurrent invalidation was not exercised")
	}
	count, err := s.MaterialPriceModel.CountDocuments(ctx, bson.M{})
	if err != nil || resp.Code != http.StatusBadRequest || count != 0 {
		t.Fatalf("pricing did not roll back after invalidation: response=%+v prices=%d err=%v", resp, count, err)
	}
}

func TestRefreshQuoteSummaryDoesNotSelectInvalidHistoricalQuote(t *testing.T) {
	ctx, s := newQuoteRemovalTestDB(t, nil)
	deliveryID, validID := primitive.NewObjectID(), primitive.NewObjectID()
	insertQuoteRemovalTestDoc(t, ctx, s.CustomerMaterialDeliveryModel, model.CustomerMaterialDelivery{Id: deliveryID, SourceValid: true})
	insertQuoteRemovalTestDoc(t, ctx, s.MaterialQuoteModel, model.MaterialQuote{
		Id: validID, DeliveryId: deliveryID.Hex(), Status: quoteCode.MaterialQuoteStatusQuoted, UpdatedAt: 10,
	})
	insertQuoteRemovalTestDoc(t, ctx, s.MaterialQuoteModel, model.MaterialQuote{
		Id: primitive.NewObjectID(), DeliveryId: deliveryID.Hex(), Status: quoteCode.MaterialQuoteStatusPriced,
		UpdatedAt: 20, SourceValid: false, SourceInvalidReason: "source_outbound_order_removed",
	})
	if err := withTransaction(ctx, s, func(tx mongo.SessionContext) error { return refreshDeliveryQuoteState(tx, s, deliveryID.Hex()) }); err != nil {
		t.Fatal(err)
	}
	var projection model.CustomerMaterialDelivery
	if err := s.CustomerMaterialDeliveryModel.FindOne(ctx, bson.M{"_id": deliveryID}).Decode(&projection); err != nil || projection.LatestQuoteId != validID.Hex() {
		t.Fatalf("invalid historical quote replaced the active quote summary: %+v, err=%v", projection, err)
	}
}

func TestQuoteCreationRejectsDifferentCustomerOrMaterialOnSameCode(t *testing.T) {
	for _, mismatch := range []string{"customer", "material"} {
		t.Run(mismatch, func(t *testing.T) {
			ctx, s := newQuoteRemovalTestDB(t, nil)
			materialID, deliveryID := primitive.NewObjectID(), primitive.NewObjectID()
			insertQuoteRemovalTestDoc(t, ctx, s.CustomerMaterialDeliveryModel, model.CustomerMaterialDelivery{
				Id: deliveryID, CustomerId: "A", MaterialId: materialID.Hex(), FirstDeliveryOrderCode: "REUSED", SourceValid: true,
			})
			customerID, lineMaterialID := "A", materialID.Hex()
			if mismatch == "customer" {
				customerID = "B"
			} else {
				lineMaterialID = primitive.NewObjectID().Hex()
			}
			insertQuoteRemovalTestDoc(t, ctx, s.OutboundOrderModel, model.OutboundOrder{Id: primitive.NewObjectID(), Code: "REUSED", CustomerId: customerID, Status: "已签收"})
			insertQuoteRemovalTestDoc(t, ctx, s.OutboundMaterialModel, model.OutboundOrderMaterial{Id: primitive.NewObjectID(), OrderCode: "REUSED", MaterialId: lineMaterialID})
			service := quoteService{ctx: ctx, svcCtx: s}
			resp, err := service.create(&types.MaterialQuoteSaveRequest{DeliveryId: deliveryID.Hex(), QuoteMode: quoteCode.QuoteModeSimple, SimplePrice: 7})
			count, countErr := s.MaterialQuoteModel.CountDocuments(ctx, bson.M{})
			if err != nil || countErr != nil || resp.Code != http.StatusBadRequest || count != 0 {
				t.Fatalf("quote attached to a different source on reused code: response=%+v count=%d err=%v/%v", resp, count, err, countErr)
			}
		})
	}
}

func seedConcurrentQuoteRemoval(t *testing.T, ctx context.Context, s *svc.ServiceContext) (model.OutboundOrder, []model.OutboundOrderMaterial, primitive.ObjectID) {
	t.Helper()
	materialID, deliveryID := primitive.NewObjectID(), primitive.NewObjectID()
	order := model.OutboundOrder{Id: primitive.NewObjectID(), Code: "SOURCE", CustomerId: "A", Status: "已签收", TypeCode: quoteCode.OutboundTypeCodeSales, DepartureTime: 10}
	line := model.OutboundOrderMaterial{Id: primitive.NewObjectID(), OrderCode: order.Code, MaterialId: materialID.Hex(), Quantity: 1, Price: 7}
	insertQuoteRemovalTestDoc(t, ctx, s.MaterialModel, bson.M{"_id": materialID, "first_delivery_time": int64(10), "first_delivery_order_code": order.Code})
	insertQuoteRemovalTestDoc(t, ctx, s.OutboundOrderModel, order)
	insertQuoteRemovalTestDoc(t, ctx, s.OutboundMaterialModel, line)
	insertQuoteRemovalTestDoc(t, ctx, s.CustomerMaterialDeliveryModel, model.CustomerMaterialDelivery{
		Id: deliveryID, CustomerId: order.CustomerId, MaterialId: materialID.Hex(), FirstDeliveryOrderCode: order.Code,
		FirstDeliveryTime: 10, SourceValid: true, QuoteStatus: quoteCode.QuoteStatusUnquoted,
	})
	return order, []model.OutboundOrderMaterial{line}, deliveryID
}

func deleteConcurrentQuoteSource(ctx context.Context, s *svc.ServiceContext, order model.OutboundOrder, lines []model.OutboundOrderMaterial, beforeRebuild func() error) error {
	session, err := s.DBClient.StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)
	_, err = session.WithTransaction(ctx, func(tx mongo.SessionContext) (interface{}, error) {
		if _, err := s.OutboundOrderModel.DeleteOne(tx, bson.M{"_id": order.Id}); err != nil {
			return nil, err
		}
		if _, err := s.OutboundMaterialModel.DeleteMany(tx, bson.M{"order_code": order.Code}); err != nil {
			return nil, err
		}
		if beforeRebuild != nil {
			if err := beforeRebuild(); err != nil {
				return nil, err
			}
		}
		return nil, materialdelivery.RebuildAfterOutboundRemoval(tx, s, order, lines)
	})
	return err
}

func TestQuoteCannotCreateOrphanWhenRemovalCommitsAfterSourceRead(t *testing.T) {
	var armed atomic.Bool
	var s *svc.ServiceContext
	var outerCtx context.Context
	var order model.OutboundOrder
	var lines []model.OutboundOrderMaterial
	removed := make(chan error, 1)
	monitor := &event.CommandMonitor{Started: func(_ context.Context, e *event.CommandStartedEvent) {
		if e.CommandName != "insert" || e.Command.Lookup("insert").StringValue() != "material_quote" || !armed.CompareAndSwap(true, false) {
			return
		}
		removed <- deleteConcurrentQuoteSource(outerCtx, s, order, lines, nil)
	}}
	ctx, db := newQuoteRemovalTestDB(t, monitor)
	outerCtx, s = ctx, db
	var deliveryID primitive.ObjectID
	order, lines, deliveryID = seedConcurrentQuoteRemoval(t, ctx, s)
	armed.Store(true)
	service := quoteService{ctx: ctx, svcCtx: s}
	resp, err := service.create(&types.MaterialQuoteSaveRequest{DeliveryId: deliveryID.Hex(), QuoteMode: quoteCode.QuoteModeSimple, SimplePrice: 7})
	if err != nil || resp.Code == http.StatusOK {
		t.Fatalf("stale quote creation unexpectedly succeeded: response=%+v err=%v", resp, err)
	}
	select {
	case err := <-removed:
		if err != nil {
			t.Fatal(err)
		}
	default:
		t.Fatal("removal race did not execute")
	}
	for _, collection := range []*mongo.Collection{s.OutboundOrderModel, s.OutboundMaterialModel, s.MaterialQuoteModel, s.CustomerMaterialDeliveryModel} {
		if count, err := collection.CountDocuments(ctx, bson.M{}); err != nil || count != 0 {
			t.Errorf("%s was resurrected after removal: count=%d err=%v", collection.Name(), count, err)
		}
	}
}

func TestRemovalRetriesWhenQuoteCommitsAfterRemovalSnapshot(t *testing.T) {
	ctx, s := newQuoteRemovalTestDB(t, nil)
	order, lines, deliveryID := seedConcurrentQuoteRemoval(t, ctx, s)
	attempts := 0
	var quoteResponse *types.MaterialQuoteResponse
	err := deleteConcurrentQuoteSource(ctx, s, order, lines, func() error {
		attempts++
		if attempts != 1 {
			return nil
		}
		service := quoteService{ctx: ctx, svcCtx: s}
		var err error
		quoteResponse, err = service.create(&types.MaterialQuoteSaveRequest{DeliveryId: deliveryID.Hex(), QuoteMode: quoteCode.QuoteModeSimple, SimplePrice: 7})
		return err
	})
	if err != nil || quoteResponse == nil || quoteResponse.Code != http.StatusOK || attempts != 2 {
		t.Fatalf("removal did not retry around newly committed quote: attempts=%d quote=%+v err=%v", attempts, quoteResponse, err)
	}
	var quote model.MaterialQuote
	if err := s.MaterialQuoteModel.FindOne(ctx, bson.M{}).Decode(&quote); err != nil || quote.SourceValid || quote.SourceOrderId != order.Id.Hex() {
		t.Fatalf("committed quote escaped removal invalidation: %+v err=%v", quote, err)
	}
	var projection model.CustomerMaterialDelivery
	if err := s.CustomerMaterialDeliveryModel.FindOne(ctx, bson.M{"_id": deliveryID}).Decode(&projection); err != nil || projection.SourceValid {
		t.Fatalf("quote history was deleted or delivery resurrected: %+v err=%v", projection, err)
	}
}
