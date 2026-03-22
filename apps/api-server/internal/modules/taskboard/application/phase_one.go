package application

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	taskboarddomain "github.com/nikkofu/studyclaw/api-server/internal/modules/taskboard/domain"
)

var (
	ErrDailyAssignmentDraftNotFound = errors.New("daily assignment draft not found")
	ErrWordListNotFound             = errors.New("word list not found")
	ErrDictationSessionNotFound     = errors.New("dictation session not found")
	ErrDictationGradingInProgress   = errors.New("dictation grading already in progress")
	ErrInvalidPointsSource          = errors.New("invalid points source")
	ErrInvalidWordListLanguage      = errors.New("word list language must be one of zh or en")
	ErrInvalidPersistenceSessionID  = errors.New("persistence session_id is required")
)

type PhaseOneRepository interface {
	SaveDraft(draft taskboarddomain.DailyAssignmentDraft) (taskboarddomain.DailyAssignmentDraft, error)
	GetDraft(draftID string) (taskboarddomain.DailyAssignmentDraft, bool, error)
	SavePublishedAssignment(assignment taskboarddomain.PublishedDailyAssignment) (taskboarddomain.PublishedDailyAssignment, error)
	GetPublishedAssignment(familyID, childID uint, date time.Time) (taskboarddomain.PublishedDailyAssignment, bool, error)
	ListPublishedAssignments(familyID, childID uint, startDate, endDate time.Time) ([]taskboarddomain.PublishedDailyAssignment, error)
	SaveManualPointsEntry(entry taskboarddomain.PointsLedgerEntry) (taskboarddomain.PointsLedgerEntry, error)
	ListManualPointsEntries(familyID, userID uint) ([]taskboarddomain.PointsLedgerEntry, error)
	SaveWordList(list taskboarddomain.WordList) (taskboarddomain.WordList, error)
	GetWordList(familyID, childID uint, date time.Time) (taskboarddomain.WordList, bool, error)
	ListWordLists(familyID, childID uint, startDate, endDate time.Time) ([]taskboarddomain.WordList, error)
	SaveDictationSession(session taskboarddomain.DictationSession) (taskboarddomain.DictationSession, error)
	GetDictationSession(sessionID string) (taskboarddomain.DictationSession, bool, error)
	ListDictationSessions(familyID, childID uint, startDate, endDate time.Time) ([]taskboarddomain.DictationSession, error)
	SaveVoiceLearningSession(session taskboarddomain.VoiceLearningSession) (taskboarddomain.VoiceLearningSession, error)
	ListVoiceLearningSessions(familyID, childID uint, startDate, endDate time.Time) ([]taskboarddomain.VoiceLearningSession, error)
	SavePersistenceEvent(event PersistenceEventRecord) (PersistenceEventRecord, bool, error)
	GetPersistenceEventByIdempotencyKey(idempotencyKey string) (PersistenceEventRecord, bool, error)
	SavePersistenceSessionSnapshot(snapshot PersistenceSessionSnapshot) (PersistenceSessionSnapshot, error)
	GetPersistenceSessionSnapshot(sessionID string) (PersistenceSessionSnapshot, bool, error)
	ListPersistenceEvents(familyID, childID uint, startDate, endDate time.Time) ([]PersistenceEventRecord, error)
	ListPersistenceSessionSnapshots(familyID, childID uint, startDate, endDate time.Time) ([]PersistenceSessionSnapshot, error)
}

type PhaseOneService struct {
	taskboard *Service
	repo      PhaseOneRepository
	flags     hotTaskFeatureFlags
}

type hotTaskFeatureFlags struct {
	launch bool
	resume bool
	reward bool
}

type PublishDailyAssignmentInput struct {
	FamilyID     uint
	ChildID      uint
	AssignedDate time.Time
	DraftID      string
	SourceText   string
	TaskItems    []taskboarddomain.TaskItem
}

type DayBundle struct {
	Date            string                                    `json:"date"`
	Published       bool                                      `json:"published"`
	DailyAssignment *taskboarddomain.PublishedDailyAssignment `json:"daily_assignment"`
	TaskBoard       taskboarddomain.Board                     `json:"task_board"`
	PointsBalance   taskboarddomain.PointsBalance             `json:"points_balance"`
	WordList        *taskboarddomain.WordList                 `json:"word_list,omitempty"`
}

type rangeSnapshot struct {
	Date               time.Time
	Board              taskboarddomain.Board
	AutoPoints         int
	ManualPoints       int
	Balance            int
	WordItems          int
	CompletedWordItems int
	Sessions           int
}

type PersistenceEventRecord struct {
	EventID         string
	SessionID       string
	FamilyID        uint
	ChildID         uint
	AssignedDate    string
	Status          string
	EventType       string
	IdempotencyKey  string
	EffectiveSeconds int
	TotalSeconds    int
	InvalidTrigger  bool
	Makeup          bool
	OccurredAt      string
	CreatedAt       string
}

type PersistenceSessionSnapshot struct {
	SessionID        string
	FamilyID         uint
	ChildID          uint
	AssignedDate     string
	Status           string
	LastEventType    string
	LastEventAt      string
	EffectiveSeconds int
	TotalSeconds     int
	InvalidTriggers  int
	InterruptedCount int
	Completed        bool
	Makeup           bool
	UpdatedAt        string
}

func NewPhaseOneService(taskboard *Service, repo PhaseOneRepository) *PhaseOneService {
	return &PhaseOneService{
		taskboard: taskboard,
		repo:      repo,
		flags: hotTaskFeatureFlags{
			launch: parseHotTaskFlag("hot_task_launch_v1"),
			resume: parseHotTaskFlag("hot_task_resume_v1"),
			reward: parseHotTaskFlag("hot_task_rewards_v1"),
		},
	}
}

func parseHotTaskFlag(key string) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	return value == "1" || value == "true" || value == "yes" || value == "on"
}

func BuildTaskItemsSummary(items []taskboarddomain.TaskItem) taskboarddomain.DailyAssignmentSummary {
	subjects := make(map[string]struct{})
	groups := make(map[string]struct{})
	summary := taskboarddomain.DailyAssignmentSummary{
		TotalTasks:     len(items),
		CompletedTasks: 0,
		PendingTasks:   0,
		Status:         "empty",
	}

	for _, item := range items {
		subject := strings.TrimSpace(item.Subject)
		if subject == "" {
			subject = "未分类"
		}
		groupTitle := strings.TrimSpace(item.GroupTitle)
		if groupTitle == "" {
			groupTitle = strings.TrimSpace(item.Title)
		}
		subjects[subject] = struct{}{}
		groups[subject+"\x00"+groupTitle] = struct{}{}
		if item.NeedsReview {
			summary.NeedsReviewTasks++
		}
		if item.Completed {
			summary.CompletedTasks++
		} else {
			summary.PendingTasks++
		}
	}

	summary.SubjectCount = len(subjects)
	summary.GroupCount = len(groups)
	if summary.TotalTasks > 0 {
		summary.CompletionRate = roundRate(float64(summary.CompletedTasks) / float64(summary.TotalTasks))
	}

	switch {
	case summary.TotalTasks == 0:
		summary.Status = "empty"
	case summary.CompletedTasks == 0:
		summary.Status = "pending"
	case summary.CompletedTasks == summary.TotalTasks:
		summary.Status = "completed"
	default:
		summary.Status = "partial"
	}

	return summary
}

