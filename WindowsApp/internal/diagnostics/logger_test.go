package diagnostics

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoggerRotatesBoundedFiles(t *testing.T) {
	dir := t.TempDir()
	name := filepath.Join(dir, "windowsapp.log")
	if err := os.WriteFile(name, []byte(strings.Repeat("x", 32)), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name+".1", []byte("older"), 0o600); err != nil {
		t.Fatal(err)
	}
	logger, err := newAt(name, 16, 2)
	if err != nil {
		t.Fatal(err)
	}
	logger.Printf("new")
	if err := logger.Close(); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(name + ".1"); err != nil || len(data) != 32 {
		t.Fatalf("first backup len=%d err=%v", len(data), err)
	}
	if data, err := os.ReadFile(name + ".2"); err != nil || string(data) != "older" {
		t.Fatalf("second backup=%q err=%v", data, err)
	}
}

func TestIncidentIDDoesNotContainWhitespace(t *testing.T) {
	id := IncidentID()
	if !strings.HasPrefix(id, "WAPP-") || strings.ContainsAny(id, " \r\n\t") {
		t.Fatalf("incident id = %q", id)
	}
}

func TestRecordBackgroundPanicUsesDefaultLoggerWithoutPanicText(t *testing.T) {
	name := filepath.Join(t.TempDir(), "background.log")
	logger, err := newAt(name, 1<<20, 1)
	if err != nil {
		t.Fatal(err)
	}
	SetDefaultLogger(logger)
	t.Cleanup(func() {
		SetDefaultLogger(nil)
		_ = logger.Close()
	})

	id := RecordBackgroundPanic("query_export_actions.go:65", "sensitive business value")
	if !strings.HasPrefix(id, "WAPP-") {
		t.Fatalf("incident id = %q", id)
	}
	data, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, `task="query_export_actions.go:65"`) || !strings.Contains(text, "recovered_type=string") {
		t.Fatalf("background panic log = %q", text)
	}
	if strings.Contains(text, "sensitive business value") {
		t.Fatalf("panic text leaked to log: %q", text)
	}
}

func TestLoggerRotatesWhileRunning(t *testing.T) {
	name := filepath.Join(t.TempDir(), "runtime.log")
	logger, err := newAt(name, 32, 2)
	if err != nil {
		t.Fatal(err)
	}
	logger.Printf("%s", strings.Repeat("a", 64))
	logger.Printf("after-rotation")
	if err := logger.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(name + ".1"); err != nil {
		t.Fatalf("runtime backup missing: %v", err)
	}
	data, err := os.ReadFile(name)
	if err != nil || !strings.Contains(string(data), "after-rotation") {
		t.Fatalf("current log=%q err=%v", data, err)
	}
}
