package llm

import "testing"

func TestGetBaseURLDefaultsToArk(t *testing.T) {
	t.Setenv("LLM_BASE_URL", "")

	if got := getBaseURL(); got != defaultArkBaseURL {
		t.Fatalf("expected default ark base url %s, got %s", defaultArkBaseURL, got)
	}
}

func TestGetBaseURLTrimsTrailingSlash(t *testing.T) {
	t.Setenv("LLM_BASE_URL", "https://ark.cn-beijing.volces.com/api/v3/")

	if got := getBaseURL(); got != defaultArkBaseURL {
		t.Fatalf("expected trimmed base url %s, got %s", defaultArkBaseURL, got)
	}
}

func TestGetModelNameUsesSharedModelNameWhenSpecificMissing(t *testing.T) {
	t.Setenv("LLM_MODEL_NAME", "shared-model")
	t.Setenv("LLM_PARSER_MODEL_NAME", "")

	if got := getModelName("LLM_PARSER_MODEL_NAME", ""); got != "shared-model" {
		t.Fatalf("expected shared model name, got %s", got)
	}
}

func TestGetModelNamePrefersSpecificOverride(t *testing.T) {
	t.Setenv("LLM_MODEL_NAME", "shared-model")
	t.Setenv("LLM_WEEKLY_MODEL_NAME", "weekly-model")

	if got := getModelName("LLM_WEEKLY_MODEL_NAME", ""); got != "weekly-model" {
		t.Fatalf("expected specific model override, got %s", got)
	}
}