func (s *PhaseOneService) SaveDraft(familyID, childID uint, assignedDate time.Time, sourceText string, parserMode string, analysis map[string]any, items []taskboarddomain.TaskItem) (taskboarddomain.DailyAssignmentDraft, error) {
	draft := taskboarddomain.DailyAssignmentDraft{
		FamilyID:     familyID,
		ChildID:      childID,
		AssignedDate: assignedDate.Format("2006-01-02"),
		SourceText:   strings.TrimSpace(sourceText),
		Status:       taskboarddomain.AssignmentStatusDraft,
		ParserMode:   parserMode,
		Analysis:     analysis,
		TaskItems:    normalizeTaskItems(items),
	}
	draft.Summary = BuildTaskItemsSummary(draft.TaskItems)
	return s.repo.SaveDraft(draft)
}

func (s *PhaseOneService) PublishDailyAssignment(input PublishDailyAssignmentInput) (taskboarddomain.PublishedDailyAssignment, taskboarddomain.Board, error) {
	items := normalizeTaskItems(input.TaskItems)
	if len(items) == 0 && strings.TrimSpace(input.DraftID) != "" {
		draft, ok, err := s.repo.GetDraft(input.DraftID)
		if err != nil {
			return taskboarddomain.PublishedDailyAssignment{}, taskboarddomain.Board{}, err
		}
		if !ok {
			return taskboarddomain.PublishedDailyAssignment{}, taskboarddomain.Board{}, ErrDailyAssignmentDraftNotFound
		}
		items = normalizeTaskItems(draft.TaskItems)
		if strings.TrimSpace(input.SourceText) == "" {
			input.SourceText = draft.SourceText
		}
	}

	if len(items) == 0 {
		return taskboarddomain.PublishedDailyAssignment{}, taskboarddomain.Board{}, fmt.Errorf("task_items cannot be empty")
	}

	boardTasks := boardTasksFromTaskItems(items)
	if err := s.taskboard.ReplaceTasks(input.FamilyID, input.ChildID, input.AssignedDate, boardTasks); err != nil {
		return taskboarddomain.PublishedDailyAssignment{}, taskboarddomain.Board{}, err
	}

	assignment, existing, err := s.repo.GetPublishedAssignment(input.FamilyID, input.ChildID, input.AssignedDate)
	if err != nil {
		return taskboarddomain.PublishedDailyAssignment{}, taskboarddomain.Board{}, err
	}
	if !existing {
		assignment = taskboarddomain.PublishedDailyAssignment{}
	}

	assignment.DraftID = strings.TrimSpace(input.DraftID)
	assignment.FamilyID = input.FamilyID
	assignment.ChildID = input.ChildID
	assignment.AssignedDate = input.AssignedDate.Format("2006-01-02")
	assignment.SourceText = strings.TrimSpace(input.SourceText)
	assignment.Status = taskboarddomain.AssignmentStatusPublished
	assignment.TaskItems = items
	assignment.Summary = BuildTaskItemsSummary(items)

	assignment, err = s.repo.SavePublishedAssignment(assignment)
	if err != nil {
		return taskboarddomain.PublishedDailyAssignment{}, taskboarddomain.Board{}, err
	}

	board, err := s.taskboard.ListBoard(input.FamilyID, input.ChildID, input.AssignedDate)
	if err != nil {
		return taskboarddomain.PublishedDailyAssignment{}, taskboarddomain.Board{}, err
	}
	assignment.TaskItems = mergeTaskItemsWithBoard(assignment.TaskItems, board.Tasks)
	assignment.Summary = BuildTaskItemsSummary(assignment.TaskItems)

	return assignment, board, nil
}

func (s *PhaseOneService) UpsertAssignmentSnapshotFromBoard(familyID, childID uint, date time.Time, draftID, sourceText string) (taskboarddomain.PublishedDailyAssignment, bool, error) {
	board, err := s.taskboard.ListBoard(familyID, childID, date)
	if err != nil {
		return taskboarddomain.PublishedDailyAssignment{}, false, err
	}
	if len(board.Tasks) == 0 {
		return taskboarddomain.PublishedDailyAssignment{}, false, nil
	}

	assignment, existing, err := s.repo.GetPublishedAssignment(familyID, childID, date)
	if err != nil {
		return taskboarddomain.PublishedDailyAssignment{}, false, err
	}
	if !existing {
		assignment = taskboarddomain.PublishedDailyAssignment{}
	}
	if strings.TrimSpace(draftID) != "" {
		assignment.DraftID = strings.TrimSpace(draftID)
	}
	if strings.TrimSpace(sourceText) != "" {
		assignment.SourceText = strings.TrimSpace(sourceText)
	}
	assignment.FamilyID = familyID
	assignment.ChildID = childID
	assignment.AssignedDate = date.Format("2006-01-02")
	assignment.Status = taskboarddomain.AssignmentStatusPublished
	assignment.TaskItems = mergeTaskItemsWithBoard(assignment.TaskItems, board.Tasks)
	assignment.Summary = BuildTaskItemsSummary(assignment.TaskItems)

	saved, err := s.repo.SavePublishedAssignment(assignment)
	if err != nil {
		return taskboarddomain.PublishedDailyAssignment{}, false, err
	}
	return saved, true, nil
}

func (s *PhaseOneService) GetDailyAssignment(familyID, childID uint, date time.Time) (taskboarddomain.PublishedDailyAssignment, bool, error) {
	board, err := s.taskboard.ListBoard(familyID, childID, date)
	if err != nil {
		return taskboarddomain.PublishedDailyAssignment{}, false, err
	}

	assignment, ok, err := s.repo.GetPublishedAssignment(familyID, childID, date)
	if err != nil {
		return taskboarddomain.PublishedDailyAssignment{}, false, err
	}
	if !ok {
		if len(board.Tasks) == 0 {
			return taskboarddomain.PublishedDailyAssignment{}, false, nil
		}
		assignment = taskboarddomain.PublishedDailyAssignment{
			FamilyID:     familyID,
			ChildID:      childID,
			AssignedDate: date.Format("2006-01-02"),
			Status:       taskboarddomain.AssignmentStatusPublished,
		}
	}

	assignment.TaskItems = mergeTaskItemsWithBoard(assignment.TaskItems, board.Tasks)
	assignment.Summary = BuildTaskItemsSummary(assignment.TaskItems)
	return assignment, true, nil
}

func (s *PhaseOneService) GetDayBundle(familyID, childID uint, date time.Time) (DayBundle, error) {
	board, err := s.taskboard.ListBoard(familyID, childID, date)
	if err != nil {
		return DayBundle{}, err
	}

	assignment, published, err := s.GetDailyAssignment(familyID, childID, date)
	if err != nil {
		return DayBundle{}, err
	}

	balance, err := s.GetPointsBalance(familyID, childID, date)
	if err != nil {
		return DayBundle{}, err
	}

	list, ok, err := s.repo.GetWordList(familyID, childID, date)
	if err != nil {
		return DayBundle{}, err
	}

	bundle := DayBundle{
		Date:          date.Format("2006-01-02"),
		Published:     published,
		TaskBoard:     board,
		PointsBalance: balance,
	}
	if published {
		bundle.DailyAssignment = &assignment
	}
	if ok {
		bundle.WordList = &list
	}

	return bundle, nil
}

