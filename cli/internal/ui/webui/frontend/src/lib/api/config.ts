import { BASE, checkedFetch, postJSON } from "./http";
import type { SourceSnapshot } from "./sources";

export interface TLSSettings {
  verify_ssl?: boolean | null;
  ca_bundle?: string;
}

export interface GeneralSettings {
  concurrency: number;
  timeout_seconds: number;
  data_dir: string;
  log_level: string;
  log_format: string;
  auto_update?: string;
  tls?: TLSSettings | null;
}

export interface AgentPreferences {
  notifications: string;
  max_notifications_per_cycle: number;
  tone: string;
}

export interface AgentSnapshot {
  provider: string;
  base_url: string;
  model: string;
  run_interval: string;
  briefing_times: string[] | null;
  has_api_key: boolean;
  user_context: string;
  preferences: AgentPreferences;
}

export interface ConfigSnapshot {
  settings: GeneralSettings;
  agent: AgentSnapshot;
  sources: SourceSnapshot[] | null;
  registries: string[] | null;
}

export interface GeneralSettingsInput {
  concurrency: number;
  timeout_seconds: number;
  verify_ssl?: boolean | null;
  ca_bundle?: string;
  log_level: string;
  log_format: string;
  auto_update?: string;
}

export async function fetchConfig(): Promise<ConfigSnapshot> {
  const resp = await checkedFetch(`${BASE}/api/config`);
  return resp.json() as Promise<ConfigSnapshot>;
}

export function setSettings(payload: GeneralSettingsInput): Promise<void> {
  return postJSON("/api/settings", payload);
}

export function setSchedule(payload: {
  run_interval: string;
  briefing_times: string[];
}): Promise<void> {
  return postJSON("/api/agent/schedule", payload);
}

export function setUserContext(context: string): Promise<void> {
  return postJSON("/api/agent/context", { context });
}

export function setAgentPreferences(payload: AgentPreferences): Promise<void> {
  return postJSON("/api/agent/preferences", payload);
}
