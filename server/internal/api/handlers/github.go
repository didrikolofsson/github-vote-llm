package handlers

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/didrikolofsson/github-vote-llm/internal/api/middleware"
	"github.com/didrikolofsson/github-vote-llm/internal/services"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2/github"
)

var (
	GitHubAuthURL = github.Endpoint.AuthURL
)

type GithubHandlers interface {
	// Auth handlers
	Authorize(c *gin.Context)
	Callback(c *gin.Context)
	Status(c *gin.Context)
	Disconnect(c *gin.Context)
	ListReposByAuthenticatedUser(c *gin.Context)
}

type GithubHandlersImpl struct {
	s services.GithubService
}

func NewGithubHandlers(s services.GithubService) GithubHandlers {
	return &GithubHandlersImpl{s: s}
}

func isRequestSecure(r *http.Request) bool {
	if r == nil {
		return false
	}
	if r.TLS != nil {
		return true
	}
	if proto := r.Header.Get("X-Forwarded-Proto"); strings.EqualFold(proto, "https") {
		return true
	}
	return false
}

func inferFrontendOrigin(r *http.Request) string {
	if r == nil {
		return ""
	}
	if o := strings.TrimSpace(r.Header.Get("Origin")); o != "" {
		return o
	}
	ref := strings.TrimSpace(r.Header.Get("Referer"))
	if ref == "" {
		return ""
	}
	u, err := url.Parse(ref)
	if err != nil || u == nil {
		return ""
	}
	if u.Scheme == "" || u.Host == "" {
		return ""
	}
	u.Path = ""
	u.RawQuery = ""
	u.Fragment = ""
	u.User = nil
	return u.String()
}

// Authorize lets the client initiate the OAuth2 flow by returning the GitHub authorization URL.
func (h *GithubHandlersImpl) Authorize(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	origin := inferFrontendOrigin(c.Request)
	if origin != "" {
		if cookie, err := h.s.BuildGitHubOAuthOriginCookie(origin, isRequestSecure(c.Request)); err == nil && cookie != nil {
			http.SetCookie(c.Writer, cookie)
		}
	}

	authorizeURL, err := h.s.CreateOAuthState(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"authorize_url": authorizeURL})
}

func (h *GithubHandlersImpl) Callback(c *gin.Context) {
	redirectBase, ok := h.s.ReadGitHubOAuthOriginCookie(c.Request)
	if !ok || redirectBase == "" {
		redirectBase = h.s.DefaultFrontendURL()
	}

	// Best-effort cleanup: clear the cookie after we’ve used it.
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "gvllm_gh_oauth_origin",
		Value:    "",
		Path:     "/v1/github/callback",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   isRequestSecure(c.Request),
		SameSite: http.SameSiteLaxMode,
	})

	if ghErr := c.Query("error"); ghErr != "" {
		desc := c.Query("error_description")
		q := url.Values{}
		q.Set("github_error", "1")
		if desc != "" {
			q.Set("error_description", desc)
		}
		loc := redirectBase + "?" + q.Encode()
		c.Redirect(http.StatusTemporaryRedirect, loc)
		return
	}

	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code is required"})
		return
	}

	state := c.Query("state")
	if state == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "state is required"})
		return
	}

	claims, err := h.s.ReadOAuthStateClaims(c.Request.Context(), state)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	userID := claims.UserID
	if err := h.s.ExchangeCodeForAccessToken(c.Request.Context(), code, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to complete github oauth"})
		return
	}

	successUrl := redirectBase + "/settings?github_connected=1"
	c.Redirect(http.StatusTemporaryRedirect, successUrl)
}

func (h *GithubHandlersImpl) Status(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	status, err := h.s.GetGitHubConnectionStatus(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, services.ErrGitHubNotConnected) {
			c.JSON(http.StatusOK, gin.H{"connected": false})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check github status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"connected": true,
		"login":     status.Login,
	})
}

func (h *GithubHandlersImpl) Disconnect(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if err := h.s.DeleteGitHubConnection(c.Request.Context(), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to disconnect github account"})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *GithubHandlersImpl) ListReposByAuthenticatedUser(c *gin.Context) {
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

	repos, hasMore, err := h.s.ListReposByAuthenticatedUser(c.Request.Context(), userID, page)
	if err != nil {
		if errors.Is(err, services.ErrGitHubNotConnected) {
			c.JSON(http.StatusPreconditionFailed, gin.H{"connected": false})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"repositories": repos, "has_more": hasMore})
}
