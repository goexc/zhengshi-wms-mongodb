package ui

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/url"
	"path"
	"strings"
	"sync"
	"time"

	"zhengshi-wms-windowsapp/internal/api"
)

// A lost reply must never turn a retry into a different financial instruction.
// Keep an immutable payload until success. A definite rejection before any
// uncertain outcome permits editing; rejection of a retry does not prove that
// the original request failed.
type documentSubmission[T any] struct {
	mu        sync.Mutex
	payload   []byte
	busy      bool
	uncertain bool
}

func (s *documentSubmission[T]) Begin(candidate T) (T, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var result T
	if s.busy {
		return result, fmt.Errorf("正在提交，请勿重复操作")
	}
	if !s.uncertain {
		data, err := json.Marshal(candidate)
		if err != nil {
			return result, err
		}
		s.payload = data
	}
	if err := json.Unmarshal(s.payload, &result); err != nil {
		return result, err
	}
	s.busy = true
	return result, nil
}

func (s *documentSubmission[T]) Finish(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.busy = false
	if err == nil {
		s.payload = nil
		s.uncertain = false
		return
	}
	var rejected *api.BusinessError
	if !errors.As(err, &rejected) || rejected.Code >= 500 {
		s.uncertain = true
	}
	if !s.uncertain {
		s.payload = nil
	}
}

func (s *documentSubmission[T]) Uncertain() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.uncertain
}

var documentLocation = time.FixedZone("Asia/Shanghai", 8*60*60)

const documentMaxAmount = 9999999999.99

func financialAttachmentIsImage(reference string) bool {
	parsed, err := url.Parse(strings.TrimSpace(reference))
	if err != nil {
		return false
	}
	switch strings.ToLower(path.Ext(parsed.Path)) {
	case ".png", ".jpg", ".jpeg", ".gif", ".bmp", ".webp":
		return true
	}
	return false
}

// DateEdit exposes a calendar date in the host zone; business dates use Beijing.
func documentCalendarDate(value time.Time) int64 {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, documentLocation).Unix()
}

func documentMoneySum(amount, tax float64) float64 {
	return float64(int64(math.Round(amount*100))+int64(math.Round(tax*100))) / 100
}

func documentInvoiceTotal(amount, tax float64) (float64, error) {
	total := documentMoneySum(amount, tax)
	if total > documentMaxAmount {
		return 0, fmt.Errorf("价税合计不能超过9,999,999,999.99元")
	}
	return total, nil
}

func documentTaxAmount(amount, rate float64) float64 {
	// Integer cents and four decimal percentage places avoid binary half-cent
	// rounding. At documentMaxAmount and 100%, the product stays below 1e18.
	cents := int64(math.Round(amount * 100))
	rateUnits := int64(math.Round(rate * 10000))
	return float64((cents*rateUnits+500000)/1000000) / 100
}
