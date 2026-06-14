import { useEffect, useState } from "react";
import { AlertCircle, ArrowRight, CheckCircle2, Loader2 } from "lucide-react";
import * as api from "@/lib/api";
import { Field } from "@/components/forms/Field";
import { PrimaryButton } from "../shared";

export function ProviderStep({ onNext }: { onNext: () => void }) {
  const [providers, setProviders] = useState<api.ProviderInfo[]>([]);
  const [provider, setProvider] = useState("openai");
  const [baseURL, setBaseURL] = useState("");
  const [model, setModel] = useState("");
  const [apiKey, setApiKey] = useState("");
  const [testState, setTestState] = useState<"idle" | "testing" | "ok" | "fail">("idle");
  const [testMsg, setTestMsg] = useState("");
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    api
      .fetchProviders()
      .then((ps) => {
        setProviders(ps);
        if (ps[0]) {
          setProvider(ps[0].id);
          setBaseURL(ps[0].default_url);
        }
      })
      .catch(() => {});
  }, []);

  const onProviderChange = (id: string) => {
    setProvider(id);
    const p = providers.find((x) => x.id === id);
    if (p) setBaseURL(p.default_url);
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
      if (res.ok) {
        setTestState("ok");
        setTestMsg("");
      } else {
        setTestState("fail");
        setTestMsg(res.error ?? "Connection failed");
      }
    } catch (e) {
      setTestState("fail");
      setTestMsg(String(e));
    }
  };

  const configured = Boolean(baseURL && model);

  const save = async () => {
    setSaving(true);
    try {
      if (configured) {
        await api.setLLM(payload());
      }
      await api.completeSetup();
      onNext();
    } catch (e) {
      setTestState("fail");
      setTestMsg(String(e));
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="space-y-3">
      <p className="text-sm text-muted-foreground">
        Connect the AI model Aide will think with. This is optional &mdash; you can skip it now and
        configure it later in Agent Settings.
      </p>

      <div>
        <label className="mb-1 block text-xs font-medium text-muted-foreground">AI provider</label>
        <select
          value={provider}
          onChange={(e) => onProviderChange(e.target.value)}
          className="w-full rounded-md border bg-background px-3 py-2 text-sm"
        >
          {providers.map((p) => (
            <option key={p.id} value={p.id}>
              {p.label}
            </option>
          ))}
        </select>
      </div>

      <Field label="Base URL" value={baseURL} onChange={setBaseURL} />
      <Field label="Model" value={model} onChange={setModel} placeholder="e.g. gpt-4o-mini" />
      <Field label="API key" secret value={apiKey} onChange={setApiKey} />

      {testState === "ok" && (
        <div className="flex items-center gap-2 text-sm text-emerald-600">
          <CheckCircle2 className="w-4 h-4" /> Connection successful
        </div>
      )}
      {testState === "fail" && (
        <div className="flex items-start gap-2 text-sm text-destructive">
          <AlertCircle className="mt-0.5 w-4 h-4 shrink-0" />
          <span>{testMsg}</span>
        </div>
      )}

      <div className="flex items-center justify-between pt-2">
        <button
          onClick={() => void test()}
          disabled={testState === "testing" || !baseURL || !model}
          className="inline-flex items-center gap-2 rounded-md border px-3 py-2 text-sm transition-colors hover:bg-accent disabled:opacity-50"
        >
          {testState === "testing" && <Loader2 className="w-4 h-4 animate-spin" />}
          Test connection
        </button>
        <PrimaryButton onClick={() => void save()} busy={saving}>
          {configured ? "Finish" : "Skip for now"} <ArrowRight className="w-4 h-4" />
        </PrimaryButton>
      </div>
    </div>
  );
}
