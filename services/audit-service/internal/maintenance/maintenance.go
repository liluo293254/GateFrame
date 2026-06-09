package maintenance

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PruneOldPartitions drops monthly audit_events_* partitions older than retentionMonths.
// Production: export to object storage before drop if compliance requires archives.
func PruneOldPartitions(ctx context.Context, pool *pgxpool.Pool, retentionMonths int) (int, error) {
	if retentionMonths <= 0 {
		return 0, nil
	}
	cutoff := time.Now().UTC().AddDate(0, -retentionMonths, 0)
	cutoffMonth := time.Date(cutoff.Year(), cutoff.Month(), 1, 0, 0, 0, 0, time.UTC)

	rows, err := pool.Query(ctx, `
		SELECT child.relname
		FROM pg_inherits
		JOIN pg_class parent ON pg_inherits.inhparent = parent.oid
		JOIN pg_class child ON pg_inherits.inhrelid = child.oid
		WHERE parent.relname = 'audit_events'
	`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	dropped := 0
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return dropped, err
		}
		month, ok := parsePartitionMonth(name)
		if !ok {
			continue
		}
		if !month.Before(cutoffMonth) {
			continue
		}
		query := fmt.Sprintf("DROP TABLE IF EXISTS %s", quoteIdent(name))
		if _, err := pool.Exec(ctx, query); err != nil {
			return dropped, fmt.Errorf("drop %s: %w", name, err)
		}
		slog.Info("audit partition dropped",
			"partition", name,
			"month", month.Format("2006-01"),
			"retention_months", retentionMonths,
		)
		dropped++
	}
	return dropped, rows.Err()
}

func parsePartitionMonth(name string) (time.Time, bool) {
	const prefix = "audit_events_"
	if !strings.HasPrefix(name, prefix) {
		return time.Time{}, false
	}
	suffix := strings.TrimPrefix(name, prefix)
	t, err := time.Parse("2006_01", suffix)
	if err != nil {
		return time.Time{}, false
	}
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC), true
}

func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func StartRetentionLoop(ctx context.Context, pool *pgxpool.Pool, retentionMonths int, interval time.Duration) {
	if retentionMonths <= 0 {
		slog.Info("audit partition retention disabled")
		return
	}
	run := func() {
		n, err := PruneOldPartitions(ctx, pool, retentionMonths)
		if err != nil {
			slog.Error("audit partition prune failed", "error", err)
			return
		}
		if n > 0 {
			slog.Info("audit partition prune complete", "dropped", n)
		}
	}
	run()
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				run()
			}
		}
	}()
}
