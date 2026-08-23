package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

type repeatingByteReader struct{}

func (repeatingByteReader) Read(buffer []byte) (int, error) {
	for index := range buffer {
		buffer[index] = 'x'
	}
	return len(buffer), nil
}

func TestLoginIdentifiesWindowsDevice(t *testing.T) {
	var payload map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(`{"code":200,"msg":"成功","data":{"token":"token-value"}}`))
	}))
	defer server.Close()

	if _, err := NewClient(server.URL).Login(context.Background(), "18800000000", "secret"); err != nil {
		t.Fatal(err)
	}
	if payload["device_type"] != "windows" {
		t.Fatalf("device_type = %q", payload["device_type"])
	}
}

func TestClientUsesRawTokenAndPagination(t *testing.T) {
	var gotAuth, gotQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":200,"msg":"成功","data":{"total":1,"list":[]}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	client.SetToken("token-value")
	if _, err := client.Materials(context.Background(), 2, 10, MaterialFilters{Name: "钢"}); err != nil {
		t.Fatal(err)
	}
	if gotAuth != "token-value" {
		t.Fatalf("Authorization = %q", gotAuth)
	}
	if gotQuery != "name=%E9%92%A2&page=2&size=10" {
		t.Fatalf("query = %q", gotQuery)
	}
}

func TestClientSendsCompleteFilterContracts(t *testing.T) {
	var queries []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.Path+"?"+r.URL.RawQuery)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":200,"msg":"成功","data":{"total":0,"list":[]}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	_, _ = client.Materials(context.Background(), 1, 20, MaterialFilters{
		Name: "螺栓", CategoryID: "category", Material: "碳钢", Specification: "M8",
		Model: "A-1", SurfaceTreatment: "镀锌", StrengthGrade: "8.8",
	})
	_, _ = client.Inventory(context.Background(), 1, 20, InventoryFilters{
		Type: "采购入库", MaterialName: "螺栓", MaterialModel: "A-1",
		WarehouseID: "warehouse", WarehouseZoneID: "zone", WarehouseRackID: "rack", WarehouseBinID: "bin",
	})
	_, _ = client.InboundReceipts(context.Background(), 1, 20, InboundFilters{
		Code: "RK-001", Status: "在途", Type: "采购入库", SupplierID: "supplier", CustomerID: "customer",
	})

	want := []string{
		"/material?category_id=category&material=%E7%A2%B3%E9%92%A2&model=A-1&name=%E8%9E%BA%E6%A0%93&page=1&size=20&specification=M8&strength_grade=8.8&surface_treatment=%E9%95%80%E9%94%8C",
		"/inventory?material_model=A-1&material_name=%E8%9E%BA%E6%A0%93&page=1&size=20&type=%E9%87%87%E8%B4%AD%E5%85%A5%E5%BA%93&warehouse_bin_id=bin&warehouse_id=warehouse&warehouse_rack_id=rack&warehouse_zone_id=zone",
		"/inbound/receipt?code=RK-001&customer_id=customer&page=1&size=20&status=%E5%9C%A8%E9%80%94&supplier_id=supplier&type=%E9%87%87%E8%B4%AD%E5%85%A5%E5%BA%93",
	}
	if len(queries) != len(want) {
		t.Fatalf("queries = %#v", queries)
	}
	for i := range want {
		if queries[i] != want[i] {
			t.Fatalf("query %d = %q, want %q", i, queries[i], want[i])
		}
	}
}

