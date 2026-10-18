package permissionx

import (
	"net/http"
	"testing"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
)

func TestPreviewCannotBeGrantedByReadPermission(t *testing.T) {
	m, err := model.NewModelFromString(`[request_definition]
r = sub, obj, act
[policy_definition]
p = sub, obj, act
[policy_effect]
e = some(where (p.eft == allow))
[matchers]
m = r.sub == p.sub && r.obj == p.obj && r.act == p.act`)
	if err != nil {
		t.Fatal(err)
	}
	enforcer, err := casbin.NewEnforcer(m)
	if err != nil {
		t.Fatal(err)
	}
	for _, rule := range [][]string{
		{"deleter", "/outbound", http.MethodDelete},
		{"reader", "/outbound/page", http.MethodGet},
		{"preview-only", "/outbound/remove-preview", http.MethodGet},
	} {
		if _, err = enforcer.AddPolicy(rule); err != nil {
			t.Fatal(err)
		}
	}
	path, method := Target("/outbound/remove-preview", http.MethodGet)
	for _, user := range []string{"deleter", "reader", "preview-only", "unknown"} {
		allowed, err := enforcer.Enforce(user, path, method)
		if err != nil || allowed != (user == "deleter") {
			t.Fatalf("preview permission user=%s allowed=%v error=%v", user, allowed, err)
		}
	}
}

func TestDeletePreviewUsesDeletePermissionOnly(t *testing.T) {
	for _, tt := range []struct{ path, method, wantPath, wantMethod string }{
		{"/outbound/remove-preview", http.MethodGet, "/outbound", http.MethodDelete},
		{"/outbound/remove-preview", http.MethodPost, "/outbound/remove-preview", http.MethodPost},
		{"/outbound/page", http.MethodGet, "/outbound/page", http.MethodGet},
		{"/outbound", http.MethodDelete, "/outbound", http.MethodDelete},
	} {
		path, method := Target(tt.path, tt.method)
		if path != tt.wantPath || method != tt.wantMethod {
			t.Fatalf("Target(%q,%q) = %q,%q; want %q,%q", tt.path, tt.method, path, method, tt.wantPath, tt.wantMethod)
		}
	}
}
