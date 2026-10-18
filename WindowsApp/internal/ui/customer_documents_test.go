package ui

import (
	"testing"
	"time"
	"zhengshi-wms-windowsapp/internal/api"
)

func TestDocumentMenusRequireExactFinancePermissions(t *testing.T) {
	for _, tc := range []struct {
		perms               []string
		acceptance, invoice bool
	}{{[]string{"business_partner:customer:list"}, false, false}, {[]string{"finance:acceptance:add"}, false, false}, {[]string{"finance:acceptance:list"}, true, false}, {[]string{"finance:invoice:list"}, false, true}, {[]string{"finance:acceptance:list", "finance:invoice:list"}, true, true}} {
		perms := api.Perms{}
		for _, p := range tc.perms {
			perms.Buttons = append(perms.Buttons, api.Button{Perms: p})
		}
		found := map[string]bool{}
		for _, s := range availableTopLevelMenuSpecs(perms, false, false, false) {
			found[s.Key] = true
		}
		if found["customer_acceptance"] != tc.acceptance || found["customer_invoice"] != tc.invoice {
			t.Fatalf("permission leak/missing menu for %#v: %#v", tc, found)
		}
	}
}
func TestAcceptanceOverdueUsesBeijingCalendarAndNeverChangesStatus(t *testing.T) {
	zone := time.FixedZone("Beijing", 8*3600)
	due := time.Date(2026, 10, 17, 12, 30, 0, 0, zone).Unix()
	today := time.Date(2026, 10, 17, 23, 59, 59, 0, zone).Unix()
	tomorrow := time.Date(2026, 10, 18, 0, 0, 0, 0, zone).Unix()
	if documentIsOverdue(due, today) {
		t.Fatal("ticket due today should not be overdue")
	}
	if !documentIsOverdue(due, tomorrow) {
		t.Fatal("ticket must become overdue at next Beijing day boundary")
	}
	if documentStatus("redeemed", 1) != "已到账" || documentStatus("voided", 1) != "已作废" {
		t.Fatal("terminal status must not become overdue")
	}
}
