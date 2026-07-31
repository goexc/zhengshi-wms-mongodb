package ui

import (
	"errors"
	"testing"
)

func TestClassifyRestoreErrorKeepsNetworkFailure(t *testing.T) {
	original := errors.New("network unavailable")
	var networkErr *RestoreNetworkError
	if !errors.As(classifyRestoreError(original), &networkErr) {
		t.Fatal("network failure should remain retryable")
	}
}
