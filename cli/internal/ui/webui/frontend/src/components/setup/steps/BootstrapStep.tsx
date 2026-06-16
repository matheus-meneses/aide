import { useEffect, useState } from "react";
import { AlertCircle, ArrowRight, CheckCircle2, ChevronRight, Loader2 } from "lucide-react";
import * as api from "@/lib/api";
import { Button } from "@/components/ui";
import { APP_NAME } from "@/lib/brand";
import type { Progress } from "../types";
import { LogPanel } from "../shared";

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

  const running = started && !progress.setupError && !progress.setupDone;

  return (
    <div>
      <p className="text-sm text-muted-foreground">
        Let’s get {APP_NAME} ready for you. This quick, one-time setup runs entirely on your computer
        and won’t change anything else on your system.
      </p>

      {running && (
        <div className="mt-4 flex items-center gap-2 text-sm text-foreground">
          <Loader2 className="w-4 h-4 animate-spin text-primary" />
          Getting things ready… this can take a minute.
        </div>
      )}

      {progress.setupDone && !progress.setupError && (
        <div className="mt-4 flex items-center gap-2 text-sm text-foreground">
          <CheckCircle2 className="w-4 h-4 text-primary" />
          All set — {APP_NAME} is ready to go.
        </div>
      )}

      {progress.setupError && (
        <div className="mt-3 flex items-start gap-2 rounded-md border border-destructive/40 bg-destructive/10 p-3 text-sm text-destructive">
          <AlertCircle className="mt-0.5 w-4 h-4 shrink-0" />
          <span>Something went wrong during setup. {progress.setupError}</span>
        </div>
      )}

      {progress.setupLines.length > 0 && (
        <details className="group mt-3">
          <summary className="flex cursor-pointer select-none items-center gap-1 text-xs text-muted-foreground transition-colors hover:text-foreground">
            <ChevronRight className="w-3 h-3 transition-transform group-open:rotate-90" />
            Show technical details
          </summary>
          <LogPanel lines={progress.setupLines} />
        </details>
      )}

      <div className="mt-5 flex justify-end">
        {progress.setupDone ? (
          <Button onClick={onNext}>
            Continue <ArrowRight className="w-4 h-4" />
          </Button>
        ) : (
          <Button onClick={() => void start()} loading={started && !progress.setupError}>
            {started ? "Setting up…" : "Get started"}
          </Button>
        )}
      </div>
    </div>
  );
}
