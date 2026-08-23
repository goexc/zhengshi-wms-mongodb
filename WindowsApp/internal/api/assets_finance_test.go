package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestImageAssetsUsesExistingPageContract(t *testing.T) {
	var gotMethod, gotQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"code":200,"msg":"成功","data":{"total":1,"list":[{"url":"asset.png","alt":"装配图"}]}}`))
	}))
	defer server.Close()

	page, err := NewClient(server.URL).ImageAssets(context.Background(), 2, 20, " 装配 ")
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodGet || gotQuery != "name=%E8%A3%85%E9%85%8D&page=2&size=20" {
		t.Fatalf("request = %s ?%s", gotMethod, gotQuery)
	}
	if page.Total != 1 || len(page.List) != 1 || page.List[0].URL != "asset.png" || page.List[0].Alt != "装配图" {
		t.Fatalf("page = %#v", page)
	}
}

func TestAddCustomerTransactionUsesExactContract(t *testing.T) {
	var gotMethod, gotPath string
	var got CustomerTransactionAddRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(`{"code":200,"msg":"成功"}`))
	}))
	defer server.Close()

	want := CustomerTransactionAddRequest{
		CustomerID: "customer", Time: 1_700_000_000, TransactionType: CustomerTransactionPayment,
		IdempotencyKey: "manual:customer:uuid", Amount: 12.5, Annex: []string{"a.png"}, Remark: "回款",
	}
	request := want
	request.CustomerID = " customer "
	request.TransactionType = " payment "
	request.IdempotencyKey = " manual:customer:uuid "
	request.Annex = []string{" a.png ", " "}
	request.Remark = " 回款 "
	if err := NewClient(server.URL).AddCustomerTransaction(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodPost || gotPath != "/customer/transaction" {
		t.Fatalf("request = %s %s", gotMethod, gotPath)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("payload = %#v, want %#v", got, want)
	}
}
