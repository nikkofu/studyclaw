package handlers

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nikkofu/studyclaw/api-server/internal/modules/agent/wordparse"
	taskboardapp "github.com/nikkofu/studyclaw/api-server/internal/modules/taskboard/application"
	taskboarddomain "github.com/nikkofu/studyclaw/api-server/internal/modules/taskboard/domain"
)

type WordsHandler struct {
	phaseOne  *taskboardapp.PhaseOneService
	wordParse *wordparse.Service
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

type ParseWordsReq struct {
	RawText string `json:"raw_text" binding:"required"`
}

func NewWordsHandler(phaseOne *taskboardapp.PhaseOneService, wordParse *wordparse.Service) *WordsHandler {
	return &WordsHandler{
		phaseOne:  phaseOne,
		wordParse: wordParse,
	}
}

func (h *WordsHandler) ParseWords(c *gin.Context) {
	var req ParseWordsReq
	if !bindJSONOrAbort(c, &req) {
		return
	}
	log.Printf("[WordsHandler] ParseWords: received raw_text_chars=%d", len(strings.TrimSpace(req.RawText)))

	items, err := h.wordParse.Parse(c.Request.Context(), req.RawText)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "internal_error", "Failed to parse words: "+err.Error(), nil)
		return
	}
	log.Printf("[WordsHandler] ParseWords: completed items=%d", len(items))

	c.JSON(http.StatusOK, gin.H{
		"items": items,
	})
}

func (h *WordsHandler) UpsertWordList(c *gin.Context) {
	var req UpsertWordListReq
	if !bindJSONOrAbort(c, &req) {
		return
	}
	log.Printf("[WordsHandler] UpsertWordList: received family_id=%d child_id=%d assigned_date=%s title=%q language=%s items=%d", req.FamilyID, req.ChildID, strings.TrimSpace(req.AssignedDate), strings.TrimSpace(req.Title), strings.TrimSpace(req.Language), len(req.Items))

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
	log.Printf("[WordsHandler] UpsertWordList: saved word_list_id=%s total_items=%d family_id=%d child_id=%d assigned_date=%s", list.WordListID, list.TotalItems, list.FamilyID, list.ChildID, list.AssignedDate)

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
		startDate := time.Now().AddDate(-1, 0, 0)
		endDate := time.Now().AddDate(0, 0, 1)
		log.Printf("[WordsHandler] GetWordList: listing family_id=%d child_id=%d start_date=%s end_date=%s", familyID, childID, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))

		lists, err := h.phaseOne.ListWordLists(familyID, childID, startDate, endDate)
		if err != nil {
			log.Printf("Failed to list word lists: %v", err)
			respondError(c, http.StatusInternalServerError, "internal_error", "Failed to load word lists", nil)
			return
		}
		log.Printf("[WordsHandler] GetWordList: listed family_id=%d child_id=%d count=%d", familyID, childID, len(lists))
		c.JSON(http.StatusOK, gin.H{
			"word_lists": lists,
		})
		return
	}

	targetDate, ok := parseOptionalDateOrAbort(c, "date", dateStr)
	if !ok {
		return
	}
	log.Printf("[WordsHandler] GetWordList: loading family_id=%d child_id=%d date=%s", familyID, childID, targetDate.Format("2006-01-02"))

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
	log.Printf("[WordsHandler] GetWordList: loaded word_list_id=%s total_items=%d family_id=%d child_id=%d date=%s", list.WordListID, list.TotalItems, list.FamilyID, list.ChildID, list.AssignedDate)

	c.JSON(http.StatusOK, gin.H{
		"word_list": list,
	})
}

