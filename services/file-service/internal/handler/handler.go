package handler

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/gateframe/file-service/internal/model"
	"github.com/gateframe/file-service/internal/repository"
	"github.com/gateframe/file-service/internal/response"
	"github.com/gateframe/file-service/internal/storage"
	"github.com/gateframe/file-service/internal/validation"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	repo         *repository.Repository
	storage      *storage.Client
	maxFileBytes int64
}

func New(repo *repository.Repository, store *storage.Client, maxFileBytes int64) *Handler {
	if maxFileBytes <= 0 {
		maxFileBytes = validation.DefaultMaxFileBytes
	}
	return &Handler{repo: repo, storage: store, maxFileBytes: maxFileBytes}
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
		items = []model.FileObject{}
	}
	response.OK(c, items)
}

func (h *Handler) Create(c *gin.Context) {
	tenantID, err := tenantFromContext(c)
	if err != nil {
		response.Unauthorized(c)
		return
	}

	var req model.CreateFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if !validation.IsAllowedExtension(req.Filename) {
		response.BadRequest(c, "file type not allowed")
		return
	}

	content, err := base64.StdEncoding.DecodeString(strings.TrimSpace(req.ContentBase64))
	if err != nil {
		response.BadRequest(c, "invalid content_base64")
		return
	}

	if err := validation.ValidateUpload(req.Filename, int64(len(content)), h.maxFileBytes); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	fileID := uuid.New()
	objectKey := fmt.Sprintf("%s/%s/%s", tenantID.String(), fileID.String(), strings.TrimSpace(req.Filename))
	contentType := strings.TrimSpace(req.ContentType)
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	ctx := c.Request.Context()
	if err := h.storage.PutObject(ctx, objectKey, bytes.NewReader(content), int64(len(content)), contentType); err != nil {
		response.Internal(c)
		return
	}

	fileObj, err := h.repo.Create(ctx, tenantID, objectKey, req.Filename, contentType, int64(len(content)))
	if err != nil {
		response.Internal(c)
		return
	}
	response.Created(c, fileObj)
}

func (h *Handler) GetDownload(c *gin.Context) {
	tenantID, err := tenantFromContext(c)
	if err != nil {
		response.Unauthorized(c)
		return
	}

	fileID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid file id")
		return
	}

	fileObj, err := h.repo.GetByID(c.Request.Context(), tenantID, fileID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			response.NotFound(c)
			return
		}
		response.Internal(c)
		return
	}

	reader, contentType, size, err := h.storage.GetObject(c.Request.Context(), fileObj.ObjectKey)
	if err != nil {
		response.Internal(c)
		return
	}
	defer reader.Close()

	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fileObj.Filename))
	if size > 0 {
		c.Header("Content-Length", fmt.Sprintf("%d", size))
	}
	c.Status(200)
	if _, err := io.Copy(c.Writer, reader); err != nil {
		return
	}
}

func (h *Handler) Delete(c *gin.Context) {
	tenantID, err := tenantFromContext(c)
	if err != nil {
		response.Unauthorized(c)
		return
	}

	fileID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid file id")
		return
	}

	fileObj, err := h.repo.Delete(c.Request.Context(), tenantID, fileID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			response.NotFound(c)
			return
		}
		response.Internal(c)
		return
	}

	if err := h.storage.RemoveObject(c.Request.Context(), fileObj.ObjectKey); err != nil {
		response.Internal(c)
		return
	}

	response.OK(c, gin.H{"deleted": true, "id": fileObj.ID.String()})
}
