package outbound

import (
	"api/internal/svc"
	"api/internal/types"
	"api/model"
	"context"
	"net/http"
	"reflect"
	"sync"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/mongo"
)

// Uses the isolated, loopback-only fixture in remove_integration_test.go.
// These cases exercise lifecycle commands against real replica-set transactions.
func TestLifecycleIntegrationAddLineFailureRollsBackHeaderAndPrice(t *testing.T) {
	for _, existingPrice := range []bool{false, true} {
		name := "new_price"
		if existingPrice {
			name = "existing_price"
		}
		t.Run(name, func(t *testing.T) {
			svcCtx, ctx := newRemovalIntegrationService(t)
			f := seedRemovalIntegrationFixture(t, svcCtx, "预发货")
			request := lifecycleAddRequest(f, "O-ADD-FAIL-"+primitive.NewObjectID().Hex())
			priceID := primitive.NewObjectID()
			var priceBefore bson.M
			if existingPrice {
				_, err := svcCtx.MaterialPriceModel.InsertOne(ctx, bson.M{
					"_id": priceID, "material": f.Material.MaterialId, "customer_id": f.Customer.Id.Hex(),
					"price": f.Material.Price, "creator_name": "原价格维护人", "created_at": int64(1),
				})
				if err != nil {
					t.Fatal(err)
				}
				priceBefore = lifecycleDocument(t, ctx, svcCtx.MaterialPriceModel, priceID)
			}
			// Reject only the new line insert, after Add has written its header and price.
			if err := svcCtx.OutboundMaterialModel.Database().RunCommand(ctx, bson.D{
				{Key: "collMod", Value: svcCtx.OutboundMaterialModel.Name()},
				{Key: "validator", Value: bson.M{"order_code": bson.M{"$ne": request.Code}}},
				{Key: "validationLevel", Value: "strict"},
				{Key: "validationAction", Value: "error"},
			}).Err(); err != nil {
				t.Fatal(err)
			}
			orderBefore := lifecycleDocument(t, ctx, svcCtx.OutboundOrderModel, f.Order.Id)
			materialBefore := lifecycleDocument(t, ctx, svcCtx.OutboundMaterialModel, f.Material.Id)
			stockBefore := lifecycleDocument(t, ctx, svcCtx.InventoryModel, f.Inventory.Id)
			response, err := NewAddLogic(ctx, svcCtx).Add(request)
			lifecycleResponse(t, response, err, http.StatusInternalServerError)
			lifecycleCount(t, ctx, svcCtx.OutboundOrderModel, 1)
			lifecycleCount(t, ctx, svcCtx.OutboundMaterialModel, 1)
			lifecycleUnchanged(t, ctx, svcCtx.OutboundOrderModel, f.Order.Id, orderBefore)
			lifecycleUnchanged(t, ctx, svcCtx.OutboundMaterialModel, f.Material.Id, materialBefore)
			lifecycleUnchanged(t, ctx, svcCtx.InventoryModel, f.Inventory.Id, stockBefore)
			if existingPrice {
				lifecycleCount(t, ctx, svcCtx.MaterialPriceModel, 1)
				lifecycleUnchanged(t, ctx, svcCtx.MaterialPriceModel, priceID, priceBefore)
			} else {
				lifecycleCount(t, ctx, svcCtx.MaterialPriceModel, 0)
			}
		})
	}
}

