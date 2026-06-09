export interface AgentEvent {
  id: number;
  type: string;
  data: string;
  timestamp: string;
  priority?: string;
}

export const STORAGE_KEY = "aide-notifications";
export const MAX_STORED = 100;
export const MAX_EVENTS = 50;

export function loadFromStorage(): AgentEvent[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (raw) return (JSON.parse(raw) as AgentEvent[]).slice(0, MAX_STORED);
  } catch (_) {
    // ignore parse errors
  }
  return [];
}

export function saveToStorage(events: AgentEvent[]) {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(events.slice(0, MAX_STORED)));
  } catch (_) {
    // ignore quota errors
  }
}
export function safeParse(raw: string): AgentEvent | null {
  try {
    return JSON.parse(raw) as AgentEvent;
  } catch {
    return null;
  }
}

export function eventKey(event: AgentEvent): string {
  if (event.id) return String(event.id);
  try {
    const parsed = JSON.parse(event.data) as Record<string, unknown>;
    if (typeof parsed.fingerprint === "string") return parsed.fingerprint;
  } catch (_) {
    // fall through
  }
  return `${event.type}-${event.timestamp}-${event.data.slice(0, 80)}`;
}
