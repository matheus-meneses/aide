package runner

import (
	"aide/cli/internal/platform/clog"
	"encoding/json"
	"strings"
)

var rlog = clog.New("runner")

func (r *Runner) debugf(format string, args ...any) { rlog.Debug(format, args...) }
func (r *Runner) infof(format string, args ...any)  { rlog.Info(format, args...) }
func (r *Runner) errorf(format string, args ...any) { rlog.Error(format, args...) }

// streamStderr routes plugin subprocess stderr through clog so it reaches the
// shared sinks (and the desktop Logs view). Lines the plugin already emitted as
// structured JSON keep their original scope and level; SDK text lines carry a
// "[level]" tag that is honored too; anything else is tagged with the source
// name at info level.
func (r *Runner) streamStderr(source, output string) {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if entry, ok := parseStructured(line); ok {
			scope := entry.Scope
			if scope == "" {
				scope = source
			}
			clog.Emit(scope, entry.Level, entry.Msg)
			continue
		}
		if scope, level, msg, ok := parseTextLevel(line, source); ok {
			clog.Emit(scope, level, msg)
			continue
		}
		clog.Emit(source, "info", line)
	}
}

func parseStructured(line string) (clog.LogEntry, bool) {
	if !strings.HasPrefix(line, "{") {
		return clog.LogEntry{}, false
	}
	var e clog.LogEntry
	if err := json.Unmarshal([]byte(line), &e); err != nil {
		return clog.LogEntry{}, false
	}
	if e.Level == "" || e.Msg == "" {
		return clog.LogEntry{}, false
	}
	return e, true
}

var textLogLevels = map[string]struct{}{
	"debug": {}, "info": {}, "warn": {}, "error": {},
}

// parseTextLevel recognizes the aide SDK text log format
// "<ts> [<level>] <scope>: <msg>" (the timestamp and scope are optional) and
// extracts the level so plugin debug/warn/error lines are not all flattened to
// info. It only matches when the first bracketed token is a known level, so
// arbitrary library output (e.g. "[1234] ..." or tracebacks) falls through.
func parseTextLevel(line, source string) (scope, level, msg string, ok bool) {
	open := strings.IndexByte(line, '[')
	if open < 0 {
		return "", "", "", false
	}
	end := strings.IndexByte(line[open:], ']')
	if end < 0 {
		return "", "", "", false
	}
	end += open

	lvl := strings.ToLower(strings.TrimSpace(line[open+1 : end]))
	if _, valid := textLogLevels[lvl]; !valid {
		return "", "", "", false
	}

	rest := strings.TrimSpace(line[end+1:])
	rest = strings.TrimPrefix(rest, source+": ")
	return source, lvl, rest, true
}
