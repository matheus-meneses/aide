import { useCallback, useEffect, useState } from "react";
import { Loader2, PanelLeft, PanelLeftClose } from "lucide-react";
import { type AgentEvent, useSSE } from "@/hooks/useSSE";
import { StatusBar } from "@/components/StatusBar";
import { NotificationFeed } from "@/components/NotificationFeed";
import { ChatPanel } from "@/components/ChatPanel";
import { ItemsView } from "@/components/ItemsView";
import { LLMBanner } from "@/components/LLMBanner";
import { SettingsView, type TabId } from "@/components/settings/SettingsView";
import SetupWizard from "@/components/setup/SetupWizard";
import { fetchSetupStatus } from "@/lib/api";

function App() {
  const [needsSetup, setNeedsSetup] = useState<boolean | null>(null);

  useEffect(() => {
    fetchSetupStatus()
      .then((s) => setNeedsSetup(s.needs_setup))
      .catch(() => setNeedsSetup(false));
  }, []);

  if (needsSetup === null) {
    return (
      <div className="flex h-screen items-center justify-center text-muted-foreground">
        <Loader2 className="mr-2 h-5 w-5 animate-spin" /> Loading…
      </div>
    );
  }

  if (needsSetup) {
    return <SetupWizard onComplete={() => window.location.reload()} />;
  }

  return <MainApp />;
}

function MainApp() {
  const { events, connected, dismiss, onChatMessage, notificationPermission, enableNotifications } =
    useSSE("/api/events");
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [isMobile, setIsMobile] = useState(false);
  const [activeSource, setActiveSource] = useState<string | null>(null);
  const [pendingEvent, setPendingEvent] = useState<AgentEvent | null>(null);
  const [showSettings, setShowSettings] = useState(false);
  const [settingsTab, setSettingsTab] = useState<TabId>("profile");

  const openSettings = useCallback((tab: TabId = "profile") => {
    setSettingsTab(tab);
    setShowSettings(true);
  }, []);

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
    const onKey = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === ",") {
        e.preventDefault();
        setShowSettings((v) => !v);
        return;
      }
      if (e.key === "Escape") {
        if (showSettings) setShowSettings(false);
        else if (activeSource) setActiveSource(null);
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [showSettings, activeSource]);

  return (
    <div className="h-screen flex flex-col">
      <StatusBar
        connected={connected}
        onToggleSidebar={() => setSidebarOpen((v) => !v)}
        activeSource={activeSource}
        onSourceClick={setActiveSource}
        onOpenSettings={() => openSettings()}
      />
      {showSettings ? (
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
            <button
              onClick={() => setSidebarOpen(false)}
              className="p-1 rounded hover:bg-accent transition-colors"
              aria-label="Close sidebar"
            >
              <PanelLeftClose className="w-4 h-4 text-muted-foreground" />
            </button>
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

        <main className="flex-1 flex flex-col overflow-hidden">
          {!sidebarOpen && !isMobile && (
            <button
              onClick={() => setSidebarOpen(true)}
              className="absolute top-2 left-2 z-10 p-1.5 rounded-md bg-card border hover:bg-accent transition-colors"
              aria-label="Open sidebar"
            >
              <PanelLeft className="w-4 h-4" />
            </button>
          )}
          {activeSource ? (
            <ItemsView source={activeSource} onClose={() => setActiveSource(null)} />
          ) : (
            <ChatPanel
              pendingEvent={pendingEvent}
              onEventConsumed={() => setPendingEvent(null)}
              onChatMessage={onChatMessage}
              onConfigure={() => openSettings("agent")}
            />
          )}
        </main>
        </div>
      </div>
      )}
    </div>
  );
}

export default App;
