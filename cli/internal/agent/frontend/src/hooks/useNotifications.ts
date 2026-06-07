import {useCallback, useEffect, useRef, useState} from 'react'
import {type AgentEvent} from '@/lib/eventStore'
import {
    GROUPING_WINDOW_MS,
    currentPermission,
    flushGroupedBuffer,
    requestNotificationPermission,
    showBrowserNotification,
} from '@/lib/notifications'

export function useNotifications() {
    const groupBufferRef = useRef<AgentEvent[]>([])
    const groupTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
    const [notificationPermission, setNotificationPermission] =
        useState<NotificationPermission | 'unsupported'>(currentPermission)

    const enableNotifications = useCallback(async () => {
        const result = await requestNotificationPermission()
        setNotificationPermission(result)
        return result
    }, [])

    const queueBrowserNotification = useCallback((event: AgentEvent) => {
        const priority = event.priority || 'normal'
        if (priority === 'urgent') {
            showBrowserNotification(event)
            return
        }

        groupBufferRef.current.push(event)
        if (!groupTimerRef.current) {
            groupTimerRef.current = setTimeout(() => {
                flushGroupedBuffer(groupBufferRef.current)
                groupBufferRef.current = []
                groupTimerRef.current = null
            }, GROUPING_WINDOW_MS)
        }
    }, [])

    useEffect(() => {
        if (currentPermission() !== 'default') return

        const onGesture = () => {
            if (currentPermission() === 'default') {
                enableNotifications()
            }
            window.removeEventListener('pointerdown', onGesture)
            window.removeEventListener('keydown', onGesture)
        }

        window.addEventListener('pointerdown', onGesture, {once: true})
        window.addEventListener('keydown', onGesture, {once: true})
        return () => {
            window.removeEventListener('pointerdown', onGesture)
            window.removeEventListener('keydown', onGesture)
        }
    }, [enableNotifications])

    const cleanupGrouping = useCallback(() => {
        if (groupTimerRef.current) {
            clearTimeout(groupTimerRef.current)
            groupTimerRef.current = null
        }
        groupBufferRef.current = []
    }, [])

    return {notificationPermission, enableNotifications, queueBrowserNotification, cleanupGrouping}
}
