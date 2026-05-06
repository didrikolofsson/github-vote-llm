package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
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
	s             *services.GithubService
	l             *logger.Logger
	webhookSecret string
}

func NewGithubHandlers(s *services.GithubService, l *logger.Logger, webhookSecret string) *GithubHandlers {
	return &GithubHandlers{s: s, l: l, webhookSecret: webhookSecret}
}

// GetAppInstallURL returns the GitHub App installation URL for the given org.
// The URL includes a short-lived signed state token that links the install back to the org.
func (h *GithubHandlers) GetAppInstallURL(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	orgID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization id"})
		return
	}

	installURL, err := h.s.CreateAppInstallURL(c.Request.Context(), orgID, userID)
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
	setupAction := c.Query("setup_action")
	state := c.Query("state")

	if setupAction == "" {
		h.l.Errorw("Missing setup_action", "request_id", request.GetRequestID(c))
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing setup_action"})
		return
	}

	if installationIDStr == "" {
		h.l.Errorw("Missing installation_id", "request_id", request.GetRequestID(c))
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing installation_id"})
		return
	}

	installationID, err := strconv.ParseInt(installationIDStr, 10, 64)
	if err != nil {
		h.l.Errorw("Failed to parse installation_id", "error", err, "installation_id", installationIDStr, "request_id", request.GetRequestID(c))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid installation_id"})
		return
	}

	if setupAction == "update" {
		orgID, err := h.s.HandleAppUpdateCallback(c.Request.Context(), installationID)
		if err != nil {
			h.l.Errorw("Failed to handle app update callback", "error", err, "installation_id", installationID, "request_id", request.GetRequestID(c))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}
		h.l.Infow("GitHub App updated", "installation_id", installationID, "org_id", orgID)
		c.Redirect(http.StatusFound, fmt.Sprintf("%s/setup/popup-complete?kind=app_update&ok=1&org_id=%d", h.s.FrontendURL(), orgID))
		return
	}

	if setupAction == "install" {
		orgID, err := h.s.HandleAppInstallCallback(c.Request.Context(), installationID, state)
		if err != nil {
			h.l.Errorw("Failed to handle app install callback", "error", err, "installation_id", installationID, "request_id", request.GetRequestID(c))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		h.l.Infow("GitHub App installed", "installation_id", installationID, "org_id", orgID)
		c.Redirect(http.StatusFound, fmt.Sprintf("%s/setup/popup-complete?kind=app_install&ok=1&org_id=%d", h.s.FrontendURL(), orgID))
		return
	}

	h.l.Errorw("Invalid setup action", "setup_action", setupAction, "request_id", request.GetRequestID(c))
	c.JSON(http.StatusBadRequest, gin.H{"error": "invalid setup action"})
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

	c.JSON(http.StatusOK, dtos.AppInstallationStatus{
		Installed:           status.Installed,
		SuspendedAt:         status.SuspendedAt,
		InstalledByUserName: status.InstalledByUserName,
		TargetLogin:         status.TargetLogin,
		AccountType:         status.AccountType,
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
	if err := verifyWebhookSignature(body, sig, h.webhookSecret); err != nil {
		h.l.Errorw("Failed to verify webhook signature", "error", err, "signature", sig, "request_id", request.GetRequestID(c))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
		return
	}

	eventHeader := c.GetHeader("X-GitHub-Event")
	event, err := validateWebhookEvent(eventHeader)
	if err != nil {
		h.l.Errorw("Failed to validate webhook event", "error", err, "event_header", eventHeader, "request_id", request.GetRequestID(c))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid webhook event"})
		return
	}

	if event == WebhookEventInstallation {
		var payload services.InstallationWebhookPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			h.l.Errorw("Failed to unmarshal installation webhook payload", "error", err, "request_id", request.GetRequestID(c))
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
			return
		}
		if err := h.s.HandleInstallationWebhook(c.Request.Context(), payload); err != nil {
			h.l.Errorw("Failed to handle installation webhook", "error", err, "action", payload.Action, "installation_id", payload.Installation.ID)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}
		c.Status(http.StatusOK)
		return
	}

	h.l.Errorw("Invalid webhook event", "event", event, "request_id", request.GetRequestID(c))
	c.JSON(http.StatusBadRequest, gin.H{"error": "invalid webhook event"})
}

func verifyWebhookSignature(body []byte, signature, secret string) error {
	if len(signature) < 7 || signature[:7] != "sha256=" {
		return errors.New("invalid signature")
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(signature), []byte(expected)) {
		return errors.New("invalid signature")
	}
	return nil
}

type WebhookEvent string

const (
	WebhookEventInstallation WebhookEvent = "installation"
	WebhookEventSuspend      WebhookEvent = "suspend"
	WebhookEventUnsuspend    WebhookEvent = "unsuspend"
	WebhookEventDeleted      WebhookEvent = "deleted"
)

func validateWebhookEvent(event string) (WebhookEvent, error) {
	switch event {
	case "installation":
		return WebhookEventInstallation, nil
	case "suspend":
		return WebhookEventSuspend, nil
	case "unsuspend":
		return WebhookEventUnsuspend, nil
	case "deleted":
		return WebhookEventDeleted, nil
	default:
		return "", fmt.Errorf("invalid webhook event: %s", event)
	}
}
