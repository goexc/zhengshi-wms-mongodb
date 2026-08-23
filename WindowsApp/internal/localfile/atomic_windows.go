//go:build windows

package localfile

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
)

const BackupSuffix = ".bak"

// WriteAtomic writes data to a same-directory temporary file, flushes it, and
// atomically replaces the destination. A failed write leaves the old file in
// place and removes the temporary file.
func WriteAtomic(name string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(name), 0o700); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(name), ".wms-state-*.tmp")
	if err != nil {
		return err
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	closeWithError := func(writeErr error) error {
		if closeErr := temporary.Close(); writeErr == nil {
			return closeErr
		}
		return writeErr
	}
	if err := temporary.Chmod(mode); err != nil {
		return closeWithError(err)
	}
	if _, err := temporary.Write(data); err != nil {
		return closeWithError(err)
	}
	if err := temporary.Sync(); err != nil {
		return closeWithError(err)
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	source, err := windows.UTF16PtrFromString(temporaryName)
	if err != nil {
		return err
	}
	target, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return err
	}
	return windows.MoveFileEx(source, target, windows.MOVEFILE_REPLACE_EXISTING|windows.MOVEFILE_WRITE_THROUGH)
}

// WriteAtomicWithBackup preserves the current destination as a backup only
// when it passes validation. This prevents a corrupt primary file from
// overwriting the last known-good backup.
func WriteAtomicWithBackup(name string, data []byte, mode os.FileMode, validate func([]byte) error) error {
	current, err := os.ReadFile(name)
	if err == nil && validate(current) == nil {
		if err := WriteAtomic(name+BackupSuffix, current, mode); err != nil {
			return fmt.Errorf("保存本地状态备份失败: %w", err)
		}
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return WriteAtomic(name, data, mode)
}

// ReadWithBackup reads and validates the primary file, falling back to the
// backup when the primary is missing or corrupt. A recovered primary is
// restored on a best-effort basis for the next launch.
func ReadWithBackup(name string, mode os.FileMode, validate func([]byte) error) ([]byte, bool, error) {
	primary, primaryErr := readValid(name, validate)
	if primaryErr == nil {
		return primary, false, nil
	}
	backup, backupErr := readValid(name+BackupSuffix, validate)
	if backupErr != nil {
		return nil, false, errors.Join(primaryErr, backupErr)
	}
	_ = WriteAtomic(name, backup, mode)
	return backup, true, nil
}

func readValid(name string, validate func([]byte) error) ([]byte, error) {
	data, err := os.ReadFile(name)
	if err != nil {
		return nil, err
	}
	if err := validate(data); err != nil {
		return nil, fmt.Errorf("%s 内容无效: %w", filepath.Base(name), err)
	}
	return data, nil
}
