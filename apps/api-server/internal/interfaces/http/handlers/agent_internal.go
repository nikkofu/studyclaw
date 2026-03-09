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
	if !bindJSONOrAbort(c, &req) {
		return
	}

	result, err := h.parser.Parse(context.Background(), strings.TrimSpace(req.RawText))
	if err != nil {
		respondError(c, http.StatusBadGateway, "parser_unavailable", "Failed to parse task text", nil)
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *AgentInternalHandler) AnalyzeWeekly(c *gin.Context) {
	var req InternalWeeklyRequest
	if !bindJSONOrAbort(c, &req) {
		return
	}

	insights, err := h.weekly.Generate(context.Background(), req.DaysData)
	if err != nil {
		respondError(c, http.StatusBadGateway, "internal_error", "Failed to generate weekly insights", nil)
		return
	}

	c.JSON(http.StatusOK, insights)
}
