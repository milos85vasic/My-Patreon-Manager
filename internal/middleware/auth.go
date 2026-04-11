package middleware

import (
	"crypto/subtle"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// Auth returns a Gin middleware that guards /admin/* paths with an
// X-Admin-Key bearer check. Preserved for backwards compatibility with the
// original package-level Auth middleware — non-admin paths pass through
// without inspection so it can be installed at the engine level. For routes
// that must always require the key (e.g. /debug/pprof behind admin auth),
// use RequireAdminKey installed on the route group instead.
func Auth(adminKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if adminKey == "" {
			adminKey = os.Getenv("ADMIN_KEY")
		}

		path := c.Request.URL.Path
		if len(path) >= 7 && path[:7] == "/admin/" {
			key := c.GetHeader("X-Admin-Key")
			if key == "" {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "admin key required"})
				return
			}
			if subtle.ConstantTimeCompare([]byte(key), []byte(adminKey)) != 1 {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "invalid admin key"})
				return
			}
		}

		c.Next()
	}
}

// RequireAdminKey returns a Gin middleware that unconditionally requires
// the X-Admin-Key header to match adminKey. Unlike Auth, it does not
// inspect the request path, so it is safe to mount on arbitrary route
// groups that must be admin-gated (for example /debug/pprof). When
// adminKey is empty the env fallback (ADMIN_KEY) is used, and when both
// are empty all requests are rejected with 401 so an unconfigured
// deployment fails closed.
func RequireAdminKey(adminKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if adminKey == "" {
			adminKey = os.Getenv("ADMIN_KEY")
		}
		if adminKey == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "admin key required"})
			return
		}
		key := c.GetHeader("X-Admin-Key")
		if key == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "admin key required"})
			return
		}
		if subtle.ConstantTimeCompare([]byte(key), []byte(adminKey)) != 1 {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "invalid admin key"})
			return
		}
		c.Next()
	}
}
