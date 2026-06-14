import { useEffect, useState } from "react";
import { Sparkles, X } from "lucide-react";
import { fetchConfig } from "@/lib/api";

const DISMISS_KEY = "aide.llmBannerDismissed";

export function LLMBanner({ onConfigure }: { onConfigure: () => void }) {
  const [show, setShow] = useState(false);

  useEffect(() => {
    if (sessionStorage.getItem(DISMISS_KEY) === "1") return;
    fetchConfig()
      .then((cfg) => {
        if (!cfg.agent.model || !cfg.agent.base_url) setShow(true);
      })
      .catch(() => {});
  }, []);

  if (!show) return null;

  const dismiss = () => {
    sessionStorage.setItem(DISMISS_KEY, "1");
    setShow(false);
  };

  return (
    <div className="flex items-center gap-3 border-b border-amber-500/20 bg-amber-500/10 px-4 py-2 text-sm">
      <Sparkles className="h-4 w-4 shrink-0 text-amber-500" />
      <span className="flex-1 text-foreground/90">
        Configure the agent to automatically triage your tasks, meetings, and notifications, and get
        proactive briefings.
      </span>
      <button
        onClick={onConfigure}
        className="shrink-0 rounded-md bg-amber-500/20 px-2.5 py-1 text-xs font-medium text-amber-700 transition-colors hover:bg-amber-500/30 dark:text-amber-300"
      >
        Configure agent
      </button>
      <button
        onClick={dismiss}
        className="rounded-md p-1 text-muted-foreground transition-colors hover:bg-accent"
        aria-label="Dismiss"
      >
        <X className="h-4 w-4" />
      </button>
    </div>
  );
}
