package pgxpool_transactor

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type txCtxKeyType struct{}

var txCtxKey = txCtxKeyType{}

func setTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, txCtxKey, tx)
}

func getTx(ctx context.Context) (pgx.Tx, bool) {
	tx, ok := ctx.Value(txCtxKey).(pgx.Tx)
	return tx, ok
}
