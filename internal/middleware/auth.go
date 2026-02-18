package middleware

import (
	"horeka/internal/auth"
	"horeka/internal/response"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const ctxUserIDKey = "userID"

func AuthRequired(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := strings.TrimSpace(c.GetHeader("Authorization"))
		if h == "" {
			response.Err(c, http.StatusUnauthorized, "AUTH_HEADER_MISSING", "missing Authorization header")
			c.Abort()
			return
		}

		parts := strings.SplitN(h, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
			response.Err(c, http.StatusUnauthorized, "AUTH_HEADER_INVALID", "invalid Authorization header")
			c.Abort()
			return
		}

		userID, err := auth.ParseToken(parts[1], secret)
		if err != nil {
			response.Err(c, http.StatusUnauthorized, "TOKEN_INVALID", "invalid token")
			c.Abort()
			return
		}

		c.Set(ctxUserIDKey, userID)
		c.Next()
	}
}

func CurrentUserID(c *gin.Context) (int64, bool) {
	v, ok := c.Get(ctxUserIDKey)
	if !ok {
		return 0, false
	}
	id, ok := v.(int64)
	return id, ok
}