func (h *WordsHandler) ListDictationSessions(c *gin.Context) {
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

	dateStr := strings.TrimSpace(c.Query("date"))
	if dateStr == "" {
		startDate := time.Now().AddDate(-1, 0, 0)
		endDate := time.Now().AddDate(0, 0, 1)
		log.Printf("[WordsHandler] ListDictationSessions: listing family_id=%d child_id=%d start_date=%s end_date=%s", familyID, childID, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))

		sessions, err := h.phaseOne.ListDictationSessions(familyID, childID, startDate, endDate)
		if err != nil {
			log.Printf("Failed to list dictation sessions: %v", err)
			respondError(c, http.StatusInternalServerError, "internal_error", "Failed to load dictation sessions", nil)
			return
		}

		log.Printf("[WordsHandler] ListDictationSessions: listed family_id=%d child_id=%d count=%d", familyID, childID, len(sessions))
		c.JSON(http.StatusOK, gin.H{
			"dictation_sessions": sessions,
		})
		return
	}

	targetDate, ok := parseOptionalDateOrAbort(c, "date", dateStr)
	if !ok {
		return
	}
	log.Printf("[WordsHandler] ListDictationSessions: listing family_id=%d child_id=%d date=%s", familyID, childID, targetDate.Format("2006-01-02"))

	sessions, err := h.phaseOne.ListDictationSessions(familyID, childID, targetDate, targetDate)
	if err != nil {
		log.Printf("Failed to list dictation sessions by date: %v", err)
		respondError(c, http.StatusInternalServerError, "internal_error", "Failed to load dictation sessions", nil)
		return
	}

	log.Printf("[WordsHandler] ListDictationSessions: listed family_id=%d child_id=%d date=%s count=%d", familyID, childID, targetDate.Format("2006-01-02"), len(sessions))
	c.JSON(http.StatusOK, gin.H{
		"dictation_sessions": sessions,
	})
}

func (h *WordsHandler) StartDictationSession(c *gin.Context) {
	var req StartDictationSessionReq
	if !bindJSONOrAbort(c, &req) {
		return
	}
	log.Printf("[WordsHandler] StartDictationSession: received family_id=%d child_id=%d assigned_date=%s", req.FamilyID, req.ChildID, strings.TrimSpace(req.AssignedDate))

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
	log.Printf("[WordsHandler] StartDictationSession: started session_id=%s word_list_id=%s family_id=%d child_id=%d total_items=%d", session.SessionID, session.WordListID, session.FamilyID, session.ChildID, session.TotalItems)

	c.JSON(http.StatusCreated, gin.H{
		"message":           "Dictation session started successfully",
		"dictation_session": session,
	})
}

func (h *WordsHandler) GetDictationSession(c *gin.Context) {
	sessionID := strings.TrimSpace(c.Param("session_id"))
	log.Printf("[WordsHandler] GetDictationSession: loading session_id=%s", sessionID)
	session, err := h.phaseOne.GetDictationSession(sessionID)
	if err != nil {
		if errors.Is(err, taskboardapp.ErrDictationSessionNotFound) {
			respondError(c, http.StatusNotFound, "dictation_session_not_found", "Dictation session not found", gin.H{
				"session_id": sessionID,
			})
			return
		}
		log.Printf("Failed to load dictation session: %v", err)
		respondError(c, http.StatusInternalServerError, "internal_error", "Failed to load dictation session", nil)
		return
	}
	log.Printf("[WordsHandler] GetDictationSession: loaded session_id=%s status=%s grading_status=%s current_index=%d completed_items=%d", session.SessionID, session.Status, session.GradingStatus, session.CurrentIndex, session.CompletedItems)

	c.JSON(http.StatusOK, gin.H{
		"dictation_session": session,
	})
}

func (h *WordsHandler) ReplayDictationSession(c *gin.Context) {
	sessionID := strings.TrimSpace(c.Param("session_id"))
	log.Printf("[WordsHandler] ReplayDictationSession: received session_id=%s", sessionID)
	session, err := h.phaseOne.ReplayDictationSession(sessionID)
	if err != nil {
		if errors.Is(err, taskboardapp.ErrDictationSessionNotFound) {
			respondError(c, http.StatusNotFound, "dictation_session_not_found", "Dictation session not found", gin.H{
				"session_id": sessionID,
			})
			return
		}
		log.Printf("Failed to replay dictation session: %v", err)
		respondError(c, http.StatusInternalServerError, "internal_error", "Failed to replay dictation session", nil)
		return
	}
	log.Printf("[WordsHandler] ReplayDictationSession: completed session_id=%s current_index=%d played_count=%d", session.SessionID, session.CurrentIndex, session.PlayedCount)

	c.JSON(http.StatusOK, gin.H{
		"message":           "Dictation session replayed successfully",
		"dictation_session": session,
	})
}

type GradeDictationReq struct {
	Photo    string `json:"photo" binding:"required"` // Base64
	Language string `json:"language"`
	Mode     string `json:"mode"`
}

