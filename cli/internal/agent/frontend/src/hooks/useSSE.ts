import {useCallback, useEffect, useRef, useState} from 'react'
import {fetchNotifications} from '@/lib/api'
import {
    type AgentEvent,
    MAX_EVENTS,
    eventKey,
    loadFromStorage,
    safeParse,
    saveToStorage,
} from '@/lib/eventStore'
import {shouldNotify} from '@/lib/notifications'
import {useNotifications} from '@/hooks/useNotifications'

export type {AgentEvent} from '@/lib/eventStore'
export {describeEvent} from '@/lib/notifications'

export function useSSE(url: string) {
    const [events, setEvents] = useState<AgentEvent[]>(loadFromStorage)
    const [connected, setConnected] = useState(false)
    const sourceRef = useRef<EventSource | null>(null)
    const chatCallbackRef = useRef<((event: AgentEvent) => void) | null>(null)
    const lastEventIdRef = useRef<number>(0)
    const {notificationPermission, enableNotifications, queueBrowserNotification, cleanupGrouping} =
        useNotifications()

    const onChatMessage = useCallback((cb: (event: AgentEvent) => void) => {
        chatCallbackRef.current = cb
    }, [])

    useEffect(() => {
        fetchNotifications(MAX_EVENTS).then(data => {
            if (data.events?.length) {
                setEvents(prev => {
                    const existingKeys = new Set(prev.map(eventKey))
                    const newOnes = data.events.filter((e: AgentEvent) => !existingKeys.has(eventKey(e)))
                    if (newOnes.length === 0) return prev
                    const merged = [...newOnes, ...prev].slice(0, MAX_EVENTS)
                    saveToStorage(merged)
                    const maxId = Math.max(...merged.map(e => e.id || 0))
                    if (maxId > lastEventIdRef.current) lastEventIdRef.current = maxId
                    return merged
                })
            }
        }).catch(err => {
            console.warn('failed to load notifications:', err)
        })
    }, [])

    useEffect(() => {
        const stored = loadFromStorage()
        if (stored.length > 0) {
            const maxId = Math.max(...stored.map(e => e.id || 0))
            if (maxId > lastEventIdRef.current) lastEventIdRef.current = maxId
        }

        const es = new EventSource(url)
        sourceRef.current = es

        es.onopen = () => setConnected(true)
        es.onerror = () => setConnected(false)

        const appendEvent = (event: AgentEvent) => {
            const key = eventKey(event)
            setEvents(prev => {
                if (prev.some(existing => eventKey(existing) === key)) return prev
                const updated = [event, ...prev].slice(0, MAX_EVENTS)
                saveToStorage(updated)
                if (event.id && event.id > lastEventIdRef.current) {
                    lastEventIdRef.current = event.id
                }
                return updated
            })
        }

        const handleNotifiable = (e: MessageEvent) => {
            const event = safeParse(e.data)
            if (!event) return
            appendEvent(event)
            if (shouldNotify(event)) {
                queueBrowserNotification(event)
            }
        }

        const handleSilent = (e: MessageEvent) => {
            const event = safeParse(e.data)
            if (!event) return
            appendEvent(event)
        }

        const handleChatMessage = (e: MessageEvent) => {
            const event = safeParse(e.data)
            if (event) {
                chatCallbackRef.current?.(event)
            }
        }

        es.addEventListener('notification', handleNotifiable)
        es.addEventListener('briefing', handleNotifiable)
        es.addEventListener('scrape_complete', handleSilent)
        es.addEventListener('cycle_error', handleSilent)
        es.addEventListener('status', handleSilent)
        es.addEventListener('chat_message', handleChatMessage)

        return () => {
            es.close()
            sourceRef.current = null
            cleanupGrouping()
        }
    }, [url, queueBrowserNotification, cleanupGrouping])

    const clear = useCallback(() => {
        setEvents([])
        saveToStorage([])
    }, [])

    const dismiss = useCallback((event: AgentEvent) => {
        const key = eventKey(event)
        setEvents(prev => {
            const updated = prev.filter(e => eventKey(e) !== key)
            saveToStorage(updated)
            return updated
        })
    }, [])

    return {events, connected, clear, dismiss, onChatMessage, notificationPermission, enableNotifications}
}
