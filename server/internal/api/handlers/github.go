package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/didrikolofsson/github-vote-llm/internal/api/middleware"
	"github.com/didrikolofsson/github-vote-llm/internal/api/request"
	"github.com/didrikolofsson/github-vote-llm/internal/dtos"
	"github.com/didrikolofsson/github-vote-llm/internal/logger"
	"github.com/didrikolofsson/github-vote-llm/internal/services"
	"github.com/gin-gonic/gin"
)

type GithubHandlers struct {
	s           *services.GithubService
	l           *logger.Logger
	webhookSecret string
}

func NewGithubHandlers(s *services.GithubService, l *logger.Logger, webhookSecret string) *GithubHandlers {
	return &GithubHandlers{s: s, l: l, webhookSecret: webhookSecret}
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

// GetAppInstallURL returns the GitHub App installation URL for the given org.
// The URL includes a short-lived signed state token that links the install back to the org.
func (h *GithubHandlers) GetAppInstallURL(c *gin.Context) {
	orgID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization id"})
		return
	}

	installURL, err := h.s.CreateAppInstallURL(c.Request.Context(), orgID)
	if err != nil {
		h.l.Errorw("Failed to create app install URL", "error", err, "org_id", orgID, "request_id", request.GetRequestID(c))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, dtos.AppInstallURLResponse{InstallURL: installURL})
}

// AppInstallCallback is the public endpoint GitHub redirects to after the user installs the App.
// It validates the state JWT, fetches installation details from GitHub, and stores the record.
func (h *GithubHandlers) AppInstallCallback(c *gin.Context) {
	installationIDStr := c.Query("installation_id")
	state := c.Query("state")

	if installationIDStr == "" || state == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing installation_id or state"})
		return
	}

	installationID, err := strconv.ParseInt(installationIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid installation_id"})
		return
	}

	orgID, err := h.s.HandleAppInstallCallback(c.Request.Context(), installationID, state)
	if err != nil {
		h.l.Errorw("Failed to handle app install callback", "error", err, "installation_id", installationID, "request_id", request.GetRequestID(c))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	h.l.Infow("GitHub App installed", "installation_id", installationID, "org_id", orgID)
	c.Redirect(http.StatusFound, fmt.Sprintf("%s?app_installed=1", h.s.FrontendURL()))
}

// GetAppInstallationStatus returns the GitHub App installation status for the given org.
// It performs a live verification against GitHub's API and self-heals stale records.
func (h *GithubHandlers) GetAppInstallationStatus(c *gin.Context) {
	orgID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization id"})
		return
	}

	status, err := h.s.GetInstallationStatus(c.Request.Context(), orgID)
	if err != nil {
		h.l.Errorw("Failed to get installation status", "error", err, "org_id", orgID, "request_id", request.GetRequestID(c))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, dtos.AppInstallationStatusResponse{
		Installed:   status.Installed,
		TargetLogin: status.TargetLogin,
		SuspendedAt: status.SuspendedAt,
	})
}

// HandleWebhook receives GitHub App webhook events and syncs installation state.
func (h *GithubHandlers) HandleWebhook(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}

	sig := c.GetHeader("X-Hub-Signature-256")
	if !verifyWebhookSignature(body, sig, h.webhookSecret) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
		return
	}

	event := c.GetHeader("X-GitHub-Event")
	if event != "installation" {
		c.Status(http.StatusNoContent)
		return
	}

	var payload services.InstallationWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	if err := h.s.HandleInstallationWebhook(c.Request.Context(), payload); err != nil {
		h.l.Errorw("Failed to handle installation webhook", "error", err, "action", payload.Action, "installation_id", payload.Installation.ID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.Status(http.StatusNoContent)
}

func verifyWebhookSignature(body []byte, signature, secret string) bool {
	if len(signature) < 7 || signature[:7] != "sha256=" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(signature), []byte(expected))
}
