import { type ReactNode, useCallback, useEffect, useState } from "react";
import { AlertTriangle, Loader2, PanelLeft, PanelLeftClose } from "lucide-react";
import { type AgentEvent, useSSE } from "@/hooks/useSSE";
import { subscribeEvent } from "@/lib/eventBus";
import { cn } from "@/lib/cn";
import { isDesktopApp } from "@/lib/platform";
import { TitleBar } from "@/components/TitleBar";
import { StatusBar } from "@/components/StatusBar";
import { NotificationFeed } from "@/components/NotificationFeed";
import { ChatPanel } from "@/components/ChatPanel";
import { ChatProvider } from "@/components/ChatProvider";
import { ItemsView } from "@/components/ItemsView";
import { LogsView } from "@/components/LogsView";
import { LLMBanner } from "@/components/LLMBanner";
import { SettingsView, type TabId } from "@/components/settings/SettingsView";
import SetupWizard from "@/components/setup/SetupWizard";
import { fetchSetupStatus } from "@/lib/api";
import { Button, EmptyState } from "@/components/ui";
import { ShortcutsDialog } from "@/components/ShortcutsDialog";

function App() {
  const [needsSetup, setNeedsSetup] = useState<boolean | null>(null);
  const [setupError, setSetupError] = useState<string | null>(null);

  const loadSetup = useCallback(() => {
    setSetupError(null);
    setNeedsSetup(null);
    fetchSetupStatus()
      .then((s) => setNeedsSetup(s.needs_setup))
      .catch((e: unknown) => setSetupError(e instanceof Error ? e.message : String(e)));
  }, []);

  useEffect(() => {
    loadSetup();
  }, [loadSetup]);

  let content: ReactNode;
  if (setupError !== null) {
    content = (
      <div className="flex h-full items-center justify-center p-6">
        <EmptyState
          icon={AlertTriangle}
          title="Couldn't reach the agent"
          description={setupError}
          action={
            <Button size="sm" variant="secondary" onClick={loadSetup}>
              Retry
            </Button>
          }
        />
      </div>
    );
  } else if (needsSetup === null) {
    content = (
      <div className="flex h-full items-center justify-center text-muted-foreground">
        <Loader2 className="mr-2 h-5 w-5 animate-spin" /> Loading…
      </div>
    );
  } else if (needsSetup) {
    content = <SetupWizard onComplete={() => window.location.reload()} />;
  } else {
    content = <MainApp />;
  }

  return (
    <div className="flex h-screen flex-col overflow-hidden">
      {isDesktopApp && <TitleBar />}
      <div className="min-h-0 flex-1">{content}</div>
    </div>
  );
}

