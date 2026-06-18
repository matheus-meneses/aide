package clog

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var levelValues = map[string]int{"debug": 10, "info": 20, "warn": 30, "error": 40}

const maxLogBytes = 5 << 20

var (
	mu     sync.Mutex
	out    io.Writer = os.Stderr
	level            = levelValues["info"]
	format           = "text"

	logFile  *os.File
	logPath  string
	logBytes int64
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

// SetOutput redirects console log output to w (defaults to os.Stderr).
func SetOutput(w io.Writer) {
	mu.Lock()
	defer mu.Unlock()
	out = w
}

// SetFile mirrors every emitted entry as a JSON line into path, creating the
// parent directory if needed. The file sink is independent of the console
// format so the tailing log viewer can always parse it. Subsequent calls
// replace the previous sink.
func SetFile(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating log dir: %w", err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("opening log file: %w", err)
	}
	info, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return fmt.Errorf("stat log file: %w", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if logFile != nil {
		_ = logFile.Close()
	}
	logFile = f
	logPath = path
	logBytes = info.Size()
	return nil
}

// ClearFile prunes the persisted logs: it truncates the active file and removes
// the rotated backup. It is a no-op when no file sink is configured.
func ClearFile() error {
	mu.Lock()
	defer mu.Unlock()
	if logFile == nil {
		return nil
	}
	if err := logFile.Truncate(0); err != nil {
		return fmt.Errorf("truncating log file: %w", err)
	}
	if _, err := logFile.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("seeking log file: %w", err)
	}
	logBytes = 0
	_ = os.Remove(logPath + ".1")
	return nil
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

// Emit routes a pre-scoped entry from another subsystem (e.g. the runner or a
// plugin subprocess) through the shared sinks, honoring the global level so
// everything lands in the same console and file output.
func Emit(scope, level, msg string) {
	if scope == "" {
		scope = "cli"
	}
	if _, ok := levelValues[level]; !ok {
		level = "info"
	}
	emit(scope, level, msg)
}

func emit(scope, lvl, msg string) {
	ts := time.Now().UTC().Format("2006-01-02T15:04:05Z")

	mu.Lock()
	defer mu.Unlock()

	if levelValues[lvl] < level {
		return
	}

	line := marshalEntry(ts, lvl, scope, msg)
	if logFile != nil {
		writeFileLocked(line)
	}

	if format == "json" {
		fmt.Fprintln(out, line)
		return
	}
	fmt.Fprintf(out, "%s [%s] %s: %s\n", ts, lvl, scope, msg)
}

func marshalEntry(ts, lvl, scope, msg string) string {
	b, _ := json.Marshal(LogEntry{TS: ts, Level: lvl, Scope: scope, Msg: msg})
	return string(b)
}

// writeFileLocked appends a JSON line to the active sink and rotates when the
// configured size cap is exceeded. Callers must hold mu.
func writeFileLocked(line string) {
	n, err := io.WriteString(logFile, line+"\n")
	if err != nil {
		return
	}
	logBytes += int64(n)
	if logBytes >= maxLogBytes {
		rotateLocked()
	}
}

// rotateLocked renames the active file to a single ".1" backup and reopens a
// fresh file. Callers must hold mu.
func rotateLocked() {
	_ = logFile.Close()
	_ = os.Remove(logPath + ".1")
	_ = os.Rename(logPath, logPath+".1")

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		logFile = nil
		logBytes = 0
		return
	}
	logFile = f
	logBytes = 0
}

type LogEntry struct {
	TS    string `json:"ts"`
	Level string `json:"level"`
	Scope string `json:"scope"`
	Msg   string `json:"msg"`
}

func (l *Logger) Debug(f string, a ...any) { emit(l.scope, "debug", fmt.Sprintf(f, a...)) }
func (l *Logger) Info(f string, a ...any)  { emit(l.scope, "info", fmt.Sprintf(f, a...)) }
func (l *Logger) Warn(f string, a ...any)  { emit(l.scope, "warn", fmt.Sprintf(f, a...)) }
func (l *Logger) Error(f string, a ...any) { emit(l.scope, "error", fmt.Sprintf(f, a...)) }

func Debug(format string, args ...any) { std.Debug(format, args...) }
func Info(format string, args ...any)  { std.Info(format, args...) }
func Warn(format string, args ...any)  { std.Warn(format, args...) }
func Error(format string, args ...any) { std.Error(format, args...) }
