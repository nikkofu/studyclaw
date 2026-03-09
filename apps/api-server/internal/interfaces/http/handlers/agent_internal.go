package handlers

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	taskparse "github.com/nikkofu/studyclaw/api-server/internal/modules/agent/taskparse"
	weeklyinsights "github.com/nikkofu/studyclaw/api-server/internal/modules/agent/weeklyinsights"
)

type AgentInternalHandler struct {
	parser *taskparse.Service
	weekly *weeklyinsights.Service
}

type InternalParseRequest struct {
	RawText string `json:"raw_text" binding:"required"`
}

type InternalWeeklyRequest struct {
	DaysData []map[string]interface{} `json:"days_data" binding:"required"`
}

func NewAgentInternalHandler(parser *taskparse.Service, weekly *weeklyinsights.Service) *AgentInternalHandler {
	return &AgentInternalHandler{
		parser: parser,
		weekly: weekly,
	}
}

func (h *AgentInternalHandler) Parse(c *gin.Context) {
	var req InternalParseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	result, err := h.parser.Parse(context.Background(), strings.TrimSpace(req.RawText))
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to parse task text"})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *AgentInternalHandler) AnalyzeWeekly(c *gin.Context) {
	var req InternalWeeklyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	insights, err := h.weekly.Generate(context.Background(), req.DaysData)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to generate weekly insights"})
		return
	}

	c.JSON(http.StatusOK, insights)
}
