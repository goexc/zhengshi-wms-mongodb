//go:build windows

package securestore

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"
)

type CachedSession struct {
	Token      string `json:"token"`
	ExpiresAt  int64  `json:"expires_at"`
	Mobile     string `json:"mobile"`
	APIBaseURL string `json:"api_base_url"`
}

func path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir = filepath.Join(dir, "ZhengshiWMS")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return filepath.Join(dir, "session.dat"), nil
}

func Save(session CachedSession) error {
	if session.Token == "" || session.ExpiresAt == 0 || session.APIBaseURL == "" {
		return errors.New("缓存会话数据不完整")
	}
	plain, err := json.Marshal(session)
	if err != nil {
		return err
	}
	encrypted, err := protect(plain)
	if err != nil {
		return err
	}
	name, err := path()
	if err != nil {
		return err
	}
	return os.WriteFile(name, encrypted, 0o600)
}

func Load() (CachedSession, error) {
	var session CachedSession
	name, err := path()
	if err != nil {
		return session, err
	}
	encrypted, err := os.ReadFile(name)
	if err != nil {
		return session, err
	}
	plain, err := unprotect(encrypted)
	if err != nil {
		return session, err
	}
	if err := json.Unmarshal(plain, &session); err != nil {
		return session, err
	}
	if session.Token == "" || session.ExpiresAt == 0 || session.APIBaseURL == "" {
		return CachedSession{}, errors.New("缓存会话数据不完整")
	}
	return session, nil
}

func Delete() error {
	name, err := path()
	if err != nil {
		return err
	}
	err = os.Remove(name)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func protect(plain []byte) ([]byte, error) {
	input := bytesToBlob(plain)
	var output windows.DataBlob
	if err := windows.CryptProtectData(&input, nil, nil, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &output); err != nil {
		return nil, err
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(output.Data)))
	return append([]byte(nil), unsafe.Slice(output.Data, output.Size)...), nil
}

func unprotect(encrypted []byte) ([]byte, error) {
	input := bytesToBlob(encrypted)
	var output windows.DataBlob
	if err := windows.CryptUnprotectData(&input, nil, nil, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &output); err != nil {
		return nil, err
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(output.Data)))
	return append([]byte(nil), unsafe.Slice(output.Data, output.Size)...), nil
}

func bytesToBlob(data []byte) windows.DataBlob {
	if len(data) == 0 {
		return windows.DataBlob{}
	}
	return windows.DataBlob{Size: uint32(len(data)), Data: &data[0]}
}
