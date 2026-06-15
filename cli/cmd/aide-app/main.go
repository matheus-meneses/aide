package main

import (
	agentapi "aide/cli/internal/agent/api"
	"aide/cli/internal/app"
	"aide/cli/internal/platform/clog"
	"aide/cli/internal/platform/config"
	"aide/cli/internal/platform/xdg"
	"aide/cli/internal/setup/bootstrap"
	"aide/cli/internal/ui/webui"
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

var version = "dev"

func fatal(format string, args ...any) {
	clog.Error(format, args...)
	os.Exit(1)
}

func main() {
	if err := bootstrap.EnsureConfigScaffold(); err != nil {
		fatal("preparing aide home: %v", err)
	}

	cfgPath := bootstrap.ConfigPath()
	cfg, err := config.Load(cfgPath)
	if err != nil {
		fatal("loading config: %v", err)
	}

	level, format := clog.Resolve("", "", cfg.Settings.LogLevel, cfg.Settings.LogFormat)
	clog.Configure(level, format)

	logPath := filepath.Join(xdg.AideHome(), "logs", "aide.log")
	if err := clog.SetFile(logPath); err != nil {
		fatal("opening log file: %v", err)
	}

	stk, err := app.New(cfg, level, format, version)
	if err != nil {
		fatal("%v", err)
	}
	ag := stk.Agent
	st := stk.Store
	ag.SetConfigPath(cfgPath)
	ag.SetNativeNotifications(true)

	port, err := freePort()
	if err != nil {
		stk.Close()
		fatal("finding port: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		if err := ag.StartAutonomous(ctx); err != nil {
			clog.Error("agent stopped: %v", err)
		}
	}()
	go func() {
		if err := webui.Serve(ctx, webui.Options{Port: port, NoBrowser: true, LogFile: logPath, RegisterAPI: func(mux *http.ServeMux) {
			agentapi.Register(ag, mux)
		}}); err != nil {
			clog.Error("web server stopped: %v", err)
		}
	}()

	url := fmt.Sprintf("http://localhost:%d", port)
	waitForServer(url, 15*time.Second)

	runApp(url, ag, st, cancel)
}

func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func waitForServer(url string, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: time.Second}
	for time.Now().Before(deadline) {
		resp, err := client.Get(url + "/api/version")
		if err == nil {
			resp.Body.Close()
			return
		}
		time.Sleep(150 * time.Millisecond)
	}
}
