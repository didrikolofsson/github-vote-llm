package api_handlers

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	api_services "github.com/didrikolofsson/github-vote-llm/internal/api/services"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

// RunResponse is the API representation of an execution.
type RunResponse struct {
	ID          int64     `json:"id"`
	Owner       string    `json:"owner"`
	Repo        string    `json:"repo"`
	IssueNumber int32     `json:"issue_number"`
	Status      string    `json:"status"`
	Branch      *string   `json:"branch"`
	PrURL       *string   `json:"pr_url"`
	Error       *string   `json:"error"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func toRunResponse(m *store.ExecutionModel) RunResponse {
	return RunResponse{
		ID:          m.ID,
		Owner:       m.Owner,
		Repo:        m.Repo,
		IssueNumber: m.IssueNumber,
		Status:      m.Status,
		Branch:      m.Branch,
		PrURL:       m.PrUrl,
		Error:       m.Error,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}

// RunsHandler handles HTTP requests for execution (run) endpoints.
type RunsHandler struct {
	svc *api_services.RunsService
}

// NewRunsHandler creates a new RunsHandler.
func NewRunsHandler(svc *api_services.RunsService) *RunsHandler {
	return &RunsHandler{svc: svc}
}

// List handles GET /api/runs
func (h *RunsHandler) List(c *gin.Context) {
	limit := int32(20)
	offset := int32(0)

	if v := c.Query("limit"); v != "" {
		n, err := strconv.ParseInt(v, 10, 32)
		if err != nil || n < 1 || n > 100 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be between 1 and 100"})
			return
		}
		limit = int32(n)
	}
	if v := c.Query("offset"); v != "" {
		n, err := strconv.ParseInt(v, 10, 32)
		if err != nil || n < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "offset must be >= 0"})
			return
		}
		offset = int32(n)
	}

	runs, err := h.svc.List(c.Request.Context(), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	resp := make([]RunResponse, len(runs))
	for i, r := range runs {
		resp[i] = toRunResponse(r)
	}
	c.JSON(http.StatusOK, resp)
}

type createRunRequest struct {
	Owner       string `json:"owner" binding:"required"`
	Repo        string `json:"repo" binding:"required"`
	IssueNumber int    `json:"issue_number" binding:"required"`
}

// Create handles POST /api/runs
func (h *RunsHandler) Create(c *gin.Context) {
	var req createRunRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	run, err := h.svc.Create(c.Request.Context(), req.Owner, req.Repo, req.IssueNumber)
	if err != nil {
		if errors.Is(err, store.ErrAlreadyExists) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, toRunResponse(run))
}

// Get handles GET /api/runs/:id
func (h *RunsHandler) Get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	run, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if run == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "run not found"})
		return
	}
	c.JSON(http.StatusOK, toRunResponse(run))
}

// Retry handles POST /api/runs/:id/retry
func (h *RunsHandler) Retry(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	run, err := h.svc.Retry(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusConflict, gin.H{"error": "execution is not in a retryable state"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toRunResponse(run))
}

// Cancel handles POST /api/runs/:id/cancel
func (h *RunsHandler) Cancel(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	run, err := h.svc.Cancel(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusConflict, gin.H{"error": "execution is not in a cancellable state"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, toRunResponse(run))
}

func parseID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return 0, false
	}
	return id, true
}
