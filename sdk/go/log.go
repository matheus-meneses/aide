package plugin

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

var levelValues = map[string]int{"debug": 10, "info": 20, "warn": 30, "error": 40}

// Logger writes leveled log lines to stderr in text or json format.
type Logger struct {
	threshold int
	format    string
	scope     string
}

// NewLogger creates a Logger with the given level ("debug"|"info"|"warn"|"error"),
// format ("text"|"json"), and scope label. Unknown levels default to info.
func NewLogger(level, format, scope string) *Logger {
	t, ok := levelValues[level]
	if !ok {
		t = 20
	}
	if format != "json" {
		format = "text"
	}
	return &Logger{threshold: t, format: format, scope: scope}
}

func (l *Logger) emit(level, msg string) {
	if levelValues[level] < l.threshold {
		return
	}
	ts := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	if l.format == "json" {
		type rec struct {
			TS    string `json:"ts"`
			Level string `json:"level"`
			Scope string `json:"scope,omitempty"`
			Msg   string `json:"msg"`
		}
		b, _ := json.Marshal(rec{TS: ts, Level: level, Scope: l.scope, Msg: msg})
		fmt.Fprintln(os.Stderr, string(b))
		return
	}
	prefix := ""
	if l.scope != "" {
		prefix = l.scope + ": "
	}
	fmt.Fprintf(os.Stderr, "%s [%s] %s%s\n", ts, level, prefix, msg)
}

func (l *Logger) Debugf(f string, a ...any) { l.emit("debug", fmt.Sprintf(f, a...)) }
func (l *Logger) Infof(f string, a ...any)  { l.emit("info", fmt.Sprintf(f, a...)) }
func (l *Logger) Warnf(f string, a ...any)  { l.emit("warn", fmt.Sprintf(f, a...)) }
func (l *Logger) Errorf(f string, a ...any) { l.emit("error", fmt.Sprintf(f, a...)) }

// Log is the package-level logger, configured by Serve before Handle is called.
var Log = NewLogger("info", "text", "")

func stringFromContext(ctx map[string]any, key string) string {
	if ctx == nil {
		return ""
	}
	if v, ok := ctx[key].(string); ok {
		return v
	}
	return ""
}

func boolFromContext(ctx map[string]any, key string, def bool) bool {
	if ctx == nil {
		return def
	}
	if v, ok := ctx[key].(bool); ok {
		return v
	}
	return def
}
