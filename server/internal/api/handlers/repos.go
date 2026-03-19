package api_handlers

import (
	"net/http"
	"strconv"
	"time"

	api_services "github.com/didrikolofsson/github-vote-llm/internal/api/services"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/gin-gonic/gin"
)

// RepoConfigResponse is the API representation of a repo configuration.
// AnthropicAPIKey is intentionally excluded.
type RepoConfigResponse struct {
	ID                  int64     `json:"id"`
	Owner               string    `json:"owner"`
	Repo                string    `json:"repo"`
	LabelApproved       string    `json:"label_approved"`
	LabelInProgress     string    `json:"label_in_progress"`
	LabelDone           string    `json:"label_done"`
	LabelFailed         string    `json:"label_failed"`
	LabelFeatureRequest string    `json:"label_feature_request"`
	VoteThreshold       int32     `json:"vote_threshold"`
	TimeoutMinutes      int32     `json:"timeout_minutes"`
	MaxBudgetUsd        float64   `json:"max_budget_usd"`
	IsBoardPublic       bool      `json:"is_board_public"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

func toRepoConfigResponse(m *store.RepoConfigModel) RepoConfigResponse {
	return RepoConfigResponse{
		ID:                  m.ID,
		Owner:               m.Owner,
		Repo:                m.Repo,
		LabelApproved:       m.LabelApproved,
		LabelInProgress:     m.LabelInProgress,
		LabelDone:           m.LabelDone,
		LabelFailed:         m.LabelFailed,
		LabelFeatureRequest: m.LabelFeatureRequest,
		VoteThreshold:       m.VoteThreshold,
		TimeoutMinutes:      m.TimeoutMinutes,
		MaxBudgetUsd:        m.MaxBudgetUsd,
		IsBoardPublic:       m.IsBoardPublic,
		CreatedAt:           m.CreatedAt,
		UpdatedAt:           m.UpdatedAt,
	}
}

// UpdateRepoConfigRequest holds the optional fields for updating a repo config.
type UpdateRepoConfigRequest struct {
	LabelApproved       *string  `json:"label_approved"`
	LabelInProgress     *string  `json:"label_in_progress"`
	LabelDone           *string  `json:"label_done"`
	LabelFailed         *string  `json:"label_failed"`
	LabelFeatureRequest *string  `json:"label_feature_request"`
	VoteThreshold       *int32   `json:"vote_threshold"`
	TimeoutMinutes      *int32   `json:"timeout_minutes"`
	MaxBudgetUsd        *float64 `json:"max_budget_usd"`
	AnthropicAPIKey     *string  `json:"anthropic_api_key"`
	IsBoardPublic       *bool    `json:"is_board_public"`
}

// ReposHandler handles HTTP requests for repo configuration endpoints.
type ReposHandler struct {
	svc *api_services.ReposService
}

// NewReposHandler creates a new ReposHandler.
func NewReposHandler(svc *api_services.ReposService) *ReposHandler {
	return &ReposHandler{svc: svc}
}

// List handles GET /api/repos
func (h *ReposHandler) List(c *gin.Context) {
	configs, err := h.svc.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	resp := make([]RepoConfigResponse, len(configs))
	for i, cfg := range configs {
		resp[i] = toRepoConfigResponse(cfg)
	}
	c.JSON(http.StatusOK, resp)
}

// GetConfig handles GET /api/repos/:owner/:repo/config
func (h *ReposHandler) GetConfig(c *gin.Context) {
	owner := c.Param("owner")
	repo := c.Param("repo")

	cfg, err := h.svc.GetConfig(c.Request.Context(), owner, repo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if cfg == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "repo config not found"})
		return
	}
	c.JSON(http.StatusOK, toRepoConfigResponse(cfg))
}

// DeleteConfig handles DELETE /api/repos/:owner/:repo/config
func (h *ReposHandler) DeleteConfig(c *gin.Context) {
	owner := c.Param("owner")
	repo := c.Param("repo")

	if err := h.svc.DeleteConfig(c.Request.Context(), owner, repo); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// ListRoadmapItems handles GET /api/repos/:owner/:repo/roadmap
func (h *ReposHandler) ListRoadmapItems(c *gin.Context) {
	owner := c.Param("owner")
	repo := c.Param("repo")

	proposals, err := h.svc.ListProposals(c.Request.Context(), owner, repo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	resp := make([]ProposalResponse, len(proposals))
	for i, p := range proposals {
		resp[i] = toProposalResponse(p)
	}
	c.JSON(http.StatusOK, resp)
}

type updateProposalStatusRequest struct {
	Status string `json:"status"`
}

// UpdateProposalStatus handles PATCH /api/repos/:owner/:repo/proposals/:id
func (h *ReposHandler) UpdateProposalStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid proposal id"})
		return
	}

	var req updateProposalStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Status != "open" && req.Status != "planned" && req.Status != "done" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status must be one of: open, planned, done"})
		return
	}

	p, err := h.svc.UpdateProposalStatus(c.Request.Context(), id, req.Status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if p == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "proposal not found"})
		return
	}
	c.JSON(http.StatusOK, toProposalResponse(p))
}

// UpdateConfig handles PUT /api/repos/:owner/:repo/config
func (h *ReposHandler) UpdateConfig(c *gin.Context) {
	owner := c.Param("owner")
	repo := c.Param("repo")

	var req UpdateRepoConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cfg, err := h.svc.UpdateConfig(c.Request.Context(), owner, repo, api_services.UpdateConfigInput{
		LabelApproved:       req.LabelApproved,
		LabelInProgress:     req.LabelInProgress,
		LabelDone:           req.LabelDone,
		LabelFailed:         req.LabelFailed,
		LabelFeatureRequest: req.LabelFeatureRequest,
		VoteThreshold:       req.VoteThreshold,
		TimeoutMinutes:      req.TimeoutMinutes,
		MaxBudgetUsd:        req.MaxBudgetUsd,
		AnthropicAPIKey:     req.AnthropicAPIKey,
		IsBoardPublic:       req.IsBoardPublic,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toRepoConfigResponse(cfg))
}
