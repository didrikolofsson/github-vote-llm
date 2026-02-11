package github

import (
	"log"
	"net/http"

	"github.com/didrikolofsson/github-vote-llm/internal/config"
	gh "github.com/google/go-github/v68/github"
)

// IssueApprovedHandler is called when an issue receives the approved label.
type IssueApprovedHandler func(owner, repo string, issue *gh.Issue)

// WebhookHandler handles incoming GitHub webhook events.
type WebhookHandler struct {
	secret          []byte
	cfg             *config.Config
	onApproved      IssueApprovedHandler
}

// NewWebhookHandler creates a new webhook handler.
func NewWebhookHandler(cfg *config.Config, onApproved IssueApprovedHandler) *WebhookHandler {
	return &WebhookHandler{
		secret:     []byte(cfg.GitHub.WebhookSecret),
		cfg:        cfg,
		onApproved: onApproved,
	}
}

// ServeHTTP handles incoming webhook requests.
func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	payload, err := gh.ValidatePayload(r, h.secret)
	if err != nil {
		log.Printf("webhook: invalid signature: %v", err)
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	event, err := gh.ParseWebHook(gh.WebHookType(r), payload)
	if err != nil {
		log.Printf("webhook: failed to parse event: %v", err)
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

	log.Printf("webhook: issues event action=%s issue=#%d repo=%s/%s", action, issue.GetNumber(), owner, repoName)

	repoConfig := h.cfg.FindRepo(owner, repoName)
	if repoConfig == nil {
		log.Printf("webhook: repo %s/%s not configured, ignoring", owner, repoName)
		return
	}

	switch action {
	case "labeled":
		h.handleLabeled(owner, repoName, issue, repoConfig)
	}
}

func (h *WebhookHandler) handleLabeled(owner, repo string, issue *gh.Issue, repoConfig *config.RepoConfig) {
	for _, label := range issue.Labels {
		if label.GetName() == repoConfig.Labels.Approved {
			log.Printf("webhook: issue #%d approved for development in %s/%s", issue.GetNumber(), owner, repo)
			if h.onApproved != nil {
				go h.onApproved(owner, repo, issue)
			}
			return
		}
	}
}
