//go:build windows

package singleinstance

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/lxn/win"
	"golang.org/x/sys/windows"

	"zhengshi-wms-windowsapp/internal/localfile"
)

type Guard struct {
	handle  windows.Handle
	primary bool
	pid     uint32
	pidPath string
	once    sync.Once
}

const (
	activationRetryCount = 100
	activationRetryDelay = 50 * time.Millisecond
)

// Acquire creates a per-logon-session named mutex. The returned guard must be
// closed even when primary is false because CreateMutex still returns a handle
// for an existing mutex.
func Acquire(name string) (guard *Guard, primary bool, err error) {
	pidPath, err := instancePIDPath(name)
	if err != nil {
		return nil, false, err
	}
	wideName, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return nil, false, err
	}
	handle, createErr := windows.CreateMutex(nil, false, wideName)
	if handle == 0 {
		return nil, false, createErr
	}
	guard = &Guard{handle: handle, pidPath: pidPath}
	if errors.Is(createErr, windows.ERROR_ALREADY_EXISTS) {
		return guard, false, nil
	}
	if createErr != nil {
		guard.Close()
		return nil, false, createErr
	}
	guard.primary = true
	guard.pid = uint32(os.Getpid())
	if err := localfile.WriteAtomic(pidPath, []byte(strconv.FormatUint(uint64(guard.pid), 10)), 0o600); err != nil {
		guard.Close()
		return nil, false, fmt.Errorf("记录主实例进程失败: %w", err)
	}
	return guard, true, nil
}

// ActivatePrimary restores and foregrounds the primary process' first visible
// top-level window. A short retry window covers a second launch that races with
// creation of the login or main window.
func (g *Guard) ActivatePrimary() error {
	if g == nil || g.primary {
		return errors.New("当前实例不是可唤醒的次实例")
	}
	pid, err := readPID(g.pidPath)
	if err != nil {
		return fmt.Errorf("读取主实例进程失败: %w", err)
	}
	var target win.HWND
	for attempt := 0; attempt < activationRetryCount; attempt++ {
		target, err = visibleTopLevelWindow(pid)
		if err == nil && target != 0 {
			break
		}
		time.Sleep(activationRetryDelay)
	}
	if target == 0 {
		if err != nil {
			return err
		}
		return errors.New("未找到主实例的可见窗口")
	}
	if win.IsIconic(target) {
		win.ShowWindow(target, win.SW_RESTORE)
	} else {
		win.ShowWindow(target, win.SW_SHOW)
	}
	win.BringWindowToTop(target)
	win.SetForegroundWindow(target)
	return nil
}

func (g *Guard) Close() error {
	if g == nil {
		return nil
	}
	var result error
	g.once.Do(func() {
		if g.primary && g.pidPath != "" {
			if pid, err := readPID(g.pidPath); err == nil && pid == g.pid {
				if removeErr := os.Remove(g.pidPath); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
					result = removeErr
				}
			}
		}
		if g.handle != 0 {
			if closeErr := windows.CloseHandle(g.handle); result == nil {
				result = closeErr
			}
			g.handle = 0
		}
	})
	return result
}

func instancePIDPath(name string) (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir = filepath.Join(dir, "ZhengshiWMS", "instances")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	digest := sha256.Sum256([]byte(strings.TrimSpace(name)))
	return filepath.Join(dir, hex.EncodeToString(digest[:8])+".pid"), nil
}

func readPID(name string) (uint32, error) {
	data, err := os.ReadFile(name)
	if err != nil {
		return 0, err
	}
	value, err := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 32)
	if err != nil || value == 0 {
		return 0, errors.New("主实例进程标识无效")
	}
	return uint32(value), nil
}

func visibleTopLevelWindow(pid uint32) (win.HWND, error) {
	var target win.HWND
	callback := syscall.NewCallback(func(hwnd uintptr, _ uintptr) uintptr {
		if target != 0 {
			return 1
		}
		window := win.HWND(hwnd)
		if !win.IsWindowVisible(window) {
			return 1
		}
		var processID uint32
		win.GetWindowThreadProcessId(window, &processID)
		if processID == pid {
			target = window
		}
		return 1
	})
	if err := windows.EnumWindows(callback, nil); err != nil {
		return 0, err
	}
	return target, nil
}
