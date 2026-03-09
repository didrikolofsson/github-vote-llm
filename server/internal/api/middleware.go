package api

import (
	"github.com/gin-gonic/gin"
)

// ValidateAPIKey returns a middleware that checks the X-Api-Key header against
// the API_KEY environment variable. Returns 503 if API_KEY is not configured,
// 401 if the header is missing or incorrect.
func ValidateAPIKey(apiKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if apiKey == "" {
			c.AbortWithStatusJSON(503, gin.H{"error": "api key not configured"})
			return
		}
		if c.GetHeader("X-Api-Key") != apiKey {
			c.AbortWithStatusJSON(401, gin.H{"error": "unauthorized"})
			return
		}
		c.Next()
	}
}
