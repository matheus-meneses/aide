package runner

import (
	"aide/cli/internal/config"
	"aide/cli/internal/plugin"
	"aide/cli/internal/store"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Runner struct {
	cfg       *config.Config
	store     *store.Store
	log       io.Writer
	logLevel  string
	logFormat string
}

func New(cfg *config.Config, s *store.Store) *Runner {
	return &Runner{cfg: cfg, store: s, log: os.Stderr, logLevel: "info", logFormat: "text"}
}

func NewWithLogger(cfg *config.Config, s *store.Store, log io.Writer) *Runner {
	return &Runner{cfg: cfg, store: s, log: log, logLevel: "info", logFormat: "text"}
}

func (r *Runner) SetLogLevel(level string)   { r.logLevel = level }
func (r *Runner) SetLogFormat(format string) { r.logFormat = format }

func (r *Runner) Run(ctx context.Context, filterSources []string) (*RunResult, error) {
	sources := r.resolveSources(filterSources)
	if len(sources) == 0 {
		return nil, fmt.Errorf("no enabled sources to run")
	}

	runID := uuid.New().String()
	startedAt := time.Now().UTC()

	run := store.Run{
		ID:        runID,
		StartedAt: startedAt.Format(time.RFC3339),
	}
	if err := r.store.Runs.Insert(run); err != nil {
		return nil, fmt.Errorf("recording run start: %w", err)
	}

	resultsChan := make(chan SourceResult, len(sources))
	sem := make(chan struct{}, r.cfg.Settings.Concurrency)
	var wg sync.WaitGroup

	for name, src := range sources {
		wg.Add(1)
		go func(sourceName string, sourceCfg config.Source) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			result := r.executeSource(ctx, sourceName, sourceCfg)
			resultsChan <- result
		}(name, src)
	}

	wg.Wait()
	close(resultsChan)

	runResult := &RunResult{
		RunID:        runID,
		SourcesTotal: len(sources),
	}

	for result := range resultsChan {
		health := store.SourceHealth{
			Source:     result.Source,
			LastRun:    time.Now().UTC().Format(time.RFC3339),
			DurationMs: result.DurationMs,
			RunID:      runID,
		}

		if result.Error != nil {
			runResult.SourcesFailed++
			health.Status = "error"
			health.ErrorMessage = result.Error.Error()
		} else {
			runResult.SourcesOK++
			health.Status = "ok"
			health.EntriesCount = len(result.Entries)

			var items []store.Item
			var metrics []metricEntry
			var members []store.Member

			if result.PluginResp != nil {
				items, metrics, members = r.normalizeResponse(result.Source, result.PluginResp)
			} else {
				items, metrics = r.partitionEntries(result)
			}

			newCount, _, upsertErr := r.store.Items.Upsert(result.Source, items)
			if upsertErr != nil {
				health.Status = "error"
				health.ErrorMessage = fmt.Sprintf("store error: %v", upsertErr)
				runResult.SourcesFailed++
				runResult.SourcesOK--
			} else {
				result.NewItems = newCount
			}

			for _, m := range metrics {
				if err := r.store.Metrics.Record(result.Source, m.name, m.value); err != nil {
					r.logf("[%s] metric store error: %v", result.Source, err)
				}
			}

			if len(members) > 0 {
				if err := r.store.Team.Upsert(members); err != nil {
					r.logf("[%s] team upsert error: %v", result.Source, err)
				}
			} else if len(result.TeamMembers) > 0 {
				legacy := make([]store.Member, 0, len(result.TeamMembers))
				for _, raw := range result.TeamMembers {
					legacy = append(legacy, store.Member{
						Name:         raw.Name,
						Email:        raw.Email,
						Role:         raw.Role,
						Department:   raw.Department,
						Branch:       raw.Branch,
						Registration: raw.Registration,
						ManagerRef:   raw.ManagerRegistration,
						Source:       result.Source,
					})
				}
				if err := r.store.Team.Upsert(legacy); err != nil {
					r.logf("[%s] team upsert error: %v", result.Source, err)
				}
			}
		}

		runResult.Results = append(runResult.Results, result)
		if err := r.store.Runs.UpsertHealth(health); err != nil {
			r.logf("[%s] health upsert error: %v", result.Source, err)
		}
	}

	finishedAt := time.Now().UTC()
	run.FinishedAt = finishedAt.Format(time.RFC3339)
	run.SourcesTotal = runResult.SourcesTotal
	run.SourcesOK = runResult.SourcesOK
	run.SourcesFailed = runResult.SourcesFailed
	if err := r.store.Runs.Update(run); err != nil {
		r.logf("failed to update run record: %v", err)
	}

	return runResult, nil
}

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

func (r *Runner) resolveSources(filter []string) map[string]config.Source {
	all := r.cfg.EnabledSources()
	if len(filter) == 0 {
		return all
	}

	filtered := make(map[string]config.Source)
	for _, name := range filter {
		if src, ok := all[name]; ok {
			filtered[name] = src
		}
	}
	return filtered
}

