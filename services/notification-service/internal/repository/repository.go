package repository

import (
	"context"
	"errors"
	"time"

	"github.com/gateframe/notification-service/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) ListForUser(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID) ([]model.Notification, error) {
	var rows pgxRows
	var err error
	if userID == nil {
		rows, err = r.pool.Query(ctx, `
			SELECT id, tenant_id, user_id, title, body, read_at, created_at
			FROM notifications
			WHERE tenant_id = $1
			ORDER BY created_at DESC
			LIMIT 100
		`, tenantID)
	} else {
		rows, err = r.pool.Query(ctx, `
			SELECT id, tenant_id, user_id, title, body, read_at, created_at
			FROM notifications
			WHERE tenant_id = $1
			  AND (user_id IS NULL OR user_id = $2)
			ORDER BY created_at DESC
			LIMIT 100
		`, tenantID, *userID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.Notification
	for rows.Next() {
		var n model.Notification
		if err := rows.Scan(&n.ID, &n.TenantID, &n.UserID, &n.Title, &n.Body, &n.ReadAt, &n.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, n)
	}
	return items, rows.Err()
}

func (r *Repository) Create(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID, title, body string) (*model.Notification, error) {
	var n model.Notification
	err := r.pool.QueryRow(ctx, `
		INSERT INTO notifications (tenant_id, user_id, title, body)
		VALUES ($1, $2, $3, $4)
		RETURNING id, tenant_id, user_id, title, body, read_at, created_at
	`, tenantID, userID, title, body).Scan(
		&n.ID, &n.TenantID, &n.UserID, &n.Title, &n.Body, &n.ReadAt, &n.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &n, nil
}

func (r *Repository) MarkRead(ctx context.Context, tenantID, notificationID uuid.UUID) (*model.Notification, error) {
	now := time.Now().UTC()
	var n model.Notification
	err := r.pool.QueryRow(ctx, `
		UPDATE notifications
		SET read_at = $3
		WHERE id = $1 AND tenant_id = $2
		RETURNING id, tenant_id, user_id, title, body, read_at, created_at
	`, notificationID, tenantID, now).Scan(
		&n.ID, &n.TenantID, &n.UserID, &n.Title, &n.Body, &n.ReadAt, &n.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &n, nil
}

type pgxRows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
	Close()
}
