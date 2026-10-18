package handler

import "testing"

// Old migration entry points must not reconnect to a database or recreate
// receivables, even if called by an existing wrapper.
func TestLegacyCustomerTransactionSyncDoesNotAccessDatabase(t *testing.T) {
	CustomerTransactionSync(nil)
}