func TestClientUsesPriceTransactionAndReviseContracts(t *testing.T) {
	var requests []string
	var revise OutboundReviseRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path+"?"+r.URL.RawQuery)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/material/price":
			_, _ = w.Write([]byte(`{"code":200,"msg":"成功","data":[{"price":12.5,"customer_id":"customer","source_valid":true}]}`))
		case "/customer/transaction":
			_, _ = w.Write([]byte(`{"code":200,"msg":"成功","data":{"total":1,"list":[{"transaction_type":"payment","amount":10}]}}`))
		case "/outbound/revise":
			if err := json.NewDecoder(r.Body).Decode(&revise); err != nil {
				t.Fatal(err)
			}
			_, _ = w.Write([]byte(`{"code":200,"msg":"成功"}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL)
	prices, err := client.MaterialPrices(context.Background(), "material", "customer")
	if err != nil || len(prices) != 1 || prices[0].Price != 12.5 || !prices[0].SourceValid {
		t.Fatalf("prices=%#v err=%v", prices, err)
	}
	transactions, err := client.CustomerTransactions(context.Background(), "customer", 2, 20)
	if err != nil || transactions.Total != 1 || len(transactions.List) != 1 {
		t.Fatalf("transactions=%#v err=%v", transactions, err)
	}
	request := OutboundReviseRequest{
		Code: "CK-001", CustomerID: "customer",
		MaterialsPrice: []OutboundMaterialPrice{{MaterialID: "material", Price: 12.5}},
	}
	if err = client.ReviseOutbound(context.Background(), request); err != nil {
		t.Fatal(err)
	}

	want := []string{
		"GET /material/price?customer_id=customer&material_id=material",
		"GET /customer/transaction?customer_id=customer&page=2&size=20",
		"PATCH /outbound/revise?",
	}
	if len(requests) != len(want) {
		t.Fatalf("requests = %#v", requests)
	}
	for index := range want {
		if requests[index] != want[index] {
			t.Fatalf("request %d = %q, want %q", index, requests[index], want[index])
		}
	}
	if revise.Code != request.Code || revise.CustomerID != request.CustomerID || len(revise.MaterialsPrice) != 1 {
		t.Fatalf("revise payload = %#v", revise)
	}
}

func TestInboundRecordDecodesBatchMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"code":200,"msg":"成功","data":[{"id":"record","inbound_receipt_id":"receipt","code":"WIN-1","carrier_name":"承运商","carrier_cost":8.5,"other_cost":1.5,"total_amount":30,"annex":["a.png"],"creator_name":"张三","materials":[]}]}`))
	}))
	defer server.Close()

	records, err := NewClient(server.URL).InboundRecords(context.Background(), "receipt")
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 || records[0].CarrierName != "承运商" || records[0].CarrierCost != 8.5 || len(records[0].Annex) != 1 {
		t.Fatalf("records = %#v", records)
	}
}

func TestClientUsesInboundLifecycleContracts(t *testing.T) {
	var requests []string
	var payloads []InboundReceiptRequest
	var check InboundCheckRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path+"?"+r.URL.RawQuery)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/inbound/receipt":
			var payload InboundReceiptRequest
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatal(err)
			}
			payloads = append(payloads, payload)
		case r.Method == http.MethodPut && r.URL.Path == "/inbound/receipt":
			var payload InboundReceiptRequest
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatal(err)
			}
			payloads = append(payloads, payload)
		case r.Method == http.MethodPatch && r.URL.Path == "/inbound/receipt/check":
			if err := json.NewDecoder(r.Body).Decode(&check); err != nil {
				t.Fatal(err)
			}
		}
		_, _ = w.Write([]byte(`{"code":200,"msg":"成功"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	request := InboundReceiptRequest{
		ID: "receipt", Code: "RK-001", Type: "采购入库", SupplierID: "supplier",
		TotalAmount: 25, ReceivingDate: 123,
		Materials: []InboundMaterialRequest{{
			Index: 1, ID: "material", Price: 12.5, EstimatedQuantity: 2,
			Position: []string{"warehouse", "zone", "rack", "bin"},
		}},
		Annex: []string{"receipt.png"}, Remark: "测试",
	}
	if err := client.CreateInboundReceipt(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if err := client.UpdateInboundReceipt(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if err := client.CheckInboundReceipt(context.Background(), "receipt", "审核通过"); err != nil {
		t.Fatal(err)
	}
	if err := client.DeleteInboundReceipt(context.Background(), "receipt"); err != nil {
		t.Fatal(err)
	}

	want := []string{
		"POST /inbound/receipt?",
		"PUT /inbound/receipt?",
		"PATCH /inbound/receipt/check?",
		"DELETE /inbound/receipt?id=receipt",
	}
	if !reflect.DeepEqual(requests, want) {
		t.Fatalf("requests = %#v, want %#v", requests, want)
	}
	if len(payloads) != 2 || payloads[0].ID != "" || payloads[1].ID != "receipt" {
		t.Fatalf("payloads = %#v", payloads)
	}
	if len(payloads[0].Materials) != 1 || len(payloads[0].Materials[0].Position) != 4 {
		t.Fatalf("materials = %#v", payloads[0].Materials)
	}
	if check.ID != "receipt" || check.Status != "审核通过" {
		t.Fatalf("check = %#v", check)
	}
}

func TestClientReturnsBusinessError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"code":403,"msg":"没有权限"}`))
	}))
	defer server.Close()

	_, err := NewClient(server.URL).Profile(context.Background())
	businessErr, ok := err.(*BusinessError)
	if !ok || businessErr.Code != 403 || businessErr.Msg != "没有权限" {
		t.Fatalf("unexpected error: %#v", err)
	}
}

