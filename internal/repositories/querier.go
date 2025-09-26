package repositories

import (
	"context"
	"database/sql"
)

// wrapper that implements common functions from sql.DB and sql.Tx
// to not write logic twice
type Querier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}
