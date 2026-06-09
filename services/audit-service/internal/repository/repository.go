package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gateframe/audit-service/internal/model"
	"github.com/gateframe/audit-service/internal/partition"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Insert(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID, action, path string, statusCode int) (*model.AuditEvent, error) {
	now := time.Now().UTC()
	if err := partition.EnsureForTime(ctx, r.pool, now); err != nil {
		return nil, err
	}
	var ev model.AuditEvent
	var uid *uuid.UUID
	err := r.pool.QueryRow(ctx, `
		INSERT INTO audit_events (tenant_id, user_id, action, path, status_code, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, tenant_id, user_id, action, path, status_code, created_at
	`, tenantID, userID, action, path, statusCode, now).Scan(
		&ev.ID, &ev.TenantID, &uid, &ev.Action, &ev.Path, &ev.StatusCode, &ev.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	ev.UserID = uid
	return &ev, nil
}

func normalizeLimit(limit int) int {
	if limit <= 0 || limit > 5000 {
		return 50
	}
	return limit
}

func normalizeOffset(offset int) int {
	if offset < 0 {
		return 0
	}
	return offset
}

func buildListWhere(filter model.ListFilter) (string, []any) {
	clauses := []string{"tenant_id = $1"}
	args := []any{filter.TenantID}
	argIdx := 2
	if action := strings.TrimSpace(filter.Action); action != "" {
		clauses = append(clauses, fmt.Sprintf("action = $%d", argIdx))
		args = append(args, strings.ToUpper(action))
		argIdx++
	}
	if path := strings.TrimSpace(filter.Path); path != "" {
		clauses = append(clauses, fmt.Sprintf("path ILIKE $%d", argIdx))
		args = append(args, "%"+path+"%")
		argIdx++
	}
	if filter.From != nil {
		clauses = append(clauses, fmt.Sprintf("created_at >= $%d", argIdx))
		args = append(args, filter.From.UTC())
		argIdx++
	}
	if filter.To != nil {
		clauses = append(clauses, fmt.Sprintf("created_at < $%d", argIdx))
		args = append(args, filter.To.UTC())
		argIdx++
	}
	_ = argIdx
	return strings.Join(clauses, " AND "), args
}

func (r *Repository) ensurePartitionsForFilter(ctx context.Context, filter model.ListFilter) error {
	from := time.Now().UTC().AddDate(0, -1, 0)
	to := time.Now().UTC().AddDate(0, 1, 0)
	if filter.From != nil {
		from = filter.From.UTC()
	}
	if filter.To != nil {
		to = filter.To.UTC()
	}
	return partition.EnsureRange(ctx, r.pool, from, to)
}

func (r *Repository) CountByTenant(ctx context.Context, filter model.ListFilter) (int64, error) {
	if err := r.ensurePartitionsForFilter(ctx, filter); err != nil {
		return 0, err
	}
	where, args := buildListWhere(filter)
	query := fmt.Sprintf(`SELECT COUNT(*) FROM audit_events WHERE %s`, where)
	var total int64
	if err := r.pool.QueryRow(ctx, query, args...).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

func (r *Repository) ListByTenant(ctx context.Context, filter model.ListFilter) ([]model.AuditEvent, error) {
	if err := r.ensurePartitionsForFilter(ctx, filter); err != nil {
		return nil, err
	}
	limit := normalizeLimit(filter.Limit)
	offset := normalizeOffset(filter.Offset)
	where, args := buildListWhere(filter)
	query := fmt.Sprintf(`
		SELECT id, tenant_id, user_id, action, path, status_code, created_at
		FROM audit_events
		WHERE %s
		ORDER BY created_at DESC
		LIMIT %d OFFSET %d
	`, where, limit, offset)
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var events []model.AuditEvent
	for rows.Next() {
		var ev model.AuditEvent
		var uid *uuid.UUID
		if err := rows.Scan(&ev.ID, &ev.TenantID, &uid, &ev.Action, &ev.Path, &ev.StatusCode, &ev.CreatedAt); err != nil {
			return nil, err
		}
		ev.UserID = uid
		events = append(events, ev)
	}
	return events, rows.Err()
}
