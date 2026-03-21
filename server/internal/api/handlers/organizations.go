package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/didrikolofsson/github-vote-llm/internal/api/request"
	"github.com/didrikolofsson/github-vote-llm/internal/api/services"
	"github.com/didrikolofsson/github-vote-llm/internal/logger"
	"github.com/didrikolofsson/github-vote-llm/internal/store"
	"github.com/gin-gonic/gin"
)

type OrganizationHandlers interface {
	CreateOrganization(c *gin.Context)
	GetOrganization(c *gin.Context)
	UpdateOrganization(c *gin.Context)
	DeleteOrganization(c *gin.Context)
}

type OrganizationHandlersImpl struct {
	s services.OrganizationService
	l *logger.Logger
}

func NewOrganizationHandlers(s services.OrganizationService, l *logger.Logger) OrganizationHandlers {
	return &OrganizationHandlersImpl{s: s, l: l}
}

type createOrganizationRequest struct {
	Name    string `json:"name" binding:"required"`
	OwnerID int64  `json:"owner_id" binding:"required"`
}

type updateOrganizationRequest struct {
	Name string `json:"name" binding:"required"`
}

func (h *OrganizationHandlersImpl) CreateOrganization(c *gin.Context) {
	var req createOrganizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.l.Errorw("Invalid request body", "error", err, "request_id", request.GetRequestID(c))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	org, err := h.s.CreateOrganization(c.Request.Context(), services.CreateOrganizationParams{
		Name:    req.Name,
		OwnerID: req.OwnerID,
	})
	if errors.Is(err, services.ErrOrganizationNameExists) {
		h.l.Warnw("Organization name exists", "name", req.Name, "request_id", request.GetRequestID(c))
		c.JSON(http.StatusBadRequest, gin.H{"error": "organization name already exists"})
		return
	}
	if err != nil {
		h.l.Errorw("Failed to create organization", "error", err, "request_id", request.GetRequestID(c))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusCreated, org)
}

func (h *OrganizationHandlersImpl) GetOrganization(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization ID"})
		return
	}

	org, err := h.s.GetOrganizationByID(c.Request.Context(), id)
	if errors.Is(err, services.ErrOrganizationNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "organization not found"})
		return
	}
	if err != nil {
		h.l.Errorw("Failed to get organization", "error", err, "organization_id", id, "request_id", request.GetRequestID(c))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, org)
}

func (h *OrganizationHandlersImpl) UpdateOrganization(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization ID"})
		return
	}

	var req updateOrganizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.l.Errorw("Invalid request body", "error", err, "request_id", request.GetRequestID(c))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	org, err := h.s.UpdateOrganizationByID(c.Request.Context(), id, &store.UpdateOrganizationByIDParams{
		ID:   id,
		Name: req.Name,
	})
	if errors.Is(err, services.ErrOrganizationNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "organization not found"})
		return
	}
	if errors.Is(err, services.ErrOrganizationNameExists) {
		h.l.Warnw("Organization name exists", "name", req.Name, "request_id", request.GetRequestID(c))
		c.JSON(http.StatusBadRequest, gin.H{"error": "organization name already exists"})
		return
	}
	if err != nil {
		h.l.Errorw("Failed to update organization", "error", err, "organization_id", id, "request_id", request.GetRequestID(c))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, org)
}

func (h *OrganizationHandlersImpl) DeleteOrganization(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization ID"})
		return
	}

	if err := h.s.DeleteOrganization(c.Request.Context(), id); err != nil {
		h.l.Errorw("Failed to delete organization", "error", err, "organization_id", id, "request_id", request.GetRequestID(c))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.Status(http.StatusNoContent)
}
