import { useCallback, useEffect, useState } from "react";
import { Boxes, Plug, Settings2, SlidersHorizontal, Trash2 } from "lucide-react";
import * as api from "@/lib/api";
import { cn } from "@/lib/cn";
import { ConfigField, Field } from "@/components/forms/Field";
import {
  Badge,
  Button,
  Card,
  ConfirmDialog,
  EmptyState,
  Skeleton,
  Switch,
  useToast,
} from "@/components/ui";

function toText(v: unknown): string {
  if (typeof v === "string") return v;
  if (typeof v === "number" || typeof v === "boolean") return String(v);
  if (v == null) return "";
  return JSON.stringify(v);
}

interface InstalledEntry {
  plugin: api.PluginItem;
  source: api.SourceSnapshot | null;
}

export function InstalledTab({
  configureTarget = "",
  onConsumeTarget,
}: {
  configureTarget?: string;
  onConsumeTarget?: () => void;
}) {
  const { toast } = useToast();
  const [entries, setEntries] = useState<InstalledEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [active, setActive] = useState<string>("");
  const [pendingUninstall, setPendingUninstall] = useState<string>("");
  const [uninstalling, setUninstalling] = useState(false);

  const reload = useCallback(async () => {
    try {
      const [plugins, sources] = await Promise.all([
        api.fetchPlugins(),
        api.fetchSources(),
      ]);
      const merged = plugins
        .filter((p) => p.installed)
        .map((p) => ({
          plugin: p,
          source: sources.find((s) => s.name === p.name) ?? null,
        }));
      setEntries(merged);
      return merged;
    } catch (e) {
      toast(String(e), "error");
      return [];
    }
  }, [toast]);

  useEffect(() => {
    void reload().finally(() => setLoading(false));
  }, [reload]);

  useEffect(() => {
    if (!configureTarget) return;
    if (entries.some((e) => e.plugin.name === configureTarget)) {
      setActive(configureTarget);
      onConsumeTarget?.();
    }
  }, [configureTarget, entries, onConsumeTarget]);

  const toggle = async (name: string, enabled: boolean) => {
    setEntries((list) =>
      list.map((e) =>
        e.source?.name === name ? { ...e, source: { ...e.source, enabled } } : e,
      ),
    );
    try {
      await api.toggleSource(name, enabled);
      toast(`${name} ${enabled ? "enabled" : "disabled"}`, "success");
    } catch (e) {
      toast(String(e), "error");
      void reload();
    }
  };

  const uninstall = async () => {
    setUninstalling(true);
    try {
      await api.uninstallPlugin(pendingUninstall);
      toast(`Uninstalled ${pendingUninstall}`, "success");
      setPendingUninstall("");
      await reload();
    } catch (e) {
      toast(String(e), "error");
    } finally {
      setUninstalling(false);
    }
  };

  return (
    <div className="space-y-4">
      <div>
        <h2 className="text-sm font-semibold">Installed plugins & sources</h2>
        <p className="text-xs text-muted-foreground">
          Configure, enable, reconfigure, or uninstall the plugins you have installed.
        </p>
      </div>

      {loading ? (
        <div className="grid gap-3 sm:grid-cols-2">
          {[0, 1, 2, 3].map((i) => (
            <Skeleton key={i} className="h-40 w-full" />
          ))}
        </div>
      ) : entries.length === 0 ? (
        <EmptyState
          icon={Boxes}
          title="No plugins installed"
          description="Head to the Marketplace tab to install a plugin, then configure it here."
        />
      ) : (
        <div className="grid items-start gap-3 sm:grid-cols-2">
          {entries.map(({ plugin, source }) => {
            const expanded = active === plugin.name;
            return (
              <Card
                key={plugin.name}
                ref={
                  expanded
                    ? (el) => el?.scrollIntoView({ behavior: "smooth", block: "nearest" })
                    : undefined
                }
                className={cn("flex flex-col gap-3 p-4", expanded && "sm:col-span-2")}
              >
                <div className="flex items-start gap-3">
                  <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-accent text-accent-foreground">
                    <Plug className="h-5 w-5" />
                  </div>
                  <div className="min-w-0 flex-1">
                    <div className="flex flex-wrap items-center gap-2 text-sm font-medium">
                      {plugin.name}
                      {plugin.runtime && <Badge tone="muted">{plugin.runtime}</Badge>}
                      {source ? (
                        source.has_credentials && <Badge tone="muted">credentials set</Badge>
                      ) : (
                        <Badge tone="warning">needs setup</Badge>
                      )}
                      {source && !source.enabled && <Badge tone="warning">disabled</Badge>}
                    </div>
                    {plugin.description && (
                      <p className="mt-0.5 line-clamp-2 text-xs text-muted-foreground">
                        {plugin.description}
                      </p>
                    )}
                  </div>
                </div>

                <div className="mt-auto flex items-center justify-between border-t pt-3">
                  {source ? (
                    <label className="flex items-center gap-2 text-xs text-muted-foreground">
                      <Switch
                        checked={source.enabled}
                        onCheckedChange={(v) => void toggle(source.name, v)}
                        aria-label={`Toggle ${plugin.name}`}
                      />
                      {source.enabled ? "Enabled" : "Disabled"}
                    </label>
                  ) : (
                    <Button
                      size="sm"
                      onClick={() => setActive((a) => (a === plugin.name ? "" : plugin.name))}
                    >
                      <SlidersHorizontal className="h-3.5 w-3.5" /> Configure
                    </Button>
                  )}
                  <div className="flex items-center gap-1">
                    {source && (
                      <Button
                        variant="ghost"
                        size="icon"
                        aria-label={`Reconfigure ${plugin.name}`}
                        onClick={() => setActive((a) => (a === plugin.name ? "" : plugin.name))}
                      >
                        <Settings2 className="h-4 w-4" />
                      </Button>
                    )}
                    <Button
                      variant="ghost"
                      size="icon"
                      aria-label={`Uninstall ${plugin.name}`}
                      onClick={() => setPendingUninstall(plugin.name)}
                    >
                      <Trash2 className="h-4 w-4 text-destructive" />
                    </Button>
                  </div>
                </div>

                {expanded && (
                  <SourceForm
                    pluginName={plugin.name}
                    source={source}
                    onDone={() => {
                      setActive("");
                      void reload();
                    }}
                  />
                )}
              </Card>
            );
          })}
        </div>
      )}

      <ConfirmDialog
        open={pendingUninstall !== ""}
        title={`Uninstall ${pendingUninstall}?`}
        description="This removes the plugin files, and (if configured) its source and stored credentials."
        confirmLabel="Uninstall"
        destructive
        loading={uninstalling}
        onConfirm={() => void uninstall()}
        onClose={() => setPendingUninstall("")}
      />
    </div>
  );
}

