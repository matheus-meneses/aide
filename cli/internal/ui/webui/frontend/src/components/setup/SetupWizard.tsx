import { useCallback, useEffect, useRef, useState } from "react";
import { Sparkles } from "lucide-react";
import { Card } from "@/components/ui";
import { APP_NAME } from "@/lib/brand";
import { type Progress, type Step, emptyProgress, parseMessage } from "./types";
import { BootstrapStep } from "./steps/BootstrapStep";
import { SourceStep } from "./steps/SourceStep";
import { ProviderStep } from "./steps/ProviderStep";

export default function SetupWizard({ onComplete }: { onComplete: () => void }) {
  const [step, setStep] = useState<Step>("bootstrap");
  const [progress, setProgress] = useState<Progress>(emptyProgress);
  const esRef = useRef<EventSource | null>(null);

  useEffect(() => {
    const es = new EventSource("/api/events");
    esRef.current = es;

    const on = (type: string, fn: (msg: string) => void) =>
      es.addEventListener(type, (e: MessageEvent<string>) => fn(parseMessage(e.data)));

    on("setup_progress", (m) => setProgress((p) => ({ ...p, setupLines: [...p.setupLines, m] })));
    on("setup_done", () => setProgress((p) => ({ ...p, setupDone: true })));
    on("setup_error", (m) => setProgress((p) => ({ ...p, setupError: m })));
    on("install_progress", (m) => setProgress((p) => ({ ...p, installLines: [...p.installLines, m] })));
    on("install_done", (m) => setProgress((p) => ({ ...p, installDone: m })));
    on("install_error", (m) => setProgress((p) => ({ ...p, installError: m })));

    return () => es.close();
  }, []);

  const resetInstall = useCallback(
    () => setProgress((p) => ({ ...p, installLines: [], installDone: "", installError: "" })),
    [],
  );

  return (
    <div className="h-full flex flex-col items-center justify-center bg-background p-6">
      <Card className="w-full max-w-xl">
        <div className="flex items-center gap-2 border-b px-5 py-4">
          <Sparkles className="w-5 h-5 text-primary" />
          <h1 className="text-base font-semibold">Welcome to {APP_NAME}</h1>
          <div className="ml-auto flex items-center gap-1.5">
            {(["bootstrap", "source", "provider"] as Step[]).map((s) => (
              <span
                key={s}
                className={`h-1.5 w-6 rounded-full ${step === s ? "bg-primary" : "bg-muted"}`}
              />
            ))}
          </div>
        </div>

        <div className="p-5">
          {step === "bootstrap" && (
            <BootstrapStep progress={progress} onNext={() => setStep("source")} />
          )}
          {step === "source" && (
            <SourceStep
              progress={progress}
              resetInstall={resetInstall}
              onBack={() => setStep("bootstrap")}
              onNext={() => setStep("provider")}
            />
          )}
          {step === "provider" && (
            <ProviderStep
              onBack={() => setStep("source")}
              onNext={() => {
                setStep("done");
                onComplete();
              }}
            />
          )}
        </div>
      </Card>
    </div>
  );
}
