package handlers

import (
	"context"
	"io"
	"net/http"

	"github.com/didrikolofsson/github-vote-llm/internal/agent"
	ghclient "github.com/didrikolofsson/github-vote-llm/internal/github"
	"github.com/didrikolofsson/github-vote-llm/internal/logger"
	"github.com/gin-gonic/gin"
	gh "github.com/google/go-github/v68/github"
)

type WebhookHandler struct {
	factory *ghclient.ClientFactory
	log     *logger.Logger
}

func NewWebhookHandler(factory *ghclient.ClientFactory, log *logger.Logger) *WebhookHandler {
	return &WebhookHandler{
		factory: factory,
		log:     log.Named("webhook"),
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
	default:
		h.log.Infow("unhandled event type", "type", gh.WebHookType(c.Request))
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
}

func (h *WebhookHandler) handleIssueEvent(c *gin.Context, e *gh.IssuesEvent) {
	if e.GetAction() != "labeled" {
		h.log.Infow("unhandled issues event action", "action", e.GetAction())
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
		return
	}

	labelName := e.GetLabel().GetName()
	if labelName != agent.LabelApproved {
		h.log.Infow("label added but not approval label, ignoring", "label", labelName)
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
		return
	}

	issue := e.GetIssue()
	repo := e.GetRepo()
	owner := repo.GetOwner().GetLogin()
	repoName := repo.GetName()
	issueNum := issue.GetNumber()

	// Guard: issue must have feature-request label
	hasFeatureRequest := false
	for _, l := range issue.Labels {
		if l.GetName() == agent.LabelFeatureRequest {
			hasFeatureRequest = true
			break
		}
	}
	if !hasFeatureRequest {
		h.log.Infow("approved label added but issue lacks feature-request label, skipping", "issue", issueNum)
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
		return
	}

	// Guard: issue must not already have in-progress label
	for _, l := range issue.Labels {
		if l.GetName() == agent.LabelInProgress {
			h.log.Infow("issue already in-progress, skipping", "issue", issueNum)
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
			return
		}
	}

	// Get installation ID from the event
	installationID := int64(0)
	if e.GetInstallation() != nil {
		installationID = e.GetInstallation().GetID()
	}
	if installationID == 0 {
		h.log.Errorw("no installation ID in webhook event", "issue", issueNum)
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing installation ID"})
		return
	}

	// Create client for this installation
	client, err := ghclient.NewClient(installationID, h.factory)
	if err != nil {
		h.log.Errorw("failed to create installation client", "installationID", installationID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create GitHub client"})
		return
	}

	h.log.Infow("issue approved for development, starting agent", "issue", issueNum, "repo", owner+"/"+repoName)

	// Run agent in background goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				h.log.Errorw("panic in agent runner", "repo", owner+"/"+repoName, "issue", issueNum, "panic", r)
			}
		}()
		runner := agent.NewRunner(client, h.log)
		runner.Run(context.Background(), owner, repoName, issue)
	}()

	c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "agent started"})
}
