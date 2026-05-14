package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/didrikolofsson/github-vote-llm/internal/api/request"
	"github.com/didrikolofsson/github-vote-llm/internal/logger"
	"github.com/didrikolofsson/github-vote-llm/internal/services"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/gin-gonic/gin"
)

type FeatureHandlers struct {
	s *services.FeaturesService
	l *logger.Logger
}

func NewFeatureHandlers(s *services.FeaturesService, l *logger.Logger) *FeatureHandlers {
	return &FeatureHandlers{s: s, l: l}
}

func (h *FeatureHandlers) ListFeatures(c *gin.Context) {
	repoID, err := strconv.ParseInt(c.Param("repoId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository ID"})
		return
	}
	features, err := h.s.ListFeatures(c.Request.Context(), repoID)
	if err != nil {
		h.l.Errorw("Failed to list features", "error", err, "request_id", request.GetRequestID(c))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"features": features})
}

func (h *FeatureHandlers) GetFeature(c *gin.Context) {
	featureID, err := strconv.ParseInt(c.Param("featureId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid feature ID"})
		return
	}
	feature, err := h.s.GetFeature(c.Request.Context(), featureID)
	if errors.Is(err, services.ErrFeatureNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "feature not found"})
		return
	}
	if err != nil {
		h.l.Errorw("Failed to get feature", "error", err, "request_id", request.GetRequestID(c))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, feature)
}

type createFeatureRequest struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description"`
}

func (h *FeatureHandlers) CreateFeature(c *gin.Context) {
	repoID, err := strconv.ParseInt(c.Param("repoId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository ID"})
		return
	}
	var req createFeatureRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	feature, err := h.s.CreateFeature(c.Request.Context(), repoID, req.Title, req.Description)
	if err != nil {
		h.l.Errorw("Failed to create feature", "error", err, "request_id", request.GetRequestID(c))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusCreated, feature)
}

func (h *FeatureHandlers) DeleteFeature(c *gin.Context) {
	featureID, ok := featureIDFromContext(c)
	if !ok {
		return
	}
	if err := h.s.DeleteFeature(c.Request.Context(), featureID); err != nil {
		h.l.Errorw("Failed to delete feature", "error", err, "request_id", request.GetRequestID(c))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.Status(http.StatusNoContent)
}

type patchFeatureRequest struct {
	Title        *string `json:"title"`
	Description  *string `json:"description"`
	ReviewStatus *string `json:"review_status"`
	BuildStatus  *string `json:"build_status"`
	Area         *string `json:"area"`
}

func (h *FeatureHandlers) PatchFeature(c *gin.Context) {
	featureID, err := strconv.ParseInt(c.Param("featureId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid feature ID"})
		return
	}
	var req patchFeatureRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	params := services.PatchFeatureParams{
		Title:       req.Title,
		Description: req.Description,
		Area:        req.Area,
	}
	if req.ReviewStatus != nil {
		rs := store.ReviewStatusType(*req.ReviewStatus)
		switch rs {
		case store.ReviewStatusTypePending, store.ReviewStatusTypeApproved, store.ReviewStatusTypeRejected:
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid review_status value"})
			return
		}
		params.ReviewStatus = &rs
	}
	if req.BuildStatus != nil {
		bs := store.BuildStatusType(*req.BuildStatus)
		switch bs {
		case store.BuildStatusTypePending, store.BuildStatusTypeInProgress,
			store.BuildStatusTypeStuck, store.BuildStatusTypeDone, store.BuildStatusTypeRejected:
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid build_status value"})
			return
		}
		params.BuildStatus = &bs
	}
	feature, err := h.s.PatchFeature(c.Request.Context(), featureID, params)
	if errors.Is(err, services.ErrFeatureNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "feature not found"})
		return
	}
	if err != nil {
		h.l.Errorw("Failed to patch feature", "error", err, "request_id", request.GetRequestID(c))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, feature)
}

