package application

import (
	"strings"
	"testing"

	taskboarddomain "github.com/nikkofu/studyclaw/api-server/internal/modules/taskboard/domain"
)

func TestBuildEncouragement(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		period       string
		totals       taskboarddomain.StatsTotals
		wantContains []string
	}{
		{
			name:   "no_data_yet",
			period: "daily",
			totals: taskboarddomain.StatsTotals{},
			wantContains: []string{
				"还没有学习记录",
				"小任务",
			},
		},
		{
			name:   "all_completed",
			period: "weekly",
			totals: taskboarddomain.StatsTotals{
				TotalTasks:     6,
				CompletedTasks: 6,
				CompletionRate: 1,
			},
			wantContains: []string{
				"本周",
				"全部完成",
				"很棒",
			},
		},
		{
			name:   "high_completion_rate",
			period: "monthly",
			totals: taskboarddomain.StatsTotals{
				TotalTasks:     10,
				CompletedTasks: 8,
				PendingTasks:   2,
				CompletionRate: 0.8,
			},
			wantContains: []string{
				"本月",
				"80%",
				"稳稳",
			},
		},
		{
			name:   "partial_progress",
			period: "daily",
			totals: taskboarddomain.StatsTotals{
				TotalTasks:     5,
				CompletedTasks: 2,
				PendingTasks:   3,
				CompletionRate: 0.4,
			},
			wantContains: []string{
				"今日",
				"2 项任务",
				"继续加油",
			},
		},
		{
			name:   "started_but_not_completed",
			period: "daily",
			totals: taskboarddomain.StatsTotals{
				TotalTasks:     4,
				CompletedTasks: 0,
				PendingTasks:   4,
				CompletionRate: 0,
			},
			wantContains: []string{
				"今日",
				"挑战已经开始",
				"慢慢来",
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := buildEncouragement(tc.period, tc.totals)
			for _, want := range tc.wantContains {
				if !strings.Contains(got, want) {
					t.Fatalf("expected %q to contain %q", got, want)
				}
			}
		})
	}
}
