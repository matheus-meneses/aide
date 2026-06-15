import { useCallback, useEffect, useRef, useState } from "react";

export interface InstallProgress {
  lines: string[];
  done: string;
  error: string;
}

const empty: InstallProgress = { lines: [], done: "", error: "" };

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

// useInstallProgress subscribes to the SSE plugin-install lifecycle events and
// exposes the running log, completion (the installed plugin name), and errors.
// Shared by the setup wizard and the marketplace.
export function useInstallProgress() {
  const [progress, setProgress] = useState<InstallProgress>(empty);
  const esRef = useRef<EventSource | null>(null);

  useEffect(() => {
    const es = new EventSource("/api/events");
    esRef.current = es;

    const on = (type: string, fn: (msg: string) => void) =>
      es.addEventListener(type, (e: MessageEvent<string>) => fn(parseMessage(e.data)));

    on("install_progress", (m) => setProgress((p) => ({ ...p, lines: [...p.lines, m] })));
    on("install_done", (m) => setProgress((p) => ({ ...p, done: m })));
    on("install_error", (m) => setProgress((p) => ({ ...p, error: m })));

    return () => es.close();
  }, []);

  const reset = useCallback(() => setProgress(empty), []);

  return { progress, reset };
}
