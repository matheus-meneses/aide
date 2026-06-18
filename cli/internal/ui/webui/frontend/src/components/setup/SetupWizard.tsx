import { useCallback, useEffect, useState } from "react";
import { Sparkles } from "lucide-react";
import { Card } from "@/components/ui";
import { APP_NAME } from "@/lib/brand";
import { subscribeEvent } from "@/lib/eventBus";
import { type Progress, type Step, emptyProgress, parseMessage } from "./types";
import { BootstrapStep } from "./steps/BootstrapStep";
import { SourceStep } from "./steps/SourceStep";
import { ProviderStep } from "./steps/ProviderStep";

export default function SetupWizard({ onComplete }: { onComplete: () => void }) {
  const [step, setStep] = useState<Step>("bootstrap");
  const [progress, setProgress] = useState<Progress>(emptyProgress);

  useEffect(() => {
    const unsubs = [
      subscribeEvent("setup_progress", (d) =>
        setProgress((p) => ({ ...p, setupLines: [...p.setupLines, parseMessage(d)] })),
      ),
      subscribeEvent("setup_done", () => setProgress((p) => ({ ...p, setupDone: true }))),
      subscribeEvent("setup_error", (d) =>
        setProgress((p) => ({ ...p, setupError: parseMessage(d) })),
      ),
      subscribeEvent("install_progress", (d) =>
        setProgress((p) => ({ ...p, installLines: [...p.installLines, parseMessage(d)] })),
      ),
      subscribeEvent("install_done", (d) =>
        setProgress((p) => ({ ...p, installDone: parseMessage(d) })),
      ),
      subscribeEvent("install_error", (d) =>
        setProgress((p) => ({ ...p, installError: parseMessage(d) })),
      ),
    ];
    return () => unsubs.forEach((u) => u());
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
