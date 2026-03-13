package recitationanalysis

import (
	"context"
	"testing"

	"github.com/nikkofu/studyclaw/api-server/internal/platform/llm"
)

func TestAnalyzeFallsBackWithReferenceText(t *testing.T) {
	service := NewService(nil)

	analysis, err := service.Analyze(context.Background(), AnalyzeInput{
		Transcript: "读办将办独步寻花糖杜甫黄思帕钳将水东春光染会以微风桃花一处开无主可爱深红爱浅红",
		Scene:      "recitation",
		Locale:     "zh-CN",
		ReferenceText: "江畔独步寻花【唐】杜甫\n" +
			"黄师塔前江水东，春光懒困倚微风。\n" +
			"桃花一簇开无主，可爱深红爱浅红？",
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}

	if analysis.ParserMode != parserModeRuleFallback {
		t.Fatalf("expected rule_fallback, got %+v", analysis)
	}
	if analysis.ReferenceTitle != "江畔独步寻花" {
		t.Fatalf("expected reference title to be kept, got %+v", analysis)
	}
	if analysis.RecognizedTitle != "江畔独步寻花" {
		t.Fatalf("expected title to be recognized from noisy transcript, got %+v", analysis)
	}
	if len(analysis.MatchedLines) != 2 {
		t.Fatalf("expected 2 matched lines, got %+v", analysis.MatchedLines)
	}
	if analysis.MatchedLines[1].Status != lineStatusMatched {
		t.Fatalf("expected second line to be matched, got %+v", analysis.MatchedLines[1])
	}
	if analysis.CompletionRatio <= 0.45 {
		t.Fatalf("expected completion ratio to be usable, got %+v", analysis)
	}
	if !analysis.NeedsRetry {
		t.Fatalf("expected noisy recitation to still require retry, got %+v", analysis)
	}
}

func TestAnalyzeUsesLLMHybridWhenAvailable(t *testing.T) {
	service := NewService(stubRecitationLLMClient{
		response: `{
			"status": "success",
			"recognized_title": "江畔独步寻花",
			"recognized_author": "杜甫",
			"reference_title": "江畔独步寻花",
			"reference_author": "杜甫",
			"reference_text": "江畔独步寻花【唐】杜甫\n黄师塔前江水东，春光懒困倚微风。\n桃花一簇开无主，可爱深红爱浅红？",
			"reconstructed_text": "江畔独步寻花 杜甫 黄师塔前江水东 春光懒困倚微风 桃花一簇开无主 可爱深红爱浅红",
			"completion_ratio": 0.91,
			"needs_retry": false,
			"summary": "标题和正文整体对上了。",
			"suggestion": "可以尝试更流畅地再背一遍。",
			"issues": [],
			"matched_lines": [
				{
					"index": 1,
					"expected": "黄师塔前江水东，春光懒困倚微风。",
					"observed": "黄思帕钳将水东春光染会以微风",
					"match_ratio": 0.79,
					"status": "partial",
					"notes": "有少量同音字替换"
				},
				{
					"index": 2,
					"expected": "桃花一簇开无主，可爱深红爱浅红？",
					"observed": "桃花一处开无主可爱深红爱浅红",
					"match_ratio": 0.94,
					"status": "matched",
					"notes": "主体正确"
				}
			]
		}`,
	})

	analysis, err := service.Analyze(context.Background(), AnalyzeInput{
		Transcript: "读办将办独步寻花糖杜甫黄思帕钳将水东春光染会以微风桃花一处开无主可爱深红爱浅红",
		Scene:      "recitation",
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}

	if analysis.ParserMode != parserModeLLMHybrid {
		t.Fatalf("expected llm_hybrid, got %+v", analysis)
	}
	if analysis.RecognizedTitle != "江畔独步寻花" || analysis.RecognizedAuthor != "杜甫" {
		t.Fatalf("expected llm result to be kept, got %+v", analysis)
	}
	if len(analysis.MatchedLines) != 2 || analysis.MatchedLines[0].MatchRatio <= 0 {
		t.Fatalf("expected line details to be preserved, got %+v", analysis.MatchedLines)
	}
	if analysis.NeedsRetry {
		t.Fatalf("expected llm result to mark this as passable, got %+v", analysis)
	}
}

type stubRecitationLLMClient struct {
	response string
	err      error
}

func (s stubRecitationLLMClient) Generate(_ context.Context, _ llm.GenerateRequest) (string, error) {
	return s.response, s.err
}
