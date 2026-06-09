package middleware

import (
	"crypto/subtle"
	"strings"

	"github.com/gateframe/user-service/internal/response"
	"github.com/gin-gonic/gin"
)

func tokenMatches(got, expected string) bool {
	return subtle.ConstantTimeCompare([]byte(got), []byte(expected)) == 1
}

// RequireInternalToken rejects direct client access; Gateway must inject token.
func RequireInternalToken(expected string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("X-Internal-Token")
		if token == "" || !tokenMatches(token, expected) {
			response.Unauthorized(c)
			c.Abort()
			return
		}
		c.Next()
	}
}

// RequireIdentity parses Gateway-injected headers (never from JSON body).
func RequireIdentity() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := strings.TrimSpace(c.GetHeader("X-User-Id"))
		tenantID := strings.TrimSpace(c.GetHeader("X-Tenant-Id"))
		if userID == "" || tenantID == "" {
			response.Unauthorized(c)
			c.Abort()
			return
		}
		c.Set("user_id", userID)
		c.Set("tenant_id", tenantID)
		c.Next()
	}
}

func RequirePermission(code string) gin.HandlerFunc {
	return func(c *gin.Context) {
		perms, _ := c.Get("permissions")
		list, ok := perms.([]string)
		if !ok {
			response.Forbidden(c)
			c.Abort()
			return
		}
		for _, p := range list {
			if p == code || p == "*" {
				c.Next()
				return
			}
		}
		response.Forbidden(c)
		c.Abort()
	}
}
