package main

import (
	"aide/cli/internal/agent"
	"aide/cli/internal/clog"
	"aide/cli/internal/store"
	"context"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"github.com/wailsapp/wails/v3/pkg/services/notifications"
)

func runApp(url string, ag *agent.Agent, st *store.Store, shutdown context.CancelFunc) {
	opts := application.Options{
		Name:        "Aide",
		Description: "Your personal work assistant",
	}

	var notifSvc *notifications.NotificationService
	if isBundled() {
		notifSvc = notifications.New()
		opts.Services = []application.Service{application.NewService(notifSvc)}
	}

	app := application.New(opts)

	if notifSvc != nil {
		app.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(*application.ApplicationEvent) {
			granted, err := notifSvc.RequestNotificationAuthorization()
			if err != nil {
				clog.Error("notification authorization: %v", err)
				return
			}
			if granted {
				ag.SetNativeNotifier(&wailsNotifier{svc: notifSvc})
			}
		})
	}

	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:      "main",
		Title:     "Aide",
		Width:     1200,
		Height:    800,
		MinWidth:  720,
		MinHeight: 520,
		URL:       url,
	})

	// Close hides the window and keeps the app resident in the menu bar; quit
	// happens from the tray menu.
	window.OnWindowEvent(events.Common.WindowClosing, func(e *application.WindowEvent) {
		e.Cancel()
		window.Hide()
	})

	tray := app.SystemTray.New()
	tray.SetLabel("Aide")
	tray.OnClick(func() {
		window.Show()
		window.Focus()
	})

	menu := app.NewMenu()
	menu.Add("Open Aide").OnClick(func(*application.Context) {
		window.Show()
		window.Focus()
	})
	menu.AddSeparator()
	menu.Add("Quit Aide").OnClick(func(*application.Context) {
		app.Quit()
	})
	tray.SetMenu(menu)

	if enabled, err := app.Autostart.IsEnabled(); err == nil && !enabled {
		if err := app.Autostart.Enable(); err != nil {
			clog.Warn("autostart: %v", err)
		}
	}

	app.OnShutdown(func() {
		shutdown()
		st.Close()
	})

	if err := app.Run(); err != nil {
		fatal("app run: %v", err)
	}
}
