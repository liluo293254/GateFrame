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

var (
	ErrInvalidTransition = errors.New("invalid transition")
	ErrCommentRequired   = errors.New("comment required")
	ErrLocked            = errors.New("workflow locked while pending review")
)

var allowedStatuses = map[string]struct{}{
	"draft":              {},
	"pending":            {},
	"approved":           {},
	"rejected":           {},
	"changes_requested":  {},
}

var allowedCategories = map[string]struct{}{
	"general":     {},
	"expense":     {},
	"leave":       {},
	"procurement": {},
}

var allowedPriorities = map[string]struct{}{
	"low":    {},
	"normal": {},
	"high":   {},
}

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func normalizeStatus(status string) string {
	status = strings.TrimSpace(strings.ToLower(status))
	if _, ok := allowedStatuses[status]; ok {
		return status
	}
	return "draft"
}

func normalizeCategory(category string) string {
	category = strings.TrimSpace(strings.ToLower(category))
	if _, ok := allowedCategories[category]; ok {
		return category
	}
	return "general"
}

func normalizePriority(priority string) string {
	priority = strings.TrimSpace(strings.ToLower(priority))
	if _, ok := allowedPriorities[priority]; ok {
		return priority
	}
	return "normal"
}

const workflowSelect = `
	SELECT id, tenant_id, name, description, category, priority, status,
	       requester_id, requester_label, reviewer_id, review_comment,
	       submitted_at, reviewed_at, created_at, updated_at
	FROM workflows
`

func scanWorkflow(row pgx.Row) (model.Workflow, error) {
	var w model.Workflow
	err := row.Scan(
		&w.ID, &w.TenantID, &w.Name, &w.Description, &w.Category, &w.Priority, &w.Status,
		&w.RequesterID, &w.RequesterLabel, &w.ReviewerID, &w.ReviewComment,
		&w.SubmittedAt, &w.ReviewedAt, &w.CreatedAt, &w.UpdatedAt,
	)
	return w, err
}

func (r *Repository) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]model.Workflow, error) {
	rows, err := r.pool.Query(ctx, workflowSelect+`
		WHERE tenant_id = $1
		ORDER BY
			CASE status WHEN 'pending' THEN 0 WHEN 'changes_requested' THEN 1 WHEN 'draft' THEN 2 ELSE 3 END,
			updated_at DESC
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.Workflow
	for rows.Next() {
		w, err := scanWorkflow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, w)
	}
	return items, rows.Err()
}

func (r *Repository) GetByID(ctx context.Context, tenantID, workflowID uuid.UUID) (*model.Workflow, error) {
	w, err := scanWorkflow(r.pool.QueryRow(ctx, workflowSelect+` WHERE id = $1 AND tenant_id = $2`, workflowID, tenantID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &w, nil
}

func (r *Repository) Create(ctx context.Context, tenantID uuid.UUID, requesterID *uuid.UUID, req model.CreateWorkflowRequest) (*model.Workflow, error) {
	now := time.Now().UTC()
	var w model.Workflow
	err := r.pool.QueryRow(ctx, `
		INSERT INTO workflows (
			tenant_id, name, description, category, priority, status,
			requester_id, requester_label, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, 'draft', $6, $7, $8, $8)
		RETURNING id, tenant_id, name, description, category, priority, status,
		          requester_id, requester_label, reviewer_id, review_comment,
		          submitted_at, reviewed_at, created_at, updated_at
	`, tenantID, strings.TrimSpace(req.Name), req.Description,
		normalizeCategory(req.Category), normalizePriority(req.Priority),
		requesterID, strings.TrimSpace(req.RequesterLabel), now,
	).Scan(
		&w.ID, &w.TenantID, &w.Name, &w.Description, &w.Category, &w.Priority, &w.Status,
		&w.RequesterID, &w.RequesterLabel, &w.ReviewerID, &w.ReviewComment,
		&w.SubmittedAt, &w.ReviewedAt, &w.CreatedAt, &w.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	_ = r.insertEvent(ctx, w.ID, tenantID, requesterID, req.RequesterLabel, "created", "", "draft", "")
	return &w, nil
}

func editableStatus(status string) bool {
	return status == "draft" || status == "changes_requested"
}

func (r *Repository) Update(ctx context.Context, tenantID, workflowID uuid.UUID, req model.UpdateWorkflowRequest) (*model.Workflow, error) {
	current, err := r.GetByID(ctx, tenantID, workflowID)
	if err != nil {
		return nil, err
	}
	if !editableStatus(current.Status) {
		return nil, ErrLocked
	}

	name := ""
	if req.Name != nil {
		name = strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, errors.New("name cannot be empty")
		}
	}
	category := req.Category
	if category != nil {
		normalized := normalizeCategory(*category)
		category = &normalized
	}
	priority := req.Priority
	if priority != nil {
		normalized := normalizePriority(*priority)
		priority = &normalized
	}

	w, err := scanWorkflow(r.pool.QueryRow(ctx, `
		UPDATE workflows
		SET
			name = COALESCE(NULLIF($3, ''), name),
			description = COALESCE($4, description),
			category = COALESCE($5, category),
			priority = COALESCE($6, priority),
			updated_at = NOW()
		WHERE id = $1 AND tenant_id = $2
		RETURNING id, tenant_id, name, description, category, priority, status,
		          requester_id, requester_label, reviewer_id, review_comment,
		          submitted_at, reviewed_at, created_at, updated_at
	`, workflowID, tenantID, name, req.Description, category, priority))
	if err != nil {
		return nil, err
	}
	return &w, nil
}

func (r *Repository) Delete(ctx context.Context, tenantID, workflowID uuid.UUID) error {
	current, err := r.GetByID(ctx, tenantID, workflowID)
	if err != nil {
		return err
	}
	if current.Status == "pending" {
		return ErrLocked
	}
	tag, err := r.pool.Exec(ctx, `DELETE FROM workflows WHERE id = $1 AND tenant_id = $2`, workflowID, tenantID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) ListEvents(ctx context.Context, tenantID, workflowID uuid.UUID) ([]model.WorkflowEvent, error) {
	if _, err := r.GetByID(ctx, tenantID, workflowID); err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, workflow_id, tenant_id, actor_id, actor_label, event_type,
		       from_status, to_status, comment, created_at
		FROM workflow_events
		WHERE workflow_id = $1 AND tenant_id = $2
		ORDER BY created_at ASC
	`, workflowID, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.WorkflowEvent
	for rows.Next() {
		var e model.WorkflowEvent
		if err := rows.Scan(
			&e.ID, &e.WorkflowID, &e.TenantID, &e.ActorID, &e.ActorLabel,
			&e.EventType, &e.FromStatus, &e.ToStatus, &e.Comment, &e.CreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, e)
	}
	return items, rows.Err()
}

func (r *Repository) insertEvent(
	ctx context.Context,
	workflowID, tenantID uuid.UUID,
	actorID *uuid.UUID,
	actorLabel, eventType, fromStatus, toStatus, comment string,
) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO workflow_events (
			workflow_id, tenant_id, actor_id, actor_label, event_type,
			from_status, to_status, comment
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, workflowID, tenantID, actorID, actorLabel, eventType, fromStatus, toStatus, comment)
	return err
}

