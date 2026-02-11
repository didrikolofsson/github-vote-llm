package github

import (
	"log"
	"net/http"

	"github.com/didrikolofsson/github-vote-llm/internal/config"
	"github.com/didrikolofsson/github-vote-llm/internal/logger"
	gh "github.com/google/go-github/v68/github"
)

// IssueApprovedHandler is called when an issue receives the approved label.
type IssueApprovedHandler func(owner, repo string, issue *gh.Issue)

// WebhookHandler handles incoming GitHub webhook events.
type WebhookHandler struct {
	secret     []byte
	cfg        *config.Config
	onApproved IssueApprovedHandler
	log        *logger.Logger
}

// NewWebhookHandler creates a new webhook handler.
func NewWebhookHandler(cfg *config.Config, onApproved IssueApprovedHandler, log *logger.Logger) *WebhookHandler {
	return &WebhookHandler{
		secret:     []byte(cfg.GitHub.WebhookSecret),
		cfg:        cfg,
		onApproved: onApproved,
		log:        log.Named("webhook"),
	}
}

// ServeHTTP handles incoming webhook requests.
func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	payload, err := gh.ValidatePayload(r, h.secret)
	if err != nil {
		h.log.Warnw("invalid signature", "error", err)
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	event, err := gh.ParseWebHook(gh.WebHookType(r), payload)
	if err != nil {
		h.log.Errorw("failed to parse event", "error", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	switch e := event.(type) {
	case *gh.IssuesEvent:
		h.handleIssueEvent(e)
	default:
		// Ignore unhandled event types
	}

	w.WriteHeader(http.StatusOK)
}

func (h *WebhookHandler) handleIssueEvent(e *gh.IssuesEvent) {
	action := e.GetAction()
	issue := e.GetIssue()
	repo := e.GetRepo()
	owner := repo.GetOwner().GetLogin()
	repoName := repo.GetName()

	h.log.Infow("issues event", "action", action, "issue", issue.GetNumber(), "repo", owner+"/"+repoName)

	repoConfig := h.cfg.FindRepo(owner, repoName)
	if repoConfig == nil {
		h.log.Infow("repo not configured, ignoring", "repo", owner+"/"+repoName)
		return
	}

	switch action {
	case "labeled":
		h.handleLabeled(owner, repoName, issue, e.GetLabel(), repoConfig)
	}
}

func (h *WebhookHandler) handleLabeled(owner, repo string, issue *gh.Issue, label *gh.Label, repoConfig *config.RepoConfig) {
	if label.GetName() != repoConfig.Labels.Approved {
		return
	}

	h.log.Infow("issue approved for development", "issue", issue.GetNumber(), "repo", owner+"/"+repo)
	// Skip if already in-progress (prevents duplicate runs from duplicate webhooks)
	for _, l := range issue.Labels {
		if l.GetName() == repoConfig.Labels.InProgress {
			h.log.Infow("issue already in-progress, skipping", "issue", issue.GetNumber())
			return
		}
	}

	h.log.Infow("issue approved for development", "issue", issue.GetNumber(), "repo", owner+"/"+repo)
	if h.onApproved != nil {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("webhook: panic in onApproved for %s/%s#%d: %v", owner, repo, issue.GetNumber(), r)
				}
			}()
			h.onApproved(owner, repo, issue)
		}()
	}
}
