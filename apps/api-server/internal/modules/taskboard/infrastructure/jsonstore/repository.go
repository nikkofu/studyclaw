package jsonstore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	taskboarddomain "github.com/nikkofu/studyclaw/api-server/internal/modules/taskboard/domain"
)

type Repository struct {
	mu sync.Mutex
}

type persistedState struct {
	NextDraftSeq           int64                                               `json:"next_draft_seq"`
	NextAssignSeq          int64                                               `json:"next_assignment_seq"`
	NextPointsSeq          int64                                               `json:"next_points_seq"`
	NextWordListSeq        int64                                               `json:"next_word_list_seq"`
	NextSessionSeq         int64                                               `json:"next_session_seq"`
	NextVoiceSessionSeq    int64                                               `json:"next_voice_session_seq"`
	Drafts                 map[string]taskboarddomain.DailyAssignmentDraft     `json:"drafts"`
	Assignments            map[string]taskboarddomain.PublishedDailyAssignment `json:"assignments"`
	ManualPoints           map[string][]taskboarddomain.PointsLedgerEntry      `json:"manual_points"`
	WordLists              map[string]taskboarddomain.WordList                 `json:"word_lists"`
	DictationSessions      map[string]taskboarddomain.DictationSession         `json:"dictation_sessions"`
	VoiceLearningSessions  map[string]taskboarddomain.VoiceLearningSession     `json:"voice_learning_sessions"`
}

func NewRepository() *Repository {
	return &Repository{}
}

