package handlers

import (
	"fmt"
	"net/http"

	"github.com/didrikolofsson/github-vote-llm/internal/api/middleware"
	"github.com/didrikolofsson/github-vote-llm/internal/api/request"
	"github.com/didrikolofsson/github-vote-llm/internal/logger"
	"github.com/didrikolofsson/github-vote-llm/internal/services"
	"github.com/gin-gonic/gin"
)

type GithubHandlers struct {
	s *services.GithubService
	l *logger.Logger
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

	c.JSON(http.StatusOK, gin.H{"authorize_url": authUrl})
}

func (h *GithubHandlers) Install(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	installURL, err := h.s.CreateInstallURL(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"install_url": installURL})
}

func (h *GithubHandlers) AppCallback(c *gin.Context) {
	installationID := c.Query("installation_id")
	state := c.Query("state")

	if installationID == "" || state == "" {
		c.Redirect(http.StatusFound, h.s.FrontendURL()+"?github_error=missing_params")
		return
	}

	var id int64
	if _, err := fmt.Sscanf(installationID, "%d", &id); err != nil {
		c.Redirect(http.StatusFound, h.s.FrontendURL()+"?github_error=invalid_installation_id")
		return
	}

	if err := h.s.CompleteInstall(c.Request.Context(), id, state); err != nil {
		h.l.Errorw("Failed to complete GitHub App install", "error", err, "installation_id", id, "request_id", request.GetRequestID(c))
		c.Redirect(http.StatusFound, h.s.FrontendURL()+"?github_error=install_failed")
		return
	}

	c.Redirect(http.StatusFound, h.s.FrontendURL()+"?github_installed=1")
}

func (h *GithubHandlers) Callback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")

	verified, err := h.s.VerifyAuthStateToken(c.Request.Context(), state)
	if err != nil {
		h.l.Errorw("Failed to verify auth state token", "error", err, "state", state, "request_id", request.GetRequestID(c))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	userID := verified.UserID
	token, err := h.s.ExchangeCode(c.Request.Context(), code, state)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	if err := h.s.UpsertGithubAccountTokenByUserID(c.Request.Context(), userID, token); err != nil {
		h.l.Errorw("Failed to store user tokens", "error", err, "user_id", userID, "request_id", request.GetRequestID(c))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.Redirect(http.StatusFound, h.s.FrontendURL()+"?github_connected=1")
}
