package runner

import (
	"aide/cli/internal/config"
	"aide/cli/internal/store"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Runner struct {
	cfg               *config.Config
	store             *store.Store
	log               io.Writer
	logLevel          string
	logFormat         string
	tlsVerifyOverride *bool
	caBundleOverride  *string
}

func New(cfg *config.Config, s *store.Store) *Runner {
	return &Runner{cfg: cfg, store: s, log: os.Stderr, logLevel: "info", logFormat: "text"}
}

func NewWithLogger(cfg *config.Config, s *store.Store, log io.Writer) *Runner {
	return &Runner{cfg: cfg, store: s, log: log, logLevel: "info", logFormat: "text"}
}

func (r *Runner) SetConfig(cfg *config.Config)     { r.cfg = cfg }
func (r *Runner) SetLogLevel(level string)         { r.logLevel = level }
func (r *Runner) SetLogFormat(format string)       { r.logFormat = format }
func (r *Runner) SetVerifySSLOverride(verify bool) { r.tlsVerifyOverride = &verify }
func (r *Runner) SetCABundleOverride(path string)  { r.caBundleOverride = &path }

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
