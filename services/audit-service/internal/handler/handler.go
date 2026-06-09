package handler

import (
	"strconv"
	"strings"
	"time"

	"github.com/gateframe/audit-service/internal/config"
	"github.com/gateframe/audit-service/internal/model"
	"github.com/gateframe/audit-service/internal/repository"
	"github.com/gateframe/audit-service/internal/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	repo   *repository.Repository
	config config.Config
}

func New(repo *repository.Repository, cfg config.Config) *Handler {
	return &Handler{repo: repo, config: cfg}
}

func (h *Handler) Health(c *gin.Context) {
	response.OK(c, gin.H{"status": "ok"})
}

func (h *Handler) CreateInternal(c *gin.Context) {
	var req model.CreateEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		response.BadRequest(c, "invalid tenant_id")
		return
	}
	var userID *uuid.UUID
	if req.UserID != "" {
		uid, err := uuid.Parse(req.UserID)
		if err != nil {
			response.BadRequest(c, "invalid user_id")
			return
		}
		userID = &uid
	}
	ev, err := h.repo.Insert(c.Request.Context(), tenantID, userID, req.Action, req.Path, req.StatusCode)
	if err != nil {
		response.Internal(c)
		return
	}
	response.OK(c, ev)
}

func parseDateBound(raw string, endOfDay bool) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return nil, err
	}
	t = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	if endOfDay {
		t = t.AddDate(0, 0, 1)
	}
	return &t, nil
}

func (h *Handler) List(c *gin.Context) {
	tenantStr, _ := c.Get("tenant_id")
	tenantID, err := uuid.Parse(tenantStr.(string))
	if err != nil {
		response.Unauthorized(c)
		return
	}
	limit := h.config.DefaultLimit
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	offset := 0
	if v := c.Query("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			offset = n
		}
	}
	from, err := parseDateBound(c.Query("from"), false)
	if err != nil {
		response.BadRequest(c, "invalid from date, use YYYY-MM-DD")
		return
	}
	to, err := parseDateBound(c.Query("to"), true)
	if err != nil {
		response.BadRequest(c, "invalid to date, use YYYY-MM-DD")
		return
	}

	filter := model.ListFilter{
		TenantID: tenantID,
		Action:   strings.TrimSpace(c.Query("action")),
		Path:     strings.TrimSpace(c.Query("path")),
		From:     from,
		To:       to,
		Limit:    limit,
		Offset:   offset,
	}
	total, err := h.repo.CountByTenant(c.Request.Context(), filter)
	if err != nil {
		response.Internal(c)
		return
	}
	events, err := h.repo.ListByTenant(c.Request.Context(), filter)
	if err != nil {
		response.Internal(c)
		return
	}
	if events == nil {
		events = []model.AuditEvent{}
	}
	response.OK(c, model.AuditListResult{
		Items:  events,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	})
}
