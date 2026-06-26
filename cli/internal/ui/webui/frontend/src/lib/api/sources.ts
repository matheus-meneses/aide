import { BASE, checkedFetch, postJSON } from "./http";

export interface SourceSnapshot {
  name: string;
  plugin?: string;
  enabled: boolean;
  config?: Record<string, unknown> | null;
  context?: string;
  has_credentials: boolean;
}

export interface SourcePayload {
  name: string;
  config: Record<string, string>;
  credentials: Record<string, string>;
}

export async function fetchSources(): Promise<SourceSnapshot[]> {
  const resp = await checkedFetch(`${BASE}/api/sources`);
  return ((await resp.json()) as SourceSnapshot[] | null) ?? [];
}

export async function addSource(payload: SourcePayload): Promise<void> {
  await checkedFetch(`${BASE}/api/sources`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
}

export function reconfigureSource(payload: SourcePayload): Promise<void> {
  return postJSON("/api/sources", payload);
}

export function toggleSource(name: string, enabled: boolean): Promise<void> {
  return postJSON("/api/sources/toggle", { name, enabled });
}

export function setSourceContext(name: string, context: string): Promise<void> {
  return postJSON("/api/sources/context", { name, context });
}
