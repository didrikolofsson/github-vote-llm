package middleware

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	gh "github.com/google/go-github/v68/github"
)

func ValidateSignature() gin.HandlerFunc {
	return func(c *gin.Context) {
		payload, err := io.ReadAll(c.Request.Body)
		if err != nil {
			log.Printf("failed to read body: %v", err)
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "bad request"})
			return
		}

		if err := gh.ValidateSignature(c.GetHeader("X-Hub-Signature-256"), payload, []byte(os.Getenv("WEBHOOK_SECRET"))); err != nil {
			log.Printf("invalid signature: %v", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
			return
		}

		c.Request.Body = io.NopCloser(bytes.NewBuffer(payload))
		c.Next()
	}
}
