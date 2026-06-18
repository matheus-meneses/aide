package agent

import (
	"aide/cli/internal/agent/events"
	"aide/cli/internal/agent/llm"
	"aide/cli/internal/agent/tools"
	"aide/cli/internal/notification"
	"aide/cli/internal/persistence/store"
	"aide/cli/internal/platform/clog"
	"aide/cli/internal/platform/config"
	"aide/cli/internal/runtime/runner"
	"aide/cli/internal/security/keychain"
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

var alog = clog.New("agent")

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
	cfgMu      sync.RWMutex
	cfg        *config.Config
	llm        llm.LLM
	configPath string

	store    *store.Store
	runner   *runner.Runner
	scraper  Scraper
	notifier notification.Notifier
	tools    *tools.ToolRegistry
	bus      *events.EventBus
	sessions *sessionManager
	clock    Clock

	scrapeMu sync.Mutex

	schedMu          sync.Mutex
	autoCtx          context.Context
	reschedule       chan struct{}
	briefingStarted  bool
	autoCycleStarted bool
	idleLogged       bool

	stateMu             sync.RWMutex
	lastRun             time.Time
	lastMemory          string
	nativeNotifications bool
	restartHandler      func()
}

// SetRestartHandler registers a callback the host (the desktop app) uses to quit
// itself so a staged in-place update can swap the bundle and relaunch.
func (a *Agent) SetRestartHandler(fn func()) {
	a.stateMu.Lock()
	a.restartHandler = fn
	a.stateMu.Unlock()
}

// RequestRestart invokes the registered restart handler, if any, and reports
// whether one was set.
func (a *Agent) RequestRestart() bool {
	a.stateMu.RLock()
	fn := a.restartHandler
	a.stateMu.RUnlock()
	if fn == nil {
		return false
	}
	go fn()
	return true
}

// SetNativeNotifications marks that the host (e.g. the desktop app) delivers OS
// notifications, so the web UI can stop using the browser Notification API.
func (a *Agent) SetNativeNotifications(v bool) {
	a.stateMu.Lock()
	a.nativeNotifications = v
	a.stateMu.Unlock()
}

func (a *Agent) NativeNotifications() bool {
	a.stateMu.RLock()
	defer a.stateMu.RUnlock()
	return a.nativeNotifications
}

// SetNativeNotifier routes OS-level notifications through the given notifier and
// flags native delivery so the web UI suppresses its own browser notifications.
// The in-app activity feed is fed separately by explicit bus events at each call
// site, so the notifier here is OS-only to avoid duplicate feed entries.
func (a *Agent) SetNativeNotifier(n notification.Notifier) {
	a.SetNotifier(n)
	a.SetNativeNotifications(true)
}

func (a *Agent) SetConfigPath(p string) {
	a.cfgMu.Lock()
	a.configPath = p
	a.cfgMu.Unlock()
}

func (a *Agent) configPathOrDefault() string {
	a.cfgMu.RLock()
	p := a.configPath
	a.cfgMu.RUnlock()
	if p != "" {
		return p
	}
	return config.DefaultConfigPath()
}

// ConfigPath returns the active config path, used by the HTTP API adapter.
func (a *Agent) ConfigPath() string {
	return a.configPathOrDefault()
}

// StoredAPIKey returns the configured LLM API key, falling back to the keychain.
func (a *Agent) StoredAPIKey() string {
	if cfg := a.getConfig(); cfg != nil && cfg.Agent.LLMAPIKey != "" {
		return cfg.Agent.LLMAPIKey
	}
	if cred, err := keychain.GetAll("agent"); err == nil {
		return cred.Fields["llm_api_key"]
	}
	return ""
}

func (a *Agent) getConfig() *config.Config {
	a.cfgMu.RLock()
	defer a.cfgMu.RUnlock()
	return a.cfg
}

func (a *Agent) getLLM() llm.LLM {
	a.cfgMu.RLock()
	defer a.cfgMu.RUnlock()
	return a.llm
}

