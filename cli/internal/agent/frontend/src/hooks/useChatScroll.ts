import {useCallback, useEffect, useRef} from 'react'

export function useChatScroll(deps: unknown) {
    const scrollRef = useRef<HTMLDivElement>(null)
    const userAtBottomRef = useRef(true)

    const isNearBottom = useCallback(() => {
        const el = scrollRef.current
        if (!el) return true
        return el.scrollHeight - el.scrollTop - el.clientHeight < 80
    }, [])

    const handleScroll = useCallback(() => {
        userAtBottomRef.current = isNearBottom()
    }, [isNearBottom])

    const markAtBottom = useCallback(() => {
        userAtBottomRef.current = true
    }, [])

    const scrollToBottom = useCallback(() => {
        userAtBottomRef.current = true
        if (scrollRef.current) {
            scrollRef.current.scrollTop = scrollRef.current.scrollHeight
        }
    }, [])

    useEffect(() => {
        if (userAtBottomRef.current && scrollRef.current) {
            scrollRef.current.scrollTop = scrollRef.current.scrollHeight
        }
    }, [deps])

    return {scrollRef, handleScroll, markAtBottom, scrollToBottom}
}
