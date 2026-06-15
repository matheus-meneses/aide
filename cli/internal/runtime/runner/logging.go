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
// structured JSON keep their original scope and level; anything else is tagged
// with the source name at info level.
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