func (r *Repository) transition(
	ctx context.Context,
	tenantID, workflowID uuid.UUID,
	actorID *uuid.UUID,
	actorLabel, eventType, fromStatus, toStatus, comment string,
	applyReview bool,
) (*model.Workflow, error) {
	current, err := r.GetByID(ctx, tenantID, workflowID)
	if err != nil {
		return nil, err
	}
	if current.Status != fromStatus {
		return nil, ErrInvalidTransition
	}

	now := time.Now().UTC()
	var w model.Workflow
	query := `
		UPDATE workflows
		SET status = $3, updated_at = $4,
		    submitted_at = CASE WHEN $5 THEN $4 ELSE submitted_at END,
		    reviewed_at = CASE WHEN $6 THEN $4 ELSE reviewed_at END,
		    reviewer_id = CASE WHEN $6 THEN $7 ELSE reviewer_id END,
		    review_comment = CASE WHEN $6 THEN $8 ELSE review_comment END
		WHERE id = $1 AND tenant_id = $2 AND status = $9
		RETURNING id, tenant_id, name, description, category, priority, status,
		          requester_id, requester_label, reviewer_id, review_comment,
		          submitted_at, reviewed_at, created_at, updated_at
	`
	submit := toStatus == "pending"
	err = r.pool.QueryRow(ctx, query,
		workflowID, tenantID, toStatus, now, submit, applyReview, actorID, comment, fromStatus,
	).Scan(
		&w.ID, &w.TenantID, &w.Name, &w.Description, &w.Category, &w.Priority, &w.Status,
		&w.RequesterID, &w.RequesterLabel, &w.ReviewerID, &w.ReviewComment,
		&w.SubmittedAt, &w.ReviewedAt, &w.CreatedAt, &w.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidTransition
		}
		return nil, err
	}
	if err := r.insertEvent(ctx, workflowID, tenantID, actorID, actorLabel, eventType, fromStatus, toStatus, comment); err != nil {
		return nil, err
	}
	return &w, nil
}

func (r *Repository) Submit(ctx context.Context, tenantID, workflowID uuid.UUID, actorID *uuid.UUID, actorLabel string) (*model.Workflow, error) {
	current, err := r.GetByID(ctx, tenantID, workflowID)
	if err != nil {
		return nil, err
	}
	from := current.Status
	if from != "draft" && from != "changes_requested" {
		return nil, ErrInvalidTransition
	}
	return r.transition(ctx, tenantID, workflowID, actorID, actorLabel, "submitted", from, "pending", "", false)
}

func (r *Repository) Approve(ctx context.Context, tenantID, workflowID uuid.UUID, actorID *uuid.UUID, actorLabel, comment string) (*model.Workflow, error) {
	return r.transition(ctx, tenantID, workflowID, actorID, actorLabel, "approved", "pending", "approved", comment, true)
}

func (r *Repository) Reject(ctx context.Context, tenantID, workflowID uuid.UUID, actorID *uuid.UUID, actorLabel, comment string) (*model.Workflow, error) {
	if strings.TrimSpace(comment) == "" {
		return nil, ErrCommentRequired
	}
	return r.transition(ctx, tenantID, workflowID, actorID, actorLabel, "rejected", "pending", "rejected", comment, true)
}

func (r *Repository) RequestChanges(ctx context.Context, tenantID, workflowID uuid.UUID, actorID *uuid.UUID, actorLabel, comment string) (*model.Workflow, error) {
	if strings.TrimSpace(comment) == "" {
		return nil, ErrCommentRequired
	}
	return r.transition(ctx, tenantID, workflowID, actorID, actorLabel, "changes_requested", "pending", "changes_requested", comment, true)
}