func TestClientUsesExistingOutboundAndInventoryContracts(t *testing.T) {
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path+"?"+r.URL.RawQuery)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/outbound/page", "/inventory/record":
			_, _ = w.Write([]byte(`{"code":200,"msg":"成功","data":{"total":0,"quantity":0,"list":[]}}`))
		case "/outbound/materials":
			_, _ = w.Write([]byte(`{"code":200,"msg":"成功","data":[]}`))
		case "/carrier", "/customer/list":
			_, _ = w.Write([]byte(`{"code":200,"msg":"成功","data":{"total":0,"list":[]}}`))
		default:
			_, _ = w.Write([]byte(`{"code":200,"msg":"成功","data":null}`))
		}
	}))
	defer server.Close()

	client := NewClient(server.URL)
	_, _ = client.OutboundOrders(context.Background(), 2, 20, OutboundFilters{
		Code: "CK-001", Status: "待打包", IsPack: 0, IsWeigh: -1,
		Type: "销售出库", SupplierID: "supplier", CustomerID: "customer",
		StartTime: 100, EndTime: 200,
	})
	_, _ = client.OutboundMaterials(context.Background(), "CK-001")
	_, _ = client.InventoryHistory(context.Background(), 1, 20, InventoryFilters{MaterialName: "螺栓"})
	_, _ = client.Carriers(context.Background())
	_, _ = client.Customers(context.Background())
	_ = client.ConfirmOutbound(context.Background(), OutboundConfirmRequest{
		Code: "CK-001", ConfirmTime: 123,
		Materials: []OutboundConfirmMaterial{{
			MaterialID: "material", Index: 1,
			Inventorys: []OutboundConfirmInventory{{InventoryID: "inventory", ShipmentQuantity: 2}},
		}},
	})

	want := []string{
		"GET /outbound/page?code=CK-001&customer_id=customer&end_time=200&is_pack=0&is_weigh=-1&page=2&size=20&start_time=100&status=%E5%BE%85%E6%89%93%E5%8C%85&supplier_id=supplier&type=%E9%94%80%E5%94%AE%E5%87%BA%E5%BA%93",
		"GET /outbound/materials?order_code=CK-001",
		"GET /inventory/record?material_name=%E8%9E%BA%E6%A0%93&page=1&size=20",
		"GET /carrier?page=1&size=100",
		"GET /customer/list?",
		"PATCH /outbound/confirm?",
	}
	if len(requests) != len(want) {
		t.Fatalf("requests = %#v", requests)
	}
	for index := range want {
		if requests[index] != want[index] {
			t.Fatalf("request %d = %q, want %q", index, requests[index], want[index])
		}
	}
}

