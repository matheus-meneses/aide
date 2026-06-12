package agent

import (
	"aide/cli/internal/config"
	"aide/cli/internal/keychain"
	"aide/cli/internal/runner"
	"aide/cli/internal/store"
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

type StatusResult struct {
	Provider     string `json:"provider"`
	LLMURL       string `json:"llm_url"`
	Model        string `json:"model"`
	RunInterval  string `json:"run_interval"`
	Briefings    string `json:"briefings"`
	LLMReachable bool   `json:"llm_reachable"`
	LLMError     string `json:"llm_error,omitempty"`
}

type Notification struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

type Agent struct {
	cfg      *config.Config
	store    *store.Store
	runner   *runner.Runner
	llm      LLM
	notifier Notifier
	tools    *ToolRegistry
	bus      *EventBus
	sessions *sessionManager

	scrapeMu sync.Mutex

	stateMu    sync.RWMutex
	lastRun    time.Time
	lastMemory string
}

func (a *Agent) setLastRun(t time.Time) {
	a.stateMu.Lock()
	a.lastRun = t
	a.stateMu.Unlock()
}

func (a *Agent) getLastRun() time.Time {
	a.stateMu.RLock()
	defer a.stateMu.RUnlock()
	return a.lastRun
}

func (a *Agent) setLastMemory(m string) {
	a.stateMu.Lock()
	a.lastMemory = m
	a.stateMu.Unlock()
}

func (a *Agent) getLastMemory() string {
	a.stateMu.RLock()
	defer a.stateMu.RUnlock()
	return a.lastMemory
}

func (a *Agent) runScrape(ctx context.Context, sources []string) (*runner.RunResult, error) {
	a.scrapeMu.Lock()
	defer a.scrapeMu.Unlock()

	result, err := a.runner.Run(ctx, sources)
	if err != nil {
		return nil, err
	}
	a.setLastRun(time.Now())
	return result, nil
}

func New(cfg *config.Config, s *store.Store, r *runner.Runner) (*Agent, error) {
	apiKey := cfg.Agent.LLMAPIKey
	if apiKey == "" {
		if cred, err := keychain.GetAll("agent"); err == nil {
			apiKey = cred.Fields["llm_api_key"]
		}
	}

	llm, err := NewLLM(cfg.Agent.LLMProvider, cfg.Agent.LLMURL, cfg.Agent.LLMModel, apiKey)
	if err != nil {
		return nil, fmt.Errorf("configuring llm: %w", err)
	}

	a := &Agent{
		cfg:      cfg,
		store:    s,
		runner:   r,
		llm:      llm,
		notifier: &NoopNotifier{},
		sessions: newSessionManager(time.Hour),
	}
	a.registerDefaultTools()

	if err := runner.SyncTeamFromConfig(cfg, s); err != nil {
		fmt.Printf("warning: team config sync: %v\n", err)
	}

	return a, nil
}

func (a *Agent) SetNotifier(n Notifier) {
	a.notifier = n
}

func (a *Agent) LLM() LLM {
	return a.llm
}

func (a *Agent) Store() *store.Store {
	return a.store
}

func (a *Agent) Config() *config.Config {
	return a.cfg
}

func (a *Agent) Runner() *runner.Runner {
	return a.runner
}

func (a *Agent) Ask(ctx context.Context, question string) (string, error) {
	a.ensureFreshData(ctx)

	sysCtx, err := BuildContext(a.store)
	if err != nil {
		return "", fmt.Errorf("building context: %w", err)
	}

	messages := []ChatMessage{
		{Role: "system", Content: sysCtx},
		{Role: "user", Content: question},
	}

	resp, usage, err := a.llm.Chat(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("llm chat: %w", err)
	}

	if usage != nil {
		if err := a.store.Tokens.Record("ask", a.llm.Model(), usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens); err != nil {
			fmt.Printf("warning: failed to record token usage: %v\n", err)
		}
	}

	return resp, nil
}

func (a *Agent) Status() (*StatusResult, error) {
	result := &StatusResult{
		Provider:    string(NormalizeProvider(a.cfg.Agent.LLMProvider)),
		LLMURL:      a.cfg.Agent.LLMURL,
		Model:       a.cfg.Agent.LLMModel,
		RunInterval: a.cfg.Agent.RunIntervalDuration().String(),
		Briefings:   strings.Join(a.cfg.Agent.BriefingTimes, ", "),
	}

	if err := a.llm.Ping(); err != nil {
		result.LLMReachable = false
		result.LLMError = err.Error()
	} else {
		result.LLMReachable = true
	}

	return result, nil
}

func (a *Agent) ensureFreshData(ctx context.Context) {
	health, err := a.store.Runs.AllHealth()
	if err != nil || len(health) == 0 {
		if _, err := a.runScrape(ctx, nil); err != nil {
			fmt.Printf("ensure fresh data: %v\n", err)
		}
		return
	}

	var mostRecent time.Time
	for _, h := range health {
		t, err := time.Parse(time.RFC3339, h.LastRun)
		if err == nil && t.After(mostRecent) {
			mostRecent = t
		}
	}

	if time.Since(mostRecent) > a.cfg.Agent.RunIntervalDuration() {
		if _, err := a.runScrape(ctx, nil); err != nil {
			fmt.Printf("ensure fresh data: %v\n", err)
		}
	}
}
