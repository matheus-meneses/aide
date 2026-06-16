import { cn } from "@/lib/cn";

interface Event {
  title: string;
  detail: string;
  category: string;
  entry_date: string;
  priority: string;
}

interface Props {
  data: Event[];
}

function parseTime(detail: string): string {
  const match = detail.match(/^(\d{1,2}:\d{2})/);
  return match?.[1] ?? "";
}

function parseDuration(detail: string): string {
  const match = detail.match(/\((\d+\s*(?:min|h|hr)(?:s)?)\)/i);
  return match?.[1] ?? "";
}

function isOutOfOffice(title: string): boolean {
  return /out of office|pto|vacation|off/i.test(title);
}

export function ScheduleView({ data }: Props) {
  if (data.length === 0) {
    return (
      <div className="rounded-lg border bg-card p-4 text-sm text-muted-foreground">
        No meetings today.
      </div>
    );
  }

  return (
    <div className="rounded-lg border bg-card overflow-hidden">
      <div className="px-3 py-2 border-b bg-accent/30">
        <span className="text-xs font-medium">Today's Schedule ({data.length})</span>
      </div>
      <div className="divide-y">
        {data.map((event, i) => {
          const time = parseTime(event.detail);
          const duration = parseDuration(event.detail);
          const ooo = isOutOfOffice(event.title);

          return (
            <div
              key={i}
              className={cn(
                "flex items-center gap-3 px-3 py-2.5 text-sm",
                "border-l-2",
                ooo ? "border-l-muted-foreground/30" : "border-l-blue-500/60",
              )}
            >
              <span className="font-mono text-xs text-muted-foreground w-12 shrink-0">{time}</span>
              <span className={cn("flex-1 truncate", ooo && "text-muted-foreground")}>
                {event.title}
              </span>
              {duration && (
                <span className="text-[10px] px-1.5 py-0.5 rounded bg-muted text-muted-foreground shrink-0">
                  {duration}
                </span>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}
