import { useCallback, useEffect, useState } from "react";
import { AlertTriangle, Check, CheckCircle2, Clock, ExternalLink, Info, RefreshCw, X } from "lucide-react";
import { fetchItems, markItemDone } from "@/lib/api";
import type { ItemData as Item } from "@/lib/api";
import { handleExternalClick } from "@/lib/openExternal";
import { Button, EmptyState, Select, Skeleton, useToast } from "@/components/ui";

interface Props {
  source: string;
  onClose: () => void;
  onItemDone?: () => void;
}

const INFO_PRIORITY = { icon: Info, color: "text-info", bg: "bg-info/10" };

const priorityConfig: Record<string, { icon: typeof AlertTriangle; color: string; bg: string }> = {
  critical: { icon: AlertTriangle, color: "text-destructive", bg: "bg-destructive/10" },
  warning: { icon: AlertTriangle, color: "text-warning", bg: "bg-warning/10" },
  info: INFO_PRIORITY,
  low: { icon: CheckCircle2, color: "text-success", bg: "bg-success/10" },
};

const categoryLabels: Record<string, string> = {
  approval: "Approval",
  event: "Event",
  task: "Task",
  alert: "Alert",
  absence: "Absence",
  issue: "Issue",
  mr: "Merge Request",
  pipeline: "Pipeline",
  email: "Email",
};

