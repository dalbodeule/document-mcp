package httpapi

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	ctxUserIDKey   = "userID"
	ctxGroupIDsKey = "groupIDs"
)

func AccessTokenMiddleware(deps Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if h == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing Authorization header"})
			return
		}
		parts := strings.SplitN(h, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid Authorization header"})
			return
		}
		tok := parts[1]
		claims, err := deps.Auth.ParseAccessToken(tok)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		c.Set(ctxUserIDKey, claims.UserID)
		c.Set(ctxGroupIDsKey, claims.GroupIDs)
		c.Next()
	}
}

func MustUserID(c *gin.Context) string {
	v, ok := c.Get(ctxUserIDKey)
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

func MustGroupIDs(c *gin.Context) []string {
	v, ok := c.Get(ctxGroupIDsKey)
	if !ok {
		return nil
	}
	ids, _ := v.([]string)
	return ids
}
