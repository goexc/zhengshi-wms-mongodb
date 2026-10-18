package images

import (
	"api/internal/svc"
	"api/model"
	"api/pkg/attachmentx"
	"bytes"
	"context"
	"hash/crc64"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/casbin/casbin/v2"
	casbinmodel "github.com/casbin/casbin/v2/model"
	"github.com/tencentyun/cos-go-sdk-v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func attachmentEnforcer(t *testing.T) *casbin.SyncedEnforcer {
	t.Helper()
	m, e := casbinmodel.NewModelFromString(`[request_definition]
r = sub, obj, act
[policy_definition]
p = sub, obj, act
[role_definition]
g = _, _
[policy_effect]
e = some(where (p.eft == allow))
[matchers]
m = g(r.sub,p.sub) && r.obj == p.obj && r.act == p.act`)
	if e != nil {
		t.Fatal(e)
	}
	enforcer, e := casbin.NewSyncedEnforcer(m)
	if e != nil {
		t.Fatal(e)
	}
	return enforcer
}

func TestScopePermissions(t *testing.T) {
	s := &svc.ServiceContext{Enforcer: attachmentEnforcer(t)}
	s.Config.Ids.Role = "admin"
	ctx := context.WithValue(context.Background(), "uid", "tester")
	if !scopeAllowed(ctx, s, "", true) {
		t.Fatal("legacy uploads changed")
	}
	if scopeAllowed(ctx, s, "invoice", true) {
		t.Fatal("ungranted invoice upload")
	}
	_, _ = s.Enforcer.AddPolicy("user_tester", "/customer/invoice", "POST")
	if !scopeAllowed(ctx, s, "invoice", true) || scopeAllowed(ctx, s, "acceptance", true) {
		t.Fatal("scope isolation failed")
	}
	_, _ = s.Enforcer.AddPolicy("user_tester", "/customer/acceptance", "GET")
	if !scopeAllowed(ctx, s, "acceptance_redemption", false) || scopeAllowed(ctx, s, "acceptance_redemption", true) {
		t.Fatal("read grants must not allow redemption upload")
	}
	_, _ = s.Enforcer.AddPolicy("user_tester", "/customer/acceptance/redeem", "POST")
	if !scopeAllowed(ctx, s, "acceptance_redemption", true) {
		t.Fatal("redeem permission rejected")
	}
}

func TestInvalidUploadRejectedBeforeStorage(t *testing.T) {
	for _, tc := range []struct {
		name, scope  string
		bytes, count int
	}{
		{"empty.pdf", "", 0, 1}, {"too-large.pdf", "", int(attachmentx.MaxFileBytes) + 1, 1},
		{"multiple.pdf", "", 1, 2}, {"scope.pdf", "../../elsewhere", 1, 1},
	} {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		_ = writer.WriteField("scope", tc.scope)
		for i := 0; i < tc.count; i++ {
			f, _ := writer.CreateFormFile("files", tc.name)
			_, _ = io.CopyN(f, strings.NewReader(strings.Repeat("x", tc.bytes)), int64(tc.bytes))
		}
		_ = writer.Close()
		r := httptest.NewRequest(http.MethodPost, "/images", &body)
		r.Header.Set("Content-Type", writer.FormDataContentType())
		response, err := NewAddLogic(context.Background(), &svc.ServiceContext{}).Add(r)
		if err != nil || response.Code != http.StatusBadRequest {
			t.Fatalf("%s: %+v %v", tc.name, response, err)
		}
	}
}

