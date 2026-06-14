export const BASE = "";

export async function checkedFetch(url: string, init?: RequestInit): Promise<Response> {
  const resp = await fetch(url, init);
  if (!resp.ok) {
    const text = await resp.text().catch(() => "");
    throw new Error(text || `HTTP ${resp.status}`);
  }
  return resp;
}

export async function postJSON(url: string, body: unknown): Promise<void> {
  await checkedFetch(`${BASE}${url}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
}
