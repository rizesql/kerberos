package kdb

import (
	"context"
	"database/sql"
)

type Database interface {
	DBTX
	Close() error
}

type DBTX interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	PrepareContext(context.Context, string) (*sql.Stmt, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

type DBTx interface {
	DBTX
	Commit() error
	Rollback() error
}
