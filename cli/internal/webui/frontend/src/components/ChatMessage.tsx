import { AlertTriangle, Bot, Loader2, RotateCcw, Sparkles, User } from "lucide-react";
import type { ComponentProps } from "react";
import { cn } from "@/lib/cn";
import { MarkdownRenderer } from "./renderers/MarkdownRenderer";
import { ScheduleView } from "./renderers/ScheduleView";
import { ItemsView } from "./renderers/ItemsView";
import { StatusView } from "./renderers/StatusView";
import { MemoryView } from "./renderers/MemoryView";
import { StatsView } from "./renderers/StatsView";
import { TeamView } from "./renderers/TeamView";

interface Props {
  role: "user" | "assistant";
  content: string;
  timestamp: number;
  isStreaming?: boolean;
  isError?: boolean;
  format?: string;
  data?: Record<string, unknown>;
  pending?: boolean;
  pendingLabel?: string;
  needsConfig?: boolean;
  onSuggestionClick?: (text: string) => void;
  onRetry?: () => void;
  onConfigure?: () => void;
}

function timeAgo(ts: number): string {
  const diff = Date.now() - ts;
  const mins = Math.floor(diff / 60000);
  if (mins < 1) return "just now";
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  return `${Math.floor(hrs / 24)}d ago`;
}

interface ScrapeData {
  sources_total: number;
  sources_ok: number;
  sources_failed: number;
  [key: string]: unknown;
}

function isScrapeData(d: Record<string, unknown>): d is ScrapeData {
  return typeof d.sources_total === "number";
}

function EmptyState({ label }: { label: string }) {
  return (
    <div className="rounded-lg border border-dashed bg-card/50 p-4 text-sm text-muted-foreground">
      {label}
    </div>
  );
}

function RichContent({
  format,
  data,
  content,
  onSuggestionClick,
}: {
  format?: string;
  data?: Record<string, unknown>;
  content: string;
  onSuggestionClick?: (text: string) => void;
}) {
  if (format === "schedule")
    return (
      <ScheduleView data={(data as unknown as ComponentProps<typeof ScheduleView>["data"]) ?? []} />
    );
  if (format === "items")
    return <ItemsView data={(data as unknown as ComponentProps<typeof ItemsView>["data"]) ?? []} />;
  if (format === "status")
    return data ? <StatusView data={data} /> : <EmptyState label="Nothing to report yet." />;
  if (format === "memory") return <MemoryView data={data} text={content} />;
  if (format === "stats")
    return data ? (
      <StatsView data={data as unknown as ComponentProps<typeof StatsView>["data"]} />
    ) : (
      <EmptyState label="No usage stats yet." />
    );
  if (format === "team")
    return data ? (
      <TeamView data={data as unknown as ComponentProps<typeof TeamView>["data"]} />
    ) : (
      <EmptyState label="No team members yet." />
    );
  if (format === "scrape" && data && isScrapeData(data)) {
    return (
      <div className="rounded-lg border bg-card p-3 text-sm">
        <div className="font-medium mb-1">Scrape Complete</div>
        <div className="text-muted-foreground">
          {data.sources_total} sources: {data.sources_ok} OK, {data.sources_failed} failed
        </div>
      </div>
    );
  }
  if (format === "text") {
    if (!content.trim()) return <EmptyState label="Nothing to show." />;
    return (
      <pre className="text-sm whitespace-pre-wrap font-mono bg-muted rounded p-3">{content}</pre>
    );
  }
  if (!content.trim()) return <EmptyState label="Nothing to show." />;
  return <MarkdownRenderer content={content} onSuggestionClick={onSuggestionClick} />;
}

