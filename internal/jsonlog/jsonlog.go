// Package jsonlog provides a thread-safe, structured JSON logger with severity filtering.
package jsonlog

import (
	"encoding/json"
	"io"
	"os"
	"runtime/debug"
	"sync"
	"time"
)

// Level represents the log severity threshold.
type Level int8

const (
	LevelInfo  Level = iota // 0: Informational messages
	LevelError              // 1:  Runtime errors with stack traces
	LevelFatal              // 2: Fatal errors
	LevelOff                // 3: Disable logging
)

// String returns the uppercase string representation of level.
func (l Level) String() string {
	switch l {
	case LevelInfo:
		return "INFO"
	case LevelError:
		return "ERROR"
	case LevelFatal:
		return "FATAL"
	default:
		return ""
	}
}

// Logger writes thread-safe, structured JSON records to an output destination.
type Logger struct {
	out      io.Writer  // Log destination stream
	minLevel Level      // Minimum active log level
	mu       sync.Mutex // Serializes concurrent writes
}

// New constructs a Logger instance writing entries >= minLvel to out.
func New(out io.Writer, minLevel Level) *Logger {
	return &Logger{
		out:      out,
		minLevel: minLevel,
	}
}

// PrintInfo writes an INFO level log entry with metadata.
func (l *Logger) PrintInfo(message string, properties map[string]string) {
	l.print(LevelInfo, message, properties)
}

// PrintError writes an ERROR lev el log entry including stack trace.
func (l *Logger) PrintError(err error, properties map[string]string) {
	l.print(LevelError, err.Error(), properties)
}

// PrintFatal writes a FATAL level log entry and terminates the application with status code 1.
func (l *Logger) PrintFatal(err error, properties map[string]string) {
	l.print(LevelFatal, err.Error(), properties)
	os.Exit(1)
}

// print marshals and writes structures log lines filtered by minLevel.
func (l *Logger) print(level Level, message string, properties map[string]string) (int, error) {
	// Skip entries below severity threshold
	if level < l.minLevel {
		return 0, nil
	}

	// Construct structured log payload
	entry := struct {
		Level      string            `json:"level"`
		Time       string            `json:"time"`
		Message    string            `json:"message"`
		Properties map[string]string `json:"properties,omitempty"`
		Trace      string            `json:"trace,omitempty"`
	}{
		Level:      level.String(),
		Time:       time.Now().UTC().Format(time.RFC3339),
		Message:    message,
		Properties: properties,
	}

	// Attach stack trace for error and fatal levels
	if level >= LevelError {
		entry.Trace = string(debug.Stack())
	}

	// Marshal payload to JSON
	var line []byte
	line, err := json.Marshal(entry)
	if err != nil {
		line = []byte(LevelError.String() + ": unable to marshal log entry: " + err.Error())

	}

	// Synchronize output writes
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.out.Write(append(line, '\n'))
}

// Write satisfies io.Writer to pipe external loggers
func (l *Logger) Write(message []byte) (n int, err error) {
	return l.print(LevelError, string(message), nil)
}