func (s *PhaseOneService) CreateManualPointsEntry(familyID, userID uint, occurredOn time.Time, delta int, sourceType string, note string) (taskboarddomain.PointsLedgerEntry, taskboarddomain.PointsBalance, error) {
	sourceType = strings.TrimSpace(sourceType)
	switch sourceType {
	case taskboarddomain.PointsSourceParentReward, taskboarddomain.PointsSourceSchoolPraise:
		if delta <= 0 {
			return taskboarddomain.PointsLedgerEntry{}, taskboarddomain.PointsBalance{}, fmt.Errorf("delta must be positive for %s", sourceType)
		}
	case taskboarddomain.PointsSourceParentPenalty, taskboarddomain.PointsSourceSchoolCritic:
		if delta >= 0 {
			return taskboarddomain.PointsLedgerEntry{}, taskboarddomain.PointsBalance{}, fmt.Errorf("delta must be negative for %s", sourceType)
		}
	default:
		return taskboarddomain.PointsLedgerEntry{}, taskboarddomain.PointsBalance{}, ErrInvalidPointsSource
	}

	entry := taskboarddomain.PointsLedgerEntry{
		FamilyID:      familyID,
		UserID:        userID,
		OccurredOn:    occurredOn.Format("2006-01-02"),
		Delta:         delta,
		SourceType:    sourceType,
		SourceOrigin:  taskboarddomain.PointsOriginParent,
		SourceRefType: "manual_adjustment",
		Note:          strings.TrimSpace(note),
	}

	entry, err := s.repo.SaveManualPointsEntry(entry)
	if err != nil {
		return taskboarddomain.PointsLedgerEntry{}, taskboarddomain.PointsBalance{}, err
	}

	entries, err := s.listLedgerEntries(familyID, userID, occurredOn, occurredOn)
	if err != nil {
		return taskboarddomain.PointsLedgerEntry{}, taskboarddomain.PointsBalance{}, err
	}
	for _, existing := range entries {
		if existing.EntryID == entry.EntryID {
			entry = existing
			break
		}
	}

	balance, err := s.GetPointsBalance(familyID, userID, occurredOn)
	if err != nil {
		return taskboarddomain.PointsLedgerEntry{}, taskboarddomain.PointsBalance{}, err
	}

	return entry, balance, nil
}

func (s *PhaseOneService) ListPointsLedger(familyID, userID uint, startDate, endDate time.Time) ([]taskboarddomain.PointsLedgerEntry, error) {
	return s.listLedgerEntries(familyID, userID, startDate, endDate)
}

func (s *PhaseOneService) GetPointsBalance(familyID, userID uint, asOfDate time.Time) (taskboarddomain.PointsBalance, error) {
	allEntries, err := s.entriesUntilDate(familyID, userID, asOfDate)
	if err != nil {
		return taskboarddomain.PointsBalance{}, err
	}

	balance := taskboarddomain.PointsBalance{
		FamilyID: familyID,
		UserID:   userID,
		AsOfDate: asOfDate.Format("2006-01-02"),
	}

	for _, entry := range allEntries {
		balance.Balance += entry.Delta
		if entry.SourceOrigin == taskboarddomain.PointsOriginSystem {
			balance.AutoPoints += entry.Delta
		} else {
			balance.ManualPoints += entry.Delta
		}
		if entry.OccurredOn == asOfDate.Format("2006-01-02") {
			balance.TodayDelta += entry.Delta
		}
	}

	return balance, nil
}

func (s *PhaseOneService) UpsertWordList(familyID, childID uint, assignedDate time.Time, title string, language string, items []taskboarddomain.WordItem) (taskboarddomain.WordList, error) {
	language = strings.TrimSpace(language)
	if language != "zh" && language != "en" {
		return taskboarddomain.WordList{}, ErrInvalidWordListLanguage
	}

	normalizedItems := make([]taskboarddomain.WordItem, 0, len(items))
	for index, item := range items {
		text := strings.TrimSpace(item.Text)
		if text == "" {
			continue
		}
		normalizedItems = append(normalizedItems, taskboarddomain.WordItem{
			Index:   index + 1,
			Text:    text,
			Meaning: strings.TrimSpace(item.Meaning),
			Hint:    strings.TrimSpace(item.Hint),
		})
	}
	if len(normalizedItems) == 0 {
		return taskboarddomain.WordList{}, fmt.Errorf("items cannot be empty")
	}

	list, existing, err := s.repo.GetWordList(familyID, childID, assignedDate)
	if err != nil {
		return taskboarddomain.WordList{}, err
	}
	if !existing {
		list = taskboarddomain.WordList{}
	}
	list.FamilyID = familyID
	list.ChildID = childID
	list.AssignedDate = assignedDate.Format("2006-01-02")
	list.Title = strings.TrimSpace(title)
	list.Language = language
	list.Items = normalizedItems
	list.TotalItems = len(normalizedItems)

	return s.repo.SaveWordList(list)
}

func (s *PhaseOneService) GetWordList(familyID, childID uint, assignedDate time.Time) (taskboarddomain.WordList, bool, error) {
	return s.repo.GetWordList(familyID, childID, assignedDate)
}

func (s *PhaseOneService) ListWordLists(familyID, childID uint, startDate, endDate time.Time) ([]taskboarddomain.WordList, error) {
	return s.repo.ListWordLists(familyID, childID, startDate, endDate)
}

func (s *PhaseOneService) StartDictationSession(familyID, childID uint, assignedDate time.Time) (taskboarddomain.DictationSession, error) {
	list, ok, err := s.repo.GetWordList(familyID, childID, assignedDate)
	if err != nil {
		return taskboarddomain.DictationSession{}, err
	}
	if !ok {
		return taskboarddomain.DictationSession{}, ErrWordListNotFound
	}

	session := taskboarddomain.DictationSession{
		WordListID:         list.WordListID,
		FamilyID:           familyID,
		ChildID:            childID,
		AssignedDate:       assignedDate.Format("2006-01-02"),
		Mode:               "dictation",
		Scene:              "word_list",
		Status:             taskboarddomain.DictationSessionActive,
		GradingStatus:      taskboarddomain.DictationGradingIdle,
		CurrentIndex:       0,
		TotalItems:         len(list.Items),
		PlayedCount:        1,
		TranscriptSegments: []taskboarddomain.TranscriptSegment{},
		MergedTranscript:   "",
		AnalysisSummary: taskboarddomain.DictationAnalysisSummary{
			Status:               "not_started",
			CompletionRatio:      0,
			NeedsRetry:           false,
			Recommendation:       "continue",
			RecommendationReason: "语音会话已创建，等待孩子开始朗读或背诵。",
			Explainability:       []string{"会话已建立，尚未收到 transcript segment。"},
		},
	}
	if len(list.Items) > 0 {
		session.CurrentItem = cloneWordItem(list.Items[0])
	}

	return s.repo.SaveDictationSession(session)
}

func (s *PhaseOneService) ReplayDictationSession(sessionID string) (taskboarddomain.DictationSession, error) {
	session, list, err := s.loadSessionAndWordList(sessionID)
	if err != nil {
		return taskboarddomain.DictationSession{}, err
	}

	if session.Status == taskboarddomain.DictationSessionCompleted {
		return session, nil
	}
	session.PlayedCount++
	if session.CurrentIndex >= 0 && session.CurrentIndex < len(list.Items) {
		session.CurrentItem = cloneWordItem(list.Items[session.CurrentIndex])
	}
	return s.repo.SaveDictationSession(session)
}

func (s *PhaseOneService) PreviousDictationSession(sessionID string) (taskboarddomain.DictationSession, error) {
	session, list, err := s.loadSessionAndWordList(sessionID)
	if err != nil {
		return taskboarddomain.DictationSession{}, err
	}

	if len(list.Items) == 0 {
		session.CurrentItem = nil
		session.CurrentIndex = 0
		return s.repo.SaveDictationSession(session)
	}

	if session.Status == taskboarddomain.DictationSessionCompleted {
		session.Status = taskboarddomain.DictationSessionActive
		session.EndedAt = ""
		if session.CurrentIndex >= len(list.Items) {
			session.CurrentIndex = len(list.Items) - 1
		}
	}

	if session.CurrentIndex > 0 {
		session.CurrentIndex--
	}
	session.CurrentItem = cloneWordItem(list.Items[session.CurrentIndex])
	return s.repo.SaveDictationSession(session)
}

