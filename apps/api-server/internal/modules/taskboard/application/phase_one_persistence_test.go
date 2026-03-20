package application_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/nikkofu/studyclaw/api-server/internal/modules/taskboard/application"
	taskboarddomain "github.com/nikkofu/studyclaw/api-server/internal/modules/taskboard/domain"
	"github.com/nikkofu/studyclaw/api-server/internal/modules/taskboard/infrastructure/jsonstore"
)

func TestPersistenceStateTransitionRules(t *testing.T) {
	t.Parallel()

	if err := taskboarddomain.ValidatePersistenceTransition(taskboarddomain.PersistenceSessionStatusPreparing, taskboarddomain.PersistenceSessionStatusActive); err != nil {
		t.Fatalf("expected preparing -> active to be allowed, got error: %v", err)
	}

	err := taskboarddomain.ValidatePersistenceTransition(taskboarddomain.PersistenceSessionStatusActive, taskboarddomain.PersistenceSessionStatusCompleted)
	if !errors.Is(err, taskboarddomain.ErrInvalidPersistenceTransition) {
		t.Fatalf("expected ErrInvalidPersistenceTransition for active -> completed, got: %v", err)
	}

	err = taskboarddomain.ValidatePersistenceTransition(taskboarddomain.PersistenceSessionStatusPaused, taskboarddomain.PersistenceSessionStatusActive)
	if !errors.Is(err, taskboarddomain.ErrInvalidPersistenceTransition) {
		t.Fatalf("expected ErrInvalidPersistenceTransition for paused -> active, got: %v", err)
	}
}

func TestPersistenceAllowedTransitionsTable(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		from string
		to   string
	}{
		{name: "preparing_to_active", from: taskboarddomain.PersistenceSessionStatusPreparing, to: taskboarddomain.PersistenceSessionStatusActive},
		{name: "active_to_paused", from: taskboarddomain.PersistenceSessionStatusActive, to: taskboarddomain.PersistenceSessionStatusPaused},
		{name: "active_to_closing", from: taskboarddomain.PersistenceSessionStatusActive, to: taskboarddomain.PersistenceSessionStatusClosing},
		{name: "paused_to_resumed", from: taskboarddomain.PersistenceSessionStatusPaused, to: taskboarddomain.PersistenceSessionStatusResumed},
		{name: "resumed_to_active", from: taskboarddomain.PersistenceSessionStatusResumed, to: taskboarddomain.PersistenceSessionStatusActive},
		{name: "resumed_to_closing", from: taskboarddomain.PersistenceSessionStatusResumed, to: taskboarddomain.PersistenceSessionStatusClosing},
		{name: "closing_to_completed", from: taskboarddomain.PersistenceSessionStatusClosing, to: taskboarddomain.PersistenceSessionStatusCompleted},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if err := taskboarddomain.ValidatePersistenceTransition(tc.from, tc.to); err != nil {
				t.Fatalf("expected transition %s -> %s to be allowed, got %v", tc.from, tc.to, err)
			}
		})
	}
}

func TestPersistenceTerminalAndUnknownTransitionsRejected(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		from string
		to   string
	}{
		{name: "completed_to_active", from: taskboarddomain.PersistenceSessionStatusCompleted, to: taskboarddomain.PersistenceSessionStatusActive},
		{name: "aborted_to_preparing", from: taskboarddomain.PersistenceSessionStatusAborted, to: taskboarddomain.PersistenceSessionStatusPreparing},
		{name: "unknown_from", from: "unknown", to: taskboarddomain.PersistenceSessionStatusActive},
		{name: "unknown_to", from: taskboarddomain.PersistenceSessionStatusActive, to: "unknown"},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := taskboarddomain.ValidatePersistenceTransition(tc.from, tc.to)
			if !errors.Is(err, taskboarddomain.ErrInvalidPersistenceTransition) {
				t.Fatalf("expected ErrInvalidPersistenceTransition for %s -> %s, got %v", tc.from, tc.to, err)
			}
		})
	}
}

func TestEffectiveDurationExcludesLongSilenceSegments(t *testing.T) {
	t.Parallel()

	segments := []taskboarddomain.PersistenceDurationSegment{
		{Kind: taskboarddomain.PersistenceDurationSpeech, DurationSeconds: 15},
		{Kind: taskboarddomain.PersistenceDurationSilence, DurationSeconds: 25},
		{Kind: taskboarddomain.PersistenceDurationSpeech, DurationSeconds: 20},
	}

	got := taskboarddomain.CalculateEffectiveDurationSeconds(segments)
	if got != 35 {
		t.Fatalf("expected effective duration to be 35s, got %ds", got)
	}
}

