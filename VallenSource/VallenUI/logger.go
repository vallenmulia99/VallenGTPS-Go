package ui

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

// LogType represents the type of log message
type LogType int

const (
	LogHTTPS LogType = iota
	LogGTPS
)

// LogEntry represents a single log entry
type LogEntry struct {
	Timestamp time.Time
	Message   string
	Type      LogType
}

// Logger is a custom logger that routes messages to appropriate panels
type Logger struct {
	mu         sync.RWMutex
	httpsPanel *Panel
	gtpsPanel  *Panel
	enabled    bool
}

// NewLogger creates a new logger instance
func NewLogger(httpsPanel, gtpsPanel *Panel) *Logger {
	return &Logger{
		httpsPanel: httpsPanel,
		gtpsPanel:  gtpsPanel,
		enabled:    true,
	}
}

// Write implements io.Writer interface to intercept log output
func (l *Logger) Write(p []byte) (n int, err error) {
	if !l.enabled {
		return len(p), nil
	}

	msg := string(p)
	msg = strings.TrimSuffix(msg, "\n")

	// Route log based on prefix
	logType := l.detectLogType(msg)

	entry := LogEntry{
		Timestamp: time.Now(),
		Message:   msg,
		Type:      logType,
	}

	l.mu.RLock()
	defer l.mu.RUnlock()

	switch logType {
	case LogHTTPS:
		if l.httpsPanel != nil {
			l.httpsPanel.AddLog(entry)
		}
	case LogGTPS:
		if l.gtpsPanel != nil {
			l.gtpsPanel.AddLog(entry)
		}
	}

	return len(p), nil
}

// detectLogType determines which panel should receive this log
func (l *Logger) detectLogType(msg string) LogType {
	// HTTPS indicators
	httpsKeywords := []string{
		"[HTTPS]",
		"HTTPS server",
		"server_data",
		"POST /growtopia",
		"GET /",
		"Valid server_data request",
		"Unknown request",
	}

	for _, keyword := range httpsKeywords {
		if strings.Contains(msg, keyword) {
			return LogHTTPS
		}
	}

	// Everything else goes to GTPS panel
	return LogGTPS
}

// SetEnabled enables or disables logging
func (l *Logger) SetEnabled(enabled bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.enabled = enabled
}

// MultiWriter creates a writer that writes to both the logger and original output
type MultiWriter struct {
	writers []io.Writer
}

func NewMultiWriter(writers ...io.Writer) *MultiWriter {
	return &MultiWriter{writers: writers}
}

func (mw *MultiWriter) Write(p []byte) (n int, err error) {
	for _, w := range mw.writers {
		n, err = w.Write(p)
		if err != nil {
			return
		}
		if n != len(p) {
			err = io.ErrShortWrite
			return
		}
	}
	return len(p), nil
}

// FormatLogEntry formats a log entry with timestamp and color
func FormatLogEntry(entry LogEntry) string {
	timestamp := entry.Timestamp.Format("15:04:05")
	return fmt.Sprintf("[%s] %s", timestamp, entry.Message)
}