func (h *WordsHandler) NextDictationSession(c *gin.Context) {
	sessionID := strings.TrimSpace(c.Param("session_id"))
	log.Printf("[WordsHandler] NextDictationSession: received session_id=%s", sessionID)
	session, err := h.phaseOne.AdvanceDictationSession(sessionID)
	if err != nil {
		if errors.Is(err, taskboardapp.ErrDictationSessionNotFound) {
			respondError(c, http.StatusNotFound, "dictation_session_not_found", "Dictation session not found", gin.H{
				"session_id": sessionID,
			})
			return
		}
		log.Printf("Failed to advance dictation session: %v", err)
		respondError(c, http.StatusInternalServerError, "internal_error", "Failed to advance dictation session", nil)
		return
	}
	log.Printf("[WordsHandler] NextDictationSession: completed session_id=%s current_index=%d completed_items=%d status=%s", session.SessionID, session.CurrentIndex, session.CompletedItems, session.Status)

	c.JSON(http.StatusOK, gin.H{
		"message":           "Dictation session advanced successfully",
		"dictation_session": session,
	})
}

func (h *WordsHandler) PreviousDictationSession(c *gin.Context) {
	sessionID := strings.TrimSpace(c.Param("session_id"))
	log.Printf("[WordsHandler] PreviousDictationSession: received session_id=%s", sessionID)
	session, err := h.phaseOne.PreviousDictationSession(sessionID)
	if err != nil {
		if errors.Is(err, taskboardapp.ErrDictationSessionNotFound) {
			respondError(c, http.StatusNotFound, "dictation_session_not_found", "Dictation session not found", gin.H{
				"session_id": sessionID,
			})
			return
		}
		log.Printf("Failed to move dictation session backward: %v", err)
		respondError(c, http.StatusInternalServerError, "internal_error", "Failed to move dictation session backward", nil)
		return
	}
	log.Printf("[WordsHandler] PreviousDictationSession: completed session_id=%s current_index=%d status=%s", session.SessionID, session.CurrentIndex, session.Status)

	c.JSON(http.StatusOK, gin.H{
		"message":           "Dictation session moved to previous item successfully",
		"dictation_session": session,
	})
}

func (h *WordsHandler) GradeDictationSession(c *gin.Context) {
	sessionID := c.Param("session_id")
	log.Printf("[WordsHandler] GradeDictationSession: received request session_id=%s", sessionID)

	var req GradeDictationReq
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[WordsHandler] GradeDictationSession: failed to bind JSON session_id=%s err=%v", sessionID, err)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_json", "message": err.Error()})
		return
	}
	photoSize := len(req.Photo)
	photoHash := shortPhotoHash(req.Photo)
	log.Printf("[WordsHandler] GradeDictationSession: accepted payload session_id=%s photo_bytes=%d photo_sha1=%s language=%s mode=%s", sessionID, photoSize, photoHash, strings.TrimSpace(req.Language), strings.TrimSpace(req.Mode))

	session, err := h.phaseOne.QueueDictationGrading(sessionID)
	if err != nil {
		switch {
		case errors.Is(err, taskboardapp.ErrDictationSessionNotFound):
			respondError(c, http.StatusNotFound, "dictation_session_not_found", "Dictation session not found", nil)
		case errors.Is(err, taskboardapp.ErrDictationGradingInProgress):
			respondError(c, http.StatusConflict, "dictation_grading_in_progress", "Dictation grading is already in progress", nil)
		default:
			log.Printf("[WordsHandler] GradeDictationSession: failed to queue grading session_id=%s err=%v", sessionID, err)
			respondError(c, http.StatusInternalServerError, "internal_error", "Failed to queue dictation grading", nil)
		}
		return
	}

	if updatedSession, debugErr := h.persistDictationDebugContextForSession(session, photoHash, photoSize, req.Language, req.Mode, "queued"); debugErr != nil {
		log.Printf("[WordsHandler] GradeDictationSession: failed to persist debug context session_id=%s err=%v", sessionID, debugErr)
	} else {
		session = updatedSession
	}

	go h.runDictationGrading(sessionID, req.Photo, req.Language, req.Mode, photoHash, photoSize)

	c.JSON(http.StatusAccepted, gin.H{
		"message":           "Dictation grading accepted",
		"dictation_session": session,
	})
}

