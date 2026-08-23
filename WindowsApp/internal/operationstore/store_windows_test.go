//go:build windows

package operationstore

import (
	"bytes"
	"os"
	"testing"
	"time"

	"zhengshi-wms-windowsapp/internal/api"
)

func TestNormalizeEventsOnlyKeepsRecentUnresolvedItems(t *testing.T) {
	now := time.Now()
	events := normalizeEvents([]api.OperationEvent{
		{ID: "pending", StartedAt: now.Add(-time.Minute), Outcome: api.OperationPending, BusinessKey: "IN-1"},
		{ID: "unknown", StartedAt: now.Add(-time.Hour), Outcome: api.OperationUnknown, BusinessKey: "OUT-1"},
		{ID: "success", StartedAt: now, Outcome: api.OperationSucceeded},
		{ID: "expired", StartedAt: now.Add(-8 * 24 * time.Hour), Outcome: api.OperationUnknown},
	}, now)
	if len(events) != 2 || events[0].ID != "pending" || events[1].ID != "unknown" {
		t.Fatalf("events = %#v", events)
	}
}

func TestJournalPathDoesNotExposeMobile(t *testing.T) {
	name, err := path("https://api.example", "18810509066")
	if err != nil {
		t.Fatal(err)
	}
	if len(name) == 0 || contains(name, "18810509066") {
		t.Fatalf("path = %q", name)
	}
}

func TestEncryptedJournalRoundTripAndClear(t *testing.T) {
	t.Setenv("AppData", t.TempDir())
	now := time.Now().UTC()
	event := api.OperationEvent{
		ID: "request-1", Method: "PATCH", Path: "/outbound/pick", KeyField: "code", BusinessKey: "OUT-SECRET",
		StartedAt: now, Outcome: api.OperationUnknown, Message: "request result unknown",
	}
	if err := Save("https://api.example", "18800000000", []api.OperationEvent{event}); err != nil {
		t.Fatal(err)
	}
	name, err := path("https://api.example", "18800000000")
	if err != nil {
		t.Fatal(err)
	}
	encrypted, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(encrypted, []byte("OUT-SECRET")) {
		t.Fatal("encrypted journal contains plaintext business key")
	}
	loaded, err := Load("https://api.example", "18800000000")
	if err != nil || len(loaded) != 1 || loaded[0].BusinessKey != event.BusinessKey {
		t.Fatalf("loaded=%#v err=%v", loaded, err)
	}
	if err := Save("https://api.example", "18800000000", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(name); !os.IsNotExist(err) {
		t.Fatalf("cleared journal stat err=%v", err)
	}
}

func contains(value, target string) bool {
	for index := 0; index+len(target) <= len(value); index++ {
		if value[index:index+len(target)] == target {
			return true
		}
	}
	return false
}
