package main

import (
	"aide/cli/internal/app"
	"aide/cli/internal/platform/config"
	"aide/cli/internal/setup/provision"
	"aide/cli/internal/ui/widgets"
	"context"
	"fmt"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
)

var (
	scheduleInterval  string
	scheduleBriefings string
	contextSource     string
)

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Local autonomous assistant agent",
}

var agentStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Run the autonomous agent in the foreground (headless, no web UI)",
	Long: `Run the autonomous agent loop in the foreground until interrupted
(Ctrl-C / SIGTERM). No HTTP server is started and no browser is opened — use
'aide ui' for the full web experience. Background it yourself via launchd,
systemd, or a trailing '&'.`,
	RunE: agentStartExecute,
}

var agentStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check LLM reachability and show agent config",
	RunE:  agentStatusExecute,
}

var agentAskCmd = &cobra.Command{
	Use:   "ask [question]",
	Short: "Ask a one-shot question about your data",
	Args:  cobra.MinimumNArgs(1),
	RunE:  agentAskExecute,
}

var agentScheduleCmd = &cobra.Command{
	Use:   "schedule",
	Short: "Set the run interval and briefing times non-interactively",
	RunE:  agentScheduleExecute,
}

var agentContextCmd = &cobra.Command{
	Use:   "context [text]",
	Short: "View or edit the context that shapes the assistant",
	Long: `Manage the free-text context that is injected into the assistant's prompts.

Without arguments it shows the current context. Provide text to set it, or use
'clear' to remove it. Use --source to target a specific source's guidance
instead of your personal context.

Examples:
  aide agent context
  aide agent context "I'm a tech lead; prioritize incidents and PR reviews."
  aide agent context --source jira "Only surface tickets assigned to me."
  aide agent context clear --source jira`,
	Args: cobra.ArbitraryArgs,
	RunE: agentContextExecute,
}

func init() {
	agentCmd.AddCommand(agentStartCmd)
	agentCmd.AddCommand(agentStatusCmd)
	agentCmd.AddCommand(agentAskCmd)
	agentCmd.AddCommand(agentConfigCmd)
	agentCmd.AddCommand(agentScheduleCmd)
	agentCmd.AddCommand(agentContextCmd)
	agentScheduleCmd.Flags().StringVar(&scheduleInterval, "interval", "", "how often the agent re-collects (e.g. 30m, 1h)")
	agentScheduleCmd.Flags().StringVar(&scheduleBriefings, "briefings", "", "comma-separated daily briefing times (24h, e.g. 08:00,17:30)")
	agentContextCmd.Flags().StringVar(&contextSource, "source", "", "target a source's guidance instead of your personal context")
	rootCmd.AddCommand(agentCmd)
}

func agentContextExecute(_ *cobra.Command, args []string) error {
	text := strings.TrimSpace(strings.Join(args, " "))

	if len(args) == 0 {
		return showContext()
	}

	if strings.EqualFold(text, "clear") {
		text = ""
	}

	if contextSource != "" {
		if err := provision.SetSourceContext(cfgFile, contextSource, text); err != nil {
			return err
		}
	} else if err := provision.SetUserContext(cfgFile, text); err != nil {
		return err
	}

	if text == "" {
		widgets.PrintSuccess("Context cleared.")
	} else {
		widgets.PrintSuccess("Context updated.")
	}
	return nil
}

func showContext() error {
	snap, err := provision.ConfigSnapshot(cfgFile)
	if err != nil {
		return err
	}

	if contextSource != "" {
		for _, src := range snap.Sources {
			if src.Name == contextSource {
				printContextValue(fmt.Sprintf("Context for source %q", contextSource), src.Context)
				return nil
			}
		}
		return fmt.Errorf("source %q not configured", contextSource)
	}

	printContextValue("Your context", snap.Agent.UserContext)
	var withCtx []provision.SourceSnapshot
	for _, src := range snap.Sources {
		if strings.TrimSpace(src.Context) != "" {
			withCtx = append(withCtx, src)
		}
	}
	if len(withCtx) > 0 {
		widgets.Println()
		widgets.Println("Per-source guidance:")
		for _, src := range withCtx {
			widgets.Printf("  %s: %s\n", src.Name, src.Context)
		}
	}
	return nil
}

func printContextValue(label, value string) {
	if strings.TrimSpace(value) == "" {
		widgets.Printf("%s: (none)\n", label)
		return
	}
	widgets.Printf("%s:\n%s\n", label, value)
}

func agentScheduleExecute(cmd *cobra.Command, _ []string) error {
	if !cmd.Flags().Changed("interval") && !cmd.Flags().Changed("briefings") {
		return fmt.Errorf("provide --interval and/or --briefings")
	}
	in := provision.ScheduleInput{}
	if cmd.Flags().Changed("interval") {
		in.RunInterval = scheduleInterval
	}
	if cmd.Flags().Changed("briefings") {
		in.BriefingTimes = parseBriefingTimes(scheduleBriefings)
	}
	if err := provision.SetSchedule(cfgFile, in); err != nil {
		return err
	}
	widgets.PrintSuccess("Schedule updated.")
	return nil
}

func newAgent(cfg *config.Config) (*app.Stack, error) {
	return app.New(cfg, logLevel(), logFormatValue(), version)
}

func agentStartExecute(_ *cobra.Command, _ []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	if cfg.Agent.LLMModel == "" || cfg.Agent.LLMURL == "" {
		widgets.PrintWarn("No AI model configured — autonomous runs are paused. Set one with: aide agent config")
	}

	stk, err := newAgent(cfg)
	if err != nil {
		return err
	}
	defer stk.Close()
	a := stk.Agent
	a.SetConfigPath(cfgFile)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	widgets.PrintInfo("Autonomous agent running. Press Ctrl-C to stop.")
	return a.StartAutonomous(ctx)
}

func agentStatusExecute(_ *cobra.Command, _ []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	stk, err := newAgent(cfg)
	if err != nil {
		return err
	}
	defer stk.Close()

	result, err := stk.Agent.Status()
	if err != nil {
		return err
	}

	widgets.Printf("Provider:     %s\n", result.Provider)
	widgets.Printf("LLM URL:      %s\n", result.LLMURL)
	widgets.Printf("Model:        %s\n", result.Model)
	widgets.Printf("Run interval: %s\n", result.RunInterval)
	widgets.Printf("Briefings:    %s\n", result.Briefings)
	widgets.Println()
	if !result.LLMReachable {
		widgets.Printf("LLM: UNREACHABLE (%s)\n", result.LLMError)
		return fmt.Errorf("LLM unreachable: %s", result.LLMError)
	}
	widgets.Println("LLM: OK")
	return nil
}

func agentAskExecute(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	stk, err := newAgent(cfg)
	if err != nil {
		return err
	}
	defer stk.Close()

	question := strings.Join(args, " ")
	answer, err := stk.Agent.Ask(cmd.Context(), question)
	if err != nil {
		return err
	}

	widgets.Println(answer)
	return nil
}
