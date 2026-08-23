package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestFastOutboundUsesExistingTransactionContract(t *testing.T) {
	var gotMethod, gotPath string
	var got FastOutboundRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(`{"code":200,"msg":"成功"}`))
	}))
	defer server.Close()

	want := FastOutboundRequest{
		Code: "O-001", Type: FastOutboundSale, CustomerID: "customer", DepartureTime: 100,
		PickingTime: 80, ReceiptTime: 100,
		Materials: []FastOutboundMaterialRequest{{MaterialID: "material", Price: 12.5, Quantity: 2}},
	}
	request := want
	request.Materials = append([]FastOutboundMaterialRequest(nil), want.Materials...)
	request.Code = " O-001 "
	request.Type = " 销售出库 "
	request.CustomerID = " customer "
	request.Materials[0].MaterialID = " material "
	if err := NewClient(server.URL).FastOutbound(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodPost || gotPath != "/outbound/fast_departure" {
		t.Fatalf("request = %s %s", gotMethod, gotPath)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("payload = %#v, want %#v", got, want)
	}
}

func TestAdminUsersUsesExistingPageContract(t *testing.T) {
	var gotMethod, gotPath, gotQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath, gotQuery = r.Method, r.URL.Path, r.URL.RawQuery
		_, _ = w.Write([]byte(`{"code":200,"msg":"成功","data":{"total":1,"list":[{"id":"u1","name":"管理员"}]}}`))
	}))
	defer server.Close()

	page, err := NewClient(server.URL).AdminUsers(context.Background(), 2, 20, " 管理 ", " 188 ")
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodGet || gotPath != "/user" || gotQuery != "mobile=188&name=%E7%AE%A1%E7%90%86&page=2&size=20" {
		t.Fatalf("request = %s %s?%s", gotMethod, gotPath, gotQuery)
	}
	if page.Total != 1 || len(page.List) != 1 || page.List[0].ID != "u1" {
		t.Fatalf("page = %#v", page)
	}
}

func TestRoleAssignmentsUseExactExistingContracts(t *testing.T) {
	type captured struct {
		Method string
		Path   string
		Body   map[string]any
	}
	var requests []captured
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		item := captured{Method: r.Method, Path: r.URL.Path}
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&item.Body)
		}
		requests = append(requests, item)
		_, _ = w.Write([]byte(`{"code":200,"msg":"成功"}`))
	}))
	defer server.Close()
	client := NewClient(server.URL)
	if err := client.SetAdminRoleMenuIDs(context.Background(), " role ", []string{" menu-1 ", "menu-1", "menu-2"}); err != nil {
		t.Fatal(err)
	}
	if err := client.SetAdminRoleAPIIDs(context.Background(), " role ", []string{" api-1 ", "api-2"}); err != nil {
		t.Fatal(err)
	}
	if len(requests) != 2 || requests[0].Method != http.MethodPost || requests[0].Path != "/role/menus" || requests[1].Path != "/role/apis" {
		t.Fatalf("requests = %#v", requests)
	}
	if !reflect.DeepEqual(requests[0].Body["menus_id"], []any{"menu-1", "menu-2"}) {
		t.Fatalf("menu body = %#v", requests[0].Body)
	}
	if !reflect.DeepEqual(requests[1].Body["apis_id"], []any{"api-1", "api-2"}) {
		t.Fatalf("api body = %#v", requests[1].Body)
	}
}

func TestAdminCatalogReadContracts(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.Method+" "+r.URL.Path)
		_, _ = w.Write([]byte(`{"code":200,"msg":"成功","data":[]}`))
	}))
	defer server.Close()
	client := NewClient(server.URL)
	if _, err := client.AdminDepartments(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := client.AdminMenus(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := client.AdminAPIs(context.Background()); err != nil {
		t.Fatal(err)
	}
	want := []string{"GET /department", "GET /menu/list", "GET /api"}
	if !reflect.DeepEqual(paths, want) {
		t.Fatalf("paths = %#v, want %#v", paths, want)
	}
}