func (r *Repository) SaveDraft(draft taskboarddomain.DailyAssignmentDraft) (taskboarddomain.DailyAssignmentDraft, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	state, err := r.loadState()
	if err != nil {
		return taskboarddomain.DailyAssignmentDraft{}, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if strings.TrimSpace(draft.DraftID) == "" {
		state.NextDraftSeq++
		draft.DraftID = fmt.Sprintf("draft_%06d", state.NextDraftSeq)
		draft.CreatedAt = now
	}
	if strings.TrimSpace(draft.CreatedAt) == "" {
		draft.CreatedAt = now
	}
	draft.UpdatedAt = now
	state.Drafts[draft.DraftID] = draft

	if err := r.saveState(state); err != nil {
		return taskboarddomain.DailyAssignmentDraft{}, err
	}

	return draft, nil
}

func (r *Repository) GetDraft(draftID string) (taskboarddomain.DailyAssignmentDraft, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	state, err := r.loadState()
	if err != nil {
		return taskboarddomain.DailyAssignmentDraft{}, false, err
	}

	draft, ok := state.Drafts[strings.TrimSpace(draftID)]
	return draft, ok, nil
}

func (r *Repository) SavePublishedAssignment(assignment taskboarddomain.PublishedDailyAssignment) (taskboarddomain.PublishedDailyAssignment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	state, err := r.loadState()
	if err != nil {
		return taskboarddomain.PublishedDailyAssignment{}, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if strings.TrimSpace(assignment.AssignmentID) == "" {
		state.NextAssignSeq++
		assignment.AssignmentID = fmt.Sprintf("assignment_%06d", state.NextAssignSeq)
		assignment.PublishedAt = now
	}
	if strings.TrimSpace(assignment.PublishedAt) == "" {
		assignment.PublishedAt = now
	}
	assignment.UpdatedAt = now
	state.Assignments[assignmentKey(assignment.FamilyID, assignment.ChildID, assignment.AssignedDate)] = assignment

	if err := r.saveState(state); err != nil {
		return taskboarddomain.PublishedDailyAssignment{}, err
	}

	return assignment, nil
}

func (r *Repository) GetPublishedAssignment(familyID, childID uint, date time.Time) (taskboarddomain.PublishedDailyAssignment, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	state, err := r.loadState()
	if err != nil {
		return taskboarddomain.PublishedDailyAssignment{}, false, err
	}

	assignment, ok := state.Assignments[assignmentKey(familyID, childID, date.Format("2006-01-02"))]
	return assignment, ok, nil
}

func (r *Repository) ListPublishedAssignments(familyID, childID uint, startDate, endDate time.Time) ([]taskboarddomain.PublishedDailyAssignment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	state, err := r.loadState()
	if err != nil {
		return nil, err
	}

	assignments := make([]taskboarddomain.PublishedDailyAssignment, 0)
	for _, assignment := range state.Assignments {
		if assignment.FamilyID != familyID || assignment.ChildID != childID {
			continue
		}
		if !dateInRange(assignment.AssignedDate, startDate, endDate) {
			continue
		}
		assignments = append(assignments, assignment)
	}

	sort.Slice(assignments, func(i, j int) bool {
		if assignments[i].AssignedDate == assignments[j].AssignedDate {
			return assignments[i].AssignmentID < assignments[j].AssignmentID
		}
		return assignments[i].AssignedDate < assignments[j].AssignedDate
	})
	return assignments, nil
}

func (r *Repository) SaveManualPointsEntry(entry taskboarddomain.PointsLedgerEntry) (taskboarddomain.PointsLedgerEntry, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	state, err := r.loadState()
	if err != nil {
		return taskboarddomain.PointsLedgerEntry{}, err
	}

	if strings.TrimSpace(entry.EntryID) == "" {
		state.NextPointsSeq++
		entry.EntryID = fmt.Sprintf("points_%06d", state.NextPointsSeq)
	}

	key := pointsKey(entry.FamilyID, entry.UserID)
	state.ManualPoints[key] = append(state.ManualPoints[key], entry)
	sort.Slice(state.ManualPoints[key], func(i, j int) bool {
		left := state.ManualPoints[key][i]
		right := state.ManualPoints[key][j]
		if left.OccurredOn == right.OccurredOn {
			return left.EntryID < right.EntryID
		}
		return left.OccurredOn < right.OccurredOn
	})

	if err := r.saveState(state); err != nil {
		return taskboarddomain.PointsLedgerEntry{}, err
	}

	return entry, nil
}

func (r *Repository) ListManualPointsEntries(familyID, userID uint) ([]taskboarddomain.PointsLedgerEntry, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	state, err := r.loadState()
	if err != nil {
		return nil, err
	}

	entries := append([]taskboarddomain.PointsLedgerEntry(nil), state.ManualPoints[pointsKey(familyID, userID)]...)
	return entries, nil
}

func (r *Repository) SaveWordList(list taskboarddomain.WordList) (taskboarddomain.WordList, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	state, err := r.loadState()
	if err != nil {
		return taskboarddomain.WordList{}, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if strings.TrimSpace(list.WordListID) == "" {
		state.NextWordListSeq++
		list.WordListID = fmt.Sprintf("wordlist_%06d", state.NextWordListSeq)
		list.CreatedAt = now
	}
	if strings.TrimSpace(list.CreatedAt) == "" {
		list.CreatedAt = now
	}
	list.UpdatedAt = now
	state.WordLists[wordListKey(list.FamilyID, list.ChildID, list.AssignedDate)] = list

	if err := r.saveState(state); err != nil {
		return taskboarddomain.WordList{}, err
	}

	return list, nil
}

func (r *Repository) GetWordList(familyID, childID uint, date time.Time) (taskboarddomain.WordList, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	state, err := r.loadState()
	if err != nil {
		return taskboarddomain.WordList{}, false, err
	}

	list, ok := state.WordLists[wordListKey(familyID, childID, date.Format("2006-01-02"))]
	return list, ok, nil
}

func (r *Repository) ListWordLists(familyID, childID uint, startDate, endDate time.Time) ([]taskboarddomain.WordList, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	state, err := r.loadState()
	if err != nil {
		return nil, err
	}

	lists := make([]taskboarddomain.WordList, 0)
	for _, list := range state.WordLists {
		if list.FamilyID != familyID || list.ChildID != childID {
			continue
		}
		if !dateInRange(list.AssignedDate, startDate, endDate) {
			continue
		}
		lists = append(lists, list)
	}

	sort.Slice(lists, func(i, j int) bool {
		if lists[i].AssignedDate == lists[j].AssignedDate {
			return lists[i].WordListID < lists[j].WordListID
		}
		return lists[i].AssignedDate < lists[j].AssignedDate
	})
	return lists, nil
}

func (r *Repository) SaveDictationSession(session taskboarddomain.DictationSession) (taskboarddomain.DictationSession, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	state, err := r.loadState()
	if err != nil {
		return taskboarddomain.DictationSession{}, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if strings.TrimSpace(session.SessionID) == "" {
		state.NextSessionSeq++
		session.SessionID = fmt.Sprintf("session_%06d", state.NextSessionSeq)
		session.StartedAt = now
	}
	if strings.TrimSpace(session.StartedAt) == "" {
		session.StartedAt = now
	}
	session.UpdatedAt = now
	state.DictationSessions[session.SessionID] = session

	if err := r.saveState(state); err != nil {
		return taskboarddomain.DictationSession{}, err
	}

	return session, nil
}

func (r *Repository) GetDictationSession(sessionID string) (taskboarddomain.DictationSession, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	state, err := r.loadState()
	if err != nil {
		return taskboarddomain.DictationSession{}, false, err
	}

	session, ok := state.DictationSessions[strings.TrimSpace(sessionID)]
	return session, ok, nil
}

func (r *Repository) ListDictationSessions(familyID, childID uint, startDate, endDate time.Time) ([]taskboarddomain.DictationSession, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	state, err := r.loadState()
	if err != nil {
		return nil, err
	}

	sessions := make([]taskboarddomain.DictationSession, 0)
	for _, session := range state.DictationSessions {
		if session.FamilyID != familyID || session.ChildID != childID {
			continue
		}
		if !dateInRange(session.AssignedDate, startDate, endDate) {
			continue
		}
		sessions = append(sessions, session)
	}

	sort.Slice(sessions, func(i, j int) bool {
		if sessions[i].AssignedDate == sessions[j].AssignedDate {
			return sessions[i].SessionID < sessions[j].SessionID
		}
		return sessions[i].AssignedDate < sessions[j].AssignedDate
	})
	return sessions, nil
}

func (r *Repository) SaveVoiceLearningSession(session taskboarddomain.VoiceLearningSession) (taskboarddomain.VoiceLearningSession, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	state, err := r.loadState()
	if err != nil {
		return taskboarddomain.VoiceLearningSession{}, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if strings.TrimSpace(session.SessionID) == "" {
		state.NextVoiceSessionSeq++
		session.SessionID = fmt.Sprintf("voice_session_%06d", state.NextVoiceSessionSeq)
		session.CreatedAt = now
	}
	if strings.TrimSpace(session.CreatedAt) == "" {
		session.CreatedAt = now
	}
	session.UpdatedAt = now
	state.VoiceLearningSessions[session.SessionID] = session

	if err := r.saveState(state); err != nil {
		return taskboarddomain.VoiceLearningSession{}, err
	}

	return session, nil
}

func (r *Repository) ListVoiceLearningSessions(familyID, childID uint, startDate, endDate time.Time) ([]taskboarddomain.VoiceLearningSession, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	state, err := r.loadState()
	if err != nil {
		return nil, err
	}

	sessions := make([]taskboarddomain.VoiceLearningSession, 0)
	for _, session := range state.VoiceLearningSessions {
		if session.FamilyID != familyID || session.ChildID != childID {
			continue
		}
		if !dateInRange(session.AssignedDate, startDate, endDate) {
			continue
		}
		sessions = append(sessions, session)
	}

	sort.Slice(sessions, func(i, j int) bool {
		leftTime := sessions[i].EndedAt
		if strings.TrimSpace(leftTime) == "" {
			leftTime = sessions[i].UpdatedAt
		}
		rightTime := sessions[j].EndedAt
		if strings.TrimSpace(rightTime) == "" {
			rightTime = sessions[j].UpdatedAt
		}
		if leftTime == rightTime {
			return sessions[i].SessionID < sessions[j].SessionID
		}
		return leftTime > rightTime
	})
	return sessions, nil
}

func (r *Repository) loadState() (*persistedState, error) {
	path := stateFilePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			state := defaultState()
			return &state, nil
		}
		return nil, err
	}

	state := defaultState()
	if err := json.Unmarshal(content, &state); err != nil {
		return nil, err
	}
	if state.Drafts == nil {
		state.Drafts = make(map[string]taskboarddomain.DailyAssignmentDraft)
	}
	if state.Assignments == nil {
		state.Assignments = make(map[string]taskboarddomain.PublishedDailyAssignment)
	}
	if state.ManualPoints == nil {
		state.ManualPoints = make(map[string][]taskboarddomain.PointsLedgerEntry)
	}
	if state.WordLists == nil {
		state.WordLists = make(map[string]taskboarddomain.WordList)
	}
	if state.DictationSessions == nil {
		state.DictationSessions = make(map[string]taskboarddomain.DictationSession)
	}
	if state.VoiceLearningSessions == nil {
		state.VoiceLearningSessions = make(map[string]taskboarddomain.VoiceLearningSession)
	}
	return &state, nil
}

func (r *Repository) saveState(state *persistedState) error {
	path := stateFilePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	content, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, content, 0o644)
}

func defaultState() persistedState {
	return persistedState{
		Drafts:                make(map[string]taskboarddomain.DailyAssignmentDraft),
		Assignments:           make(map[string]taskboarddomain.PublishedDailyAssignment),
		ManualPoints:          make(map[string][]taskboarddomain.PointsLedgerEntry),
		WordLists:             make(map[string]taskboarddomain.WordList),
		DictationSessions:     make(map[string]taskboarddomain.DictationSession),
		VoiceLearningSessions: make(map[string]taskboarddomain.VoiceLearningSession),
	}
}

func getDataRoot() string {
	if root := os.Getenv("STUDYCLAW_DATA_DIR"); strings.TrimSpace(root) != "" {
		return strings.TrimSpace(root)
	}

	cwd, _ := os.Getwd()
	return filepath.Join(cwd, "..", "..", "data")
}

func stateFilePath() string {
	return filepath.Join(getDataRoot(), "phase1", "taskboard_state.json")
}

func assignmentKey(familyID, childID uint, date string) string {
	return fmt.Sprintf("%d:%d:%s", familyID, childID, strings.TrimSpace(date))
}

func pointsKey(familyID, userID uint) string {
	return fmt.Sprintf("%d:%d", familyID, userID)
}

func wordListKey(familyID, childID uint, date string) string {
	return fmt.Sprintf("%d:%d:%s", familyID, childID, strings.TrimSpace(date))
}

func dateInRange(rawDate string, startDate, endDate time.Time) bool {
	date, err := time.Parse("2006-01-02", strings.TrimSpace(rawDate))
	if err != nil {
		return false
	}
	if date.Before(startDate) {
		return false
	}
	if date.After(endDate) {
		return false
	}
	return true
}
