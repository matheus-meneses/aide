import {useCallback, useEffect, useRef, useState} from 'react'

export interface Message {
    id: string
    role: 'user' | 'assistant'
    content: string
    timestamp: number
    isError?: boolean
    format?: string
    data?: any
}

let msgCounter = 0

function nextId(): string {
    return `msg-${Date.now()}-${++msgCounter}`
}

export function useChatStream(url: string) {
    const [messages, setMessages] = useState<Message[]>([])
    const [isStreaming, setIsStreaming] = useState(false)
    const abortRef = useRef<AbortController | null>(null)

    useEffect(() => {
        fetch('/api/sessions/web-default')
            .then(r => r.ok ? r.json() : null)
            .then(data => {
                if (data?.messages?.length) {
                    const hydrated: Message[] = data.messages.map((m: any) => ({
                        id: nextId(),
                        role: m.role,
                        content: m.content,
                        timestamp: m.timestamp ? new Date(m.timestamp).getTime() : Date.now(),
                    }))
                    setMessages(hydrated)
                }
            })
            .catch(err => {
                console.warn('failed to hydrate chat history:', err)
            })
    }, [])

    const appendAssistantFromSSE = useCallback((data: string) => {
        try {
            const inner = JSON.parse(data)
            if (inner.role === 'assistant' && inner.content) {
                setMessages(prev => {
                    const recent = prev.slice(-6)
                    if (recent.some(m => m.role === 'assistant' && m.content === inner.content)) {
                        return prev
                    }
                    const msg: Message = {
                        id: nextId(),
                        role: 'assistant',
                        content: inner.content,
                        timestamp: inner.timestamp ? new Date(inner.timestamp).getTime() : Date.now(),
                    }
                    return [...prev, msg]
                })
            }
        } catch (err) {
            console.warn('failed to parse SSE chat message:', err)
        }
    }, [])

    const send = useCallback(async (message: string) => {
        const userMsg: Message = {id: nextId(), role: 'user', content: message, timestamp: Date.now()}
        const assistantId = nextId()
        setMessages(prev => [...prev, userMsg, {
            id: assistantId,
            role: 'assistant',
            content: '',
            timestamp: Date.now()
        }])
        setIsStreaming(true)

        const controller = new AbortController()
        abortRef.current = controller

        try {
            const resp = await fetch(url, {
                method: 'POST',
                headers: {'Content-Type': 'application/json'},
                body: JSON.stringify({message, session_id: 'web-default'}),
                signal: controller.signal,
            })

            if (!resp.ok) {
                const text = await resp.text().catch(() => '')
                throw new Error(text || `HTTP ${resp.status}`)
            }

            if (!resp.body) {
                throw new Error('No response body')
            }

            const reader = resp.body.getReader()
            const decoder = new TextDecoder()
            let buffer = ''
            let pendingEventType = ''

            while (true) {
                const {done, value} = await reader.read()
                if (done) break

                buffer += decoder.decode(value, {stream: true})
                const lines = buffer.split('\n')
                buffer = lines.pop() || ''

                for (const line of lines) {
                    if (line.startsWith('event:')) {
                        pendingEventType = line.slice(6).trim()
                        continue
                    }

                    if (line.startsWith('data: ')) {
                        const data = line.slice(6)

                        if (pendingEventType === 'error') {
                            pendingEventType = ''
                            try {
                                const errData = JSON.parse(data)
                                throw new Error(errData.error || 'LLM error')
                            } catch (e: any) {
                                if (e instanceof SyntaxError) {
                                    throw new Error('LLM request failed')
                                }
                                throw e
                            }
                        }

                        pendingEventType = ''
                        try {
                            const parsed = JSON.parse(data)
                            if (parsed.content) {
                                setMessages(prev => {
                                    const updated = [...prev]
                                    const last = updated[updated.length - 1]
                                    if (last && last.role === 'assistant') {
                                        updated[updated.length - 1] = {...last, content: last.content + parsed.content}
                                    }
                                    return updated
                                })
                            }
                            if (parsed.error) {
                                throw new Error(parsed.error)
                            }
                        } catch (e: any) {
                            if (e instanceof SyntaxError) continue
                            throw e
                        }
                    }
                }
            }
        } catch (err: any) {
            if (err.name === 'AbortError') {
                setMessages(prev => {
                    const updated = [...prev]
                    const last = updated[updated.length - 1]
                    if (last && last.role === 'assistant') {
                        if (!last.content.trim()) {
                            return updated.slice(0, -1)
                        }
                        updated[updated.length - 1] = {...last, content: last.content + ' _(cancelled)_'}
                    }
                    return updated
                })
            } else {
                const errorMsg = err.message || 'Failed to get response'
                setMessages(prev => {
                    const updated = [...prev]
                    const last = updated[updated.length - 1]
                    if (last && last.role === 'assistant') {
                        updated[updated.length - 1] = {...last, content: errorMsg, isError: true}
                    }
                    return updated
                })
            }
        } finally {
            setIsStreaming(false)
            abortRef.current = null
        }
    }, [url])

    const cancel = useCallback(() => {
        abortRef.current?.abort()
    }, [])

    const clearMessages = useCallback(() => setMessages([]), [])

    const injectMessage = useCallback((msg: Message) => {
        setMessages(prev => [...prev, msg])
    }, [])

    const retry = useCallback(() => {
        const last = [...messages].reverse().find(m => m.role === 'user')
        if (last) {
            setMessages(prev => prev.filter(m => !m.isError))
            send(last.content)
        }
    }, [messages, send])

    return {messages, send, isStreaming, cancel, clearMessages, injectMessage, appendAssistantFromSSE, retry}
}
