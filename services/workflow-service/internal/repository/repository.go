package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/gateframe/workflow-service/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var allowedStatuses = map[string]struct{}{
	"draft":    {},
	"active":   {},
	"archived": {},
}

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func normalizeStatus(status string) string {
	status = strings.TrimSpace(strings.ToLower(status))
	if status == "" {
		return "draft"
	}
	if _, ok := allowedStatuses[status]; ok {
		return status
	}
	return "draft"
}

func (r *Repository) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]model.Workflow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, tenant_id, name, description, status, created_at, updated_at
		FROM workflows
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.Workflow
	for rows.Next() {
		var w model.Workflow
		if err := rows.Scan(
			&w.ID, &w.TenantID, &w.Name, &w.Description, &w.Status, &w.CreatedAt, &w.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, w)
	}
	return items, rows.Err()
}

func (r *Repository) Create(ctx context.Context, tenantID uuid.UUID, req model.CreateWorkflowRequest) (*model.Workflow, error) {
	now := time.Now().UTC()
	var w model.Workflow
	err := r.pool.QueryRow(ctx, `
		INSERT INTO workflows (tenant_id, name, description, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $5)
		RETURNING id, tenant_id, name, description, status, created_at, updated_at
	`, tenantID, strings.TrimSpace(req.Name), req.Description, normalizeStatus(req.Status), now).Scan(
		&w.ID, &w.TenantID, &w.Name, &w.Description, &w.Status, &w.CreatedAt, &w.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &w, nil
}

func (r *Repository) Update(ctx context.Context, tenantID, workflowID uuid.UUID, req model.UpdateWorkflowRequest) (*model.Workflow, error) {
	name := ""
	if req.Name != nil {
		name = strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, errors.New("name cannot be empty")
		}
	}
	description := req.Description
	status := req.Status
	if status != nil {
		normalized := normalizeStatus(*status)
		status = &normalized
	}

	var w model.Workflow
	err := r.pool.QueryRow(ctx, `
		UPDATE workflows
		SET
			name = COALESCE(NULLIF($3, ''), name),
			description = COALESCE($4, description),
			status = COALESCE($5, status),
			updated_at = NOW()
		WHERE id = $1 AND tenant_id = $2
		RETURNING id, tenant_id, name, description, status, created_at, updated_at
	`, workflowID, tenantID, name, description, status).Scan(
		&w.ID, &w.TenantID, &w.Name, &w.Description, &w.Status, &w.CreatedAt, &w.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &w, nil
}

func (r *Repository) Delete(ctx context.Context, tenantID, workflowID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM workflows WHERE id = $1 AND tenant_id = $2`, workflowID, tenantID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
