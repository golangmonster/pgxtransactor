package pgxtransactor

import (
	"context"
	"testing"

	"github.com/golangmonster/pgxtransactor/test"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
)

func createTable(ctx context.Context, pool *Pool) error {
	_, err := pool.Querier(ctx).Exec(ctx, `
					CREATE TABLE accounts (
						id SERIAL PRIMARY KEY,
						balance INT
					)
				`)

	return err
}

type testcase struct {
	name    string
	fn      func(ctx context.Context, pool *Pool) func(ctx context.Context) error
	assert  func(ctx context.Context, pool *Pool)
	wantErr bool
}

func TestInTx(t *testing.T) {
	tt := []testcase{
		{
			name: "Tx commit",
			fn: func(ctx context.Context, pool *Pool) func(ctx context.Context) error {
				return func(ctx context.Context) error {
					_, err := pool.Querier(ctx).Exec(ctx, `INSERT INTO accounts(balance) VALUES($1)`, 100)
					if err != nil {
						return err
					}

					_, err = pool.Querier(ctx).Exec(ctx, `INSERT INTO accounts(balance) VALUES($1)`, 100)
					if err != nil {
						return err
					}

					return nil
				}
			},
			assert: func(ctx context.Context, pool *Pool) {
				var count int
				err := pool.Querier(ctx).
					QueryRow(ctx, `SELECT count(*) FROM accounts`).
					Scan(&count)

				assert.NoError(t, err)
				assert.Equal(t, 2, count)

				var sum int
				err = pool.Querier(ctx).
					QueryRow(ctx, `SELECT sum(balance) FROM accounts`).
					Scan(&sum)

				assert.NoError(t, err)
				assert.Equal(t, 200, sum)
			},
			wantErr: false,
		},
		{
			name: "Tx atomicity",
			fn: func(ctx context.Context, pool *Pool) func(ctx context.Context) error {
				return func(ctx context.Context) error {
					_, err := pool.Querier(ctx).Exec(ctx, `INSERT INTO accounts(balance) VALUES($1)`, 100)
					if err != nil {
						return err
					}

					_, err = pool.Querier(ctx).Exec(ctx, `INSERT INTO accounts(error) VALUES($1)`, 100)
					if err != nil {
						return err
					}

					return nil
				}
			},
			assert: func(ctx context.Context, pool *Pool) {
				var count int

				err := pool.Querier(ctx).QueryRow(ctx,
					`SELECT balance FROM accounts where id = 1`,
				).Scan(&count)

				assert.ErrorIs(t, err, pgx.ErrNoRows)
				assert.Equal(t, 0, count)

				err = pool.Querier(ctx).QueryRow(ctx,
					`SELECT balance FROM accounts where id = 2`,
				).Scan(&count)

				assert.ErrorIs(t, err, pgx.ErrNoRows)
				assert.Equal(t, 0, count)
			},
			wantErr: true,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			pool, stop := test.NewPostgres(t)
			defer stop()

			transactor := New(pool)

			// arrange
			ctx := context.Background()

			err := createTable(ctx, transactor)
			if err != nil {
				t.Fatal(err)
			}

			// act
			err = transactor.InTx(ctx, tc.fn(ctx, transactor))

			// assert
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			tc.assert(ctx, transactor)
		})

	}
}

func TestInTxNested(t *testing.T) {
	pool, stop := test.NewPostgres(t)
	defer stop()

	transactor := New(pool)

	// arrange
	ctx := context.Background()

	err := createTable(ctx, transactor)
	if err != nil {
		t.Fatal(err)
	}

	// act + assert
	err = transactor.InTx(ctx, func(ctx context.Context) error {
		_, err := transactor.Querier(ctx).Exec(ctx, `INSERT INTO accounts(balance) VALUES($1)`, 100)
		if err != nil {
			return err
		}

		outerTx, inTxOuter := getTx(ctx)

		assert.True(t, inTxOuter)

		return transactor.InTx(ctx, func(ctx context.Context) error {
			innerTx, inTxInner := getTx(ctx)

			assert.True(t, inTxInner)
			assert.Same(t, outerTx, innerTx)

			var count int

			err = transactor.Querier(ctx).QueryRow(ctx,
				`SELECT balance FROM accounts where id = 1`,
			).Scan(&count)
			if err != nil {
				return err
			}

			assert.Equal(t, 100, count)

			_, err = transactor.Querier(ctx).Exec(ctx, `UPDATE accounts SET balance = balance + $1 where id = 1`, count)
			if err != nil {
				return err
			}

			return nil
		})
	})

	assert.NoError(t, err)

	var count int

	err = transactor.Querier(ctx).QueryRow(ctx,
		`SELECT balance FROM accounts where id = 1`,
	).Scan(&count)

	assert.NoError(t, err)
	assert.Equal(t, 200, count)
}

func TestInTxPanicRollback(t *testing.T) {
	pool, stop := test.NewPostgres(t)
	defer stop()

	transactor := New(pool)

	// arrange
	ctx := context.Background()

	err := createTable(ctx, transactor)
	if err != nil {
		t.Fatal(err)
	}

	// act + assert
	assert.Panics(t, func() {
		_ = transactor.InTx(ctx, func(ctx context.Context) error {
			_, err := transactor.Querier(ctx).Exec(ctx, `INSERT INTO accounts(balance) VALUES($1)`, 100)
			if err != nil {
				return err
			}

			panic("AAAAAAA")
		})
	})

	var count int

	err = transactor.Querier(ctx).
		QueryRow(ctx, `SELECT balance FROM accounts WHERE id = 1`).
		Scan(&count)

	assert.ErrorIs(t, err, pgx.ErrNoRows)
	assert.Equal(t, 0, count)
}
