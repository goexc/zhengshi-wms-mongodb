//go:build windows

package securestore

import (
	"bytes"
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
