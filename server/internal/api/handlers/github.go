package handlers

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/didrikolofsson/github-vote-llm/internal/api/middleware"
	"github.com/didrikolofsson/github-vote-llm/internal/helpers"
	"github.com/didrikolofsson/github-vote-llm/internal/services"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2/github"
)

var (
	GitHubAuthURL = github.Endpoint.AuthURL
)

type GithubHandlers struct {
	s *services.GithubService
}

func NewGithubHandlers(s *services.GithubService) *GithubHandlers {
	return &GithubHandlers{s: s}
}

var (
	REQUEST_ORIGIN_COOKIE_NAME = "gvllm_request_origin"
)

// Authorize lets the client initiate the OAuth2 flow by returning the GitHub authorization URL.
func (h *GithubHandlers) Authorize(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	origin := c.Request.Referer()
	cookie, err := helpers.BuildRequestOriginCookie(
		REQUEST_ORIGIN_COOKIE_NAME, origin, true,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	http.SetCookie(c.Writer, cookie)

	authorizeURL, err := h.s.CreateOAuthState(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"authorize_url": authorizeURL})
}

func (h *GithubHandlers) Callback(c *gin.Context) {
	redirectBase, ok := helpers.ReadRequestOriginCookie(REQUEST_ORIGIN_COOKIE_NAME, c.Request)
	if !ok || redirectBase == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing or invalid request origin"})
		return
	}

	if ghErr := c.Query("error"); ghErr != "" {
		desc := c.Query("error_description")
		q := url.Values{}
		q.Set("github_error", "1")
		if desc != "" {
			q.Set("error_description", desc)
		}
		helpers.DeleteRequestOriginCookie(REQUEST_ORIGIN_COOKIE_NAME, c.Writer)
		loc := strings.TrimSuffix(redirectBase, "/") + "?" + q.Encode()
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

	helpers.DeleteRequestOriginCookie(REQUEST_ORIGIN_COOKIE_NAME, c.Writer)
	successUrl := strings.TrimSuffix(redirectBase, "/") + "?github_connected=1"
	c.Redirect(http.StatusTemporaryRedirect, successUrl)
}

func (h *GithubHandlers) Status(c *gin.Context) {
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

func (h *GithubHandlers) Disconnect(c *gin.Context) {
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

func (h *GithubHandlers) ListReposByAuthenticatedUser(c *gin.Context) {
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
