package utils

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	// DefaultLogPath is the default location for the log file
	DefaultLogPath = "/var/log/reghost.log"
	// MaxLogSize is the maximum size of a log file before rotation (5MB)
	MaxLogSize = 5 * 1024 * 1024
	// MaxLogAge is the maximum age of log files to keep (7 days)
	MaxLogAge = 7 * 24 * time.Hour
	// MaxLogBackups is the maximum number of old log files to keep
	MaxLogBackups = 7
)

// Logger handles application logging with rotation
type Logger struct {
	mu       sync.Mutex
	file     *os.File
	path     string
	size     int64
	maxSize  int64
	maxAge   time.Duration
	maxFiles int
}

// NewLogger creates a new logger with rotation support
func NewLogger(path string) (*Logger, error) {
	// Create log directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open or create log file
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	// Get current file size
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to stat log file: %w", err)
	}

	logger := &Logger{
		file:     file,
		path:     path,
		size:     info.Size(),
		maxSize:  MaxLogSize,
		maxAge:   MaxLogAge,
		maxFiles: MaxLogBackups,
	}

	// Clean old log files
	go logger.cleanOldLogs()

	return logger, nil
}

// Write implements io.Writer
func (l *Logger) Write(p []byte) (n int, err error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Check if rotation is needed
	if l.size+int64(len(p)) > l.maxSize {
		if err := l.rotate(); err != nil {
			return 0, err
		}
	}

	// Write to file
	n, err = l.file.Write(p)
	l.size += int64(n)
	return n, err
}

// rotate rotates the log file
func (l *Logger) rotate() error {
	// Close current file
	if err := l.file.Close(); err != nil {
		return err
	}

	// Rename current file with timestamp
	timestamp := time.Now().Format("20060102-150405")
	backupPath := fmt.Sprintf("%s.%s", l.path, timestamp)
	if err := os.Rename(l.path, backupPath); err != nil {
		return err
	}

	// Open new file
	file, err := os.OpenFile(l.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	l.file = file
	l.size = 0

	// Clean old logs asynchronously
	go l.cleanOldLogs()

	return nil
}

// cleanOldLogs removes old log files based on age and count
func (l *Logger) cleanOldLogs() {
	dir := filepath.Dir(l.path)
	basename := filepath.Base(l.path)

	// Find all log backup files
	pattern := filepath.Join(dir, basename+".*")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return
	}

	// Sort by modification time (oldest first)
	type logFile struct {
		path    string
		modTime time.Time
	}

	var logs []logFile
	now := time.Now()

	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil {
			continue
		}

		logs = append(logs, logFile{
			path:    match,
			modTime: info.ModTime(),
		})
	}

	// Remove files older than maxAge or exceeding maxFiles
	for i, log := range logs {
		// Remove if too old
		if now.Sub(log.modTime) > l.maxAge {
			os.Remove(log.path)
			continue
		}

		// Remove if exceeding max count (keep newest files)
		if len(logs)-i > l.maxFiles {
			os.Remove(log.path)
		}
	}
}

// Close closes the logger
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// Info logs an info message
func (l *Logger) Info(format string, args ...interface{}) {
	l.log("INFO", format, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log("WARN", format, args...)
}

// Error logs an error message
func (l *Logger) Error(format string, args ...interface{}) {
	l.log("ERROR", format, args...)
}

// log writes a formatted log message
func (l *Logger) log(level, format string, args ...interface{}) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	message := fmt.Sprintf(format, args...)
	logLine := fmt.Sprintf("[%s] [%s] %s\n", timestamp, level, message)

	// Write to log file
	l.Write([]byte(logLine))

	// Also write to stdout
	fmt.Print(logLine)
}

// MultiWriter returns a writer that writes to both the logger and stdout
func (l *Logger) MultiWriter() io.Writer {
	return io.MultiWriter(l, os.Stdout)
}
