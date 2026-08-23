//go:build windows

package securestore

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestDPAPIRoundTrip(t *testing.T) {
	plain := []byte(`{"token":"secret-token","expires_at":1893456000}`)
	encrypted, err := protect(plain)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(encrypted, []byte("secret-token")) {
		t.Fatal("encrypted data contains plaintext token")
	}
	decrypted, err := unprotect(encrypted)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(decrypted, plain) {
		t.Fatalf("decrypted = %q", decrypted)
	}
}

func TestSealRejectsEmptyData(t *testing.T) {
	if _, err := Seal(nil); err == nil {
		t.Fatal("expected empty data error")
	}
}

func TestCachedSessionCorruptPrimaryRecoversEncryptedBackup(t *testing.T) {
	name := filepath.Join(t.TempDir(), "session.dat")
	first := CachedSession{Token: "first-secret", ExpiresAt: 1893456000, Mobile: "18810509066", APIBaseURL: "https://example.invalid"}
	if err := saveAt(name, first); err != nil {
		t.Fatal(err)
	}
	second := first
	second.Token = "second-secret"
	if err := saveAt(name, second); err != nil {
		t.Fatal(err)
	}
	backup, err := os.ReadFile(name + ".bak")
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(backup, []byte(first.Token)) {
		t.Fatal("backup contains plaintext token")
	}
	if err := os.WriteFile(name, []byte("corrupt"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := loadAt(name)
	if err != nil {
		t.Fatal(err)
	}
	if got.Token != first.Token {
		t.Fatalf("recovered token = %q", got.Token)
	}
}