func (h *WordsHandler) runDictationGrading(sessionID, photo, language, mode, photoHash string, photoSize int) {
	log.Printf("[WordsHandler] Dictation grading worker started session_id=%s photo_bytes=%d photo_sha1=%s", sessionID, photoSize, photoHash)

	if _, err := h.persistDictationDebugContext(sessionID, photoHash, photoSize, language, mode, "processing"); err != nil {
		log.Printf("[WordsHandler] Dictation grading worker failed to persist processing debug context session_id=%s err=%v", sessionID, err)
	}

	if _, err := h.phaseOne.MarkDictationGradingProcessing(sessionID); err != nil {
		log.Printf("[WordsHandler] Dictation grading worker failed to mark processing session_id=%s err=%v", sessionID, err)
		if _, debugErr := h.persistDictationDebugContext(sessionID, photoHash, photoSize, language, mode, "mark_processing_failed"); debugErr != nil {
			log.Printf("[WordsHandler] Dictation grading worker failed to persist mark_processing_failed debug context session_id=%s err=%v", sessionID, debugErr)
		}
		return
	}

	if _, err := h.persistDictationDebugContext(sessionID, photoHash, photoSize, language, mode, "loading_word_list"); err != nil {
		log.Printf("[WordsHandler] Dictation grading worker failed to persist loading_word_list debug context session_id=%s err=%v", sessionID, err)
	}

	session, list, err := h.phaseOne.GetDictationSessionWordList(sessionID)
	if err != nil {
		log.Printf("[WordsHandler] Dictation grading worker failed to load session bundle session_id=%s err=%v", sessionID, err)
		if _, failErr := h.phaseOne.FailDictationGrading(sessionID, err.Error()); failErr != nil {
			log.Printf("[WordsHandler] Dictation grading worker failed to persist load failure session_id=%s err=%v", sessionID, failErr)
		}
		if _, debugErr := h.persistDictationDebugContext(sessionID, photoHash, photoSize, language, mode, "load_word_list_failed"); debugErr != nil {
			log.Printf("[WordsHandler] Dictation grading worker failed to persist load_word_list_failed debug context session_id=%s err=%v", sessionID, debugErr)
		}
		return
	}

	if _, err := h.persistDictationDebugContextForSession(session, photoHash, photoSize, language, mode, "llm_grading"); err != nil {
		log.Printf("[WordsHandler] Dictation grading worker failed to persist llm_grading debug context session_id=%s err=%v", sessionID, err)
	}

	wordItems := make([]wordparse.ParsedWord, 0, len(list.Items))
	for _, item := range list.Items {
		wordItems = append(wordItems, wordparse.ParsedWord{
			Text:    item.Text,
			Meaning: item.Meaning,
		})
	}

	result, err := h.wordParse.GradeDictation(context.Background(), wordItems, photo, language, mode)
	if err != nil {
		log.Printf("[WordsHandler] Dictation grading worker failed session_id=%s word_list_id=%s err=%v", sessionID, session.WordListID, err)
		if _, failErr := h.phaseOne.FailDictationGrading(sessionID, err.Error()); failErr != nil {
			log.Printf("[WordsHandler] Dictation grading worker failed to persist grading failure session_id=%s err=%v", sessionID, failErr)
		}
		if _, debugErr := h.persistDictationDebugContextForSession(session, photoHash, photoSize, language, mode, "llm_grading_failed"); debugErr != nil {
			log.Printf("[WordsHandler] Dictation grading worker failed to persist llm_grading_failed debug context session_id=%s err=%v", sessionID, debugErr)
		}
		return
	}

	domainResult := toDomainGradingResult(result)
	savedSession, err := h.phaseOne.CompleteDictationGrading(sessionID, domainResult)
	if err != nil {
		log.Printf("[WordsHandler] Dictation grading worker failed to persist success session_id=%s err=%v", sessionID, err)
		if _, debugErr := h.persistDictationDebugContextForSession(session, photoHash, photoSize, language, mode, "persist_result_failed"); debugErr != nil {
			log.Printf("[WordsHandler] Dictation grading worker failed to persist persist_result_failed debug context session_id=%s err=%v", sessionID, debugErr)
		}
		return
	}

	if _, err := h.persistDictationDebugContextForSession(savedSession, photoHash, photoSize, language, mode, "completed"); err != nil {
		log.Printf("[WordsHandler] Dictation grading worker failed to persist completed debug context session_id=%s err=%v", sessionID, err)
	}

	log.Printf("[WordsHandler] Dictation grading worker completed session_id=%s grading_status=%s score=%d incorrect_items=%d feedback=%q", savedSession.SessionID, savedSession.GradingStatus, result.Score, countIncorrectItems(result.GradedItems), result.Feedback)
}

