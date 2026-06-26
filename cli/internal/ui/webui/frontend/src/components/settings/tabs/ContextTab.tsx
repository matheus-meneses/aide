import { useEffect, useState } from "react";
import * as api from "@/lib/api";
import { Field } from "@/components/forms/Field";
import { Button, Label, Select, Skeleton, Textarea, useToast } from "@/components/ui";

const NOTIFICATION_LEVELS: { value: string; label: string }[] = [
  { value: "silent", label: "Silent (activity feed only)" },
  { value: "urgent_only", label: "Urgent only (default)" },
  { value: "normal", label: "Normal (important changes)" },
  { value: "all", label: "All noteworthy changes" },
];

export function ContextTab() {
  const { toast } = useToast();
  const [context, setContext] = useState("");
  const [notifications, setNotifications] = useState("urgent_only");
  const [maxNotifs, setMaxNotifs] = useState("");
  const [tone, setTone] = useState("");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    api
      .fetchConfig()
      .then((cfg) => {
        setContext(cfg.agent.user_context);
        const p = cfg.agent.preferences;
        setNotifications(p.notifications || "urgent_only");
        setMaxNotifs(p.max_notifications_per_cycle ? String(p.max_notifications_per_cycle) : "");
        setTone(p.tone);
      })
      .catch((e: unknown) => toast(String(e), "error"))
      .finally(() => setLoading(false));
  }, [toast]);

  const save = async () => {
    setSaving(true);
    try {
      await api.setAgentPreferences({
        notifications,
        max_notifications_per_cycle: Number(maxNotifs) || 0,
        tone: tone.trim(),
      });
      await api.setUserContext(context);
      toast("Saved", "success");
    } catch (e) {
      toast(String(e), "error");
    } finally {
      setSaving(false);
    }
  };

  if (loading) return <Skeleton className="h-72 w-full" />;

  return (
    <div className="max-w-lg space-y-3">
      <div>
        <h2 className="text-sm font-semibold">Behavior &amp; context</h2>
        <p className="text-xs text-muted-foreground">
          These preferences override the assistant&apos;s defaults but never its
          safety rules. Tell it how to notify you, how to talk, and anything else
          it should know about you.
        </p>
      </div>

      <div>
        <Label>Notifications</Label>
        <Select
          value={notifications}
          onChange={(e) => setNotifications(e.target.value)}
          aria-label="Notification level"
        >
          {NOTIFICATION_LEVELS.map((l) => (
            <option key={l.value} value={l.value}>
              {l.label}
            </option>
          ))}
        </Select>
      </div>

      <Field
        label="Max notifications per cycle (blank = default)"
        numeric
        value={maxNotifs}
        onChange={setMaxNotifs}
        placeholder="1"
      />

      <Field
        label="Tone"
        value={tone}
        onChange={setTone}
        placeholder="e.g. formal, friendly, terse (blank = default)"
      />

      <div>
        <Label>About you</Label>
        <Textarea
          rows={6}
          value={context}
          onChange={(e) => setContext(e.target.value)}
          placeholder="e.g. I'm a tech lead. Prioritize production incidents and PR reviews."
          aria-label="Your context"
        />
      </div>

      <div className="flex justify-end">
        <Button onClick={() => void save()} loading={saving}>
          Save
        </Button>
      </div>
    </div>
  );
}