func TestLifecycleIntegrationAddIntermediateWritesInvisibleToDelete(t *testing.T) {
	code := "O-ADD-PAUSE-" + primitive.NewObjectID().Hex()
	insertedHeader := make(chan primitive.ObjectID, 1)
	linesReached := make(chan bool, 1)
	resume := make(chan struct{})
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(resume) }) }
	monitor := &event.CommandMonitor{Started: func(ctx context.Context, command *event.CommandStartedEvent) {
		if command.CommandName != "insert" {
			return
		}
		collection, ok := command.Command.Lookup("insert").StringValueOK()
		if !ok || (collection != "outbound_order" && collection != "outbound_material") {
			return
		}
		array, ok := command.Command.Lookup("documents").ArrayOK()
		if !ok {
			return
		}
		values, err := array.Values()
		if err != nil || len(values) == 0 {
			return
		}
		document, ok := values[0].DocumentOK()
		if !ok {
			return
		}
		if collection == "outbound_order" {
			orderCode, _ := document.Lookup("code").StringValueOK()
			if orderCode == code {
				id, _ := document.Lookup("_id").ObjectIDOK()
				insertedHeader <- id
			}
			return
		}
		orderCode, _ := document.Lookup("order_code").StringValueOK()
		if orderCode != code {
			return
		}
		autocommit, inTransaction := command.Command.Lookup("autocommit").BooleanOK()
		linesReached <- inTransaction && !autocommit
		select {
		case <-resume:
		case <-ctx.Done():
		}
	}}
	svcCtx, ctx := newRemovalIntegrationService(t, monitor)
	f := seedRemovalIntegrationFixture(t, svcCtx, "预发货")
	request := lifecycleAddRequest(f, code)
	type addResult struct {
		response *types.BaseResponse
		err      error
	}
	result := make(chan addResult, 1)
	completed := make(chan struct{})
	t.Cleanup(func() {
		release()
		select {
		case <-completed:
		case <-time.After(5 * time.Second):
			t.Error("timed out waiting for the paused Add command to stop")
		}
	})
	go func() {
		defer close(completed)
		response, err := NewAddLogic(ctx, svcCtx).Add(request)
		result <- addResult{response: response, err: err}
	}()
	select {
	case transactional := <-linesReached:
		if !transactional {
			t.Fatal("line insertion is not part of the header transaction")
		}
	case early := <-result:
		t.Fatalf("Add returned before reaching material insert: %+v / %v", early.response, early.err)
	case <-time.After(10 * time.Second):
		t.Fatal("Add did not reach material insert")
	}
	var id primitive.ObjectID
	select {
	case id = <-insertedHeader:
	default:
		t.Fatal("header insert was not observed before material insert")
	}
	// Header and price writes have completed in Add's transaction, but an independent
	// read/deletion cannot observe either until the lines and header commit together.
	if err := svcCtx.OutboundOrderModel.FindOne(ctx, bson.M{"_id": id}).Err(); err != mongo.ErrNoDocuments {
		t.Fatalf("uncommitted header visible outside Add transaction: %v", err)
	}
	lifecycleCount(t, ctx, svcCtx.OutboundOrderModel, 1)
	lifecycleCount(t, ctx, svcCtx.OutboundMaterialModel, 1)
	lifecycleCount(t, ctx, svcCtx.MaterialPriceModel, 0)
	response, err := NewRemoveLogic(ctx, svcCtx).Remove(&types.OutboundOrderRemoveRequest{Id: id.Hex()})
	lifecycleResponse(t, response, err, http.StatusOK)
	release()
	select {
	case added := <-result:
		lifecycleResponse(t, added.response, added.err, http.StatusOK)
	case <-time.After(10 * time.Second):
		t.Fatal("Add did not finish after releasing the insert barrier")
	}
	lifecycleCount(t, ctx, svcCtx.OutboundOrderModel, 2)
	lifecycleCount(t, ctx, svcCtx.OutboundMaterialModel, 2)
	lifecycleCount(t, ctx, svcCtx.MaterialPriceModel, 1)
	response, err = NewRemoveLogic(ctx, svcCtx).Remove(&types.OutboundOrderRemoveRequest{Id: id.Hex()})
	lifecycleResponse(t, response, err, http.StatusOK)
	lifecycleCount(t, ctx, svcCtx.OutboundOrderModel, 1)
	lifecycleCount(t, ctx, svcCtx.OutboundMaterialModel, 1)
}

func lifecycleAddRequest(f removalIntegrationFixture, code string) *types.OutboundOrderAddRequest {
	return &types.OutboundOrderAddRequest{
		Code: code, Type: "销售出库", CustomerId: f.Customer.Id.Hex(),
		Materials: []types.OutboundMaterial{{MaterialId: f.Material.MaterialId, Index: 1, Price: f.Material.Price, Quantity: 2}},
	}
}

