import { useCallback, useEffect, useState } from "react";
import { subscribeEvent } from "@/lib/eventBus";

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

  useEffect(() => {
    const unsubs = [
      subscribeEvent("install_progress", (d) =>
        setProgress((p) => ({ ...p, lines: [...p.lines, parseMessage(d)] })),
      ),
      subscribeEvent("install_done", (d) => setProgress((p) => ({ ...p, done: parseMessage(d) }))),
      subscribeEvent("install_error", (d) => setProgress((p) => ({ ...p, error: parseMessage(d) }))),
    ];
    return () => unsubs.forEach((u) => u());
  }, []);

  const reset = useCallback(() => setProgress(empty), []);

  return { progress, reset };
}
