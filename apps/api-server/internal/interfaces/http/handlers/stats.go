package handlers

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	taskboardapp "github.com/nikkofu/studyclaw/api-server/internal/modules/taskboard/application"
)

type StatsHandler struct {
	phaseOne *taskboardapp.PhaseOneService
}

func NewStatsHandler(phaseOne *taskboardapp.PhaseOneService) *StatsHandler {
	return &StatsHandler{phaseOne: phaseOne}
}

func (h *StatsHandler) GetDailyStats(c *gin.Context) {
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

	stats, err := h.phaseOne.GetDailyStats(familyID, userID, targetDate)
	if err != nil {
		log.Printf("Failed to build daily stats: %v", err)
		respondError(c, http.StatusInternalServerError, "internal_error", "Failed to build daily stats", nil)
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *StatsHandler) GetWeeklyStats(c *gin.Context) {
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

	endDate, ok := parseOptionalDateOrAbort(c, "end_date", c.Query("end_date"))
	if !ok {
		return
	}

	stats, err := h.phaseOne.GetWeeklyStats(familyID, userID, endDate)
	if err != nil {
		log.Printf("Failed to build weekly stats: %v", err)
		respondError(c, http.StatusInternalServerError, "internal_error", "Failed to build weekly stats", nil)
		return
	}

	rawStats := make([]gin.H, 0, len(stats.CompletionSeries))
	for _, point := range stats.CompletionSeries {
		if point.TotalTasks == 0 {
			continue
		}
		rawStats = append(rawStats, gin.H{
			"date":            point.Date,
			"total_tasks":     point.TotalTasks,
			"completed_tasks": point.CompletedTasks,
			"completion_rate": point.CompletionRate,
		})
	}

	if len(rawStats) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"message":           "No data found for the past week in the Markdown logs.",
			"data":              nil,
			"period":            stats.Period,
			"start_date":        stats.StartDate,
			"end_date":          stats.EndDate,
			"totals":            stats.Totals,
			"subject_breakdown": stats.SubjectBreakdown,
			"completion_series": stats.CompletionSeries,
			"points_series":     stats.PointsSeries,
			"word_series":       stats.WordSeries,
			"encouragement":     stats.Encouragement,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":           "Weekly stats generated successfully",
		"raw_stats":         rawStats,
		"insights":          gin.H{"summary": stats.Encouragement},
		"period":            stats.Period,
		"start_date":        stats.StartDate,
		"end_date":          stats.EndDate,
		"totals":            stats.Totals,
		"subject_breakdown": stats.SubjectBreakdown,
		"completion_series": stats.CompletionSeries,
		"points_series":     stats.PointsSeries,
		"word_series":       stats.WordSeries,
		"encouragement":     stats.Encouragement,
	})
}

func (h *StatsHandler) GetMonthlyStats(c *gin.Context) {
	queryValues, ok := requireQueryParams(c, "family_id", "user_id", "month")
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

	year, month, ok := parseMonthOrAbort(c, queryValues["month"])
	if !ok {
		return
	}

	stats, err := h.phaseOne.GetMonthlyStats(familyID, userID, year, month)
	if err != nil {
		log.Printf("Failed to build monthly stats: %v", err)
		respondError(c, http.StatusInternalServerError, "internal_error", "Failed to build monthly stats", nil)
		return
	}

	c.JSON(http.StatusOK, stats)
}

func parseMonthOrAbort(c *gin.Context, rawValue string) (int, time.Month, bool) {
	trimmed := strings.TrimSpace(rawValue)
	parts := strings.Split(trimmed, "-")
	if len(parts) != 2 {
		respondError(c, http.StatusBadRequest, "invalid_month", "month must be in YYYY-MM format", gin.H{
			"field": "month",
		})
		return 0, 0, false
	}

	year, err := strconv.Atoi(parts[0])
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid_month", "month must be in YYYY-MM format", gin.H{
			"field": "month",
		})
		return 0, 0, false
	}

	monthValue, err := strconv.Atoi(parts[1])
	if err != nil || monthValue < 1 || monthValue > 12 {
		respondError(c, http.StatusBadRequest, "invalid_month", "month must be in YYYY-MM format", gin.H{
			"field": "month",
		})
		return 0, 0, false
	}

	return year, time.Month(monthValue), true
}
