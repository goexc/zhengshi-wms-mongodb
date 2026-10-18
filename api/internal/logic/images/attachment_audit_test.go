package images

import (
	"api/internal/logic/customer/documents"
	"api/internal/types"
	"bytes"
	"context"
	"fmt"
	"hash/crc64"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tencentyun/cos-go-sdk-v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestLibraryScopesPreserveLegacyAndSearchLiteralNames(t *testing.T) {
	s, ctx, _ := attachmentIntegration(t)
	for _, row := range []bson.M{
		{"object_key": "old.png", "alt": "凭证[1]"},
		{"object_key": "old.pdf", "scope": "", "alt": "凭证[1]"},
		{"object_key": "acceptance/2026/10/a.pdf", "scope": "acceptance", "alt": "凭证[1]"},
		{"object_key": "acceptance/redemption/2026/10/a.pdf", "scope": "acceptance_redemption", "alt": "凭证[1]"},
		{"object_key": "invoice/2026/10/a.pdf", "scope": "invoice", "alt": "凭证[1]"},
		{"object_key": "invoice/2026/10/deleting.pdf", "scope": "invoice", "alt": "凭证[1]", "deleting": true},
	} {
		if _, err := s.ImageModel.InsertOne(ctx, row); err != nil {
			t.Fatal(err)
		}
	}
	for scope, expected := range map[string]int64{"": 2, "acceptance": 1, "acceptance_redemption": 1, "invoice": 1} {
		resp, err := NewPageLogic(ctx, s).Page(&types.ImagesRequest{Page: 1, Size: 20, Scope: scope, Name: "凭证[1]"})
		if err != nil || resp.Code != 200 || resp.Data.Total != expected || len(resp.Data.List) != int(expected) {
			t.Fatalf("scope %q: %+v, %v", scope, resp, err)
		}
	}
}

func TestPhysicalDeletePreservesScopedPathsAndRetriesCOSFailure(t *testing.T) {
	s, ctx, _ := attachmentIntegration(t)
	var fail atomic.Bool
	fail.Store(true)
	paths := make(chan string, 32)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths <- r.Method + " " + r.URL.Path
		if fail.Load() {
			w.WriteHeader(500)
			_, _ = io.WriteString(w, `<Error><Code>InternalError</Code></Error>`)
			return
		}
		w.WriteHeader(204)
	}))
	defer server.Close()
	u, _ := url.Parse(server.URL)
	s.Cos = cos.NewClient(&cos.BaseURL{BucketURL: u}, server.Client())
	for _, key := range []string{"legacy.pdf", "acceptance/2026/10/a.pdf", "acceptance/redemption/2026/10/a.pdf", "invoice/2026/10/a.pdf"} {
		if _, err := s.ImageModel.InsertOne(ctx, bson.M{"object_key": key}); err != nil {
			t.Fatal(err)
		}
		resp, err := NewRemoveLogic(ctx, s).Remove(&types.ImageRemoveRequest{Key: key})
		if fail.Load() {
			if err != nil || resp.Code == 200 {
				t.Fatalf("failed COS delete reported success: %+v %v", resp, err)
			}
			if err = s.ImageModel.FindOne(ctx, bson.M{"object_key": key, "deleting": true}).Err(); err != nil {
				t.Fatal(err)
			}
			fail.Store(false)
			resp, err = NewRemoveLogic(ctx, s).Remove(&types.ImageRemoveRequest{Key: key})
		}
		if err != nil || resp.Code != 200 {
			t.Fatalf("delete %s: %+v %v", key, resp, err)
		}
		if err = s.ImageModel.FindOne(ctx, bson.M{"object_key": key}).Err(); err != mongo.ErrNoDocuments {
			t.Fatalf("metadata not removed: %v", err)
		}
		found := false
		for len(paths) > 0 {
			if got := <-paths; got != "DELETE /images/"+key {
				t.Fatalf("wrong COS delete path: %s", got)
			}
			found = true
		}
		if !found {
			t.Fatal("COS delete was never sent")
		}
	}
}

func TestLegacyAndFinancialReferencesBlockPhysicalDeletion(t *testing.T) {
	s, ctx, _ := attachmentIntegration(t)
	for i, entry := range []struct {
		collection *mongo.Collection
		field      string
	}{
		{s.OutboundOrderModel, "annex"}, {s.MaterialModel, "image"}, {s.CustomerTransactionModel, "annex"},
		{s.CustomerAcceptanceModel, "annex"}, {s.CustomerAcceptanceModel, "redemption_annex"},
		{s.CustomerInvoiceModel, "annex"}, {s.CustomerInvoiceModel, "attachment_history"},
	} {
		key := fmt.Sprintf("old%d.png", i)
		if _, err := s.ImageModel.InsertOne(ctx, bson.M{"object_key": key}); err != nil {
			t.Fatal(err)
		}
		value := any(key)
		if entry.field == "annex" {
			value = "another.pdf," + key + ",last.png"
		}
		if entry.field == "attachment_history" {
			value = bson.A{key}
		}
		if _, err := entry.collection.InsertOne(ctx, bson.M{entry.field: value, "status": "voided"}); err != nil {
			t.Fatal(err)
		}
		// Cos is nil: reaching physical deletion would panic and fail the test.
		resp, err := NewRemoveLogic(ctx, s).Remove(&types.ImageRemoveRequest{Key: key})
		if err != nil || resp.Code != 400 || !strings.Contains(resp.Msg, "引用") {
			t.Fatalf("reference not protected: %+v %v", resp, err)
		}
	}
}

