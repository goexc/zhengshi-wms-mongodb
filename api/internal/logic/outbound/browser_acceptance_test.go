package outbound_test

import (
	customerhandler "api/internal/handler/customer"
	transactionhandler "api/internal/handler/customer/transaction"
	inventoryhandler "api/internal/handler/inventory"
	materialhandler "api/internal/handler/material"
	outboundhandler "api/internal/handler/outbound"
	supplierhandler "api/internal/handler/supplier"
	"api/internal/logic/customer/finance"
	"api/internal/svc"
	"api/internal/types"
	"api/model"
	"api/pkg/code"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
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
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
)

// TestOutboundBrowserAcceptance is an opt-in local browser acceptance server.
// It uses the production HTTP handlers (including parsing and validation), but
// never loads application configuration, credentials, middleware, or real data.
//
// PowerShell, from api/:
// $env:WMS_BROWSER_ACCEPTANCE='1'
// $env:WMS_TEST_MONGO_URI='mongodb://127.0.0.1:37171/?replicaSet=wms_delete_test'
// go test ./internal/logic/outbound -run '^TestOutboundBrowserAcceptance$' -v -count=1 -timeout=30m
// POST http://127.0.0.1:37272/__test/shutdown ends the test and drops its database.
func TestOutboundBrowserAcceptance(t *testing.T) {
	if os.Getenv("WMS_BROWSER_ACCEPTANCE") != "1" {
		t.Skip("set WMS_BROWSER_ACCEPTANCE=1 to start the disposable browser acceptance server")
	}
	uri := os.Getenv("WMS_TEST_MONGO_URI")
	parsed, err := url.Parse(uri)
	if err != nil || parsed.Scheme != "mongodb" || parsed.User != nil || !browserLoopbackHost(parsed.Hostname()) || strings.Contains(parsed.Host, ",") || (parsed.Path != "" && parsed.Path != "/") {
		t.Fatal("browser acceptance requires a credential-free loopback MongoDB URI without a database path")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 29*time.Minute)
	defer cancel()
	databaseName := "wms_delete_browser_" + primitive.NewObjectID().Hex()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri).SetAppName(databaseName).SetServerSelectionTimeout(5*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	db := client.Database(databaseName)
	t.Cleanup(func() {
		cleanup, done := context.WithTimeout(context.Background(), 10*time.Second)
		defer done()
		if err := db.Drop(cleanup); err != nil {
			t.Errorf("drop disposable browser database: %v", err)
		}
		_ = client.Disconnect(cleanup)
	})
	if err := client.Ping(ctx, nil); err != nil {
		t.Fatal(err)
	}
	h := &browserAcceptanceHarness{db: db, uid: primitive.NewObjectID().Hex(), next: make(map[string]browserDeleteFault), events: make(map[string][]browserDeleteEvent), shutdown: make(chan struct{})}
	h.svc = &svc.ServiceContext{
		DBClient: client, OutboundOrderModel: db.Collection("outbound_order"), OutboundMaterialModel: db.Collection("outbound_material"),
		InventoryModel: db.Collection("inventory"), CustomerModel: db.Collection("customer"), CustomerTransactionModel: db.Collection("customer_transaction"),
		CustomerMaterialDeliveryModel: db.Collection("customer_material_delivery"), MaterialModel: db.Collection("material"), MaterialQuoteModel: db.Collection("material_quote"), MaterialPriceModel: db.Collection("material_price"),
		InboundReceiptModel: db.Collection("inbound_receipt"), InboundReceiptReceiveModel: db.Collection("inbound_receipt_receive"), SupplierModel: db.Collection("supplier"), UserModel: db.Collection("user"), MaterialCategoryModel: db.Collection("material_category"),
	}
	for _, name := range browserAcceptanceCollections {
		if err := db.CreateCollection(ctx, name); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := h.svc.CustomerTransactionModel.Indexes().CreateOne(ctx, mongo.IndexModel{Keys: bson.D{{Key: "idempotency_key", Value: 1}}, Options: options.Index().SetUnique(true).SetPartialFilterExpression(bson.M{"idempotency_key": bson.M{"$type": "string", "$gt": ""}})}); err != nil {
		t.Fatal(err)
	}
	if err := h.reset(ctx); err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:37272")
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: h.handler(), ReadHeaderTimeout: 5 * time.Second}
	serveErr := make(chan error, 1)
	go func() { serveErr <- server.Serve(listener) }()
	t.Logf("BROWSER_ACCEPTANCE_READY url=http://127.0.0.1:37272 database=%s", databaseName)
	select {
	case <-h.shutdown:
	case <-ctx.Done():
		t.Error("browser acceptance timed out before shutdown")
	case err := <-serveErr:
		if !errors.Is(err, http.ErrServerClosed) {
			t.Errorf("serve browser acceptance: %v", err)
		}
	}
	stopCtx, done := context.WithTimeout(context.Background(), 10*time.Second)
	defer done()
	if err := server.Shutdown(stopCtx); err != nil {
		t.Errorf("shutdown browser acceptance: %v", err)
	}
}

