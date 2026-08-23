package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestClientUsesMaterialCenterContracts(t *testing.T) {
	var requests []string
	var bodies []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.RequestURI())
		if r.Body != nil && r.ContentLength != 0 {
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			bodies = append(bodies, body)
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/material/info":
			_, _ = w.Write([]byte(`{"code":200,"msg":"成功","data":{"id":"material","category_id":"category","updated_at":9}}`))
		default:
			_, _ = w.Write([]byte(`{"code":200,"msg":"成功"}`))
		}
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx := context.Background()
	if err := client.CreateMaterial(ctx, MaterialRequest{ID: "clear", CategoryID: "category", Name: "螺栓", Model: "M8", Price: 12.5}); err != nil {
		t.Fatal(err)
	}
	if err := client.UpdateMaterial(ctx, MaterialRequest{ID: "material", CategoryID: "category", Name: "螺栓", Model: "M8"}); err != nil {
		t.Fatal(err)
	}
	material, err := client.MaterialInfo(ctx, "material")
	if err != nil || material.CategoryID != "category" || material.UpdatedAt != 9 {
		t.Fatalf("material=%#v err=%v", material, err)
	}
	if err := client.CreateMaterialCategory(ctx, MaterialCategoryRequest{ID: "clear", Name: "标准件", Status: "启用", SortID: 1}); err != nil {
		t.Fatal(err)
	}
	if err := client.UpdateMaterialCategory(ctx, MaterialCategoryRequest{ID: "category", ParentID: "parent", Name: "标准件", Status: "停用", SortID: 2}); err != nil {
		t.Fatal(err)
	}

	want := []string{
		"POST /material",
		"PUT /material",
		"GET /material/info?id=material",
		"POST /material/category",
		"PUT /material/category",
	}
	if !reflect.DeepEqual(requests, want) {
		t.Fatalf("requests=%#v want=%#v", requests, want)
	}
	if _, exists := bodies[0]["id"]; exists {
		t.Fatalf("create material id must be omitted: %#v", bodies[0])
	}
	if got := bodies[1]["id"]; got != "material" {
		t.Fatalf("update material id=%#v", got)
	}
	if _, exists := bodies[2]["id"]; exists {
		t.Fatalf("create category id must be omitted: %#v", bodies[2])
	}
}

func TestClientUsesMaterialQuoteContracts(t *testing.T) {
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.RequestURI())
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/material/new_delivery":
			_, _ = w.Write([]byte(`{"code":200,"msg":"成功","data":{"total":1,"list":[{"id":"delivery","quote_status":"unquoted"}]}}`))
		case "/material/quote":
			if r.Method == http.MethodGet {
				_, _ = w.Write([]byte(`{"code":200,"msg":"成功","data":{"total":1,"list":[{"id":"quote","status":"draft"}]}}`))
			} else {
				_, _ = w.Write([]byte(`{"code":200,"msg":"成功","data":{"id":"quote","status":"draft","source_valid":true}}`))
			}
		case "/material/quote/info", "/material/quote/submit", "/material/quote/price":
			_, _ = w.Write([]byte(`{"code":200,"msg":"成功","data":{"id":"quote","status":"quoted","source_valid":true}}`))
		default:
			_, _ = w.Write([]byte(`{"code":200,"msg":"成功"}`))
		}
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx := context.Background()
	page, err := client.NewCustomerMaterials(ctx, 2, 20, NewCustomerMaterialFilters{
		CustomerID: "customer", StartTime: 10, EndTime: 20, QuoteStatus: "unquoted", MaterialName: "螺栓", MaterialModel: "M8",
	})
	if err != nil || page.Total != 1 {
		t.Fatalf("deliveries=%#v err=%v", page, err)
	}
	quotes, err := client.MaterialQuotes(ctx, 1, 10, MaterialQuoteFilters{CustomerID: "customer", Status: "draft", QuoteMode: "detailed", MaterialModel: "M8"})
	if err != nil || quotes.Total != 1 {
		t.Fatalf("quotes=%#v err=%v", quotes, err)
	}
	quote, err := client.SaveMaterialQuote(ctx, MaterialQuoteSaveRequest{DeliveryID: "delivery", QuoteMode: "simple", SimplePrice: 8})
	if err != nil || quote.ID != "quote" {
		t.Fatalf("saved=%#v err=%v", quote, err)
	}
	if _, err = client.SaveMaterialQuote(ctx, MaterialQuoteSaveRequest{ID: "quote", DeliveryID: "delivery", QuoteMode: "detailed", FinalPrice: 9}); err != nil {
		t.Fatal(err)
	}
	if _, err = client.MaterialQuoteInfo(ctx, "quote"); err != nil {
		t.Fatal(err)
	}
	if _, err = client.SubmitMaterialQuote(ctx, "quote"); err != nil {
		t.Fatal(err)
	}
	if _, err = client.PriceMaterialQuote(ctx, MaterialQuotePriceRequest{ID: "quote", FinalPrice: 10, EffectiveAt: 30}); err != nil {
		t.Fatal(err)
	}
	if err = client.VoidMaterialQuote(ctx, "quote"); err != nil {
		t.Fatal(err)
	}

	want := []string{
		"GET /material/new_delivery?customer_id=customer&end_time=20&material_model=M8&material_name=%E8%9E%BA%E6%A0%93&page=2&quote_status=unquoted&size=20&start_time=10",
		"GET /material/quote?customer_id=customer&material_model=M8&page=1&quote_mode=detailed&size=10&status=draft",
		"POST /material/quote",
		"PUT /material/quote",
		"GET /material/quote/info?id=quote",
		"POST /material/quote/submit",
		"POST /material/quote/price",
		"PATCH /material/quote/void",
	}
	if !reflect.DeepEqual(requests, want) {
		t.Fatalf("requests=%#v want=%#v", requests, want)
	}
}

func TestClientDownloadsMaterialQuoteCSVAndHandlesJSONError(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount == 1 {
			w.Header().Set("Content-Type", "text/csv; charset=utf-8")
			w.Header().Set("Content-Disposition", `attachment; filename="Q-001.csv"`)
			_, _ = w.Write([]byte("quote_no,price\nQ-001,12.5\n"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":400,"msg":"报价单不可导出"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	data, name, err := client.ExportMaterialQuote(context.Background(), "quote")
	if err != nil || name != "Q-001.csv" || !strings.Contains(string(data), "Q-001") {
		t.Fatalf("data=%q name=%q err=%v", data, name, err)
	}
	_, _, err = client.ExportMaterialQuote(context.Background(), "quote")
	businessErr, ok := err.(*BusinessError)
	if !ok || businessErr.Code != 400 {
		t.Fatalf("error=%#v", err)
	}
}

func TestOutboundOrdersIncludesExactMaterialModel(t *testing.T) {
	var query string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"code":200,"msg":"成功","data":{"total":0,"list":[]}}`))
	}))
	defer server.Close()

	_, err := NewClient(server.URL).OutboundOrders(context.Background(), 1, 20, OutboundFilters{Model: "M8", IsPack: -1, IsWeigh: -1})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(query, "model=M8") {
		t.Fatalf("query=%q", query)
	}
}
