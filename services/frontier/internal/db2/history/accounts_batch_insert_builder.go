package history

import (
	"github.com/digitalbits/go/support/db"
	"github.com/digitalbits/go/xdr"
)

// accountsBatchInsertBuilder is a simple wrapper around db.BatchInsertBuilder
type accountsBatchInsertBuilder struct {
	builder db.BatchInsertBuilder
}

func (i *accountsBatchInsertBuilder) Add(entry xdr.LedgerEntry) error {
	return i.builder.Row(accountToMap(entry))
}

func (i *accountsBatchInsertBuilder) Exec() error {
	return i.builder.Exec()
}
