import { BASE, checkedFetch, postJSON } from "./http";

export interface NextEvent {
  title: string;
  member: string;
  time: string;
  start: string;
  minutes_until: number;
  in_progress: boolean;
}

export async function fetchNextEvent(): Promise<NextEvent | null> {
  const resp = await checkedFetch(`${BASE}/api/events/next`);
  return (await resp.json()) as NextEvent | null;
}

export type UICommandAction = "show" | "navigate" | "quit";

export async function sendUICommand(action: UICommandAction, view?: string): Promise<void> {
  await postJSON("/api/ui/command", { action, view: view ?? "" });
}

export async function triggerSync(): Promise<void> {
  await postJSON("/api/ui/sync", {});
}
