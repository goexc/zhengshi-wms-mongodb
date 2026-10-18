package attachmentx

import (
	"strings"
	"testing"
	"time"
)

func TestStorageContract(t *testing.T) {
	now := time.Date(2026, 10, 18, 0, 0, 0, 0, time.UTC)
	for _, tc := range []struct{ scope, prefix string }{
		{"", ""}, {"acceptance", "acceptance/2026/10/"},
		{"acceptance_redemption", "acceptance/redemption/2026/10/"}, {"invoice", "invoice/2026/10/"},
	} {
		key, err := NewReference(tc.scope, "凭证.PDF", now)
		if err != nil || !strings.HasPrefix(key, tc.prefix) || !strings.HasSuffix(key, ".pdf") {
			t.Fatalf("%s: %s %v", tc.scope, key, err)
		}
		if err = ValidateReference(key); err != nil {
			t.Fatal(err)
		}
		if Scope(key) != tc.scope {
			t.Fatalf("wrong scope: %s", key)
		}
		if strings.Contains("images/"+key, "images/images/") {
			t.Fatal("double prefix")
		}
	}
	if _, err := NewReference("../../", "a.pdf", now); err == nil {
		t.Fatal("accepted arbitrary directory")
	}
	if _, err := NewReference("invoice", "a.exe", now); err == nil {
		t.Fatal("accepted executable")
	}
}

func TestLegacyAndUnsafeReferences(t *testing.T) {
	got, err := ValidateReferences([]string{"legacy.png", "legacy.png", "invoice/2026/10/file.pdf"})
	if err != nil || len(got) != 2 {
		t.Fatalf("%v %v", got, err)
	}
	for _, key := range []string{"", "../file", "a/../b", "/a", "images/a", "https://example.com/a", "a,b", "a\\b", "a%2fb", "a//b"} {
		if ValidateReference(key) == nil {
			t.Fatalf("accepted unsafe reference %q", key)
		}
	}
	if _, err := ValidateReferences(make([]string, 11)); err == nil {
		t.Fatal("accepted excess attachments")
	}
	if got, err := ValidateReferences(nil); err != nil || got == nil || len(got) != 0 {
		t.Fatal("optional empty attachments not supported")
	}
}

func TestScopedBindingsKeepLegacyButRejectCrossUse(t *testing.T) {
	keys := map[string]string{
		"acceptance":            "acceptance/2026/10/a.pdf",
		"acceptance_redemption": "acceptance/redemption/2026/10/a.pdf",
		"invoice":               "invoice/2026/10/a.pdf",
	}
	for scope, own := range keys {
		if err := ValidateScopedReferences([]string{"legacy.png", own}, scope, nil); err != nil {
			t.Fatalf("valid new and legacy refs in %s: %v", scope, err)
		}
		for other, ref := range keys {
			if scope == other {
				continue
			}
			if err := ValidateScopedReferences([]string{ref}, scope, nil); err == nil {
				t.Fatalf("accepted new %s file as %s", other, scope)
			}
			if err := ValidateScopedReferences([]string{ref}, scope, []string{ref}); err != nil {
				t.Fatalf("rewrote historical ref %s: %v", ref, err)
			}
		}
	}
	if err := ValidateScopedReferences(nil, "unknown", nil); err == nil {
		t.Fatal("accepted unknown use")
	}
}