var browserAcceptanceCollections = []string{"outbound_order", "outbound_material", "inventory", "customer", "customer_transaction", "customer_material_delivery", "material", "material_quote", "material_price", "material_category", "inbound_receipt", "inbound_receipt_receive", "supplier", "user"}

type browserAcceptanceCase struct {
	Case        string             `json:"case"`
	ID          primitive.ObjectID `json:"id"`
	Code        string             `json:"code"`
	CustomerID  primitive.ObjectID `json:"customer_id"`
	InventoryID primitive.ObjectID `json:"inventory_id"`
	MaterialID  primitive.ObjectID `json:"material_id"`
}

type browserDeleteFault struct {
	DelayMS         int  `json:"delay_ms"`
	FailAfterCommit bool `json:"fail_after_commit"`
}

type browserDeleteEvent struct {
	RequestNumber    int    `json:"request_number"`
	PreviewToken     string `json:"preview_token"`
	BusinessCode     int    `json:"business_code"`
	HTTPStatus       int    `json:"http_status"`
	SimulatedFailure bool   `json:"simulated_failure"`
	Finished         bool   `json:"finished"`
}

type browserAcceptanceHarness struct {
	db       *mongo.Database
	svc      *svc.ServiceContext
	uid      string
	gate     sync.RWMutex // Reset waits for all in-flight business requests.
	mu       sync.Mutex
	cases    []browserAcceptanceCase
	next     map[string]browserDeleteFault
	events   map[string][]browserDeleteEvent
	nextList int
	shutdown chan struct{}
	stopOnce sync.Once
}

func browserLoopbackHost(host string) bool {
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func browserJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func (h *browserAcceptanceHarness) handler() http.Handler {
	routes := map[string]http.HandlerFunc{
		"GET /outbound/page": outboundhandler.PageHandler(h.svc), "GET /outbound/page2": outboundhandler.Page2Handler(h.svc),
		"GET /outbound/materials": outboundhandler.MaterialsHandler(h.svc), "GET /outbound/remove-preview": outboundhandler.RemovePreviewHandler(h.svc),
		"DELETE /outbound": h.remove(outboundhandler.RemoveHandler(h.svc)), "GET /customer": customerhandler.PageHandler(h.svc),
		"GET /customer/list": customerhandler.ListHandler(h.svc), "GET /customer/transaction": transactionhandler.PageHandler(h.svc),
		"GET /inventory": inventoryhandler.PageHandler(h.svc), "GET /inventory/list": inventoryhandler.ListHandler(h.svc),
		"GET /supplier": supplierhandler.PaginateHandler(h.svc), "GET /material": materialhandler.ListHandler(h.svc),
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil || net.ParseIP(host) == nil || !net.ParseIP(host).IsLoopback() || !strings.HasPrefix(h.db.Name(), "wms_delete_browser_") {
			http.Error(w, "isolated loopback server only", http.StatusForbidden)
			return
		}
		if origin := r.Header.Get("Origin"); origin != "" {
			parsed, err := url.Parse(origin)
			if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || !browserLoopbackHost(parsed.Hostname()) {
				http.Error(w, "loopback origin required", http.StatusForbidden)
				return
			}
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,DELETE,OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization,Token")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		path := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api"), "/")
		ctx := context.WithValue(r.Context(), "uid", h.uid)
		ctx = context.WithValue(ctx, "name", "浏览器隔离验收")
		r = r.WithContext(ctx)
		if strings.HasPrefix(path, "/__test/") {
			h.control(w, r, path)
			return
		}
		h.gate.RLock()
		defer h.gate.RUnlock()
		if r.Method == http.MethodGet && (path == "/outbound/page" || path == "/outbound/page2") {
			h.mu.Lock()
			fault := h.nextList
			h.nextList = 0
			h.mu.Unlock()
			if fault != 0 {
				browserJSON(w, 200, map[string]any{"code": fault, "msg": "隔离测试列表刷新失败"})
				return
			}
		}
		if handler, ok := routes[r.Method+" "+path]; ok {
			handler(w, r)
			return
		}
		http.NotFound(w, r)
	})
}

