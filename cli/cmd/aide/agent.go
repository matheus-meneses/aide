package main

import (
	"aide/cli/internal/agent"
	"aide/cli/internal/config"
	"aide/cli/internal/runner"
	"aide/cli/internal/store"
	"context"
	"fmt"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
)

var startPort int

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

func init() {
	agentCmd.AddCommand(agentStartCmd)
	agentCmd.AddCommand(agentStatusCmd)
	agentCmd.AddCommand(agentAskCmd)
	agentStartCmd.Flags().IntVarP(&startPort, "port", "p", 8531, "Web UI port")
	rootCmd.AddCommand(agentCmd)
}

func newAgent(cfg *config.Config) (*agent.Agent, *store.Store, error) {
	s, err := store.Open(cfg.Settings.DataDir)
	if err != nil {
		return nil, nil, fmt.Errorf("opening store: %w", err)
	}

	r := runner.New(cfg, s)
	a := agent.New(cfg, s, r)
	return a, s, nil
}

func agentStartExecute(_ *cobra.Command, _ []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	agent.Version = version

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
