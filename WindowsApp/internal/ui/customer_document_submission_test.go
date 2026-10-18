package ui

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/image/bmp"
	"zhengshi-wms-windowsapp/internal/api"
)

// Exercise the production submission state and HTTP client together. The
// server commits, drops the first reply, and only accepts a byte-identical replay.
func verifyDocumentLostReply[T any](t *testing.T, original, edited T, send func(*api.Client, T) error) {
	t.Helper()
	var first []byte
	var writes, requests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		requests++
		if requests == 1 {
			first = body
			writes++
			conn, _, err := w.(http.Hijacker).Hijack()
			if err != nil {
				t.Error(err)
				return
			}
			conn.Close()
			return
		}
		if !bytes.Equal(first, body) {
			t.Errorf("retry changed the committed request: first=%s retry=%s", first, body)
			io.WriteString(w, `{"code":409,"msg":"idempotency mismatch"}`)
			return
		}
		io.WriteString(w, `{"code":200,"data":{"id":"same-document","version":2}}`)
	}))
	defer server.Close()
	client := api.NewClient(server.URL)
	var attempt documentSubmission[T]
	request, err := attempt.Begin(original)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := attempt.Begin(edited); err == nil {
		t.Fatal("double submission was accepted")
	}
	err = send(client, request)
	if err == nil {
		t.Fatal("expected lost reply")
	}
	attempt.Finish(err)
	if !attempt.Uncertain() {
		t.Fatal("lost reply must freeze the instruction")
	}
	request, err = attempt.Begin(edited)
	if err != nil {
		t.Fatal(err)
	}
	err = send(client, request)
	attempt.Finish(err)
	if err != nil || attempt.Uncertain() {
		t.Fatalf("retry failed: %v", err)
	}
	if writes != 1 || requests != 2 {
		t.Fatalf("writes=%d requests=%d", writes, requests)
	}
}

func TestFinancialActionsKeepExactInstructionAfterCommittedReplyIsLost(t *testing.T) {
	for _, action := range []string{"redeem", "reverse", "void"} {
		for _, payment := range []string{"", "existing-manual-payment"} {
			t.Run(action+"/"+payment, func(t *testing.T) {
				first := api.DocumentActionRequest{ID: "bill", Version: 4, RedemptionDate: 1800000000, IdempotencyKey: "original-key", TransactionID: payment, RedemptionAnnex: []string{"acceptance/redemption/old.pdf"}, Reason: "原原因"}
				edited := first
				edited.TransactionID = "other-payment"
				edited.IdempotencyKey = "new-key"
				edited.RedemptionDate++
				edited.RedemptionAnnex = []string{"acceptance/redemption/changed.pdf"}
				edited.Reason = "改后的原因"
				verifyDocumentLostReply(t, first, edited, func(c *api.Client, request api.DocumentActionRequest) error {
					_, err := c.CustomerDocumentAction(context.Background(), "acceptance", action, request)
					return err
				})
			})
		}
	}
}

func TestFinancialCreateAndEditKeepExactInstructionAfterLostReply(t *testing.T) {
	for _, kind := range []string{"acceptance", "invoice"} {
		for _, id := range []string{"", "edited-document"} {
			t.Run(kind+"/"+id, func(t *testing.T) {
				first := api.CustomerDocument{ID: id, Version: 2, CustomerID: "customer", BillNumber: "bill", InvoiceNumber: "invoice", Amount: 10, AmountExcludingTax: 10, TotalAmount: 10, IdempotencyKey: "original-key", Annex: []string{kind + "/old.pdf"}}
				edited := first
				edited.Amount = 20
				edited.TotalAmount = 20
				edited.TaxRate = 13
				edited.CustomerID = "other-customer"
				edited.IdempotencyKey = "changed-key"
				edited.Annex = []string{kind + "/new.pdf"}
				verifyDocumentLostReply(t, first, edited, func(c *api.Client, request api.CustomerDocument) error {
					_, err := c.SaveCustomerDocument(context.Background(), kind, request)
					return err
				})
			})
		}
	}
}

