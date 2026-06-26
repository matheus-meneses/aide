import { useEffect, useState } from "react";
import { AlertCircle, ArrowLeft, ArrowRight, CheckCircle2 } from "lucide-react";
import * as api from "@/lib/api";
import { Field } from "@/components/forms/Field";
import { ModelPicker } from "@/components/forms/ModelPicker";
import { Button, Label, Select } from "@/components/ui";
import { APP_NAME } from "@/lib/brand";

export function ProviderStep({ onBack, onNext }: { onBack?: () => void; onNext: () => void }) {
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
        Connect the AI model {APP_NAME} will think with. This is optional &mdash; you can skip it now and
        configure it later in Agent Settings.
      </p>

      <div>
        <Label htmlFor="setup-provider">AI provider</Label>
        <Select
          id="setup-provider"
          value={provider}
          onChange={(e) => onProviderChange(e.target.value)}
        >
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
        <div className="flex items-center gap-2">
          {onBack && (
            <Button variant="ghost" size="sm" onClick={onBack} disabled={saving}>
              <ArrowLeft className="w-4 h-4" /> Back
            </Button>
          )}
          <Button
            variant="outline"
            size="sm"
            onClick={() => void test()}
            disabled={testState === "testing" || !baseURL || !model}
            loading={testState === "testing"}
          >
            Test connection
          </Button>
        </div>
        <Button onClick={() => void save()} loading={saving}>
          {configured ? "Finish" : "Skip for now"} <ArrowRight className="w-4 h-4" />
        </Button>
      </div>
    </div>
  );
}
