package handlers

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	taskparse "github.com/nikkofu/studyclaw/api-server/internal/modules/agent/taskparse"
	taskboardapp "github.com/nikkofu/studyclaw/api-server/internal/modules/taskboard/application"
	taskboarddomain "github.com/nikkofu/studyclaw/api-server/internal/modules/taskboard/domain"
)

type DailyAssignmentHandler struct {
	phaseOne *taskboardapp.PhaseOneService
	parser   *taskparse.Service
}

type ParseDailyAssignmentDraftReq struct {
	FamilyID     uint   `json:"family_id" binding:"required"`
	ChildID      uint   `json:"child_id" binding:"required"`
	AssignedDate string `json:"assigned_date,omitempty"`
	SourceText   string `json:"source_text" binding:"required"`
}

type PublishDailyAssignmentReq struct {
	FamilyID     uint                       `json:"family_id" binding:"required"`
	ChildID      uint                       `json:"child_id" binding:"required"`
	AssignedDate string                     `json:"assigned_date" binding:"required"`
	DraftID      string                     `json:"draft_id,omitempty"`
	SourceText   string                     `json:"source_text,omitempty"`
	TaskItems    []taskboarddomain.TaskItem `json:"task_items,omitempty"`
}

func NewDailyAssignmentHandler(phaseOne *taskboardapp.PhaseOneService, parser *taskparse.Service) *DailyAssignmentHandler {
	return &DailyAssignmentHandler{
		phaseOne: phaseOne,
		parser:   parser,
	}
}

func (h *DailyAssignmentHandler) ParseDraft(c *gin.Context) {
	var req ParseDailyAssignmentDraftReq
	if !bindJSONOrAbort(c, &req) {
		return
	}

	assignedDate, ok := parseOptionalDateOrAbort(c, "assigned_date", req.AssignedDate)
	if !ok {
		return
	}

	parsed, err := h.parser.Parse(context.Background(), strings.TrimSpace(req.SourceText))
	if err != nil {
		log.Printf("Failed to parse daily assignment draft: %v", err)
		respondError(c, http.StatusBadGateway, "parser_unavailable", "Failed to parse task text", nil)
		return
	}
	if parsed.Status != "success" || len(parsed.Data) == 0 {
		respondError(c, http.StatusUnprocessableEntity, "tasks_not_extractable", "Agent workflow could not extract valid tasks", gin.H{
			"agent_response": parsed,
		})
		return
	}

	draft, err := h.phaseOne.SaveDraft(req.FamilyID, req.ChildID, assignedDate, req.SourceText, parsed.ParserMode, analysisToMap(parsed.Analysis), mapParsedTasksToTaskItems(parsed.Data))
	if err != nil {
		log.Printf("Failed to save daily assignment draft: %v", err)
		respondError(c, http.StatusInternalServerError, "internal_error", "Failed to save daily assignment draft", nil)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":                "Daily assignment draft parsed successfully",
		"daily_assignment_draft": draft,
	})
}

func (h *DailyAssignmentHandler) Publish(c *gin.Context) {
	var req PublishDailyAssignmentReq
	if !bindJSONOrAbort(c, &req) {
		return
	}

	assignedDate, ok := parseOptionalDateOrAbort(c, "assigned_date", req.AssignedDate)
	if !ok {
		return
	}

	if len(req.TaskItems) == 0 && strings.TrimSpace(req.DraftID) == "" {
		respondError(c, http.StatusBadRequest, "invalid_request", "task_items cannot be empty when draft_id is absent", gin.H{
			"field": "task_items",
		})
		return
	}

	dailyAssignment, board, err := h.phaseOne.PublishDailyAssignment(taskboardapp.PublishDailyAssignmentInput{
		FamilyID:     req.FamilyID,
		ChildID:      req.ChildID,
		AssignedDate: assignedDate,
		DraftID:      req.DraftID,
		SourceText:   req.SourceText,
		TaskItems:    req.TaskItems,
	})
	if err != nil {
		switch {
		case errors.Is(err, taskboardapp.ErrDailyAssignmentDraftNotFound):
			respondError(c, http.StatusNotFound, "daily_assignment_draft_not_found", "Daily assignment draft not found", gin.H{
				"draft_id": strings.TrimSpace(req.DraftID),
			})
		default:
			status := http.StatusInternalServerError
			errorCode := "internal_error"
			details := any(nil)
			if strings.Contains(err.Error(), "task_items cannot be empty") {
				status = http.StatusBadRequest
				errorCode = "invalid_request"
				details = gin.H{"field": "task_items"}
			}
			respondError(c, status, errorCode, err.Error(), details)
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":          "Daily assignment published successfully",
		"daily_assignment": dailyAssignment,
		"task_board":       board,
	})
}

func (h *DailyAssignmentHandler) GetDailyAssignment(c *gin.Context) {
	queryValues, ok := requireQueryParams(c, "family_id", "child_id")
	if !ok {
		return
	}

	familyID, ok := parseUintQueryParam(c, "family_id", queryValues["family_id"])
	if !ok {
		return
	}
	childID, ok := parseUintQueryParam(c, "child_id", queryValues["child_id"])
	if !ok {
		return
	}

	targetDate, ok := parseOptionalDateOrAbort(c, "date", c.Query("date"))
	if !ok {
		return
	}

	bundle, err := h.phaseOne.GetDayBundle(familyID, childID, targetDate)
	if err != nil {
		log.Printf("Failed to load day bundle: %v", err)
		respondError(c, http.StatusInternalServerError, "internal_error", "Failed to load daily assignment", nil)
		return
	}

	c.JSON(http.StatusOK, bundle)
}
