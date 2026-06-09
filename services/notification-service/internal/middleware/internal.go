package middleware

import (
	"crypto/subtle"
	"strings"

	"github.com/gateframe/notification-service/internal/response"
	"github.com/gin-gonic/gin"
)

func tokenMatches(got, expected string) bool {
	return subtle.ConstantTimeCompare([]byte(got), []byte(expected)) == 1
}

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

func RequireIdentity() gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := strings.TrimSpace(c.GetHeader("X-Tenant-Id"))
		if tenantID == "" {
			response.Unauthorized(c)
			c.Abort()
			return
		}
		c.Set("tenant_id", tenantID)
		c.Set("user_id", strings.TrimSpace(c.GetHeader("X-User-Id")))
		c.Next()
	}
}

func LoadPermissionsFromGateway() gin.HandlerFunc {
	return func(c *gin.Context) {
		list := parsePermissions(c.GetHeader("X-Permissions"))
		if len(list) == 0 {
			response.Forbidden(c)
			c.Abort()
			return
		}
		c.Set("permissions", list)
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

func parsePermissions(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	list := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			list = append(list, p)
		}
	}
	return list
}
