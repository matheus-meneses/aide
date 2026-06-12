interface HealthEntry {
  source: string;
  status: string;
  last_run: string;
  entries_count: number;
}

interface Props {
  data: {
    counts?: Record<string, number>;
    health?: HealthEntry[];
    today_events?: number;
  };
}

function statusDot(status: string) {
  if (status === "ok" || status === "success") return "bg-green-500";
  if (status === "error" || status === "failed") return "bg-red-500";
  return "bg-muted-foreground";
}

function timeAgo(isoStr: string): string {
  if (!isoStr) return "never";
  const diff = Date.now() - new Date(isoStr).getTime();
  const mins = Math.floor(diff / 60000);
  if (mins < 1) return "just now";
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  return `${Math.floor(hrs / 24)}d ago`;
}

export function StatusView({ data }: Props) {
  const counts = data?.counts || {};
  const health = data?.health || [];
  const total = Object.values(counts).reduce((a, b) => a + b, 0);

  return (
    <div className="rounded-lg border bg-card overflow-hidden">
      <div className="px-3 py-2 border-b bg-accent/30 flex items-center justify-between">
        <span className="text-xs font-medium">Status</span>
        <span className="text-[10px] text-muted-foreground">{total} total open</span>
      </div>

      {Object.keys(counts).length > 0 && (
        <div className="grid grid-cols-2 sm:grid-cols-3 gap-px bg-border">
          {Object.entries(counts).map(([source, count]) => {
            const h = health.find((x) => x.source === source);
            return (
              <div key={source} className="bg-card px-3 py-2.5">
                <div className="flex items-center gap-1.5">
                  {h && <div className={`w-1.5 h-1.5 rounded-full ${statusDot(h.status)}`} />}
                  <span className="text-xs text-muted-foreground">{source}</span>
                </div>
                <div className="text-lg font-semibold mt-0.5">{count}</div>
                {h && (
                  <div className="text-[10px] text-muted-foreground">{timeAgo(h.last_run)}</div>
                )}
              </div>
            );
          })}
        </div>
      )}

      {data?.today_events != null && (
        <div className="px-3 py-2 border-t text-xs text-muted-foreground">
          {data.today_events} meeting{data.today_events !== 1 ? "s" : ""} today
        </div>
      )}
    </div>
  );
}
