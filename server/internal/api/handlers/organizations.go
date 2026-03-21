package handlers

import (
	"net/http"

	"github.com/didrikolofsson/github-vote-llm/internal/api/services"
	"github.com/gin-gonic/gin"
)

type OrganizationHandlers interface {
	CreateOrganization(c *gin.Context)
	DeleteOrganization(c *gin.Context)
}

type OrganizationHandlersImpl struct {
	s services.OrganizationService
}

func NewOrganizationHandlers(s services.OrganizationService) OrganizationHandlers {
	return &OrganizationHandlersImpl{s: s}
}

func (h *OrganizationHandlersImpl) CreateOrganization(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Organization created"})
}

func (h *OrganizationHandlersImpl) DeleteOrganization(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Organization deleted"})
}