func TestLifecycleIntegrationConfirmShortageRollsBackAllChanges(t *testing.T) {
	svcCtx, ctx := newRemovalIntegrationService(t)
	f := seedRemovalIntegrationFixture(t, svcCtx, "预发货")
	shortInventory := f.Inventory
	shortInventory.Id = primitive.NewObjectID()
	shortInventory.Quantity = 0
	shortInventory.AvailableQuantity = 0
	shortInventory.LockedQuantity = 0
	if _, err := svcCtx.InventoryModel.InsertOne(ctx, shortInventory); err != nil {
		t.Fatal(err)
	}
	orderBefore := lifecycleDocument(t, ctx, svcCtx.OutboundOrderModel, f.Order.Id)
	materialBefore := lifecycleDocument(t, ctx, svcCtx.OutboundMaterialModel, f.Material.Id)
	stockBefore := lifecycleDocument(t, ctx, svcCtx.InventoryModel, f.Inventory.Id)
	shortBefore := lifecycleDocument(t, ctx, svcCtx.InventoryModel, shortInventory.Id)

	response, err := NewConfirmLogic(ctx, svcCtx).Confirm(&types.OutboundOrderConfirmRequest{
		Code: f.Order.Code, ConfirmTime: time.Now().Unix() - 1,
		Materials: []types.OutboundConfirmMaterial{{MaterialId: f.Material.MaterialId, Index: 1,
			Inventorys: []types.OutboundConfirmMaterialInventory{
				{InventoryId: f.Inventory.Id.Hex(), ShipmentQuantity: 1},
				{InventoryId: shortInventory.Id.Hex(), ShipmentQuantity: 1},
			},
		}},
	})
	lifecycleResponse(t, response, err, http.StatusConflict)
	lifecycleUnchanged(t, ctx, svcCtx.OutboundOrderModel, f.Order.Id, orderBefore)
	lifecycleUnchanged(t, ctx, svcCtx.OutboundMaterialModel, f.Material.Id, materialBefore)
	lifecycleUnchanged(t, ctx, svcCtx.InventoryModel, f.Inventory.Id, stockBefore)
	lifecycleUnchanged(t, ctx, svcCtx.InventoryModel, shortInventory.Id, shortBefore)
	lifecycleCount(t, ctx, svcCtx.OutboundOrderModel, 1)
	lifecycleCount(t, ctx, svcCtx.OutboundMaterialModel, 1)
}

func TestLifecycleIntegrationConfirmPickDeleteRestoresStockOnce(t *testing.T) {
	svcCtx, ctx := newRemovalIntegrationService(t)
	f := seedRemovalIntegrationFixture(t, svcCtx, "预发货")
	stockBefore := lifecycleDocument(t, ctx, svcCtx.InventoryModel, f.Inventory.Id)
	response, err := NewConfirmLogic(ctx, svcCtx).Confirm(lifecycleConfirmRequest(f))
	lifecycleResponse(t, response, err, http.StatusOK)
	lifecycleStock(t, ctx, svcCtx, f.Inventory.Id, 10, 2)
	response, err = NewPickLogic(ctx, svcCtx).Pick(&types.OutboundOrderPickRequest{
		Code: f.Order.Code, PickingTime: time.Now().Unix() - 1,
	})
	lifecycleResponse(t, response, err, http.StatusOK)
	lifecycleStock(t, ctx, svcCtx, f.Inventory.Id, 8, 0)

	preview, err := NewRemovePreviewLogic(ctx, svcCtx).RemovePreview(&types.OutboundOrderIdRequest{Id: f.Order.Id.Hex()})
	if err != nil || preview == nil || preview.Code != http.StatusOK || !preview.Data.Allowed {
		t.Fatalf("preview after pick: response=%+v error=%v", preview, err)
	}
	request := &types.OutboundOrderRemoveRequest{Id: f.Order.Id.Hex(), PreviewToken: preview.Data.PreviewToken}
	response, err = NewRemoveLogic(ctx, svcCtx).Remove(request)
	lifecycleResponse(t, response, err, http.StatusOK)
	lifecycleStock(t, ctx, svcCtx, f.Inventory.Id, 10, 0)
	response, err = NewRemoveLogic(ctx, svcCtx).Remove(request)
	lifecycleResponse(t, response, err, http.StatusOK)
	lifecycleUnchanged(t, ctx, svcCtx.InventoryModel, f.Inventory.Id, stockBefore)
	lifecycleCount(t, ctx, svcCtx.OutboundOrderModel, 0)
	lifecycleCount(t, ctx, svcCtx.OutboundMaterialModel, 0)
}

