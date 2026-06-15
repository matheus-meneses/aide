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

var (
	mu     sync.Mutex
	out    io.Writer = os.Stderr
	level            = levelValues["info"]
	format           = "text"
)

// Logger is a lightweight, scoped front-end to the shared global sink. All
// loggers share one configured level, format, and writer so CLI, agent, and
// runner output interleave consistently.
type Logger struct {
	scope string
}

var std = &Logger{scope: "cli"}

// Configure sets the global logger threshold and format.
func Configure(lvl, fmtName string) {
	mu.Lock()
	defer mu.Unlock()
	if v, ok := levelValues[lvl]; ok {
		level = v
	}
	if fmtName == "json" {
		format = "json"
	} else {
		format = "text"
	}
}

// SetOutput redirects log output to w (defaults to os.Stderr).
func SetOutput(w io.Writer) {
	mu.Lock()
	defer mu.Unlock()
	out = w
}

// New returns a logger that tags its lines with the given scope.
func New(scope string) *Logger {
	if scope == "" {
		scope = "cli"
	}
	return &Logger{scope: scope}
}

// Resolve applies precedence flag > env > config > default for the log level
// and format, returning normalized values safe to pass to Configure.
func Resolve(flagLevel, flagFormat, cfgLevel, cfgFormat string) (lvl, fmtName string) {
	lvl = firstNonEmpty(flagLevel, os.Getenv("AIDE_LOG_LEVEL"), cfgLevel, "info")
	fmtName = firstNonEmpty(flagFormat, os.Getenv("AIDE_LOG_FORMAT"), cfgFormat, "text")
	if _, ok := levelValues[lvl]; !ok {
		lvl = "info"
	}
	if fmtName != "json" {
		fmtName = "text"
	}
	return lvl, fmtName
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func emit(scope, lvl, msg string) {
	ts := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	broadcast(LogEntry{TS: ts, Level: lvl, Scope: scope, Msg: msg})

	mu.Lock()
	defer mu.Unlock()
	if levelValues[lvl] < level {
		return
	}
	if format == "json" {
		b, _ := json.Marshal(struct {
			TS    string `json:"ts"`
			Level string `json:"level"`
			Scope string `json:"scope"`
			Msg   string `json:"msg"`
		}{TS: ts, Level: lvl, Scope: scope, Msg: msg})
		fmt.Fprintln(out, string(b))
		return
	}
	fmt.Fprintf(out, "%s [%s] %s: %s\n", ts, lvl, scope, msg)
}

type LogEntry struct {
	TS    string `json:"ts"`
	Level string `json:"level"`
	Scope string `json:"scope"`
	Msg   string `json:"msg"`
}

const logHistoryCap = 500

var (
	subMu       sync.RWMutex
	subscribers = make(map[chan LogEntry]struct{})

	histMu  sync.Mutex
	history = make([]LogEntry, 0, logHistoryCap)
)

func Subscribe() (<-chan LogEntry, func()) {
	ch := make(chan LogEntry, 256)
	subMu.Lock()
	subscribers[ch] = struct{}{}
	subMu.Unlock()

	return ch, func() {
		subMu.Lock()
		delete(subscribers, ch)
		subMu.Unlock()
		close(ch)
	}
}

func Recent() []LogEntry {
	histMu.Lock()
	defer histMu.Unlock()
	out := make([]LogEntry, len(history))
	copy(out, history)
	return out
}

func broadcast(e LogEntry) {
	histMu.Lock()
	if len(history) >= logHistoryCap {
		history = history[1:]
	}
	history = append(history, e)
	histMu.Unlock()

	subMu.RLock()
	for ch := range subscribers {
		select {
		case ch <- e:
		default:
		}
	}
	subMu.RUnlock()
}

func (l *Logger) Debug(f string, a ...any) { emit(l.scope, "debug", fmt.Sprintf(f, a...)) }
func (l *Logger) Info(f string, a ...any)  { emit(l.scope, "info", fmt.Sprintf(f, a...)) }
func (l *Logger) Warn(f string, a ...any)  { emit(l.scope, "warn", fmt.Sprintf(f, a...)) }
func (l *Logger) Error(f string, a ...any) { emit(l.scope, "error", fmt.Sprintf(f, a...)) }

func Debug(format string, args ...any) { std.Debug(format, args...) }
func Info(format string, args ...any)  { std.Info(format, args...) }
func Warn(format string, args ...any)  { std.Warn(format, args...) }
func Error(format string, args ...any) { std.Error(format, args...) }
