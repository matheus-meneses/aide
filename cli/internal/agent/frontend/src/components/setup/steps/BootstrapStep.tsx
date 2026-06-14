import { useEffect, useState } from "react";
import { AlertCircle, ArrowRight, Loader2 } from "lucide-react";
import * as api from "@/lib/api";
import type { Progress } from "../types";
import { LogPanel, PrimaryButton } from "../shared";

export function BootstrapStep({ progress, onNext }: { progress: Progress; onNext: () => void }) {
  const [started, setStarted] = useState(false);
  const [skipChecked, setSkipChecked] = useState(false);

  useEffect(() => {
    api
      .fetchSetupStatus()
      .then((s) => {
        if (s.python_ready) onNext();
      })
      .catch(() => {})
      .finally(() => setSkipChecked(true));
  }, [onNext]);

  const start = async () => {
    setStarted(true);
    await api.startBootstrap();
  };

  if (!skipChecked) {
    return (
      <div className="flex items-center gap-2 text-sm text-muted-foreground">
        <Loader2 className="w-4 h-4 animate-spin" /> Checking your setup…
      </div>
    );
  }

  return (
    <div>
      <p className="text-sm text-muted-foreground">
        Aide needs a one-time setup: it downloads a private Python runtime and the plugin catalog.
        Nothing is installed system-wide.
      </p>

      {progress.setupError && (
        <div className="mt-3 flex items-start gap-2 rounded-md border border-destructive/40 bg-destructive/10 p-3 text-sm text-destructive">
          <AlertCircle className="mt-0.5 w-4 h-4 shrink-0" />
          <span>{progress.setupError}</span>
        </div>
      )}

      <LogPanel lines={progress.setupLines} />

      <div className="mt-5 flex justify-end">
        {progress.setupDone ? (
          <PrimaryButton onClick={onNext}>
            Continue <ArrowRight className="w-4 h-4" />
          </PrimaryButton>
        ) : (
          <PrimaryButton onClick={() => void start()} busy={started && !progress.setupError}>
            {started ? "Setting up…" : "Get started"}
          </PrimaryButton>
        )}
      </div>
    </div>
  );
}
