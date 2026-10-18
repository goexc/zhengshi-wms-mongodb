package handler

import (
	"api/internal/svc"
	"api/internal/types"
	"bytes"
	"context"
	"encoding/json"
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
	"github.com/golang-jwt/jwt/v4"
	"github.com/tencentyun/cos-go-sdk-v5"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/rest"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Exercise the actual registered JWT route and size/timeout middleware, rather
// than only calling AddLogic. The COS substitute is deliberately slower than
// go-zero's ordinary three-second default and the body exceeds its 1 MiB default.
func TestRegisteredUploadAcceptsLargeSlowScopedFile(t *testing.T) {
	uri := os.Getenv("WMS_TEST_MONGO_URI")
	if uri == "" {
		t.Skip("set WMS_TEST_MONGO_URI to isolated loopback replica set")
	}
	u, err := url.Parse(uri)
	if err != nil || u.Scheme != "mongodb" || u.User != nil || strings.Contains(u.Host, ",") || (u.Hostname() != "127.0.0.1" && u.Hostname() != "localhost") {
		t.Fatal("only credential-free loopback MongoDB is allowed")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		t.Fatal(err)
	}
	db := client.Database("wms_upload_route_test_" + primitive.NewObjectID().Hex())
	defer func() {
		cleanup, done := context.WithTimeout(context.Background(), 5*time.Second)
		defer done()
		if err := db.Drop(cleanup); err != nil {
			t.Error(err)
		}
		_ = client.Disconnect(cleanup)
	}()
	model, err := casbinmodel.NewModelFromString(`[request_definition]
r = sub, obj, act
[policy_definition]
p = sub, obj, act
[role_definition]
g = _, _
[policy_effect]
e = some(where (p.eft == allow))
[matchers]
m = g(r.sub,p.sub) && r.obj == p.obj && r.act == p.act`)
	if err != nil {
		t.Fatal(err)
	}
	enforcer, err := casbin.NewSyncedEnforcer(model)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = enforcer.AddPolicy("user_upload-test", "/customer/invoice", "POST")
	var objectPath string
	cosServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		objectPath = r.URL.Path
		select {
		case <-time.After(3400 * time.Millisecond):
		case <-r.Context().Done():
			return
		}
		w.Header().Set("x-cos-hash-crc64ecma", strconv.FormatUint(crc64.Checksum(body, crc64.MakeTable(crc64.ECMA)), 10))
		w.WriteHeader(200)
	}))
	defer cosServer.Close()
	cosURL, _ := url.Parse(cosServer.URL)
	s := &svc.ServiceContext{DBClient: client, ImageModel: db.Collection("image"), Enforcer: enforcer, Cos: cos.NewClient(&cos.BaseURL{BucketURL: cosURL}, cosServer.Client())}
	s.Config.Auth.AccessSecret = "isolated-upload-route-secret"
	server, err := rest.NewServer(rest.RestConf{ServiceConf: service.ServiceConf{Name: "upload-route-test", Mode: service.TestMode}, Host: "127.0.0.1", Port: 0, Timeout: 3000, MaxBytes: 1 << 20, Middlewares: rest.MiddlewaresConf{Timeout: true, MaxBytes: true}})
	if err != nil {
		t.Fatal(err)
	}
	defer server.Stop()
	RegisterHandlers(server, s)
	httpServer := httptest.NewServer(server)
	defer httpServer.Close()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("scope", "invoice")
	part, _ := writer.CreateFormFile("files", "large.pdf")
	_, _ = io.Copy(part, strings.NewReader(strings.Repeat("x", 2<<20)))
	_ = writer.Close()
	r, err := http.NewRequestWithContext(ctx, http.MethodPost, httpServer.URL+"/images", &body)
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Set("Content-Type", writer.FormDataContentType())
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"uid": "upload-test", "name": "上传测试", "iat": time.Now().Unix(), "exp": time.Now().Add(time.Minute).Unix()}).SignedString([]byte(s.Config.Auth.AccessSecret))
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Set("Authorization", "Bearer "+token)
	response, err := httpServer.Client().Do(r)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	var result types.ImageResponse
	if response.StatusCode != 200 || json.Unmarshal(data, &result) != nil || result.Code != 200 || !strings.HasPrefix(result.Data.Url, "invoice/") {
		t.Fatalf("registered upload response: HTTP%d %s", response.StatusCode, data)
	}
	if objectPath != "/images/"+result.Data.Url {
		t.Fatalf("COS path %q", objectPath)
	}
	if err = db.Collection("image").FindOne(ctx, bson.M{"object_key": result.Data.Url, "scope": "invoice"}).Err(); err != nil {
		t.Fatal(err)
	}
}
