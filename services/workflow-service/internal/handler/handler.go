package handler

import (
	"errors"

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
	wf, err := h.repo.Create(c.Request.Context(), tenantID, req)
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
		response.Internal(c)
		return
	}
	response.OK(c, gin.H{"deleted": true, "id": workflowID.String()})
}
