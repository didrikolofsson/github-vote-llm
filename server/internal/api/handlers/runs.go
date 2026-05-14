package handlers

import (
	"bufio"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/didrikolofsson/github-vote-llm/internal/api/request"
	"github.com/didrikolofsson/github-vote-llm/internal/dtos"
	"github.com/didrikolofsson/github-vote-llm/internal/hub"
	"github.com/didrikolofsson/github-vote-llm/internal/logger"
	"github.com/didrikolofsson/github-vote-llm/internal/services"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/gin-gonic/gin"
)

type RunsHandlers struct {
	s      *services.RunService
	h      hub.Hub
	logHub *hub.RunLogHub
	l      *logger.Logger
}

func NewRunsHandlers(s *services.RunService, l *logger.Logger) *RunsHandlers {
	return &RunsHandlers{s: s, logHub: s.LogHub(), l: l}
}

func (h *RunsHandlers) SetHub(hub hub.Hub) {
	h.h = hub
}

func (h *RunsHandlers) Get(c *gin.Context) {
	runID, err := strconv.ParseInt(c.Param("runId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid run ID"})
		return
	}

	run, err := h.s.GetRunByID(c.Request.Context(), runID)
	if err != nil {
		h.l.Errorw("Failed to get run", "error", err, "request_id", request.GetRequestID(c))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get run"})
		return
	}

	c.JSON(http.StatusOK, runToDTO(run))
}

func runToDTO(run store.GetRunByIDRow) dtos.RunDTO {
	var completedAt *time.Time
	if run.CompletedAt.Valid {
		completedAt = &run.CompletedAt.Time
	}
	return dtos.RunDTO{
		ID:              run.ID,
		Prompt:          run.Prompt,
		FeatureID:       run.FeatureID,
		Status:          dtos.RunStatus(run.Status),
		CreatedByUserID: run.CreatedByUserID,
		CreatedAt:       run.CreatedAt.Time,
		CompletedAt:     completedAt,
		PRURL:           run.PrUrl,
	}
}

func (h *RunsHandlers) Logs(c *gin.Context) {
	runID, err := strconv.ParseInt(c.Param("runId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid run ID"})
		return
	}

	run, err := h.s.GetRunByID(c.Request.Context(), runID)
	if err != nil {
		h.l.Errorw("Failed to get run for logs", "error", err, "request_id", request.GetRequestID(c))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get run"})
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Writer.Flush()

	ctx := c.Request.Context()
	isTerminal := run.Status == store.FeatureRunStatusCompleted ||
		run.Status == store.FeatureRunStatusFailed ||
		run.Status == store.FeatureRunStatusCancelled

	if isTerminal {
		h.streamLogFile(c, ctx, run, runID)
		return
	}

	// Active run — subscribe to the in-memory hub for zero-latency delivery.
	existing, ch := h.logHub.Subscribe(runID)
	if ch == nil {
		// Run finished between our DB read and hub subscribe — fall back to file.
		h.streamLogFile(c, ctx, run, runID)
		return
	}
	defer h.logHub.Unsubscribe(runID, ch)

	// Stream lines buffered before this subscription.
	for _, line := range existing {
		c.SSEvent("data", line)
	}
	c.Writer.Flush()

	// Stream new lines as they arrive.
	for {
		select {
		case <-ctx.Done():
			return
		case line, ok := <-ch:
			if !ok {
				return // hub closed — run is done
			}
			c.SSEvent("data", line)
			c.Writer.Flush()
		}
	}
}

func (h *RunsHandlers) streamLogFile(c *gin.Context, ctx interface{ Done() <-chan struct{} }, run store.GetRunByIDRow, runID int64) {
	logPath := filepath.Join(run.Workspace, "worktrees", fmt.Sprintf("run-%d.log", runID))
	//nolint:gosec // logPath is workspace-relative, derived from server-controlled workspace + run ID.
	f, err := os.Open(logPath)
	if err != nil {
		return
	}
	defer f.Close() //nolint:errcheck
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
			c.SSEvent("data", scanner.Text())
		}
	}
	c.Writer.Flush()
}

func (h *RunsHandlers) ListByRepository(c *gin.Context) {
	repoID, err := strconv.ParseInt(c.Param("repoId"), 10, 64)
	if err != nil {
		h.l.Errorw("Invalid repository ID", "error", err, "request_id", request.GetRequestID(c))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository ID"})
		return
	}

	runs, err := h.s.ListRunsByRepository(c.Request.Context(), repoID)
	if err != nil {
		h.l.Errorw("Failed to list runs", "error", err, "request_id", request.GetRequestID(c))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list runs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"runs": runs})
}

type createRunBody struct {
	Prompt          string `json:"prompt"`
	CreatedByUserID int64  `json:"created_by_user_id"`
	Owner           string `json:"owner"`
	Name            string `json:"name"`
}

func (h *RunsHandlers) Create(c *gin.Context) {
	featureID, err := strconv.ParseInt(c.Param("featureId"), 10, 64)
	if err != nil {
		h.l.Errorw("Invalid feature ID", "error", err, "request_id", request.GetRequestID(c))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid feature ID"})
		return
	}

	var body createRunBody
	if err := c.ShouldBindJSON(&body); err != nil {
		h.l.Errorw("Invalid request body", "error", err, "request_id", request.GetRequestID(c))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	run, err := h.s.CreateRun(c.Request.Context(), services.CreateRunParams{
		Prompt:    body.Prompt,
		FeatureID: featureID,
		UserID:    body.CreatedByUserID,
	})
	if err != nil {
		h.l.Errorw("Failed to create run", "error", err, "request_id", request.GetRequestID(c))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create run"})
		return
	}

	c.JSON(http.StatusCreated, run)
}

func (h *RunsHandlers) Delete(c *gin.Context) {
	runID, err := strconv.ParseInt(c.Param("runId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid run ID"})
		return
	}

	if err := h.s.DeleteRun(c.Request.Context(), runID); err != nil {
		switch {
		case errors.Is(err, services.ErrRunNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "run not found"})
		case errors.Is(err, services.ErrRunNotDeletable):
			c.JSON(http.StatusConflict, gin.H{"error": "only cancelled runs can be deleted"})
		default:
			h.l.Errorw("Failed to delete run", "error", err, "request_id", request.GetRequestID(c))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete run"})
		}
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *RunsHandlers) Cancel(c *gin.Context) {
	runID, err := strconv.ParseInt(c.Param("runId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid run ID"})
		return
	}

	if err := h.s.CancelRun(c.Request.Context(), runID); err != nil {
		switch {
		case errors.Is(err, services.ErrRunNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "run not found"})
		case errors.Is(err, services.ErrRunNotCancellable):
			c.JSON(http.StatusConflict, gin.H{"error": "run cannot be cancelled in its current state"})
		default:
			h.l.Errorw("Failed to cancel run", "error", err, "request_id", request.GetRequestID(c))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to cancel run"})
		}
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *RunsHandlers) Events(c *gin.Context) {
	repoID, err := strconv.ParseInt(c.Param("repoId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository ID"})
		return
	}

	ch := h.h.Subscribe(repoID)
	defer h.h.Unsubscribe(repoID, ch)

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
			if msg == hub.EventRunUpdated {
				c.SSEvent("event", msg)
				c.Writer.Flush()
			}
		}
	}
}
