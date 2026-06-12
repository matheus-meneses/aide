import type { AgentEvent } from "@/lib/eventStore";

const BASE = "";

async function checkedFetch(url: string, init?: RequestInit): Promise<Response> {
  const resp = await fetch(url, init);
  if (!resp.ok) {
    const text = await resp.text().catch(() => "");
    throw new Error(text || `HTTP ${resp.status}`);
  }
  return resp;
}

export interface StatusData {
  connected: boolean;
  sources: string[];
  [key: string]: unknown;
}

export interface ItemData {
  id: number;
  fingerprint: string;
  source: string;
  member: string;
  category: string;
  title: string;
  detail: string;
  entry_date: string;
  priority: string;
  link: string;
  status: string;
  first_seen_at: string;
  last_seen_at: string;
}

export interface WhoAmI {
  preferred_name?: string;
  [key: string]: unknown;
}

export async function fetchStatus(): Promise<StatusData> {
  const resp = await checkedFetch(`${BASE}/api/status`);
  return resp.json() as Promise<StatusData>;
}

export async function fetchItems(source?: string): Promise<ItemData[]> {
  const params = source ? `?source=${encodeURIComponent(source)}` : "";
  const resp = await checkedFetch(`${BASE}/api/items${params}`);
  return resp.json() as Promise<ItemData[]>;
}

export async function fetchToday(): Promise<ItemData[]> {
  const resp = await checkedFetch(`${BASE}/api/today`);
  return resp.json() as Promise<ItemData[]>;
}

export async function fetchSessions(): Promise<unknown> {
  const resp = await checkedFetch(`${BASE}/api/sessions`);
  return resp.json();
}

export interface VersionInfo {
  current: string;
  latest: string;
  update_available: boolean;
  update_url: string;
}

export async function fetchVersion(): Promise<VersionInfo> {
  const resp = await checkedFetch(`${BASE}/api/version`);
  return resp.json() as Promise<VersionInfo>;
}

export async function fetchNotifications(
  limit = 50,
): Promise<{ events: AgentEvent[]; total: number }> {
  const resp = await checkedFetch(`${BASE}/api/notifications?limit=${limit}`);
  return resp.json() as Promise<{ events: AgentEvent[]; total: number }>;
}
