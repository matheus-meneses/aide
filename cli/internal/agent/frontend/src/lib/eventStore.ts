export interface AgentEvent {
    id: number
    type: string
    data: string
    timestamp: string
    priority?: string
}

export const STORAGE_KEY = 'aide-notifications'
export const MAX_STORED = 100
export const MAX_EVENTS = 50

export function loadFromStorage(): AgentEvent[] {
    try {
        const raw = localStorage.getItem(STORAGE_KEY)
        if (raw) return JSON.parse(raw).slice(0, MAX_STORED)
    } catch {}
    return []
}

export function saveToStorage(events: AgentEvent[]) {
    try {
        localStorage.setItem(STORAGE_KEY, JSON.stringify(events.slice(0, MAX_STORED)))
    } catch {}
}

export function safeParse(raw: string): AgentEvent | null {
    try {
        return JSON.parse(raw)
    } catch {
        return null
    }
}

export function eventKey(event: AgentEvent): string {
    if (event.id) return String(event.id)
    try {
        const parsed = JSON.parse(event.data)
        if (parsed.fingerprint) return parsed.fingerprint
    } catch {}
    return `${event.type}-${event.timestamp}-${event.data.slice(0, 80)}`
}
