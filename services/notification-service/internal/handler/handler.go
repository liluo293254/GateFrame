package handler

import (
	"errors"
	"strings"

	"github.com/gateframe/notification-service/internal/model"
	"github.com/gateframe/notification-service/internal/repository"
	"github.com/gateframe/notification-service/internal/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	repo *repository.Repository
}

func New(repo *repository.Repository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) Health(c *gin.Context) {
	response.OK(c, gin.H{"status": "ok"})
}

func (h *Handler) List(c *gin.Context) {
	tenantStr, _ := c.Get("tenant_id")
	tenantID, err := uuid.Parse(tenantStr.(string))
	if err != nil {
		response.Unauthorized(c)
		return
	}

	var userID *uuid.UUID
	if userStr := strings.TrimSpace(c.GetHeader("X-User-Id")); userStr != "" {
		uid, err := uuid.Parse(userStr)
		if err == nil {
			userID = &uid
		}
	}

	items, err := h.repo.ListForUser(c.Request.Context(), tenantID, userID)
	if err != nil {
		response.Internal(c)
		return
	}
	if items == nil {
		items = []model.Notification{}
	}
	response.OK(c, items)
}

func (h *Handler) Create(c *gin.Context) {
	tenantStr, _ := c.Get("tenant_id")
	tenantID, err := uuid.Parse(tenantStr.(string))
	if err != nil {
		response.Unauthorized(c)
		return
	}
	var req model.CreateNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	var userID *uuid.UUID
	if uidStr := strings.TrimSpace(req.UserID); uidStr != "" {
		uid, err := uuid.Parse(uidStr)
		if err != nil {
			response.BadRequest(c, "invalid user_id")
			return
		}
		userID = &uid
	}
	n, err := h.repo.Create(c.Request.Context(), tenantID, userID, req.Title, req.Body)
	if err != nil {
		response.Internal(c)
		return
	}
	response.Created(c, n)
}

func (h *Handler) MarkRead(c *gin.Context) {
	tenantStr, _ := c.Get("tenant_id")
	tenantID, err := uuid.Parse(tenantStr.(string))
	if err != nil {
		response.Unauthorized(c)
		return
	}
	notificationID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid notification id")
		return
	}
	n, err := h.repo.MarkRead(c.Request.Context(), tenantID, notificationID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			response.NotFound(c)
			return
		}
		response.Internal(c)
		return
	}
	response.OK(c, n)
}
