package handlers

import (
	"net/http"
	"time"

	"github.com/didrikolofsson/github-vote-llm/internal/api/middleware"
	"github.com/didrikolofsson/github-vote-llm/internal/api/services"
	"github.com/didrikolofsson/github-vote-llm/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type GitHubOAuthHandlers interface {
	Authorize(c *gin.Context)
	Callback(c *gin.Context)
	Status(c *gin.Context)
}

type GitHubOAuthHandlersImpl struct {
	oauthService services.GitHubOAuthService
	env          *config.Environment
}

func NewGitHubOAuthHandlers(oauthService services.GitHubOAuthService, env *config.Environment) GitHubOAuthHandlers {
	return &GitHubOAuthHandlersImpl{oauthService: oauthService, env: env}
}

// Authorize returns the GitHub OAuth URL. Requires auth. Frontend redirects user to this URL.
func (h *GitHubOAuthHandlersImpl) Authorize(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	// Build redirect_uri (this server's callback URL)
	scheme := "https"
	if c.GetHeader("X-Forwarded-Proto") == "http" || c.Request.URL.Scheme == "http" {
		scheme = "http"
	}
	redirectURI := scheme + "://" + c.Request.Host + "/v1/auth/github/callback"

	// State: JWT with user_id so we know who to attach the token to
	state, err := h.createStateJWT(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create state"})
		return
	}

	authorizeURL := h.oauthService.BuildAuthorizeURL(redirectURI, state)
	c.JSON(http.StatusOK, gin.H{"authorize_url": authorizeURL})
}

// Callback handles the GitHub OAuth callback. Exchanges code for token, stores, redirects to frontend.
func (h *GitHubOAuthHandlersImpl) Callback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")
	if code == "" || state == "" {
		h.redirectError(c, "missing_code_or_state")
		return
	}

	userID, err := h.verifyStateJWT(state)
	if err != nil {
		h.redirectError(c, "invalid_state")
		return
	}

	scheme := "https"
	if c.GetHeader("X-Forwarded-Proto") == "http" {
		scheme = "http"
	}
	redirectURI := scheme + "://" + c.Request.Host + "/v1/auth/github/callback"

	tokens, err := h.oauthService.ExchangeCode(c.Request.Context(), code, redirectURI)
	if err != nil {
		h.redirectError(c, "token_exchange_failed")
		return
	}

	if err := h.oauthService.StoreConnection(c.Request.Context(), userID, tokens, h.env.TOKEN_ENCRYPTION_KEY); err != nil {
		h.redirectError(c, "storage_failed")
		return
	}

	// Success: redirect to frontend
	c.Redirect(http.StatusFound, h.env.FRONTEND_URL+"/?github_connected=1")
}

func (h *GitHubOAuthHandlersImpl) createStateJWT(userID int64) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(10 * time.Minute).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(h.env.JWT_SECRET))
}

func (h *GitHubOAuthHandlersImpl) verifyStateJWT(state string) (int64, error) {
	token, err := jwt.Parse(state, func(t *jwt.Token) (interface{}, error) {
		return []byte(h.env.JWT_SECRET), nil
	})
	if err != nil || !token.Valid {
		return 0, err
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, jwt.ErrTokenInvalidClaims
	}
	userID, ok := claims["user_id"].(float64)
	if !ok {
		return 0, jwt.ErrTokenInvalidClaims
	}
	return int64(userID), nil
}

func (h *GitHubOAuthHandlersImpl) redirectError(c *gin.Context, reason string) {
	c.Redirect(http.StatusFound, h.env.FRONTEND_URL+"/?github_error="+reason)
}

// Status returns whether the user has connected their GitHub account. Requires auth.
func (h *GitHubOAuthHandlersImpl) Status(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	status, err := h.oauthService.GetConnectionStatus(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"connected": false})
		return
	}
	login := ""
	if status.Login != nil {
		login = *status.Login
	}
	c.JSON(http.StatusOK, gin.H{
		"connected": true,
		"login":    login,
	})
}