func TestClientUsesOutboundCreateAndDeleteContracts(t *testing.T) {
	var requests []string
	var created OutboundOrderRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path+"?"+r.URL.RawQuery)
		if r.Method == http.MethodPost {
			if err := json.NewDecoder(r.Body).Decode(&created); err != nil {
				t.Fatal(err)
			}
		}
		_, _ = w.Write([]byte(`{"code":200,"msg":"成功"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	request := OutboundOrderRequest{
		Code: "WIN-OUT-001", Type: "销售出库", CustomerID: "customer", TotalAmount: 25,
		Materials: []OutboundMaterialRequest{{Index: 1, MaterialID: "material", Price: 12.5, Quantity: 2}},
		Annex:     []string{"order.png"}, Remark: "Windows 客户端",
	}
	if err := client.CreateOutbound(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if err := client.DeleteOutbound(context.Background(), "order-id"); err != nil {
		t.Fatal(err)
	}

	want := []string{"POST /outbound?", "DELETE /outbound?id=order-id"}
	if !reflect.DeepEqual(requests, want) {
		t.Fatalf("requests = %#v, want %#v", requests, want)
	}
	if !reflect.DeepEqual(created, request) {
		t.Fatalf("payload = %#v, want %#v", created, request)
	}
}

func TestClientNotifiesUnauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"code":401,"msg":"登录状态已失效"}`))
	}))
	defer server.Close()

	called := 0
	client := NewClient(server.URL)
	client.SetUnauthorizedHandler(func() { called++ })
	_, err := client.Profile(context.Background())
	if err == nil || called != 1 {
		t.Fatalf("err=%v called=%d", err, called)
	}
}

func TestClientClassifiesTransportTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		_, _ = w.Write([]byte(`{"code":200,"msg":"成功","data":{}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	client.http.Timeout = 5 * time.Millisecond
	_, err := client.Profile(context.Background())
	var transportErr *TransportError
	if !errors.As(err, &transportErr) || !transportErr.Timeout() {
		t.Fatalf("error = %#v", err)
	}
}

func TestOperationTimeoutPolicyAndCallerDeadline(t *testing.T) {
	if got := operationTimeout(http.MethodGet, "/material"); got != readRequestTimeout {
		t.Fatalf("GET timeout = %s", got)
	}
	if got := operationTimeout(http.MethodPatch, "/outbound/pick"); got != writeRequestTimeout {
		t.Fatalf("write timeout = %s", got)
	}
	if got := operationTimeout(http.MethodPost, "/images"); got != imageUploadTimeout {
		t.Fatalf("image upload timeout = %s", got)
	}

	parent, parentCancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer parentCancel()
	child, childCancel := withOperationTimeout(parent, time.Minute)
	defer childCancel()
	parentDeadline, _ := parent.Deadline()
	childDeadline, _ := child.Deadline()
	if !childDeadline.Equal(parentDeadline) {
		t.Fatalf("child deadline %v should preserve earlier caller deadline %v", childDeadline, parentDeadline)
	}
}

func TestClientUsesDirectoryWarehouseAndSummaryContracts(t *testing.T) {
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path+"?"+r.URL.RawQuery)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/outbound/summary":
			_, _ = w.Write([]byte(`{"code":200,"msg":"成功","data":[]}`))
		default:
			_, _ = w.Write([]byte(`{"code":200,"msg":"成功","data":{"total":0,"list":[]}}`))
		}
	}))
	defer server.Close()

	client := NewClient(server.URL)
	filters := PartnerFilters{
		Name: "正时", Code: "ZS", Manager: "王", Contact: "18800000000",
		Email: "audit@example.com", Level: 3,
	}
	_, _ = client.SupplierDirectory(context.Background(), 2, 20, filters)
	_, _ = client.CustomerDirectory(context.Background(), 3, 50, filters)
	_, _ = client.CarrierDirectory(context.Background(), 1, 10, filters)
	_, _ = client.Warehouses(context.Background(), 1, 20, WarehouseFilters{
		Type: "生产仓库", Name: "一号", Code: "WH-01", Status: "激活",
	})
	_, _ = client.WarehouseZones(context.Background(), 1, 20, WarehouseFilters{
		Name: "原料区", WarehouseID: "warehouse",
	})
	_, _ = client.WarehouseRacks(context.Background(), 1, 20, WarehouseFilters{
		Name: "A架", WarehouseID: "warehouse", WarehouseZoneID: "zone",
	})
	_, _ = client.WarehouseBins(context.Background(), 1, 20, WarehouseFilters{
		Name: "A01", WarehouseID: "warehouse", WarehouseZoneID: "zone", WarehouseRackID: "rack",
	})
	_, _ = client.OutboundSummary(context.Background(), "customer", 100, 200)

	want := []string{
		"GET /supplier?code=ZS&contact=18800000000&email=audit%40example.com&level=3&manager=%E7%8E%8B&name=%E6%AD%A3%E6%97%B6&page=2&size=20",
		"GET /customer?code=ZS&contact=18800000000&email=audit%40example.com&manager=%E7%8E%8B&name=%E6%AD%A3%E6%97%B6&page=3&size=50",
		"GET /carrier?code=ZS&contact=18800000000&email=audit%40example.com&manager=%E7%8E%8B&name=%E6%AD%A3%E6%97%B6&page=1&size=10",
		"GET /warehouse?code=WH-01&name=%E4%B8%80%E5%8F%B7&page=1&size=20&status=%E6%BF%80%E6%B4%BB&type=%E7%94%9F%E4%BA%A7%E4%BB%93%E5%BA%93",
		"GET /warehouse_zone?name=%E5%8E%9F%E6%96%99%E5%8C%BA&page=1&size=20&warehouse_id=warehouse",
		"GET /warehouse_rack?name=A%E6%9E%B6&page=1&size=20&warehouse_id=warehouse&warehouse_zone_id=zone",
		"GET /warehouse_bin?name=A01&page=1&size=20&warehouse_id=warehouse&warehouse_rack_id=rack&warehouse_zone_id=zone",
		"GET /outbound/summary?customer_id=customer&end_date=200&start_date=100",
	}
	if len(requests) != len(want) {
		t.Fatalf("requests = %#v", requests)
	}
	for index := range want {
		if requests[index] != want[index] {
			t.Fatalf("request %d = %q, want %q", index, requests[index], want[index])
		}
	}
}

func TestInboundReceiptDecodesAttachments(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"code":200,"msg":"成功","data":{"total":1,"list":[{"id":"inbound","code":"RK-1","annex":["a.png","b.jpg"]}]}}`))
	}))
	defer server.Close()

	page, err := NewClient(server.URL).InboundReceipts(context.Background(), 1, 20, InboundFilters{})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.List) != 1 || len(page.List[0].Annex) != 2 {
		t.Fatalf("attachments = %#v", page.List)
	}
}

