package ui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"golang.org/x/sys/windows"
)

func writeQueryExportXLSXAtomic(ctx context.Context, target, title, filters, source string, columns []queryExportColumn, rows [][]any, queriedAt time.Time) error {
	return writeAtomicTarget(ctx, target, func(temporary string) error {
		return writeQueryExportXLSX(temporary, title, filters, source, columns, rows, queriedAt)
	})
}

func writeFileAtomic(ctx context.Context, target string, data []byte, mode os.FileMode) error {
	return writeAtomicTarget(ctx, target, func(temporary string) error {
		return os.WriteFile(temporary, data, mode)
	})
}

func writeAtomicTarget(ctx context.Context, target string, write func(string) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	directory := filepath.Dir(target)
	temporaryFile, err := os.CreateTemp(directory, ".zhengshi-wms-*.partial")
	if err != nil {
		return err
	}
	temporary := temporaryFile.Name()
	if closeErr := temporaryFile.Close(); closeErr != nil {
		_ = os.Remove(temporary)
		return closeErr
	}
	_ = os.Remove(temporary)
	defer os.Remove(temporary)
	if err := write(temporary); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
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
		return fmt.Errorf("提交导出文件失败: %w", err)
	}
	runtime.KeepAlive(from)
	runtime.KeepAlive(to)
	return nil
}
