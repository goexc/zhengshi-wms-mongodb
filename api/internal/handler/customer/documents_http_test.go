package customer_test

import (
	acceptancehandler "api/internal/handler/customer/acceptance"
	invoicehandler "api/internal/handler/customer/invoice"
	"api/internal/svc"
	"api/internal/types"
	"api/model"
	code "api/pkg/code"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestDocumentHandlersConsumeZeroTaxAndOptionalRedemptionAttachments(t *testing.T) {
	uri := os.Getenv("WMS_TEST_MONGO_URI")
	if uri == "" {
		t.Skip("set WMS_TEST_MONGO_URI to disposable loopback replica set")
	}
	u, err := url.Parse(uri)
	if err != nil || u.Scheme != "mongodb" || u.User != nil || strings.Contains(u.Host, ",") || (u.Hostname() != "127.0.0.1" && u.Hostname() != "localhost" && u.Hostname() != "::1") {
		t.Fatal("only credential-free loopback MongoDB is allowed")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	ctx = context.WithValue(ctx, "uid", primitive.NewObjectID().Hex())
	ctx = context.WithValue(ctx, "name", "HTTP财务测试")
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	db := client.Database("wms_documents_http_test_" + primitive.NewObjectID().Hex())
	t.Cleanup(func() {
		cleanup, done := context.WithTimeout(context.Background(), 10*time.Second)
		defer done()
		if err := db.Drop(cleanup); err != nil {
			t.Error(err)
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
	_, err = s.CustomerTransactionModel.Indexes().CreateOne(ctx, mongo.IndexModel{Keys: bson.D{{Key: "idempotency_key", Value: 1}}, Options: options.Index().SetUnique(true)})
	if err != nil {
		t.Fatal(err)
	}
	customer := model.Customer{Id: primitive.NewObjectID(), Name: "HTTP客户", Status: "活动", ReceivableBalance: 1000}
	if _, err = s.CustomerModel.InsertOne(ctx, customer); err != nil {
		t.Fatal(err)
	}
	if _, err = s.CustomerTransactionModel.InsertOne(ctx, model.CustomerTransaction{Id: primitive.NewObjectID(), CustomerId: customer.Id.Hex(), TransactionType: code.TransactionTypeOpeningAR, Direction: code.TransactionDirectionIncrease, Status: code.TransactionStatusConfirmed, Amount: 1000, IdempotencyKey: "opening"}); err != nil {
		t.Fatal(err)
	}
	invoke := func(method, path string, handler http.HandlerFunc, payload any, destination any) {
		t.Helper()
		body, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
		r := httptest.NewRequest(method, path, bytes.NewReader(body)).WithContext(ctx)
		if method == http.MethodGet {
			r = httptest.NewRequest(method, path, nil).WithContext(ctx)
		} else {
			r.Header.Set("Content-Type", "application/json")
		}
		w := httptest.NewRecorder()
		handler(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("%s HTTP %d: %s", path, w.Code, w.Body.String())
		}
		if err = json.Unmarshal(w.Body.Bytes(), destination); err != nil {
			t.Fatalf("%s envelope: %s (%v)", path, w.Body.String(), err)
		}
	}
	var invoice types.CustomerInvoiceResponse
	invoke(http.MethodPost, "/customer/invoice", invoicehandler.AddHandler(s), map[string]any{"idempotency_key": "http-invoice", "customer_id": customer.Id.Hex(), "invoice_number": "HTTP-FP", "invoice_date": int64(1700000000), "tax_rate": 0, "amount_excluding_tax": 100, "tax_amount": 0, "total_amount": 100}, &invoice)
	if invoice.Code != 200 || invoice.Data.Id == "" || invoice.Data.TaxRate != 0 || invoice.Data.TaxAmount != 0 || invoice.Data.TotalAmount != 100 {
		t.Fatalf("zero-tax payload: %+v", invoice)
	}
	var acceptance types.CustomerAcceptanceResponse
	invoke(http.MethodPost, "/customer/acceptance", acceptancehandler.AddHandler(s), map[string]any{"idempotency_key": "http-acceptance", "customer_id": customer.Id.Hex(), "bill_number": "HTTP-BILL", "amount": 100, "issue_date": int64(1700000000), "received_date": int64(1700000100), "due_date": int64(1800000000)}, &acceptance)
	if acceptance.Code != 200 || acceptance.Data.Status != "pending" {
		t.Fatalf("acceptance payload: %+v", acceptance)
	}
	var before model.Customer
	if err = s.CustomerModel.FindOne(ctx, bson.M{"_id": customer.Id}).Decode(&before); err != nil {
		t.Fatal(err)
	}
	if before.ReceivableBalance != 1000 {
		t.Fatal("HTTP registration changed balance")
	}
	var redeemed types.CustomerAcceptanceResponse
	// redemption_annex and remark deliberately omitted, matching both clients'
	// optional voucher flow. Zero-tax fields above are present numeric zeros.
	invoke(http.MethodPost, "/customer/acceptance/redeem", acceptancehandler.RedeemHandler(s), map[string]any{"id": acceptance.Data.Id, "version": acceptance.Data.Version, "redemption_date": int64(1700000200), "idempotency_key": "http-redeem"}, &redeemed)
	if redeemed.Code != 200 || redeemed.Data.Status != "redeemed" || redeemed.Data.TransactionId == "" || len(redeemed.Data.RedemptionAnnex) != 0 {
		t.Fatalf("optional redemption voucher payload: %+v", redeemed)
	}
	var detail types.CustomerAcceptanceResponse
	invoke(http.MethodGet, "/customer/acceptance/detail?id="+acceptance.Data.Id, acceptancehandler.DetailHandler(s), nil, &detail)
	if detail.Code != 200 || detail.Data.TransactionId != redeemed.Data.TransactionId {
		t.Fatalf("detail envelope: %+v", detail)
	}
	var after model.Customer
	if err = s.CustomerModel.FindOne(ctx, bson.M{"_id": customer.Id}).Decode(&after); err != nil {
		t.Fatal(err)
	}
	if after.ReceivableBalance != 900 || after.CreditBalance != 0 {
		t.Fatalf("HTTP redemption balance: %+v", after)
	}
}