func TestMakeupDoesNotAffectCoreStreakKPI(t *testing.T) {
	t.Parallel()

	days := []taskboarddomain.PersistenceDayRecord{
		{Completed: true},
		{Completed: true, Makeup: true},
		{Completed: true},
	}

	got := taskboarddomain.ComputePersistenceStreak(days)
	if got.DisplayStreak != 3 {
		t.Fatalf("expected display streak to include makeup day (3), got %d", got.DisplayStreak)
	}
	if got.CoreKPIStreak != 2 {
		t.Fatalf("expected core KPI streak to exclude makeup day (2), got %d", got.CoreKPIStreak)
	}
}

func TestStreakResetsOnMissedDay(t *testing.T) {
	t.Parallel()

	days := []taskboarddomain.PersistenceDayRecord{
		{Completed: true},
		{Completed: false},
		{Completed: true},
	}

	got := taskboarddomain.ComputePersistenceStreak(days)
	if got.DisplayStreak != 1 {
		t.Fatalf("expected display streak reset to 1 after missed day, got %d", got.DisplayStreak)
	}
	if got.CoreKPIStreak != 1 {
		t.Fatalf("expected core KPI streak reset to 1 after missed day, got %d", got.CoreKPIStreak)
	}
}

func TestSavePersistenceEvent_IsIdempotentByKey(t *testing.T) {
	t.Setenv("STUDYCLAW_DATA_DIR", t.TempDir())
	repo := jsonstore.NewRepository()
	service := application.NewPhaseOneService(nil, repo)
	baseDate := time.Date(2026, 3, 12, 0, 0, 0, 0, time.UTC)

	event := application.PersistenceEventRecord{
		SessionID:        "session-A",
		FamilyID:         306,
		ChildID:          1,
		AssignedDate:     "2026-03-12",
		EventType:        taskboarddomain.PersistenceEventStarted,
		IdempotencyKey:   "key-1",
		EffectiveSeconds: 10,
		TotalSeconds:     12,
		OccurredAt:       "2026-03-12T09:00:00Z",
	}

	firstSnapshot, firstCreated, err := service.SavePersistenceEvent(event)
	if err != nil {
		t.Fatalf("first SavePersistenceEvent returned error: %v", err)
	}
	if !firstCreated {
		t.Fatalf("expected first SavePersistenceEvent call to create event")
	}

	secondSnapshot, secondCreated, err := service.SavePersistenceEvent(event)
	if err != nil {
		t.Fatalf("second SavePersistenceEvent returned error: %v", err)
	}
	if secondCreated {
		t.Fatalf("expected second SavePersistenceEvent call to be idempotent")
	}

	events, err := repo.ListPersistenceEvents(306, 1, baseDate, baseDate)
	if err != nil {
		t.Fatalf("ListPersistenceEvents returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected one persisted event after duplicate writes, got %d", len(events))
	}
	if firstSnapshot != secondSnapshot {
		t.Fatalf("expected stable snapshot for duplicate idempotency key")
	}
}

func TestSavePersistenceEvent_AccumulatesSnapshotCounters(t *testing.T) {
	t.Setenv("STUDYCLAW_DATA_DIR", t.TempDir())
	repo := jsonstore.NewRepository()
	service := application.NewPhaseOneService(nil, repo)

	fixtures := []application.PersistenceEventRecord{
		{
			SessionID:        "session-accumulate",
			FamilyID:         306,
			ChildID:          1,
			AssignedDate:     "2026-03-12",
			EventType:        taskboarddomain.PersistenceEventStarted,
			IdempotencyKey:   "accumulate-1",
			EffectiveSeconds: 10,
			TotalSeconds:     12,
			OccurredAt:       "2026-03-12T09:00:00Z",
		},
		{
			SessionID:        "session-accumulate",
			FamilyID:         306,
			ChildID:          1,
			AssignedDate:     "2026-03-12",
			EventType:        taskboarddomain.PersistenceEventPaused,
			IdempotencyKey:   "accumulate-2",
			EffectiveSeconds: 3,
			TotalSeconds:     5,
			InvalidTrigger:   true,
			OccurredAt:       "2026-03-12T09:01:00Z",
		},
		{
			SessionID:        "session-accumulate",
			FamilyID:         306,
			ChildID:          1,
			AssignedDate:     "2026-03-12",
			EventType:        taskboarddomain.PersistenceEventResumed,
			IdempotencyKey:   "accumulate-3",
			EffectiveSeconds: 7,
			TotalSeconds:     9,
			OccurredAt:       "2026-03-12T09:02:00Z",
		},
		{
			SessionID:        "session-accumulate",
			FamilyID:         306,
			ChildID:          1,
			AssignedDate:     "2026-03-12",
			EventType:        taskboarddomain.PersistenceEventStarted,
			IdempotencyKey:   "accumulate-4",
			EffectiveSeconds: 2,
			TotalSeconds:     4,
			OccurredAt:       "2026-03-12T09:03:00Z",
		},
	}

	var snapshot application.PersistenceSessionSnapshot
	for _, event := range fixtures {
		var err error
		snapshot, _, err = service.SavePersistenceEvent(event)
		if err != nil {
			t.Fatalf("SavePersistenceEvent returned error: %v", err)
		}
	}

	if snapshot.EffectiveSeconds != 22 {
		t.Fatalf("expected accumulated effective seconds 22, got %d", snapshot.EffectiveSeconds)
	}
	if snapshot.TotalSeconds != 30 {
		t.Fatalf("expected accumulated total seconds 30, got %d", snapshot.TotalSeconds)
	}
	if snapshot.InvalidTriggers != 1 {
		t.Fatalf("expected invalid trigger count 1, got %d", snapshot.InvalidTriggers)
	}
}