type updatePositionRequest struct {
	X      *float64 `json:"x"`
	Y      *float64 `json:"y"`
	Locked bool     `json:"locked"`
}

func (h *FeatureHandlers) UpdatePosition(c *gin.Context) {
	featureID, err := strconv.ParseInt(c.Param("featureId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid feature ID"})
		return
	}
	var req updatePositionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	feature, err := h.s.UpdatePosition(c.Request.Context(), featureID, req.X, req.Y, req.Locked)
	if errors.Is(err, services.ErrFeatureNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "feature not found"})
		return
	}
	if err != nil {
		h.l.Errorw("Failed to update feature position", "error", err, "request_id", request.GetRequestID(c))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, feature)
}

func (h *FeatureHandlers) GetRoadmap(c *gin.Context) {
	repoID, err := strconv.ParseInt(c.Param("repoId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository ID"})
		return
	}
	roadmap, err := h.s.GetRoadmap(c.Request.Context(), repoID)
	if err != nil {
		h.l.Errorw("Failed to get roadmap", "error", err, "request_id", request.GetRequestID(c))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, roadmap)
}

type addDependencyRequest struct {
	DependsOn int64 `json:"depends_on" binding:"required"`
}

func (h *FeatureHandlers) AddDependency(c *gin.Context) {
	featureID, err := strconv.ParseInt(c.Param("featureId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid feature ID"})
		return
	}
	var req addDependencyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	if err := h.s.AddDependency(c.Request.Context(), featureID, req.DependsOn); err != nil {
		h.l.Errorw("Failed to add dependency", "error", err, "request_id", request.GetRequestID(c))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.Status(http.StatusCreated)
}

func (h *FeatureHandlers) RemoveDependency(c *gin.Context) {
	featureID, err := strconv.ParseInt(c.Param("featureId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid feature ID"})
		return
	}
	dependsOn, err := strconv.ParseInt(c.Param("dependsOn"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid dependency ID"})
		return
	}
	if err := h.s.RemoveDependency(c.Request.Context(), featureID, dependsOn); err != nil {
		h.l.Errorw("Failed to remove dependency", "error", err, "request_id", request.GetRequestID(c))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.Status(http.StatusNoContent)
}

type toggleVoteRequest struct {
	VoterToken string `json:"voter_token" binding:"required"`
	Reason     string `json:"reason"`
	Urgency    string `json:"urgency"`
}

func (h *FeatureHandlers) ToggleVote(c *gin.Context) {
	featureID, err := strconv.ParseInt(c.Param("featureId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid feature ID"})
		return
	}
	var req toggleVoteRequest
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
	count, err := h.s.ToggleVote(c.Request.Context(), featureID, req.VoterToken, req.Reason, urgency)
	if err != nil {
		h.l.Errorw("Failed to toggle vote", "error", err, "request_id", request.GetRequestID(c))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"vote_count": count})
}

func (h *FeatureHandlers) ListComments(c *gin.Context) {
	featureID, err := strconv.ParseInt(c.Param("featureId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid feature ID"})
		return
	}
	comments, err := h.s.ListComments(c.Request.Context(), featureID)
	if err != nil {
		h.l.Errorw("Failed to list comments", "error", err, "request_id", request.GetRequestID(c))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"comments": comments})
}

type createCommentRequest struct {
	Body       string `json:"body" binding:"required"`
	AuthorName string `json:"author_name"`
}

func (h *FeatureHandlers) CreateComment(c *gin.Context) {
	featureID, err := strconv.ParseInt(c.Param("featureId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid feature ID"})
		return
	}
	var req createCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	authorName := req.AuthorName
	if authorName == "" {
		authorName = "Anonymous"
	}
	comment, err := h.s.CreateComment(c.Request.Context(), featureID, req.Body, authorName)
	if err != nil {
		h.l.Errorw("Failed to create comment", "error", err, "request_id", request.GetRequestID(c))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusCreated, comment)
}

func featureIDFromContext(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("featureId"), 10, 64)
	if err != nil {
		return 0, false
	}
	return id, true
}
