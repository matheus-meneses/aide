import { type AgentEvent } from "@/lib/eventStore";
import { APP_NAME } from "@/lib/brand";

export const GROUPING_WINDOW_MS = 5000;

export type NotificationState = NotificationPermission | "unsupported" | "native" | "unknown";

interface ParsedEventData {
  title?: string;
  body?: string;
  sources_ok?: number;
  sources_total?: number;
  sources_failed?: number;
  error?: string;
  fingerprint?: string;
}

export function describeEvent(event: AgentEvent): { title: string; body: string } {
  let parsed: ParsedEventData;
  try {
    parsed = JSON.parse(event.data) as ParsedEventData;
  } catch {
    return { title: "", body: event.data };
  }

  if (parsed.title ?? parsed.body) {
    return { title: parsed.title ?? "", body: parsed.body ?? "" };
  }

  switch (event.type) {
    case "scrape_complete": {
      const ok = parsed.sources_ok ?? 0;
      const total = parsed.sources_total ?? 0;
      const failed = parsed.sources_failed ?? 0;
      const body =
        failed > 0
          ? `Scraped ${ok}/${total} sources, ${failed} failed`
          : `Scraped ${ok}/${total} sources`;
      return { title: "Data refreshed", body };
    }
    case "cycle_error":
      return {
        title: "Agent error",
        body: parsed.error ?? "A background action failed",
      };
    case "status":
      return { title: "Status update", body: "" };
    default:
      return { title: "", body: "" };
  }
}

export function currentPermission(): NotificationState {
  if (!("Notification" in window)) return "unsupported";
  return Notification.permission;
}

export async function requestNotificationPermission(): Promise<NotificationState> {
  if (!("Notification" in window)) return "unsupported";
  try {
    return await Notification.requestPermission();
  } catch (err: unknown) {
    console.warn("failed to request notification permission:", err);
    return Notification.permission;
  }
}

export function showBrowserNotification(event: AgentEvent) {
  if (!("Notification" in window) || Notification.permission !== "granted") return;

  let title = event.type === "briefing" ? `${APP_NAME} - Briefing` : APP_NAME;
  let tag = String(event.id || event.timestamp);

  let body: string;
  try {
    const parsed = JSON.parse(event.data) as ParsedEventData;
    if (parsed.title) title = parsed.title;
    body = parsed.body ?? "";
    if (parsed.fingerprint) tag = parsed.fingerprint;
  } catch {
    body = event.data.length > 200 ? `${event.data.slice(0, 200)}...` : event.data;
  }

  const n = new Notification(title, { body, icon: "/favicon.png", tag });
  n.onclick = () => {
    window.focus();
    n.close();
  };
}

export function shouldNotify(event: AgentEvent): boolean {
  const priority = event.priority ?? "normal";
  if (priority === "silent") return false;
  if (priority === "urgent") return true;
  return document.visibilityState === "hidden";
}

export function flushGroupedBuffer(buffer: AgentEvent[]) {
  if (buffer.length === 0) return;

  const first = buffer[0];
  if (buffer.length === 1 && first) {
    showBrowserNotification(first);
  } else if ("Notification" in window && Notification.permission === "granted") {
    new Notification(APP_NAME, {
      body: `${buffer.length} new updates`,
      icon: "/favicon.png",
      tag: "aide-grouped",
    });
  }
}