func (h *browserAcceptanceHarness) reset(ctx context.Context) error {
	h.gate.Lock()
	defer h.gate.Unlock()
	if !strings.HasPrefix(h.db.Name(), "wms_delete_browser_") {
		return errors.New("refusing to reset a non-browser database")
	}
	for _, name := range browserAcceptanceCollections {
		if _, err := h.db.Collection(name).DeleteMany(ctx, bson.M{}); err != nil {
			return err
		}
	}
	cases := make([]browserAcceptanceCase, 0, 8)
	for index, name := range []string{"page-normal", "page-stale", "page-repeat", "page-uncertain", "page2-normal", "page2-stale", "page2-repeat", "page2-uncertain"} {
		f := browserAcceptanceCase{Case: name, ID: primitive.NewObjectID(), Code: "BROWSER-" + strings.ToUpper(name), CustomerID: primitive.NewObjectID(), InventoryID: primitive.NewObjectID(), MaterialID: primitive.NewObjectID()}
		now := time.Now().Unix() - int64(index)
		customer := model.Customer{Id: f.CustomerID, Code: "CUSTOMER-" + name, Name: "验收客户 " + name, Status: "活动", CreatorName: "浏览器隔离验收", CreatedAt: now}
		order := model.OutboundOrder{Id: f.ID, Code: f.Code, CustomerId: customer.Id.Hex(), CustomerName: customer.Name, Status: "已签收", Type: "销售出库", TypeCode: code.OutboundTypeCodeSales, PriceStatus: code.OutboundPriceStatusFullyPriced, ArStatus: code.OutboundARStatusPosted, TotalAmount: 100, DepartureTime: now - 100, ReceiptTime: now - 50, CreatedAt: now, Remark: "仅隔离浏览器验收：应收100、已回款100、原批次可用8，删除恢复2"}
		inventory := model.Inventory{Id: f.InventoryID, MaterialId: f.MaterialID.Hex(), Name: "验收物料 " + name, Model: name, Unit: "件", Type: "采购入库", Quantity: 10, AvailableQuantity: 8, WarehouseId: primitive.NewObjectID().Hex(), WarehouseName: "浏览器隔离仓", Price: 50, CreatedAt: now}
		material := model.OutboundOrderMaterial{Id: primitive.NewObjectID(), OrderCode: f.Code, MaterialId: f.MaterialID.Hex(), Name: inventory.Name, Model: inventory.Model, Unit: "件", Price: 50, Quantity: 2, Inventorys: []model.OutboundMaterialInventory{{InventoryId: f.InventoryID.Hex(), ShipmentQuantity: 2}}}
		for _, item := range []struct {
			collection *mongo.Collection
			document   any
		}{{h.svc.CustomerModel, customer}, {h.svc.OutboundOrderModel, order}, {h.svc.InventoryModel, inventory}, {h.svc.OutboundMaterialModel, material}, {h.svc.MaterialModel, model.Material{Id: f.MaterialID, Name: inventory.Name, Model: name, Unit: "件", CreatedAt: now}}} {
			if _, err := item.collection.InsertOne(ctx, item.document); err != nil {
				return err
			}
		}
		if err := h.post(ctx, f, code.TransactionTypeOutboundAR, 100); err != nil {
			return err
		}
		if err := h.post(ctx, f, code.TransactionTypePayment, 100); err != nil {
			return err
		}
		cases = append(cases, f)
	}
	h.mu.Lock()
	h.cases, h.next, h.events = cases, make(map[string]browserDeleteFault), make(map[string][]browserDeleteEvent)
	h.nextList = 0
	h.mu.Unlock()
	return nil
}

