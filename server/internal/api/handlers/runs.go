package handlers

import (
	"net/http"
	"strconv"

	"github.com/didrikolofsson/github-vote-llm/internal/api/services"
	"github.com/gin-gonic/gin"
)

type RunsHandlers interface {
	Create(c *gin.Context)
}

type RunsHandlersImpl struct {
	s services.RunService
}

func NewRunsHandlers(s services.RunService) RunsHandlers {
	return &RunsHandlersImpl{s: s}
}

type createRunBody struct {
	Prompt          string `json:"prompt"`
	CreatedByUserID int64  `json:"created_by_user_id"`
}

func (h *RunsHandlersImpl) Create(c *gin.Context) {
	var body createRunBody
	var featureID int64

	featureID, err := strconv.ParseInt(c.Param("featureId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid feature ID"})
		return
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	run, err := h.s.CreateRun(c.Request.Context(), body.Prompt, featureID, body.CreatedByUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusCreated, run)
}
