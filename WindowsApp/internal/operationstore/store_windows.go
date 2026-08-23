//go:build windows

package operationstore

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"golang.org/x/sys/windows"

	"zhengshi-wms-windowsapp/internal/api"
	"zhengshi-wms-windowsapp/internal/securestore"
)

const (
	journalVersion = 1
	journalTTL     = 7 * 24 * time.Hour
	journalLimit   = 100
)

type journal struct {
	Version    int                  `json:"version"`
	APIBaseURL string               `json:"api_base_url"`
	Mobile     string               `json:"mobile"`
	SavedAt    time.Time            `json:"saved_at"`
	Events     []api.OperationEvent `json:"events"`
}

func Load(apiBaseURL, mobile string) ([]api.OperationEvent, error) {
	name, err := path(apiBaseURL, mobile)
	if err != nil {
		return nil, err
	}
	encrypted, err := os.ReadFile(name)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	plain, err := securestore.Open(encrypted)
	if err != nil {
		return nil, err
	}
	var stored journal
	if err := json.Unmarshal(plain, &stored); err != nil {
		return nil, err
	}
	if stored.Version != journalVersion || !sameScope(stored.APIBaseURL, stored.Mobile, apiBaseURL, mobile) {
		return nil, nil
	}
	events := normalizeEvents(stored.Events, time.Now())
	if len(events) != len(stored.Events) {
		if err := Save(apiBaseURL, mobile, events); err != nil {
			return nil, err
		}
	}
	return events, nil
}

func Save(apiBaseURL, mobile string, events []api.OperationEvent) error {
	name, err := path(apiBaseURL, mobile)
	if err != nil {
		return err
	}
	events = normalizeEvents(events, time.Now())
	if len(events) == 0 {
		if err := os.Remove(name); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	data, err := json.Marshal(journal{
		Version: journalVersion, APIBaseURL: strings.TrimSpace(apiBaseURL), Mobile: strings.TrimSpace(mobile),
		SavedAt: time.Now().UTC(), Events: events,
	})
	if err != nil {
		return err
	}
	encrypted, err := securestore.Seal(data)
	if err != nil {
		return err
	}
	return atomicWrite(name, encrypted)
}

func path(apiBaseURL, mobile string) (string, error) {
	apiBaseURL = strings.ToLower(strings.TrimRight(strings.TrimSpace(apiBaseURL), "/"))
	mobile = strings.TrimSpace(mobile)
	if apiBaseURL == "" || mobile == "" {
		return "", errors.New("操作复核日志缺少账号或 API 环境")
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir = filepath.Join(dir, "ZhengshiWMS", "operations")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	digest := sha256.Sum256([]byte(apiBaseURL + "\n" + mobile))
	return filepath.Join(dir, hex.EncodeToString(digest[:8])+".dat"), nil
}

func sameScope(storedAPI, storedMobile, apiBaseURL, mobile string) bool {
	return strings.EqualFold(strings.TrimRight(strings.TrimSpace(storedAPI), "/"), strings.TrimRight(strings.TrimSpace(apiBaseURL), "/")) &&
		strings.TrimSpace(storedMobile) == strings.TrimSpace(mobile)
}

func normalizeEvents(events []api.OperationEvent, now time.Time) []api.OperationEvent {
	result := make([]api.OperationEvent, 0, len(events))
	seen := make(map[string]bool, len(events))
	for _, event := range events {
		if event.Outcome != api.OperationPending && event.Outcome != api.OperationUnknown {
			continue
		}
		if event.ID == "" || seen[event.ID] || event.StartedAt.IsZero() || now.Sub(event.StartedAt) > journalTTL {
			continue
		}
		seen[event.ID] = true
		event.Method = limit(strings.ToUpper(strings.TrimSpace(event.Method)), 16)
		event.Path = limit(strings.TrimSpace(event.Path), 160)
		event.KeyField = limit(strings.TrimSpace(event.KeyField), 32)
		event.BusinessKey = limit(strings.TrimSpace(event.BusinessKey), 256)
		event.Message = limit(strings.TrimSpace(event.Message), 512)
		result = append(result, event)
	}
	sort.SliceStable(result, func(i, j int) bool { return result[i].StartedAt.After(result[j].StartedAt) })
	if len(result) > journalLimit {
		result = result[:journalLimit]
	}
	return result
}

func limit(value string, maximum int) string {
	characters := []rune(value)
	if len(characters) <= maximum {
		return value
	}
	return string(characters[:maximum])
}

func atomicWrite(name string, data []byte) error {
	dir := filepath.Dir(name)
	temporary, err := os.CreateTemp(dir, "operation-*.tmp")
	if err != nil {
		return err
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return err
	}
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
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
