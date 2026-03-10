package handlers

import (
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	taskboardapp "github.com/nikkofu/studyclaw/api-server/internal/modules/taskboard/application"
	taskboarddomain "github.com/nikkofu/studyclaw/api-server/internal/modules/taskboard/domain"
)

type WordsHandler struct {
	phaseOne *taskboardapp.PhaseOneService
}

type UpsertWordListReq struct {
	FamilyID     uint          `json:"family_id" binding:"required"`
	ChildID      uint          `json:"child_id" binding:"required"`
	AssignedDate string        `json:"assigned_date" binding:"required"`
	Title        string        `json:"title" binding:"required"`
	Language     string        `json:"language" binding:"required"`
	Items        []WordItemReq `json:"items" binding:"required"`
}

type WordItemReq struct {
	Text    string `json:"text"`
	Meaning string `json:"meaning,omitempty"`
	Hint    string `json:"hint,omitempty"`
}

type StartDictationSessionReq struct {
	FamilyID     uint   `json:"family_id" binding:"required"`
	ChildID      uint   `json:"child_id" binding:"required"`
	AssignedDate string `json:"assigned_date" binding:"required"`
}

func NewWordsHandler(phaseOne *taskboardapp.PhaseOneService) *WordsHandler {
	return &WordsHandler{phaseOne: phaseOne}
}

func (h *WordsHandler) UpsertWordList(c *gin.Context) {
	var req UpsertWordListReq
	if !bindJSONOrAbort(c, &req) {
		return
	}

	assignedDate, ok := parseOptionalDateOrAbort(c, "assigned_date", req.AssignedDate)
	if !ok {
		return
	}

	items := make([]taskboarddomain.WordItem, 0, len(req.Items))
	for index, item := range req.Items {
		items = append(items, taskboarddomain.WordItem{
			Index:   index + 1,
			Text:    strings.TrimSpace(item.Text),
			Meaning: strings.TrimSpace(item.Meaning),
			Hint:    strings.TrimSpace(item.Hint),
		})
	}

	list, err := h.phaseOne.UpsertWordList(req.FamilyID, req.ChildID, assignedDate, req.Title, req.Language, items)
	if err != nil {
		status := http.StatusInternalServerError
		errorCode := "internal_error"
		details := any(nil)
		switch {
		case errors.Is(err, taskboardapp.ErrInvalidWordListLanguage):
			status = http.StatusBadRequest
			errorCode = "invalid_request"
			details = gin.H{"field": "language"}
		case strings.Contains(err.Error(), "items cannot be empty"):
			status = http.StatusBadRequest
			errorCode = "invalid_request"
			details = gin.H{"field": "items"}
		}
		respondError(c, status, errorCode, err.Error(), details)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":   "Word list saved successfully",
		"word_list": list,
	})
}

func (h *WordsHandler) GetWordList(c *gin.Context) {
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

	dateStr := c.Query("date")
	if dateStr == "" {
		// List all word lists for this child (using a wide range)
		startDate := time.Now().AddDate(-1, 0, 0) // Past 1 year
		endDate := time.Now().AddDate(0, 0, 1)   // Tomorrow

		lists, err := h.phaseOne.ListWordLists(familyID, childID, startDate, endDate)
		if err != nil {
			log.Printf("Failed to list word lists: %v", err)
			respondError(c, http.StatusInternalServerError, "internal_error", "Failed to load word lists", nil)
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"word_lists": lists,
		})
		return
	}

	targetDate, ok := parseOptionalDateOrAbort(c, "date", dateStr)
	if !ok {
		return
	}

	list, exists, err := h.phaseOne.GetWordList(familyID, childID, targetDate)
	if err != nil {
		log.Printf("Failed to get word list: %v", err)
		respondError(c, http.StatusInternalServerError, "internal_error", "Failed to load word list", nil)
		return
	}
	if !exists {
		respondError(c, http.StatusNotFound, "word_list_not_found", "Word list not found", gin.H{
			"date": targetDate.Format("2006-01-02"),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"word_list": list,
	})
}

func (h *WordsHandler) StartDictationSession(c *gin.Context) {
	var req StartDictationSessionReq
	if !bindJSONOrAbort(c, &req) {
		return
	}

	assignedDate, ok := parseOptionalDateOrAbort(c, "assigned_date", req.AssignedDate)
	if !ok {
		return
	}

	session, err := h.phaseOne.StartDictationSession(req.FamilyID, req.ChildID, assignedDate)
	if err != nil {
		if errors.Is(err, taskboardapp.ErrWordListNotFound) {
			respondError(c, http.StatusNotFound, "word_list_not_found", "Word list not found", gin.H{
				"date": assignedDate.Format("2006-01-02"),
			})
			return
		}
		log.Printf("Failed to start dictation session: %v", err)
		respondError(c, http.StatusInternalServerError, "internal_error", "Failed to start dictation session", nil)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":           "Dictation session started successfully",
		"dictation_session": session,
	})
}

func (h *WordsHandler) GetDictationSession(c *gin.Context) {
	session, err := h.phaseOne.GetDictationSession(c.Param("session_id"))
	if err != nil {
		if errors.Is(err, taskboardapp.ErrDictationSessionNotFound) {
			respondError(c, http.StatusNotFound, "dictation_session_not_found", "Dictation session not found", gin.H{
				"session_id": strings.TrimSpace(c.Param("session_id")),
			})
			return
		}
		log.Printf("Failed to load dictation session: %v", err)
		respondError(c, http.StatusInternalServerError, "internal_error", "Failed to load dictation session", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"dictation_session": session,
	})
}

func (h *WordsHandler) ReplayDictationSession(c *gin.Context) {
	session, err := h.phaseOne.ReplayDictationSession(c.Param("session_id"))
	if err != nil {
		if errors.Is(err, taskboardapp.ErrDictationSessionNotFound) {
			respondError(c, http.StatusNotFound, "dictation_session_not_found", "Dictation session not found", gin.H{
				"session_id": strings.TrimSpace(c.Param("session_id")),
			})
			return
		}
		log.Printf("Failed to replay dictation session: %v", err)
		respondError(c, http.StatusInternalServerError, "internal_error", "Failed to replay dictation session", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":           "Dictation session replayed successfully",
		"dictation_session": session,
	})
}

func (h *WordsHandler) NextDictationSession(c *gin.Context) {
	session, err := h.phaseOne.AdvanceDictationSession(c.Param("session_id"))
	if err != nil {
		if errors.Is(err, taskboardapp.ErrDictationSessionNotFound) {
			respondError(c, http.StatusNotFound, "dictation_session_not_found", "Dictation session not found", gin.H{
				"session_id": strings.TrimSpace(c.Param("session_id")),
			})
			return
		}
		log.Printf("Failed to advance dictation session: %v", err)
		respondError(c, http.StatusInternalServerError, "internal_error", "Failed to advance dictation session", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":           "Dictation session advanced successfully",
		"dictation_session": session,
	})
}
