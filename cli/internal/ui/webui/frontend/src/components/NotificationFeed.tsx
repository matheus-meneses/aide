import { useState } from "react";
import { AlertCircle, Bell, BellOff, CheckCircle, Info, X, AlertTriangle } from "lucide-react";
import { type AgentEvent, describeEvent } from "@/hooks/useSSE";
import { type NotificationState } from "@/lib/notifications";
import { APP_NAME } from "@/lib/brand";
import { EmptyState, useToast } from "@/components/ui";

interface Props {
  events: AgentEvent[];
  onEventClick?: (event: AgentEvent) => void;
  onDismiss?: (event: AgentEvent) => void;
  notificationPermission?: NotificationState;
  onEnableNotifications?: () => void;
}

function PermissionBanner({
  permission,
  onEnable,
}: {
  permission?: NotificationState;
  onEnable?: () => void;
}) {
  const [showHelp, setShowHelp] = useState(false);

  if (permission === "default") {
    return (
      <button
        onClick={onEnable}
        className="flex w-full items-center gap-2 border-b bg-primary/5 px-3 py-2 text-left text-xs font-medium text-primary transition-colors hover:bg-primary/10"
      >
        <Bell className="h-3.5 w-3.5 shrink-0" />
        Enable notifications to get alerts for new items and briefings
      </button>
    );
  }

  if (permission === "denied") {
    return (
      <div className="border-b bg-warning/5 px-3 py-2 text-xs text-warning">
        <div className="flex items-start gap-2">
          <BellOff className="mt-0.5 h-3.5 w-3.5 shrink-0" />
          <span className="flex-1">Notifications are blocked for {APP_NAME}.</span>
          <button
            onClick={() => setShowHelp((v) => !v)}
            className="shrink-0 font-medium underline underline-offset-2 hover:text-warning/80"
          >
            {showHelp ? "Hide" : "How to enable"}
          </button>
        </div>
        {showHelp && (
          <ol className="mt-2 list-decimal space-y-1 pl-7 text-warning/90">
            <li>Click the lock or site icon to the left of the address bar.</li>
            <li>
              Find <span className="font-medium">Notifications</span> and switch it to{" "}
              <span className="font-medium">Allow</span>.
            </li>
            <li>Reload the page.</li>
          </ol>
        )}
      </div>
    );
  }

  return null;
}

function parseEventData(data: string): { title?: string; body?: string; fingerprint?: string } {
  try {
    return JSON.parse(data) as { title?: string; body?: string; fingerprint?: string };
  } catch {
    return { body: data };
  }
}

const displayContent = describeEvent;

async function ackAlert(fingerprint: string, title: string) {
  const resp = await fetch("/api/ack", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ fingerprint, title }),
  });
  if (!resp.ok) {
    throw new Error(`ack failed: HTTP ${resp.status}`);
  }
}

function stableKey(event: AgentEvent): string {
  if (typeof event.id === "number" && event.id > 0) return String(event.id);
  const parsed = parseEventData(event.data);
  return parsed.fingerprint || `${event.type}-${event.timestamp}`;
}

function EventIcon({ type, priority }: { type: string; priority?: string }) {
  if (priority === "urgent") return <AlertTriangle className="w-4 h-4 text-destructive shrink-0" />;
  switch (type) {
    case "notification":
      return <AlertCircle className="w-4 h-4 text-warning shrink-0" />;
    case "briefing":
      return <Info className="w-4 h-4 text-info shrink-0" />;
    case "scrape_complete":
      return <CheckCircle className="w-4 h-4 text-success shrink-0" />;
    case "cycle_error":
      return <AlertTriangle className="w-4 h-4 text-destructive shrink-0" />;
    default:
      return <Bell className="w-4 h-4 text-muted-foreground shrink-0" />;
  }
}

function priorityClass(priority?: string): string {
  switch (priority) {
    case "urgent":
      return "border-l-2 border-l-destructive";
    case "normal":
      return "border-l-2 border-l-info";
    default:
      return "";
  }
}

function timeAgo(timestamp: string): string {
  const diff = Date.now() - new Date(timestamp).getTime();
  const mins = Math.floor(diff / 60000);
  if (mins < 1) return "now";
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  return `${Math.floor(hrs / 24)}d ago`;
}

export function NotificationFeed({
  events,
  onEventClick,
  onDismiss,
  notificationPermission,
  onEnableNotifications,
}: Props) {
  const { toast } = useToast();

  const visibleEvents = events.filter((event) => {
    const { title, body } = displayContent(event);
    return title !== "" || body !== "";
  });

  if (visibleEvents.length === 0) {
    return (
      <div className="flex flex-col h-full">
        <PermissionBanner permission={notificationPermission} onEnable={onEnableNotifications} />
        <div className="flex flex-1 items-center justify-center p-4">
          <EmptyState
            icon={Bell}
            title={events.length === 0 ? "No notifications yet" : "All caught up"}
            description={
              events.length === 0 ? `${APP_NAME} events will appear here` : "Acknowledged everything"
            }
            className="border-0"
          />
        </div>
      </div>
    );
  }

  const handleDismiss = (event: AgentEvent) => {
    const key = stableKey(event);
    const parsed = parseEventData(event.data);
    const fp = parsed.fingerprint || key;
    const { title, body } = displayContent(event);
    onDismiss?.(event);
    ackAlert(fp, title || body).catch(() => {
      toast("Failed to acknowledge on the server.", "error");
    });
  };

  return (
    <div className="flex flex-col h-full">
      <PermissionBanner permission={notificationPermission} onEnable={onEnableNotifications} />
      <div className="flex flex-col gap-1 p-2 flex-1 overflow-y-auto scrollbar-thin">
        {visibleEvents.map((event) => {
          const { title, body } = displayContent(event);
          return (
            <div
              key={stableKey(event)}
              role="button"
              tabIndex={0}
              className={`flex gap-2 p-2 rounded-md hover:bg-accent/50 transition-colors animate-slide-in group cursor-pointer focus:outline-none focus:ring-2 focus:ring-primary/30 ${priorityClass(event.priority)}`}
              onClick={() => onEventClick?.(event)}
              onKeyDown={(e) => {
                if (e.key === "Enter" || e.key === " ") {
                  e.preventDefault();
                  onEventClick?.(event);
                }
              }}
              aria-label={`${event.type} notification${title ? ": " + title : ""}`}
            >
              <EventIcon type={event.type} priority={event.priority} />
              <div className="flex-1 min-w-0">
                {title && <div className="text-xs font-medium truncate">{title}</div>}
                {body && <div className="text-xs text-muted-foreground line-clamp-2">{body}</div>}
              </div>
              <div className="flex items-center gap-1 shrink-0">
                <span className="text-[10px] text-muted-foreground">
                  {timeAgo(event.timestamp)}
                </span>
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    handleDismiss(event);
                  }}
                  className="opacity-100 sm:opacity-0 sm:group-hover:opacity-100 p-0.5 rounded hover:bg-accent transition-all"
                  title="Acknowledge"
                  aria-label="Dismiss notification"
                >
                  <X className="w-3 h-3 text-muted-foreground" />
                </button>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}
