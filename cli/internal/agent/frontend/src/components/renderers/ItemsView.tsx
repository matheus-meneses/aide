import { useState } from "react";
import { cn } from "@/lib/cn";

interface Item {
  source: string;
  category: string;
  title: string;
  detail: string;
  priority: string;
  link: string;
}

interface Props {
  data: Item[];
}

const PRIORITY_COLORS: Record<string, string> = {
  critical: "bg-red-500/10 text-red-600 dark:text-red-400 border-l-red-500",
  high: "bg-amber-500/10 text-amber-600 dark:text-amber-400 border-l-amber-500",
  medium: "bg-blue-500/10 text-blue-600 dark:text-blue-400 border-l-blue-500",
  low: "bg-muted text-muted-foreground border-l-muted-foreground/30",
  info: "bg-muted text-muted-foreground border-l-muted-foreground/30",
};

const CATEGORY_COLORS: Record<string, string> = {
  bug: "bg-red-500/10 text-red-600 dark:text-red-400",
  task: "bg-blue-500/10 text-blue-600 dark:text-blue-400",
  story: "bg-purple-500/10 text-purple-600 dark:text-purple-400",
  approval: "bg-amber-500/10 text-amber-600 dark:text-amber-400",
  event: "bg-green-500/10 text-green-600 dark:text-green-400",
};

export function ItemsView({ data }: Props) {
  const [expanded, setExpanded] = useState(false);
  const LIMIT = 10;

  if (!data || data.length === 0) {
    return (
      <div className="rounded-lg border bg-card p-4 text-sm text-muted-foreground">
        No open items.
      </div>
    );
  }

  const grouped = data.reduce<Record<string, Item[]>>((acc, item) => {
    const key = item.source || "unknown";
    if (!acc[key]) acc[key] = [];
    acc[key].push(item);
    return acc;
  }, {});

  const visible = expanded ? data : data.slice(0, LIMIT);
  const visibleGrouped = visible.reduce<Record<string, Item[]>>((acc, item) => {
    const key = item.source || "unknown";
    if (!acc[key]) acc[key] = [];
    acc[key].push(item);
    return acc;
  }, {});

  return (
    <div className="rounded-lg border bg-card overflow-hidden">
      <div className="px-3 py-2 border-b bg-accent/30 flex items-center justify-between">
        <span className="text-xs font-medium">Open Items ({data.length})</span>
        <div className="flex gap-1.5">
          {Object.entries(grouped).map(([source, items]) => (
            <span
              key={source}
              className="text-[10px] px-1.5 py-0.5 rounded bg-muted text-muted-foreground"
            >
              {source}: {items.length}
            </span>
          ))}
        </div>
      </div>
      <div className="divide-y">
        {Object.entries(visibleGrouped).map(([source, items]) => (
          <div key={source}>
            <div className="px-3 py-1.5 bg-accent/20 text-[10px] font-medium uppercase tracking-wider text-muted-foreground">
              {source}
            </div>
            {items.map((item, i) => (
              <div
                key={i}
                className={cn(
                  "flex items-center gap-2 px-3 py-2 text-sm border-l-2",
                  PRIORITY_COLORS[item.priority] || PRIORITY_COLORS.info,
                )}
              >
                <span
                  className={cn(
                    "text-[10px] px-1.5 py-0.5 rounded shrink-0",
                    CATEGORY_COLORS[item.category] || "bg-muted text-muted-foreground",
                  )}
                >
                  {item.category}
                </span>
                <span className="flex-1 truncate">{item.title}</span>
              </div>
            ))}
          </div>
        ))}
      </div>
      {data.length > LIMIT && !expanded && (
        <button
          onClick={() => setExpanded(true)}
          className="w-full px-3 py-2 text-xs text-muted-foreground hover:text-foreground hover:bg-accent/50 transition-colors border-t"
        >
          Show all {data.length} items
        </button>
      )}
    </div>
  );
}