func (h *WordsHandler) persistDictationDebugContext(sessionID, photoHash string, photoSize int, language, mode, workerStage string) (taskboarddomain.DictationSession, error) {
	session, err := h.phaseOne.GetDictationSession(sessionID)
	if err != nil {
		return taskboarddomain.DictationSession{}, err
	}

	return h.persistDictationDebugContextForSession(session, photoHash, photoSize, language, mode, workerStage)
}

func (h *WordsHandler) persistDictationDebugContextForSession(session taskboarddomain.DictationSession, photoHash string, photoSize int, language, mode, workerStage string) (taskboarddomain.DictationSession, error) {
	return h.phaseOne.UpsertDictationDebugContext(session.SessionID, taskboarddomain.DictationDebugContext{
		PhotoSHA1:   strings.TrimSpace(photoHash),
		PhotoBytes:  photoSize,
		Language:    strings.TrimSpace(language),
		Mode:        strings.TrimSpace(mode),
		WorkerStage: strings.TrimSpace(workerStage),
		LogFile:     buildDailyLogFileName(time.Now()),
		LogKeywords: buildDictationLogKeywords(session, photoHash),
	})
}

func toDomainGradingResult(result *wordparse.DictationGradeResult) taskboarddomain.DictationGradingResult {
	gradedItems := make([]taskboarddomain.GradedWordItem, 0)
	status := "passed"
	if result != nil {
		gradedItems = make([]taskboarddomain.GradedWordItem, 0, len(result.GradedItems))
		for _, item := range result.GradedItems {
			if !item.IsCorrect || item.NeedsRetry {
				status = "needs_correction"
			}
			gradedItems = append(gradedItems, taskboarddomain.GradedWordItem{
				Index:      item.Index,
				Expected:   item.Expected,
				Meaning:    item.Meaning,
				Actual:     item.Actual,
				IsCorrect:  item.IsCorrect,
				Comment:    item.Comment,
				NeedsRetry: item.NeedsRetry,
			})
		}
	}

	return taskboarddomain.DictationGradingResult{
		Status:      status,
		Score:       safeGradeScore(result),
		GradedItems: gradedItems,
		AIFeedback:  safeGradeFeedback(result),
	}
}

func safeGradeScore(result *wordparse.DictationGradeResult) int {
	if result == nil {
		return 0
	}
	return result.Score
}

func safeGradeFeedback(result *wordparse.DictationGradeResult) string {
	if result == nil {
		return ""
	}
	return strings.TrimSpace(result.Feedback)
}

func countIncorrectItems(items []wordparse.GradedWord) int {
	incorrect := 0
	for _, item := range items {
		if !item.IsCorrect || item.NeedsRetry {
			incorrect++
		}
	}
	return incorrect
}

func shortPhotoHash(photo string) string {
	sum := sha1.Sum([]byte(strings.TrimSpace(photo)))
	return hex.EncodeToString(sum[:])[:12]
}

func buildDailyLogFileName(now time.Time) string {
	return fmt.Sprintf("api-server-%s.log", now.Format("2006-01-02"))
}

func buildDictationLogKeywords(session taskboarddomain.DictationSession, photoHash string) []string {
	candidates := []string{
		fmt.Sprintf("session_id=%s", strings.TrimSpace(session.SessionID)),
		fmt.Sprintf("word_list_id=%s", strings.TrimSpace(session.WordListID)),
		fmt.Sprintf("grading_id=%s", strings.TrimSpace(safeGradingID(session.GradingResult))),
		fmt.Sprintf("photo_sha1=%s", strings.TrimSpace(photoHash)),
	}

	seen := make(map[string]struct{})
	keywords := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" || strings.HasSuffix(candidate, "=") {
			continue
		}
		if _, exists := seen[candidate]; exists {
			continue
		}
		seen[candidate] = struct{}{}
		keywords = append(keywords, candidate)
	}
	return keywords
}

func safeGradingID(result *taskboarddomain.DictationGradingResult) string {
	if result == nil {
		return ""
	}
	return result.GradingID
}