func (a *Agent) ReloadConfig() error {
	cfg, err := config.Load(a.configPathOrDefault())
	if err != nil {
		return fmt.Errorf("reloading config: %w", err)
	}

	apiKey := cfg.Agent.LLMAPIKey
	if apiKey == "" {
		if cred, err := keychain.GetAll("agent"); err == nil {
			apiKey = cred.Fields["llm_api_key"]
		}
	}

	client, err := llm.NewLLM(cfg.Agent.LLMProvider, cfg.Agent.LLMURL, cfg.Agent.LLMModel, apiKey)
	if err != nil {
		return fmt.Errorf("configuring llm: %w", err)
	}

	a.cfgMu.Lock()
	a.cfg = cfg
	a.llm = client
	a.cfgMu.Unlock()

	level, format := clog.Resolve("", "", cfg.Settings.LogLevel, cfg.Settings.LogFormat)
	clog.Configure(level, format)

	a.scrapeMu.Lock()
	if a.runner != nil {
		a.runner.SetConfig(cfg)
		a.runner.SetLogLevel(level)
		a.runner.SetLogFormat(format)
	}
	a.scrapeMu.Unlock()

	a.registerDefaultTools()

	a.signalReschedule()
	return nil
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

	if err := a.scraper.ValidateFilter(sources); err != nil {
		return nil, err
	}

	result, err := a.scraper.Run(ctx, sources)
	if err != nil {
		return nil, err
	}
	a.setLastRun(a.clock.Now())
	return result, nil
}

func New(cfg *config.Config, s *store.Store, r *runner.Runner) (*Agent, error) {
	apiKey := cfg.Agent.LLMAPIKey
	if apiKey == "" {
		if cred, err := keychain.GetAll("agent"); err == nil {
			apiKey = cred.Fields["llm_api_key"]
		}
	}

	client, err := llm.NewLLM(cfg.Agent.LLMProvider, cfg.Agent.LLMURL, cfg.Agent.LLMModel, apiKey)
	if err != nil {
		return nil, fmt.Errorf("configuring llm: %w", err)
	}

	clk := realClock{}
	a := &Agent{
		cfg:      cfg,
		store:    s,
		runner:   r,
		scraper:  r,
		llm:      client,
		notifier: &notification.MacNotifier{},
		sessions: newSessionManager(time.Hour, clk),
		clock:    clk,
	}
	a.bus = events.NewEventBus()
	a.registerDefaultTools()

	return a, nil
}

func (a *Agent) SetNotifier(n notification.Notifier) {
	a.notifier = n
}

func TestLLM(provider, baseURL, model, apiKey string) error {
	client, err := llm.NewLLM(provider, baseURL, model, apiKey)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	reply, _, err := client.Chat(ctx, []llm.ChatMessage{
		{Role: "user", Content: "Reply with a single short word to confirm you are reachable."},
	})
	if err != nil {
		return fmt.Errorf("model %q did not respond: %w", client.Model(), err)
	}
	if strings.TrimSpace(reply) == "" {
		return fmt.Errorf("model %q returned an empty response", client.Model())
	}
	return nil
}

func (a *Agent) LLM() llm.LLM {
	return a.getLLM()
}

func (a *Agent) Store() *store.Store {
	return a.store
}

func (a *Agent) Config() *config.Config {
	return a.getConfig()
}

func (a *Agent) Runner() *runner.Runner {
	return a.runner
}

func (a *Agent) Ask(ctx context.Context, question string) (string, error) {
	a.ensureFreshData(ctx)

	sysCtx, err := BuildContext(a.store, a.clock.Now())
	if err != nil {
		return "", fmt.Errorf("building context: %w", err)
	}

	messages := []llm.ChatMessage{
		{Role: "system", Content: sysCtx},
		{Role: "user", Content: question},
	}

	client := a.getLLM()
	resp, usage, err := client.Chat(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("llm chat: %w", err)
	}

	if usage != nil {
		if err := a.store.Tokens.Record("ask", client.Model(), usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens); err != nil {
			alog.Warn("failed to record token usage: %v", err)
		}
	}

	return resp, nil
}

func (a *Agent) Status() (*StatusResult, error) {
	cfg := a.getConfig()
	result := &StatusResult{
		Provider:    string(llm.NormalizeProvider(cfg.Agent.LLMProvider)),
		LLMURL:      cfg.Agent.LLMURL,
		Model:       cfg.Agent.LLMModel,
		RunInterval: cfg.Agent.RunIntervalDuration().String(),
		Briefings:   strings.Join(cfg.Agent.BriefingTimes, ", "),
	}

	if err := a.getLLM().Ping(); err != nil {
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
			alog.Error("ensure fresh data: %v", err)
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

	if a.clock.Now().Sub(mostRecent) > a.getConfig().Agent.RunIntervalDuration() {
		if _, err := a.runScrape(ctx, nil); err != nil {
			alog.Error("ensure fresh data: %v", err)
		}
	}
}