func TestSavePersistenceEvent_RejectsInvalidStateTransition(t *testing.T) {
	t.Setenv("STUDYCLAW_DATA_DIR", t.TempDir())
	repo := jsonstore.NewRepository()
	service := application.NewPhaseOneService(nil, repo)

	_, _, err := service.SavePersistenceEvent(application.PersistenceEventRecord{
		SessionID:      "session-transition",
		FamilyID:       306,
		ChildID:        1,
		AssignedDate:   "2026-03-12",
		EventType:      taskboarddomain.PersistenceEventStarted,
		IdempotencyKey: "transition-1",
		OccurredAt:     "2026-03-12T09:00:00Z",
	})
	if err != nil {
		t.Fatalf("failed to save started event: %v", err)
	}

	_, _, err = service.SavePersistenceEvent(application.PersistenceEventRecord{
		SessionID:      "session-transition",
		FamilyID:       306,
		ChildID:        1,
		AssignedDate:   "2026-03-12",
		EventType:      taskboarddomain.PersistenceEventCompleted,
		IdempotencyKey: "transition-2",
		OccurredAt:     "2026-03-12T09:01:00Z",
	})
	if !errors.Is(err, taskboarddomain.ErrInvalidPersistenceTransition) {
		t.Fatalf("expected ErrInvalidPersistenceTransition, got %v", err)
	}
}

