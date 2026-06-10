package handler

import (
	"errors"
	"strings"

	"github.com/gateframe/workflow-service/internal/model"
	"github.com/gateframe/workflow-service/internal/repository"
	"github.com/gateframe/workflow-service/internal/response"
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

func tenantFromContext(c *gin.Context) (uuid.UUID, error) {
	tenantStr, _ := c.Get("tenant_id")
	return uuid.Parse(tenantStr.(string))
}

func actorFromContext(c *gin.Context) (*uuid.UUID, string) {
	raw, ok := c.Get("user_id")
	if !ok {
		return nil, ""
	}
	userStr, _ := raw.(string)
	userStr = strings.TrimSpace(userStr)
	if userStr == "" {
		return nil, ""
	}
	id, err := uuid.Parse(userStr)
	if err != nil {
		return nil, userStr
	}
	return &id, userStr
}

func (h *Handler) List(c *gin.Context) {
	tenantID, err := tenantFromContext(c)
	if err != nil {
		response.Unauthorized(c)
		return
	}

	items, err := h.repo.ListByTenant(c.Request.Context(), tenantID)
	if err != nil {
		response.Internal(c)
		return
	}
	if items == nil {
		items = []model.Workflow{}
	}
	response.OK(c, items)
}

func (h *Handler) Get(c *gin.Context) {
	tenantID, err := tenantFromContext(c)
	if err != nil {
		response.Unauthorized(c)
		return
	}
	workflowID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid workflow id")
		return
	}
	wf, err := h.repo.GetByID(c.Request.Context(), tenantID, workflowID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			response.NotFound(c)
			return
		}
		response.Internal(c)
		return
	}
	response.OK(c, wf)
}

func (h *Handler) ListEvents(c *gin.Context) {
	tenantID, err := tenantFromContext(c)
	if err != nil {
		response.Unauthorized(c)
		return
	}
	workflowID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid workflow id")
		return
	}
	items, err := h.repo.ListEvents(c.Request.Context(), tenantID, workflowID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			response.NotFound(c)
			return
		}
		response.Internal(c)
		return
	}
	if items == nil {
		items = []model.WorkflowEvent{}
	}
	response.OK(c, items)
}

func (h *Handler) Create(c *gin.Context) {
	tenantID, err := tenantFromContext(c)
	if err != nil {
		response.Unauthorized(c)
		return
	}
	var req model.CreateWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	actorID, _ := actorFromContext(c)
	if req.RequesterLabel == "" {
		req.RequesterLabel = c.GetHeader("X-User-Id")
	}
	wf, err := h.repo.Create(c.Request.Context(), tenantID, actorID, req)
	if err != nil {
		response.Internal(c)
		return
	}
	response.Created(c, wf)
}

func (h *Handler) Update(c *gin.Context) {
	tenantID, err := tenantFromContext(c)
	if err != nil {
		response.Unauthorized(c)
		return
	}
	workflowID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid workflow id")
		return
	}
	var req model.UpdateWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	wf, err := h.repo.Update(c.Request.Context(), tenantID, workflowID, req)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			response.NotFound(c)
			return
		}
		if errors.Is(err, repository.ErrLocked) {
			response.BadRequest(c, "cannot edit while pending review")
			return
		}
		if err.Error() == "name cannot be empty" {
			response.BadRequest(c, err.Error())
			return
		}
		response.Internal(c)
		return
	}
	response.OK(c, wf)
}

func (h *Handler) Delete(c *gin.Context) {
	tenantID, err := tenantFromContext(c)
	if err != nil {
		response.Unauthorized(c)
		return
	}
	workflowID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid workflow id")
		return
	}
	if err := h.repo.Delete(c.Request.Context(), tenantID, workflowID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			response.NotFound(c)
			return
		}
		if errors.Is(err, repository.ErrLocked) {
			response.BadRequest(c, "cannot delete while pending review")
			return
		}
		response.Internal(c)
		return
	}
	response.OK(c, gin.H{"deleted": true, "id": workflowID.String()})
}

func (h *Handler) bindReview(c *gin.Context) (model.ReviewWorkflowRequest, bool) {
	var req model.ReviewWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return req, false
	}
	return req, true
}

func (h *Handler) Submit(c *gin.Context) {
	tenantID, err := tenantFromContext(c)
	if err != nil {
		response.Unauthorized(c)
		return
	}
	workflowID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid workflow id")
		return
	}
	var req model.ReviewWorkflowRequest
	_ = c.ShouldBindJSON(&req)
	actorID, actorLabel := actorFromContext(c)
	if req.ActorLabel != "" {
		actorLabel = req.ActorLabel
	}
	wf, err := h.repo.Submit(c.Request.Context(), tenantID, workflowID, actorID, actorLabel)
	if err != nil {
		h.handleTransitionError(c, err)
		return
	}
	response.OK(c, wf)
}

func (h *Handler) Approve(c *gin.Context) {
	h.reviewAction(c, "approve")
}

func (h *Handler) Reject(c *gin.Context) {
	h.reviewAction(c, "reject")
}

func (h *Handler) RequestChanges(c *gin.Context) {
	h.reviewAction(c, "request_changes")
}

func (h *Handler) reviewAction(c *gin.Context, action string) {
	tenantID, err := tenantFromContext(c)
	if err != nil {
		response.Unauthorized(c)
		return
	}
	workflowID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid workflow id")
		return
	}
	req, ok := h.bindReview(c)
	if !ok {
		return
	}
	actorID, actorLabel := actorFromContext(c)
	if req.ActorLabel != "" {
		actorLabel = req.ActorLabel
	}

	var wf *model.Workflow
	switch action {
	case "approve":
		wf, err = h.repo.Approve(c.Request.Context(), tenantID, workflowID, actorID, actorLabel, req.Comment)
	case "reject":
		wf, err = h.repo.Reject(c.Request.Context(), tenantID, workflowID, actorID, actorLabel, req.Comment)
	case "request_changes":
		wf, err = h.repo.RequestChanges(c.Request.Context(), tenantID, workflowID, actorID, actorLabel, req.Comment)
	default:
		response.BadRequest(c, "unknown action")
		return
	}
	if err != nil {
		h.handleTransitionError(c, err)
		return
	}
	response.OK(c, wf)
}

func (h *Handler) handleTransitionError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		response.NotFound(c)
	case errors.Is(err, repository.ErrInvalidTransition):
		response.BadRequest(c, "invalid status transition")
	case errors.Is(err, repository.ErrCommentRequired):
		response.BadRequest(c, "comment is required")
	default:
		response.Internal(c)
	}
}
