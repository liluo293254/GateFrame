package model

import (
	"time"

	"github.com/google/uuid"
)

type Workflow struct {
	ID              uuid.UUID  `json:"id"`
	TenantID        uuid.UUID  `json:"tenant_id"`
	Name            string     `json:"name"`
	Description     string     `json:"description"`
	Category        string     `json:"category"`
	Priority        string     `json:"priority"`
	Status          string     `json:"status"`
	RequesterID     *uuid.UUID `json:"requester_id,omitempty"`
	RequesterLabel  string     `json:"requester_label"`
	ReviewerID      *uuid.UUID `json:"reviewer_id,omitempty"`
	ReviewComment   string     `json:"review_comment,omitempty"`
	SubmittedAt     *time.Time `json:"submitted_at,omitempty"`
	ReviewedAt      *time.Time `json:"reviewed_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type WorkflowEvent struct {
	ID          uuid.UUID  `json:"id"`
	WorkflowID  uuid.UUID  `json:"workflow_id"`
	TenantID    uuid.UUID  `json:"tenant_id"`
	ActorID     *uuid.UUID `json:"actor_id,omitempty"`
	ActorLabel  string     `json:"actor_label"`
	EventType   string     `json:"event_type"`
	FromStatus  string     `json:"from_status"`
	ToStatus    string     `json:"to_status"`
	Comment     string     `json:"comment,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}