function SourceForm({
  pluginName,
  source,
  onDone,
}: {
  pluginName: string;
  source: api.SourceSnapshot | null;
  onDone: () => void;
}) {
  const { toast } = useToast();
  const [manifest, setManifest] = useState<api.PluginManifest | null>(null);
  const [values, setValues] = useState<Record<string, string>>({});
  const [saving, setSaving] = useState(false);
  const configured = source !== null;

  useEffect(() => {
    api
      .fetchPluginManifest(pluginName)
      .then((m) => {
        setManifest(m);
        const init: Record<string, string> = {};
        (m.config ?? []).forEach((f) => {
          const existing = source?.config?.[f.key];
          init[f.key] = Array.isArray(existing)
            ? existing.map(toText).join("\n")
            : toText(existing) || f.default;
        });
        setValues(init);
      })
      .catch((e: unknown) => toast(String(e), "error"));
  }, [pluginName, source, toast]);

  const save = async () => {
    if (!manifest) return;
    setSaving(true);
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
      if (configured) {
        await api.reconfigureSource({ name: pluginName, config, credentials });
      } else {
        await api.addSource({ name: pluginName, config, credentials });
      }
      toast(`Saved ${pluginName}`, "success");
      onDone();
    } catch (e) {
      toast(String(e), "error");
    } finally {
      setSaving(false);
    }
  };

  if (!manifest) return null;

  return (
    <div className="mt-3 space-y-3 border-t pt-3">
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
          label={`${c.label || c.key}${source?.has_credentials ? " (leave blank to keep)" : ""}`}
          required={!configured}
          secret={c.secret}
          value={values[`cred:${c.key}`] ?? ""}
          onChange={(v) => setValues((s) => ({ ...s, [`cred:${c.key}`]: v }))}
        />
      ))}
      <div className="flex justify-end">
        <Button size="sm" onClick={() => void save()} loading={saving}>
          {configured ? "Save" : "Connect source"}
        </Button>
      </div>
    </div>
  );
}
