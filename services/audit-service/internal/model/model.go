package model

import (
	"time"

	"github.com/google/uuid"
)

type AuditEvent struct {
	ID         uuid.UUID  `json:"id"`
	TenantID   uuid.UUID  `json:"tenant_id"`
	UserID     *uuid.UUID `json:"user_id,omitempty"`
	Action     string     `json:"action"`
	Path       string     `json:"path"`
	StatusCode int        `json:"status_code"`
	CreatedAt  time.Time  `json:"created_at"`
}

type CreateEventRequest struct {
	TenantID   string `json:"tenant_id" binding:"required,uuid"`
	UserID     string `json:"user_id" binding:"omitempty,uuid"`
	Action     string `json:"action" binding:"required,max=16"`
	Path       string `json:"path" binding:"required,max=512"`
	StatusCode int    `json:"status_code" binding:"required"`
}

type ListFilter struct {
	TenantID uuid.UUID
	Action   string
	Path     string
	From     *time.Time
	To       *time.Time
	Limit    int
	Offset   int
}

type AuditListResult struct {
	Items  []AuditEvent `json:"items"`
	Total  int64        `json:"total"`
	Limit  int          `json:"limit"`
	Offset int          `json:"offset"`
}
