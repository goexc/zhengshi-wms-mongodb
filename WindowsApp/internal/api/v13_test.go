package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOperationObserverRecordsWriteWithoutRequestBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/outbound/pick" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"code":200,"msg":"success"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	var events []OperationEvent
	client.SetOperationObserver(func(value OperationEvent) { events = append(events, value) })
	if err := client.PickOutbound(context.Background(), OutboundPickRequest{Code: "OUT-100"}); err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 || events[0].Outcome != OperationPending || events[0].ID != events[1].ID {
		t.Fatalf("events = %#v", events)
	}
	event := events[1]
	if event.Outcome != OperationSucceeded || event.BusinessKey != "OUT-100" || event.KeyField != "code" {
		t.Fatalf("event = %#v", event)
	}
}

func TestChangeAvatarUsesTargetedEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/account/avatar" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"code":200,"msg":"success"}`))
	}))
	defer server.Close()

	if err := NewClient(server.URL).ChangeAvatar(context.Background(), "https://files.example.invalid/avatar.png"); err != nil {
		t.Fatal(err)
	}
}

func TestOperationBusinessKeyNeverSelectsAvatarOrPassword(t *testing.T) {
	field, value := operationBusinessKey(nil, []byte(`{"avatar":"https://secret.invalid/a.png","password":"secret"}`))
	if field != "" || value != "" {
		t.Fatalf("key = %q/%q", field, value)
	}
}

func TestOperationObserverTreatsServerFailureAsUnknown(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"code":500,"msg":"internal"}`))
	}))
	defer server.Close()
	client := NewClient(server.URL)
	var events []OperationEvent
	client.SetOperationObserver(func(value OperationEvent) { events = append(events, value) })
	if err := client.ChangeAvatar(context.Background(), "https://files.example.invalid/avatar.png"); err == nil {
		t.Fatal("expected server failure")
	}
	if len(events) != 2 || events[0].Outcome != OperationPending {
		t.Fatalf("events = %#v", events)
	}
	event := events[1]
	if event.Outcome != OperationUnknown {
		t.Fatalf("event = %#v", event)
	}
}

func TestFindInboundReceiptSupportsExactCodeWithoutID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/inbound/receipt" || r.URL.Query().Get("code") != "IN-100" {
			t.Fatalf("request = %s?%s", r.URL.Path, r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`{"code":200,"msg":"success","data":{"total":1,"list":[{"id":"receipt","code":"IN-100","status":"待审核"}]}}`))
	}))
	defer server.Close()

	receipt, found, err := NewClient(server.URL).FindInboundReceipt(context.Background(), "", "IN-100")
	if err != nil || !found || receipt.ID != "receipt" {
		t.Fatalf("receipt = %#v, found=%v, err=%v", receipt, found, err)
	}
}

func TestFindOutboundSupportsIDWithoutCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"code":200,"msg":"success","data":{"total":1,"list":[{"id":"order","code":"OUT-100","status":"预发货"}]}}`))
	}))
	defer server.Close()

	order, found, err := NewClient(server.URL).FindOutbound(context.Background(), "order", "")
	if err != nil || !found || order.Code != "OUT-100" {
		t.Fatalf("order = %#v, found=%v, err=%v", order, found, err)
	}
}