func TestLifecycleIntegrationPickShortageRollsBackEarlierBatch(t *testing.T) {
	svcCtx, ctx := newRemovalIntegrationService(t)
	f := seedRemovalIntegrationFixture(t, svcCtx, "待拣货")
	shortInventory := f.Inventory
	shortInventory.Id = primitive.NewObjectID()
	shortInventory.LockedQuantity = 0
	if _, err := svcCtx.InventoryModel.InsertOne(ctx, shortInventory); err != nil {
		t.Fatal(err)
	}
	if _, err := svcCtx.InventoryModel.UpdateByID(ctx, f.Inventory.Id, bson.M{"$set": bson.M{"locked_quantity": 1}}); err != nil {
		t.Fatal(err)
	}
	allocations := []model.OutboundMaterialInventory{
		{InventoryId: f.Inventory.Id.Hex(), ShipmentQuantity: 1},
		{InventoryId: shortInventory.Id.Hex(), ShipmentQuantity: 1},
	}
	if _, err := svcCtx.OutboundMaterialModel.UpdateByID(ctx, f.Material.Id, bson.M{"$set": bson.M{"inventorys": allocations}}); err != nil {
		t.Fatal(err)
	}
	orderBefore := lifecycleDocument(t, ctx, svcCtx.OutboundOrderModel, f.Order.Id)
	materialBefore := lifecycleDocument(t, ctx, svcCtx.OutboundMaterialModel, f.Material.Id)
	stockBefore := lifecycleDocument(t, ctx, svcCtx.InventoryModel, f.Inventory.Id)
	shortBefore := lifecycleDocument(t, ctx, svcCtx.InventoryModel, shortInventory.Id)
	response, err := NewPickLogic(ctx, svcCtx).Pick(&types.OutboundOrderPickRequest{
		Code: f.Order.Code, PickingTime: time.Now().Unix() - 1,
	})
	lifecycleResponse(t, response, err, http.StatusConflict)
	lifecycleUnchanged(t, ctx, svcCtx.OutboundOrderModel, f.Order.Id, orderBefore)
	lifecycleUnchanged(t, ctx, svcCtx.OutboundMaterialModel, f.Material.Id, materialBefore)
	lifecycleUnchanged(t, ctx, svcCtx.InventoryModel, f.Inventory.Id, stockBefore)
	lifecycleUnchanged(t, ctx, svcCtx.InventoryModel, shortInventory.Id, shortBefore)
}

func TestLifecycleIntegrationWeighMissingMaterialRollsBackStatusAndWeight(t *testing.T) {
	svcCtx, ctx := newRemovalIntegrationService(t)
	f := seedRemovalIntegrationFixture(t, svcCtx, "已拣货")
	orderBefore := lifecycleDocument(t, ctx, svcCtx.OutboundOrderModel, f.Order.Id)
	materialBefore := lifecycleDocument(t, ctx, svcCtx.OutboundMaterialModel, f.Material.Id)
	stockBefore := lifecycleDocument(t, ctx, svcCtx.InventoryModel, f.Inventory.Id)
	response, err := NewWeighLogic(ctx, svcCtx).Weigh(&types.OutboundOrderWeighRequest{
		Code: f.Order.Code, WeighingTime: time.Now().Unix() - 1,
		Materials: []types.OutboundWeighMaterial{
			{MaterialId: f.Material.MaterialId, Weight: 3.25},
			{MaterialId: primitive.NewObjectID().Hex(), Weight: 1},
		},
	})
	lifecycleResponse(t, response, err, http.StatusConflict)
	lifecycleUnchanged(t, ctx, svcCtx.OutboundOrderModel, f.Order.Id, orderBefore)
	lifecycleUnchanged(t, ctx, svcCtx.OutboundMaterialModel, f.Material.Id, materialBefore)
	lifecycleUnchanged(t, ctx, svcCtx.InventoryModel, f.Inventory.Id, stockBefore)
}

