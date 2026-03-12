package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nikkofu/studyclaw/api-server/config"
)

type dailyFileWriter struct {
	mu          sync.Mutex
	dir         string
	filePrefix  string
	currentDate string
	file        *os.File
}

func Init() error {
	logDir := resolveLogDir()
	writer := &dailyFileWriter{
		dir:        logDir,
		filePrefix: "api-server",
	}
	multiWriter := io.MultiWriter(os.Stdout, writer)

	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.SetOutput(multiWriter)
	gin.DefaultWriter = multiWriter
	gin.DefaultErrorWriter = multiWriter

	log.Printf("[Logging] initialized stdout+daily-file logging log_dir=%s", logDir)
	return nil
}

func resolveLogDir() string {
	if configured := strings.TrimSpace(config.GetEnv("STUDYCLAW_LOG_DIR")); configured != "" {
		return configured
	}

	dataDir := strings.TrimSpace(config.GetEnv("STUDYCLAW_DATA_DIR"))
	if dataDir == "" {
		dataDir = "./data"
	}

	return filepath.Join(dataDir, "logs")
}

func (w *dailyFileWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := w.rotateIfNeeded(); err != nil {
		return 0, err
	}
	return w.file.Write(p)
}

func (w *dailyFileWriter) rotateIfNeeded() error {
	today := time.Now().Format("2006-01-02")
	if w.file != nil && w.currentDate == today {
		return nil
	}

	if err := os.MkdirAll(w.dir, 0o755); err != nil {
		return fmt.Errorf("create log dir %s: %w", w.dir, err)
	}

	if w.file != nil {
		_ = w.file.Close()
		w.file = nil
	}

	path := filepath.Join(w.dir, fmt.Sprintf("%s-%s.log", w.filePrefix, today))
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open log file %s: %w", path, err)
	}

	w.file = file
	w.currentDate = today
	return nil
}
