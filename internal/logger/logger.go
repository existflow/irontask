package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// Level represents log severity
type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
)

// String returns the string representation of the log level
func (l Level) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// ParseLevel converts a string to a Level
func ParseLevel(s string) Level {
	switch s {
	case "DEBUG":
		return DEBUG
	case "INFO":
		return INFO
	case "WARN":
		return WARN
	case "ERROR":
		return ERROR
	default:
		return INFO
	}
}

// Field represents a key-value pair for structured logging
type Field struct {
	Key   string
	Value interface{}
}

// F is a shorthand for creating a Field
func F(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}

// Config holds logger configuration
type Config struct {
	Level      Level  // Minimum log level
	FilePath   string // Path to log file
	MaxSize    int64  // Max size in bytes before rotation (default: 10MB)
	MaxAge     int    // Max age in days (default: 7)
	MaxBackups int    // Max number of backup files (default: 5)
	Console    bool   // Enable console logging
}

// DefaultConfig returns default logger configuration
func DefaultConfig() Config {
	home, _ := os.UserHomeDir()
	logPath := filepath.Join(home, ".irontask", "logs", "irontask.log")

	return Config{
		Level:      INFO,
		FilePath:   logPath,
		MaxSize:    10 * 1024 * 1024, // 10MB
		MaxAge:     7,
		MaxBackups: 5,
		Console:    false, // Disabled by default to not interfere with TUI
	}
}

// Logger is the main logger instance
type Logger struct {
	config  Config
	file    *os.File
	mu      sync.Mutex
	fields  []Field
	writers []io.Writer
}

var (
	globalLogger *Logger
	once         sync.Once
)

// Init initializes the global logger
func Init(config Config) error {
	var err error
	once.Do(func() {
		globalLogger, err = New(config)
	})
	return err
}

// New creates a new logger instance
func New(config Config) (*Logger, error) {
	l := &Logger{
		config:  config,
		fields:  []Field{},
		writers: []io.Writer{},
	}

	// Create log directory if it doesn't exist
	if config.FilePath != "" {
		logDir := filepath.Dir(config.FilePath)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %w", err)
		}

		// Open log file
		file, err := os.OpenFile(config.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}
		l.file = file
		l.writers = append(l.writers, file)

		// Check if rotation is needed
		if err := l.rotateIfNeeded(); err != nil {
			return nil, err
		}
	}

	// Add console output if enabled
	if config.Console {
		l.writers = append(l.writers, os.Stderr)
	}

	return l, nil
}

// rotateIfNeeded checks if log rotation is needed and performs it
func (l *Logger) rotateIfNeeded() error {
	if l.file == nil {
		return nil
	}

	info, err := l.file.Stat()
	if err != nil {
		return err
	}

	// Check size
	if info.Size() >= l.config.MaxSize {
		return l.rotate()
	}

	// Check age
	if time.Since(info.ModTime()) > time.Duration(l.config.MaxAge)*24*time.Hour {
		return l.rotate()
	}

	return nil
}

// rotate performs log rotation
func (l *Logger) rotate() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		l.file.Close()
	}

	// Rotate existing backups
	for i := l.config.MaxBackups - 1; i >= 1; i-- {
		oldPath := fmt.Sprintf("%s.%d", l.config.FilePath, i)
		newPath := fmt.Sprintf("%s.%d", l.config.FilePath, i+1)
		os.Rename(oldPath, newPath)
	}

	// Move current log to .1
	if _, err := os.Stat(l.config.FilePath); err == nil {
		backupPath := fmt.Sprintf("%s.1", l.config.FilePath)
		if err := os.Rename(l.config.FilePath, backupPath); err != nil {
			return err
		}
	}

	// Open new log file
	file, err := os.OpenFile(l.config.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	l.file = file
	l.writers = []io.Writer{file}
	if l.config.Console {
		l.writers = append(l.writers, os.Stderr)
	}

	return nil
}

// log writes a log entry
func (l *Logger) log(level Level, msg string, fields []Field) {
	if level < l.config.Level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// Check rotation before writing
	l.rotateIfNeeded()

	// Get caller info
	_, file, line, ok := runtime.Caller(2)
	caller := "???"
	if ok {
		caller = fmt.Sprintf("%s:%d", filepath.Base(file), line)
	}

	// Build log entry
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	entry := fmt.Sprintf("[%s] %s %s: %s", timestamp, level.String(), caller, msg)

	// Add fields
	allFields := append(l.fields, fields...)
	if len(allFields) > 0 {
		entry += " |"
		for _, f := range allFields {
			entry += fmt.Sprintf(" %s=%v", f.Key, f.Value)
		}
	}
	entry += "\n"

	// Write to all outputs
	for _, w := range l.writers {
		w.Write([]byte(entry))
	}
}

// WithFields creates a new logger with preset fields
func (l *Logger) WithFields(fields ...Field) *Logger {
	newLogger := &Logger{
		config:  l.config,
		file:    l.file,
		fields:  append(l.fields, fields...),
		writers: l.writers,
	}
	return newLogger
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, fields ...Field) {
	l.log(DEBUG, msg, fields)
}

// Info logs an info message
func (l *Logger) Info(msg string, fields ...Field) {
	l.log(INFO, msg, fields)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, fields ...Field) {
	l.log(WARN, msg, fields)
}

// Error logs an error message
func (l *Logger) Error(msg string, fields ...Field) {
	l.log(ERROR, msg, fields)
}

// Close closes the logger and flushes any buffered data
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// Global logger functions

// Debug logs a debug message using the global logger
func Debug(msg string, fields ...Field) {
	if globalLogger != nil {
		globalLogger.Debug(msg, fields...)
	}
}

// Info logs an info message using the global logger
func Info(msg string, fields ...Field) {
	if globalLogger != nil {
		globalLogger.Info(msg, fields...)
	}
}

// Warn logs a warning message using the global logger
func Warn(msg string, fields ...Field) {
	if globalLogger != nil {
		globalLogger.Warn(msg, fields...)
	}
}

// Error logs an error message using the global logger
func Error(msg string, fields ...Field) {
	if globalLogger != nil {
		globalLogger.Error(msg, fields...)
	}
}

// WithFields creates a new logger with preset fields using the global logger
func WithFields(fields ...Field) *Logger {
	if globalLogger != nil {
		return globalLogger.WithFields(fields...)
	}
	return nil
}

// Close closes the global logger
func Close() error {
	if globalLogger != nil {
		return globalLogger.Close()
	}
	return nil
}

// GetConfig returns the current logger configuration
func GetConfig() Config {
	if globalLogger != nil {
		return globalLogger.config
	}
	return DefaultConfig()
}
