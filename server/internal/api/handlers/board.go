package handlers

import (
	"net/http"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/gin-gonic/gin"
)

type ProposalResponse struct {
	ID          int64     `json:"id"`
	Owner       string    `json:"owner"`
	Repo        string    `json:"repo"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	VoteCount   int32     `json:"vote_count"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ProposalCommentResponse struct {
	ID         int64     `json:"id"`
	ProposalID int64     `json:"proposal_id"`
	Body       string    `json:"body"`
	AuthorName string    `json:"author_name"`
	CreatedAt  time.Time `json:"created_at"`
}

func toProposalResponse(m *store.ProposalModel) ProposalResponse {
	return ProposalResponse{
		ID:          m.ID,
		Owner:       m.Owner,
		Repo:        m.Repo,
		Title:       m.Title,
		Description: m.Description,
		VoteCount:   m.VoteCount,
		Status:      m.Status,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}

// BoardHandler handles public-facing board API endpoints (no auth required).
type BoardHandler struct {
	store  store.Store
}

// NewBoardHandler creates a new BoardHandler.
func NewBoardHandler(s store.Store) *BoardHandler {
	return &BoardHandler{store: s}
}

// checkBoardPublic returns false and writes a 403 if the board is not public.
func (h *BoardHandler) checkBoardPublic(c *gin.Context, owner, repo string) bool {
	cfg, err := h.store.GetRepoConfig(c.Request.Context(), owner, repo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check board visibility"})
		return false
	}
	if cfg == nil || !cfg.IsBoardPublic {
		c.JSON(http.StatusForbidden, gin.H{"error": "board is not public"})
		return false
	}
	return true
}

// ListProposals handles GET /board/:owner/:repo/proposals
func (h *BoardHandler) ListProposals(c *gin.Context) {
	owner := c.Param("owner")
	repo := c.Param("repo")
	if !h.checkBoardPublic(c, owner, repo) {
		return
	}

	proposals, err := h.store.ListProposals(c.Request.Context(), owner, repo)
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

type createProposalRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

// CreateProposal handles POST /board/:owner/:repo/proposals
func (h *BoardHandler) CreateProposal(c *gin.Context) {
	owner := c.Param("owner")
	repo := c.Param("repo")
	if !h.checkBoardPublic(c, owner, repo) {
		return
	}

	var req createProposalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if utf8.RuneCountInString(req.Title) < 3 || utf8.RuneCountInString(req.Title) > 200 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "title must be between 3 and 200 characters"})
		return
	}
	if utf8.RuneCountInString(req.Description) > 5000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "description must be at most 5000 characters"})
		return
	}

	p, err := h.store.CreateProposal(c.Request.Context(), owner, repo, req.Title, req.Description)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, toProposalResponse(p))
}

// VoteProposal handles POST /board/:owner/:repo/proposals/:id/vote
func (h *BoardHandler) VoteProposal(c *gin.Context) {
	owner := c.Param("owner")
	repo := c.Param("repo")
	if !h.checkBoardPublic(c, owner, repo) {
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid proposal id"})
		return
	}

	p, err := h.store.IncrementProposalVote(c.Request.Context(), id)
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

// ListComments handles GET /board/:owner/:repo/proposals/:id/comments
func (h *BoardHandler) ListComments(c *gin.Context) {
	owner := c.Param("owner")
	repo := c.Param("repo")
	if !h.checkBoardPublic(c, owner, repo) {
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid proposal id"})
		return
	}

	comments, err := h.store.ListProposalComments(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	resp := make([]ProposalCommentResponse, len(comments))
	for i, cm := range comments {
		resp[i] = ProposalCommentResponse{
			ID:         cm.ID,
			ProposalID: cm.ProposalID,
			Body:       cm.Body,
			AuthorName: cm.AuthorName,
			CreatedAt:  cm.CreatedAt,
		}
	}
	c.JSON(http.StatusOK, resp)
}

type createCommentRequest struct {
	Body       string `json:"body"`
	AuthorName string `json:"author_name"`
}

// CreateComment handles POST /board/:owner/:repo/proposals/:id/comments
func (h *BoardHandler) CreateComment(c *gin.Context) {
	owner := c.Param("owner")
	repo := c.Param("repo")
	if !h.checkBoardPublic(c, owner, repo) {
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid proposal id"})
		return
	}

	var req createCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if utf8.RuneCountInString(req.Body) < 1 || utf8.RuneCountInString(req.Body) > 5000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "comment body must be between 1 and 5000 characters"})
		return
	}
	if utf8.RuneCountInString(req.AuthorName) > 50 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "author name must be at most 50 characters"})
		return
	}
	if req.AuthorName == "" {
		req.AuthorName = "Anonymous"
	}

	cm, err := h.store.CreateProposalComment(c.Request.Context(), id, req.Body, req.AuthorName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, ProposalCommentResponse{
		ID:         cm.ID,
		ProposalID: cm.ProposalID,
		Body:       cm.Body,
		AuthorName: cm.AuthorName,
		CreatedAt:  cm.CreatedAt,
	})
}