export function ItemsView({ source, onClose, onItemDone }: Props) {
  const [items, setItems] = useState<Item[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [groupBy, setGroupBy] = useState<"category" | "member" | "priority">("category");
  const { toast } = useToast();

  const displaySource =
    source === "__all" ? undefined : source === "__meetings" ? undefined : source;

  const load = useCallback(() => {
    setLoading(true);
    setError(null);
    fetchItems(displaySource)
      .then((data) => {
        if (source === "__meetings") {
          setItems(data.filter((i) => i.category === "event"));
        } else {
          setItems(data);
        }
      })
      .catch((e: unknown) => {
        const message = e instanceof Error ? e.message : String(e);
        setError(message);
        toast(`Failed to load items: ${message}`, "error");
      })
      .finally(() => setLoading(false));
  }, [displaySource, source, toast]);

  useEffect(() => {
    load();
  }, [load]);

  const handleDone = useCallback(
    (item: Item) => {
      setItems((prev) => prev.filter((i) => i.id !== item.id));
      markItemDone(item.fingerprint)
        .then(() => onItemDone?.())
        .catch((e: unknown) => {
          const message = e instanceof Error ? e.message : String(e);
          toast(`Failed to mark done: ${message}`, "error");
          load();
        });
    },
    [load, onItemDone, toast],
  );

  const grouped = items.reduce<Record<string, Item[]>>((acc, item) => {
    const key =
      groupBy === "category"
        ? categoryLabels[item.category] || item.category
        : groupBy === "member"
          ? item.member || "Unknown"
          : item.priority || "info";
    if (!acc[key]) acc[key] = [];
    acc[key].push(item);
    return acc;
  }, {});

  const sortedGroups = Object.entries(grouped).sort(([a], [b]) => {
    if (groupBy === "priority") {
      const order = ["critical", "warning", "info", "low"];
      return order.indexOf(a) - order.indexOf(b);
    }
    return a.localeCompare(b);
  });

  const title =
    source === "__all" ? "All Items" : source === "__meetings" ? "Today's Meetings" : source;

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-4 py-3 border-b bg-card">
        <div className="flex items-center gap-3">
          <h2 className="text-sm font-semibold">{title}</h2>
          <span className="text-xs text-muted-foreground px-2 py-0.5 rounded-full bg-muted">
            {items.length} items
          </span>
        </div>
        <div className="flex items-center gap-2">
          <Select
            value={groupBy}
            onChange={(e) => setGroupBy(e.target.value as typeof groupBy)}
            className="w-auto px-2 py-1 text-xs"
          >
            <option value="category">Group by category</option>
            <option value="member">Group by member</option>
            <option value="priority">Group by priority</option>
          </Select>
          <button
            onClick={load}
            className="p-1.5 rounded hover:bg-accent transition-colors"
            title="Refresh"
            aria-label="Refresh items"
          >
            <RefreshCw className={`w-3.5 h-3.5 ${loading ? "animate-spin" : ""}`} />
          </button>
          <button
            onClick={onClose}
            className="p-1.5 rounded hover:bg-accent transition-colors"
            aria-label="Close panel"
          >
            <X className="w-4 h-4" />
          </button>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto scrollbar-thin p-4">
        {loading && items.length === 0 && (
          <div className="space-y-1.5 max-w-4xl mx-auto" aria-busy="true" aria-label="Loading items">
            {Array.from({ length: 6 }).map((_, i) => (
              <Skeleton key={i} className="h-16 w-full" />
            ))}
          </div>
        )}

        {!loading && error && (
          <EmptyState
            icon={AlertTriangle}
            title="Couldn't load items"
            description={error}
            action={
              <Button size="sm" variant="secondary" onClick={load}>
                Retry
              </Button>
            }
          />
        )}

        {!loading && items.length === 0 && !error && (
          <EmptyState icon={CheckCircle2} title="No open items" description="You're all caught up." />
        )}

        <div className="space-y-4 max-w-4xl mx-auto">
          {sortedGroups.map(([group, groupItems]) => (
            <div key={group}>
              <div className="flex items-center gap-2 mb-2">
                <h3 className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
                  {group}
                </h3>
                <span className="text-xs text-muted-foreground">({groupItems.length})</span>
              </div>
              <div className="space-y-1.5">
                {groupItems.map((item) => {
                  const prio = priorityConfig[item.priority] ?? INFO_PRIORITY;
                  const PrioIcon = prio.icon;
                  return (
                    <div
                      key={item.id}
                      className="flex items-start gap-3 rounded-lg border p-3 hover:bg-accent/30 transition-colors group"
                    >
                      <div className={`mt-0.5 p-1 rounded ${prio.bg}`}>
                        <PrioIcon className={`w-3.5 h-3.5 ${prio.color}`} />
                      </div>
                      <div className="flex-1 min-w-0">
                        <div className="flex items-start justify-between gap-2">
                          <p className="text-sm font-medium leading-tight truncate">{item.title}</p>
                          <div className="flex items-center gap-1 shrink-0">
                            {item.link && (
                              <a
                                href={item.link}
                                target="_blank"
                                rel="noopener noreferrer"
                                onClick={(e) => handleExternalClick(e, item.link)}
                                className="p-1 rounded opacity-100 sm:opacity-0 sm:group-hover:opacity-100 hover:bg-accent transition-all"
                              >
                                <ExternalLink className="w-3.5 h-3.5 text-muted-foreground" />
                              </a>
                            )}
                            <button
                              onClick={() => handleDone(item)}
                              className="p-1 rounded opacity-100 sm:opacity-0 sm:group-hover:opacity-100 hover:bg-success/10 transition-all"
                              title="Mark done"
                              aria-label="Mark done"
                            >
                              <Check className="w-3.5 h-3.5 text-success" />
                            </button>
                          </div>
                        </div>
                        {item.detail && (
                          <p className="text-xs text-muted-foreground mt-0.5 line-clamp-2">
                            {item.detail}
                          </p>
                        )}
                        <div className="flex items-center gap-3 mt-1.5 text-xs text-muted-foreground">
                          {item.member && <span>{item.member}</span>}
                          {item.source !== source && source === "__all" && (
                            <span className="px-1.5 py-0.5 rounded bg-muted text-[10px]">
                              {item.source}
                            </span>
                          )}
                          <span className="flex items-center gap-1">
                            <Clock className="w-3 h-3" />
                            {item.entry_date}
                          </span>
                        </div>
                      </div>
                    </div>
                  );
                })}
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
