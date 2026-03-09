// Package logging provides a structured logging abstraction for the application.
//
// It wraps zerolog to provide component-scoped loggers with consistent field naming.
// Every subsystem should create its own logger via logging.Component("name") to make
// it easy to filter and trace log output by component.
//
// Log levels:
//   - Trace: Very fine-grained, per-iteration details (e.g., each chunk written)
//   - Debug: Diagnostic info useful during development (e.g., ffmpeg args, SQL queries)
//   - Info:  Normal operational events (e.g., sync started, download complete)
//   - Warn:  Recoverable issues that deserve attention (e.g., missing optional metadata)
//   - Error: Failures that affect a single operation but not the whole system
//   - Fatal: Unrecoverable errors that prevent the application from starting
package logging

import (
	"io"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
)

// Logger is a component-scoped structured logger.
//
// Component loggers are often created at package init time before Init runs.
// To ensure output format and level still follow runtime config, each event
// call builds a logger from current global settings.
type Logger struct {
	component string
	fields    map[string]string
}

var globalLevel = zerolog.InfoLevel
var useJSONOutput = true

// Init configures the global logging defaults. Call once at startup.
func Init(level string, jsonOutput bool) {
	globalLevel = parseLevel(level)
	useJSONOutput = jsonOutput

	zerolog.SetGlobalLevel(globalLevel)
	zerolog.TimeFieldFormat = time.RFC3339

	zl := zerolog.New(outputWriter()).With().Timestamp().Logger().Level(globalLevel)
	setGlobalLogger(zl)
}

func setGlobalLogger(zl zerolog.Logger) {
	zlog.Logger = zl
}

func outputWriter() io.Writer {
	if useJSONOutput {
		return os.Stderr
	}
	return zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: "15:04:05",
	}
}

func (l *Logger) build() zerolog.Logger {
	ctx := zerolog.New(outputWriter()).With().Timestamp()
	if l.component != "" {
		ctx = ctx.Str("component", l.component)
	}
	for k, v := range l.fields {
		ctx = ctx.Str(k, v)
	}
	return ctx.Logger().Level(globalLevel)
}

// Component creates a logger scoped to a named component.
// The component name appears in every log line for easy filtering.
//
//	log := logging.Component("sync")
//	log.Info().Str("asin", asin).Msg("book synced")
func Component(name string) *Logger {
	return &Logger{component: name, fields: map[string]string{}}
}

// GetZerolog returns the underlying zerolog.Logger for advanced use cases.
func (l *Logger) GetZerolog() zerolog.Logger {
	return l.build()
}

// --- Trace ---

func (l *Logger) Trace() *zerolog.Event {
	zl := l.build()
	return zl.Trace()
}

// --- Debug ---

func (l *Logger) Debug() *zerolog.Event {
	zl := l.build()
	return zl.Debug()
}

// --- Info ---

func (l *Logger) Info() *zerolog.Event {
	zl := l.build()
	return zl.Info()
}

// --- Warn ---

func (l *Logger) Warn() *zerolog.Event {
	zl := l.build()
	return zl.Warn()
}

// --- Error ---

func (l *Logger) Error() *zerolog.Event {
	zl := l.build()
	return zl.Error()
}

// --- Fatal ---

func (l *Logger) Fatal() *zerolog.Event {
	zl := l.build()
	return zl.Fatal()
}

// --- Convenience methods for common patterns ---

// Err returns an error-level event with the error already attached.
func (l *Logger) Err(err error) *zerolog.Event {
	zl := l.build()
	return zl.Error().Err(err)
}

// WithField returns a new Logger with an additional field baked in.
func (l *Logger) WithField(key, value string) *Logger {
	fields := make(map[string]string, len(l.fields)+1)
	for k, v := range l.fields {
		fields[k] = v
	}
	fields[key] = value
	return &Logger{component: l.component, fields: fields}
}

// WithFields returns a new Logger with additional fields baked in.
func (l *Logger) WithFields(fields map[string]string) *Logger {
	merged := make(map[string]string, len(l.fields)+len(fields))
	for k, v := range l.fields {
		merged[k] = v
	}
	for k, v := range fields {
		merged[k] = v
	}
	return &Logger{component: l.component, fields: merged}
}

func parseLevel(level string) zerolog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "trace":
		return zerolog.TraceLevel
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn", "warning":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "fatal":
		return zerolog.FatalLevel
	default:
		return zerolog.InfoLevel
	}
}
