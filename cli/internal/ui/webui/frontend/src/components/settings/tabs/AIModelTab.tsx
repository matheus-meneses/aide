import { useEffect, useState } from "react";
import { CheckCircle2, AlertCircle } from "lucide-react";
import * as api from "@/lib/api";
import { Field } from "@/components/forms/Field";
import { ModelPicker } from "@/components/forms/ModelPicker";
import { Button, Label, Select, Skeleton, useToast } from "@/components/ui";
import { APP_NAME } from "@/lib/brand";

export function AIModelTab() {
  const { toast } = useToast();
  const [providers, setProviders] = useState<api.ProviderInfo[]>([]);
  const [provider, setProvider] = useState("openai");
  const [baseURL, setBaseURL] = useState("");
  const [model, setModel] = useState("");
  const [apiKey, setApiKey] = useState("");
  const [hasKey, setHasKey] = useState(false);
  const [loading, setLoading] = useState(true);
  const [testState, setTestState] = useState<"idle" | "testing" | "ok" | "fail">("idle");
  const [testMsg, setTestMsg] = useState("");
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    Promise.all([api.fetchProviders(), api.fetchConfig()])
      .then(([ps, cfg]) => {
        setProviders(ps);
        setProvider(cfg.agent.provider || ps[0]?.id || "openai");
        setBaseURL(cfg.agent.base_url);
        setModel(cfg.agent.model);
        setHasKey(cfg.agent.has_api_key);
      })
      .catch((e: unknown) => toast(String(e), "error"))
      .finally(() => setLoading(false));
  }, [toast]);

  const onProviderChange = (id: string) => {
    setProvider(id);
    const p = providers.find((x) => x.id === id);
    if (p && !baseURL) setBaseURL(p.default_url);
    setTestState("idle");
  };

  const payload = (): api.LLMPayload => ({
    provider,
    base_url: baseURL,
    model,
    api_key: apiKey,
  });

  const test = async () => {
    setTestState("testing");
    try {
      const res = await api.testConnection(payload());
      setTestState(res.ok ? "ok" : "fail");
      setTestMsg(res.ok ? "" : (res.error ?? "Connection failed"));
    } catch (e) {
      setTestState("fail");
      setTestMsg(String(e));
    }
  };

  const save = async () => {
    setSaving(true);
    try {
      await api.setLLM(payload());
      toast("AI model saved", "success");
      if (apiKey) {
        setHasKey(true);
        setApiKey("");
      }
    } catch (e) {
      toast(String(e), "error");
    } finally {
      setSaving(false);
    }
  };

  if (loading) return <Skeleton className="h-64 w-full" />;

  return (
    <div className="max-w-lg space-y-3">
      <div>
        <h2 className="text-sm font-semibold">AI model</h2>
        <p className="text-xs text-muted-foreground">The endpoint {APP_NAME} thinks with.</p>
      </div>

      <div>
        <Label>AI provider</Label>
        <Select value={provider} onChange={(e) => onProviderChange(e.target.value)}>
          {providers.map((p) => (
            <option key={p.id} value={p.id}>
              {p.label}
            </option>
          ))}
        </Select>
      </div>

      <Field label="Base URL" value={baseURL} onChange={setBaseURL} />
      <ModelPicker
        provider={provider}
        baseURL={baseURL}
        apiKey={apiKey}
        value={model}
        onChange={setModel}
      />
      <Field
        label={`API key${hasKey ? " (leave blank to keep)" : ""}`}
        secret
        value={apiKey}
        onChange={setApiKey}
      />

      {testState === "ok" && (
        <div className="flex items-center gap-2 text-sm text-success">
          <CheckCircle2 className="h-4 w-4" /> Connection successful
        </div>
      )}
      {testState === "fail" && (
        <div className="flex items-start gap-2 text-sm text-destructive">
          <AlertCircle className="mt-0.5 h-4 w-4 shrink-0" />
          <span>{testMsg}</span>
        </div>
      )}

      <div className="flex items-center justify-between pt-1">
        <Button
          variant="outline"
          onClick={() => void test()}
          loading={testState === "testing"}
          disabled={!baseURL || !model}
        >
          Test connection
        </Button>
        <Button onClick={() => void save()} loading={saving} disabled={!baseURL || !model}>
          Save
        </Button>
      </div>
    </div>
  );
}
