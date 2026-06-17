import { useCallback, useEffect, useState } from "react";
import { subscribeEvent } from "@/lib/eventBus";

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

  useEffect(() => {
    const unsubs = [
      subscribeEvent("update_progress", (d) =>
        setProgress((p) => ({ ...p, running: true, lines: [...p.lines, parseMessage(d)] })),
      ),
      subscribeEvent("update_done", (d) =>
        setProgress((p) => ({ ...p, running: false, done: parseMessage(d) })),
      ),
      subscribeEvent("update_error", (d) =>
        setProgress((p) => ({ ...p, running: false, error: parseMessage(d) })),
      ),
    ];
    return () => unsubs.forEach((u) => u());
  }, []);

  const start = useCallback(() => setProgress({ ...empty, running: true }), []);
  const reset = useCallback(() => setProgress(empty), []);

  return { progress, start, reset };
}
