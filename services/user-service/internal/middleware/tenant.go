package middleware

import (
	"github.com/gateframe/user-service/internal/repository"
	"github.com/gateframe/user-service/internal/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequireActiveTenant rejects requests when the caller's tenant is not active.
func RequireActiveTenant(repo *repository.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantStr, ok := c.Get("tenant_id")
		if !ok {
			response.Unauthorized(c)
			c.Abort()
			return
		}
		tenantID, err := uuid.Parse(tenantStr.(string))
		if err != nil {
			response.Unauthorized(c)
			c.Abort()
			return
		}
		status, err := repo.GetTenantStatus(c.Request.Context(), tenantID)
		if err != nil || status != "active" {
			response.Unauthorized(c)
			c.Abort()
			return
		}
		c.Next()
	}
}
