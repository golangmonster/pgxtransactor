package pgxpool_transactor

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func (p *Pool) Querier(ctx context.Context) Querier {
	tx, inTx := getTx(ctx)
	if inTx {
		return tx
	}

	return p.pool
}
