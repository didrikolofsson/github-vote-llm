package handlers

import (
	"io"
	"net/http"

	"github.com/didrikolofsson/github-vote-llm/internal/github"
	"github.com/didrikolofsson/github-vote-llm/internal/logger"
	"github.com/gin-gonic/gin"
	gh "github.com/google/go-github/v68/github"
)

type WebhookHandler struct {
	log    *logger.Logger
	client github.ClientAPI
}

func NewWebhookHandler() *WebhookHandler {
	return &WebhookHandler{
		log: logger.New().Named("webhook"),
		// client: github.NewClient(),
	}
}

func (h *WebhookHandler) HandleGithubWebhook(c *gin.Context) {
	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.log.Errorw("failed to read body", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}

	event, err := gh.ParseWebHook(gh.WebHookType(c.Request), payload)
	if err != nil {
		h.log.Errorw("failed to parse webhook", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse webhook"})
		return
	}

	switch e := event.(type) {
	case *gh.IssuesEvent:
		h.handleIssueEvent(c, e)
		return
	default:
		h.log.Infow("unhandled event type", "type", gh.WebHookType(c.Request))
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
		return
	}
}

type IssueLabel string

const (
	LabelApprovedForDevelopment IssueLabel = "approved-for-development"
)

type IssueEventAction string

const (
	IssueEventActionLabeled IssueEventAction = "labeled"
)

func (h *WebhookHandler) handleIssueEvent(c *gin.Context, e *gh.IssuesEvent) {
	action := IssueEventAction(e.GetAction())

	if action != IssueEventActionLabeled {
		h.log.Infow("unhandled issues event action", "action", action)
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
		return
	}

	label := e.GetLabel()
	labelName := IssueLabel(label.GetName())

	if labelName == LabelApprovedForDevelopment {
		h.log.Infow("approved for development", "issue", e.GetIssue().GetNumber())
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
		return
	}

}