func (s *PhaseOneService) AdvanceDictationSession(sessionID string) (taskboarddomain.DictationSession, error) {
	session, list, err := s.loadSessionAndWordList(sessionID)
	if err != nil {
		return taskboarddomain.DictationSession{}, err
	}

	if session.Status == taskboarddomain.DictationSessionCompleted {
		return session, nil
	}

	session.CompletedItems = maxInt(session.CompletedItems, session.CurrentIndex+1)
	session.CurrentIndex++
	if session.CurrentIndex >= len(list.Items) {
		session.Status = taskboarddomain.DictationSessionCompleted
		session.CompletedItems = len(list.Items)
		session.CurrentItem = nil
		session.EndedAt = time.Now().UTC().Format(time.RFC3339)
	} else {
		session.CurrentItem = cloneWordItem(list.Items[session.CurrentIndex])
		session.PlayedCount++
	}

	return s.repo.SaveDictationSession(session)
}

func (s *PhaseOneService) GetDictationSession(sessionID string) (taskboarddomain.DictationSession, error) {
	session, ok, err := s.repo.GetDictationSession(sessionID)
	if err != nil {
		return taskboarddomain.DictationSession{}, err
	}
	if !ok {
		return taskboarddomain.DictationSession{}, ErrDictationSessionNotFound
	}
	return session, nil
}

func (s *PhaseOneService) ListDictationSessions(familyID, childID uint, startDate, endDate time.Time) ([]taskboarddomain.DictationSession, error) {
	return s.repo.ListDictationSessions(familyID, childID, startDate, endDate)
}

func (s *PhaseOneService) SaveVoiceLearningSession(session taskboarddomain.VoiceLearningSession) (taskboarddomain.VoiceLearningSession, error) {
	session.Status = strings.TrimSpace(session.Status)
	if session.Status == "" {
		session.Status = taskboarddomain.VoiceLearningSessionCompleted
	}
	session.Mode = strings.TrimSpace(session.Mode)
	session.Scene = strings.TrimSpace(session.Scene)
	session.AssignedDate = strings.TrimSpace(session.AssignedDate)
	session.TaskTitle = strings.TrimSpace(session.TaskTitle)
	session.TaskType = strings.TrimSpace(session.TaskType)
	session.ReferenceTitle = strings.TrimSpace(session.ReferenceTitle)
	session.ReferenceAuthor = strings.TrimSpace(session.ReferenceAuthor)
	session.ReferenceSource = strings.TrimSpace(session.ReferenceSource)
	session.MergedTranscript = strings.TrimSpace(session.MergedTranscript)
	session.Summary = strings.TrimSpace(session.Summary)
	session.Encouragement = strings.TrimSpace(session.Encouragement)
	if session.TranscriptSegments == nil {
		session.TranscriptSegments = []taskboarddomain.TranscriptSegment{}
	}
	return s.repo.SaveVoiceLearningSession(session)
}

func (s *PhaseOneService) ListVoiceLearningSessions(familyID, childID uint, startDate, endDate time.Time) ([]taskboarddomain.VoiceLearningSession, error) {
	return s.repo.ListVoiceLearningSessions(familyID, childID, startDate, endDate)
}

func (s *PhaseOneService) SavePersistenceEvent(event PersistenceEventRecord) (PersistenceSessionSnapshot, bool, error) {
	event.EventType = strings.TrimSpace(event.EventType)
	event.Status = strings.TrimSpace(event.Status)
	event.IdempotencyKey = strings.TrimSpace(event.IdempotencyKey)
	event.AssignedDate = strings.TrimSpace(event.AssignedDate)
	event.SessionID = strings.TrimSpace(event.SessionID)
	if event.SessionID == "" {
		return PersistenceSessionSnapshot{}, false, ErrInvalidPersistenceSessionID
	}
	if event.OccurredAt == "" {
		event.OccurredAt = time.Now().UTC().Format(time.RFC3339)
	}

	if event.IdempotencyKey != "" {
		existingEvent, ok, err := s.repo.GetPersistenceEventByIdempotencyKey(event.IdempotencyKey)
		if err != nil {
			return PersistenceSessionSnapshot{}, false, err
		}
		if ok {
			snapshot, found, err := s.repo.GetPersistenceSessionSnapshot(existingEvent.SessionID)
			if err != nil {
				return PersistenceSessionSnapshot{}, false, err
			}
			if !found {
				snapshot, err = s.upsertPersistenceSnapshotFromEvent(existingEvent)
				if err != nil {
					return PersistenceSessionSnapshot{}, false, err
				}
			}
			return snapshot, false, nil
		}
	}

	targetStatus := event.Status
	if targetStatus == "" {
		targetStatus = statusForPersistenceEventType(event.EventType)
	}
	if event.Status == "" {
		event.Status = targetStatus
	}

	if existingSnapshot, ok, err := s.repo.GetPersistenceSessionSnapshot(event.SessionID); err != nil {
		return PersistenceSessionSnapshot{}, false, err
	} else if ok {
		if targetStatus != "" {
			fromStatus := strings.TrimSpace(existingSnapshot.Status)
			if fromStatus == "" {
				fromStatus = taskboarddomain.PersistenceSessionStatusPreparing
			}
			if err := taskboarddomain.ValidatePersistenceTransition(fromStatus, targetStatus); err != nil {
				return PersistenceSessionSnapshot{}, false, err
			}
		}
	} else if targetStatus != "" {
		if err := taskboarddomain.ValidatePersistenceTransition(taskboarddomain.PersistenceSessionStatusPreparing, targetStatus); err != nil {
			return PersistenceSessionSnapshot{}, false, err
		}
	}

	savedEvent, created, err := s.repo.SavePersistenceEvent(event)
	if err != nil {
		return PersistenceSessionSnapshot{}, false, err
	}

	snapshot, err := s.upsertPersistenceSnapshotFromEvent(savedEvent)
	if err != nil {
		return PersistenceSessionSnapshot{}, false, err
	}
	return snapshot, created, nil
}

func (s *PhaseOneService) AggregatePersistenceSummary(familyID, childID uint, startDate, endDate time.Time) (taskboarddomain.PersistenceSummary, error) {
	events, err := s.repo.ListPersistenceEvents(familyID, childID, startDate, endDate)
	if err != nil {
		return taskboarddomain.PersistenceSummary{}, err
	}
	snapshots, err := s.repo.ListPersistenceSessionSnapshots(familyID, childID, startDate, endDate)
	if err != nil {
		return taskboarddomain.PersistenceSummary{}, err
	}

	sessionIDs := make(map[string]struct{})
	completedSessionIDs := make(map[string]struct{})
	totalSeconds := 0
	effectiveSeconds := 0
	invalidTriggers := 0
	for _, event := range events {
		sessionID := strings.TrimSpace(event.SessionID)
		if sessionID != "" {
			sessionIDs[sessionID] = struct{}{}
		}
		switch event.EventType {
		case taskboarddomain.PersistenceEventCompleted:
			if sessionID != "" {
				completedSessionIDs[sessionID] = struct{}{}
			}
		}
		totalSeconds += maxInt(event.TotalSeconds, 0)
		effectiveSeconds += maxInt(event.EffectiveSeconds, 0)
		if event.InvalidTrigger {
			invalidTriggers++
		}
	}

	days := make([]taskboarddomain.PersistenceDayRecord, 0, len(snapshots))
	for _, snapshot := range snapshots {
		sessionID := strings.TrimSpace(snapshot.SessionID)
		if sessionID != "" {
			sessionIDs[sessionID] = struct{}{}
		}
		if snapshot.Completed || strings.TrimSpace(snapshot.Status) == taskboarddomain.PersistenceSessionStatusCompleted {
			if sessionID != "" {
				completedSessionIDs[sessionID] = struct{}{}
			}
		}
		days = append(days, taskboarddomain.PersistenceDayRecord{
			Completed: snapshot.Completed,
			Makeup:    snapshot.Makeup,
		})
	}
	started := len(sessionIDs)
	completed := len(completedSessionIDs)

	summary := taskboarddomain.PersistenceSummary{
		Streak: taskboarddomain.ComputePersistenceStreak(days),
		CompletionRate: taskboarddomain.PersistenceCompletionRate{
			Completed: completed,
			Total:     started,
			Rate:      safeRate(completed, started),
		},
		EffectiveDuration: taskboarddomain.PersistenceEffectiveDuration{
			TotalSeconds:     totalSeconds,
			EffectiveSeconds: effectiveSeconds,
		},
		Guardrails: taskboarddomain.PersistenceGuardrails{
			InvalidTriggerRate: safeRate(invalidTriggers, len(events)),
		},
	}
	return summary, nil
}

