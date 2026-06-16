import { useCallback, useEffect, useState } from "react";
import { ArrowUpCircle, RefreshCw } from "lucide-react";
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
      const v = await loadVersion();
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

      <div className="rounded-md border p-4">
        <div className="flex items-center justify-between">
          <div>
            <div className="text-lg font-semibold">{APP_NAME}</div>
            <div className="text-sm text-muted-foreground">Version {info.current}</div>
            {info.platform && (
              <div className="text-xs text-muted-foreground">{info.platform}</div>
            )}
          </div>
          <Button variant="outline" onClick={() => void check()} loading={checking}>
            <RefreshCw className="mr-1.5 h-3.5 w-3.5" />
            Check for updates
          </Button>
        </div>

        {info.update_available && (
          <div className="mt-4 rounded-md border border-warning/25 bg-warning/10 p-3">
            <div className="flex items-center gap-2 text-sm font-medium text-warning-foreground">
              <ArrowUpCircle className="h-4 w-4 text-warning" />
              {info.latest} is available
            </div>

            <UpdateAction
              info={info}
              running={progress.running}
              onUpdate={() => void update()}
            />

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

        {!info.update_available && !isDev && (
          <p className="mt-3 text-xs text-muted-foreground">You're on the latest version.</p>
        )}
      </div>

      {info.notes && (
        <div className="rounded-md border p-4">
          <h3 className="mb-2 text-sm font-semibold">
            What's new in {info.latest || info.current}
          </h3>
          <MarkdownRenderer content={info.notes} />
        </div>
      )}

      <div className="rounded-md border p-4">
        <Label>Automatic updates</Label>
        <p className="mb-2 text-xs text-muted-foreground">
          How aide reacts when a newer version is published.
        </p>
        <Select
          value={settings?.auto_update || "notify"}
          onChange={(e) => void changeAutoUpdate(e.target.value)}
        >
          <option value="off">Off — never check</option>
          <option value="notify">Notify — show a banner (default)</option>
          <option value="auto">Automatic — download and install</option>
        </Select>
      </div>
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
