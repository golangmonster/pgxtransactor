package pgxtransactor

import (
	"context"
)

func (p *Pool) Querier(ctx context.Context) Querier {
	tx, inTx := getTx(ctx)
	if inTx {
		return tx
	}

	return p.pool
}
