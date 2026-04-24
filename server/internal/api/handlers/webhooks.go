package handlers

import (
	"io"
	"net/http"

	"github.com/didrikolofsson/github-vote-llm/internal/githubapp"
	"github.com/didrikolofsson/github-vote-llm/internal/logger"
	"github.com/didrikolofsson/github-vote-llm/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/go-github/v84/github"
)

type WebhooksHandlers struct {
	s      *services.GithubService
	secret string
	log    *logger.Logger
}

func NewWebhooksHandlers(s *services.GithubService, secret string, log *logger.Logger) *WebhooksHandlers {
	return &WebhooksHandlers{s: s, secret: secret, log: log.Named("webhooks")}
}

// Github handles POST /webhooks/github. Verifies X-Hub-Signature-256 and dispatches
// installation + installation_repositories events to the service. Returns 2xx fast.
func (h *WebhooksHandlers) Github(c *gin.Context) {
	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	sig := c.GetHeader(githubapp.SignatureHeader)
	if !githubapp.VerifySignature(payload, sig, h.secret) {
		c.Status(http.StatusUnauthorized)
		return
	}

	eventType := c.GetHeader(githubapp.EventHeader)
	deliveryID := c.GetHeader(githubapp.DeliveryHeader)

	parsed, err := github.ParseWebHook(eventType, payload)
	if err != nil {
		h.log.Warnw("failed to parse webhook", "event", eventType, "delivery", deliveryID, "err", err)
		c.Status(http.StatusBadRequest)
		return
	}

	ctx := c.Request.Context()

	switch event := parsed.(type) {
	case *github.InstallationEvent:
		if err := h.s.HandleInstallationEvent(ctx, event); err != nil {
			h.log.Errorw("installation event handler failed", "delivery", deliveryID, "err", err)
			c.Status(http.StatusInternalServerError)
			return
		}
	case *github.InstallationRepositoriesEvent:
		if err := h.s.HandleInstallationRepositoriesEvent(ctx, event); err != nil {
			h.log.Errorw("installation_repositories event handler failed", "delivery", deliveryID, "err", err)
			c.Status(http.StatusInternalServerError)
			return
		}
	default:
		// Ignore unsupported event types.
	}

	c.Status(http.StatusNoContent)
}
