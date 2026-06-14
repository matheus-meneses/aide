import { useEffect, useState } from "react";
import * as api from "@/lib/api";
import { Field } from "@/components/forms/Field";
import { Button, Label, Select, Skeleton, Switch, useToast } from "@/components/ui";

export function GeneralTab() {
  const { toast } = useToast();
  const [concurrency, setConcurrency] = useState("");
  const [timeout, setTimeout] = useState("");
  const [verifySSL, setVerifySSL] = useState(true);
  const [caBundle, setCABundle] = useState("");
  const [logLevel, setLogLevel] = useState("info");
  const [logFormat, setLogFormat] = useState("text");
  const [dataDir, setDataDir] = useState("");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    api
      .fetchConfig()
      .then((cfg) => {
        const s = cfg.settings;
        setConcurrency(String(s.concurrency));
        setTimeout(String(s.timeout_seconds));
        setVerifySSL(s.tls?.verify_ssl !== false);
        setCABundle(s.tls?.ca_bundle ?? "");
        setLogLevel(s.log_level || "info");
        setLogFormat(s.log_format || "text");
        setDataDir(s.data_dir);
      })
      .catch((e: unknown) => toast(String(e), "error"))
      .finally(() => setLoading(false));
  }, [toast]);

  const save = async () => {
    setSaving(true);
    try {
      await api.setSettings({
        concurrency: Number(concurrency) || 0,
        timeout_seconds: Number(timeout) || 0,
        verify_ssl: verifySSL,
        ca_bundle: caBundle,
        log_level: logLevel,
        log_format: logFormat,
      });
      toast("Settings saved", "success");
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
        <h2 className="text-sm font-semibold">General</h2>
        <p className="text-xs text-muted-foreground">Runtime, networking, and logging.</p>
      </div>

      <Field label="Concurrency" numeric value={concurrency} onChange={setConcurrency} />
      <Field label="Timeout (seconds)" numeric value={timeout} onChange={setTimeout} />

      <div className="flex items-center justify-between rounded-md border p-3">
        <div>
          <div className="text-sm font-medium">Verify TLS certificates</div>
          <div className="text-xs text-muted-foreground">Disable only for trusted internal hosts.</div>
        </div>
        <Switch checked={verifySSL} onCheckedChange={setVerifySSL} aria-label="Verify TLS" />
      </div>

      <Field
        label="CA bundle path"
        value={caBundle}
        onChange={setCABundle}
        placeholder="/path/to/ca.pem (optional)"
      />

      <div>
        <Label>Log level</Label>
        <Select value={logLevel} onChange={(e) => setLogLevel(e.target.value)}>
          {["debug", "info", "warn", "error"].map((l) => (
            <option key={l} value={l}>
              {l}
            </option>
          ))}
        </Select>
      </div>

      <div>
        <Label>Log format</Label>
        <Select value={logFormat} onChange={(e) => setLogFormat(e.target.value)}>
          {["text", "json"].map((f) => (
            <option key={f} value={f}>
              {f}
            </option>
          ))}
        </Select>
      </div>

      <div>
        <Label>Data directory</Label>
        <div className="rounded-md border bg-muted/40 px-3 py-2 text-sm text-muted-foreground break-all">
          {dataDir || "—"}
        </div>
      </div>

      <div className="flex justify-end">
        <Button onClick={() => void save()} loading={saving}>
          Save
        </Button>
      </div>
    </div>
  );
}
