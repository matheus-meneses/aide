import { useCallback, useEffect, useMemo, useState } from "react";
import { ArrowLeft, Pause, Play, ScrollText, SearchX, Trash2, WifiOff } from "lucide-react";
import { Button, EmptyState, Input, Select } from "@/components/ui";
import { useLogStream } from "@/hooks/useLogStream";
import { useChatScroll } from "@/hooks/useChatScroll";
import { cn } from "@/lib/cn";

interface Props {
  onClose: () => void;
}

const LEVELS = ["all", "debug", "info", "warn", "error"];

const LEVEL_RANK: Record<string, number> = { debug: 10, info: 20, warn: 30, error: 40 };

const LEVEL_STYLES: Record<string, string> = {
  debug: "text-muted-foreground",
  info: "text-foreground",
  warn: "text-warning",
  error: "text-destructive",
};

function formatLocalTime(ts: string): string {
  const d = new Date(ts);
  if (Number.isNaN(d.getTime())) return ts;
  return d.toLocaleTimeString([], {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    hour12: false,
  });
}

export function LogsView({ onClose }: Props) {
  const { logs, connected, paused, clear, togglePaused } = useLogStream();
  const [minLevel, setMinLevel] = useState("all");
  const [filter, setFilter] = useState("");

  const filtered = useMemo(() => {
    const threshold = minLevel === "all" ? 0 : (LEVEL_RANK[minLevel] ?? 0);
    const needle = filter.trim().toLowerCase();
    return logs.filter((l) => {
      if ((LEVEL_RANK[l.level] ?? 0) < threshold) return false;
      if (needle && !`${l.scope} ${l.msg}`.toLowerCase().includes(needle)) return false;
      return true;
    });
  }, [logs, minLevel, filter]);

  const { scrollRef, handleScroll } = useChatScroll(filtered);

  const [confirmingPrune, setConfirmingPrune] = useState(false);

  useEffect(() => {
    if (!confirmingPrune) return;
    const t = setTimeout(() => setConfirmingPrune(false), 3000);
    return () => clearTimeout(t);
  }, [confirmingPrune]);

  const prune = useCallback(async () => {
    if (!confirmingPrune) {
      setConfirmingPrune(true);
      return;
    }
    setConfirmingPrune(false);
    try {
      await fetch("/api/logs", { method: "DELETE" });
    } catch {
      // The tail stream reflects the real file state regardless.
    }
    clear();
  }, [confirmingPrune, clear]);

  return (
    <div className="flex h-full flex-col">
      <div className="flex flex-wrap items-center gap-2 border-b bg-card px-4 py-2.5">
        <Button variant="ghost" size="sm" onClick={onClose} aria-label="Back">
          <ArrowLeft className="h-4 w-4" />
          Back
        </Button>
        <span className="text-sm font-semibold">Logs</span>
        <span
          className={cn(
            "flex items-center gap-1.5 rounded-full px-2 py-0.5 text-[11px] font-medium",
            connected ? "bg-success/15 text-success" : "bg-muted text-muted-foreground",
          )}
          title={connected ? "Streaming" : paused ? "Paused" : "Disconnected"}
        >
          <span
            className={cn(
              "h-2 w-2 rounded-full",
              connected ? "bg-success" : "bg-muted-foreground/60",
            )}
          />
          {connected ? "live" : paused ? "paused" : "offline"}
        </span>
        <div className="ml-auto flex items-center gap-2">
          <Select
            value={minLevel}
            onChange={(e) => setMinLevel(e.target.value)}
            className="h-8 w-28 py-1"
            aria-label="Minimum level"
          >
            {LEVELS.map((lvl) => (
              <option key={lvl} value={lvl}>
                {lvl === "all" ? "All levels" : lvl}
              </option>
            ))}
          </Select>
          <Input
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
            placeholder="Filter…"
            className="h-8 w-44"
            aria-label="Filter logs"
          />
          <Button variant="outline" size="sm" onClick={togglePaused}>
            {paused ? <Play className="h-4 w-4" /> : <Pause className="h-4 w-4" />}
            {paused ? "Resume" : "Pause"}
          </Button>
          <Button
            variant={confirmingPrune ? "destructive" : "outline"}
            size="sm"
            onClick={() => void prune()}
          >
            <Trash2 className="h-4 w-4" />
            {confirmingPrune ? "Confirm" : "Prune"}
          </Button>
        </div>
      </div>

      <div
        ref={scrollRef}
        onScroll={handleScroll}
        className="flex-1 overflow-auto bg-background px-4 py-2 font-mono text-xs leading-relaxed"
      >
        {filtered.length === 0 ? (
          <div className="flex h-full items-center justify-center p-4">
            {logs.length > 0 ? (
              <EmptyState
                icon={SearchX}
                title="No matching logs"
                description="No logs match the current level or filter."
                className="border-0"
              />
            ) : paused ? (
              <EmptyState
                icon={Pause}
                title="Streaming paused"
                description="Resume to keep tailing the agent log."
                action={
                  <Button size="sm" variant="secondary" onClick={togglePaused}>
                    Resume
                  </Button>
                }
                className="border-0"
              />
            ) : !connected ? (
              <EmptyState
                icon={WifiOff}
                title="Log stream disconnected"
                description="Trying to reconnect to the agent…"
                className="border-0"
              />
            ) : (
              <EmptyState
                icon={ScrollText}
                title="Waiting for logs"
                description="New agent activity will appear here."
                className="border-0"
              />
            )}
          </div>
        ) : (
          filtered.map((l, i) => (
            <div key={`${l.ts}-${i}`} className="flex gap-2 whitespace-pre-wrap break-words py-0.5">
              <span className="shrink-0 text-muted-foreground" title={l.ts}>
                {formatLocalTime(l.ts)}
              </span>
              <span className={cn("w-12 shrink-0 uppercase", LEVEL_STYLES[l.level])}>{l.level}</span>
              <span className="shrink-0 text-primary">{l.scope}</span>
              <span className="text-foreground">{l.msg}</span>
            </div>
          ))
        )}
      </div>
    </div>
  );
}