func (h *browserAcceptanceHarness) post(ctx context.Context, f browserAcceptanceCase, kind string, amount float64) error {
	session, err := h.svc.DBClient.StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)
	sourceType, sourceID := code.SourceTypePayment, primitive.NewObjectID().Hex()
	if kind == code.TransactionTypeOutboundAR {
		sourceType, sourceID = code.SourceTypeOutboundOrder, f.ID.Hex()
	}
	key := code.IdempotencyKey(kind, sourceID)
	_, err = session.WithTransaction(ctx, func(sc mongo.SessionContext) (interface{}, error) {
		return finance.PostTransaction(sc, h.svc, finance.PostTransactionRequest{TransactionType: kind, CustomerId: f.CustomerID.Hex(), CustomerName: "验收客户 " + f.Case, SourceType: sourceType, SourceId: sourceID, SourceCode: f.Code, OrderCode: f.Code, IdempotencyKey: key, Amount: amount, Creator: h.uid, CreatorName: "浏览器隔离验收", Remark: "仅隔离浏览器验收"})
	}, options.Transaction().SetReadConcern(readconcern.Snapshot()).SetWriteConcern(writeconcern.Majority()))
	return err
}

func (h *browserAcceptanceHarness) remove(actual http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Observe query data only; parsing/validation and all business mutations
		// are still performed by the real production RemoveHandler.
		id, token := r.URL.Query().Get("id"), r.URL.Query().Get("preview_token")
		h.mu.Lock()
		name := ""
		for _, f := range h.cases {
			if f.ID.Hex() == id {
				name = f.Case
				break
			}
		}
		fault := h.next[name]
		delete(h.next, name)
		index := len(h.events[name])
		h.events[name] = append(h.events[name], browserDeleteEvent{RequestNumber: index + 1, PreviewToken: token})
		h.mu.Unlock()
		if fault.DelayMS > 0 {
			select {
			case <-time.After(time.Duration(fault.DelayMS) * time.Millisecond):
			case <-r.Context().Done():
				return
			}
		}
		recorder := httptest.NewRecorder()
		actual(recorder, r)
		var result types.BaseResponse
		_ = json.Unmarshal(recorder.Body.Bytes(), &result)
		simulated := fault.FailAfterCommit && recorder.Code == http.StatusOK && result.Code == http.StatusOK
		status := recorder.Code
		if simulated {
			status = http.StatusServiceUnavailable
		}
		h.mu.Lock()
		h.events[name][index] = browserDeleteEvent{RequestNumber: index + 1, PreviewToken: token, BusinessCode: result.Code, HTTPStatus: status, SimulatedFailure: simulated, Finished: true}
		h.mu.Unlock()
		if simulated {
			browserJSON(w, status, map[string]any{"code": status, "msg": "仅验收：模拟服务响应失败"})
			return
		}
		for key, values := range recorder.Header() {
			w.Header()[key] = values
		}
		w.WriteHeader(recorder.Code)
		_, _ = w.Write(recorder.Body.Bytes())
	}
}

