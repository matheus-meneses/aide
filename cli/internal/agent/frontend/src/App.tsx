import {useCallback, useEffect, useState} from 'react'
import {PanelLeft, PanelLeftClose} from 'lucide-react'
import {type AgentEvent, useSSE} from '@/hooks/useSSE'
import {StatusBar} from '@/components/StatusBar'
import {NotificationFeed} from '@/components/NotificationFeed'
import {ChatPanel} from '@/components/ChatPanel'
import {ItemsView} from '@/components/ItemsView'

function App() {
    const {events, connected, dismiss, onChatMessage, notificationPermission, enableNotifications} = useSSE('/api/events')
    const [sidebarOpen, setSidebarOpen] = useState(false)
    const [isMobile, setIsMobile] = useState(false)
    const [activeSource, setActiveSource] = useState<string | null>(null)
    const [pendingEvent, setPendingEvent] = useState<AgentEvent | null>(null)

    const handleEventClick = useCallback((event: AgentEvent) => {
        setActiveSource(null)
        setPendingEvent(event)
    }, [])

    useEffect(() => {
        const mq = window.matchMedia('(min-width: 768px)')
        const handler = (e: MediaQueryListEvent | MediaQueryList) => {
            setIsMobile(!e.matches)
            if (e.matches) setSidebarOpen(true)
        }
        handler(mq)
        mq.addEventListener('change', handler)
        return () => mq.removeEventListener('change', handler)
    }, [])

    const closeSidebar = useCallback(() => {
        if (isMobile) setSidebarOpen(false)
    }, [isMobile])

    return (
        <div className="h-screen flex flex-col">
            <StatusBar connected={connected} onToggleSidebar={() => setSidebarOpen(v => !v)} activeSource={activeSource}
                       onSourceClick={setActiveSource}/>
            <div className="flex flex-1 overflow-hidden relative">
                {isMobile && sidebarOpen && (
                    <div
                        className="fixed inset-0 z-30 bg-black/40 backdrop-blur-sm md:hidden animate-fade-in"
                        onClick={closeSidebar}
                    />
                )}

                <aside
                    className={`
            ${isMobile ? 'fixed inset-y-0 left-0 z-40 w-72 pt-[49px]' : 'relative w-72 shrink-0'}
            border-r flex flex-col bg-card transition-transform duration-200 ease-out
            ${sidebarOpen ? 'translate-x-0' : '-translate-x-full'}
            ${!isMobile && !sidebarOpen ? 'hidden' : ''}
          `}
                >
                    <div className="flex items-center justify-between px-3 py-2 border-b">
            <span className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
              Activity
            </span>
                        <button
                            onClick={() => setSidebarOpen(false)}
                            className="p-1 rounded hover:bg-accent transition-colors"
                            aria-label="Close sidebar"
                        >
                            <PanelLeftClose className="w-4 h-4 text-muted-foreground"/>
                        </button>
                    </div>
                    <div className="flex-1 overflow-hidden">
                        <NotificationFeed events={events} onEventClick={handleEventClick} onDismiss={dismiss}
                                          notificationPermission={notificationPermission}
                                          onEnableNotifications={enableNotifications}/>
                    </div>
                </aside>

                <main className="flex-1 flex flex-col overflow-hidden">
                    {!sidebarOpen && !isMobile && (
                        <button
                            onClick={() => setSidebarOpen(true)}
                            className="absolute top-2 left-2 z-10 p-1.5 rounded-md bg-card border hover:bg-accent transition-colors"
                            aria-label="Open sidebar"
                        >
                            <PanelLeft className="w-4 h-4"/>
                        </button>
                    )}
                    {activeSource ? (
                        <ItemsView source={activeSource} onClose={() => setActiveSource(null)}/>
                    ) : (
                        <ChatPanel pendingEvent={pendingEvent} onEventConsumed={() => setPendingEvent(null)} onChatMessage={onChatMessage}/>
                    )}
                </main>
            </div>
        </div>
    )
}

export default App
