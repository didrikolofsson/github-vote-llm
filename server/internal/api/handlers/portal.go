package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/didrikolofsson/github-vote-llm/internal/api/request"
	"github.com/didrikolofsson/github-vote-llm/internal/services"
	"github.com/didrikolofsson/github-vote-llm/internal/hub"
	"github.com/didrikolofsson/github-vote-llm/internal/logger"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/gin-gonic/gin"
)

type PortalHandlers struct {
	s *services.PortalService
	l *logger.Logger
	h hub.Hub
}

func NewPortalHandlers(s *services.PortalService, l *logger.Logger, h hub.Hub) *PortalHandlers {
	return &PortalHandlers{s: s, l: l, h: h}
}

// GET /v1/portal/:orgSlug/:repoName?voter_token=<uuid>
func (h *PortalHandlers) GetPortalPage(c *gin.Context) {
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
	Reason     string `json:"reason"`
	Urgency    string `json:"urgency"`
}

// POST /v1/portal/:orgSlug/:repoName/features/:featureId/vote
func (h *PortalHandlers) ToggleVote(c *gin.Context) {
	orgSlug := c.Param("orgSlug")
	repoName := c.Param("repoName")
	featureID, err := strconv.ParseInt(c.Param("featureId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid feature ID"})
		return
	}

	var req portalToggleVoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "voter_token and reason are required"})
		return
	}

	urgency := store.NullVoteUrgencyType{}
	if req.Urgency != "" {
		u := store.VoteUrgencyType(req.Urgency)
		switch u {
		case store.VoteUrgencyTypeBlocking, store.VoteUrgencyTypeImportant, store.VoteUrgencyTypeNiceToHave:
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid urgency value"})
			return
		}
		urgency = store.NullVoteUrgencyType{VoteUrgencyType: u, Valid: true}
	}

	count, err := h.s.ToggleVote(c.Request.Context(), orgSlug, repoName, featureID, req.VoterToken, req.Reason, urgency)
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
func (h *PortalHandlers) ListComments(c *gin.Context) {
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
func (h *PortalHandlers) CreateComment(c *gin.Context) {
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

// GET /v1/portal/:orgSlug/:repoName/events?repo_id=<id>
// EventSource is GET-only and cannot send a JSON body; repo_id must be a query param.
type SubscribeRequestParams struct {
	RepoID int64 `form:"repo_id" binding:"required"`
}

func (h *PortalHandlers) Subscribe(c *gin.Context) {
	var params SubscribeRequestParams
	if err := c.ShouldBindQuery(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request parameters"})
		return
	}

	ch := h.h.Subscribe(params.RepoID)
	defer h.h.Unsubscribe(params.RepoID, ch)

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	c.SSEvent("event", "connected")
	c.Writer.Flush()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case msg := <-ch:
			c.SSEvent("event", msg)
			c.Writer.Flush()
		}
	}

}
