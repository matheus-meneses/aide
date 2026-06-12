import { AlertTriangle, Bot, RotateCcw, User } from "lucide-react";
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
  onSuggestionClick?: (text: string) => void;
  onRetry?: () => void;
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
  if (format === "schedule" && data)
    return <ScheduleView data={data as unknown as ComponentProps<typeof ScheduleView>["data"]} />;
  if (format === "items" && data)
    return <ItemsView data={data as unknown as ComponentProps<typeof ItemsView>["data"]} />;
  if (format === "status" && data)
    return <StatusView data={data} />;
  if (format === "memory")
    return (
      <MemoryView
        data={data}
        text={content}
      />
    );
  if (format === "stats" && data)
    return <StatsView data={data as unknown as ComponentProps<typeof StatsView>["data"]} />;
  if (format === "team" && data)
    return <TeamView data={data as unknown as ComponentProps<typeof TeamView>["data"]} />;
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
    return (
      <pre className="text-sm whitespace-pre-wrap font-mono bg-muted rounded p-3">{content}</pre>
    );
  }
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
  onSuggestionClick,
  onRetry,
}: Props) {
  const isUser = role === "user";

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
