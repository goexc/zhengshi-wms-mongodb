package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestClientUsesAccountAndMasterDataWriteContracts(t *testing.T) {
	var requests []string
	var bodies []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		if r.Body != nil {
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			bodies = append(bodies, body)
		}
		_, _ = w.Write([]byte(`{"code":200,"msg":"成功"}`))
	}))
	defer server.Close()

	ctx := context.Background()
	client := NewClient(server.URL)
	if err := client.ChangePassword(ctx, "new-secret"); err != nil {
		t.Fatal(err)
	}
	if err := client.CreateSupplier(ctx, SupplierRequest{ID: "must-clear", Type: "企业", Level: 3, Code: "SUP-001", Name: "供应商"}); err != nil {
		t.Fatal(err)
	}
	if err := client.UpdateCustomer(ctx, CustomerRequest{ID: "customer", Type: "企业", Code: "CUS-001", Name: "客户"}); err != nil {
		t.Fatal(err)
	}
	if err := client.UpdateCarrierStatus(ctx, "carrier", "活跃"); err != nil {
		t.Fatal(err)
	}
	if err := client.CreateWarehouse(ctx, WarehouseRequest{ID: "must-clear", Type: "生产仓库", Code: "WH-001", Name: "一号仓"}); err != nil {
		t.Fatal(err)
	}
	if err := client.UpdateWarehouseZone(ctx, WarehouseZoneRequest{ID: "zone", WarehouseID: "warehouse", Code: "ZONE-1", Name: "原料区", Image: "zone.png"}); err != nil {
		t.Fatal(err)
	}
	if err := client.UpdateWarehouseRackStatus(ctx, "rack", "盘点中"); err != nil {
		t.Fatal(err)
	}
	if err := client.CreateWarehouseBin(ctx, WarehouseBinRequest{ID: "must-clear", WarehouseRackID: "rack", Code: "BIN-1", Name: "A01"}); err != nil {
		t.Fatal(err)
	}

	want := []string{
		"PATCH /account/password",
		"POST /supplier",
		"PUT /customer",
		"PATCH /carrier/status",
		"POST /warehouse",
		"PUT /warehouse_zone",
		"PATCH /warehouse_rack/status",
		"POST /warehouse_bin",
	}
	if !reflect.DeepEqual(requests, want) {
		t.Fatalf("requests = %#v, want %#v", requests, want)
	}
	if got := bodies[0]["password"]; got != "new-secret" {
		t.Fatalf("password payload = %#v", got)
	}
	if _, exists := bodies[1]["id"]; exists {
		t.Fatalf("create supplier must omit id: %#v", bodies[1])
	}
	if got := bodies[2]["id"]; got != "customer" {
		t.Fatalf("update customer id = %#v", got)
	}
	if _, exists := bodies[4]["id"]; exists {
		t.Fatalf("create warehouse must omit id: %#v", bodies[4])
	}
	if _, exists := bodies[7]["id"]; exists {
		t.Fatalf("create warehouse bin must omit id: %#v", bodies[7])
	}
}

func TestClientFindsMasterDataByExactID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/supplier":
			_, _ = w.Write([]byte(`{"code":200,"msg":"成功","data":{"total":2,"list":[{"id":"other","code":"SUP"},{"id":"target","code":"SUP","updated_at":9}]}}`))
		case "/warehouse_bin":
			_, _ = w.Write([]byte(`{"code":200,"msg":"成功","data":{"total":1,"list":[{"id":"bin","code":"BIN","image":"bin.png"}]}}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL)
	supplier, found, err := client.FindSupplier(context.Background(), "target", "SUP")
	if err != nil || !found || supplier.UpdatedAt != 9 {
		t.Fatalf("supplier=%#v found=%v err=%v", supplier, found, err)
	}
	bin, found, err := client.FindWarehouseBin(context.Background(), "bin", "BIN")
	if err != nil || !found || bin.Image != "bin.png" {
		t.Fatalf("bin=%#v found=%v err=%v", bin, found, err)
	}
}

func TestProfileDecodesPersonalCenterFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"code":200,"msg":"成功","data":{"name":"张三","sex":"男","department_id":"dep","department_name":"仓储部","mobile":"18800000000","email":"user@example.com","status":"启用","avatar":"avatar.png","remark":"值班","created_at":1,"updated_at":2}}`))
	}))
	defer server.Close()

	profile, err := NewClient(server.URL).Profile(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if profile.Sex != "男" || profile.DepartmentID != "dep" || profile.Email != "user@example.com" || profile.Avatar != "avatar.png" || profile.Remark != "值班" {
		t.Fatalf("profile = %#v", profile)
	}
}
