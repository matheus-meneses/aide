import { useCallback, useEffect, useState } from "react";
import {
  AppWindow,
  CalendarClock,
  ChevronRight,
  Loader2,
  Power,
  RefreshCw,
  ScrollText,
  Settings,
} from "lucide-react";
import {
  fetchNextEvent,
  sendUICommand,
  triggerSync,
  type NextEvent,
  type UICommandAction,
} from "@/lib/api";
import { cn } from "@/lib/cn";

function relativeLabel(ev: NextEvent): string {
  if (ev.in_progress) return "Now";
  const m = ev.minutes_until;
  if (m <= 0) return "Now";
  if (m < 60) return `in ${m}m`;
  const h = Math.floor(m / 60);
  const mm = m % 60;
  if (h < 24) return mm ? `in ${h}h ${mm}m` : `in ${h}h`;
  const d = Math.round(h / 24);
  return `in ${d}d`;
}

function dayPrefix(startISO: string): string {
  const start = new Date(startISO);
  const now = new Date();
  if (start.toDateString() === now.toDateString()) return "";
  return start.toLocaleDateString(undefined, { weekday: "short" }) + " ";
}

export function TrayPanel() {
  const [event, setEvent] = useState<NextEvent | null>(null);
  const [loaded, setLoaded] = useState(false);
  const [error, setError] = useState(false);
  const [syncing, setSyncing] = useState(false);

  const refresh = useCallback(() => {
    fetchNextEvent()
      .then((e) => {
        setEvent(e);
        setError(false);
      })
      .catch(() => setError(true))
      .finally(() => setLoaded(true));
  }, []);

  useEffect(() => {
    refresh();
    const id = window.setInterval(refresh, 30_000);
    return () => window.clearInterval(id);
  }, [refresh]);

  const command = useCallback((action: UICommandAction, view?: string) => {
    void sendUICommand(action, view);
  }, []);

  const onSync = useCallback(() => {
    setSyncing(true);
    void triggerSync().finally(() => {
      window.setTimeout(() => {
        refresh();
        setSyncing(false);
      }, 2500);
    });
  }, [refresh]);

  const imminent = !!event && (event.in_progress || (event.minutes_until >= 0 && event.minutes_until <= 10));

  return (
    <div className="h-screen w-screen p-1.5">
      <div className="flex h-full flex-col overflow-hidden rounded-2xl border bg-popover/95 shadow-xl backdrop-blur-xl">
        <div className="flex items-center gap-2.5 px-4 pt-3.5 pb-2.5">
          <img src="/favicon.svg" alt="" className="h-6 w-6 rounded-md" />
          <div className="min-w-0">
            <div className="text-sm font-semibold leading-tight">Aide</div>
            <div className="truncate text-[11px] text-muted-foreground">Your personal work assistant</div>
          </div>
        </div>

        <div className="px-2">
          {loaded && error && !event ? (
            <div className="flex w-full items-center gap-3 rounded-xl px-2.5 py-2.5">
              <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-destructive/10 text-destructive">
                <CalendarClock className="h-[18px] w-[18px]" />
              </span>
              <span className="min-w-0 flex-1">
                <span className="block truncate text-sm font-medium leading-tight">Couldn't load events</span>
                <span className="block truncate text-[11px] text-muted-foreground">
                  Check that the agent is running.
                </span>
              </span>
              <button
                type="button"
                onClick={refresh}
                className="shrink-0 rounded-md px-2 py-1 text-[11px] font-medium text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
              >
                Retry
              </button>
            </div>
          ) : (
          <button
            type="button"
            onClick={() => command("show")}
            className="group flex w-full items-center gap-3 rounded-xl px-2.5 py-2.5 text-left transition-colors hover:bg-accent"
          >
            <span
              className={cn(
                "flex h-9 w-9 shrink-0 items-center justify-center rounded-full",
                imminent ? "bg-warning/15 text-warning" : "bg-primary/10 text-foreground/80",
              )}
            >
              <CalendarClock className="h-[18px] w-[18px]" />
            </span>
            <span className="min-w-0 flex-1">
              {!loaded ? (
                <span className="text-sm text-muted-foreground">Loading…</span>
              ) : event ? (
                <>
                  <span className="block truncate text-sm font-medium leading-tight">{event.title}</span>
                  <span className="block truncate text-[11px] text-muted-foreground">
                    {dayPrefix(event.start)}
                    {event.time}
                    <span className={cn("ml-1", imminent && "font-medium text-warning")}>
                      · {relativeLabel(event)}
                    </span>
                    {event.member ? ` · ${event.member}` : ""}
                  </span>
                </>
              ) : (
                <span className="text-sm text-muted-foreground">No upcoming meetings</span>
              )}
            </span>
            <ChevronRight className="h-4 w-4 shrink-0 text-muted-foreground transition-transform group-hover:translate-x-0.5" />
          </button>
          )}
        </div>

        <div className="mt-auto flex items-center justify-between border-t px-2.5 py-2">
          <ActionButton label="Open" icon={AppWindow} onClick={() => command("show")} />
          <ActionButton label="Settings" icon={Settings} onClick={() => command("navigate", "settings")} />
          <ActionButton
            label="Sync now"
            icon={syncing ? Loader2 : RefreshCw}
            spinning={syncing}
            disabled={syncing}
            onClick={onSync}
          />
          <ActionButton label="Logs" icon={ScrollText} onClick={() => command("navigate", "logs")} />
          <ActionButton label="Quit" icon={Power} danger onClick={() => command("quit")} />
        </div>
      </div>
    </div>
  );
}

interface ActionButtonProps {
  label: string;
  icon: React.ComponentType<{ className?: string }>;
  onClick: () => void;
  spinning?: boolean;
  disabled?: boolean;
  danger?: boolean;
}

function ActionButton({ label, icon: Icon, onClick, spinning, disabled, danger }: ActionButtonProps) {
  return (
    <button
      type="button"
      title={label}
      aria-label={label}
      disabled={disabled}
      onClick={onClick}
      className={cn(
        "flex flex-1 flex-col items-center gap-1 rounded-lg px-1 py-1.5 text-[10px] font-medium text-muted-foreground transition-colors hover:bg-accent hover:text-foreground disabled:opacity-60",
        danger && "hover:bg-destructive/10 hover:text-destructive",
      )}
    >
      <Icon className={cn("h-[18px] w-[18px]", spinning && "animate-spin")} />
      {label}
    </button>
  );
}