func (h *browserAcceptanceHarness) control(w http.ResponseWriter, r *http.Request, path string) {
	if path == "/__test/reset" && r.Method == http.MethodPost {
		if err := h.reset(r.Context()); err != nil {
			browserJSON(w, 500, map[string]any{"error": err.Error()})
			return
		}
		browserJSON(w, 200, map[string]any{"ok": true, "database": h.db.Name()})
		return
	}
	if path == "/__test/shutdown" && r.Method == http.MethodPost {
		browserJSON(w, 200, map[string]any{"ok": true})
		h.stopOnce.Do(func() { close(h.shutdown) })
		return
	}
	h.gate.RLock()
	defer h.gate.RUnlock()
	if path == "/__test/next-list" && r.Method == http.MethodPost {
		var fault struct {
			BusinessCode int `json:"business_code"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2048)).Decode(&fault); err != nil || fault.BusinessCode != 500 {
			browserJSON(w, 400, map[string]any{"error": "expected {business_code:500}"})
			return
		}
		h.mu.Lock()
		h.nextList = fault.BusinessCode
		h.mu.Unlock()
		browserJSON(w, 200, map[string]any{"ok": true})
		return
	}
	if (path == "/__test/state" || path == "/__test/health") && r.Method == http.MethodGet {
		state, err := h.state(r.Context())
		if err != nil {
			browserJSON(w, 500, map[string]any{"error": err.Error()})
			return
		}
		browserJSON(w, 200, state)
		return
	}
	parts := strings.Split(strings.TrimPrefix(path, "/__test/cases/"), "/")
	if !strings.HasPrefix(path, "/__test/cases/") || len(parts) != 2 || r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	h.mu.Lock()
	var target *browserAcceptanceCase
	for _, f := range h.cases {
		if f.Case == parts[0] {
			copy := f
			target = &copy
			break
		}
	}
	h.mu.Unlock()
	if target == nil {
		http.NotFound(w, r)
		return
	}
	switch parts[1] {
	case "payment":
		if err := h.post(r.Context(), *target, code.TransactionTypePayment, 20); err != nil {
			browserJSON(w, 500, map[string]any{"error": err.Error()})
			return
		}
		browserJSON(w, 200, map[string]any{"ok": true, "amount": 20})
	case "next-delete":
		var fault browserDeleteFault
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2048)).Decode(&fault); err != nil || fault.DelayMS < 0 || fault.DelayMS > 3000 {
			browserJSON(w, 400, map[string]any{"error": "expected {delay_ms:0..3000,fail_after_commit:boolean}"})
			return
		}
		h.mu.Lock()
		h.next[target.Case] = fault
		h.mu.Unlock()
		browserJSON(w, 200, map[string]any{"ok": true, "fault": fault})
	default:
		http.NotFound(w, r)
	}
}

func (h *browserAcceptanceHarness) state(ctx context.Context) (map[string]any, error) {
	h.mu.Lock()
	cases := append([]browserAcceptanceCase(nil), h.cases...)
	events := make(map[string][]browserDeleteEvent, len(h.events))
	for key, value := range h.events {
		events[key] = append([]browserDeleteEvent(nil), value...)
	}
	h.mu.Unlock()
	rows := make([]map[string]any, 0, len(cases))
	for _, f := range cases {
		count, err := h.svc.OutboundOrderModel.CountDocuments(ctx, bson.M{"_id": f.ID})
		if err != nil {
			return nil, err
		}
		materialCount, err := h.svc.OutboundMaterialModel.CountDocuments(ctx, bson.M{"order_code": f.Code})
		if err != nil {
			return nil, err
		}
		var inventory model.Inventory
		if err := h.svc.InventoryModel.FindOne(ctx, bson.M{"_id": f.InventoryID}).Decode(&inventory); err != nil {
			return nil, err
		}
		var customer model.Customer
		if err := h.svc.CustomerModel.FindOne(ctx, bson.M{"_id": f.CustomerID}).Decode(&customer); err != nil {
			return nil, err
		}
		cursor, err := h.svc.CustomerTransactionModel.Find(ctx, bson.M{"customer_id": f.CustomerID.Hex()}, options.Find().SetSort(bson.D{{Key: "_id", Value: 1}}))
		if err != nil {
			return nil, err
		}
		transactions := make([]model.CustomerTransaction, 0)
		if err := cursor.All(ctx, &transactions); err != nil {
			return nil, err
		}
		rows = append(rows, map[string]any{"case": f.Case, "id": f.ID.Hex(), "code": f.Code, "customer_id": f.CustomerID.Hex(), "inventory_id": f.InventoryID.Hex(), "material_id": f.MaterialID.Hex(), "order_exists": count == 1, "material_count": materialCount, "inventory": inventory, "customer": customer, "transactions": transactions, "delete_requests": len(events[f.Case]), "delete_events": events[f.Case], "initial": map[string]any{"quantity": 10, "available_quantity": 8, "locked_quantity": 0, "receivable_balance": 0, "credit_balance": 0, "outbound_ar": 100, "payment": 100}})
	}
	return map[string]any{"database": h.db.Name(), "cases": rows, "server": "127.0.0.1:37272", "purpose": fmt.Sprintf("disposable browser acceptance, %d isolated cases", len(cases))}, nil
}
