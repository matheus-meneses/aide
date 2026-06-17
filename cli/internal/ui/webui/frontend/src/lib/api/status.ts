import type { AgentEvent } from "@/lib/eventStore";
import { BASE, checkedFetch, postJSON } from "./http";

export interface StatusData {
  connected: boolean;
  sources: string[];
  [key: string]: unknown;
}

export interface RuntimeInfo {
  native_notifications: boolean;
}

export interface WhoAmI {
  name?: string;
  email?: string;
  preferred_name?: string;
}

export async function fetchWhoami(): Promise<WhoAmI> {
  const resp = await checkedFetch(`${BASE}/api/whoami`);
  return resp.json() as Promise<WhoAmI>;
}

export function setWhoami(payload: WhoAmI): Promise<void> {
  return postJSON("/api/whoami", payload);
}

export interface VersionInfo {
  current: string;
  latest: string;
  update_available: boolean;
  update_url: string;
  can_self_update: boolean;
  notes: string;
  release_url: string;
  platform: string;
}

export async function fetchRuntime(): Promise<RuntimeInfo> {
  const resp = await checkedFetch(`${BASE}/api/runtime`);
  return resp.json() as Promise<RuntimeInfo>;
}

export async function fetchStatus(): Promise<StatusData> {
  const resp = await checkedFetch(`${BASE}/api/status`);
  return resp.json() as Promise<StatusData>;
}

export async function fetchSessions(): Promise<unknown> {
  const resp = await checkedFetch(`${BASE}/api/sessions`);
  return resp.json();
}

export async function fetchVersion(): Promise<VersionInfo> {
  const resp = await checkedFetch(`${BASE}/api/version`);
  return resp.json() as Promise<VersionInfo>;
}

export async function checkVersion(): Promise<VersionInfo> {
  const resp = await checkedFetch(`${BASE}/api/version/check`);
  return resp.json() as Promise<VersionInfo>;
}

export function triggerUpdate(): Promise<void> {
  return postJSON("/api/update", {});
}

export async function fetchNotifications(
  limit = 50,
): Promise<{ events: AgentEvent[]; total: number }> {
  const resp = await checkedFetch(`${BASE}/api/notifications?limit=${limit}`);
  return resp.json() as Promise<{ events: AgentEvent[]; total: number }>;
}
