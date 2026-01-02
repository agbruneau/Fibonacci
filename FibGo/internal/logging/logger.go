// Package logging provides a unified logging interface for the Fibonacci calculator.
// It abstracts the underlying logging implementation, allowing consistent logging
// across different components while supporting multiple backends (zerolog, std log).
package logging

import (
	"io"
	stdlog "log"
	"os"

	"github.com/rs/zerolog"
)

// Logger is the unified logging interface used across the application.
// It provides a consistent API for logging at different levels.
type Logger interface {
	// Info logs an informational message.
	Info(msg string, fields ...Field)

	// Error logs an error message with the associated error.
	Error(msg string, err error, fields ...Field)

	// Debug logs a debug message.
	Debug(msg string, fields ...Field)

	// Printf provides compatibility with the standard log.Logger Printf method.
	Printf(format string, args ...any)

	// Println provides compatibility with the standard log.Logger Println method.
	Println(args ...any)
}

// Field represents a key-value pair for structured logging.
type Field struct {
	Key   string
	Value any
}

// String creates a string field.
func String(key, value string) Field {
	return Field{Key: key, Value: value}
}

// Int creates an integer field.
func Int(key string, value int) Field {
	return Field{Key: key, Value: value}
}

// Uint64 creates a uint64 field.
func Uint64(key string, value uint64) Field {
	return Field{Key: key, Value: value}
}

// Float64 creates a float64 field.
func Float64(key string, value float64) Field {
	return Field{Key: key, Value: value}
}

// Err creates an error field.
func Err(err error) Field {
	return Field{Key: "error", Value: err}
}

// ZerologAdapter adapts a zerolog.Logger to the Logger interface.
type ZerologAdapter struct {
	logger zerolog.Logger
}

// NewZerologAdapter creates a new Logger backed by zerolog.
func NewZerologAdapter(logger zerolog.Logger) *ZerologAdapter {
	return &ZerologAdapter{logger: logger}
}

// NewDefaultLogger creates a Logger with sensible defaults for the application.
func NewDefaultLogger() *ZerologAdapter {
	return NewZerologAdapter(
		zerolog.New(os.Stderr).With().Timestamp().Logger(),
	)
}

// NewLogger creates a Logger writing to the specified output.
func NewLogger(w io.Writer, prefix string) *ZerologAdapter {
	return NewZerologAdapter(
		zerolog.New(w).With().Str("component", prefix).Timestamp().Logger(),
	)
}

func (z *ZerologAdapter) applyFields(event *zerolog.Event, fields []Field) *zerolog.Event {
	for _, f := range fields {
		switch v := f.Value.(type) {
		case string:
			event = event.Str(f.Key, v)
		case int:
			event = event.Int(f.Key, v)
		case int64:
			event = event.Int64(f.Key, v)
		case uint64:
			event = event.Uint64(f.Key, v)
		case float64:
			event = event.Float64(f.Key, v)
		case error:
			event = event.Err(v)
		case bool:
			event = event.Bool(f.Key, v)
		default:
			event = event.Interface(f.Key, v)
		}
	}
	return event
}

// Info logs an informational message.
func (z *ZerologAdapter) Info(msg string, fields ...Field) {
	event := z.logger.Info()
	z.applyFields(event, fields).Msg(msg)
}

// Error logs an error message.
func (z *ZerologAdapter) Error(msg string, err error, fields ...Field) {
	event := z.logger.Error().Err(err)
	z.applyFields(event, fields).Msg(msg)
}

// Debug logs a debug message.
func (z *ZerologAdapter) Debug(msg string, fields ...Field) {
	event := z.logger.Debug()
	z.applyFields(event, fields).Msg(msg)
}

// Printf provides compatibility with standard log.Printf.
func (z *ZerologAdapter) Printf(format string, args ...any) {
	z.logger.Info().Msgf(format, args...)
}

// Println provides compatibility with standard log.Println.
func (z *ZerologAdapter) Println(args ...any) {
	z.logger.Info().Msgf("%v", args)
}

// StdLoggerAdapter adapts a standard log.Logger to the Logger interface.
// This is useful for backward compatibility with code using log.Logger.
type StdLoggerAdapter struct {
	logger *stdlog.Logger
}

// NewStdLoggerAdapter creates a new Logger backed by standard log.Logger.
func NewStdLoggerAdapter(logger *stdlog.Logger) *StdLoggerAdapter {
	return &StdLoggerAdapter{logger: logger}
}

// Info logs an informational message.
func (s *StdLoggerAdapter) Info(msg string, fields ...Field) {
	if len(fields) == 0 {
		s.logger.Println("[INFO]", msg)
	} else {
		s.logger.Printf("[INFO] %s %v\n", msg, fields)
	}
}

// Error logs an error message.
func (s *StdLoggerAdapter) Error(msg string, err error, fields ...Field) {
	if len(fields) == 0 {
		s.logger.Printf("[ERROR] %s: %v\n", msg, err)
	} else {
		s.logger.Printf("[ERROR] %s: %v %v\n", msg, err, fields)
	}
}

// Debug logs a debug message.
func (s *StdLoggerAdapter) Debug(msg string, fields ...Field) {
	if len(fields) == 0 {
		s.logger.Println("[DEBUG]", msg)
	} else {
		s.logger.Printf("[DEBUG] %s %v\n", msg, fields)
	}
}

// Printf provides compatibility with standard log.Printf.
func (s *StdLoggerAdapter) Printf(format string, args ...any) {
	s.logger.Printf(format, args...)
}

// Println provides compatibility with standard log.Println.
func (s *StdLoggerAdapter) Println(args ...any) {
	s.logger.Println(args...)
}