func TestLifecycleIntegrationPackWeighPreserveStockAndRejectReplay(t *testing.T) {
	for _, weighFirst := range []bool{false, true} {
		name := "pack_then_weigh"
		if weighFirst {
			name = "weigh_then_pack"
		}
		t.Run(name, func(t *testing.T) {
			svcCtx, ctx := newRemovalIntegrationService(t)
			f := seedRemovalIntegrationFixture(t, svcCtx, "已拣货")
			stockBefore := lifecycleDocument(t, ctx, svcCtx.InventoryModel, f.Inventory.Id)
			pack := func() (*types.BaseResponse, error) {
				return NewPackLogic(ctx, svcCtx).Pack(&types.OutboundOrderPackRequest{Code: f.Order.Code, PackingTime: time.Now().Unix() - 1})
			}
			weigh := func() (*types.BaseResponse, error) {
				return NewWeighLogic(ctx, svcCtx).Weigh(&types.OutboundOrderWeighRequest{
					Code: f.Order.Code, WeighingTime: time.Now().Unix() - 1,
					Materials: []types.OutboundWeighMaterial{{MaterialId: f.Material.MaterialId, Weight: 3.25}},
				})
			}
			first, second := pack, weigh
			if weighFirst {
				first, second = weigh, pack
			}
			response, err := first()
			lifecycleResponse(t, response, err, http.StatusOK)
			response, err = second()
			lifecycleResponse(t, response, err, http.StatusOK)
			var order model.OutboundOrder
			if err := svcCtx.OutboundOrderModel.FindOne(ctx, bson.M{"_id": f.Order.Id}).Decode(&order); err != nil {
				t.Fatal(err)
			}
			if order.IsPack != 1 || order.IsWeigh != 1 {
				t.Fatalf("expected both lifecycle flags, got %+v", order)
			}
			orderBefore := lifecycleDocument(t, ctx, svcCtx.OutboundOrderModel, f.Order.Id)
			materialBefore := lifecycleDocument(t, ctx, svcCtx.OutboundMaterialModel, f.Material.Id)
			response, err = first()
			lifecycleResponse(t, response, err, http.StatusBadRequest)
			response, err = second()
			lifecycleResponse(t, response, err, http.StatusBadRequest)
			lifecycleUnchanged(t, ctx, svcCtx.OutboundOrderModel, f.Order.Id, orderBefore)
			lifecycleUnchanged(t, ctx, svcCtx.OutboundMaterialModel, f.Material.Id, materialBefore)
			lifecycleUnchanged(t, ctx, svcCtx.InventoryModel, f.Inventory.Id, stockBefore)
		})
	}
}

func TestLifecycleIntegrationReceiptFinanceFailureRollsBackOrderAndLedger(t *testing.T) {
	svcCtx, ctx := newRemovalIntegrationService(t)
	f := seedRemovalIntegrationFixture(t, svcCtx, "已出库")
	if _, err := svcCtx.CustomerModel.DeleteOne(ctx, bson.M{"_id": f.Customer.Id}); err != nil {
		t.Fatal(err)
	}
	orderBefore := lifecycleDocument(t, ctx, svcCtx.OutboundOrderModel, f.Order.Id)
	materialBefore := lifecycleDocument(t, ctx, svcCtx.OutboundMaterialModel, f.Material.Id)
	stockBefore := lifecycleDocument(t, ctx, svcCtx.InventoryModel, f.Inventory.Id)
	response, err := NewReceiptLogic(ctx, svcCtx).Receipt(&types.OutboundOrderReceiptRequest{
		Code: f.Order.Code, ReceiptTime: time.Now().Unix() - 1, Annex: []string{"test-image"},
	})
	lifecycleResponse(t, response, err, http.StatusBadRequest)
	lifecycleUnchanged(t, ctx, svcCtx.OutboundOrderModel, f.Order.Id, orderBefore)
	lifecycleUnchanged(t, ctx, svcCtx.OutboundMaterialModel, f.Material.Id, materialBefore)
	lifecycleUnchanged(t, ctx, svcCtx.InventoryModel, f.Inventory.Id, stockBefore)
	lifecycleCount(t, ctx, svcCtx.CustomerTransactionModel, 0)
}

