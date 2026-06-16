import { useCallback, useEffect, useRef, useState } from "react";

export interface UpdateProgress {
  running: boolean;
  lines: string[];
  done: string;
  error: string;
}

const empty: UpdateProgress = { running: false, lines: [], done: "", error: "" };

function parseMessage(raw: string): string {
  let inner = raw;
  try {
    const envelope = JSON.parse(raw) as { data?: string };
    if (typeof envelope.data === "string") inner = envelope.data;
  } catch {
    return raw;
  }
  try {
    return (JSON.parse(inner) as { message?: string }).message ?? inner;
  } catch {
    return inner;
  }
}

// useUpdateProgress subscribes to the SSE self-update lifecycle events and
// exposes the running log, completion (the installed version), and errors.
export function useUpdateProgress() {
  const [progress, setProgress] = useState<UpdateProgress>(empty);
  const esRef = useRef<EventSource | null>(null);

  useEffect(() => {
    const es = new EventSource("/api/events");
    esRef.current = es;

    const on = (type: string, fn: (msg: string) => void) =>
      es.addEventListener(type, (e: MessageEvent<string>) => fn(parseMessage(e.data)));

    on("update_progress", (m) =>
      setProgress((p) => ({ ...p, running: true, lines: [...p.lines, m] })),
    );
    on("update_done", (m) => setProgress((p) => ({ ...p, running: false, done: m })));
    on("update_error", (m) => setProgress((p) => ({ ...p, running: false, error: m })));

    return () => es.close();
  }, []);

  const start = useCallback(() => setProgress({ ...empty, running: true }), []);
  const reset = useCallback(() => setProgress(empty), []);

  return { progress, start, reset };
}
