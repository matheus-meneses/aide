package main

import (
	agentapi "aide/cli/internal/agent/api"
	"aide/cli/internal/app"
	"aide/cli/internal/platform/clog"
	"aide/cli/internal/platform/config"
	"aide/cli/internal/setup/provision"
	"aide/cli/internal/ui/webui"
	"aide/cli/internal/ui/widgets"
	"context"
	"fmt"
	"net/http"
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

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := a.StartAutonomous(ctx); err != nil {
			clog.Error("agent stopped: %v", err)
		}
	}()

	return webui.Serve(ctx, webui.Options{Port: startPort, RegisterAPI: func(mux *http.ServeMux) {
		agentapi.Register(a, mux)
	}})
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

	fmt.Println(answer)
	return nil
}