function MainApp() {
  const {
    events,
    connected,
    dismiss,
    clear,
    onChatMessage,
    notificationPermission,
    enableNotifications,
  } = useSSE("/api/events");
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [lastSeenCount, setLastSeenCount] = useState(0);
  const [isMobile, setIsMobile] = useState(false);
  const [activeSource, setActiveSource] = useState<string | null>(null);
  const [statusRefreshKey, setStatusRefreshKey] = useState(0);
  const [pendingEvent, setPendingEvent] = useState<AgentEvent | null>(null);
  const [showSettings, setShowSettings] = useState(false);
  const [settingsTab, setSettingsTab] = useState<TabId>("profile");
  const [showLogs, setShowLogs] = useState(false);
  const [showShortcuts, setShowShortcuts] = useState(false);
  const [everConnected, setEverConnected] = useState(false);

  useEffect(() => {
    if (connected) setEverConnected(true);
  }, [connected]);

  useEffect(() => {
    if (sidebarOpen) setLastSeenCount(events.length);
  }, [sidebarOpen, events.length]);

  const unreadCount = Math.max(0, events.length - lastSeenCount);

  const openSettings = useCallback((tab: TabId = "profile") => {
    setShowLogs(false);
    setSettingsTab(tab);
    setShowSettings(true);
  }, []);

  const openLogs = useCallback(() => {
    setShowSettings(false);
    setShowLogs(true);
  }, []);

  useEffect(() => {
    return subscribeEvent("ui_command", (data) => {
      try {
        const outer = JSON.parse(data) as { data?: string };
        const inner = JSON.parse(outer.data ?? "{}") as { action?: string; view?: string };
        if (inner.action !== "navigate") return;
        if (inner.view === "logs") openLogs();
        else if (inner.view === "settings") openSettings();
      } catch {
        // ignore malformed command
      }
    });
  }, [openLogs, openSettings]);

  const handleEventClick = useCallback((event: AgentEvent) => {
    setActiveSource(null);
    setPendingEvent(event);
  }, []);

  useEffect(() => {
    const mq = window.matchMedia("(min-width: 768px)");
    const handler = (e: MediaQueryListEvent | MediaQueryList) => {
      setIsMobile(!e.matches);
      if (e.matches) setSidebarOpen(true);
    };
    handler(mq);
    mq.addEventListener("change", handler);
    return () => mq.removeEventListener("change", handler);
  }, []);

  const closeSidebar = useCallback(() => {
    if (isMobile) setSidebarOpen(false);
  }, [isMobile]);

  useEffect(() => {
    const isTextField = (el: EventTarget | null): boolean => {
      const node = el as HTMLElement | null;
      if (!node) return false;
      const tag = node.tagName;
      return tag === "INPUT" || tag === "TEXTAREA" || node.isContentEditable;
    };
    const onKey = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === ",") {
        e.preventDefault();
        setShowSettings((v) => !v);
        return;
      }
      if (isDesktopApp && (e.metaKey || e.ctrlKey) && (e.key === "l" || e.key === "L")) {
        e.preventDefault();
        setShowLogs((v) => !v);
        return;
      }
      if (e.key === "?" && !isTextField(e.target)) {
        e.preventDefault();
        setShowShortcuts((v) => !v);
        return;
      }
      if (e.key === "Escape") {
        if (showShortcuts) setShowShortcuts(false);
        else if (showLogs) setShowLogs(false);
        else if (showSettings) setShowSettings(false);
        else if (activeSource) setActiveSource(null);
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [showSettings, showLogs, activeSource, showShortcuts]);

  return (
    <ChatProvider
      registerChatMessage={onChatMessage}
      pendingEvent={pendingEvent}
      onEventConsumed={() => setPendingEvent(null)}
    >
    <div className="h-full flex flex-col">
      <a
        href="#main-content"
        className="sr-only focus:not-sr-only focus:absolute focus:left-2 focus:top-2 focus:z-50 focus:rounded-md focus:bg-primary focus:px-3 focus:py-2 focus:text-sm focus:font-medium focus:text-primary-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
      >
        Skip to content
      </a>
      <StatusBar
        connected={connected}
        onToggleSidebar={() => setSidebarOpen((v) => !v)}
        unreadCount={unreadCount}
        activeSource={activeSource}
        onSourceClick={setActiveSource}
        refreshKey={statusRefreshKey}
        onOpenSettings={() => openSettings()}
        onOpenLogs={isDesktopApp ? openLogs : undefined}
      />
      {!connected && (
        <div
          role="status"
          aria-live="polite"
          className={cn(
            "animate-fade-in relative overflow-hidden border-b text-xs",
            everConnected
              ? "border-warning/25 bg-warning/10 text-warning-foreground"
              : "border-border/60 bg-muted/30 text-muted-foreground",
          )}
        >
          <div className="absolute inset-x-0 top-0 h-0.5 overflow-hidden">
            <div
              className={cn(
                "animate-indeterminate absolute inset-y-0 rounded-full",
                everConnected ? "bg-warning" : "bg-primary",
              )}
            />
          </div>
          <div className="flex items-center justify-center gap-2 px-4 py-1.5">
            <Loader2
              className={cn(
                "h-3.5 w-3.5 animate-spin",
                everConnected ? "text-warning" : "text-primary",
              )}
            />
            <span>
              {everConnected ? "Reconnecting to the agent…" : "Connecting to the agent…"}
            </span>
          </div>
        </div>
      )}
      {isDesktopApp && showLogs ? (
        <div className="flex-1 overflow-hidden">
          <LogsView onClose={() => setShowLogs(false)} />
        </div>
      ) : showSettings ? (
        <div className="flex-1 overflow-hidden">
          <SettingsView onClose={() => setShowSettings(false)} initialTab={settingsTab} />
        </div>
      ) : (
      <div className="flex flex-1 flex-col overflow-hidden">
        <LLMBanner onConfigure={() => openSettings("agent")} />
        <div className="flex flex-1 overflow-hidden relative">
        {isMobile && sidebarOpen && (
          <div
            className="fixed inset-0 z-30 bg-black/40 backdrop-blur-sm md:hidden animate-fade-in"
            onClick={closeSidebar}
          />
        )}

        <aside
          className={`
            ${isMobile ? "fixed inset-y-0 left-0 z-40 w-72 pt-[49px]" : "relative w-72 shrink-0"}
            border-r flex flex-col bg-card transition-transform duration-200 ease-out
            ${sidebarOpen ? "translate-x-0" : "-translate-x-full"}
            ${!isMobile && !sidebarOpen ? "hidden" : ""}
          `}
        >
          <div className="flex items-center justify-between px-3 py-2 border-b">
            <span className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
              Activity
            </span>
            <div className="flex items-center gap-1">
              {events.length > 0 && (
                <button
                  onClick={clear}
                  className="rounded px-1.5 py-0.5 text-[11px] font-medium text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
                  aria-label="Clear all notifications"
                >
                  Clear all
                </button>
              )}
              <button
                onClick={() => setSidebarOpen(false)}
                className="p-1 rounded hover:bg-accent transition-colors"
                aria-label="Close sidebar"
              >
                <PanelLeftClose className="w-4 h-4 text-muted-foreground" />
              </button>
            </div>
          </div>
          <div className="flex-1 overflow-hidden">
            <NotificationFeed
              events={events}
              onEventClick={handleEventClick}
              onDismiss={dismiss}
              notificationPermission={notificationPermission}
              onEnableNotifications={() => {
                void enableNotifications();
              }}
            />
          </div>
        </aside>

        <main id="main-content" tabIndex={-1} className="flex-1 flex flex-col overflow-hidden focus:outline-none">
          {!sidebarOpen && !isMobile && (
            <button
              onClick={() => setSidebarOpen(true)}
              className="absolute top-2 left-2 z-10 p-1.5 rounded-md bg-card border hover:bg-accent transition-colors"
              aria-label={
                unreadCount > 0 ? `Open sidebar, ${unreadCount} new` : "Open sidebar"
              }
            >
              <PanelLeft className="w-4 h-4" />
              {unreadCount > 0 && (
                <span className="absolute -right-1 -top-1 flex h-4 min-w-4 items-center justify-center rounded-full bg-primary px-1 text-[10px] font-semibold leading-none text-primary-foreground">
                  {unreadCount > 99 ? "99+" : unreadCount}
                </span>
              )}
            </button>
          )}
          {activeSource ? (
            <ItemsView
              source={activeSource}
              onClose={() => setActiveSource(null)}
              onItemDone={() => setStatusRefreshKey((k) => k + 1)}
            />
          ) : (
            <ChatPanel onConfigure={() => openSettings("agent")} />
          )}
        </main>
        </div>
      </div>
      )}
      <ShortcutsDialog open={showShortcuts} onClose={() => setShowShortcuts(false)} />
    </div>
    </ChatProvider>
  );
}

export default App;
