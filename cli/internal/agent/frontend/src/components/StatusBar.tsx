import { useEffect, useState } from "react";
import {
  ArrowUpCircle,
  CalendarClock,
  Inbox,
  Moon,
  PanelLeft,
  Settings,
  Sparkles,
  Sun,
  Wifi,
  WifiOff,
} from "lucide-react";
import { fetchStatus, fetchVersion } from "@/lib/api";
import { cn } from "@/lib/cn";

interface StatusData {
  counts: Record<string, number>;
  today_events: number;
  metrics: Array<{ name: string; value: number; source: string }>;
}

interface Props {
  connected: boolean;
  onToggleSidebar: () => void;
  activeSource?: string | null;
  onSourceClick?: (source: string | null) => void;
  onOpenSettings?: () => void;
}

export function StatusBar({
  connected,
  onToggleSidebar,
  activeSource,
  onSourceClick,
  onOpenSettings,
}: Props) {
  const [status, setStatus] = useState<StatusData | null>(null);
  const [statusError, setStatusError] = useState(false);
  const [userName, setUserName] = useState("");
  const [updateAvailable, setUpdateAvailable] = useState<{
    current: string;
    latest: string;
    url: string;
  } | null>(null);
  const [dark, setDark] = useState(() => document.documentElement.classList.contains("dark"));

  useEffect(() => {
    void fetchStatus()
      .then((d) => {
        setStatus(d as unknown as StatusData);
        setStatusError(false);
      })
      .catch(() => {
        setStatusError(true);
      });

    void fetch("/api/whoami")
      .then((r) => r.json() as Promise<{ preferred_name?: string }>)
      .then((p) => {
        if (p.preferred_name) setUserName(p.preferred_name);
      })
      .catch(() => {});

    void fetchVersion()
      .then((v) => {
        if (v.update_available && v.current !== "dev") {
          setUpdateAvailable({ current: v.current, latest: v.latest, url: v.update_url });
        }
      })
      .catch(() => {});

    const interval = setInterval(() => {
      void fetchStatus()
        .then((d) => {
          setStatus(d as unknown as StatusData);
          setStatusError(false);
        })
        .catch(() => {
          setStatusError(true);
        });
    }, 60000);
    return () => {
      clearInterval(interval);
    };
  }, []);

  const toggleDark = () => {
    const next = !dark;
    document.documentElement.classList.toggle("dark", next);
    localStorage.setItem("theme", next ? "dark" : "light");
    setDark(next);
  };

  const counts = status?.counts ?? {};
  const metrics = status?.metrics ?? [];
  const unread = metrics.find((m) => m.name === "Inbox Unread")?.value;
  const total = Object.values(counts).reduce((a, b) => a + b, 0);

  return (
    <header className="flex flex-col border-b bg-card/80 backdrop-blur supports-[backdrop-filter]:bg-card/60">
      {updateAvailable && (
        <div className="flex items-center gap-2 border-b border-warning/25 bg-warning/10 px-4 py-1.5 text-xs text-warning-foreground">
          <ArrowUpCircle className="h-3.5 w-3.5 shrink-0 text-warning" />
          <span>
            Update available{updateAvailable.latest ? `: ${updateAvailable.latest}` : ""} (current:{" "}
            {updateAvailable.current})
          </span>
          <code className="ml-auto rounded bg-warning/15 px-1.5 py-0.5 font-mono text-[10px]">
            curl -fsSL {updateAvailable.url} | bash
          </code>
        </div>
      )}
      <div className="flex items-center justify-between gap-3 px-4 py-2.5">
        <div className="flex min-w-0 items-center gap-3 text-sm">
          <button
            onClick={onToggleSidebar}
            className="rounded p-1 transition-colors hover:bg-accent md:hidden"
            aria-label="Toggle sidebar"
          >
            <PanelLeft className="h-4 w-4" />
          </button>

          <div className="flex items-center gap-2">
            <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-gradient-to-br from-primary to-primary/70 text-primary-foreground shadow-sm">
              <Sparkles className="h-4 w-4" />
            </div>
            <span className="text-base font-semibold tracking-tight">Aide</span>
          </div>

          {status && (status.today_events > 0 || Object.keys(counts).length > 0) && (
            <div className="ml-1 hidden h-5 w-px bg-border sm:block" />
          )}

          {status ? (
            <div className="hidden items-center gap-1.5 sm:flex">
              {status.today_events > 0 && (
                <StatPill
                  active={activeSource === "__meetings"}
                  onClick={() =>
                    onSourceClick?.(activeSource === "__meetings" ? null : "__meetings")
                  }
                  icon={<CalendarClock className="h-3.5 w-3.5" />}
                  count={status.today_events}
                  label="meetings"
                />
              )}
              {Object.entries(counts).map(([source, count]) => (
                <StatPill
                  key={source}
                  active={activeSource === source}
                  onClick={() => onSourceClick?.(activeSource === source ? null : source)}
                  count={count}
                  label={source}
                />
              ))}
              {unread != null && unread > 0 && (
                <span className="inline-flex items-center gap-1 rounded-full px-2 py-1 text-xs text-muted-foreground">
                  <Inbox className="h-3.5 w-3.5" />
                  <span className="font-semibold text-foreground">{unread}</span> unread
                </span>
              )}
              {total > 0 && (
                <StatPill
                  active={activeSource === "__all"}
                  onClick={() => onSourceClick?.(activeSource ? null : "__all")}
                  count={total}
                  label="total"
                />
              )}
            </div>
          ) : statusError ? (
            <span className="hidden text-xs text-destructive sm:inline">Could not load status</span>
          ) : (
            <div className="hidden items-center gap-2 sm:flex">
              <div className="h-6 w-20 animate-pulse rounded-full bg-muted" />
              <div className="h-6 w-16 animate-pulse rounded-full bg-muted" />
            </div>
          )}
        </div>
        <div className="flex shrink-0 items-center gap-2">
          {userName && (
            <span className="text-xs text-muted-foreground hidden sm:inline">{userName}</span>
          )}
          <div
            className={`flex items-center gap-1.5 rounded-full px-2 py-0.5 text-[11px] font-medium ${
              connected ? "bg-success/15 text-success" : "bg-destructive/15 text-destructive"
            }`}
            title={connected ? "Connected to the agent" : "Disconnected from the agent"}
          >
            {connected ? <Wifi className="w-3.5 h-3.5" /> : <WifiOff className="w-3.5 h-3.5" />}
            <span className="hidden sm:inline">{connected ? "live" : "disconnected"}</span>
          </div>
          <button
            onClick={toggleDark}
            className="p-1.5 rounded-md hover:bg-accent transition-colors"
            aria-label={dark ? "Switch to light mode" : "Switch to dark mode"}
          >
            {dark ? <Sun className="w-4 h-4" /> : <Moon className="w-4 h-4" />}
          </button>
          {onOpenSettings && (
            <button
              onClick={onOpenSettings}
              className="rounded-md p-1.5 transition-colors hover:bg-accent"
              aria-label="Open settings"
            >
              <Settings className="h-4 w-4" />
            </button>
          )}
        </div>
      </div>
    </header>
  );
}

function StatPill({
  active,
  onClick,
  count,
  label,
  icon,
}: {
  active: boolean;
  onClick: () => void;
  count: number;
  label: string;
  icon?: React.ReactNode;
}) {
  return (
    <button
      onClick={onClick}
      aria-pressed={active}
      className={cn(
        "inline-flex items-center gap-1 rounded-full border px-2.5 py-1 text-xs transition-colors",
        active
          ? "border-primary/30 bg-primary/10 text-primary"
          : "border-transparent bg-muted/60 text-muted-foreground hover:bg-accent hover:text-foreground",
      )}
    >
      {icon}
      <span className={cn("font-semibold", active ? "text-primary" : "text-foreground")}>
        {count}
      </span>
      {label}
    </button>
  );
}
