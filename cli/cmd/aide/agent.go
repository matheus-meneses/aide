package main

import (
	"aide/cli/internal/agent"
	"aide/cli/internal/config"
	"aide/cli/internal/provision"
	"aide/cli/internal/runner"
	"aide/cli/internal/store"
	"aide/cli/internal/ui"
	"context"
	"fmt"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
)

var startPort int

var (
	scheduleInterval  string
	scheduleBriefings string
)

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Local autonomous assistant agent",
}

var agentStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start autonomous agent with web UI",
	RunE:  agentStartExecute,
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

func init() {
	agentCmd.AddCommand(agentStartCmd)
	agentCmd.AddCommand(agentStatusCmd)
	agentCmd.AddCommand(agentAskCmd)
	agentCmd.AddCommand(agentConfigCmd)
	agentCmd.AddCommand(agentScheduleCmd)
	agentStartCmd.Flags().IntVarP(&startPort, "port", "p", 8531, "Web UI port")
	agentScheduleCmd.Flags().StringVar(&scheduleInterval, "interval", "", "how often the agent re-collects (e.g. 30m, 1h)")
	agentScheduleCmd.Flags().StringVar(&scheduleBriefings, "briefings", "", "comma-separated daily briefing times (24h, e.g. 08:00,17:30)")
	rootCmd.AddCommand(agentCmd)
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
	ui.PrintSuccess("Schedule updated.")
	return nil
}

func newAgent(cfg *config.Config) (*agent.Agent, *store.Store, error) {
	s, err := store.Open(cfg.Settings.DataDir)
	if err != nil {
		return nil, nil, fmt.Errorf("opening store: %w", err)
	}

	r := runner.New(cfg, s)
	r.SetLogLevel(logLevel())
	r.SetLogFormat(logFormatValue())
	a, err := agent.New(cfg, s, r)
	if err != nil {
		s.Close()
		return nil, nil, err
	}
	return a, s, nil
}

func agentStartExecute(_ *cobra.Command, _ []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	agent.Version = version

	if cfg.Agent.LLMModel == "" || cfg.Agent.LLMURL == "" {
		ui.PrintWarn("No AI model configured — autonomous runs are paused. Set one with: aide agent config")
	}

	a, s, err := newAgent(cfg)
	if err != nil {
		return err
	}
	defer s.Close()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	return a.StartAutonomous(ctx, startPort)
}

func agentStatusExecute(_ *cobra.Command, _ []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	a, s, err := newAgent(cfg)
	if err != nil {
		return err
	}
	defer s.Close()

	result, err := a.Status()
	if err != nil {
		return err
	}

	fmt.Printf("Provider:     %s\n", result.Provider)
	fmt.Printf("LLM URL:      %s\n", result.LLMURL)
	fmt.Printf("Model:        %s\n", result.Model)
	fmt.Printf("Run interval: %s\n", result.RunInterval)
	fmt.Printf("Briefings:    %s\n", result.Briefings)
	fmt.Println()
	if !result.LLMReachable {
		fmt.Printf("LLM: UNREACHABLE (%s)\n", result.LLMError)
		return fmt.Errorf("LLM unreachable: %s", result.LLMError)
	}
	fmt.Println("LLM: OK")
	return nil
}

func agentAskExecute(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	a, s, err := newAgent(cfg)
	if err != nil {
		return err
	}
	defer s.Close()

	question := strings.Join(args, " ")
	answer, err := a.Ask(cmd.Context(), question)
	if err != nil {
		return err
	}

	fmt.Println(answer)
	return nil
}
