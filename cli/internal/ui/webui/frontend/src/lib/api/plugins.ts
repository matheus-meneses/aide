import { BASE, checkedFetch, postJSON } from "./http";

export interface PluginItem {
  name: string;
  description: string;
  runtime?: string;
  installed: boolean;
  configured: boolean;
}

export interface ManifestField {
  key: string;
  label: string;
  type: string;
  default: string;
  required: boolean;
}

export interface ManifestCredential {
  key: string;
  label: string;
  secret: boolean;
}

export interface PluginManifest {
  name: string;
  description: string;
  config: ManifestField[] | null;
  credentials: ManifestCredential[] | null;
}

export async function fetchPlugins(): Promise<PluginItem[]> {
  const resp = await checkedFetch(`${BASE}/api/plugins`);
  return resp.json() as Promise<PluginItem[]>;
}

export async function fetchPluginManifest(name: string): Promise<PluginManifest> {
  const resp = await checkedFetch(`${BASE}/api/plugins/${encodeURIComponent(name)}/manifest`);
  return resp.json() as Promise<PluginManifest>;
}

export async function installPlugin(name: string): Promise<void> {
  await checkedFetch(`${BASE}/api/plugins/install`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ name, acknowledge_capabilities: true }),
  });
}

export function uninstallPlugin(name: string): Promise<void> {
  return postJSON("/api/plugins/uninstall", { name });
}
