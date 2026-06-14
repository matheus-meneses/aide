package main

import (
	"aide/cli/internal/agent"
	"aide/cli/internal/bootstrap"
	"aide/cli/internal/clog"
	"aide/cli/internal/config"
	"aide/cli/internal/runner"
	"aide/cli/internal/store"
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
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

	st, err := store.Open(cfg.Settings.DataDir)
	if err != nil {
		fatal("opening store: %v", err)
	}

	r := runner.New(cfg, st)
	r.SetLogLevel(level)
	r.SetLogFormat(format)
	ag, err := agent.New(cfg, st, r)
	if err != nil {
		st.Close()
		fatal("creating agent: %v", err)
	}
	agent.Version = version
	ag.SetNoBrowser(true)
	ag.SetConfigPath(cfgPath)
	ag.SetNativeNotifications(true)

	port, err := freePort()
	if err != nil {
		st.Close()
		fatal("finding port: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		if err := ag.StartAutonomous(ctx, port); err != nil {
			clog.Error("agent stopped: %v", err)
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
