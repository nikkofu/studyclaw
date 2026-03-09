package weeklyinsights

import (
	"testing"

	taskboarddomain "github.com/nikkofu/studyclaw/api-server/internal/modules/taskboard/domain"
)

func TestGenerateMockCountsTypedDomainTasks(t *testing.T) {
	daysData := []map[string]interface{}{
		{
			"date": "2026-03-06",
			"tasks": []taskboarddomain.Task{
				{TaskID: 1, Subject: "数学", Content: "口算", Completed: true},
				{TaskID: 2, Subject: "英语", Content: "听写", Completed: false},
				{TaskID: 3, Subject: "语文", Content: "背诵", Completed: true},
			},
		},
	}

	insight := generateMock(daysData)

	if insight.RawMetricTotal != 3 {
		t.Fatalf("expected total 3, got %d", insight.RawMetricTotal)
	}
	if insight.RawMetricCompleted != 2 {
		t.Fatalf("expected completed 2, got %d", insight.RawMetricCompleted)
	}
	if insight.AgenticPattern.Primary != "single-agent system" {
		t.Fatalf("unexpected primary pattern: %+v", insight.AgenticPattern)
	}
}
