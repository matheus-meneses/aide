package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"aide/cli/internal/config"
	"aide/cli/internal/keychain"
	"aide/cli/internal/store"

	"github.com/google/uuid"
)

type Runner struct {
	cfg   *config.Config
	store *store.Store
	log   io.Writer
}

func New(cfg *config.Config, s *store.Store) *Runner {
	return &Runner{cfg: cfg, store: s, log: os.Stderr}
}

func NewWithLogger(cfg *config.Config, s *store.Store, log io.Writer) *Runner {
	return &Runner{cfg: cfg, store: s, log: log}
}

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

			items, metrics := r.partitionEntries(result)

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

			if len(result.TeamMembers) > 0 {
				members := make([]store.Member, 0, len(result.TeamMembers))
				for _, raw := range result.TeamMembers {
					members = append(members, store.Member{
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
				if err := r.store.Team.Upsert(members); err != nil {
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

	configJSON, err := json.Marshal(src.Config)
	if err != nil {
		return SourceResult{
			Source:     name,
			Error:      fmt.Errorf("marshaling config: %w", err),
			DurationMs: time.Since(start).Milliseconds(),
		}
	}

	cmd := exec.CommandContext(ctx,
		r.cfg.Settings.PythonBin,
		"-m", "framework.runner",
		name,
		"--config", string(configJSON),
	)
	cmd.Dir = r.cfg.Settings.ScrapersDir
	cmd.Env = os.Environ()
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process != nil {
			syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
		return nil
	}

	if prefix, ok := src.Config["credentials_env"].(string); ok && prefix != "" {
		cred, credErr := keychain.GetAll(name)
		if credErr == nil && cred != nil {
			for key, val := range cred.Fields {
				envKey := prefix + "_" + strings.ToUpper(key)
				cmd.Env = append(cmd.Env, envKey+"="+val)
			}
		}
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			r.streamStderr(name, stderr.String())
		}
		errMsg := err.Error()
		if stderr.Len() > 0 {
			errMsg = stderr.String()
		}
		return SourceResult{
			Source:     name,
			Error:      fmt.Errorf("%s", errMsg),
			DurationMs: time.Since(start).Milliseconds(),
			Stderr:     stderr.String(),
		}
	}

	if stderr.Len() > 0 {
		r.streamStderr(name, stderr.String())
	}

	entries, err := parseScraperOutput(stdout.Bytes())
	if err != nil {
		return SourceResult{
			Source:     name,
			Error:      fmt.Errorf("parsing output: %w", err),
			DurationMs: time.Since(start).Milliseconds(),
			Stderr:     stderr.String(),
		}
	}

	return SourceResult{
		Source:      name,
		Entries:     entries.Entries,
		TeamMembers: entries.TeamMembers,
		DurationMs:  time.Since(start).Milliseconds(),
		Stderr:      stderr.String(),
	}
}

func (r *Runner) streamStderr(source, output string) {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
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