export function ChatMessage({
  role,
  content,
  timestamp,
  isStreaming,
  isError,
  format,
  data,
  pending,
  pendingLabel,
  needsConfig,
  onSuggestionClick,
  onRetry,
  onConfigure,
}: Props) {
  const isUser = role === "user";

  if (pending) {
    return (
      <div className="group flex gap-3 px-4 py-3">
        <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-accent">
          <Bot className="h-4 w-4" />
        </div>
        <div className="flex items-center gap-2 rounded-lg border bg-card px-3 py-2 text-sm text-muted-foreground">
          <Loader2 className="h-3.5 w-3.5 animate-spin" />
          {pendingLabel ?? "Working…"}
        </div>
      </div>
    );
  }

  if (needsConfig) {
    return (
      <div className="flex gap-3 px-4 py-3">
        <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-amber-500/15">
          <Sparkles className="h-4 w-4 text-amber-500" />
        </div>
        <div className="max-w-[80%] space-y-2 rounded-lg border border-amber-500/30 bg-amber-500/5 px-3 py-2.5 text-sm">
          <p className="text-foreground/90">
            I'm not connected to an AI model yet, so I can't answer that. Connect a model and I'll
            start triaging your tasks, meetings, and notifications.
          </p>
          {onConfigure && (
            <button
              onClick={onConfigure}
              className="inline-flex items-center gap-1.5 rounded-md bg-primary px-3 py-1.5 text-xs font-medium text-primary-foreground transition-colors hover:bg-primary/90"
            >
              <Sparkles className="h-3.5 w-3.5" /> Configure agent
            </button>
          )}
        </div>
      </div>
    );
  }

  if (isError) {
    return (
      <div className="flex gap-3 px-4 py-3">
        <div className="w-7 h-7 rounded-full flex items-center justify-center shrink-0 bg-red-500/10">
          <AlertTriangle className="w-4 h-4 text-red-500" />
        </div>
        <div className="max-w-[80%] rounded-lg px-3 py-2 text-sm border border-red-500/30 bg-red-500/5">
          <p className="text-red-600 dark:text-red-400">{content}</p>
          {onRetry && (
            <button
              onClick={onRetry}
              className="mt-1.5 flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors"
            >
              <RotateCcw className="w-3 h-3" /> Retry
            </button>
          )}
        </div>
      </div>
    );
  }

  return (
    <div className={cn("group flex gap-3 px-4 py-3", isUser && "flex-row-reverse")}>
      <div
        className={cn(
          "w-7 h-7 rounded-full flex items-center justify-center shrink-0",
          isUser ? "bg-primary text-primary-foreground" : "bg-accent",
        )}
      >
        {isUser ? <User className="w-4 h-4" /> : <Bot className="w-4 h-4" />}
      </div>
      <div className="flex flex-col gap-0.5 max-w-[80%]">
        <div
          className={cn(
            "rounded-lg px-3 py-2 text-sm",
            isUser
              ? "bg-primary text-primary-foreground"
              : format && format !== "markdown"
                ? ""
                : "bg-card border",
          )}
        >
          {isUser ? (
            <p className="whitespace-pre-wrap">{content}</p>
          ) : (
            <div>
              <RichContent
                format={format}
                data={data}
                content={content}
                onSuggestionClick={onSuggestionClick}
              />
              {isStreaming && content && (
                <span className="inline-block w-1.5 h-4 bg-foreground/60 animate-pulse ml-0.5" />
              )}
            </div>
          )}
          {isStreaming && !content && (
            <div className="flex gap-1 py-1">
              <span className="w-1.5 h-1.5 bg-muted-foreground rounded-full animate-bounce [animation-delay:0ms]" />
              <span className="w-1.5 h-1.5 bg-muted-foreground rounded-full animate-bounce [animation-delay:150ms]" />
              <span className="w-1.5 h-1.5 bg-muted-foreground rounded-full animate-bounce [animation-delay:300ms]" />
            </div>
          )}
        </div>
        <span className="text-[10px] text-muted-foreground opacity-0 group-hover:opacity-100 transition-opacity px-1">
          {timeAgo(timestamp)}
        </span>
      </div>
    </div>
  );
}
