package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveEnvFileCandidatesPrefersPrivateRuntimeEnv(t *testing.T) {
	t.Setenv("HOME", "/tmp/studyclaw-home")
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("STUDYCLAW_ENV_FILE", "")
	t.Setenv("STUDYCLAW_CONFIG_DIR", "")

	candidates := resolveEnvFileCandidates("/repo/.env")
	if len(candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(candidates))
	}

	expectedRuntimeEnv := filepath.Join("/tmp/studyclaw-home", ".config", "studyclaw", "runtime.env")
	if candidates[0] != expectedRuntimeEnv {
		t.Fatalf("expected first candidate %s, got %s", expectedRuntimeEnv, candidates[0])
	}
	if candidates[1] != "/repo/.env" {
		t.Fatalf("expected repo env fallback, got %s", candidates[1])
	}
}

func TestLoadEnvFileKeepsProcessEnvValues(t *testing.T) {
	tempDir := t.TempDir()
	envFile := filepath.Join(tempDir, "runtime.env")
	content := "LLM_API_KEY=file-key\nAPI_PORT=9000\n"
	if err := os.WriteFile(envFile, []byte(content), 0o600); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	t.Setenv("LLM_API_KEY", "process-key")
	loaded, err := loadEnvFile(envFile)
	if err != nil {
		t.Fatalf("load env file: %v", err)
	}
	if !loaded {
		t.Fatal("expected env file to be loaded")
	}

	if got := os.Getenv("LLM_API_KEY"); got != "process-key" {
		t.Fatalf("expected process env to win, got %s", got)
	}
	if got := os.Getenv("API_PORT"); got != "9000" {
		t.Fatalf("expected API_PORT from file, got %s", got)
	}
}

func TestParseEnvLineSupportsQuotedValues(t *testing.T) {
	key, value, ok, err := parseEnvLine(`LLM_MODEL_NAME="doubao-pro-32k"`)
	if err != nil {
		t.Fatalf("parse env line: %v", err)
	}
	if !ok {
		t.Fatal("expected env line to be parsed")
	}
	if key != "LLM_MODEL_NAME" || value != "doubao-pro-32k" {
		t.Fatalf("unexpected parsed result: %s=%s", key, value)
	}
}
