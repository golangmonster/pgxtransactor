package pgxpool_transactor

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Querier interface {
	Exec(ctx context.Context, sql string, arguments ...any) (commandTag pgconn.CommandTag, err error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error)
}

type Transactor interface {
	InTx(ctx context.Context, fn func(ctx context.Context) error) error
	InTxWithIsoLevel(ctx context.Context, isoLevel pgx.TxIsoLevel, fn func(ctx context.Context) error) error
}
