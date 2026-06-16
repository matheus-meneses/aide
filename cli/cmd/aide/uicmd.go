package main

import (
	"aide/cli/internal/platform/clog"
	"aide/cli/internal/platform/config"
	"aide/cli/internal/setup/bootstrap"
	"aide/cli/internal/ui/webui"
	"context"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"

	agentapi "aide/cli/internal/agent/api"

	"github.com/spf13/cobra"
)

var (
	uiPort      int
	uiNoBrowser bool
)

var uiCmd = &cobra.Command{
	Use:   "ui",
	Short: "Launch the web UI and run the autonomous agent",
	Long: `Serve the aide web UI and run the autonomous agent loop, mirroring the
desktop app. It works without a prior 'aide init': if the agent is not yet
configured, the in-browser setup wizard guides you through it, and the
autonomous loop starts as soon as a model is set.`,
	RunE: uiExecute,
}

func init() {
	uiCmd.Flags().IntVarP(&uiPort, "port", "p", 8531, "web UI port")
	uiCmd.Flags().BoolVar(&uiNoBrowser, "no-browser", false, "do not open a browser window")
	rootCmd.AddCommand(uiCmd)
}

func uiExecute(_ *cobra.Command, _ []string) error {
	if err := bootstrap.EnsureConfigScaffold(); err != nil {
		return fmt.Errorf("preparing aide home: %w", err)
	}

	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	return serveUI(cfg, cfgFile, uiPort, uiNoBrowser)
}

func serveUI(cfg *config.Config, cfgPath string, port int, noBrowser bool) error {
	stk, err := newAgent(cfg)
	if err != nil {
		return err
	}
	defer stk.Close()

	a := stk.Agent
	a.SetConfigPath(cfgPath)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := a.StartAutonomous(ctx); err != nil {
			clog.Error("agent stopped: %v", err)
		}
	}()

	return webui.Serve(ctx, webui.Options{Port: port, NoBrowser: noBrowser, RegisterAPI: func(mux *http.ServeMux) {
		agentapi.Register(a, mux)
	}})
}
