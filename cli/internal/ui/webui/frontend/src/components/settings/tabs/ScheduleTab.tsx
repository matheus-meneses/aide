import { useEffect, useState } from "react";
import * as api from "@/lib/api";
import { Field } from "@/components/forms/Field";
import { Button, Skeleton, useToast } from "@/components/ui";

export function ScheduleTab() {
  const { toast } = useToast();
  const [interval, setInterval] = useState("");
  const [briefings, setBriefings] = useState("");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    api
      .fetchConfig()
      .then((cfg) => {
        setInterval(cfg.agent.run_interval);
        setBriefings((cfg.agent.briefing_times ?? []).join(", "));
      })
      .catch((e: unknown) => toast(String(e), "error"))
      .finally(() => setLoading(false));
  }, [toast]);

  const save = async () => {
    setSaving(true);
    try {
      await api.setSchedule({
        run_interval: interval,
        briefing_times: briefings
          .split(/[,\s]+/)
          .map((s) => s.trim())
          .filter(Boolean),
      });
      toast("Schedule saved", "success");
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
        <h2 className="text-sm font-semibold">Schedule</h2>
        <p className="text-xs text-muted-foreground">
          How often the agent re-collects, and when it sends daily briefings.
        </p>
      </div>
      <Field
        label="Run interval"
        value={interval}
        onChange={setInterval}
        placeholder="e.g. 30m, 1h"
      />
      <Field
        label="Daily briefing times (comma-separated, 24h)"
        value={briefings}
        onChange={setBriefings}
        placeholder="e.g. 08:00, 17:30"
      />
      <div className="flex justify-end">
        <Button onClick={() => void save()} loading={saving}>
          Save
        </Button>
      </div>
    </div>
  );
}
