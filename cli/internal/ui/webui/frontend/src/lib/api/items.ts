import { BASE, checkedFetch, postJSON } from "./http";

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

export async function fetchItems(source?: string): Promise<ItemData[]> {
  const params = source ? `?source=${encodeURIComponent(source)}` : "";
  const resp = await checkedFetch(`${BASE}/api/items${params}`);
  return resp.json() as Promise<ItemData[]>;
}

export async function fetchToday(): Promise<ItemData[]> {
  const resp = await checkedFetch(`${BASE}/api/today`);
  return resp.json() as Promise<ItemData[]>;
}

export async function markItemDone(fingerprint: string): Promise<void> {
  await postJSON("/api/items/done", { fingerprint });
}
