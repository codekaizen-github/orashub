package logger

import (
	"fmt"
	"log"
	"net/http"
	"strings"
)

// LogLevel represents the level of logging
type LogLevel int

const (
	// LogLevelError only logs errors
	LogLevelError LogLevel = iota
	// LogLevelWarn logs warnings and errors
	LogLevelWarn
	// LogLevelInfo logs info, warnings and errors
	LogLevelInfo
	// LogLevelDebug logs debug, info, warnings and errors
	LogLevelDebug
	// LogLevelTrace logs everything including detailed trace information
	LogLevelTrace
)

// Logger defines the interface for logging
type Logger interface {
	Error(format string, v ...interface{})
	Warn(format string, v ...interface{})
	Info(format string, v ...interface{})
	Debug(format string, v ...interface{})
	Trace(format string, v ...interface{})
	GetLevel() LogLevel
	SetLevel(level LogLevel)
	SetLevelFromString(level string) error
}

// DefaultLogger implements the Logger interface
type DefaultLogger struct {
	currentLevel LogLevel
}

// NewDefaultLogger creates a new DefaultLogger with the specified log level
func NewDefaultLogger(level LogLevel) *DefaultLogger {
	return &DefaultLogger{
		currentLevel: level,
	}
}

// Error logs an error message
func (l *DefaultLogger) Error(format string, v ...interface{}) {
	if l.currentLevel >= LogLevelError {
		l.logMessage(LogLevelError, format, v...)
	}
}

// Warn logs a warning message
func (l *DefaultLogger) Warn(format string, v ...interface{}) {
	if l.currentLevel >= LogLevelWarn {
		l.logMessage(LogLevelWarn, format, v...)
	}
}

// Info logs an informational message
func (l *DefaultLogger) Info(format string, v ...interface{}) {
	if l.currentLevel >= LogLevelInfo {
		l.logMessage(LogLevelInfo, format, v...)
	}
}

// Debug logs a debug message
func (l *DefaultLogger) Debug(format string, v ...interface{}) {
	if l.currentLevel >= LogLevelDebug {
		l.logMessage(LogLevelDebug, format, v...)
	}
}

// Trace logs a trace message
func (l *DefaultLogger) Trace(format string, v ...interface{}) {
	if l.currentLevel >= LogLevelTrace {
		l.logMessage(LogLevelTrace, format, v...)
	}
}

// GetLevel returns the current log level
func (l *DefaultLogger) GetLevel() LogLevel {
	return l.currentLevel
}

// SetLevel sets the current log level
func (l *DefaultLogger) SetLevel(level LogLevel) {
	l.currentLevel = level
	l.Info("Log level set to %s", l.getLevelName(level))
}

// SetLevelFromString sets the current log level from a string
func (l *DefaultLogger) SetLevelFromString(level string) error {
	level = strings.ToLower(level)
	switch level {
	case "error":
		l.SetLevel(LogLevelError)
	case "warn", "warning":
		l.SetLevel(LogLevelWarn)
	case "info":
		l.SetLevel(LogLevelInfo)
	case "debug":
		l.SetLevel(LogLevelDebug)
	case "trace":
		l.SetLevel(LogLevelTrace)
	default:
		return fmt.Errorf("unknown log level: %s", level)
	}
	return nil
}

// logMessage logs a message with the specified level
func (l *DefaultLogger) logMessage(level LogLevel, format string, v ...interface{}) {
	prefix := l.getLevelPrefix(level)
	log.Printf(prefix+format, v...)
}

// getLevelPrefix returns the prefix for the specified log level
func (l *DefaultLogger) getLevelPrefix(level LogLevel) string {
	prefix := ""
	switch level {
	case LogLevelError:
		prefix = "ERROR: "
	case LogLevelWarn:
		prefix = "WARN: "
	case LogLevelInfo:
		prefix = "INFO: "
	case LogLevelDebug:
		prefix = "DEBUG: "
	case LogLevelTrace:
		prefix = "TRACE: "
	}
	return prefix
}

// getLevelName returns the name of the specified log level
func (l *DefaultLogger) getLevelName(level LogLevel) string {
	switch level {
	case LogLevelError:
		return "ERROR"
	case LogLevelWarn:
		return "WARN"
	case LogLevelInfo:
		return "INFO"
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelTrace:
		return "TRACE"
	default:
		return "UNKNOWN"
	}
}

// LoggingMiddleware creates middleware that logs HTTP requests
func LoggingMiddleware(logger Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Always log requests at INFO level
		logger.Info("%s %s %s", r.RemoteAddr, r.Method, r.URL.Path)

		// Log more details at DEBUG level
		if logger.GetLevel() >= LogLevelDebug {
			logger.Debug("Request Headers: %v", r.Header)
			logger.Debug("Request Query: %v", r.URL.Query())
		}

		next.ServeHTTP(w, r)
	})
}
