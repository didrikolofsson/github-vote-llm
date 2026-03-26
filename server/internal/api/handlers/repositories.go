package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/didrikolofsson/github-vote-llm/internal/api/middleware"
	"github.com/didrikolofsson/github-vote-llm/internal/api/request"
	"github.com/didrikolofsson/github-vote-llm/internal/api/services"
	"github.com/didrikolofsson/github-vote-llm/internal/logger"
	"github.com/gin-gonic/gin"
)

type RepositoryHandlers interface {
	List(c *gin.Context)
	Add(c *gin.Context)
	Remove(c *gin.Context)
}

type RepositoryHandlersImpl struct {
	s services.RepositoriesService
	l *logger.Logger
}

func NewRepositoryHandlers(s services.RepositoriesService, l *logger.Logger) RepositoryHandlers {
	return &RepositoryHandlersImpl{s: s, l: l}
}

func (h *RepositoryHandlersImpl) List(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	orgID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization ID"})
		return
	}

	repos, err := h.s.ListForOrganization(c.Request.Context(), orgID, userID)
	if err != nil {
		h.l.Errorw("Failed to list repositories", "error", err, "organization_id", orgID, "request_id", request.GetRequestID(c))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"repositories": repos})
}

type addRepositoryRequest struct {
	Owner string `json:"owner" binding:"required"`
	Repo  string `json:"repo" binding:"required"`
}

func (h *RepositoryHandlersImpl) Add(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	orgID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization ID"})
		return
	}

	var req addRepositoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	repo, err := h.s.AddRepository(c.Request.Context(), orgID, userID, req.Owner, req.Repo)
	if errors.Is(err, services.ErrNotOrgMember) {
		c.JSON(http.StatusForbidden, gin.H{"error": "not a member of this organization"})
		return
	}
	if errors.Is(err, services.ErrRepositoryAlreadyAdded) {
		c.JSON(http.StatusConflict, gin.H{"error": "repository already added"})
		return
	}
	if err != nil {
		h.l.Errorw("Failed to add repository", "error", err, "organization_id", orgID, "request_id", request.GetRequestID(c))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusCreated, repo)
}

func (h *RepositoryHandlersImpl) Remove(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	repoID, err := strconv.ParseInt(c.Param("repoId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository ID"})
		return
	}

	err = h.s.RemoveRepository(c.Request.Context(), repoID, userID)
	if errors.Is(err, services.ErrNotOrgMember) {
		c.JSON(http.StatusForbidden, gin.H{"error": "not a member of this organization"})
		return
	}
	if errors.Is(err, services.ErrRepositoryNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
		return
	}
	if err != nil {
		h.l.Errorw("Failed to remove repository", "error", err, "repo_id", repoID, "request_id", request.GetRequestID(c))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.Status(http.StatusNoContent)
}
