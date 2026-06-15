import { useCallback, useEffect, useState } from "react";
import { Plus, RefreshCw, Server, Trash2 } from "lucide-react";
import * as api from "@/lib/api";
import { Field } from "@/components/forms/Field";
import { Button, Card, EmptyState, Skeleton, useToast } from "@/components/ui";

export function RegistriesTab() {
  const { toast } = useToast();
  const [registries, setRegistries] = useState<string[]>([]);
  const [loading, setLoading] = useState(true);
  const [url, setURL] = useState("");
  const [token, setToken] = useState("");
  const [adding, setAdding] = useState(false);
  const [refreshing, setRefreshing] = useState(false);

  const reload = useCallback(async () => {
    try {
      setRegistries(await api.fetchRegistries());
    } catch (e) {
      toast(String(e), "error");
    }
  }, [toast]);

  useEffect(() => {
    void reload().finally(() => setLoading(false));
  }, [reload]);

  const add = async () => {
    if (!url.trim()) return;
    setAdding(true);
    try {
      await api.addRegistry(url.trim(), token.trim() || undefined);
      toast("Registry added", "success");
      setURL("");
      setToken("");
      await reload();
    } catch (e) {
      toast(String(e), "error");
    } finally {
      setAdding(false);
    }
  };

  const remove = async (r: string) => {
    try {
      await api.removeRegistry(r);
      toast("Registry removed", "success");
      await reload();
    } catch (e) {
      toast(String(e), "error");
    }
  };

  const refresh = async () => {
    setRefreshing(true);
    try {
      const count = await api.refreshRegistries();
      toast(`Catalog refreshed: ${count} plugin(s)`, "success");
    } catch (e) {
      toast(String(e), "error");
    } finally {
      setRefreshing(false);
    }
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-sm font-semibold">Registries</h2>
          <p className="text-xs text-muted-foreground">
            Extra plugin sources merged with the default registry.
          </p>
        </div>
        <Button variant="outline" size="sm" onClick={() => void refresh()} loading={refreshing}>
          {!refreshing && <RefreshCw className="h-3.5 w-3.5" />} Refresh
        </Button>
      </div>

      <Card className="space-y-3 p-4">
        <Field label="Registry URL" value={url} onChange={setURL} placeholder="https://…" />
        <Field
          label="Auth token (private registries, stored in keychain)"
          secret
          value={token}
          onChange={setToken}
        />
        <div className="flex justify-end">
          <Button size="sm" onClick={() => void add()} loading={adding} disabled={!url.trim()}>
            <Plus className="h-3.5 w-3.5" /> Add registry
          </Button>
        </div>
      </Card>

      {loading ? (
        <Skeleton className="h-16 w-full" />
      ) : registries.length === 0 ? (
        <EmptyState
          icon={Server}
          title="No custom registries"
          description="The default registry is always included."
        />
      ) : (
        <div className="grid gap-2">
          {registries.map((r) => (
            <Card key={r} className="flex items-center gap-3 p-3">
              <span className="flex-1 break-all text-sm">{r}</span>
              <Button variant="ghost" size="icon" aria-label={`Remove ${r}`} onClick={() => void remove(r)}>
                <Trash2 className="h-4 w-4 text-destructive" />
              </Button>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}