func TestAggregatePersistenceSummary_ComputesCompletionRateAndEffectiveDuration(t *testing.T) {
	t.Setenv("STUDYCLAW_DATA_DIR", t.TempDir())
	repo := jsonstore.NewRepository()
	service := application.NewPhaseOneService(nil, repo)
	baseDate := time.Date(2026, 3, 12, 0, 0, 0, 0, time.UTC)

	for i := 0; i < 10; i++ {
		sessionID := fmt.Sprintf("session-%d", i)
		_, _, err := service.SavePersistenceEvent(application.PersistenceEventRecord{
			SessionID:        sessionID,
			FamilyID:         306,
			ChildID:          1,
			AssignedDate:     "2026-03-12",
			EventType:        taskboarddomain.PersistenceEventStarted,
			IdempotencyKey:   fmt.Sprintf("start-%d", i),
			EffectiveSeconds: 20,
			TotalSeconds:     30,
			OccurredAt:       "2026-03-12T09:00:00Z",
		})
		if err != nil {
			t.Fatalf("failed to save started event: %v", err)
		}
		if i < 7 {
			_, _, err = service.SavePersistenceEvent(application.PersistenceEventRecord{
				SessionID:      sessionID,
				FamilyID:       306,
				ChildID:        1,
				AssignedDate:   "2026-03-12",
				EventType:      taskboarddomain.PersistenceEventPaused,
				IdempotencyKey: fmt.Sprintf("paused-%d", i),
				OccurredAt:     "2026-03-12T09:05:00Z",
			})
			if err != nil {
				t.Fatalf("failed to save paused event: %v", err)
			}
			_, _, err = service.SavePersistenceEvent(application.PersistenceEventRecord{
				SessionID:      sessionID,
				FamilyID:       306,
				ChildID:        1,
				AssignedDate:   "2026-03-12",
				EventType:      taskboarddomain.PersistenceEventResumed,
				IdempotencyKey: fmt.Sprintf("resumed-%d", i),
				OccurredAt:     "2026-03-12T09:06:00Z",
			})
			if err != nil {
				t.Fatalf("failed to save resumed event: %v", err)
			}
			_, _, err = service.SavePersistenceEvent(application.PersistenceEventRecord{
				SessionID:      sessionID,
				FamilyID:       306,
				ChildID:        1,
				AssignedDate:   "2026-03-12",
				EventType:      taskboarddomain.PersistenceEventStarted,
				IdempotencyKey: fmt.Sprintf("active-again-%d", i),
				OccurredAt:     "2026-03-12T09:08:00Z",
			})
			if err != nil {
				t.Fatalf("failed to save active-again event: %v", err)
			}
			_, _, err = service.SavePersistenceEvent(application.PersistenceEventRecord{
				SessionID:      sessionID,
				FamilyID:       306,
				ChildID:        1,
				AssignedDate:   "2026-03-12",
				Status:         taskboarddomain.PersistenceSessionStatusClosing,
				IdempotencyKey: fmt.Sprintf("closing-%d", i),
				OccurredAt:     "2026-03-12T09:09:30Z",
			})
			if err != nil {
				t.Fatalf("failed to save closing event: %v", err)
			}
			_, _, err = service.SavePersistenceEvent(application.PersistenceEventRecord{
				SessionID:         sessionID,
				FamilyID:          306,
				ChildID:           1,
				AssignedDate:      "2026-03-12",
				EventType:         taskboarddomain.PersistenceEventCompleted,
				IdempotencyKey:    fmt.Sprintf("completed-%d", i),
				EffectiveSeconds:  20,
				TotalSeconds:      30,
				OccurredAt:        "2026-03-12T09:10:00Z",
			})
			if err != nil {
				t.Fatalf("failed to save completed event: %v", err)
			}
		}
	}

	summary, err := service.AggregatePersistenceSummary(306, 1, baseDate, baseDate)
	if err != nil {
		t.Fatalf("AggregatePersistenceSummary returned error: %v", err)
	}
	if summary.CompletionRate.Completed != 7 || summary.CompletionRate.Total != 10 {
		t.Fatalf("expected completion counters 7/10, got %d/%d", summary.CompletionRate.Completed, summary.CompletionRate.Total)
	}
	if summary.CompletionRate.Rate != 0.7 {
		t.Fatalf("expected completion rate 0.7, got %v", summary.CompletionRate.Rate)
	}
	if summary.EffectiveDuration.EffectiveSeconds != 340 {
		t.Fatalf("expected effective seconds 340, got %d", summary.EffectiveDuration.EffectiveSeconds)
	}
	if summary.EffectiveDuration.TotalSeconds != 510 {
		t.Fatalf("expected total seconds 510, got %d", summary.EffectiveDuration.TotalSeconds)
	}
}

func TestPromptGuardrailCounters(t *testing.T) {
	t.Setenv("STUDYCLAW_DATA_DIR", t.TempDir())
	repo := jsonstore.NewRepository()
	service := application.NewPhaseOneService(nil, repo)
	baseDate := time.Date(2026, 3, 12, 0, 0, 0, 0, time.UTC)

	fixtures := []bool{true, true, false, false}
	for i, invalid := range fixtures {
		_, _, err := service.SavePersistenceEvent(application.PersistenceEventRecord{
			SessionID:        "guard-session",
			FamilyID:         306,
			ChildID:          1,
			AssignedDate:     "2026-03-12",
			EventType:        taskboarddomain.PersistenceEventInterrupted,
			IdempotencyKey:   fmt.Sprintf("guard-%d", i),
			EffectiveSeconds: 5,
			TotalSeconds:     10,
			InvalidTrigger:   invalid,
			OccurredAt:       "2026-03-12T09:20:00Z",
		})
		if err != nil {
			t.Fatalf("failed to save guardrail event: %v", err)
		}
	}

	summary, err := service.AggregatePersistenceSummary(306, 1, baseDate, baseDate)
	if err != nil {
		t.Fatalf("AggregatePersistenceSummary returned error: %v", err)
	}
	if summary.Guardrails.InvalidTriggerRate != 0.5 {
		t.Fatalf("expected invalid trigger rate 0.5, got %v", summary.Guardrails.InvalidTriggerRate)
	}
	if summary.Guardrails.InvalidTriggerRate < 0 || summary.Guardrails.InvalidTriggerRate > 1 {
		t.Fatalf("invalid trigger rate out of bounds: %v", summary.Guardrails.InvalidTriggerRate)
	}
}