func (s *PhaseOneService) upsertPersistenceSnapshotFromEvent(event PersistenceEventRecord) (PersistenceSessionSnapshot, error) {
	snapshot := PersistenceSessionSnapshot{
		SessionID:    event.SessionID,
		FamilyID:     event.FamilyID,
		ChildID:      event.ChildID,
		AssignedDate: event.AssignedDate,
		Status:       event.Status,
		UpdatedAt:    event.OccurredAt,
	}

	if existing, ok, err := s.repo.GetPersistenceSessionSnapshot(event.SessionID); err != nil {
		return PersistenceSessionSnapshot{}, err
	} else if ok {
		snapshot = existing
		snapshot.UpdatedAt = event.OccurredAt
		snapshot.FamilyID = event.FamilyID
		snapshot.ChildID = event.ChildID
		if strings.TrimSpace(event.AssignedDate) != "" {
			snapshot.AssignedDate = event.AssignedDate
		}
		if strings.TrimSpace(event.Status) != "" {
			snapshot.Status = strings.TrimSpace(event.Status)
		}
	}

	if snapshot.Status == "" {
		snapshot.Status = statusForPersistenceEventType(event.EventType)
	}
	snapshot.LastEventType = event.EventType
	snapshot.LastEventAt = event.OccurredAt
	snapshot.TotalSeconds += maxInt(event.TotalSeconds, 0)
	snapshot.EffectiveSeconds += maxInt(event.EffectiveSeconds, 0)
	if event.InvalidTrigger {
		snapshot.InvalidTriggers++
	}
	if event.EventType == taskboarddomain.PersistenceEventInterrupted {
		snapshot.InterruptedCount++
	}
	snapshot.Completed = snapshot.Completed || event.EventType == taskboarddomain.PersistenceEventCompleted || snapshot.Status == taskboarddomain.PersistenceSessionStatusCompleted
	if event.Makeup {
		snapshot.Makeup = true
	}

	return s.repo.SavePersistenceSessionSnapshot(snapshot)
}

func statusForPersistenceEventType(eventType string) string {
	switch eventType {
	case taskboarddomain.PersistenceEventStarted:
		return taskboarddomain.PersistenceSessionStatusActive
	case taskboarddomain.PersistenceEventPaused:
		return taskboarddomain.PersistenceSessionStatusPaused
	case taskboarddomain.PersistenceEventResumed, taskboarddomain.PersistenceEventRecovered:
		return taskboarddomain.PersistenceSessionStatusResumed
	case taskboarddomain.PersistenceEventCompleted:
		return taskboarddomain.PersistenceSessionStatusCompleted
	case taskboarddomain.PersistenceEventAborted:
		return taskboarddomain.PersistenceSessionStatusAborted
	default:
		return ""
	}
}

func (s *PhaseOneService) GetDictationSessionWordList(sessionID string) (taskboarddomain.DictationSession, taskboarddomain.WordList, error) {
	return s.loadSessionAndWordList(sessionID)
}

func (s *PhaseOneService) QueueDictationGrading(sessionID string) (taskboarddomain.DictationSession, error) {
	session, err := s.GetDictationSession(sessionID)
	if err != nil {
		return taskboarddomain.DictationSession{}, err
	}
	if session.GradingStatus == taskboarddomain.DictationGradingPending || session.GradingStatus == taskboarddomain.DictationGradingProcessing {
		return taskboarddomain.DictationSession{}, ErrDictationGradingInProgress
	}

	now := time.Now().UTC().Format(time.RFC3339)
	session.GradingStatus = taskboarddomain.DictationGradingPending
	session.GradingError = ""
	session.GradingRequestedAt = now
	session.GradingCompletedAt = ""
	session.GradingResult = nil
	return s.repo.SaveDictationSession(session)
}

func (s *PhaseOneService) MarkDictationGradingProcessing(sessionID string) (taskboarddomain.DictationSession, error) {
	session, err := s.GetDictationSession(sessionID)
	if err != nil {
		return taskboarddomain.DictationSession{}, err
	}

	session.GradingStatus = taskboarddomain.DictationGradingProcessing
	if strings.TrimSpace(session.GradingRequestedAt) == "" {
		session.GradingRequestedAt = time.Now().UTC().Format(time.RFC3339)
	}
	session.GradingError = ""
	return s.repo.SaveDictationSession(session)
}