func TestDocumentSubmissionConcurrencyAndDefiniteRejection(t *testing.T) {
	var attempt documentSubmission[api.DocumentActionRequest]
	var accepted atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 24; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := attempt.Begin(api.DocumentActionRequest{IdempotencyKey: "first"}); err == nil {
				accepted.Add(1)
			}
		}()
	}
	wg.Wait()
	if accepted.Load() != 1 {
		t.Fatalf("accepted %d concurrent submissions", accepted.Load())
	}
	attempt.Finish(&api.BusinessError{Code: 400, Msg: "required field"})
	if attempt.Uncertain() {
		t.Fatal("definite initial rejection must permit correcting the form")
	}
	request, err := attempt.Begin(api.DocumentActionRequest{IdempotencyKey: "corrected", RedemptionAnnex: []string{"original.pdf"}})
	if err != nil || request.IdempotencyKey != "corrected" {
		t.Fatalf("correction not used: %#v %v", request, err)
	}
	request.RedemptionAnnex[0] = "mutated.pdf"
	attempt.Finish(io.ErrUnexpectedEOF)
	request, _ = attempt.Begin(api.DocumentActionRequest{})
	if request.RedemptionAnnex[0] != "original.pdf" {
		t.Fatal("caller mutated frozen attachments")
	}
	attempt.Finish(&api.BusinessError{Code: 409, Msg: "version changed"})
	if !attempt.Uncertain() {
		t.Fatal("a rejected retry cannot prove the first request did not commit")
	}
}

func TestDocumentAmountsAndBeijingCalendar(t *testing.T) {
	for _, row := range []struct{ amount, rate, want float64 }{{0.29, 50, 0.15}, {100, 0, 0}, {100, 13, 13}, {123.45, 0.1234, 0.15}, {documentMaxAmount, 100, documentMaxAmount}} {
		if got := documentTaxAmount(row.amount, row.rate); got != row.want {
			t.Errorf("tax(%v,%v)=%v, want %v", row.amount, row.rate, got, row.want)
		}
	}
	if got := documentMoneySum(100, 12.99); got != 112.99 {
		t.Errorf("manual tax correction lost: %v", got)
	}
	if got, err := documentInvoiceTotal(documentMaxAmount, 0); err != nil || got != documentMaxAmount {
		t.Fatalf("valid upper limit rejected: %v %v", got, err)
	}
	if _, err := documentInvoiceTotal(documentMaxAmount, 0.01); err == nil {
		t.Fatal("invoice total above server limit accepted")
	}
	selected := time.Date(2026, 10, 18, 17, 42, 1, 0, time.FixedZone("client-west", -7*3600))
	want := time.Date(2026, 10, 18, 0, 0, 0, 0, documentLocation).Unix()
	if got := documentCalendarDate(selected); got != want {
		t.Fatalf("date picked on another host timezone drifted: %d != %d", got, want)
	}
	if got := documentDate(want); got != "2026-10-18" {
		t.Fatalf("display date=%s", got)
	}
}

func TestFinancialImagesAcceptLegacySignedURLsAndDecodeBMP(t *testing.T) {
	for _, ref := range []string{"old.jpg", "acceptance/2026/10/a.webp", "https://old.test/a.PNG?signature=old", "invoice/a.bmp#page"} {
		if !financialAttachmentIsImage(ref) {
			t.Error(ref)
		}
	}
	if financialAttachmentIsImage("invoice/a.pdf") {
		t.Fatal("PDF treated as bitmap")
	}
	var data bytes.Buffer
	picture := image.NewRGBA(image.Rect(0, 0, 1, 1))
	picture.Set(0, 0, color.White)
	if err := bmp.Encode(&data, picture); err != nil {
		t.Fatal(err)
	}
	if _, format, err := image.Decode(&data); err != nil || format != "bmp" {
		t.Fatal(fmt.Sprintf("BMP preview decoder: %s %v", format, err))
	}
}
