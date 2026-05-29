package pgxtransactor

import "github.com/jackc/pgx/v5/pgxpool"

type Pool struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Pool {
	return &Pool{pool: pool}
}