func (s *PhaseOneService) CompleteDictationGrading(sessionID string, result taskboarddomain.DictationGradingResult) (taskboarddomain.DictationSession, error) {
	session, err := s.GetDictationSession(sessionID)
	if err != nil {
		return taskboarddomain.DictationSession{}, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if strings.TrimSpace(result.GradingID) == "" {
		result.GradingID = fmt.Sprintf("grading_%d", time.Now().UTC().UnixNano())
	}
	if strings.TrimSpace(result.CreatedAt) == "" {
		result.CreatedAt = now
	}

	session.GradingStatus = taskboarddomain.DictationGradingCompleted
	session.GradingError = ""
	session.GradingCompletedAt = now
	session.GradingResult = &result
	return s.repo.SaveDictationSession(session)
}

func (s *PhaseOneService) FailDictationGrading(sessionID string, message string) (taskboarddomain.DictationSession, error) {
	session, err := s.GetDictationSession(sessionID)
	if err != nil {
		return taskboarddomain.DictationSession{}, err
	}

	session.GradingStatus = taskboarddomain.DictationGradingFailed
	session.GradingError = strings.TrimSpace(message)
	session.GradingCompletedAt = time.Now().UTC().Format(time.RFC3339)
	return s.repo.SaveDictationSession(session)
}

func (s *PhaseOneService) UpsertDictationDebugContext(sessionID string, patch taskboarddomain.DictationDebugContext) (taskboarddomain.DictationSession, error) {
	session, err := s.GetDictationSession(sessionID)
	if err != nil {
		return taskboarddomain.DictationSession{}, err
	}

	session.DebugContext = mergeDictationDebugContext(session.DebugContext, patch)
	return s.repo.SaveDictationSession(session)
}

func mergeDictationDebugContext(current *taskboarddomain.DictationDebugContext, patch taskboarddomain.DictationDebugContext) *taskboarddomain.DictationDebugContext {
	next := taskboarddomain.DictationDebugContext{}
	if current != nil {
		next = *current
		if current.LogKeywords != nil {
			next.LogKeywords = append([]string(nil), current.LogKeywords...)
		}
	}

	if value := strings.TrimSpace(patch.PhotoSHA1); value != "" {
		next.PhotoSHA1 = value
	}
	if patch.PhotoBytes > 0 {
		next.PhotoBytes = patch.PhotoBytes
	}
	if value := strings.TrimSpace(patch.Language); value != "" {
		next.Language = value
	}
	if value := strings.TrimSpace(patch.Mode); value != "" {
		next.Mode = value
	}
	if value := strings.TrimSpace(patch.WorkerStage); value != "" {
		next.WorkerStage = value
	}
	if value := strings.TrimSpace(patch.LogFile); value != "" {
		next.LogFile = value
	}
	if len(patch.LogKeywords) > 0 {
		next.LogKeywords = append([]string(nil), patch.LogKeywords...)
	}

	if isEmptyDictationDebugContext(next) {
		return nil
	}
	return &next
}

func isEmptyDictationDebugContext(ctx taskboarddomain.DictationDebugContext) bool {
	return strings.TrimSpace(ctx.PhotoSHA1) == "" &&
		ctx.PhotoBytes == 0 &&
		strings.TrimSpace(ctx.Language) == "" &&
		strings.TrimSpace(ctx.Mode) == "" &&
		strings.TrimSpace(ctx.WorkerStage) == "" &&
		strings.TrimSpace(ctx.LogFile) == "" &&
		len(ctx.LogKeywords) == 0
}

func (s *PhaseOneService) GetDailyStats(familyID, userID uint, date time.Time) (taskboarddomain.StatsResponse, error) {
	return s.buildStatsResponse("daily", familyID, userID, date, date, nil)
}

func (s *PhaseOneService) GetWeeklyStats(familyID, userID uint, endDate time.Time) (taskboarddomain.StatsResponse, error) {
	startDate := endDate.AddDate(0, 0, -6)
	return s.buildStatsResponse("weekly", familyID, userID, startDate, endDate, nil)
}

func (s *PhaseOneService) GetMonthlyStats(familyID, userID uint, year int, month time.Month) (taskboarddomain.StatsResponse, error) {
	location := time.Local
	startDate := time.Date(year, month, 1, 0, 0, 0, 0, location)
	endDate := startDate.AddDate(0, 1, -1)
	seriesBuilder := func(snapshots []rangeSnapshot) ([]taskboarddomain.CompletionSeriesPoint, []taskboarddomain.PointsSeriesPoint, []taskboarddomain.WordSeriesPoint) {
		type weeklyBucket struct {
			index          int
			startDate      time.Time
			endDate        time.Time
			totalTasks     int
			completedTasks int
			pointsDelta    int
			balance        int
			totalItems     int
			completedItems int
			sessions       int
		}

		buckets := make([]weeklyBucket, 0)
		for _, snapshot := range snapshots {
			index := (snapshot.Date.Day() - 1) / 7
			for len(buckets) <= index {
				start := startDate.AddDate(0, 0, len(buckets)*7)
				end := start.AddDate(0, 0, 6)
				if end.After(endDate) {
					end = endDate
				}
				buckets = append(buckets, weeklyBucket{
					index:     len(buckets),
					startDate: start,
					endDate:   end,
				})
			}

			bucket := &buckets[index]
			bucket.totalTasks += snapshot.Board.Summary.Total
			bucket.completedTasks += snapshot.Board.Summary.Completed
			bucket.pointsDelta += snapshot.AutoPoints + snapshot.ManualPoints
			bucket.balance = snapshot.Balance
			bucket.totalItems += snapshot.WordItems
			bucket.completedItems += snapshot.CompletedWordItems
			bucket.sessions += snapshot.Sessions
		}

		completionSeries := make([]taskboarddomain.CompletionSeriesPoint, 0, len(buckets))
		pointsSeries := make([]taskboarddomain.PointsSeriesPoint, 0, len(buckets))
		wordSeries := make([]taskboarddomain.WordSeriesPoint, 0, len(buckets))

		for _, bucket := range buckets {
			label := fmt.Sprintf("week_%d", bucket.index+1)
			completionSeries = append(completionSeries, taskboarddomain.CompletionSeriesPoint{
				Label:          label,
				Date:           bucket.startDate.Format("2006-01-02"),
				TotalTasks:     bucket.totalTasks,
				CompletedTasks: bucket.completedTasks,
				CompletionRate: safeRate(bucket.completedTasks, bucket.totalTasks),
			})
			pointsSeries = append(pointsSeries, taskboarddomain.PointsSeriesPoint{
				Label:   label,
				Date:    bucket.startDate.Format("2006-01-02"),
				Delta:   bucket.pointsDelta,
				Balance: bucket.balance,
			})
			wordSeries = append(wordSeries, taskboarddomain.WordSeriesPoint{
				Label:          label,
				Date:           bucket.startDate.Format("2006-01-02"),
				TotalItems:     bucket.totalItems,
				CompletedItems: bucket.completedItems,
				Sessions:       bucket.sessions,
			})
		}
		return completionSeries, pointsSeries, wordSeries
	}

	return s.buildStatsResponse("monthly", familyID, userID, startDate, endDate, seriesBuilder)
}

func (s *PhaseOneService) buildStatsResponse(period string, familyID, userID uint, startDate, endDate time.Time, customSeries func([]rangeSnapshot) ([]taskboarddomain.CompletionSeriesPoint, []taskboarddomain.PointsSeriesPoint, []taskboarddomain.WordSeriesPoint)) (taskboarddomain.StatsResponse, error) {
	snapshots, totals, subjects, err := s.collectRangeSnapshots(familyID, userID, startDate, endDate)
	if err != nil {
		return taskboarddomain.StatsResponse{}, err
	}

	response := taskboarddomain.StatsResponse{
		Period:           period,
		StartDate:        startDate.Format("2006-01-02"),
		EndDate:          endDate.Format("2006-01-02"),
		Totals:           totals,
		SubjectBreakdown: subjects,
		Encouragement:    buildEncouragement(period, totals),
	}

	if customSeries != nil {
		response.CompletionSeries, response.PointsSeries, response.WordSeries = customSeries(snapshots)
		return response, nil
	}

	response.CompletionSeries = make([]taskboarddomain.CompletionSeriesPoint, 0, len(snapshots))
	response.PointsSeries = make([]taskboarddomain.PointsSeriesPoint, 0, len(snapshots))
	response.WordSeries = make([]taskboarddomain.WordSeriesPoint, 0, len(snapshots))
	for _, snapshot := range snapshots {
		response.CompletionSeries = append(response.CompletionSeries, taskboarddomain.CompletionSeriesPoint{
			Label:          snapshot.Date.Format("2006-01-02"),
			Date:           snapshot.Date.Format("2006-01-02"),
			TotalTasks:     snapshot.Board.Summary.Total,
			CompletedTasks: snapshot.Board.Summary.Completed,
			CompletionRate: safeRate(snapshot.Board.Summary.Completed, snapshot.Board.Summary.Total),
		})
		response.PointsSeries = append(response.PointsSeries, taskboarddomain.PointsSeriesPoint{
			Label:   snapshot.Date.Format("2006-01-02"),
			Date:    snapshot.Date.Format("2006-01-02"),
			Delta:   snapshot.AutoPoints + snapshot.ManualPoints,
			Balance: snapshot.Balance,
		})
		response.WordSeries = append(response.WordSeries, taskboarddomain.WordSeriesPoint{
			Label:          snapshot.Date.Format("2006-01-02"),
			Date:           snapshot.Date.Format("2006-01-02"),
			TotalItems:     snapshot.WordItems,
			CompletedItems: snapshot.CompletedWordItems,
			Sessions:       snapshot.Sessions,
		})
	}

	return response, nil
}

func (s *PhaseOneService) collectRangeSnapshots(familyID, userID uint, startDate, endDate time.Time) ([]rangeSnapshot, taskboarddomain.StatsTotals, []taskboarddomain.SubjectStats, error) {
	snapshots := make([]rangeSnapshot, 0)
	subjectMap := make(map[string]*taskboarddomain.SubjectStats)

	allEntries, err := s.entriesUntilDate(familyID, userID, endDate)
	if err != nil {
		return nil, taskboarddomain.StatsTotals{}, nil, err
	}

	dailyPointsDelta := make(map[string]int)
	dailyBalance := make(map[string]int)
	totals := taskboarddomain.StatsTotals{}
	runningBalance := 0
	for _, entry := range allEntries {
		dailyBalance[entry.OccurredOn] = entry.BalanceAfter
		if entry.OccurredOn < startDate.Format("2006-01-02") {
			runningBalance = entry.BalanceAfter
		}
		if entry.OccurredOn >= startDate.Format("2006-01-02") && entry.OccurredOn <= endDate.Format("2006-01-02") {
			dailyPointsDelta[entry.OccurredOn] += entry.Delta
			if entry.SourceOrigin == taskboarddomain.PointsOriginSystem {
				totals.AutoPoints += entry.Delta
			} else {
				totals.ManualPoints += entry.Delta
			}
		}
	}
	totals.TotalPointsDelta = totals.AutoPoints + totals.ManualPoints
	if len(allEntries) > 0 {
		totals.PointsBalance = allEntries[len(allEntries)-1].BalanceAfter
	}

	wordLists, err := s.repo.ListWordLists(familyID, userID, startDate, endDate)
	if err != nil {
		return nil, taskboarddomain.StatsTotals{}, nil, err
	}
	wordListByDate := make(map[string]taskboarddomain.WordList, len(wordLists))
	for _, list := range wordLists {
		wordListByDate[list.AssignedDate] = list
	}

	sessions, err := s.repo.ListDictationSessions(familyID, userID, startDate, endDate)
	if err != nil {
		return nil, taskboarddomain.StatsTotals{}, nil, err
	}
	sessionCountByDate := make(map[string]int)
	completedWordByDate := make(map[string]int)
	for _, session := range sessions {
		sessionCountByDate[session.AssignedDate]++
		if session.CompletedItems > completedWordByDate[session.AssignedDate] {
			completedWordByDate[session.AssignedDate] = session.CompletedItems
		}
	}

	for date := startDate; !date.After(endDate); date = date.AddDate(0, 0, 1) {
		board, err := s.taskboard.ListBoard(familyID, userID, date)
		if err != nil {
			return nil, taskboarddomain.StatsTotals{}, nil, err
		}

		dateKey := date.Format("2006-01-02")
		if balance, exists := dailyBalance[dateKey]; exists {
			runningBalance = balance
		}
		wordList, hasWordList := wordListByDate[dateKey]
		wordItems := 0
		if hasWordList {
			wordItems = wordList.TotalItems
		}
		autoPoints := countCompletedTasks(board.Tasks)
		manualPoints := dailyPointsDelta[dateKey] - autoPoints

		snapshot := rangeSnapshot{
			Date:               date,
			Board:              board,
			AutoPoints:         autoPoints,
			ManualPoints:       manualPoints,
			Balance:            runningBalance,
			WordItems:          wordItems,
			CompletedWordItems: minInt(completedWordByDate[dateKey], wordItems),
			Sessions:           sessionCountByDate[dateKey],
		}
		snapshots = append(snapshots, snapshot)

		totals.TotalTasks += board.Summary.Total
		totals.CompletedTasks += board.Summary.Completed
		totals.PendingTasks += board.Summary.Pending
		totals.WordItems += snapshot.WordItems
		totals.CompletedWordItems += snapshot.CompletedWordItems
		totals.DictationSessions += snapshot.Sessions

		for _, group := range board.Groups {
			subject, exists := subjectMap[group.Subject]
			if !exists {
				subject = &taskboarddomain.SubjectStats{Subject: group.Subject}
				subjectMap[group.Subject] = subject
			}
			subject.TotalTasks += group.Total
			subject.CompletedTasks += group.Completed
			subject.PendingTasks += group.Pending
		}
	}

	totals.CompletionRate = safeRate(totals.CompletedTasks, totals.TotalTasks)

	subjects := make([]taskboarddomain.SubjectStats, 0, len(subjectMap))
	for _, subject := range subjectMap {
		subject.CompletionRate = safeRate(subject.CompletedTasks, subject.TotalTasks)
		subjects = append(subjects, *subject)
	}
	sort.Slice(subjects, func(i, j int) bool {
		return subjects[i].Subject < subjects[j].Subject
	})

	return snapshots, totals, subjects, nil
}

func (s *PhaseOneService) listLedgerEntries(familyID, userID uint, startDate, endDate time.Time) ([]taskboarddomain.PointsLedgerEntry, error) {
	allEntries, err := s.entriesUntilDate(familyID, userID, endDate)
	if err != nil {
		return nil, err
	}

	filtered := make([]taskboarddomain.PointsLedgerEntry, 0)
	for _, entry := range allEntries {
		if entry.OccurredOn < startDate.Format("2006-01-02") || entry.OccurredOn > endDate.Format("2006-01-02") {
			continue
		}
		filtered = append(filtered, entry)
	}

	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].OccurredOn == filtered[j].OccurredOn {
			return filtered[i].EntryID < filtered[j].EntryID
		}
		return filtered[i].OccurredOn > filtered[j].OccurredOn
	})
	return filtered, nil
}