func attachmentIntegration(t *testing.T) (*svc.ServiceContext, context.Context, *mongo.Database) {
	t.Helper()
	uri := os.Getenv("WMS_TEST_MONGO_URI")
	if uri == "" {
		t.Skip("set WMS_TEST_MONGO_URI to isolated loopback replica set")
	}
	u, e := url.Parse(uri)
	if e != nil || u.Scheme != "mongodb" || u.User != nil || (u.Hostname() != "127.0.0.1" && u.Hostname() != "localhost") || strings.Contains(u.Host, ",") {
		t.Fatal("only credential-free loopback test MongoDB is allowed")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	c, e := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if e != nil {
		cancel()
		t.Fatal(e)
	}
	db := c.Database("wms_attachment_test_" + primitive.NewObjectID().Hex())
	for _, name := range []string{"image", "customer_acceptance", "customer_invoice", "customer_transaction", "customer", "outbound_order", "material", "user", "supplier", "carrier", "menu", "api"} {
		if e = db.CreateCollection(ctx, name); e != nil {
			t.Fatal(e)
		}
	}
	t.Cleanup(func() {
		cleanup, done := context.WithTimeout(context.Background(), 10*time.Second)
		defer done()
		if e := db.Drop(cleanup); e != nil {
			t.Error(e)
		}
		_ = c.Disconnect(cleanup)
		cancel()
	})
	s := &svc.ServiceContext{DBClient: c, ImageModel: db.Collection("image"), CustomerAcceptanceModel: db.Collection("customer_acceptance"), CustomerInvoiceModel: db.Collection("customer_invoice"), CustomerTransactionModel: db.Collection("customer_transaction"), MenuModel: db.Collection("menu"), ApiModel: db.Collection("api"), Enforcer: attachmentEnforcer(t)}
	s.CustomerModel, s.OutboundOrderModel, s.MaterialModel = db.Collection("customer"), db.Collection("outbound_order"), db.Collection("material")
	s.UserModel, s.SupplierModel, s.CarrierModel = db.Collection("user"), db.Collection("supplier"), db.Collection("carrier")
	s.Config.Ids.Role = "admin"
	_, _ = s.Enforcer.AddRoleForUser("user_test", "role_admin")
	ctx = context.WithValue(ctx, "uid", "test")
	return s, ctx, db
}

func TestUploadScopesAndLegacyCOSPaths(t *testing.T) {
	s, ctx, _ := attachmentIntegration(t)
	paths := make(chan string, 8)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("x-cos-hash-crc64ecma", strconv.FormatUint(crc64.Checksum(body, crc64.MakeTable(crc64.ECMA)), 10))
		paths <- r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	u, _ := url.Parse(server.URL)
	s.Cos = cos.NewClient(&cos.BaseURL{BucketURL: u}, server.Client())
	for _, scope := range []string{"", "acceptance", "acceptance_redemption", "invoice"} {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		_ = writer.WriteField("scope", scope)
		f, _ := writer.CreateFormFile("files", "原件.PDF")
		_, _ = f.Write([]byte("%PDF-1.7 test"))
		_ = writer.Close()
		r := httptest.NewRequest(http.MethodPost, "/images", &body)
		r.Header.Set("Content-Type", writer.FormDataContentType())
		resp, e := NewAddLogic(ctx, s).Add(r)
		if e != nil || resp.Code != 200 {
			t.Fatalf("%s: %+v %v", scope, resp, e)
		}
		if got := <-paths; got != "/images/"+resp.Data.Url {
			t.Fatalf("wrong COS path %s", got)
		}
		if attachmentx.Scope(resp.Data.Url) != scope {
			t.Fatalf("wrong relative key %s", resp.Data.Url)
		}
		var stored model.Image
		if e = s.ImageModel.FindOne(ctx, bson.M{"object_key": resp.Data.Url}).Decode(&stored); e != nil || stored.Scope != scope {
			t.Fatalf("metadata: %+v %v", stored, e)
		}
	}
}

func TestMetadataFailureDoesNotDeleteUploadedObject(t *testing.T) {
	s, ctx, db := attachmentIntegration(t)
	// A storage response failure must never trigger a compensating COS delete.
	// Use a real rejected insert to exercise the conservative recovery branch.
	if err := db.RunCommand(ctx, bson.D{{Key: "collMod", Value: "image"}, {Key: "validator", Value: bson.M{"$jsonSchema": bson.M{"required": bson.A{"test_reject_insert"}}}}}).Err(); err != nil {
		t.Fatal(err)
	}
	methods := make(chan string, 4)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("x-cos-hash-crc64ecma", strconv.FormatUint(crc64.Checksum(body, crc64.MakeTable(crc64.ECMA)), 10))
		methods <- r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	u, _ := url.Parse(server.URL)
	s.Cos = cos.NewClient(&cos.BaseURL{BucketURL: u}, server.Client())
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("scope", "invoice")
	f, _ := writer.CreateFormFile("files", "invoice.pdf")
	_, _ = f.Write([]byte("%PDF-1.7 test"))
	_ = writer.Close()
	r := httptest.NewRequest(http.MethodPost, "/images", &body)
	r.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := NewAddLogic(ctx, s).Add(r)
	if err != nil || resp.Code != http.StatusInternalServerError || resp.Data.Url != "" {
		t.Fatalf("unexpected metadata failure response: %+v %v", resp, err)
	}
	if method := <-methods; method != http.MethodPut {
		t.Fatalf("expected COS upload, got %s", method)
	}
	select {
	case method := <-methods:
		t.Fatalf("unexpected COS operation after uncertain save: %s", method)
	default:
	}
}

func TestFinancialReferenceHistoryAndRemovalReservation(t *testing.T) {
	s, ctx, _ := attachmentIntegration(t)
	key := "invoice/2026/10/test.pdf"
	_, e := s.ImageModel.InsertOne(ctx, bson.M{"object_key": key})
	if e != nil {
		t.Fatal(e)
	}
	result, e := s.CustomerInvoiceModel.InsertOne(ctx, bson.M{"annex": "other.pdf", "attachment_history": bson.A{key}, "status": "voided"})
	if e != nil {
		t.Fatal(e)
	}
	if e = reserveAttachmentRemoval(ctx, s, key); e == nil {
		t.Fatal("deleted historical invoice attachment")
	}
	_, _ = s.CustomerInvoiceModel.DeleteOne(ctx, bson.M{"_id": result.InsertedID})
	if e = reserveAttachmentRemoval(ctx, s, key); e != nil {
		t.Fatal(e)
	}
	bound, e := s.ImageModel.UpdateOne(ctx, bson.M{"object_key": key, "deleting": bson.M{"$ne": true}}, bson.M{"$inc": bson.M{"reference_revision": 1}})
	if e != nil || bound.MatchedCount != 0 {
		t.Fatal("bound a file reserved for deletion")
	}
	if e = reserveAttachmentRemoval(ctx, s, key); e != nil {
		t.Fatalf("COS failure retry not supported: %v", e)
	}
}

func TestFinanceCatalogIsAdditiveAndRepeatable(t *testing.T) {
	s, ctx, _ := attachmentIntegration(t)
	_, e := s.MenuModel.InsertOne(ctx, bson.M{"type": 1, "path": "/finance", "name": "已有自定义财务", "parent_id": "", "perms": "custom_finance"})
	if e != nil {
		t.Fatal(e)
	}
	if e = svc.EnsureFinanceCatalog(s); e != nil {
		t.Fatal(e)
	}
	before, _ := s.MenuModel.CountDocuments(ctx, bson.M{})
	apis, _ := s.ApiModel.CountDocuments(ctx, bson.M{})
	if e = svc.EnsureFinanceCatalog(s); e != nil {
		t.Fatal(e)
	}
	after, _ := s.MenuModel.CountDocuments(ctx, bson.M{})
	afterAPIs, _ := s.ApiModel.CountDocuments(ctx, bson.M{})
	if before != after || apis != afterAPIs {
		t.Fatal("catalog duplicates on restart")
	}
	var row bson.M
	_ = s.MenuModel.FindOne(ctx, bson.M{"path": "/finance"}).Decode(&row)
	if row["name"] != "已有自定义财务" {
		t.Fatal("overwrote existing menu")
	}
}
