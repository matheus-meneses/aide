interface Props {
  data?: {
    created_at?: string;
    last_scrape_at?: string;
    content?: string;
  };
  text?: string;
}

function timeAgo(isoStr: string): string {
  if (!isoStr) return "";
  const diff = Date.now() - new Date(isoStr).getTime();
  const mins = Math.floor(diff / 60000);
  if (mins < 1) return "just now";
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  return `${Math.floor(hrs / 24)}d ago`;
}

export function MemoryView({ data, text }: Props) {
  if (text) {
    return (
      <div className="rounded-lg border bg-card p-4 text-sm text-muted-foreground">{text}</div>
    );
  }

  if (!data?.content) {
    return (
      <div className="rounded-lg border bg-card p-4 text-sm text-muted-foreground">
        No memory stored yet. The agent will save its first memory after the next cycle.
      </div>
    );
  }

  const parts = data.content.split(" | ");

  return (
    <div className="rounded-lg border bg-card overflow-hidden">
      <div className="px-3 py-2 border-b bg-accent/30 flex items-center justify-between">
        <span className="text-xs font-medium">Agent Memory</span>
        {data.created_at && (
          <span className="text-[10px] text-muted-foreground">
            saved {timeAgo(data.created_at)}
          </span>
        )}
      </div>
      <div className="p-3 space-y-1.5">
        {parts.map((part, i) => (
          <div key={i} className="text-sm flex items-start gap-2">
            <span className="text-muted-foreground shrink-0">•</span>
            <span>{part.trim()}</span>
          </div>
        ))}
      </div>
      {data.last_scrape_at && (
        <div className="px-3 py-2 border-t text-[10px] text-muted-foreground">
          Last scrape: {timeAgo(data.last_scrape_at)}
        </div>
      )}
    </div>
  );
}
