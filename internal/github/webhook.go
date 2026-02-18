package github

import (
	"io"
	"net/http"

	"github.com/didrikolofsson/github-vote-llm/internal/logger"
	"github.com/gin-gonic/gin"
	gh "github.com/google/go-github/v68/github"
)

type webhookService struct {
	log *logger.Logger
}

type WebhookService interface {
	HandleGithubWebhook(c *gin.Context)
}

func NewWebhookService() WebhookService {
	log := logger.New().Named("webhook")
	return &webhookService{
		log: log,
	}
}

func (h *webhookService) HandleGithubWebhook(c *gin.Context) {
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
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *webhookService) handleIssueEvent(c *gin.Context, e *gh.IssuesEvent) {
	h.log.Infow("issue event", "action", e.GetAction(), "issue", e.GetIssue().GetNumber())
}

// // IssueApprovedHandler is called when an issue receives the approved label.
// type IssueApprovedHandler func(owner, repo string, issue *gh.Issue, installationID int64)

// // VoteCheckHandler is called when a vote check should be performed.
// type VoteCheckHandler func(owner, repo string, issue *gh.Issue, installationID int64)

// // WebhookHandler handles incoming GitHub webhook events.
// type WebhookHandler struct {
// 	secret      []byte
// 	cfg         *config.Config
// 	onApproved  IssueApprovedHandler
// 	onVoteCheck VoteCheckHandler
// 	log         *logger.Logger
// }

// // NewWebhookHandler creates a new webhook handler.
// func NewWebhookHandler(cfg *config.Config, onApproved IssueApprovedHandler, onVoteCheck VoteCheckHandler, log *logger.Logger) *WebhookHandler {
// 	return &WebhookHandler{
// 		secret:      []byte(cfg.GitHub.WebhookSecret),
// 		cfg:         cfg,
// 		onApproved:  onApproved,
// 		onVoteCheck: onVoteCheck,
// 		log:         log.Named("webhook"),
// 	}
// }

// // ServeHTTP handles incoming webhook requests.
// func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
// 	payload, err := gh.ValidatePayload(r, h.secret)
// 	if err != nil {
// 		h.log.Warnw("invalid signature", "error", err)
// 		http.Error(w, "invalid signature", http.StatusUnauthorized)
// 		return
// 	}

// 	event, err := gh.ParseWebHook(gh.WebHookType(r), payload)
// 	if err != nil {
// 		h.log.Errorw("failed to parse event", "error", err)
// 		http.Error(w, "bad request", http.StatusBadRequest)
// 		return
// 	}

// 	switch e := event.(type) {
// 	case *gh.IssuesEvent:
// 		h.handleIssueEvent(e)
// 	case *gh.IssueCommentEvent:
// 		h.handleIssueCommentEvent(e)
// 	default:
// 		// Ignore unhandled event types
// 	}

// 	w.WriteHeader(http.StatusOK)
// }

// func (h *WebhookHandler) handleIssueEvent(e *gh.IssuesEvent) {
// 	action := e.GetAction()
// 	issue := e.GetIssue()
// 	repo := e.GetRepo()
// 	owner := repo.GetOwner().GetLogin()
// 	repoName := repo.GetName()
// 	installationID := int64(0)
// 	if e.GetInstallation() != nil {
// 		installationID = e.GetInstallation().GetID()
// 	}

// 	h.log.Infow("issues event", "action", action, "issue", issue.GetNumber(), "repo", owner+"/"+repoName)

// 	repoConfig := h.cfg.FindRepo(owner, repoName)
// 	if repoConfig == nil {
// 		h.log.Infow("repo not configured, ignoring", "repo", owner+"/"+repoName)
// 		return
// 	}

// 	switch action {
// 	case "labeled":
// 		h.handleLabeled(owner, repoName, issue, e.GetLabel(), repoConfig, installationID)
// 	}
// }

// func (h *WebhookHandler) handleLabeled(owner, repo string, issue *gh.Issue, label *gh.Label, repoConfig *config.RepoConfig, installationID int64) {
// 	labelName := label.GetName()

// 	// When feature-request label is added, check if it already has enough votes
// 	if labelName == repoConfig.Labels.FeatureRequest {
// 		h.log.Infow("feature-request label added, checking votes", "issue", issue.GetNumber())
// 		h.fireVoteCheck(owner, repo, issue, installationID)
// 		return
// 	}

// 	if labelName != repoConfig.Labels.Approved {
// 		return
// 	}

// 	// Guard: only process if issue has the feature-request label
// 	hasFeatureRequest := false
// 	for _, l := range issue.Labels {
// 		if l.GetName() == repoConfig.Labels.FeatureRequest {
// 			hasFeatureRequest = true
// 			break
// 		}
// 	}
// 	if !hasFeatureRequest {
// 		h.log.Infow("approved label added but issue lacks feature-request label, skipping", "issue", issue.GetNumber())
// 		return
// 	}

// 	// Skip if already in-progress (prevents duplicate runs from duplicate webhooks)
// 	for _, l := range issue.Labels {
// 		if l.GetName() == repoConfig.Labels.InProgress {
// 			h.log.Infow("issue already in-progress, skipping", "issue", issue.GetNumber())
// 			return
// 		}
// 	}

// 	h.log.Infow("issue approved for development", "issue", issue.GetNumber(), "repo", owner+"/"+repo)
// 	if h.onApproved != nil {
// 		go func() {
// 			defer func() {
// 				if r := recover(); r != nil {
// 					h.log.Errorw("panic in onApproved", "repo", owner+"/"+repo, "issue", issue.GetNumber(), "panic", r)
// 				}
// 			}()
// 			h.onApproved(owner, repo, issue, installationID)
// 		}()
// 	}
// }

// func (h *WebhookHandler) handleIssueCommentEvent(e *gh.IssueCommentEvent) {
// 	if e.GetAction() != "created" {
// 		return
// 	}

// 	issue := e.GetIssue()
// 	repo := e.GetRepo()
// 	owner := repo.GetOwner().GetLogin()
// 	repoName := repo.GetName()
// 	installationID := int64(0)
// 	if e.GetInstallation() != nil {
// 		installationID = e.GetInstallation().GetID()
// 	}

// 	repoConfig := h.cfg.FindRepo(owner, repoName)
// 	if repoConfig == nil {
// 		return
// 	}

// 	// Only check votes on issues with the feature-request label
// 	for _, l := range issue.Labels {
// 		if l.GetName() == repoConfig.Labels.FeatureRequest {
// 			h.log.Infow("comment on feature-request issue, checking votes", "issue", issue.GetNumber(), "repo", owner+"/"+repoName)
// 			h.fireVoteCheck(owner, repoName, issue, installationID)
// 			return
// 		}
// 	}
// }

// func (h *WebhookHandler) fireVoteCheck(owner, repo string, issue *gh.Issue, installationID int64) {
// 	if h.onVoteCheck == nil {
// 		return
// 	}
// 	go func() {
// 		defer func() {
// 			if r := recover(); r != nil {
// 				h.log.Errorw("panic in onVoteCheck", "repo", owner+"/"+repo, "issue", issue.GetNumber(), "panic", r)
// 			}
// 		}()
// 		h.onVoteCheck(owner, repo, issue, installationID)
// 	}()
// }
