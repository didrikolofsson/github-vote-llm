package handlers

import (
	"net/http"
	"strconv"

	"github.com/didrikolofsson/github-vote-llm/internal/api/request"
	"github.com/didrikolofsson/github-vote-llm/internal/logger"
	"github.com/didrikolofsson/github-vote-llm/internal/services"
	"github.com/gin-gonic/gin"
)

type RunsHandlers struct {
	s *services.RunService
	l *logger.Logger
}

func NewRunsHandlers(s *services.RunService, l *logger.Logger) *RunsHandlers {
	return &RunsHandlers{s: s, l: l}
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