func TestUploadImageUsesMultipartFilesField(t *testing.T) {
	var gotFilename, gotBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data; boundary=") {
			t.Fatalf("Content-Type = %q", r.Header.Get("Content-Type"))
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatal(err)
		}
		file, header, err := r.FormFile("files")
		if err != nil {
			t.Fatal(err)
		}
		defer file.Close()
		gotFilename = header.Filename
		data, _ := io.ReadAll(file)
		gotBody = string(data)
		_, _ = w.Write([]byte(`{"code":200,"msg":"成功","data":{"url":"receipt.png"}}`))
	}))
	defer server.Close()

	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "签收.png")
	if err := os.WriteFile(filePath, []byte("image-bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	url, err := NewClient(server.URL).UploadImage(context.Background(), filePath)
	if err != nil {
		t.Fatal(err)
	}
	if gotFilename != "签收.png" || gotBody != "image-bytes" || url != "receipt.png" {
		t.Fatalf("filename=%q body=%q url=%q", gotFilename, gotBody, url)
	}
}

func TestResolveAndDownloadImage(t *testing.T) {
	var gotAuthorization string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuthorization = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte("image-bytes"))
	}))
	defer server.Close()

	imageURL, err := ResolveImageURL(server.URL+"/images/", "物料图纸/test 01.png")
	if err != nil {
		t.Fatal(err)
	}
	if imageURL != server.URL+"/images/%E7%89%A9%E6%96%99%E5%9B%BE%E7%BA%B8/test%2001.png" {
		t.Fatalf("imageURL = %q", imageURL)
	}

	client := NewClient(server.URL)
	client.SetToken("must-not-leak")
	data, err := client.DownloadImage(context.Background(), imageURL)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "image-bytes" {
		t.Fatalf("data = %q", data)
	}
	if gotAuthorization != "" {
		t.Fatalf("Authorization leaked to image host: %q", gotAuthorization)
	}
}

func TestDownloadImageRejectsOversizedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.CopyN(w, repeatingByteReader{}, maxImageDownloadBytes+1)
	}))
	defer server.Close()

	_, err := NewClient(server.URL).DownloadImage(context.Background(), server.URL+"/large.png")
	if err == nil || !strings.Contains(err.Error(), "25 MB") {
		t.Fatalf("error = %v", err)
	}
}