func (r *Runner) ValidateFilter(filter []string) error {
	if len(filter) == 0 {
		return nil
	}
	all := r.cfg.EnabledSources()
	var unknown []string
	for _, name := range filter {
		if _, ok := all[name]; !ok {
			unknown = append(unknown, name)
		}
	}
	if len(unknown) > 0 {
		return fmt.Errorf("unknown or disabled sources: %s", strings.Join(unknown, ", "))
	}
	return nil
}

func (r *Runner) executeSource(ctx context.Context, name string, src config.Source) SourceResult {
	timeout := time.Duration(r.cfg.Settings.TimeoutSeconds) * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()

	mgr := plugin.NewManager()
	pluginName := src.Plugin
	if pluginName == "" {
		pluginName = name
	}
	m, err := mgr.Get(pluginName)
	if err != nil {
		return SourceResult{
			Source:     name,
			Error:      fmt.Errorf("plugin %q not installed: %w", pluginName, err),
			DurationMs: time.Since(start).Milliseconds(),
		}
	}

	secrets, _ := plugin.ScopedSecrets(name, m)

	req := &plugin.Request{
		Action:  "scrape",
		Config:  src.Config,
		Secrets: secrets,
		Context: map[string]any{
			"data_dir":   r.cfg.Settings.DataDir,
			"log_level":  r.logLevel,
			"log_format": r.logFormat,
		},
	}

	r.debugf("scraping %s via plugin %s", name, pluginName)
	r.infof("starting %s", name)

	resp, stderr, err := plugin.Execute(ctx, m, req)
	if stderr != "" {
		r.streamStderr(name, stderr)
	}
	if err != nil {
		r.errorf("%s failed: %v", name, err)
		return SourceResult{
			Source:     name,
			Error:      err,
			DurationMs: time.Since(start).Milliseconds(),
			Stderr:     stderr,
		}
	}
	if !resp.OK {
		msg := resp.Error
		if msg == "" {
			msg = "plugin returned ok=false"
		}
		r.errorf("%s failed: %s", name, msg)
		return SourceResult{
			Source:     name,
			Error:      fmt.Errorf("%s", msg),
			DurationMs: time.Since(start).Milliseconds(),
			Stderr:     stderr,
		}
	}

	entries := make([]ScraperEntry, 0, len(resp.Entries))
	for _, e := range resp.Entries {
		entries = append(entries, ScraperEntry{
			Source:    name,
			Member:    e.Member,
			Category:  e.Category,
			Title:     e.Title,
			Detail:    e.Detail,
			EntryDate: e.EntryDate,
			Priority:  e.Priority,
			Metadata:  e.Metadata,
		})
	}

	for _, met := range resp.Metrics {
		entries = append(entries, ScraperEntry{
			Source:   name,
			Category: "metric",
			Title:    met.Name,
			Metadata: map[string]any{"mode": "metric", "metric_value": met.Value},
		})
	}

	teamMembers := make([]TeamMemberRaw, 0, len(resp.TeamMembers))
	for _, t := range resp.TeamMembers {
		teamMembers = append(teamMembers, TeamMemberRaw{
			Name:                t.Name,
			Email:               t.Email,
			Role:                t.Role,
			Department:          t.Department,
			Branch:              t.Branch,
			Registration:        t.Registration,
			ManagerRegistration: t.ManagerRegistration,
		})
	}

	result := SourceResult{
		Source:      name,
		Entries:     entries,
		TeamMembers: teamMembers,
		PluginResp:  resp,
		DurationMs:  time.Since(start).Milliseconds(),
		Stderr:      stderr,
	}
	r.debugf("%s finished in %dms (entries=%d)", name, result.DurationMs, len(entries))
	r.infof("%s done in %dms (%d entries)", name, result.DurationMs, len(entries))
	return result
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

type metricEntry struct {
	name  string
	value float64
}

func (r *Runner) partitionEntries(result SourceResult) ([]store.Item, []metricEntry) {
	var items []store.Item
	var metrics []metricEntry

	for _, e := range result.Entries {
		mode := ""
		if e.Metadata != nil {
			if m, ok := e.Metadata["mode"].(string); ok {
				mode = m
			}
		}

		if mode == "metric" {
			value := 0.0
			if e.Metadata != nil {
				if v, ok := e.Metadata["metric_value"].(float64); ok {
					value = v
				}
			}
			metrics = append(metrics, metricEntry{name: e.Title, value: value})
			continue
		}

		member := r.cfg.ResolveMember(e.Member)
		link := ""
		if e.Metadata != nil {
			if url, ok := e.Metadata["web_url"].(string); ok {
				link = url
			}
		}
		fp := store.Fingerprint(e.Source, link, e.Title, member)
		items = append(items, store.Item{
			Fingerprint: fp,
			Source:      e.Source,
			Member:      member,
			Category:    e.Category,
			Title:       e.Title,
			Detail:      e.Detail,
			EntryDate:   e.EntryDate,
			Priority:    e.Priority,
			Link:        link,
		})
	}
	return items, metrics
}
