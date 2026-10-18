package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestCustomerDocumentQueriesAndSummary(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/customer/acceptance":
			q := r.URL.Query()
			for k, v := range map[string]string{"page": "2", "size": "20", "customer_id": "customer", "keyword": "票号", "status": "pending", "date_start": "1700000000", "due_end": "1800000000", "due_state": "overdue", "redemption_start": "1700100000"} {
				if q.Get(k) != v {
					t.Errorf("%s=%q, want %q", k, q.Get(k), v)
				}
			}
			io.WriteString(w, `{"code":200,"data":{"total":21,"list":[{"id":"bill","annex":["acceptance/2026/10/ticket.pdf"],"history":[{"action":"create","time":1700000000,"operator_name":"财务"}]}],"summary":{"pending_amount":1200,"overdue_amount":600}}}`)
		case "/customer/invoice/detail":
			if r.URL.Query().Get("id") != "invoice" {
				t.Error("missing id")
			}
			io.WriteString(w, `{"code":200,"data":{"id":"invoice","tax_rate":13,"tax_amount":12.99,"total_amount":112.99,"annex":["invoice/2026/10/a.pdf"]}}`)
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()
	client := NewClient(server.URL)
	page, err := client.CustomerDocuments(context.Background(), "acceptance", 2, 20, DocumentFilters{CustomerID: "customer", Keyword: " 票号 ", Status: "pending", DateStart: 1700000000, DueEnd: 1800000000, DueState: "overdue", RedemptionStart: 1700100000})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 21 || page.Summary.OverdueAmount != 600 || len(page.List[0].Annex) != 1 || page.List[0].History[0].OperatorName != "财务" {
		t.Fatalf("bad response: %#v", page)
	}
	invoice, err := client.CustomerDocument(context.Background(), "invoice", "invoice")
	if err != nil || invoice.TaxAmount != 12.99 {
		t.Fatalf("tax rounding correction lost: %#v %v", invoice, err)
	}
	if _, err = client.CustomerDocuments(context.Background(), "invalid", 1, 20, DocumentFilters{}); err == nil {
		t.Fatal("unknown kind accepted")
	}
	if calls != 2 {
		t.Fatalf("unexpected calls=%d", calls)
	}
}

func TestFinancialUploadsPreserveAllScopesAndRejectMismatchedResponses(t *testing.T) {
	for _, scope := range []string{"acceptance", "acceptance_redemption", "invoice"} {
		t.Run(scope, func(t *testing.T) {
			prefix := map[string]string{"acceptance": "acceptance/", "acceptance_redemption": "acceptance/redemption/", "invoice": "invoice/"}[scope]
			response := prefix + "2026/10/file.pdf"
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if err := r.ParseMultipartForm(1 << 20); err != nil {
					t.Error(err)
					return
				}
				if r.FormValue("scope") != scope {
					t.Errorf("scope=%q", r.FormValue("scope"))
				}
				file, header, err := r.FormFile("files")
				if err != nil {
					t.Error(err)
					return
				}
				defer file.Close()
				content, _ := io.ReadAll(file)
				if header.Filename != "file.pdf" || string(content) != "%PDF-fixture" {
					t.Error("upload payload changed")
				}
				fmt.Fprintf(w, `{"code":200,"data":{"url":%q}}`, response)
			}))
			defer server.Close()
			file := filepath.Join(t.TempDir(), "file.pdf")
			if err := os.WriteFile(file, []byte("%PDF-fixture"), 0600); err != nil {
				t.Fatal(err)
			}
			client := NewClient(server.URL)
			got, err := client.UploadDocumentAttachment(context.Background(), file, scope)
			if err != nil || got != response {
				t.Fatalf("valid response: %q %v", got, err)
			}
			for _, bad := range []string{"", "legacy-flat.pdf", "https://external.test/file.pdf", prefix + "../bad.pdf", "acceptance/redemption/wrong.pdf"} {
				if scope == "acceptance_redemption" && bad == "acceptance/redemption/wrong.pdf" {
					continue
				}
				response = bad
				if _, err := client.UploadDocumentAttachment(context.Background(), file, scope); err == nil {
					t.Errorf("accepted wrong upload reference %q", bad)
				}
			}
			before := calls
			if err := os.Truncate(file, 21<<20); err != nil {
				t.Fatal(err)
			}
			if _, err := client.UploadDocumentAttachment(context.Background(), file, scope); err == nil {
				t.Error("oversized file accepted")
			}
			if calls != before {
				t.Error("oversized attachment sent to server")
			}
		})
	}
}