func (s *PhaseOneService) entriesUntilDate(familyID, userID uint, endDate time.Time) ([]taskboarddomain.PointsLedgerEntry, error) {
	autoEntries, err := s.buildAutoPointsEntries(familyID, userID, endDate)
	if err != nil {
		return nil, err
	}

	manualEntries, err := s.repo.ListManualPointsEntries(familyID, userID)
	if err != nil {
		return nil, err
	}

	entries := make([]taskboarddomain.PointsLedgerEntry, 0, len(autoEntries)+len(manualEntries))
	entries = append(entries, autoEntries...)
	for _, entry := range manualEntries {
		if entry.OccurredOn > endDate.Format("2006-01-02") {
			continue
		}
		entries = append(entries, entry)
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].OccurredOn == entries[j].OccurredOn {
			return entries[i].EntryID < entries[j].EntryID
		}
		return entries[i].OccurredOn < entries[j].OccurredOn
	})

	balance := 0
	for index := range entries {
		balance += entries[index].Delta
		entries[index].BalanceAfter = balance
	}
	return entries, nil
}

func (s *PhaseOneService) buildAutoPointsEntries(familyID, userID uint, endDate time.Time) ([]taskboarddomain.PointsLedgerEntry, error) {
	dates, err := s.taskboard.ListAvailableDates(familyID, userID)
	if err != nil {
		return nil, err
	}

	entries := make([]taskboarddomain.PointsLedgerEntry, 0)
	for _, date := range dates {
		if date.After(endDate) {
			continue
		}
		tasks, err := s.taskboard.ListTasks(familyID, userID, date)
		if err != nil {
			return nil, err
		}
		for _, task := range tasks {
			if !task.Completed {
				continue
			}
			entries = append(entries, taskboarddomain.PointsLedgerEntry{
				EntryID:       fmt.Sprintf("auto:%s:%d", date.Format("2006-01-02"), task.TaskID),
				FamilyID:      familyID,
				UserID:        userID,
				OccurredOn:    date.Format("2006-01-02"),
				Delta:         1,
				SourceType:    taskboarddomain.PointsSourceTaskCompletion,
				SourceOrigin:  taskboarddomain.PointsOriginSystem,
				SourceRefType: "task_item",
				SourceRefID:   fmt.Sprintf("%s:%d", date.Format("2006-01-02"), task.TaskID),
				Note:          task.Content,
			})
		}
	}
	return entries, nil
}

