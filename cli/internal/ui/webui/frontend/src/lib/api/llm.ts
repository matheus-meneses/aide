import { BASE, checkedFetch } from "./http";

export interface ProviderInfo {
  id: string;
  label: string;
  default_url: string;
}

export interface LLMPayload {
  provider: string;
  base_url: string;
  model: string;
  api_key: string;
}

export async function fetchProviders(): Promise<ProviderInfo[]> {
  const resp = await checkedFetch(`${BASE}/api/providers`);
  return resp.json() as Promise<ProviderInfo[]>;
}

export async function setLLM(payload: LLMPayload): Promise<void> {
  await checkedFetch(`${BASE}/api/llm`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
}

export async function testConnection(payload: LLMPayload): Promise<{ ok: boolean; error?: string }> {
  const resp = await checkedFetch(`${BASE}/api/test-connection`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
  return resp.json() as Promise<{ ok: boolean; error?: string }>;
}

export async function listModels(payload: {
  provider: string;
  base_url: string;
  api_key: string;
}): Promise<{ models?: string[]; error?: string }> {
  const resp = await checkedFetch(`${BASE}/api/models`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
  return resp.json() as Promise<{ models?: string[]; error?: string }>;
}
