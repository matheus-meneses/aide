import { BASE, checkedFetch } from "./http";

export interface SetupStatus {
  needs_setup: boolean;
  python_ready: boolean;
}

export async function fetchSetupStatus(): Promise<SetupStatus> {
  const resp = await checkedFetch(`${BASE}/api/setup/status`);
  return resp.json() as Promise<SetupStatus>;
}

export async function startBootstrap(): Promise<void> {
  await checkedFetch(`${BASE}/api/setup/bootstrap`, { method: "POST" });
}

export async function completeSetup(): Promise<void> {
  await checkedFetch(`${BASE}/api/setup/complete`, { method: "POST" });
}
