import { useCallback, useEffect, useState } from "react";
import { AlertCircle, ArrowLeft, ArrowRight, Loader2 } from "lucide-react";
import * as api from "@/lib/api";
import { ConfigField, Field } from "@/components/forms/Field";
import { Button } from "@/components/ui";
import { APP_NAME } from "@/lib/brand";
import type { Progress } from "../types";
import { LogPanel } from "../shared";

export function SourceStep({
  progress,
  resetInstall,
  onBack,
  onNext,
}: {
  progress: Progress;
  resetInstall: () => void;
  onBack?: () => void;
  onNext: () => void;
}) {
  const [plugins, setPlugins] = useState<api.PluginItem[]>([]);
  const [catalogLoading, setCatalogLoading] = useState(true);
  const [selected, setSelected] = useState<string>("");
  const [manifest, setManifest] = useState<api.PluginManifest | null>(null);
  const [values, setValues] = useState<Record<string, string>>({});
  const [installing, setInstalling] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  const reload = useCallback(async () => {
    try {
      const list = await api.fetchPlugins();
      setPlugins(list);
      return list.length;
    } catch {
      return 0;
    }
  }, []);

  const retryCatalog = useCallback(() => {
    setCatalogLoading(true);
    void reload().finally(() => setCatalogLoading(false));
  }, [reload]);

  useEffect(() => {
    let cancelled = false;
    let attempts = 0;
    const tick = async () => {
      const count = await reload();
      attempts += 1;
      if (cancelled) return;
      if (count > 0 || attempts >= 10) {
        setCatalogLoading(false);
        return;
      }
      setTimeout(tick, 1000);
    };
    void tick();
    return () => {
      cancelled = true;
    };
  }, [reload]);

  const loadManifest = useCallback(async (name: string) => {
    const m = await api.fetchPluginManifest(name);
    setManifest(m);
    const init: Record<string, string> = {};
    (m.config ?? []).forEach((f) => (init[f.key] = f.default));
    setValues(init);
  }, []);

  useEffect(() => {
    if (progress.installDone && progress.installDone === selected) {
      setInstalling(false);
      void reload();
      loadManifest(selected).catch((e: unknown) => setError(String(e)));
    }
    if (progress.installError) {
      setInstalling(false);
      setError(progress.installError);
    }
  }, [progress.installDone, progress.installError, selected, reload, loadManifest]);

  const choose = async (p: api.PluginItem) => {
    setSelected(p.name);
    setManifest(null);
    setError("");
    resetInstall();
    if (p.installed) {
      await loadManifest(p.name).catch((e: unknown) => setError(String(e)));
    } else {
      setInstalling(true);
      await api.installPlugin(p.name).catch((e: unknown) => {
        setInstalling(false);
        setError(String(e));
      });
    }
  };

  const save = async () => {
    if (!manifest) return;

    const missing: string[] = [];
    (manifest.config ?? []).forEach((f) => {
      if (f.required && !values[f.key]?.trim()) missing.push(f.label || f.key);
    });
    (manifest.credentials ?? []).forEach((c) => {
      if (!values[`cred:${c.key}`]?.trim()) missing.push(c.label || c.key);
    });
    if (missing.length > 0) {
      setError(`Please fill in required field(s): ${missing.join(", ")}`);
      return;
    }

    setSaving(true);
    setError("");
    const config: Record<string, string> = {};
    const credentials: Record<string, string> = {};
    (manifest.config ?? []).forEach((f) => {
      const v = values[f.key];
      if (v) config[f.key] = v;
    });
    (manifest.credentials ?? []).forEach((c) => {
      const v = values[`cred:${c.key}`];
      if (v) credentials[c.key] = v;
    });
    try {
      await api.addSource({ name: manifest.name, config, credentials });
      onNext();
    } catch (e) {
      setError(String(e));
    } finally {
      setSaving(false);
    }
  };

  return (
    <div>
      <p className="text-sm text-muted-foreground">Connect a source for {APP_NAME} to watch (optional).</p>

      {error && (
        <div className="mt-3 flex items-start gap-2 rounded-md border border-destructive/40 bg-destructive/10 p-3 text-sm text-destructive">
          <AlertCircle className="mt-0.5 w-4 h-4 shrink-0" />
          <span>{error}</span>
        </div>
      )}

      {!manifest && (
        <>
          <div className="mt-3 max-h-56 space-y-1.5 overflow-auto">
            {plugins.map((p) => (
              <button
                key={p.name}
                onClick={() => void choose(p)}
                disabled={installing}
                className="flex w-full items-center gap-3 rounded-md border px-3 py-2 text-left transition-colors hover:bg-accent disabled:opacity-50"
              >
                <div className="flex-1">
                  <div className="text-sm font-medium">{p.name}</div>
                  {p.description && (
                    <div className="text-xs text-muted-foreground">{p.description}</div>
                  )}
                </div>
                {p.configured ? (
                  <span className="text-xs text-emerald-600">configured</span>
                ) : p.installed ? (
                  <span className="text-xs text-muted-foreground">installed</span>
                ) : (
                  <span className="text-xs text-muted-foreground">available</span>
                )}
              </button>
            ))}
            {plugins.length === 0 && catalogLoading && (
              <div className="flex items-center gap-2 text-sm text-muted-foreground">
                <Loader2 className="w-4 h-4 animate-spin" /> Loading catalog…
              </div>
            )}
            {plugins.length === 0 && !catalogLoading && (
              <div className="flex items-center justify-between text-sm text-muted-foreground">
                <span>No plugins found in the catalog.</span>
                <button onClick={retryCatalog} className="hover:text-foreground">
                  Retry
                </button>
              </div>
            )}
          </div>

          {installing && (
            <>
              <div className="mt-3 flex items-center gap-2 text-sm text-muted-foreground">
                <Loader2 className="w-4 h-4 animate-spin" /> Installing {selected}…
              </div>
              <LogPanel lines={progress.installLines} />
            </>
          )}

          <div className="mt-5 flex items-center justify-between">
            {onBack ? (
              <Button variant="ghost" size="sm" onClick={onBack} disabled={installing || saving}>
                <ArrowLeft className="w-4 h-4" /> Back
              </Button>
            ) : (
              <span />
            )}
            <Button variant="ghost" size="sm" onClick={onNext} disabled={installing || saving}>
              Skip for now
            </Button>
          </div>
        </>
      )}

      {manifest && (
        <div className="mt-4 space-y-3">
          <div className="text-sm font-medium">{manifest.name}</div>
          {(manifest.config ?? []).map((f) => (
            <ConfigField
              key={f.key}
              field={f}
              value={values[f.key] ?? ""}
              onChange={(v) => setValues((s) => ({ ...s, [f.key]: v }))}
            />
          ))}
          {(manifest.credentials ?? []).map((c) => (
            <Field
              key={c.key}
              label={c.label || c.key}
              required
              secret={c.secret}
              value={values[`cred:${c.key}`] ?? ""}
              onChange={(v) => setValues((s) => ({ ...s, [`cred:${c.key}`]: v }))}
            />
          ))}
          <div className="flex items-center justify-between pt-2">
            <Button variant="ghost" size="sm" onClick={() => setManifest(null)}>
              <ArrowLeft className="w-4 h-4" /> Back
            </Button>
            <Button onClick={() => void save()} loading={saving}>
              Connect source <ArrowRight className="w-4 h-4" />
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}
