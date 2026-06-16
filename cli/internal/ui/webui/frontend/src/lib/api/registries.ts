import { BASE, checkedFetch, postJSON } from "./http";

export async function fetchRegistries(): Promise<string[]> {
  const resp = await checkedFetch(`${BASE}/api/registries`);
  return ((await resp.json()) as string[] | null) ?? [];
}

export function addRegistry(url: string, token?: string): Promise<void> {
  return postJSON("/api/registries/add", { url, token: token ?? "" });
}

export function removeRegistry(url: string): Promise<void> {
  return postJSON("/api/registries/remove", { url });
}

export async function refreshRegistries(): Promise<number> {
  const resp = await checkedFetch(`${BASE}/api/registries/refresh`, { method: "POST" });
  const data = (await resp.json()) as { plugins?: number };
  return data.plugins ?? 0;
}
