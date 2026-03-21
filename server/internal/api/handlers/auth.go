package handlers

import (
	"errors"
	"net/http"

	"github.com/didrikolofsson/github-vote-llm/internal/api/services"
	"github.com/gin-gonic/gin"
)

type AuthHandlers interface {
	Authorize(c *gin.Context)
	Token(c *gin.Context)
	Revoke(c *gin.Context)
	Signup(c *gin.Context)
	Login(c *gin.Context)
	Logout(c *gin.Context)
}

type AuthHandlersImpl struct {
	s         services.AuthService
	jwtSecret string
}

func NewAuthHandlers(s services.AuthService, jwtSecret string) AuthHandlers {
	return &AuthHandlersImpl{s: s, jwtSecret: jwtSecret}
}

type authorizeRequest struct {
	Email         string `json:"email" binding:"required,email"`
	Password      string `json:"password" binding:"required"`
	CodeChallenge string `json:"code_challenge" binding:"required"`
	RedirectURI   string `json:"redirect_uri" binding:"required"`
}

type tokenRequest struct {
	GrantType    string `json:"grant_type" binding:"required"`
	Code         string `json:"code"`
	CodeVerifier string `json:"code_verifier"`
	RedirectURI  string `json:"redirect_uri"`
	RefreshToken string `json:"refresh_token"`
}

type revokeRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

func (h *AuthHandlersImpl) Authorize(c *gin.Context) {
	var req authorizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	code, err := h.s.Authorize(c.Request.Context(), req.Email, req.Password, req.CodeChallenge, req.RedirectURI)
	if errors.Is(err, services.ErrInvalidCredentials) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": code, "redirect_uri": req.RedirectURI})
}

func (h *AuthHandlersImpl) Token(c *gin.Context) {
	var req tokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	switch req.GrantType {
	case "authorization_code":
		if req.Code == "" || req.CodeVerifier == "" || req.RedirectURI == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "code, code_verifier, and redirect_uri are required"})
			return
		}
		accessToken, refreshToken, err := h.s.ExchangeCode(c.Request.Context(), req.Code, req.CodeVerifier, req.RedirectURI)
		if errors.Is(err, services.ErrInvalidAuthCode) || errors.Is(err, services.ErrInvalidPKCE) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"access_token":  accessToken,
			"refresh_token": refreshToken,
			"token_type":    "Bearer",
			"expires_in":    900,
		})

	case "refresh_token":
		if req.RefreshToken == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "refresh_token is required"})
			return
		}
		accessToken, err := h.s.Refresh(c.Request.Context(), req.RefreshToken)
		if errors.Is(err, services.ErrInvalidRefreshToken) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired refresh token"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"access_token": accessToken,
			"token_type":   "Bearer",
			"expires_in":   900,
		})

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported grant_type"})
	}
}

func (h *AuthHandlersImpl) Revoke(c *gin.Context) {
	var req revokeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	if err := h.s.Revoke(c.Request.Context(), req.RefreshToken); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.Status(http.StatusOK)
}

func (h *AuthHandlersImpl) Signup(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "User signed up"})
}

func (h *AuthHandlersImpl) Login(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "User logged in"})
}

func (h *AuthHandlersImpl) Logout(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "User logged out"})
}
