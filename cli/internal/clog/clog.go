package clog

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

var levelValues = map[string]int{"debug": 10, "info": 20, "warn": 30, "error": 40}

// Logger emits format-aware (text or JSON) log lines to a writer, matching the
// runner's wire format so CLI and plugin logs interleave consistently.
type Logger struct {
	mu     sync.Mutex
	out    io.Writer
	level  int
	format string
	scope  string
}

var std = &Logger{out: os.Stderr, level: levelValues["info"], format: "text", scope: "cli"}

// Configure sets the global logger threshold and format from the CLI flags.
func Configure(level, format string) {
	std.mu.Lock()
	defer std.mu.Unlock()
	if v, ok := levelValues[level]; ok {
		std.level = v
	}
	if format == "json" {
		std.format = "json"
	} else {
		std.format = "text"
	}
}

func (l *Logger) emit(level, msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if levelValues[level] < l.level {
		return
	}
	ts := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	if l.format == "json" {
		b, _ := json.Marshal(struct {
			TS    string `json:"ts"`
			Level string `json:"level"`
			Scope string `json:"scope"`
			Msg   string `json:"msg"`
		}{TS: ts, Level: level, Scope: l.scope, Msg: msg})
		fmt.Fprintln(l.out, string(b))
		return
	}
	fmt.Fprintf(l.out, "%s [%s] %s: %s\n", ts, level, l.scope, msg)
}

func Debug(format string, args ...any) { std.emit("debug", fmt.Sprintf(format, args...)) }
func Info(format string, args ...any)  { std.emit("info", fmt.Sprintf(format, args...)) }
func Warn(format string, args ...any)  { std.emit("warn", fmt.Sprintf(format, args...)) }
func Error(format string, args ...any) { std.emit("error", fmt.Sprintf(format, args...)) }
