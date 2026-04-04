package handlers

import (
	"net/http"

	"github.com/didrikolofsson/github-vote-llm/internal/api/services"
	"github.com/gin-gonic/gin"
)

type RunsHandlers interface {
	CreateRun(c *gin.Context)
}

type RunsHandlersImpl struct {
	s services.RunService
}

func NewRunsHandlers(s services.RunService) RunsHandlers {
	return &RunsHandlersImpl{s: s}
}

func (h *RunsHandlersImpl) CreateRun(c *gin.Context) {
	var params services.CreateRunParams
	if err := c.ShouldBindJSON(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	run, err := h.s.CreateRun(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusCreated, run)
}
