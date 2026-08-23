package diagnostics

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"golang.org/x/sys/windows"
)

// ExportSupportBundle creates a local-only support package containing the
// already-redacted diagnostic summary and bounded request logs. The package is
// committed atomically so a failed export does not replace an existing file.
func ExportSupportBundle(target, summary string) error {
	target = strings.TrimSpace(target)
	if target == "" {
		return fmt.Errorf("诊断包保存位置为空")
	}
	temporaryFile, err := os.CreateTemp(filepath.Dir(target), ".zhengshi-wms-diagnostics-*.partial")
	if err != nil {
		return err
	}
	temporary := temporaryFile.Name()
	committed := false
	defer func() {
		_ = temporaryFile.Close()
		if !committed {
			_ = os.Remove(temporary)
		}
	}()

	archive := zip.NewWriter(temporaryFile)
	if err := addBundleText(archive, "diagnostic-summary.txt", summary); err != nil {
		_ = archive.Close()
		return err
	}
	logPath, pathErr := Path()
	if pathErr == nil {
		for index := 0; index <= defaultLogBackups; index++ {
			name := logPath
			if index > 0 {
				name = fmt.Sprintf("%s.%d", logPath, index)
			}
			if err := addBundleFile(archive, name); err != nil && !os.IsNotExist(err) {
				_ = archive.Close()
				return err
			}
		}
	}
	if err := archive.Close(); err != nil {
		return err
	}
	if err := temporaryFile.Sync(); err != nil {
		return err
	}
	if err := temporaryFile.Close(); err != nil {
		return err
	}
	from, err := windows.UTF16PtrFromString(temporary)
	if err != nil {
		return err
	}
	to, err := windows.UTF16PtrFromString(target)
	if err != nil {
		return err
	}
	if err := windows.MoveFileEx(from, to, windows.MOVEFILE_REPLACE_EXISTING|windows.MOVEFILE_WRITE_THROUGH); err != nil {
		return fmt.Errorf("提交诊断包失败: %w", err)
	}
	runtime.KeepAlive(from)
	runtime.KeepAlive(to)
	committed = true
	return nil
}

func addBundleText(archive *zip.Writer, name, text string) error {
	header := &zip.FileHeader{Name: name, Method: zip.Deflate}
	header.SetModTime(time.Now())
	header.SetMode(0o600)
	writer, err := archive.CreateHeader(header)
	if err != nil {
		return err
	}
	_, err = io.WriteString(writer, text)
	return err
}

func addBundleFile(archive *zip.Writer, source string) error {
	file, err := os.Open(source)
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	header := &zip.FileHeader{Name: filepath.Base(source), Method: zip.Deflate}
	header.SetModTime(info.ModTime())
	header.SetMode(0o600)
	writer, err := archive.CreateHeader(header)
	if err != nil {
		return err
	}
	_, err = io.CopyN(writer, file, info.Size())
	return err
}