func TestLifecycleIntegrationDeletedOrderRejectsAllStaleCommands(t *testing.T) {
	svcCtx, ctx := newRemovalIntegrationService(t)
	f := seedRemovalIntegrationFixture(t, svcCtx, "预发货")
	response, err := NewRemoveLogic(ctx, svcCtx).Remove(&types.OutboundOrderRemoveRequest{Id: f.Order.Id.Hex()})
	lifecycleResponse(t, response, err, http.StatusOK)
	stockBefore := lifecycleDocument(t, ctx, svcCtx.InventoryModel, f.Inventory.Id)
	customerBefore := lifecycleDocument(t, ctx, svcCtx.CustomerModel, f.Customer.Id)
	now := time.Now().Unix() - 1
	commands := []struct {
		name string
		call func() (*types.BaseResponse, error)
	}{
		{"confirm", func() (*types.BaseResponse, error) {
			return NewConfirmLogic(ctx, svcCtx).Confirm(lifecycleConfirmRequest(f))
		}},
		{"pick", func() (*types.BaseResponse, error) {
			return NewPickLogic(ctx, svcCtx).Pick(&types.OutboundOrderPickRequest{Code: f.Order.Code, PickingTime: now})
		}},
		{"pack", func() (*types.BaseResponse, error) {
			return NewPackLogic(ctx, svcCtx).Pack(&types.OutboundOrderPackRequest{Code: f.Order.Code, PackingTime: now})
		}},
		{"weigh", func() (*types.BaseResponse, error) {
			return NewWeighLogic(ctx, svcCtx).Weigh(&types.OutboundOrderWeighRequest{Code: f.Order.Code, WeighingTime: now, Materials: []types.OutboundWeighMaterial{{MaterialId: f.Material.MaterialId, Weight: 2}}})
		}},
		{"receipt", func() (*types.BaseResponse, error) {
			return NewReceiptLogic(ctx, svcCtx).Receipt(&types.OutboundOrderReceiptRequest{Code: f.Order.Code, ReceiptTime: now})
		}},
	}
	for _, command := range commands {
		t.Run(command.name, func(t *testing.T) {
			response, err := command.call()
			lifecycleResponse(t, response, err, http.StatusBadRequest)
			lifecycleCount(t, ctx, svcCtx.OutboundOrderModel, 0)
			lifecycleCount(t, ctx, svcCtx.OutboundMaterialModel, 0)
			lifecycleCount(t, ctx, svcCtx.CustomerTransactionModel, 0)
			lifecycleCount(t, ctx, svcCtx.InventoryModel, 1)
			lifecycleUnchanged(t, ctx, svcCtx.InventoryModel, f.Inventory.Id, stockBefore)
			lifecycleUnchanged(t, ctx, svcCtx.CustomerModel, f.Customer.Id, customerBefore)
		})
	}
}

func lifecycleConfirmRequest(f removalIntegrationFixture) *types.OutboundOrderConfirmRequest {
	return &types.OutboundOrderConfirmRequest{
		Code: f.Order.Code, ConfirmTime: time.Now().Unix() - 1,
		Materials: []types.OutboundConfirmMaterial{{MaterialId: f.Material.MaterialId, Index: 1,
			Inventorys: []types.OutboundConfirmMaterialInventory{{InventoryId: f.Inventory.Id.Hex(), ShipmentQuantity: f.Material.Quantity}},
		}},
	}
}

func lifecycleResponse(t *testing.T, response *types.BaseResponse, err error, code int) {
	t.Helper()
	if err != nil || response == nil || response.Code != code {
		t.Fatalf("expected response code %d, got response=%+v error=%v", code, response, err)
	}
}

func lifecycleDocument(t *testing.T, ctx context.Context, collection *mongo.Collection, id primitive.ObjectID) bson.M {
	t.Helper()
	var document bson.M
	if err := collection.FindOne(ctx, bson.M{"_id": id}).Decode(&document); err != nil {
		t.Fatalf("read %s/%s: %v", collection.Name(), id.Hex(), err)
	}
	return document
}

func lifecycleUnchanged(t *testing.T, ctx context.Context, collection *mongo.Collection, id primitive.ObjectID, before bson.M) {
	t.Helper()
	after := lifecycleDocument(t, ctx, collection, id)
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("%s/%s changed: before=%+v after=%+v", collection.Name(), id.Hex(), before, after)
	}
}

func lifecycleCount(t *testing.T, ctx context.Context, collection *mongo.Collection, expected int64) {
	t.Helper()
	count, err := collection.CountDocuments(ctx, bson.M{})
	if err != nil || count != expected {
		t.Fatalf("%s count: want=%d got=%d error=%v", collection.Name(), expected, count, err)
	}
}

func lifecycleStock(t *testing.T, ctx context.Context, svcCtx *svc.ServiceContext, id primitive.ObjectID, available, locked float64) {
	t.Helper()
	var inventory model.Inventory
	if err := svcCtx.InventoryModel.FindOne(ctx, bson.M{"_id": id}).Decode(&inventory); err != nil {
		t.Fatal(err)
	}
	if inventory.AvailableQuantity != available || inventory.LockedQuantity != locked || inventory.Quantity != 10 {
		t.Fatalf("stock expected available=%g locked=%g total=10, got %+v", available, locked, inventory)
	}
}
