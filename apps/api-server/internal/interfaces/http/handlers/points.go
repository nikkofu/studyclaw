package handlers

import (
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	taskboardapp "github.com/nikkofu/studyclaw/api-server/internal/modules/taskboard/application"
)

type PointsRequest struct {
	UserID   uint   `json:"user_id" binding:"required"`
	FamilyID uint   `json:"family_id" binding:"required"`
	Amount   int    `json:"amount" binding:"required"`
	Reason   string `json:"reason" binding:"required"`
}

type CreatePointsLedgerReq struct {
	UserID     uint   `json:"user_id" binding:"required"`
	FamilyID   uint   `json:"family_id" binding:"required"`
	Delta      int    `json:"delta" binding:"required"`
	SourceType string `json:"source_type" binding:"required"`
	Note       string `json:"note,omitempty"`
	OccurredOn string `json:"occurred_on,omitempty"`
}

type PointsHandler struct {
	phaseOne *taskboardapp.PhaseOneService
}

func NewPointsHandler(phaseOne *taskboardapp.PhaseOneService) *PointsHandler {
	return &PointsHandler{phaseOne: phaseOne}
}

func (h *PointsHandler) UpdatePoints(c *gin.Context) {
	var req PointsRequest
	if !bindJSONOrAbort(c, &req) {
		return
	}

	sourceType := "parent_reward"
	if req.Amount < 0 {
		sourceType = "parent_penalty"
	}

	entry, balance, err := h.phaseOne.CreateManualPointsEntry(req.FamilyID, req.UserID, time.Now(), req.Amount, sourceType, req.Reason)
	if err != nil {
		handlePointsError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Points updated successfully",
		"balance": balance.Balance,
		"entry":   entry,
	})
}

func (h *PointsHandler) CreateLedgerEntry(c *gin.Context) {
	var req CreatePointsLedgerReq
	if !bindJSONOrAbort(c, &req) {
		return
	}

	occurredOn, ok := parseOptionalDateOrAbort(c, "occurred_on", req.OccurredOn)
	if !ok {
		return
	}

	entry, balance, err := h.phaseOne.CreateManualPointsEntry(req.FamilyID, req.UserID, occurredOn, req.Delta, req.SourceType, req.Note)
	if err != nil {
		handlePointsError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":        "Points ledger entry created successfully",
		"points_entry":   entry,
		"points_balance": balance,
	})
}

func (h *PointsHandler) ListLedger(c *gin.Context) {
	queryValues, ok := requireQueryParams(c, "family_id", "user_id")
	if !ok {
		return
	}

	familyID, ok := parseUintQueryParam(c, "family_id", queryValues["family_id"])
	if !ok {
		return
	}
	userID, ok := parseUintQueryParam(c, "user_id", queryValues["user_id"])
	if !ok {
		return
	}
	startDate, ok := parseOptionalDateOrAbort(c, "start_date", c.Query("start_date"))
	if !ok {
		return
	}
	endDate, ok := parseOptionalDateOrAbort(c, "end_date", c.Query("end_date"))
	if !ok {
		return
	}
	if endDate.Before(startDate) {
		respondError(c, http.StatusBadRequest, "invalid_request", "end_date must not be earlier than start_date", gin.H{
			"field": "end_date",
		})
		return
	}

	entries, err := h.phaseOne.ListPointsLedger(familyID, userID, startDate, endDate)
	if err != nil {
		log.Printf("Failed to list points ledger: %v", err)
		respondError(c, http.StatusInternalServerError, "internal_error", "Failed to load points ledger", nil)
		return
	}

	balance, err := h.phaseOne.GetPointsBalance(familyID, userID, endDate)
	if err != nil {
		log.Printf("Failed to load points balance: %v", err)
		respondError(c, http.StatusInternalServerError, "internal_error", "Failed to load points balance", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"entries":        entries,
		"points_balance": balance,
	})
}

func (h *PointsHandler) GetBalance(c *gin.Context) {
	queryValues, ok := requireQueryParams(c, "family_id", "user_id")
	if !ok {
		return
	}

	familyID, ok := parseUintQueryParam(c, "family_id", queryValues["family_id"])
	if !ok {
		return
	}
	userID, ok := parseUintQueryParam(c, "user_id", queryValues["user_id"])
	if !ok {
		return
	}
	targetDate, ok := parseOptionalDateOrAbort(c, "date", c.Query("date"))
	if !ok {
		return
	}

	balance, err := h.phaseOne.GetPointsBalance(familyID, userID, targetDate)
	if err != nil {
		log.Printf("Failed to get points balance: %v", err)
		respondError(c, http.StatusInternalServerError, "internal_error", "Failed to load points balance", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"points_balance": balance,
	})
}

func handlePointsError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	errorCode := "internal_error"
	details := any(nil)

	switch {
	case errors.Is(err, taskboardapp.ErrInvalidPointsSource):
		status = http.StatusBadRequest
		errorCode = "invalid_points_source"
		details = gin.H{"field": "source_type"}
	case strings.Contains(err.Error(), "delta must be"):
		status = http.StatusBadRequest
		errorCode = "invalid_request"
		details = gin.H{"field": "delta"}
	}

	respondError(c, status, errorCode, err.Error(), details)
}
