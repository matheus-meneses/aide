import { useCallback, useEffect, useState } from "react";
import {
  ArrowUpCircle,
  Boxes,
  Download,
  Loader2,
  Lock,
  Package,
  PackageSearch,
  RefreshCw,
  Search,
  SlidersHorizontal,
} from "lucide-react";
import * as api from "@/lib/api";
import { useInstallProgress } from "@/hooks/useInstallProgress";
import { Badge, Button, Card, EmptyState, Input, Skeleton, useToast } from "@/components/ui";

export function MarketplaceTab({ onConfigure }: { onConfigure?: (plugin: string) => void }) {
  const { toast } = useToast();
  const { progress, reset } = useInstallProgress();
  const [plugins, setPlugins] = useState<api.PluginItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [installing, setInstalling] = useState<string>("");
  const [action, setAction] = useState<"install" | "update">("install");
  const [refreshing, setRefreshing] = useState(false);
  const [query, setQuery] = useState("");

  const reload = useCallback(async () => {
    try {
      setPlugins(await api.fetchPlugins());
    } catch (e) {
      toast(String(e), "error");
    }
  }, [toast]);

  useEffect(() => {
    void reload().finally(() => setLoading(false));
  }, [reload]);

  useEffect(() => {
    if (installing && progress.done === installing) {
      const name = installing;
      if (action === "update") {
        toast(`Updated ${name} to the latest version`, "success");
      } else {
        toast(`Installed ${name} — set it up to start using it`, "success");
      }
      setInstalling("");
      reset();
      void reload();
      if (action === "install") onConfigure?.(name);
    }
    if (installing && progress.error) {
      toast(progress.error, "error");
      setInstalling("");
      reset();
    }
  }, [progress.done, progress.error, installing, action, reset, reload, toast, onConfigure]);

  const install = async (name: string) => {
    setAction("install");
    setInstalling(name);
    reset();
    try {
      await api.installPlugin(name);
    } catch (e) {
      toast(String(e), "error");
      setInstalling("");
    }
  };

  const update = async (name: string) => {
    setAction("update");
    setInstalling(name);
    reset();
    try {
      await api.updatePlugin(name);
    } catch (e) {
      toast(String(e), "error");
      setInstalling("");
    }
  };

  const refresh = async () => {
    setRefreshing(true);
    try {
      const count = await api.refreshRegistries();
      toast(`Catalog refreshed: ${count} plugin(s)`, "success");
      await reload();
    } catch (e) {
      toast(String(e), "error");
    } finally {
      setRefreshing(false);
    }
  };

  const renderCard = (p: api.PluginItem) => (
    <Card key={p.name} className="flex flex-col gap-3 p-4">
      <div className="flex items-start gap-3">
        <PluginIcon icon={p.icon} />
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-2 text-sm font-medium">
            {p.name}
            {p.source === "private" ? (
              <Badge tone="info" className="gap-1">
                <Lock className="h-3 w-3" />
                private
              </Badge>
            ) : (
              p.source === "builtin" && (
                <Badge tone="muted" className="gap-1">
                  <Boxes className="h-3 w-3" />
                  builtin
                </Badge>
              )
            )}
            {p.configured ? (
              <Badge tone="success">configured</Badge>
            ) : (
              p.installed && <Badge tone="warning">needs setup</Badge>
            )}
            {p.update_available && <Badge tone="info">update → {p.latest_version}</Badge>}
          </div>
          {p.description && (
            <p className="mt-0.5 line-clamp-2 text-xs text-muted-foreground">{p.description}</p>
          )}
        </div>
      </div>
      <div className="mt-auto flex flex-col gap-2">
        {p.update_available && (
          <Button
            className="w-full"
            size="sm"
            onClick={() => void update(p.name)}
            loading={installing === p.name && action === "update"}
            disabled={installing !== "" && installing !== p.name}
          >
            {!(installing === p.name && action === "update") && (
              <ArrowUpCircle className="h-3.5 w-3.5" />
            )}{" "}
            Update to {p.latest_version}
          </Button>
        )}
        {p.configured ? (
          <Button
            className="w-full"
            size="sm"
            variant="outline"
            onClick={() => onConfigure?.(p.name)}
          >
            <SlidersHorizontal className="h-3.5 w-3.5" /> Manage
          </Button>
        ) : p.installed ? (
          <Button className="w-full" size="sm" onClick={() => onConfigure?.(p.name)}>
            <SlidersHorizontal className="h-3.5 w-3.5" /> Configure
          </Button>
        ) : (
          <Button
            className="w-full"
            size="sm"
            onClick={() => void install(p.name)}
            loading={installing === p.name && action === "install"}
            disabled={installing !== "" && installing !== p.name}
          >
            {!(installing === p.name && action === "install") && (
              <Download className="h-3.5 w-3.5" />
            )}{" "}
            Install
          </Button>
        )}
      </div>
    </Card>
  );

  const q = query.trim().toLowerCase();
  const filtered = q
    ? plugins.filter(
        (p) =>
          p.name.toLowerCase().includes(q) || p.description.toLowerCase().includes(q),
      )
    : plugins;

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-sm font-semibold">Marketplace</h2>
          <p className="text-xs text-muted-foreground">Browse and install plugins from the catalog.</p>
        </div>
        <Button variant="outline" size="sm" onClick={() => void refresh()} loading={refreshing}>
          {!refreshing && <RefreshCw className="h-3.5 w-3.5" />} Refresh catalog
        </Button>
      </div>

      {!loading && plugins.length > 0 && (
        <div className="relative">
          <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Search plugins…"
            className="pl-9"
            aria-label="Search plugins"
          />
        </div>
      )}

      {loading ? (
        <div className="grid gap-3 sm:grid-cols-2">
          {[0, 1, 2, 3].map((i) => (
            <Skeleton key={i} className="h-40 w-full" />
          ))}
        </div>
      ) : plugins.length === 0 ? (
        <EmptyState
          icon={PackageSearch}
          title="No plugins in the catalog"
          description="No registries returned plugins. Add a registry or refresh the catalog."
        />
      ) : filtered.length === 0 ? (
        <EmptyState
          icon={PackageSearch}
          title="No matching plugins"
          description={`Nothing matches "${query}". Try a different search.`}
        />
      ) : (
        <div className="grid gap-3 sm:grid-cols-2">{filtered.map(renderCard)}</div>
      )}

      {installing && (
        <div className="space-y-2 rounded-md border bg-muted/40 p-3">
          <div className="flex items-center gap-2 text-xs font-medium">
            <Loader2 className="h-3.5 w-3.5 animate-spin text-primary" />
            {action === "update" ? "Updating" : "Installing"} {installing}…
          </div>
          {progress.lines.length > 0 && (
            <div className="max-h-40 overflow-auto font-mono text-xs text-muted-foreground">
              {progress.lines.map((l, i) => (
                <div key={i}>{l}</div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

function PluginIcon({ icon }: { icon?: string }) {
  const wrapper =
    "flex h-9 w-9 shrink-0 items-center justify-center overflow-hidden rounded-lg bg-accent text-accent-foreground";
  if (icon && /^(https?:\/\/|data:image\/)/.test(icon)) {
    return (
      <div className={wrapper}>
        <img src={icon} alt="" className="h-5 w-5 object-contain" />
      </div>
    );
  }
  if (icon) {
    return (
      <div className={wrapper}>
        <span className="text-lg leading-none">{icon}</span>
      </div>
    );
  }
  return (
    <div className={wrapper}>
      <Package className="h-5 w-5" />
    </div>
  );
}
