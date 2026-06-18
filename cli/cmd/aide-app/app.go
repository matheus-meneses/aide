//go:build cgo

package main

import (
	"aide/cli/internal/agent"
	"aide/cli/internal/persistence/store"
	"aide/cli/internal/platform/clog"
	"context"
	_ "embed"
	"encoding/json"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"github.com/wailsapp/wails/v3/pkg/services/notifications"
)

//go:embed assets/tray-icon.png
var trayIcon []byte

//go:embed assets/tray-icon-alert.png
var alertIcon []byte

const eventAlertWindowMinutes = 10

func runApp(url string, ag *agent.Agent, st *store.Store, shutdown context.CancelFunc) {
	opts := application.Options{
		Name:        "Aide",
		Description: "Your personal work assistant",
		Mac: application.MacOptions{
			ActivationPolicy: application.ActivationPolicyAccessory,
		},
	}

	var notifSvc *notifications.NotificationService
	if isBundled() {
		notifSvc = notifications.New()
		opts.Services = []application.Service{application.NewService(notifSvc)}
	}

	app := application.New(opts)

	// Let the in-process agent quit the app so a staged in-place update can
	// swap the bundle and relaunch.
	ag.SetRestartHandler(func() { app.Quit() })

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
		URL:       url + "/?desktop=1",
		Mac: application.MacWindow{
			TitleBar: application.MacTitleBarHidden,
		},
	})

	showWindow := func() {
		window.Show()
		window.Focus()
	}

	window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		e.Cancel()
		window.Hide()
	})

	app.Event.OnApplicationEvent(events.Mac.ApplicationShouldHandleReopen, func(*application.ApplicationEvent) {
		showWindow()
	})

	const panelWidth = 320
	panel := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:           "tray-panel",
		Width:          panelWidth,
		Height:         240,
		Hidden:         true,
		Frameless:      true,
		DisableResize:  true,
		AlwaysOnTop:    true,
		URL:            url + "/?panel=tray",
		BackgroundType: application.BackgroundTypeTransparent,
		Mac: application.MacWindow{
			Backdrop: application.MacBackdropTranslucent,
			CollectionBehavior: application.MacWindowCollectionBehaviorCanJoinAllSpaces |
				application.MacWindowCollectionBehaviorFullScreenAuxiliary,
		},
	})

	tray := app.SystemTray.New()
	tray.SetTemplateIcon(trayIcon)
	tray.SetTooltip("Aide")

	// Anchor the panel's left edge under the tray icon (like the OneDrive
	// menu-bar panel) instead of the centered default, clamping to the screen.
	positionPanel := func() {
		_ = tray.PositionWindow(panel, 6)
		cx, cy := panel.Position()
		targetX := cx + panelWidth/2 - 12
		if screen, err := panel.GetScreen(); err == nil && screen != nil {
			wa := screen.WorkArea
			if maxX := wa.X + wa.Width - panelWidth - 8; targetX > maxX {
				targetX = maxX
			}
			if targetX < wa.X+8 {
				targetX = wa.X + 8
			}
		}
		panel.SetPosition(targetX, cy)
	}

	var lastPanelHide atomic.Int64
	hidePanel := func() {
		lastPanelHide.Store(time.Now().UnixNano())
		panel.Hide()
	}

	// HideOnFocusLost only fires the common WindowLostFocus event, which macOS
	// never emits, so dismiss-on-click-out is wired to the native resign-key
	// event instead.
	panel.OnWindowEvent(events.Mac.WindowDidResignKey, func(*application.WindowEvent) {
		hidePanel()
	})

	tray.OnClick(func() {
		if panel.IsVisible() {
			hidePanel()
			return
		}
		// The same click that dismissed the panel (via resign-key) must not
		// reopen it.
		if time.Since(time.Unix(0, lastPanelHide.Load())) < 250*time.Millisecond {
			return
		}
		positionPanel()
		panel.Show().Focus()
	})

	menu := app.NewMenu()
	menu.Add("Open Aide").OnClick(func(*application.Context) {
		showWindow()
	})
	menu.AddSeparator()
	menu.Add("Quit Aide").OnClick(func(*application.Context) {
		app.Quit()
	})
	tray.SetMenu(menu)

	done := make(chan struct{})

	refreshIcon := func() {
		next, err := st.Items.NextEvent()
		if err != nil {
			return
		}
		if next != nil && (next.InProgress || (next.MinutesUntil >= 0 && next.MinutesUntil <= eventAlertWindowMinutes)) {
			tray.SetTemplateIcon(alertIcon)
			if next.InProgress || next.MinutesUntil <= 0 {
				tray.SetLabel("Now")
			} else {
				tray.SetLabel(strconv.Itoa(next.MinutesUntil) + "m")
			}
			return
		}
		tray.SetTemplateIcon(trayIcon)
		tray.SetLabel("")
	}

	ch, unsubscribe := ag.Bus().Subscribe()
	go func() {
		for {
			select {
			case <-done:
				return
			case ev, ok := <-ch:
				if !ok {
					return
				}
				if ev.Type != "ui_command" {
					continue
				}
				var cmd struct {
					Action string `json:"action"`
					View   string `json:"view"`
				}
				if json.Unmarshal([]byte(ev.Data), &cmd) != nil {
					continue
				}
				if cmd.Action == "quit" {
					app.Quit()
					continue
				}
				hidePanel()
				showWindow()
			}
		}
	}()

	go func() {
		refreshIcon()
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				refreshIcon()
			}
		}
	}()

	app.OnShutdown(func() {
		close(done)
		unsubscribe()
		shutdown()
		st.Close()
	})

	if err := app.Run(); err != nil {
		fatal("app run: %v", err)
	}
}
