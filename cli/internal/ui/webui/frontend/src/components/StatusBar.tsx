import { useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import {
  ArrowUpCircle,
  CalendarClock,
  Check,
  ChevronDown,
  ChevronUp,
  Inbox,
  Layers,
  Moon,
  PanelLeft,
  ScrollText,
  Settings,
  Sparkles,
  Sun,
} from "lucide-react";
import { fetchStatus, fetchVersion, fetchWhoami, triggerUpdate } from "@/lib/api";
import type { VersionInfo } from "@/lib/api";
import { APP_NAME } from "@/lib/brand";
import { cn } from "@/lib/cn";
import { handleExternalClick } from "@/lib/openExternal";
import { MarkdownRenderer } from "@/components/renderers/MarkdownRenderer";
import { useUpdateProgress } from "@/hooks/useUpdateProgress";

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
  onOpenLogs?: () => void;
}

export function StatusBar({
  connected,
  onToggleSidebar,
  activeSource,
  onSourceClick,
  onOpenSettings,
  onOpenLogs,
}: Props) {
  const [status, setStatus] = useState<StatusData | null>(null);
  const [statusError, setStatusError] = useState(false);
  const [userName, setUserName] = useState("");
  const [updateInfo, setUpdateInfo] = useState<VersionInfo | null>(null);
  const [showNotes, setShowNotes] = useState(false);
  const [dismissed, setDismissed] = useState(false);
  const { progress, start } = useUpdateProgress();
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

    void fetchWhoami()
      .then((p) => {
        if (p.preferred_name) setUserName(p.preferred_name);
      })
      .catch(() => {});

    void fetchVersion()
      .then((v) => {
        if (v.update_available && v.current !== "dev") {
          setUpdateInfo(v);
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

  const runUpdate = () => {
    start();
    void triggerUpdate().catch(() => {});
  };

  const counts = status?.counts ?? {};
  const metrics = status?.metrics ?? [];
  const unread = metrics.find((m) => m.name === "Inbox Unread")?.value;
  const total = Object.values(counts).reduce((a, b) => a + b, 0);

  const hasStats = !!status && (status.today_events > 0 || total > 0);

  return (
    <header className="flex flex-col border-b bg-card/80 backdrop-blur supports-[backdrop-filter]:bg-card/60">
      {updateInfo && !dismissed && (
        <div className="border-b border-warning/25 bg-warning/10 px-4 py-1.5 text-xs text-warning-foreground">
          <div className="flex items-center gap-2">
            <ArrowUpCircle className="h-3.5 w-3.5 shrink-0 text-warning" />
            <span>
              Update available{updateInfo.latest ? `: ${updateInfo.latest}` : ""} (current:{" "}
              {updateInfo.current})
            </span>
            {updateInfo.notes && (
              <button
                onClick={() => setShowNotes((s) => !s)}
                className="inline-flex items-center gap-0.5 rounded px-1 py-0.5 font-medium hover:bg-warning/20"
              >
                What's new
                {showNotes ? (
                  <ChevronUp className="h-3 w-3" />
                ) : (
                  <ChevronDown className="h-3 w-3" />
                )}
              </button>
            )}
            <div className="ml-auto flex items-center gap-2">
              {progress.done ? (
                <span className="font-medium text-success">Restart to finish</span>
              ) : updateInfo.can_self_update ? (
                <button
                  onClick={runUpdate}
                  disabled={progress.running}
                  className="inline-flex items-center gap-1 rounded bg-warning/15 px-2 py-0.5 font-medium transition-colors hover:bg-warning/25 disabled:opacity-60 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                >
                  {progress.running ? "Updating…" : "Update now"}
                </button>
              ) : (
                <a
                  href={updateInfo.release_url || "https://github.com/matheus-meneses/aide/releases/latest"}
                  target="_blank"
                  rel="noreferrer"
                  onClick={(e) =>
                    handleExternalClick(
                      e,
                      updateInfo.release_url ||
                        "https://github.com/matheus-meneses/aide/releases/latest",
                    )
                  }
                  className="inline-flex items-center gap-1 rounded bg-warning/15 px-2 py-0.5 font-medium transition-colors hover:bg-warning/25 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                >
                  View release
                </a>
              )}
              {!progress.running && (
                <button
                  onClick={() => setDismissed(true)}
                  className="rounded px-1 py-0.5 font-medium hover:bg-warning/20"
                  aria-label="Dismiss"
                >
                  ✕
                </button>
              )}
            </div>
          </div>

          {showNotes && updateInfo.notes && (
            <div className="mt-1.5 max-h-48 overflow-y-auto rounded bg-background/50 p-2 text-foreground">
              <MarkdownRenderer content={updateInfo.notes} />
            </div>
          )}

          {(progress.lines.length > 0 || progress.error) && (
            <div className="mt-1.5 max-h-24 overflow-y-auto rounded bg-background/50 p-2 font-mono text-[11px] text-muted-foreground">
              {progress.lines.map((l, i) => (
                <div key={i}>{l}</div>
              ))}
              {progress.error && <div className="text-destructive">{progress.error}</div>}
            </div>
          )}
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
            <span className="text-base font-semibold tracking-tight">{APP_NAME}</span>
          </div>

          {hasStats && <div className="ml-1 h-5 w-px bg-border" />}

          {status && hasStats ? (
            <StatsSummary
              counts={counts}
              todayEvents={status.today_events}
              total={total}
              unread={unread}
              activeSource={activeSource ?? null}
              onSourceClick={onSourceClick}
            />
          ) : statusError ? (
            <span className="hidden text-xs text-destructive sm:inline">Could not load status</span>
          ) : !status ? (
            <div className="h-8 w-28 animate-pulse rounded-full bg-muted" />
          ) : null}
        </div>
        <div className="flex shrink-0 items-center gap-2">
          {userName && (
            <span className="text-xs text-muted-foreground hidden sm:inline">{userName}</span>
          )}
          <span
            role="status"
            aria-label={connected ? "Connected to the agent" : "Disconnected from the agent"}
            title={connected ? "Connected to the agent" : "Disconnected from the agent"}
            className="mr-0.5 inline-flex h-7 w-7 items-center justify-center"
          >
            <span
              className={cn(
                "h-2 w-2 rounded-full",
                connected
                  ? "bg-success shadow-[0_0_0_3px] shadow-success/20"
                  : "animate-pulse bg-destructive shadow-[0_0_0_3px] shadow-destructive/20",
              )}
            />
          </span>
          <button
            onClick={toggleDark}
            className="p-1.5 rounded-md hover:bg-accent transition-colors"
            aria-label={dark ? "Switch to light mode" : "Switch to dark mode"}
          >
            {dark ? <Sun className="w-4 h-4" /> : <Moon className="w-4 h-4" />}
          </button>
          {onOpenLogs && (
            <button
              onClick={onOpenLogs}
              className="rounded-md p-1.5 transition-colors hover:bg-accent"
              aria-label="Open logs"
            >
              <ScrollText className="h-4 w-4" />
            </button>
          )}
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

const DOT_COLORS = [
  "bg-sky-500",
  "bg-violet-500",
  "bg-emerald-500",
  "bg-amber-500",
  "bg-rose-500",
  "bg-cyan-500",
  "bg-fuchsia-500",
  "bg-lime-500",
];

function sourceDot(source: string): string {
  let hash = 0;
  for (let i = 0; i < source.length; i++) {
    hash = (hash * 31 + source.charCodeAt(i)) >>> 0;
  }
  return DOT_COLORS[hash % DOT_COLORS.length] ?? "bg-muted-foreground";
}

function StatsSummary({
  counts,
  todayEvents,
  total,
  unread,
  activeSource,
  onSourceClick,
}: {
  counts: Record<string, number>;
  todayEvents: number;
  total: number;
  unread?: number;
  activeSource: string | null;
  onSourceClick?: (source: string | null) => void;
}) {
  const [open, setOpen] = useState(false);
  const [pos, setPos] = useState<{ top: number; left: number } | null>(null);
  const triggerRef = useRef<HTMLButtonElement>(null);
  const menuRef = useRef<HTMLDivElement>(null);

  const openMenu = () => {
    const r = triggerRef.current?.getBoundingClientRect();
    if (r) setPos({ top: r.bottom + 6, left: r.left });
    setOpen(true);
  };

  useEffect(() => {
    if (!open) return;
    const onDoc = (e: MouseEvent) => {
      const t = e.target as Node;
      if (triggerRef.current?.contains(t) || menuRef.current?.contains(t)) return;
      setOpen(false);
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") setOpen(false);
    };
    const close = () => setOpen(false);
    document.addEventListener("mousedown", onDoc);
    document.addEventListener("keydown", onKey);
    window.addEventListener("resize", close);
    window.addEventListener("scroll", close, true);
    return () => {
      document.removeEventListener("mousedown", onDoc);
      document.removeEventListener("keydown", onKey);
      window.removeEventListener("resize", close);
      window.removeEventListener("scroll", close, true);
    };
  }, [open]);

  const sources = Object.entries(counts).filter(([, c]) => c > 0);
  const isAll = !activeSource || activeSource === "__all";
  const filtered = !isAll;

  let triggerIcon: React.ReactNode;
  let triggerLabel: string;
  let triggerCount: number;
  if (activeSource === "__meetings") {
    triggerIcon = <CalendarClock className="h-3.5 w-3.5" />;
    triggerLabel = "meetings";
    triggerCount = todayEvents;
  } else if (filtered) {
    triggerIcon = <span className={cn("h-2 w-2 rounded-full", sourceDot(activeSource))} />;
    triggerLabel = activeSource;
    triggerCount = counts[activeSource] ?? 0;
  } else {
    triggerIcon = <Layers className="h-3.5 w-3.5" />;
    triggerLabel = "open";
    triggerCount = total;
  }

  const select = (s: string | null) => {
    onSourceClick?.(activeSource === s ? null : s);
    setOpen(false);
  };

  return (
    <>
      <button
        ref={triggerRef}
        onClick={() => (open ? setOpen(false) : openMenu())}
        aria-haspopup="menu"
        aria-expanded={open}
        className={cn(
          "inline-flex items-center gap-1.5 rounded-full border px-3 py-1.5 text-xs transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
          filtered || activeSource === "__meetings"
            ? "border-primary/30 bg-primary/10 text-primary"
            : "border-border/60 bg-muted/40 text-muted-foreground hover:bg-muted hover:text-foreground",
        )}
      >
        {triggerIcon}
        <span className="font-semibold tabular-nums text-foreground">{triggerCount}</span>
        <span className="capitalize">{triggerLabel}</span>
        <ChevronDown
          className={cn("h-3.5 w-3.5 opacity-60 transition-transform", open && "rotate-180")}
        />
      </button>

      {open &&
        pos &&
        createPortal(
          <div
            ref={menuRef}
            role="menu"
            style={{ position: "fixed", top: pos.top, left: pos.left }}
            className="z-[100] w-60 origin-top-left animate-msg-in rounded-lg border bg-popover p-1 text-popover-foreground shadow-lg"
          >
            <MenuRow
              icon={<Layers className="h-4 w-4 text-muted-foreground" />}
              label="All items"
              count={total}
              active={isAll}
              onClick={() => select("__all")}
            />
            {sources.length > 0 && <div className="my-1 h-px bg-border" />}
            {sources.map(([source, count]) => (
              <MenuRow
                key={source}
                dotClass={sourceDot(source)}
                label={source}
                count={count}
                active={activeSource === source}
                onClick={() => select(source)}
              />
            ))}
            {todayEvents > 0 && (
              <>
                <div className="my-1 h-px bg-border" />
                <MenuRow
                  icon={<CalendarClock className="h-4 w-4 text-muted-foreground" />}
                  label="Today's meetings"
                  count={todayEvents}
                  active={activeSource === "__meetings"}
                  onClick={() => select("__meetings")}
                />
              </>
            )}
            {unread != null && unread > 0 && (
              <>
                <div className="my-1 h-px bg-border" />
                <div className="flex items-center gap-2.5 px-2.5 py-2 text-xs text-muted-foreground">
                  <Inbox className="h-4 w-4 shrink-0" />
                  <span className="flex-1">Unread in inbox</span>
                  <span className="font-semibold tabular-nums text-foreground">{unread}</span>
                </div>
              </>
            )}
          </div>,
          document.body,
        )}
    </>
  );
}

function MenuRow({
  icon,
  dotClass,
  label,
  count,
  active,
  onClick,
}: {
  icon?: React.ReactNode;
  dotClass?: string;
  label: string;
  count: number;
  active: boolean;
  onClick: () => void;
}) {
  return (
    <button
      role="menuitem"
      onClick={onClick}
      className={cn(
        "flex w-full items-center gap-2.5 rounded-md px-2.5 py-2 text-left text-xs transition-colors",
        active ? "bg-accent" : "hover:bg-accent",
      )}
    >
      {dotClass ? (
        <span className={cn("h-2 w-2 shrink-0 rounded-full", dotClass)} />
      ) : (
        <span className="shrink-0">{icon}</span>
      )}
      <span className="flex-1 truncate capitalize text-foreground">{label}</span>
      <span className="tabular-nums font-semibold text-foreground">{count}</span>
      <Check className={cn("h-3.5 w-3.5 text-primary", active ? "opacity-100" : "opacity-0")} />
    </button>
  );
}
