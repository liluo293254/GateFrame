package model

import (
	"time"

	"github.com/google/uuid"
)

type SearchDocument struct {
	ID        uuid.UUID `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type SearchResult struct {
	Query string           `json:"query"`
	Items []SearchDocument `json:"items"`
	Total int              `json:"total"`
}
