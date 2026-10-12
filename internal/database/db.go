// Package database contains the generated helper types used to execute SQL queries.
package database

import (
	"context"
	"database/sql"
)

// DBTX is the narrow database interface required by the generated sqlc queries.
type DBTX interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	PrepareContext(context.Context, string) (*sql.Stmt, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

// New constructs a Queries object backed by the provided database connection.
func New(db DBTX) *Queries {
	return &Queries{db: db}
}

// Queries wraps the database connection and exposes the generated query methods.
type Queries struct {
	db DBTX
}

// WithTx returns a query bundle that runs against the provided transaction.
func (q *Queries) WithTx(tx *sql.Tx) *Queries {
	return &Queries{
		db: tx,
	}
}
