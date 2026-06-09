package partition

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// EnsureForTime creates the monthly partition for t if it does not exist.
func EnsureForTime(ctx context.Context, pool *pgxpool.Pool, t time.Time) error {
	monthStart := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
	_, err := pool.Exec(ctx, `SELECT ensure_audit_partition($1::date)`, monthStart)
	return err
}

// EnsureRange creates partitions for every month touched by [from, to).
func EnsureRange(ctx context.Context, pool *pgxpool.Pool, from, to time.Time) error {
	cur := time.Date(from.Year(), from.Month(), 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(to.Year(), to.Month(), 1, 0, 0, 0, 0, time.UTC)
	for !cur.After(end) {
		if err := EnsureForTime(ctx, pool, cur); err != nil {
			return err
		}
		cur = cur.AddDate(0, 1, 0)
	}
	return nil
}
