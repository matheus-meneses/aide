const BASE = ''

async function checkedFetch(url: string, init?: RequestInit): Promise<Response> {
    const resp = await fetch(url, init)
    if (!resp.ok) {
        const text = await resp.text().catch(() => '')
        throw new Error(text || `HTTP ${resp.status}`)
    }
    return resp
}

export async function fetchStatus() {
    const resp = await checkedFetch(`${BASE}/api/status`)
    return resp.json()
}

export async function fetchItems(source?: string) {
    const params = source ? `?source=${source}` : ''
    const resp = await checkedFetch(`${BASE}/api/items${params}`)
    return resp.json()
}

export async function fetchToday() {
    const resp = await checkedFetch(`${BASE}/api/today`)
    return resp.json()
}

export async function fetchSessions() {
    const resp = await checkedFetch(`${BASE}/api/sessions`)
    return resp.json()
}

export async function fetchVersion(): Promise<{ current: string; update_url: string }> {
    const resp = await checkedFetch(`${BASE}/api/version`)
    return resp.json()
}

export async function fetchNotifications(limit = 50): Promise<{ events: any[]; total: number }> {
    const resp = await checkedFetch(`${BASE}/api/notifications?limit=${limit}`)
    return resp.json()
}
