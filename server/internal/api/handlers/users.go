package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/didrikolofsson/github-vote-llm/internal/api/middleware"
	"github.com/didrikolofsson/github-vote-llm/internal/api/request"
	"github.com/didrikolofsson/github-vote-llm/internal/services"
	"github.com/didrikolofsson/github-vote-llm/internal/logger"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/gin-gonic/gin"
)

type UserHandlers interface {
	SignupUser(c *gin.Context)
	DeleteUser(c *gin.Context)
	GetMe(c *gin.Context)
	UpdateUsername(c *gin.Context)
}

type UserHandlersImpl struct {
	s services.UserService
	l *logger.Logger
}

func NewUserHandlers(s services.UserService, l *logger.Logger) UserHandlers {
	return &UserHandlersImpl{
		s: s,
		l: l,
	}
}

func (h *UserHandlersImpl) SignupUser(c *gin.Context) {
	// Check request body
	var params store.CreateUserParams
	if err := c.ShouldBindJSON(&params); err != nil {
		h.l.Errorw("Invalid request body", "error", err, "request_id", request.GetRequestID(c))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	h.l.Infow("Signing up user", "email", params.Email, "request_id", request.GetRequestID(c))

	user, err := h.s.SignupUser(c.Request.Context(), &params)
	if errors.Is(err, services.ErrUserExists) {
		h.l.Warnw("User already exists", "email", params.Email, "request_id", request.GetRequestID(c))
		c.JSON(http.StatusBadRequest, gin.H{"error": "User already exists"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, user)
}

func (h *UserHandlersImpl) GetMe(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	user, err := h.s.GetUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, user)
}

type updateUsernameRequest struct {
	Username string `json:"username" binding:"required"`
}

func (h *UserHandlersImpl) UpdateUsername(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var req updateUsernameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username is required"})
		return
	}
	user, err := h.s.UpdateUsername(c.Request.Context(), userID, req.Username)
	if errors.Is(err, services.ErrUsernameTaken) {
		c.JSON(http.StatusConflict, gin.H{"error": "username already taken"})
		return
	}
	if err != nil {
		h.l.Errorw("Failed to update username", "error", err, "request_id", request.GetRequestID(c))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, user)
}

func (h *UserHandlersImpl) DeleteUser(c *gin.Context) {
	requestingUserID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	targetUserID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}
	if err := h.s.DeleteUser(c.Request.Context(), requestingUserID, targetUserID); err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		if errors.Is(err, services.ErrForbiddenUserDelete) {
			c.JSON(http.StatusForbidden, gin.H{"error": "you cannot delete this user"})
			return
		}
		h.l.Errorw("Failed to delete user", "error", err, "request_id", request.GetRequestID(c))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.Status(http.StatusNoContent)
}
