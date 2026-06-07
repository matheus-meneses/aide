import {useEffect, useState} from 'react'
import {ArrowUpCircle, Moon, PanelLeft, Sun, Wifi, WifiOff} from 'lucide-react'
import {fetchStatus, fetchVersion} from '@/lib/api'

interface StatusData {
    counts: Record<string, number>
    today_events: number
    metrics: Array<{ name: string; value: number; source: string }>
}

interface Props {
    connected: boolean
    onToggleSidebar: () => void
    activeSource?: string | null
    onSourceClick?: (source: string | null) => void
}

export function StatusBar({connected, onToggleSidebar, activeSource, onSourceClick}: Props) {
    const [status, setStatus] = useState<StatusData | null>(null)
    const [statusError, setStatusError] = useState(false)
    const [userName, setUserName] = useState('')
    const [updateAvailable, setUpdateAvailable] = useState<{
        current: string;
        latest: string;
        url: string
    } | null>(null)
    const [dark, setDark] = useState(() =>
        document.documentElement.classList.contains('dark')
    )

    useEffect(() => {
        fetchStatus().then(d => { setStatus(d); setStatusError(false) }).catch(() => setStatusError(true))
        fetch('/api/whoami').then(r => r.json()).then(p => {
            if (p.preferred_name) setUserName(p.preferred_name)
        }).catch(() => {
        })
        fetchVersion().then(v => {
            fetch('https://nexus.sharedservices.local/repository/aide/VERSION')
                .then(r => r.text())
                .then(latest => {
                    const trimmed = latest.trim()
                    if (trimmed && trimmed !== v.current && v.current !== 'dev') {
                        setUpdateAvailable({current: v.current, latest: trimmed, url: v.update_url})
                    }
                })
                .catch(() => {
                })
        }).catch(() => {
        })
        const interval = setInterval(() => {
            fetchStatus().then(d => { setStatus(d); setStatusError(false) }).catch(() => setStatusError(true))
        }, 60000)
        return () => clearInterval(interval)
    }, [])

    const toggleDark = () => {
        const next = !dark
        document.documentElement.classList.toggle('dark', next)
        localStorage.setItem('theme', next ? 'dark' : 'light')
        setDark(next)
    }

    const unread = status?.metrics?.find(m => m.name === 'Inbox Unread')?.value
    const total = status?.counts ? Object.values(status.counts).reduce((a, b) => a + b, 0) : 0

    return (
        <header className="flex flex-col border-b bg-card">
            {updateAvailable && (
                <div
                    className="flex items-center gap-2 px-4 py-1.5 bg-amber-500/10 border-b border-amber-500/20 text-xs text-amber-700 dark:text-amber-400">
                    <ArrowUpCircle className="w-3.5 h-3.5 shrink-0"/>
                    <span>
            New version available: <strong>{updateAvailable.latest}</strong> (current: {updateAvailable.current})
          </span>
                    <code className="ml-auto text-[10px] bg-amber-500/10 rounded px-1.5 py-0.5 font-mono">
                        curl -fsSL {updateAvailable.url} | bash
                    </code>
                </div>
            )}
            <div className="flex items-center justify-between px-4 py-2">
                <div className="flex items-center gap-3 text-sm">
                    <button
                        onClick={onToggleSidebar}
                        className="p-1 rounded hover:bg-accent transition-colors md:hidden"
                        aria-label="Toggle sidebar"
                    >
                        <PanelLeft className="w-4 h-4"/>
                    </button>
                    <span className="font-semibold text-base">Aide</span>
                    {status ? (
                        <div className="hidden sm:flex items-center gap-3 text-muted-foreground text-xs">
                            {status.today_events > 0 && (
                                <button
                                    onClick={() => onSourceClick?.(activeSource === '__meetings' ? null : '__meetings')}
                                    className={`hover:text-foreground transition-colors rounded px-1.5 py-0.5 ${activeSource === '__meetings' ? 'bg-primary/10 text-primary font-medium' : ''}`}
                                    aria-pressed={activeSource === '__meetings'}
                                >
                                    {status.today_events} meetings
                                </button>
                            )}
                            {Object.entries(status.counts || {}).map(([source, count]) => (
                                <button
                                    key={source}
                                    onClick={() => onSourceClick?.(activeSource === source ? null : source)}
                                    className={`hover:text-foreground transition-colors rounded px-1.5 py-0.5 ${activeSource === source ? 'bg-primary/10 text-primary font-medium' : ''}`}
                                    aria-pressed={activeSource === source}
                                >
                                    {count} {source}
                                </button>
                            ))}
                            {unread != null && unread > 0 && <span>{unread} unread</span>}
                            <button
                                onClick={() => onSourceClick?.(activeSource ? null : '__all')}
                                className={`opacity-60 hover:opacity-100 transition-opacity rounded px-1.5 py-0.5 ${activeSource === '__all' ? 'bg-primary/10 text-primary font-medium opacity-100' : ''}`}
                            >
                                {total} total
                            </button>
                        </div>
                    ) : statusError ? (
                        <span className="hidden sm:inline text-xs text-red-500">Could not load status</span>
                    ) : (
                        <div className="hidden sm:flex items-center gap-3">
                            <div className="h-3 w-16 rounded bg-muted animate-pulse"/>
                            <div className="h-3 w-12 rounded bg-muted animate-pulse"/>
                        </div>
                    )}
                </div>
                <div className="flex items-center gap-3">
                    {userName && (
                        <span className="text-xs text-muted-foreground hidden sm:inline">{userName}</span>
                    )}
                    <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                        {connected ? (
                            <Wifi className="w-3.5 h-3.5 text-green-500"/>
                        ) : (
                            <WifiOff className="w-3.5 h-3.5 text-red-500"/>
                        )}
                        <span className="hidden sm:inline">{connected ? 'live' : 'disconnected'}</span>
                    </div>
                    <button
                        onClick={toggleDark}
                        className="p-1.5 rounded-md hover:bg-accent transition-colors"
                        aria-label={dark ? 'Switch to light mode' : 'Switch to dark mode'}
                    >
                        {dark ? <Sun className="w-4 h-4"/> : <Moon className="w-4 h-4"/>}
                    </button>
                </div>
            </div>
        </header>
    )
}
