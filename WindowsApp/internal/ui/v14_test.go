package ui

import (
	"reflect"
	"testing"

	"zhengshi-wms-windowsapp/internal/api"
	"zhengshi-wms-windowsapp/internal/config"
)

func TestTopLevelMenusFollowPermissionMatrix(t *testing.T) {
	minimal := availableTopLevelMenuSpecs(api.Perms{}, false, false, false)
	if got := menuSpecKeys(minimal); !reflect.DeepEqual(got, []string{"global_lookup", "operations", "profile", "system"}) {
		t.Fatalf("minimal keys = %#v", got)
	}

	perms := api.Perms{Menus: []api.Menu{
		{Path: "/material/list"}, {Path: "/inventory/index"}, {Path: "/inbound/receipt"},
		{Path: "/outbound/receipt"}, {Path: "/outbound/report"}, {Path: "/acl/user"},
	}}
	full := availableTopLevelMenuSpecs(perms, true, true, true)
	for _, key := range []string{"dashboard", "material", "inventory", "inbound", "outbound", "outbound_report", "partner", "warehouse", "admin"} {
		if stringIndex(menuSpecKeys(full), key) < 0 {
			t.Errorf("expected permitted menu %q in %#v", key, full)
		}
	}
}

func TestInitialTopLevelPagesAreLazyAndSessionScoped(t *testing.T) {
	ui := &mainUI{
		menuKeys: []string{"dashboard", "global_lookup", "material", "operations", "profile", "system"},
		cfg: config.Config{APIBaseURL: "https://api.example", Workspace: config.WorkspaceState{
			APIBaseURL: "https://api.example", Mobile: "18800000000",
			OpenPages: []string{"material", "system"}, CurrentPage: "material",
		}},
		session: &Session{Login: api.LoginData{Mobile: "18800000000"}},
	}
	if got := ui.initialTopLevelPageKeys(); !reflect.DeepEqual(got, []string{"material"}) {
		t.Fatalf("restored keys = %#v", got)
	}
	ui.session.Login.Mobile = "19900000000"
	if got := ui.initialTopLevelPageKeys(); !reflect.DeepEqual(got, []string{"dashboard"}) {
		t.Fatalf("other account keys = %#v", got)
	}
	ui.menuKeys = []string{"global_lookup", "operations", "profile", "system"}
	if got := ui.initialTopLevelPageKeys(); !reflect.DeepEqual(got, []string{"global_lookup"}) {
		t.Fatalf("fallback keys = %#v", got)
	}
}

func menuSpecKeys(specs []topLevelMenuSpec) []string {
	keys := make([]string, 0, len(specs))
	for _, spec := range specs {
		keys = append(keys, spec.Key)
	}
	return keys
}
