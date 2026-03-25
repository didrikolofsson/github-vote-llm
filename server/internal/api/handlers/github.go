package handlers

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/didrikolofsson/github-vote-llm/internal/api/middleware"
	"github.com/didrikolofsson/github-vote-llm/internal/api/services"
	"github.com/didrikolofsson/github-vote-llm/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
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
}

type GithubHandlersImpl struct {
	env *config.Environment
	s   services.GithubService
}

func NewGithubHandlers(env *config.Environment, s services.GithubService) GithubHandlers {
	return &GithubHandlersImpl{env: env, s: s}
}

// oauthStateClaims is signed into the GitHub `state` query param so /callback can bind the code to a user.
type oauthStateClaims struct {
	UserID int64 `json:"uid"`
	jwt.RegisteredClaims
}

// Authorize lets the client initiate the OAuth2 flow by returning the GitHub authorization URL.
// Requires JWT (see api router). Response matches client: { "authorize_url": "..." }.
func (h *GithubHandlersImpl) Authorize(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	stateTok := jwt.NewWithClaims(jwt.SigningMethodHS256, oauthStateClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(10 * time.Minute)),
		},
	})
	stateStr, err := stateTok.SignedString([]byte(h.env.JWT_SECRET))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to build oauth state"})
		return
	}

	v := url.Values{}
	v.Set("client_id", h.env.GITHUB_CLIENT_ID)
	v.Set("redirect_uri", h.env.SERVER_URL+"/v1/github/callback")
	v.Set("scope", "repo read:org")
	v.Set("state", stateStr)

	authorizeURL := GitHubAuthURL + "?" + v.Encode()
	c.JSON(http.StatusOK, gin.H{"authorize_url": authorizeURL})
}

func (h *GithubHandlersImpl) Callback(c *gin.Context) {
	if ghErr := c.Query("error"); ghErr != "" {
		desc := c.Query("error_description")
		q := url.Values{}
		q.Set("github_error", "1")
		if desc != "" {
			q.Set("error_description", desc)
		}
		loc := strings.TrimSuffix(h.env.FRONTEND_URL, "/") + "?" + q.Encode()
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

	var claims oauthStateClaims
	tok, err := jwt.ParseWithClaims(state, &claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(h.env.JWT_SECRET), nil
	})
	if err != nil || tok == nil || !tok.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired oauth state"})
		return
	}
	userID := claims.UserID

	if err := h.s.Callback(c.Request.Context(), code, userID, h.env.TOKEN_ENCRYPTION_KEY); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to complete github oauth"})
		return
	}

	success := strings.TrimSuffix(h.env.FRONTEND_URL, "/") + "?github_connected=1"
	c.Redirect(http.StatusTemporaryRedirect, success)
}

func (h *GithubHandlersImpl) Status(c *gin.Context) {
}
