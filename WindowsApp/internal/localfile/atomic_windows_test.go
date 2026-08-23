//go:build windows

package localfile

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func requireNonEmpty(data []byte) error {
	if len(bytes.TrimSpace(data)) == 0 {
		return errors.New("empty")
	}
	return nil
}

func TestWriteAtomicWithBackupAndRecover(t *testing.T) {
	name := filepath.Join(t.TempDir(), "state.json")
	if err := WriteAtomicWithBackup(name, []byte("first"), 0o600, requireNonEmpty); err != nil {
		t.Fatal(err)
	}
	if err := WriteAtomicWithBackup(name, []byte("second"), 0o600, requireNonEmpty); err != nil {
		t.Fatal(err)
	}
	backup, err := os.ReadFile(name + BackupSuffix)
	if err != nil {
		t.Fatal(err)
	}
	if string(backup) != "first" {
		t.Fatalf("backup = %q", backup)
	}
	if err := os.WriteFile(name, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	recovered, usedBackup, err := ReadWithBackup(name, 0o600, requireNonEmpty)
	if err != nil {
		t.Fatal(err)
	}
	if !usedBackup || string(recovered) != "first" {
		t.Fatalf("usedBackup = %v, recovered = %q", usedBackup, recovered)
	}
	restored, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	if string(restored) != "first" {
		t.Fatalf("restored = %q", restored)
	}
}

func TestCorruptPrimaryDoesNotReplaceGoodBackup(t *testing.T) {
	name := filepath.Join(t.TempDir(), "state.json")
	if err := WriteAtomic(name+BackupSuffix, []byte("known-good"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := WriteAtomicWithBackup(name, []byte("new-good"), 0o600, requireNonEmpty); err != nil {
		t.Fatal(err)
	}
	backup, err := os.ReadFile(name + BackupSuffix)
	if err != nil {
		t.Fatal(err)
	}
	if string(backup) != "known-good" {
		t.Fatalf("backup = %q", backup)
	}
	matches, err := filepath.Glob(filepath.Join(filepath.Dir(name), ".wms-state-*.tmp"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("temporary files remain: %v", matches)
	}
}