func (s *PhaseOneService) loadSessionAndWordList(sessionID string) (taskboarddomain.DictationSession, taskboarddomain.WordList, error) {
	session, ok, err := s.repo.GetDictationSession(sessionID)
	if err != nil {
		return taskboarddomain.DictationSession{}, taskboarddomain.WordList{}, err
	}
	if !ok {
		return taskboarddomain.DictationSession{}, taskboarddomain.WordList{}, ErrDictationSessionNotFound
	}

	assignedDate, err := time.Parse("2006-01-02", session.AssignedDate)
	if err != nil {
		return taskboarddomain.DictationSession{}, taskboarddomain.WordList{}, err
	}
	list, ok, err := s.repo.GetWordList(session.FamilyID, session.ChildID, assignedDate)
	if err != nil {
		return taskboarddomain.DictationSession{}, taskboarddomain.WordList{}, err
	}
	if !ok {
		return taskboarddomain.DictationSession{}, taskboarddomain.WordList{}, ErrWordListNotFound
	}
	return session, list, nil
}

func normalizeTaskItems(items []taskboarddomain.TaskItem) []taskboarddomain.TaskItem {
	normalized := make([]taskboarddomain.TaskItem, 0, len(items))
	for index, item := range items {
		title := strings.TrimSpace(item.Title)
		content := strings.TrimSpace(item.Content)
		if title == "" {
			title = content
		}
		if content == "" {
			content = title
		}
		if title == "" {
			continue
		}

		subject, groupTitle, _ := NormalizeTaskFields(item.Subject, item.GroupTitle, title)
		taskID := item.TaskID
		if taskID == 0 {
			taskID = index + 1
		}
		status := "pending"
		if item.Completed {
			status = "completed"
		}
		pointsValue := item.PointsValue
		if pointsValue == 0 {
			pointsValue = 1
		}

		normalized = append(normalized, taskboarddomain.TaskItem{
			TaskID:                 taskID,
			Subject:                subject,
			GroupTitle:             groupTitle,
			Title:                  title,
			Content:                content,
			Type:                   strings.TrimSpace(item.Type),
			Confidence:             item.Confidence,
			NeedsReview:            item.NeedsReview,
			Notes:                  append([]string(nil), item.Notes...),
			Completed:              item.Completed,
			Status:                 status,
			PointsValue:            pointsValue,
			ReferenceTitle:         strings.TrimSpace(item.ReferenceTitle),
			ReferenceAuthor:        strings.TrimSpace(item.ReferenceAuthor),
			ReferenceText:          strings.TrimSpace(item.ReferenceText),
			ReferenceSource:        strings.ToLower(strings.TrimSpace(item.ReferenceSource)),
			HideReferenceFromChild: item.HideReferenceFromChild && strings.TrimSpace(item.ReferenceText) != "",
			AnalysisMode:           strings.TrimSpace(item.AnalysisMode),
		})
	}
	return normalized
}

func boardTasksFromTaskItems(items []taskboarddomain.TaskItem) []taskboarddomain.Task {
	tasks := make([]taskboarddomain.Task, 0, len(items))
	for index, item := range normalizeTaskItems(items) {
		taskID := item.TaskID
		if taskID == 0 {
			taskID = index + 1
		}
		status := "pending"
		if item.Completed {
			status = "completed"
		}
		tasks = append(tasks, taskboarddomain.Task{
			TaskID:                 taskID,
			Completed:              item.Completed,
			Status:                 status,
			Subject:                item.Subject,
			GroupTitle:             item.GroupTitle,
			Content:                item.Title,
			TaskType:               item.Type,
			ReferenceTitle:         item.ReferenceTitle,
			ReferenceAuthor:        item.ReferenceAuthor,
			ReferenceText:          item.ReferenceText,
			ReferenceSource:        item.ReferenceSource,
			HideReferenceFromChild: item.HideReferenceFromChild,
			AnalysisMode:           item.AnalysisMode,
		})
	}
	return tasks
}

func mergeTaskItemsWithBoard(existing []taskboarddomain.TaskItem, boardTasks []taskboarddomain.Task) []taskboarddomain.TaskItem {
	boardItems := make([]taskboarddomain.TaskItem, 0, len(boardTasks))
	for _, task := range boardTasks {
		boardItems = append(boardItems, taskboarddomain.TaskItem{
			TaskID:                 task.TaskID,
			Subject:                task.Subject,
			GroupTitle:             task.GroupTitle,
			Title:                  task.Content,
			Content:                task.Content,
			Type:                   task.TaskType,
			Completed:              task.Completed,
			Status:                 task.Status,
			PointsValue:            1,
			ReferenceTitle:         task.ReferenceTitle,
			ReferenceAuthor:        task.ReferenceAuthor,
			ReferenceText:          task.ReferenceText,
			ReferenceSource:        task.ReferenceSource,
			HideReferenceFromChild: task.HideReferenceFromChild,
			AnalysisMode:           task.AnalysisMode,
		})
	}
	if len(existing) != len(boardItems) {
		return boardItems
	}

	merged := make([]taskboarddomain.TaskItem, 0, len(boardItems))
	for index := range boardItems {
		item := boardItems[index]
		stored := existing[index]
		item.Type = stored.Type
		item.Confidence = stored.Confidence
		item.NeedsReview = stored.NeedsReview
		item.Notes = append([]string(nil), stored.Notes...)
		item.ReferenceTitle = stored.ReferenceTitle
		item.ReferenceAuthor = stored.ReferenceAuthor
		item.ReferenceText = stored.ReferenceText
		item.ReferenceSource = stored.ReferenceSource
		item.HideReferenceFromChild = stored.HideReferenceFromChild
		item.AnalysisMode = stored.AnalysisMode
		if strings.TrimSpace(stored.Title) != "" {
			item.Title = strings.TrimSpace(stored.Title)
			item.Content = item.Title
		}
		if stored.PointsValue > 0 {
			item.PointsValue = stored.PointsValue
		}
		merged = append(merged, item)
	}
	return merged
}

func cloneWordItem(item taskboarddomain.WordItem) *taskboarddomain.WordItem {
	cloned := item
	return &cloned
}

func countCompletedTasks(tasks []taskboarddomain.Task) int {
	count := 0
	for _, task := range tasks {
		if task.Completed {
			count++
		}
	}
	return count
}

func buildEncouragement(period string, totals taskboarddomain.StatsTotals) string {
	switch {
	case totals.TotalTasks == 0 && totals.WordItems == 0:
		return "当前时间段还没有学习记录，先从一个小任务开始，慢慢进入状态。"
	case totals.CompletionRate >= 1:
		return fmt.Sprintf("%s的任务已经全部完成啦，你把坚持这件事做得很棒。", periodLabel(period))
	case totals.CompletionRate >= 0.7:
		return fmt.Sprintf("%s已经完成 %.0f%%，离收尾不远了，稳稳把最后几步走完。", periodLabel(period), totals.CompletionRate*100)
	case totals.CompletedTasks > 0:
		return fmt.Sprintf("%s已经完成 %d 项任务，每前进一步都算数，继续加油。", periodLabel(period), totals.CompletedTasks)
	default:
		return fmt.Sprintf("%s的挑战已经开始啦，先拿下一小步，我们慢慢来。", periodLabel(period))
	}
}

func periodLabel(period string) string {
	switch period {
	case "daily":
		return "今日"
	case "weekly":
		return "本周"
	case "monthly":
		return "本月"
	default:
		return "当前阶段"
	}
}

func roundRate(value float64) float64 {
	return float64(int(value*100+0.5)) / 100
}

func safeRate(numerator, denominator int) float64 {
	if denominator == 0 {
		return 0
	}
	return roundRate(float64(numerator) / float64(denominator))
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}
