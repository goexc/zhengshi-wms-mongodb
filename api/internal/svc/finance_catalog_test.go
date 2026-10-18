package svc

import "testing"

func TestFinanceAPICatalogHasEveryAction(t *testing.T) {
	entries := map[string]bool{}
	ids := map[string]bool{}
	for _, entry := range financeAPIEntries() {
		key := entry.Method + " " + entry.Uri
		if entries[key] {
			t.Fatalf("duplicate API %s", key)
		}
		entries[key] = true
		id := financeCatalogID("api:" + entry.Method + ":" + entry.Uri).Hex()
		if ids[id] {
			t.Fatal("catalog ID collision")
		}
		ids[id] = true
	}
	for _, key := range []string{"GET /customer", "GET /customer/list", "GET /customer/acceptance", "GET /customer/acceptance/detail", "GET /customer/acceptance/payments", "POST /customer/acceptance/redeem", "POST /customer/acceptance/reverse", "POST /customer/acceptance/void", "PUT /customer/invoice", "POST /customer/invoice/void", "POST /images"} {
		if !entries[key] {
			t.Fatalf("missing permission entry %s", key)
		}
	}
}
