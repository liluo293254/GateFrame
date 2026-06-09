package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/gateframe/file-service/internal/model"
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

func (r *Repository) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]model.FileObject, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, tenant_id, object_key, filename, content_type, size_bytes, created_at
		FROM file_objects
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.FileObject
	for rows.Next() {
		var f model.FileObject
		if err := rows.Scan(
			&f.ID, &f.TenantID, &f.ObjectKey, &f.Filename, &f.ContentType, &f.SizeBytes, &f.CreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, f)
	}
	return items, rows.Err()
}

func (r *Repository) Create(ctx context.Context, tenantID uuid.UUID, objectKey, filename, contentType string, sizeBytes int64) (*model.FileObject, error) {
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	now := time.Now().UTC()
	var f model.FileObject
	err := r.pool.QueryRow(ctx, `
		INSERT INTO file_objects (tenant_id, object_key, filename, content_type, size_bytes, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, tenant_id, object_key, filename, content_type, size_bytes, created_at
	`, tenantID, objectKey, strings.TrimSpace(filename), contentType, sizeBytes, now).Scan(
		&f.ID, &f.TenantID, &f.ObjectKey, &f.Filename, &f.ContentType, &f.SizeBytes, &f.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (r *Repository) GetByID(ctx context.Context, tenantID, fileID uuid.UUID) (*model.FileObject, error) {
	var f model.FileObject
	err := r.pool.QueryRow(ctx, `
		SELECT id, tenant_id, object_key, filename, content_type, size_bytes, created_at
		FROM file_objects
		WHERE id = $1 AND tenant_id = $2
	`, fileID, tenantID).Scan(
		&f.ID, &f.TenantID, &f.ObjectKey, &f.Filename, &f.ContentType, &f.SizeBytes, &f.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &f, nil
}

func (r *Repository) Delete(ctx context.Context, tenantID, fileID uuid.UUID) (*model.FileObject, error) {
	fileObj, err := r.GetByID(ctx, tenantID, fileID)
	if err != nil {
		return nil, err
	}
	tag, err := r.pool.Exec(ctx, `DELETE FROM file_objects WHERE id = $1 AND tenant_id = $2`, fileID, tenantID)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	return fileObj, nil
}
