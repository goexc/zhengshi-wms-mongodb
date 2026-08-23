//go:build windows

package securestore

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"

	"zhengshi-wms-windowsapp/internal/localfile"
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
	name, err := path()
	if err != nil {
		return err
	}
	return saveAt(name, session)
}

func saveAt(name string, session CachedSession) error {
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
	return localfile.WriteAtomicWithBackup(name, encrypted, 0o600, validateEncryptedSession)
}

func Load() (CachedSession, error) {
	name, err := path()
	if err != nil {
		return CachedSession{}, err
	}
	return loadAt(name)
}

func loadAt(name string) (CachedSession, error) {
	encrypted, _, err := localfile.ReadWithBackup(name, 0o600, validateEncryptedSession)
	if err != nil {
		return CachedSession{}, err
	}
	return decodeEncryptedSession(encrypted)
}

func validateEncryptedSession(encrypted []byte) error {
	_, err := decodeEncryptedSession(encrypted)
	return err
}

func decodeEncryptedSession(encrypted []byte) (CachedSession, error) {
	var session CachedSession
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
	var result error
	for _, candidate := range []string{name, name + localfile.BackupSuffix} {
		if removeErr := os.Remove(candidate); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			result = errors.Join(result, removeErr)
		}
	}
	return result
}

// Seal protects non-session local state with the same current-Windows-user
// DPAPI boundary used by the cached login session.
func Seal(plain []byte) ([]byte, error) {
	if len(plain) == 0 {
		return nil, errors.New("待加密数据为空")
	}
	return protect(plain)
}

// Open decrypts data previously returned by Seal for the current Windows user.
func Open(encrypted []byte) ([]byte, error) {
	if len(encrypted) == 0 {
		return nil, errors.New("待解密数据为空")
	}
	return unprotect(encrypted)
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
