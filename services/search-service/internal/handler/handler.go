package handler

import (
	"errors"
	"strings"

	"github.com/gateframe/search-service/internal/model"
	"github.com/gateframe/search-service/internal/repository"
	"github.com/gateframe/search-service/internal/response"
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

func (h *Handler) Search(c *gin.Context) {
	tenantStr, _ := c.Get("tenant_id")
	tenantID, err := uuid.Parse(tenantStr.(string))
	if err != nil {
		response.Unauthorized(c)
		return
	}

	query := strings.TrimSpace(c.Query("q"))
	items, err := h.repo.Search(c.Request.Context(), tenantID, query)
	if err != nil {
		response.Internal(c)
		return
	}
	if items == nil {
		items = []model.SearchDocument{}
	}
	response.OK(c, model.SearchResult{
		Query: query,
		Items: items,
		Total: len(items),
	})
}

func (h *Handler) CreateDocument(c *gin.Context) {
	tenantStr, _ := c.Get("tenant_id")
	tenantID, err := uuid.Parse(tenantStr.(string))
	if err != nil {
		response.Unauthorized(c)
		return
	}
	var req model.CreateDocumentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	doc, err := h.repo.Create(c.Request.Context(), tenantID, req.Title, req.Content)
	if err != nil {
		response.Internal(c)
		return
	}
	response.Created(c, doc)
}

func (h *Handler) DeleteDocument(c *gin.Context) {
	tenantStr, _ := c.Get("tenant_id")
	tenantID, err := uuid.Parse(tenantStr.(string))
	if err != nil {
		response.Unauthorized(c)
		return
	}
	documentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid document id")
		return
	}
	if err := h.repo.Delete(c.Request.Context(), tenantID, documentID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			response.NotFound(c)
			return
		}
		response.Internal(c)
		return
	}
	response.OK(c, gin.H{"deleted": true, "id": documentID.String()})
}
