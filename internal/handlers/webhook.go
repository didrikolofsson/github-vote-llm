package handlers

import (
	"context"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/didrikolofsson/github-vote-llm/internal/agent"
	"github.com/didrikolofsson/github-vote-llm/internal/config"
	ghclient "github.com/didrikolofsson/github-vote-llm/internal/github"
	"github.com/didrikolofsson/github-vote-llm/internal/logger"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/gin-gonic/gin"
	gh "github.com/google/go-github/v68/github"
)

type WebhookHandler struct {
	factory      *ghclient.ClientFactory
	log          *logger.Logger
	workspaceDir string
	store        store.Store
}

func NewWebhookHandler(factory *ghclient.ClientFactory, log *logger.Logger, workspaceDir string, st store.Store) *WebhookHandler {
	return &WebhookHandler{
		factory:      factory,
		log:          log.Named("webhook"),
		workspaceDir: workspaceDir,
		store:        st,
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

func (h *WebhookHandler) handleGithubIssueLabeledEvent(c *gin.Context, e *gh.IssuesEvent) {
	issue := e.GetIssue()
	issueNum := issue.GetNumber()
	labelName := e.GetLabel().GetName()
	repo := e.GetRepo()
	repoLabels := issue.Labels
	owner := repo.GetOwner().GetLogin()
	repoName := repo.GetName()

	h.log.Infow("issue labeled", "issue", issueNum, "label", labelName)

	repoConfig, err := h.store.GetRepoConfig(c.Request.Context(), owner, repoName)
	if err != nil {
		h.log.Errorw("failed to get repo config", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get repo config"})
		return
	}

	if repoConfig == nil {
		h.log.Warnw("no repo config found, using defaults", "owner", owner, "repo", repoName)
		repoConfig = &store.RepoConfigModel{
			Owner:               owner,
			Repo:                repoName,
			LabelApproved:       config.LabelApproved,
			LabelFeatureRequest: config.LabelFeatureRequest,
			LabelInProgress:     config.LabelInProgress,
			LabelDone:           config.LabelDone,
			LabelFailed:         config.LabelFailed,
			VoteThreshold:       config.AgentMaxTurns,
			TimeoutMinutes:      config.AgentTimeoutMinutes,
			MaxBudgetUsd:        config.AgentMaxBudgetUSD,
			AnthropicAPIKey:     os.Getenv("ANTHROPIC_API_KEY"),
			CreatedAt:           time.Now(),
			UpdatedAt:           time.Now(),
		}
	}

	if labelName != repoConfig.LabelApproved {
		h.log.Infow("incoming label is not the approved label, skipping", "issue", issueNum, "label", labelName)
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
		return
	}

	// Does the issue have the feature-request label?
	// Does the issue already have the in-progress label?
	hasFeatureRequest := false
	hasInProgress := false

	for _, l := range repoLabels {
		if l.GetName() == repoConfig.LabelFeatureRequest {
			hasFeatureRequest = true
			break
		}
		if l.GetName() == repoConfig.LabelInProgress {
			hasInProgress = true
			break
		}
	}

	if !hasFeatureRequest {
		h.log.Infow("issue lacks feature-request label, skipping", "issue", issueNum)
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
		return
	}

	if hasInProgress {
		h.log.Infow("issue already in-progress, skipping", "issue", issueNum)
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
		return
	}

	// Here we have an approved label added under the right conditions.
	// Now we check for an existing execution record for this issue.
	// If an execution record exists, we check if it failed.
	// If it failed, we reset and launch the agent again.

	execution, err := h.store.GetExecutionByOwnerRepoIssueNumber(c.Request.Context(), owner, repoName, issueNum)
	if err != nil {
		h.log.Errorw("failed to get execution record", "issue", issueNum, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	if execution == nil {
		h.log.Infow("no execution record found, creating new one", "issue", issueNum)
		execution, err = h.store.CreateExecution(c.Request.Context(), owner, repoName, issueNum)
		if err != nil {
			h.log.Errorw("failed to create execution record", "issue", issueNum, "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		}
	}

	if execution.Status == "failed" {
		h.log.Infow("execution failed, resetting", "issue", issueNum)
		execution, err = h.store.ResetFailedExecution(c.Request.Context(), owner, repoName, issueNum)
		if err != nil {
			h.log.Errorw("failed to reset execution record", "issue", issueNum, "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		}
	}

	if execution.Status == "success" {
		h.log.Infow("execution already successful, skipping", "issue", issueNum)
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
		return
	}

	// Here we have an execution record that is pending
	// Launch agent for implementation in this case
	var installationID int64
	if e.GetInstallation() != nil {
		installationID = *e.GetInstallation().ID
	}
	if installationID == 0 {
		h.log.Errorw("no installation ID in webhook event", "issue", issueNum)
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing installation ID"})
		return
	}

	client, err := ghclient.NewClient(installationID, h.factory)
	if err != nil {
		h.log.Errorw("failed to create installation client", "installationID", installationID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create GitHub client"})
		return
	}

	h.log.Infow("issue approved for development, starting agent", "issue", issueNum, "repo", owner+"/"+repoName)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				h.log.Errorw("panic in agent runner", "repo", owner+"/"+repoName, "issue", issueNum, "panic", r)
			}
		}()
		runner := agent.NewRunner(client, h.log, h.workspaceDir, h.store)
		runner.Run(context.Background(), owner, repoName, issue, execution.ID, *repoConfig)
	}()

	c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "agent started"})
}

func (h *WebhookHandler) handleIssueEvent(c *gin.Context, e *gh.IssuesEvent) {
	switch e.GetAction() {
	case "labeled":
		h.handleGithubIssueLabeledEvent(c, e)
	default:
		h.log.Infow("unhandled issues event action", "action", e.GetAction())
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
		return
	}
}

