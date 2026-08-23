package ui

import (
	"strings"
	"testing"

	"zhengshi-wms-windowsapp/internal/api"
)

func TestImageAssetMenuUsesOnlyGrantedMenuTree(t *testing.T) {
	perms := api.Perms{Menus: []api.Menu{{Name: "系统管理", Children: []api.Menu{{Name: "图片素材", Path: "/image"}}}}}
	if !imageAssetMenuAvailable(perms) {
		t.Fatal("expected granted image menu to enable image assets")
	}
	if imageAssetMenuAvailable(api.Perms{Menus: []api.Menu{{Name: "物料管理", Path: "/material/list"}}}) {
		t.Fatal("unrelated menu must not enable image assets")
	}
}

func TestAppendUniqueReferencesPreservesOrderAndLimit(t *testing.T) {
	got := appendUniqueReferences([]string{"a.png", "b.png"}, []string{" b.png ", "c.png", "d.png"}, 3)
	want := []string{"a.png", "b.png", "c.png"}
	if len(got) != len(want) {
		t.Fatalf("references = %#v", got)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("references = %#v", got)
		}
	}
}

func TestManualTransactionIdempotencyKeyIsCustomerScoped(t *testing.T) {
	first := newManualTransactionIdempotencyKey(" customer ")
	second := newManualTransactionIdempotencyKey("customer")
	if !strings.HasPrefix(first, "manual:customer:") {
		t.Fatalf("key = %q", first)
	}
	if first == second {
		t.Fatal("new dialogs must receive distinct idempotency keys")
	}
}
