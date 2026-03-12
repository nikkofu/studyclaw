package voicecommand

import (
	"context"
	"testing"
)

func TestResolveFallsBackForDictationNext(t *testing.T) {
	service := NewService(nil)

	resolution, err := service.Resolve(context.Background(), ResolveInput{
		Transcript: "好了，下一个",
		Context: CommandContext{
			Surface: SurfaceDictation,
			Dictation: &DictationContext{
				SessionID:   "session_001",
				CanNext:     true,
				CanPrevious: false,
			},
		},
	})
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}

	if resolution.Action != ActionDictationNext {
		t.Fatalf("expected dictation next action, got %+v", resolution)
	}
	if resolution.Target.SessionID != "session_001" {
		t.Fatalf("expected session target to be kept, got %+v", resolution.Target)
	}
}

func TestResolveFallsBackForTaskBoardGroup(t *testing.T) {
	service := NewService(nil)

	resolution, err := service.Resolve(context.Background(), ResolveInput{
		Transcript: "一课一练做完了",
		Context: CommandContext{
			Surface: SurfaceTaskBoard,
			TaskBoard: &TaskBoardContext{
				Subjects: []TaskSubjectContext{
					{Subject: "数学", Pending: 2, Total: 2, Status: "pending"},
				},
				Groups: []TaskGroupContext{
					{Subject: "数学", GroupTitle: "一课一练", Pending: 2, Total: 2, Status: "pending"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}

	if resolution.Action != ActionTaskCompleteGroup {
		t.Fatalf("expected group action, got %+v", resolution)
	}
	if resolution.Target.GroupTitle != "一课一练" || resolution.Target.Subject != "数学" {
		t.Fatalf("unexpected target: %+v", resolution.Target)
	}
}
