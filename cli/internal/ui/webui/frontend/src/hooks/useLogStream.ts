import { useCallback, useEffect, useRef, useState } from "react";

export interface LogEntry {
  ts: string;
  level: string;
  scope: string;
  msg: string;
}

const MAX_LOGS = 2000;

export function useLogStream(url = "/api/logs") {
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const [connected, setConnected] = useState(false);
  const [paused, setPaused] = useState(false);
  const sourceRef = useRef<EventSource | null>(null);

  useEffect(() => {
    if (paused) {
      setConnected(false);
      return;
    }

    const es = new EventSource(url);
    sourceRef.current = es;

    es.onopen = () => setConnected(true);
    es.onerror = () => setConnected(false);
    es.onmessage = (e: MessageEvent<string>) => {
      let entry: LogEntry;
      try {
        entry = JSON.parse(e.data) as LogEntry;
      } catch {
        return;
      }
      setLogs((prev) => {
        const next = prev.length >= MAX_LOGS ? prev.slice(prev.length - MAX_LOGS + 1) : prev;
        return [...next, entry];
      });
    };

    return () => {
      es.close();
      sourceRef.current = null;
    };
  }, [url, paused]);

  const clear = useCallback(() => setLogs([]), []);
  const togglePaused = useCallback(() => setPaused((v) => !v), []);

  return { logs, connected, paused, clear, togglePaused };
}
