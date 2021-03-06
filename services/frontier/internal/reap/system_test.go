package reap

import (
	"testing"

	"github.com/digitalbits/go/services/frontier/internal/ledger"
	"github.com/digitalbits/go/services/frontier/internal/test"
)

func TestDeleteUnretainedHistory(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	ledgerState := &ledger.State{}
	ledgerState.SetStatus(tt.Scenario("kahuna"))

	db := tt.FrontierSession()

	sys := New(0, db, ledgerState)

	var (
		prev int
		cur  int
	)
	err := db.GetRaw(&prev, `SELECT COUNT(*) FROM history_ledgers`)
	tt.Require.NoError(err)

	err = sys.DeleteUnretainedHistory()
	if tt.Assert.NoError(err) {
		err = db.GetRaw(&cur, `SELECT COUNT(*) FROM history_ledgers`)
		tt.Require.NoError(err)
		tt.Assert.Equal(prev, cur, "Ledgers deleted when RetentionCount == 0")
	}

	ledgerState.SetStatus(tt.LoadLedgerStatus())
	sys.RetentionCount = 10
	err = sys.DeleteUnretainedHistory()
	if tt.Assert.NoError(err) {
		err = db.GetRaw(&cur, `SELECT COUNT(*) FROM history_ledgers`)
		tt.Require.NoError(err)
		tt.Assert.Equal(10, cur)
	}

	ledgerState.SetStatus(tt.LoadLedgerStatus())
	sys.RetentionCount = 1
	err = sys.DeleteUnretainedHistory()
	if tt.Assert.NoError(err) {
		err = db.GetRaw(&cur, `SELECT COUNT(*) FROM history_ledgers`)
		tt.Require.NoError(err)
		tt.Assert.Equal(1, cur)
	}
}