func TestInvoiceBindingRacesDeletionWithoutDanglingFile(t *testing.T) {
	s, ctx, db := attachmentIntegration(t)
	customerID := primitive.NewObjectID()
	if _, err := s.CustomerModel.InsertOne(ctx, bson.M{"_id": customerID, "name": "附件并发测试客户", "status": "活动"}); err != nil {
		t.Fatal(err)
	}
	var deleted sync.Map
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		deleted.Store(strings.TrimPrefix(r.URL.Path, "/images/"), true)
		w.WriteHeader(204)
	}))
	defer server.Close()
	u, _ := url.Parse(server.URL)
	s.Cos = cos.NewClient(&cos.BaseURL{BucketURL: u}, server.Client())
	for i := 0; i < 12; i++ {
		key := fmt.Sprintf("invoice/2026/10/race-%d.pdf", i)
		if _, err := s.ImageModel.InsertOne(ctx, bson.M{"object_key": key, "scope": "invoice"}); err != nil {
			t.Fatal(err)
		}
		req := types.CustomerInvoiceSaveRequest{CustomerId: customerID.Hex(), InvoiceNumber: fmt.Sprintf("RACE-%d", i), InvoiceDate: 1700000000, AmountExcludingTax: 100, TaxAmount: 0, TaxRate: 0, TotalAmount: 100, Annex: []string{key}, IdempotencyKey: fmt.Sprintf("race-%d", i)}
		start := make(chan struct{})
		var wg sync.WaitGroup
		wg.Add(2)
		var save *types.CustomerInvoiceResponse
		var remove *types.BaseResponse
		var removeErr error
		go func() { defer wg.Done(); <-start; save = documents.SaveInvoice(ctx, s, &req, true) }()
		go func() {
			defer wg.Done()
			<-start
			remove, removeErr = NewRemoveLogic(ctx, s).Remove(&types.ImageRemoveRequest{Key: key})
		}()
		close(start)
		wg.Wait()
		if removeErr != nil {
			t.Fatal(removeErr)
		}
		_, gone := deleted.Load(key)
		if save.Code == 200 {
			if gone || remove.Code == 200 {
				t.Fatalf("bound file deleted: save=%+v remove=%+v", save, remove)
			}
			if err := s.ImageModel.FindOne(ctx, bson.M{"object_key": key, "deleting": bson.M{"$ne": true}}).Err(); err != nil {
				t.Fatal(err)
			}
		} else {
			if !gone || remove.Code != 200 {
				t.Fatalf("neither operation succeeded: save=%+v remove=%+v", save, remove)
			}
			if n, err := db.Collection("customer_invoice").CountDocuments(ctx, bson.M{"annex": key}); err != nil || n != 0 {
				t.Fatalf("dangling reference: %d %v", n, err)
			}
		}
	}
}

func TestMetadataWriteConcernFailureRecoversCommittedUpload(t *testing.T) {
	if os.Getenv("WMS_TEST_MONGO_FAILPOINTS") != "1" {
		t.Skip("requires explicit disposable MongoDB failpoint opt-in")
	}
	s, ctx, db := attachmentIntegration(t)
	appName := "wms_attachment_failure_" + primitive.NewObjectID().Hex()
	var injected atomic.Bool
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(os.Getenv("WMS_TEST_MONGO_URI")).SetAppName(appName).SetRetryWrites(false).SetMonitor(&event.CommandMonitor{Succeeded: func(_ context.Context, e *event.CommandSucceededEvent) {
		if e.CommandName == "insert" && e.Reply.Lookup("writeConcernError").Type != 0 {
			injected.Store(true)
		}
	}}))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Disconnect(context.Background())
	s.ImageModel = client.Database(db.Name()).Collection("image")
	admin := s.DBClient.Database("admin")
	cmd := bson.D{{Key: "configureFailPoint", Value: "failCommand"}, {Key: "mode", Value: bson.M{"times": 1}}, {Key: "data", Value: bson.M{"failCommands": bson.A{"insert"}, "appName": appName, "writeConcernError": bson.M{"code": 64, "errmsg": "isolated metadata acknowledgement failure"}}}}
	if err = admin.RunCommand(ctx, cmd).Err(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = admin.RunCommand(cleanup, bson.D{{Key: "configureFailPoint", Value: "failCommand"}, {Key: "mode", Value: "off"}}).Err()
	}()
	var puts, deletes atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if r.Method == "PUT" {
			puts.Add(1)
		} else if r.Method == "DELETE" {
			deletes.Add(1)
		}
		w.Header().Set("x-cos-hash-crc64ecma", strconv.FormatUint(crc64.Checksum(body, crc64.MakeTable(crc64.ECMA)), 10))
		w.WriteHeader(200)
	}))
	defer server.Close()
	u, _ := url.Parse(server.URL)
	s.Cos = cos.NewClient(&cos.BaseURL{BucketURL: u}, server.Client())
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("scope", "invoice")
	f, _ := writer.CreateFormFile("files", "test.pdf")
	_, _ = f.Write([]byte("%PDF-test"))
	_ = writer.Close()
	r := httptest.NewRequest("POST", "/images", &body)
	r.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := NewAddLogic(ctx, s).Add(r)
	if !injected.Load() || err != nil || resp.Code != 200 || resp.Data.Url == "" || puts.Load() != 1 || deletes.Load() != 0 {
		t.Fatalf("recovery injected=%t response=%+v err=%v puts=%d deletes=%d", injected.Load(), resp, err, puts.Load(), deletes.Load())
	}
	if count, err := s.ImageModel.CountDocuments(ctx, bson.M{"object_key": resp.Data.Url}); err != nil || count != 1 {
		t.Fatalf("metadata recovery: %d %v", count, err)
	}
}
