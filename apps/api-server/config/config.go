package config

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/viper"
)

func LoadConfig() {
	loadedFiles, err := loadRuntimeEnvFiles(findRepoEnvPath())
	if err != nil {
		log.Fatalf("Failed to load runtime environment: %v", err)
	}

	if len(loadedFiles) == 0 {
		log.Println("No runtime env file found; relying on process environment variables and built-in defaults")
	} else {
		for _, path := range loadedFiles {
			log.Printf("Loaded runtime environment from %s", path)
		}
	}

	viper.SetDefault("DB_HOST", "127.0.0.1")
	viper.SetDefault("DB_PORT", "3306")
	viper.SetDefault("DB_USER", "root")
	viper.SetDefault("DB_PASSWORD", "studyclaw_dev_secret")
	viper.SetDefault("DB_NAME", "studyclaw_dev")
	viper.SetDefault("API_PORT", "8080")
}

func loadRuntimeEnvFiles(repoEnvPath string) ([]string, error) {
	loadedPaths := make([]string, 0)

	for _, candidate := range resolveEnvFileCandidates(repoEnvPath) {
		loaded, err := loadEnvFile(candidate)
		if err != nil {
			return loadedPaths, err
		}
		if loaded {
			loadedPaths = append(loadedPaths, candidate)
		}
	}

	return loadedPaths, nil
}

func resolveEnvFileCandidates(repoEnvPath string) []string {
	candidates := make([]string, 0, 3)

	addCandidate := func(path string) {
		path = strings.TrimSpace(path)
		if path == "" {
			return
		}

		for _, existing := range candidates {
			if existing == path {
				return
			}
		}
		candidates = append(candidates, path)
	}

	if explicitEnvFile := strings.TrimSpace(os.Getenv("STUDYCLAW_ENV_FILE")); explicitEnvFile != "" {
		addCandidate(expandUserPath(explicitEnvFile))
	}

	configDir := strings.TrimSpace(os.Getenv("STUDYCLAW_CONFIG_DIR"))
	if configDir == "" {
		configDir = defaultConfigDir()
	}
	if configDir != "" {
		addCandidate(filepath.Join(expandUserPath(configDir), "runtime.env"))
	}

	addCandidate(repoEnvPath)
	return candidates
}

func defaultConfigDir() string {
	if xdgConfigHome := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME")); xdgConfigHome != "" {
		return filepath.Join(expandUserPath(xdgConfigHome), "studyclaw")
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(homeDir, ".config", "studyclaw")
}

func expandUserPath(path string) string {
	if path == "~" {
		if homeDir, err := os.UserHomeDir(); err == nil {
			return homeDir
		}
		return path
	}

	if !strings.HasPrefix(path, "~/") {
		return path
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(homeDir, strings.TrimPrefix(path, "~/"))
}

func findRepoEnvPath() string {
	workingDir, err := os.Getwd()
	if err != nil {
		return filepath.Clean("../../.env")
	}

	currentDir := workingDir
	for {
		envExamplePath := filepath.Join(currentDir, ".env.example")
		gitDirPath := filepath.Join(currentDir, ".git")
		if _, err := os.Stat(envExamplePath); err == nil {
			if _, gitErr := os.Stat(gitDirPath); gitErr == nil {
				return filepath.Join(currentDir, ".env")
			}
		}

		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			break
		}
		currentDir = parentDir
	}

	return filepath.Clean("../../.env")
}

func loadEnvFile(path string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("open env file %s: %w", path, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++

		key, value, ok, err := parseEnvLine(scanner.Text())
		if err != nil {
			return false, fmt.Errorf("parse env file %s line %d: %w", path, lineNumber, err)
		}
		if !ok {
			continue
		}
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		if err := os.Setenv(key, value); err != nil {
			return false, fmt.Errorf("set env %s from %s: %w", key, path, err)
		}
	}

	if err := scanner.Err(); err != nil {
		return false, fmt.Errorf("scan env file %s: %w", path, err)
	}

	return true, nil
}

func parseEnvLine(line string) (string, string, bool, error) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return "", "", false, nil
	}

	if strings.HasPrefix(trimmed, "export ") {
		trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "export "))
	}

	parts := strings.SplitN(trimmed, "=", 2)
	if len(parts) != 2 {
		return "", "", false, fmt.Errorf("invalid env line %q", line)
	}

	key := strings.TrimSpace(parts[0])
	if key == "" {
		return "", "", false, fmt.Errorf("empty env key")
	}

	value := strings.TrimSpace(parts[1])
	if len(value) >= 2 && strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
		unquoted, err := strconv.Unquote(value)
		if err != nil {
			return "", "", false, fmt.Errorf("invalid quoted value for %s", key)
		}
		value = unquoted
	} else if len(value) >= 2 && strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'") {
		value = value[1 : len(value)-1]
	}

	return key, value, true, nil
}

func GetEnv(key string) string {
	// Let OS env override viper config
	if val := os.Getenv(key); val != "" {
		return val
	}
	return viper.GetString(key)
}
