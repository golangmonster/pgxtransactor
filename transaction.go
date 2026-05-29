package pgxpool_transactor

import (
	"context"

	"github.com/jackc/pgx/v5"
)

func (p *Pool) InTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return p.inTx(ctx, pgx.ReadCommitted, fn)
}

func (p *Pool) InTxWithIsoLevel(ctx context.Context, isoLevel pgx.TxIsoLevel, fn func(ctx context.Context) error) error {
	return p.inTx(ctx, isoLevel, fn)
}

func (p *Pool) inTx(ctx context.Context, isoLevel pgx.TxIsoLevel, fn func(ctx context.Context) error) error {
	// Если уже в транзакции, то новую не открываем
	tx, inTx := getTx(ctx)
	if inTx {
		return fn(ctx)
	}

	// Начинаем транзакцию
	tx, err := p.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: isoLevel})
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}

		_ = tx.Rollback(ctx)
	}()

	// Кладем транзакцию в контекст
	ctx = setTx(ctx, tx)

	if err = fn(ctx); err != nil {
		return err
	}

	// Завершаем транзакцию
	return tx.Commit(ctx)
}
