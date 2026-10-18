package delivery

import (
	"context"
	"errors"
	"net"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"api/internal/svc"
	"api/model"
	quoteCode "api/pkg/code"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// These tests deliberately never use application configuration or ServiceContext
// initialization. Only an explicitly supplied loopback replica set is accepted.
func newDeliveryTestDB(t *testing.T) (context.Context, *svc.ServiceContext) {
	t.Helper()
	uri := os.Getenv("WMS_TEST_MONGO_URI")
	if uri == "" {
		t.Skip("WMS_TEST_MONGO_URI is not set; isolated MongoDB integration test skipped")
	}
	parsed, err := url.Parse(uri)
	if err != nil || parsed.Scheme != "mongodb" || parsed.User != nil {
		t.Fatal("integration tests require an unauthenticated mongodb:// loopback URI")
	}
	for _, address := range strings.Split(parsed.Host, ",") {
		host, _, e := net.SplitHostPort(address)
		if e != nil || (host != "localhost" && (net.ParseIP(host) == nil || !net.ParseIP(host).IsLoopback())) {
			t.Fatalf("integration tests refuse non-loopback MongoDB address %q", address)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	if err = client.Ping(ctx, readpref.Primary()); err != nil {
		_ = client.Disconnect(ctx)
		cancel()
		t.Fatal(err)
	}
	db := client.Database("wms_delete_test_delivery_" + primitive.NewObjectID().Hex())
	t.Cleanup(func() {
		cleanup, done := context.WithTimeout(context.Background(), 10*time.Second)
		defer done()
		if err := db.Drop(cleanup); err != nil {
			t.Errorf("drop isolated test database: %v", err)
		}
		_ = client.Disconnect(cleanup)
		cancel()
	})
	return ctx, &svc.ServiceContext{
		DBClient: client, MaterialModel: db.Collection("material"),
		OutboundOrderModel: db.Collection("outbound_order"), OutboundMaterialModel: db.Collection("outbound_material"),
		CustomerMaterialDeliveryModel: db.Collection("customer_material_delivery"),
		MaterialQuoteModel:            db.Collection("material_quote"), MaterialPriceModel: db.Collection("material_price"),
	}
}

func insertDeliveryTestDoc(t *testing.T, ctx context.Context, collection *mongo.Collection, doc interface{}) {
	t.Helper()
	if _, err := collection.InsertOne(ctx, doc); err != nil {
		t.Fatal(err)
	}
}

func removeDeliveryTestOrder(t *testing.T, ctx context.Context, s *svc.ServiceContext, order model.OutboundOrder, materials []model.OutboundOrderMaterial) {
	t.Helper()
	session, err := s.DBClient.StartSession()
	if err != nil {
		t.Fatal(err)
	}
	defer session.EndSession(ctx)
	_, err = session.WithTransaction(ctx, func(tx mongo.SessionContext) (interface{}, error) {
		if _, err := s.OutboundOrderModel.DeleteOne(tx, bson.M{"_id": order.Id}); err != nil {
			return nil, err
		}
		if _, err := s.OutboundMaterialModel.DeleteMany(tx, bson.M{"order_code": order.Code}); err != nil {
			return nil, err
		}
		return nil, RebuildAfterOutboundRemoval(tx, s, order, materials)
	})
	if err != nil {
		t.Fatal(err)
	}
}

func seedDeliveryTestOrder(t *testing.T, ctx context.Context, s *svc.ServiceContext, materialID primitive.ObjectID, customerID, code string, at int64, quantities ...float64) (model.OutboundOrder, []model.OutboundOrderMaterial) {
	t.Helper()
	order := model.OutboundOrder{Id: primitive.NewObjectID(), Code: code, CustomerId: customerID,
		CustomerName: customerID, TypeCode: quoteCode.OutboundTypeCodeSales, Status: "已签收", DepartureTime: at}
	insertDeliveryTestDoc(t, ctx, s.OutboundOrderModel, order)
	lines := make([]model.OutboundOrderMaterial, 0, len(quantities))
	for _, quantity := range quantities {
		line := model.OutboundOrderMaterial{Id: primitive.NewObjectID(), OrderCode: code,
			MaterialId: materialID.Hex(), Quantity: quantity, Price: 7, Name: "测试物料", Unit: "件"}
		insertDeliveryTestDoc(t, ctx, s.OutboundMaterialModel, line)
		lines = append(lines, line)
	}
	return order, lines
}

func TestRemovalRebuildsEarliestAndGlobalAcrossCustomers(t *testing.T) {
	ctx, s := newDeliveryTestDB(t)
	materialID, deliveryID, quoteID := primitive.NewObjectID(), primitive.NewObjectID(), primitive.NewObjectID()
	insertDeliveryTestDoc(t, ctx, s.MaterialModel, bson.M{"_id": materialID, "first_delivery_time": int64(10), "first_delivery_order_code": "DELETE"})
	deleted, deletedLines := seedDeliveryTestOrder(t, ctx, s, materialID, "A", "DELETE", 10, 1)
	seedDeliveryTestOrder(t, ctx, s, materialID, "A", "NEXT", 20, 2, 3)
	seedDeliveryTestOrder(t, ctx, s, materialID, "A", "LAST", 30, 4)
	seedDeliveryTestOrder(t, ctx, s, materialID, "B", "OTHER-CUSTOMER", 15, 8)
	insertDeliveryTestDoc(t, ctx, s.CustomerMaterialDeliveryModel, model.CustomerMaterialDelivery{
		Id: deliveryID, CustomerId: "A", MaterialId: materialID.Hex(), FirstDeliveryTime: 10,
		FirstDeliveryOrderCode: "DELETE", QuoteStatus: quoteCode.QuoteStatusPriced,
		LatestQuoteId: quoteID.Hex(), LatestQuoteNo: "Q1", LatestPrice: 7, SourceValid: true,
	})
	insertDeliveryTestDoc(t, ctx, s.MaterialQuoteModel, model.MaterialQuote{
		Id: quoteID, DeliveryId: deliveryID.Hex(), SourceOrderCode: "DELETE", SourceValid: true, Status: quoteCode.MaterialQuoteStatusPriced,
	})
	insertDeliveryTestDoc(t, ctx, s.MaterialPriceModel, model.MaterialPrice{
		Id: primitive.NewObjectID(), SourceQuoteId: quoteID.Hex(), SourceDeliveryId: deliveryID.Hex(), SourceValid: true,
	})
	removeDeliveryTestOrder(t, ctx, s, deleted, deletedLines)
	var projection model.CustomerMaterialDelivery
	if err := s.CustomerMaterialDeliveryModel.FindOne(ctx, bson.M{"_id": deliveryID}).Decode(&projection); err != nil {
		t.Fatal(err)
	}
	if projection.FirstDeliveryOrderCode != "NEXT" || projection.FirstDeliveryTime != 20 || projection.FirstDeliveryQuantity != 5 || projection.LastDeliveryTime != 30 || projection.DeliveryCount != 2 || !projection.SourceValid {
		t.Fatalf("unexpected remaining delivery projection: %+v", projection)
	}
	if projection.LatestQuoteId != "" || projection.LatestPrice != 0 || projection.QuoteStatus != quoteCode.QuoteStatusUnquoted {
		t.Fatalf("deleted source quote was reattached: %+v", projection)
	}
	var global deliveryProjectionVersion
	if err := s.MaterialModel.FindOne(ctx, bson.M{"_id": materialID}).Decode(&global); err != nil {
		t.Fatal(err)
	}
	if global.FirstTime != 15 || global.FirstOrderCode != "OTHER-CUSTOMER" || global.Revision != 1 {
		t.Fatalf("global first delivery ignored another customer: %+v", global)
	}
	var quote model.MaterialQuote
	if err := s.MaterialQuoteModel.FindOne(ctx, bson.M{"_id": quoteID}).Decode(&quote); err != nil {
		t.Fatal(err)
	}
	if quote.SourceValid || quote.SourceOrderCode != "DELETE" || quote.SourceInvalidReason == "" {
		t.Fatalf("quote evidence changed or remained valid: %+v", quote)
	}
	var price model.MaterialPrice
	if err := s.MaterialPriceModel.FindOne(ctx, bson.M{"source_quote_id": quoteID.Hex()}).Decode(&price); err != nil || price.SourceValid {
		t.Fatalf("source price should remain as invalid history: %+v, %v", price, err)
	}
	if referenced, err := HasOutboundQuoteReference(ctx, s, " DELETE "); err != nil || !referenced {
		t.Fatalf("deleted source order number is no longer reserved: referenced=%v, err=%v", referenced, err)
	}
}

func TestRemovalOfLastDeliveryClearsGlobalAndPreservesQuotedHistory(t *testing.T) {
	for _, quoted := range []bool{false, true} {
		t.Run(map[bool]string{false: "unquoted", true: "quoted"}[quoted], func(t *testing.T) {
			ctx, s := newDeliveryTestDB(t)
			materialID, deliveryID := primitive.NewObjectID(), primitive.NewObjectID()
			insertDeliveryTestDoc(t, ctx, s.MaterialModel, bson.M{"_id": materialID, "first_delivery_time": int64(10), "first_delivery_order_code": "LAST"})
			order, lines := seedDeliveryTestOrder(t, ctx, s, materialID, "A", "LAST", 10, 1)
			insertDeliveryTestDoc(t, ctx, s.CustomerMaterialDeliveryModel, model.CustomerMaterialDelivery{
				Id: deliveryID, CustomerId: "A", MaterialId: materialID.Hex(), SourceValid: true, QuoteStatus: quoteCode.QuoteStatusUnquoted,
			})
			if quoted {
				insertDeliveryTestDoc(t, ctx, s.MaterialQuoteModel, model.MaterialQuote{
					Id: primitive.NewObjectID(), DeliveryId: deliveryID.Hex(), SourceOrderCode: "LAST", SourceValid: true,
				})
			}
			removeDeliveryTestOrder(t, ctx, s, order, lines)
			var global deliveryProjectionVersion
			if err := s.MaterialModel.FindOne(ctx, bson.M{"_id": materialID}).Decode(&global); err != nil {
				t.Fatal(err)
			}
			if global.FirstTime != 0 || global.FirstOrderCode != "" {
				t.Fatalf("last delivery still leaves a global source: %+v", global)
			}
			var projection model.CustomerMaterialDelivery
			err := s.CustomerMaterialDeliveryModel.FindOne(ctx, bson.M{"_id": deliveryID}).Decode(&projection)
			if !quoted && err != mongo.ErrNoDocuments {
				t.Fatalf("unquoted projection should be removed, err=%v", err)
			}
			if quoted && (err != nil || projection.SourceValid || projection.SourceInvalidReason == "") {
				t.Fatalf("quoted history should remain invalid: %+v, %v", projection, err)
			}
		})
	}
}

func TestRemovalInvalidatesOnlyPricesWithoutQuoteForOriginalDeliverySource(t *testing.T) {
	ctx, s := newDeliveryTestDB(t)
	materialID, deliveryID, otherDeliveryID, survivingQuoteID := primitive.NewObjectID(), primitive.NewObjectID(), primitive.NewObjectID(), primitive.NewObjectID()
	insertDeliveryTestDoc(t, ctx, s.MaterialModel, bson.M{"_id": materialID})
	removed, lines := seedDeliveryTestOrder(t, ctx, s, materialID, "A", "FIRST", 10, 1)
	seedDeliveryTestOrder(t, ctx, s, materialID, "A", "NEXT", 20, 2)
	insertDeliveryTestDoc(t, ctx, s.CustomerMaterialDeliveryModel, model.CustomerMaterialDelivery{
		Id: deliveryID, CustomerId: "A", MaterialId: materialID.Hex(), FirstDeliveryOrderCode: "FIRST",
		FirstDeliveryTime: 10, QuoteStatus: quoteCode.QuoteStatusPriced, LatestPrice: 9, SourceValid: true,
	})
	insertDeliveryTestDoc(t, ctx, s.CustomerMaterialDeliveryModel, model.CustomerMaterialDelivery{
		Id: otherDeliveryID, CustomerId: "B", MaterialId: materialID.Hex(), FirstDeliveryOrderCode: "UNRELATED", SourceValid: true,
	})
	// There is deliberately no quote for FIRST: legacy delivery-only prices must
	// still be invalidated, while an explicit surviving quote takes precedence.
	insertDeliveryTestDoc(t, ctx, s.MaterialQuoteModel, model.MaterialQuote{
		Id: survivingQuoteID, DeliveryId: deliveryID.Hex(), SourceOrderCode: "NEXT", SourceValid: true,
	})
	legacyPriceIDs := make([]primitive.ObjectID, 0, 3)
	for _, variant := range []string{"missing", "empty", "null"} {
		id := primitive.NewObjectID()
		legacyPriceIDs = append(legacyPriceIDs, id)
		price := bson.M{"_id": id, "source_delivery_id": deliveryID.Hex(), "source_valid": true, "price": 9}
		switch variant {
		case "empty":
			price["source_quote_id"] = ""
		case "null":
			price["source_quote_id"] = nil
		}
		insertDeliveryTestDoc(t, ctx, s.MaterialPriceModel, price)
	}
	explicitPriceID, unrelatedPriceID := primitive.NewObjectID(), primitive.NewObjectID()
	insertDeliveryTestDoc(t, ctx, s.MaterialPriceModel, model.MaterialPrice{
		Id: explicitPriceID, SourceQuoteId: survivingQuoteID.Hex(), SourceDeliveryId: deliveryID.Hex(), SourceValid: true,
	})
	insertDeliveryTestDoc(t, ctx, s.MaterialPriceModel, model.MaterialPrice{
		Id: unrelatedPriceID, SourceDeliveryId: otherDeliveryID.Hex(), SourceValid: true,
	})
	removeDeliveryTestOrder(t, ctx, s, removed, lines)
	for _, id := range legacyPriceIDs {
		var price model.MaterialPrice
		if err := s.MaterialPriceModel.FindOne(ctx, bson.M{"_id": id}).Decode(&price); err != nil || price.SourceValid || price.SourceInvalidReason == "" || price.SourceDeliveryId != deliveryID.Hex() {
			t.Fatalf("legacy source price was not preserved as invalid history: %+v, err=%v", price, err)
		}
	}
	for _, id := range []primitive.ObjectID{explicitPriceID, unrelatedPriceID} {
		var price model.MaterialPrice
		if err := s.MaterialPriceModel.FindOne(ctx, bson.M{"_id": id}).Decode(&price); err != nil || !price.SourceValid || price.SourceInvalidReason != "" {
			t.Fatalf("unrelated or explicitly quoted price was invalidated: %+v, err=%v", price, err)
		}
	}
	var survivingQuote model.MaterialQuote
	if err := s.MaterialQuoteModel.FindOne(ctx, bson.M{"_id": survivingQuoteID}).Decode(&survivingQuote); err != nil || !survivingQuote.SourceValid || survivingQuote.SourceOrderCode != "NEXT" {
		t.Fatalf("surviving quote was changed: %+v, err=%v", survivingQuote, err)
	}
	var projection model.CustomerMaterialDelivery
	if err := s.CustomerMaterialDeliveryModel.FindOne(ctx, bson.M{"_id": deliveryID}).Decode(&projection); err != nil || !projection.SourceValid || projection.FirstDeliveryOrderCode != "NEXT" || projection.LatestPrice != 0 || projection.QuoteStatus != quoteCode.QuoteStatusUnquoted {
		t.Fatalf("delivery projection retained the invalid legacy price: %+v, err=%v", projection, err)
	}
}

func TestFullRebuildCannotRestoreRemovedDeliveryFromOldVersion(t *testing.T) {
	ctx, s := newDeliveryTestDB(t)
	materialID := primitive.NewObjectID()
	insertDeliveryTestDoc(t, ctx, s.MaterialModel, bson.M{"_id": materialID, "first_delivery_time": int64(10), "first_delivery_order_code": "DELETE"})
	order, lines := seedDeliveryTestOrder(t, ctx, s, materialID, "A", "DELETE", 10, 1)
	versions, err := readDeliveryProjectionVersions(ctx, s)
	if err != nil {
		t.Fatal(err)
	}
	summaries, globals := make(map[string]rebuildSummary), make(map[string]materialFirstDelivery)
	if err = accumulateDeliveryOrder(order, groupDeliveryMaterials(lines)[order.Code], summaries, globals, 10); err != nil {
		t.Fatal(err)
	}
	removeDeliveryTestOrder(t, ctx, s, order, lines)
	if err = publishRebuildSummaries(ctx, s, versions, summaries, globals, 20); err == nil {
		t.Fatal("stale full rebuild unexpectedly overwrote the committed removal")
	}
	count, err := s.CustomerMaterialDeliveryModel.CountDocuments(ctx, bson.M{})
	if err != nil || count != 0 {
		t.Fatalf("stale source was recreated: count=%d err=%v", count, err)
	}
	var global deliveryProjectionVersion
	if err = s.MaterialModel.FindOne(ctx, bson.M{"_id": materialID}).Decode(&global); err != nil || global.FirstTime != 0 || global.FirstOrderCode != "" {
		t.Fatalf("stale global source was recreated: %+v err=%v", global, err)
	}
}

func TestFullRebuildClearsOrphanedGlobalFirstDelivery(t *testing.T) {
	ctx, s := newDeliveryTestDB(t)
	id := primitive.NewObjectID()
	insertDeliveryTestDoc(t, ctx, s.MaterialModel, bson.M{"_id": id, "first_delivery_time": int64(10), "first_delivery_order_code": "MISSING"})
	if _, _, err := RebuildCustomerMaterialDeliveries(ctx, s); err != nil {
		t.Fatal(err)
	}
	var global deliveryProjectionVersion
	if err := s.MaterialModel.FindOne(ctx, bson.M{"_id": id}).Decode(&global); err != nil || global.FirstTime != 0 || global.FirstOrderCode != "" {
		t.Fatalf("orphaned global source remains: %+v err=%v", global, err)
	}
}

func TestRemovingLastDeliveryDoesNotInvalidateExplicitIndependentQuotePrice(t *testing.T) {
	ctx, s := newDeliveryTestDB(t)
	materialID, deliveryID, foreignDeliveryID := primitive.NewObjectID(), primitive.NewObjectID(), primitive.NewObjectID()
	quoteID, independentQuoteID := primitive.NewObjectID(), primitive.NewObjectID()
	insertDeliveryTestDoc(t, ctx, s.MaterialModel, bson.M{"_id": materialID})
	order, lines := seedDeliveryTestOrder(t, ctx, s, materialID, "A", "LAST", 10, 1)
	insertDeliveryTestDoc(t, ctx, s.CustomerMaterialDeliveryModel, model.CustomerMaterialDelivery{
		Id: deliveryID, CustomerId: "A", MaterialId: materialID.Hex(), FirstDeliveryOrderCode: "LAST",
		SourceValid: true, QuoteStatus: quoteCode.QuoteStatusPriced, LatestQuoteId: quoteID.Hex(),
	})
	insertDeliveryTestDoc(t, ctx, s.MaterialQuoteModel, model.MaterialQuote{
		Id: quoteID, DeliveryId: deliveryID.Hex(), SourceOrderCode: "LAST", SourceValid: true,
	})
	insertDeliveryTestDoc(t, ctx, s.MaterialQuoteModel, model.MaterialQuote{
		Id: independentQuoteID, DeliveryId: foreignDeliveryID.Hex(), SourceOrderCode: "INDEPENDENT", SourceValid: true,
	})
	priceID := primitive.NewObjectID()
	insertDeliveryTestDoc(t, ctx, s.MaterialPriceModel, model.MaterialPrice{
		Id: priceID, SourceDeliveryId: deliveryID.Hex(), SourceQuoteId: independentQuoteID.Hex(), SourceValid: true,
	})
	removeDeliveryTestOrder(t, ctx, s, order, lines)
	var price model.MaterialPrice
	if err := s.MaterialPriceModel.FindOne(ctx, bson.M{"_id": priceID}).Decode(&price); err != nil || !price.SourceValid || price.SourceInvalidReason != "" {
		t.Fatalf("independent quote price was invalidated by a stale delivery reference: %+v err=%v", price, err)
	}
}

func TestRemovalRetriesWrappedDeliveryWriteConflictWithoutDuplicateEffects(t *testing.T) {
	ctx, s := newDeliveryTestDB(t)
	materialID := primitive.NewObjectID()
	insertDeliveryTestDoc(t, ctx, s.MaterialModel, bson.M{"_id": materialID, "first_delivery_time": int64(10), "first_delivery_order_code": "DELETE"})
	order, lines := seedDeliveryTestOrder(t, ctx, s, materialID, "A", "DELETE", 10, 1)
	session, err := s.DBClient.StartSession()
	if err != nil {
		t.Fatal(err)
	}
	defer session.EndSession(ctx)
	attempts := 0
	var conflict error
	_, err = session.WithTransaction(ctx, func(tx mongo.SessionContext) (interface{}, error) {
		attempts++
		if _, err := s.OutboundOrderModel.DeleteOne(tx, bson.M{"_id": order.Id}); err != nil {
			return nil, err
		}
		if _, err := s.OutboundMaterialModel.DeleteMany(tx, bson.M{"order_code": order.Code}); err != nil {
			return nil, err
		}
		if attempts == 1 {
			// The deletion has opened its transaction snapshot. A separately
			// committed projection mutation makes the helper's revision write
			// fail with a real TransientTransactionError wrapped using %w.
			if _, err := s.MaterialModel.UpdateByID(ctx, materialID, bson.M{"$inc": bson.M{"delivery_projection_revision": int64(1)}}); err != nil {
				return nil, err
			}
		}
		err := RebuildAfterOutboundRemoval(tx, s, order, lines)
		if attempts == 1 {
			conflict = err
		}
		return nil, err
	})
	if err != nil || attempts != 2 {
		t.Fatalf("wrapped conflict was not retried exactly once: attempts=%d err=%v conflict=%v", attempts, err, conflict)
	}
	var commandError mongo.CommandError
	if !errors.As(conflict, &commandError) || !commandError.HasErrorLabel("TransientTransactionError") {
		t.Fatalf("wrapped conflict lost its MongoDB transaction label: %v", conflict)
	}
	var global deliveryProjectionVersion
	if err := s.MaterialModel.FindOne(ctx, bson.M{"_id": materialID}).Decode(&global); err != nil || global.Revision != 2 || global.FirstTime != 0 || global.FirstOrderCode != "" {
		t.Fatalf("retry duplicated projection effects or restored old source: %+v err=%v", global, err)
	}
	if count, err := s.OutboundOrderModel.CountDocuments(ctx, bson.M{"_id": order.Id}); err != nil || count != 0 {
		t.Fatalf("source order was not removed: count=%d err=%v", count, err)
	}
}

func TestRemovalQuoteOwnershipDoesNotFollowReusedHistoricalCode(t *testing.T) {
	ctx, s := newDeliveryTestDB(t)
	materialID, otherMaterialID, deliveryID, otherDeliveryID := primitive.NewObjectID(), primitive.NewObjectID(), primitive.NewObjectID(), primitive.NewObjectID()
	insertDeliveryTestDoc(t, ctx, s.MaterialModel, bson.M{"_id": materialID})
	order, lines := seedDeliveryTestOrder(t, ctx, s, materialID, "A", "REUSED", 10, 1)
	insertDeliveryTestDoc(t, ctx, s.CustomerMaterialDeliveryModel, model.CustomerMaterialDelivery{
		Id: deliveryID, CustomerId: "A", MaterialId: materialID.Hex(), FirstDeliveryOrderCode: "REUSED", SourceValid: true,
	})
	insertDeliveryTestDoc(t, ctx, s.CustomerMaterialDeliveryModel, model.CustomerMaterialDelivery{
		Id: otherDeliveryID, CustomerId: "B", MaterialId: materialID.Hex(), FirstDeliveryOrderCode: "REUSED", SourceValid: true,
	})
	cases := []struct {
		name       string
		orderID    string
		customerID string
		materialID string
		deliveryID string
		invalid    bool
	}{
		{"frozen-current-order", order.Id.Hex(), "A", materialID.Hex(), deliveryID.Hex(), true},
		{"frozen-other-order", primitive.NewObjectID().Hex(), "A", materialID.Hex(), deliveryID.Hex(), false},
		{"legacy-owned", "", "A", materialID.Hex(), deliveryID.Hex(), true},
		{"legacy-other-customer", "", "B", materialID.Hex(), deliveryID.Hex(), false},
		{"legacy-other-material", "", "A", otherMaterialID.Hex(), deliveryID.Hex(), false},
		{"legacy-delivery-only", "", "", "", deliveryID.Hex(), true},
		{"legacy-explicit-without-delivery", "", "A", materialID.Hex(), "", true},
		{"legacy-unknown", "", "", "", "", false},
		{"legacy-other-delivery", "", "A", materialID.Hex(), otherDeliveryID.Hex(), false},
	}
	quoteIDs := make([]primitive.ObjectID, 0, len(cases))
	for _, tc := range cases {
		id := primitive.NewObjectID()
		quoteIDs = append(quoteIDs, id)
		insertDeliveryTestDoc(t, ctx, s.MaterialQuoteModel, model.MaterialQuote{
			Id: id, SourceOrderId: tc.orderID, SourceOrderCode: "REUSED", CustomerId: tc.customerID,
			MaterialId: tc.materialID, DeliveryId: tc.deliveryID, SourceValid: true,
		})
		insertDeliveryTestDoc(t, ctx, s.MaterialPriceModel, model.MaterialPrice{
			Id: primitive.NewObjectID(), SourceQuoteId: id.Hex(), SourceDeliveryId: deliveryID.Hex(), SourceValid: true,
		})
	}
	otherPriceID := primitive.NewObjectID()
	insertDeliveryTestDoc(t, ctx, s.MaterialPriceModel, model.MaterialPrice{Id: otherPriceID, SourceDeliveryId: otherDeliveryID.Hex(), SourceValid: true})
	removeDeliveryTestOrder(t, ctx, s, order, lines)
	for i, tc := range cases {
		var quote model.MaterialQuote
		if err := s.MaterialQuoteModel.FindOne(ctx, bson.M{"_id": quoteIDs[i]}).Decode(&quote); err != nil || quote.SourceValid == tc.invalid || quote.SourceOrderId != tc.orderID {
			t.Errorf("%s: quote ownership was overwritten: %+v err=%v", tc.name, quote, err)
		}
		var price model.MaterialPrice
		if err := s.MaterialPriceModel.FindOne(ctx, bson.M{"source_quote_id": quoteIDs[i].Hex()}).Decode(&price); err != nil || price.SourceValid == tc.invalid {
			t.Errorf("%s: explicit quote price ownership was overwritten: %+v err=%v", tc.name, price, err)
		}
	}
	var otherPrice model.MaterialPrice
	if err := s.MaterialPriceModel.FindOne(ctx, bson.M{"_id": otherPriceID}).Decode(&otherPrice); err != nil || !otherPrice.SourceValid {
		t.Fatalf("other customer's legacy price was invalidated by reused code: %+v err=%v", otherPrice, err)
	}
}

func TestLastRemovalInvalidatesLegacyPriceBeforeDeletingUnquotedProjection(t *testing.T) {
	ctx, s := newDeliveryTestDB(t)
	materialID, deliveryID, priceID := primitive.NewObjectID(), primitive.NewObjectID(), primitive.NewObjectID()
	insertDeliveryTestDoc(t, ctx, s.MaterialModel, bson.M{"_id": materialID})
	order, lines := seedDeliveryTestOrder(t, ctx, s, materialID, "A", "LAST", 20, 1)
	insertDeliveryTestDoc(t, ctx, s.CustomerMaterialDeliveryModel, model.CustomerMaterialDelivery{
		Id: deliveryID, CustomerId: "A", MaterialId: materialID.Hex(), FirstDeliveryOrderCode: "OLD-MISSING",
		SourceValid: true, QuoteStatus: quoteCode.QuoteStatusUnquoted,
	})
	insertDeliveryTestDoc(t, ctx, s.MaterialPriceModel, model.MaterialPrice{Id: priceID, SourceDeliveryId: deliveryID.Hex(), SourceValid: true})
	removeDeliveryTestOrder(t, ctx, s, order, lines)
	if err := s.CustomerMaterialDeliveryModel.FindOne(ctx, bson.M{"_id": deliveryID}).Err(); err != mongo.ErrNoDocuments {
		t.Fatalf("unquoted stale projection should be removed: %v", err)
	}
	var price model.MaterialPrice
	if err := s.MaterialPriceModel.FindOne(ctx, bson.M{"_id": priceID}).Decode(&price); err != nil || price.SourceValid || price.SourceInvalidReason == "" {
		t.Fatalf("valid legacy price was orphaned by projection deletion: %+v err=%v", price, err)
	}
}
