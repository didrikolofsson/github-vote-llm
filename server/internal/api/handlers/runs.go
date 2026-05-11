package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/didrikolofsson/github-vote-llm/internal/api/request"
	"github.com/didrikolofsson/github-vote-llm/internal/hub"
	"github.com/didrikolofsson/github-vote-llm/internal/logger"
	"github.com/didrikolofsson/github-vote-llm/internal/services"
	"github.com/gin-gonic/gin"
)

type RunsHandlers struct {
	s *services.RunService
	h hub.Hub
	l *logger.Logger
}

func NewRunsHandlers(s *services.RunService, l *logger.Logger) *RunsHandlers {
	return &RunsHandlers{s: s, l: l}
}

func (h *RunsHandlers) SetHub(hub hub.Hub) {
	h.h = hub
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