func TestCustomerDocumentWritesPreserveVersionIdempotencyAndOptionalVoucher(t *testing.T) {
	count := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count++
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		switch count {
		case 1:
			if r.Method != "PUT" || r.URL.Path != "/customer/invoice" || body["version"] != float64(7) || body["tax_amount"] != 12.99 || body["total_amount"] != 112.99 {
				t.Errorf("unexpected update: %s %s %#v", r.Method, r.URL.Path, body)
			}
		case 2:
			if r.URL.Path != "/customer/acceptance/redeem" || body["transaction_id"] != "existing-payment" || body["idempotency_key"] != "same-operation" || body["redemption_date"] != float64(1700000000) {
				t.Errorf("unexpected redeem %#v", body)
			}
			if a, ok := body["redemption_annex"].([]any); !ok || len(a) != 0 {
				t.Errorf("optional voucher must be empty array: %#v", body)
			}
			if _, ok := body["account"]; ok {
				t.Error("unexpected receiving account")
			}
		case 3:
			if r.URL.Path != "/customer/acceptance/reverse" || body["reason"] != "误操作" || body["version"] != float64(8) {
				t.Errorf("unexpected reverse %#v", body)
			}
		}
		io.WriteString(w, `{"code":200,"data":{"id":"document","version":8}}`)
	}))
	defer server.Close()
	client := NewClient(server.URL)
	if _, err := client.SaveCustomerDocument(context.Background(), "invoice", CustomerDocument{ID: "document", CustomerID: "customer", Version: 7, AmountExcludingTax: 100, TaxRate: 13, TaxAmount: 12.99, TotalAmount: 112.99, Annex: []string{}}); err != nil {
		t.Fatal(err)
	}
	if _, err := client.CustomerDocumentAction(context.Background(), "acceptance", "redeem", DocumentActionRequest{ID: "document", Version: 7, TransactionID: "existing-payment", IdempotencyKey: "same-operation", RedemptionDate: 1700000000, RedemptionAnnex: []string{}}); err != nil {
		t.Fatal(err)
	}
	if _, err := client.CustomerDocumentAction(context.Background(), "acceptance", "reverse", DocumentActionRequest{ID: "document", Version: 8, Reason: "误操作", IdempotencyKey: "reverse-operation"}); err != nil {
		t.Fatal(err)
	}
	if _, err := client.CustomerDocumentAction(context.Background(), "invoice", "redeem", DocumentActionRequest{}); err == nil {
		t.Fatal("invoice redemption accepted")
	}
	if count != 3 {
		t.Fatalf("unexpected retry count %d", count)
	}
}

func TestFinancialAttachmentUploadSendsScopedMultipart(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/images" || r.Method != "POST" {
			t.Errorf("wrong upload endpoint %s", r.URL.Path)
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatal(err)
		}
		if r.FormValue("scope") != "acceptance_redemption" {
			t.Fatal("scope missing")
		}
		file, header, err := r.FormFile("files")
		if err != nil {
			t.Fatal(err)
		}
		defer file.Close()
		data, _ := io.ReadAll(file)
		if header.Filename != "voucher.pdf" || string(data) != "%PDF-fixture" {
			t.Fatal("multipart file changed")
		}
		io.WriteString(w, `{"code":200,"data":{"url":"acceptance/redemption/2026/10/voucher.pdf"}}`)
	}))
	defer server.Close()
	path := filepath.Join(t.TempDir(), "voucher.pdf")
	if err := os.WriteFile(path, []byte("%PDF-fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	ref, err := NewClient(server.URL).UploadDocumentAttachment(context.Background(), path, "acceptance_redemption")
	if err != nil || ref != "acceptance/redemption/2026/10/voucher.pdf" {
		t.Fatalf("unexpected result %q %v", ref, err)
	}
}

func TestZeroRateInvoiceWritesRequiredZeroFieldsWithoutResponseMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if r.Method != "POST" || r.URL.Path != "/customer/invoice" {
			t.Fatalf("wrong create endpoint: %s %s", r.Method, r.URL.Path)
		}
		for _, name := range []string{"tax_rate", "tax_amount"} {
			v, ok := body[name]
			if !ok || v != float64(0) {
				t.Errorf("required zero field %s missing or changed: %#v", name, body)
			}
		}
		for _, name := range []string{"status", "history", "creator_name", "created_at", "transaction_id", "bill_number"} {
			if _, ok := body[name]; ok {
				t.Errorf("response-only field sent: %s", name)
			}
		}
		if a, ok := body["annex"].([]any); !ok || len(a) != 0 {
			t.Error("empty attachments must serialize as []")
		}
		io.WriteString(w, `{"code":200,"data":{"id":"new-invoice","tax_rate":0,"tax_amount":0}}`)
	}))
	defer server.Close()
	_, err := NewClient(server.URL).SaveCustomerDocument(context.Background(), "invoice", CustomerDocument{CustomerID: "customer", InvoiceNumber: "zero-rate", InvoiceDate: 1700000000, AmountExcludingTax: 100, TotalAmount: 100, IdempotencyKey: "create-zero-rate", Status: "active", CreatorName: "response-only", TransactionID: "irrelevant", BillNumber: "not-for-invoice"})
	if err != nil {
		t.Fatal(err)
	}
}
