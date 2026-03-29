package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/didrikolofsson/github-vote-llm/internal/api/request"
	"github.com/didrikolofsson/github-vote-llm/internal/api/services"
	"github.com/didrikolofsson/github-vote-llm/internal/logger"
	"github.com/gin-gonic/gin"
)

type PortalHandlers interface {
	GetPortalPage(c *gin.Context)
	ToggleVote(c *gin.Context)
	ListComments(c *gin.Context)
	CreateComment(c *gin.Context)
}

type PortalHandlersImpl struct {
	s services.PortalService
	l *logger.Logger
}

func NewPortalHandlers(s services.PortalService, l *logger.Logger) PortalHandlers {
	return &PortalHandlersImpl{s: s, l: l}
}

// GET /v1/portal/:orgSlug/:repoName?voter_token=<uuid>
func (h *PortalHandlersImpl) GetPortalPage(c *gin.Context) {
	orgSlug := c.Param("orgSlug")
	repoName := c.Param("repoName")
	voterToken := c.Query("voter_token")

	page, err := h.s.GetPortalPage(c.Request.Context(), orgSlug, repoName, voterToken)
	if errors.Is(err, services.ErrPortalNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "portal not found or not public"})
		return
	}
	if err != nil {
		h.l.Errorw("Failed to get portal page", "error", err, "org_slug", orgSlug, "repo_name", repoName, "request_id", request.GetRequestID(c))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, page)
}

type portalToggleVoteRequest struct {
	VoterToken string `json:"voter_token" binding:"required"`
}

// POST /v1/portal/:orgSlug/:repoName/features/:featureId/vote
func (h *PortalHandlersImpl) ToggleVote(c *gin.Context) {
	orgSlug := c.Param("orgSlug")
	repoName := c.Param("repoName")
	featureID, err := strconv.ParseInt(c.Param("featureId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid feature ID"})
		return
	}

	var req portalToggleVoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "voter_token is required"})
		return
	}

	count, err := h.s.ToggleVote(c.Request.Context(), orgSlug, repoName, featureID, req.VoterToken)
	if errors.Is(err, services.ErrPortalNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "portal not found or not public"})
		return
	}
	if err != nil {
		h.l.Errorw("Failed to toggle vote", "error", err, "feature_id", featureID, "request_id", request.GetRequestID(c))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"vote_count": count})
}

// GET /v1/portal/:orgSlug/:repoName/features/:featureId/comments
func (h *PortalHandlersImpl) ListComments(c *gin.Context) {
	orgSlug := c.Param("orgSlug")
	repoName := c.Param("repoName")
	featureID, err := strconv.ParseInt(c.Param("featureId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid feature ID"})
		return
	}

	comments, err := h.s.ListComments(c.Request.Context(), orgSlug, repoName, featureID)
	if errors.Is(err, services.ErrPortalNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "portal not found or not public"})
		return
	}
	if err != nil {
		h.l.Errorw("Failed to list comments", "error", err, "feature_id", featureID, "request_id", request.GetRequestID(c))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"comments": comments})
}

type portalCreateCommentRequest struct {
	Body       string `json:"body" binding:"required"`
	AuthorName string `json:"author_name"`
}

// POST /v1/portal/:orgSlug/:repoName/features/:featureId/comments
func (h *PortalHandlersImpl) CreateComment(c *gin.Context) {
	orgSlug := c.Param("orgSlug")
	repoName := c.Param("repoName")
	featureID, err := strconv.ParseInt(c.Param("featureId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid feature ID"})
		return
	}

	var req portalCreateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "body is required"})
		return
	}

	comment, err := h.s.CreateComment(c.Request.Context(), orgSlug, repoName, featureID, req.Body, req.AuthorName)
	if errors.Is(err, services.ErrPortalNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "portal not found or not public"})
		return
	}
	if err != nil {
		h.l.Errorw("Failed to create comment", "error", err, "feature_id", featureID, "request_id", request.GetRequestID(c))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusCreated, comment)
}
