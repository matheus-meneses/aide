import { useCallback, useEffect, useState } from "react";
import { ArrowUpCircle, CheckCircle2, Package, RefreshCw } from "lucide-react";
import * as api from "@/lib/api";
import type { GeneralSettings, VersionInfo } from "@/lib/api";
import { APP_NAME } from "@/lib/brand";
import { Button, Label, Select, Skeleton, useToast } from "@/components/ui";
import { MarkdownRenderer } from "@/components/renderers/MarkdownRenderer";
import { useUpdateProgress } from "@/hooks/useUpdateProgress";

export function AboutTab() {
  const { toast } = useToast();
  const [info, setInfo] = useState<VersionInfo | null>(null);
  const [settings, setSettings] = useState<GeneralSettings | null>(null);
  const [loading, setLoading] = useState(true);
  const [checking, setChecking] = useState(false);
  const { progress, start } = useUpdateProgress();

  const loadVersion = useCallback(async () => {
    const v = await api.fetchVersion();
    setInfo(v);
    return v;
  }, []);

  useEffect(() => {
    void Promise.all([loadVersion(), api.fetchConfig()])
      .then(([, cfg]) => setSettings(cfg.settings))
      .catch((e: unknown) => toast(String(e), "error"))
      .finally(() => setLoading(false));
  }, [loadVersion, toast]);

  const check = async () => {
    setChecking(true);
    try {
      const v = await api.checkVersion();
      setInfo(v);
      toast(v.update_available ? `Update available: ${v.latest}` : "You're on the latest version", "success");
    } catch (e) {
      toast(String(e), "error");
    } finally {
      setChecking(false);
    }
  };

  const update = async () => {
    start();
    try {
      await api.triggerUpdate();
    } catch (e) {
      toast(String(e), "error");
    }
  };

  const changeAutoUpdate = async (mode: string) => {
    if (!settings) return;
    const next = { ...settings, auto_update: mode };
    setSettings(next);
    try {
      await api.setSettings({
        concurrency: settings.concurrency,
        timeout_seconds: settings.timeout_seconds,
        verify_ssl: settings.tls?.verify_ssl ?? true,
        ca_bundle: settings.tls?.ca_bundle ?? "",
        log_level: settings.log_level,
        log_format: settings.log_format,
        auto_update: mode,
      });
      toast("Auto-update preference saved", "success");
    } catch (e) {
      toast(String(e), "error");
    }
  };

  if (loading || !info) return <Skeleton className="h-72 w-full" />;

  const isDev = info.current === "dev";

  return (
    <div className="max-w-xl space-y-4">
      <div>
        <h2 className="text-sm font-semibold">About</h2>
        <p className="text-xs text-muted-foreground">Version and updates.</p>
      </div>

      <div className="rounded-md border">
        <div className="flex items-center justify-between gap-4 p-4">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-muted">
              <Package className="h-5 w-5 text-muted-foreground" />
            </div>
            <div className="min-w-0">
              <div className="flex items-center gap-2">
                <span className="text-base font-semibold">{APP_NAME}</span>
                <span className="rounded bg-muted px-1.5 py-0.5 font-mono text-xs text-muted-foreground">
                  {info.current}
                </span>
              </div>
              <div className="mt-0.5 flex items-center gap-2 text-xs text-muted-foreground">
                {info.platform && <span>{info.platform}</span>}
                {!info.update_available && !isDev && (
                  <span className="flex items-center gap-1 text-success">
                    <CheckCircle2 className="h-3 w-3" />
                    Up to date
                  </span>
                )}
                {isDev && <span>Development build</span>}
              </div>
            </div>
          </div>
          <Button variant="outline" size="sm" onClick={() => void check()} loading={checking}>
            <RefreshCw className="mr-1.5 h-3.5 w-3.5" />
            Check for updates
          </Button>
        </div>

        {info.update_available && (
          <div className="border-t border-warning/25 bg-warning/10 p-4">
            <div className="flex items-center justify-between gap-3">
              <div className="flex items-center gap-2 text-sm font-medium text-warning-foreground">
                <ArrowUpCircle className="h-4 w-4 text-warning" />
                Version {info.latest} is available
              </div>
              <UpdateAction
                info={info}
                running={progress.running}
                onUpdate={() => void update()}
              />
            </div>

            {(progress.lines.length > 0 || progress.error || progress.done) && (
              <div className="mt-3 max-h-40 overflow-y-auto rounded bg-background/60 p-2 font-mono text-[11px] leading-relaxed">
                {progress.lines.map((l, i) => (
                  <div key={i} className="text-muted-foreground">
                    {l}
                  </div>
                ))}
                {progress.error && <div className="text-destructive">{progress.error}</div>}
                {progress.done && !progress.error && (
                  <div className="text-success">Done. Restart aide to finish.</div>
                )}
              </div>
            )}
          </div>
        )}

        <div className="flex items-center justify-between gap-4 border-t p-4">
          <div className="min-w-0">
            <Label className="text-sm">Automatic updates</Label>
            <p className="text-xs text-muted-foreground">
              How {APP_NAME} reacts when a newer version is published.
            </p>
          </div>
          <Select
            className="w-56 shrink-0"
            value={settings?.auto_update || "notify"}
            onChange={(e) => void changeAutoUpdate(e.target.value)}
          >
            <option value="off">Off — never check</option>
            <option value="notify">Notify — show a banner</option>
            <option value="auto">Automatic — install</option>
          </Select>
        </div>
      </div>

      {info.notes && (
        <div className="rounded-md border p-4">
          <h3 className="mb-2 text-sm font-semibold">
            What's new in {info.update_available ? info.latest : info.current}
          </h3>
          <MarkdownRenderer content={info.notes} />
        </div>
      )}
    </div>
  );
}

function UpdateAction({
  info,
  running,
  onUpdate,
}: {
  info: VersionInfo;
  running: boolean;
  onUpdate: () => void;
}) {
  if (info.can_self_update) {
    return (
      <Button className="mt-3" onClick={onUpdate} loading={running} disabled={running}>
        Update now
      </Button>
    );
  }
  return (
    <a
      href={info.release_url || "https://github.com/matheus-meneses/aide/releases/latest"}
      target="_blank"
      rel="noreferrer"
      className="mt-3 inline-block text-sm underline text-info"
    >
      View release
    </a>
  );
}
