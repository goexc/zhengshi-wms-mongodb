package diagnostics

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExportSupportBundleContainsSummary(t *testing.T) {
	target := filepath.Join(t.TempDir(), "support.zip")
	if err := ExportSupportBundle(target, "redacted summary"); err != nil {
		t.Fatal(err)
	}
	archive, err := zip.OpenReader(target)
	if err != nil {
		t.Fatal(err)
	}
	defer archive.Close()
	if len(archive.File) == 0 || archive.File[0].Name != "diagnostic-summary.txt" {
		t.Fatalf("entries = %#v", archive.File)
	}
	reader, err := archive.File[0].Open()
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(reader)
	_ = reader.Close()
	if err != nil || string(data) != "redacted summary" {
		t.Fatalf("summary = %q err=%v", data, err)
	}
	matches, err := filepath.Glob(filepath.Join(filepath.Dir(target), ".zhengshi-wms-diagnostics-*.partial"))
	if err != nil || len(matches) != 0 {
		t.Fatalf("partial files = %#v err=%v", matches, err)
	}
	if info, err := os.Stat(target); err != nil || info.Size() == 0 || strings.ToLower(filepath.Ext(info.Name())) != ".zip" {
		t.Fatalf("bundle info=%v err=%v", info, err)
	}
}
