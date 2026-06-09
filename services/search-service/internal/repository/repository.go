package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/gateframe/search-service/internal/model"
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

func (r *Repository) Search(ctx context.Context, tenantID uuid.UUID, query string) ([]model.SearchDocument, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return r.ListByTenant(ctx, tenantID)
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id, tenant_id, title, content, created_at
		FROM search_documents
		WHERE tenant_id = $1
		  AND to_tsvector('english', coalesce(title, '') || ' ' || coalesce(content, ''))
		      @@ plainto_tsquery('english', $2)
		ORDER BY ts_rank(
			to_tsvector('english', coalesce(title, '') || ' ' || coalesce(content, '')),
			plainto_tsquery('english', $2)
		) DESC, created_at DESC
		LIMIT 50
	`, tenantID, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanDocuments(rows)
}

func (r *Repository) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]model.SearchDocument, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, tenant_id, title, content, created_at
		FROM search_documents
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT 50
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanDocuments(rows)
}

func scanDocuments(rows pgxRows) ([]model.SearchDocument, error) {
	var items []model.SearchDocument
	for rows.Next() {
		var doc model.SearchDocument
		if err := rows.Scan(&doc.ID, &doc.TenantID, &doc.Title, &doc.Content, &doc.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, doc)
	}
	return items, rows.Err()
}

func (r *Repository) Create(ctx context.Context, tenantID uuid.UUID, title, content string) (*model.SearchDocument, error) {
	var doc model.SearchDocument
	err := r.pool.QueryRow(ctx, `
		INSERT INTO search_documents (tenant_id, title, content)
		VALUES ($1, $2, $3)
		RETURNING id, tenant_id, title, content, created_at
	`, tenantID, title, content).Scan(&doc.ID, &doc.TenantID, &doc.Title, &doc.Content, &doc.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

func (r *Repository) Delete(ctx context.Context, tenantID, documentID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `
		DELETE FROM search_documents WHERE id = $1 AND tenant_id = $2
	`, documentID, tenantID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) GetByID(ctx context.Context, tenantID, documentID uuid.UUID) (*model.SearchDocument, error) {
	var doc model.SearchDocument
	err := r.pool.QueryRow(ctx, `
		SELECT id, tenant_id, title, content, created_at
		FROM search_documents
		WHERE id = $1 AND tenant_id = $2
	`, documentID, tenantID).Scan(&doc.ID, &doc.TenantID, &doc.Title, &doc.Content, &doc.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &doc, nil
}

type pgxRows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
	Close()
}
