package runner

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

func (r *Runner) logf(format string, args ...any) {
	fmt.Fprintf(r.log, "  "+format+"\n", args...)
}

func (r *Runner) logLine(level, msg string) {
	levelValues := map[string]int{"debug": 10, "info": 20, "warn": 30, "error": 40}
	threshold := levelValues[r.logLevel]
	if threshold == 0 {
		threshold = 20
	}
	if levelValues[level] < threshold {
		return
	}
	ts := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	if r.logFormat == "json" {
		type rec struct {
			TS    string `json:"ts"`
			Level string `json:"level"`
			Scope string `json:"scope"`
			Msg   string `json:"msg"`
		}
		b, _ := json.Marshal(rec{TS: ts, Level: level, Scope: "runner", Msg: msg})
		fmt.Fprintln(r.log, string(b))
	} else {
		fmt.Fprintf(r.log, "%s [%s] runner: %s\n", ts, level, msg)
	}
}

func (r *Runner) debugf(format string, args ...any) {
	r.logLine("debug", fmt.Sprintf(format, args...))
}

func (r *Runner) infof(format string, args ...any) {
	r.logLine("info", fmt.Sprintf(format, args...))
}

func (r *Runner) errorf(format string, args ...any) {
	r.logLine("error", fmt.Sprintf(format, args...))
}

func (r *Runner) streamStderr(source, output string) {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if r.logFormat == "json" {
			fmt.Fprintln(r.log, line)
		} else {
			r.logf("[%s] %s", source, line)
		}
	}
}
