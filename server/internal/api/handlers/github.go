package handlers

import (
	"net/http"

	"github.com/didrikolofsson/github-vote-llm/internal/api/middleware"
	"github.com/didrikolofsson/github-vote-llm/internal/services"
	"github.com/gin-gonic/gin"
)

type GithubHandlers struct {
	s *services.GithubService
}

func NewGithubHandlers(s *services.GithubService) *GithubHandlers {
	return &GithubHandlers{s: s}
}

func (h *GithubHandlers) Authorize(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	authUrl, err := h.s.CreateAuthURL(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, authUrl)
}
