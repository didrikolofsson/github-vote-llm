package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/didrikolofsson/github-vote-llm/internal/api/request"
	"github.com/didrikolofsson/github-vote-llm/internal/api/services"
	"github.com/didrikolofsson/github-vote-llm/internal/logger"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/gin-gonic/gin"
)

type UserHandlers interface {
	Signup(c *gin.Context)
	Login(c *gin.Context)
	Logout(c *gin.Context)
	DeleteUser(c *gin.Context)
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

func (h *UserHandlersImpl) Signup(c *gin.Context) {
	// Check request body
	var params store.CreateUserParams
	if err := c.ShouldBindJSON(&params); err != nil {
		h.l.Errorw("Invalid request body", "error", err, "request_id", request.GetRequestID(c))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	h.l.Infow("Signing up user", "email", params.Email, "request_id", request.GetRequestID(c))

	user, err := h.s.Signup(c.Request.Context(), &params)
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

func (h *UserHandlersImpl) Login(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "User logged in"})
}

func (h *UserHandlersImpl) Logout(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "User logged out"})
}

func (h *UserHandlersImpl) DeleteUser(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}
	if err := h.s.DeleteUser(c.Request.Context(), userID); err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted user " + strconv.FormatInt(userID, 10)})
}
