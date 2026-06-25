import { useEffect, useState } from "react";
import * as api from "@/lib/api";
import { Button, Skeleton, Textarea, useToast } from "@/components/ui";

export function ContextTab() {
  const { toast } = useToast();
  const [context, setContext] = useState("");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    api
      .fetchConfig()
      .then((cfg) => setContext(cfg.agent.user_context))
      .catch((e: unknown) => toast(String(e), "error"))
      .finally(() => setLoading(false));
  }, [toast]);

  const save = async () => {
    setSaving(true);
    try {
      await api.setUserContext(context);
      toast("Context saved", "success");
    } catch (e) {
      toast(String(e), "error");
    } finally {
      setSaving(false);
    }
  };

  if (loading) return <Skeleton className="h-40 w-full" />;

  return (
    <div className="max-w-lg space-y-3">
      <div>
        <h2 className="text-sm font-semibold">Your context</h2>
        <p className="text-xs text-muted-foreground">
          Tell the assistant about you and how to help — your role, priorities,
          and preferences. This shapes every response and briefing.
        </p>
      </div>
      <Textarea
        rows={6}
        value={context}
        onChange={(e) => setContext(e.target.value)}
        placeholder="e.g. I'm a tech lead. Prioritize production incidents and PR reviews; keep updates terse."
        aria-label="Your context"
      />
      <div className="flex justify-end">
        <Button onClick={() => void save()} loading={saving}>
          Save
        </Button>
      </div>
    </div>
  );
}
