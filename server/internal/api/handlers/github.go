package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/didrikolofsson/github-vote-llm/internal/api/middleware"
	"github.com/didrikolofsson/github-vote-llm/internal/githubapp"
	"github.com/didrikolofsson/github-vote-llm/internal/services"
	"github.com/gin-gonic/gin"
)

type GithubHandlers struct {
	s            *services.GithubService
	frontendURL  string
}

func NewGithubHandlers(s *services.GithubService, frontendURL string) *GithubHandlers {
	return &GithubHandlers{s: s, frontendURL: frontendURL}
}

// Install returns the github.com URL where the user will install the GitHub App.
// The URL includes a single-use state token bound to the user's session.
func (h *GithubHandlers) Install(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	installURL, err := h.s.CreateInstallURL(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to build install url"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"install_url": installURL})
}

// Callback is the GitHub App "Setup URL". GitHub sends the user here as a top-level
// browser redirect after they pick which repos to grant access to. The state nonce
// (DB-backed, bound to the user who initiated the install) authenticates the request —
// we can't require a Bearer token here since GitHub's redirect is cross-site.
func (h *GithubHandlers) Callback(c *gin.Context) {
	installationIDStr := c.Query("installation_id")
	if installationIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "installation_id is required"})
		return
	}
	installationID, err := strconv.ParseInt(installationIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "installation_id must be a number"})
		return
	}

	state := c.Query("state")
	setupAction := c.Query("setup_action") // "install" | "update"

	if err := h.s.CompleteInstall(c.Request.Context(), installationID, state, setupAction); err != nil {
		if errors.Is(err, githubapp.ErrInvalidState) || errors.Is(err, githubapp.ErrStateUserMismatch) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid install state"})
			return
		}
		if errors.Is(err, services.ErrUserHasNoOrg) {
			c.JSON(http.StatusPreconditionFailed, gin.H{"error": "user has no organization"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to complete install"})
		return
	}

	loc := strings.TrimSuffix(h.frontendURL, "/") + "/settings?github_installed=1"
	c.Redirect(http.StatusTemporaryRedirect, loc)
}

// Status reports whether the user's organization has an active GitHub App installation.
func (h *GithubHandlers) Status(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	status, err := h.s.GetInstallationStatus(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, services.ErrUserHasNoOrg) {
			c.JSON(http.StatusOK, gin.H{"installed": false})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check github status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"installed":            status.Installed,
		"login":                status.Login,
		"account_type":         status.AccountType,
		"repository_selection": status.RepositorySelection,
		"suspended":            status.Suspended,
	})
}

// Disconnect removes the installation row from our DB. This does NOT uninstall the
// app on GitHub — the user must do that from github.com. The webhook will reconcile.
func (h *GithubHandlers) Disconnect(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if err := h.s.DeleteInstallation(c.Request.Context(), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to disconnect github installation"})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *GithubHandlers) ListRepositories(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	page := 1
	if p := c.Query("page"); p != "" {
		if n, err := strconv.Atoi(p); err == nil && n > 0 {
			page = n
		}
	}

	repos, hasMore, err := h.s.ListInstallationRepositories(c.Request.Context(), userID, page)
	if err != nil {
		if errors.Is(err, services.ErrGitHubNotInstalled) {
			c.JSON(http.StatusPreconditionFailed, gin.H{"installed": false})
			return
		}
		if errors.Is(err, services.ErrInstallationSuspended) {
			c.JSON(http.StatusPreconditionFailed, gin.H{"installed": true, "suspended": true})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"repositories": repos, "has_more": hasMore})
}
