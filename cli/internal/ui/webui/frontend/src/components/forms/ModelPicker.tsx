import { useEffect, useRef, useState } from "react";
import { Check, ChevronDown, Loader2 } from "lucide-react";
import * as api from "@/lib/api";
import { Input, Label } from "@/components/ui";
import { cn } from "@/lib/cn";

export function ModelPicker({
  provider,
  baseURL,
  apiKey,
  value,
  onChange,
}: {
  provider: string;
  baseURL: string;
  apiKey: string;
  value: string;
  onChange: (v: string) => void;
}) {
  const containerRef = useRef<HTMLDivElement>(null);
  const loadedSig = useRef<string | null>(null);
  const [models, setModels] = useState<string[]>([]);
  const [loading, setLoading] = useState(false);
  const [hint, setHint] = useState("");
  const [open, setOpen] = useState(false);
  const [active, setActive] = useState(0);

  const sig = `${provider}\u0000${baseURL}\u0000${apiKey}`;

  useEffect(() => {
    if (loadedSig.current !== null && loadedSig.current !== sig) {
      loadedSig.current = null;
      setModels([]);
      setHint("");
    }
  }, [sig]);

  useEffect(() => {
    if (!open) return;
    const onClick = (e: MouseEvent) => {
      if (!containerRef.current?.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener("mousedown", onClick);
    return () => document.removeEventListener("mousedown", onClick);
  }, [open]);

  const query = value.trim().toLowerCase();
  const filtered = query ? models.filter((m) => m.toLowerCase().includes(query)) : models;

  const ensureLoaded = async () => {
    if (loading || !baseURL || loadedSig.current === sig) return;
    setLoading(true);
    setHint("");
    try {
      const res = await api.listModels({ provider, base_url: baseURL, api_key: apiKey });
      loadedSig.current = sig;
      if (res.error || !res.models || res.models.length === 0) {
        setModels([]);
        setHint(res.error ?? "No models advertised — type the name manually.");
      } else {
        setModels(res.models);
      }
    } catch (e) {
      loadedSig.current = sig;
      setModels([]);
      setHint(String(e));
    } finally {
      setLoading(false);
    }
  };

  const focusOpen = () => {
    setOpen(true);
    setActive(0);
    void ensureLoaded();
  };

  const select = (m: string) => {
    onChange(m);
    setOpen(false);
  };

  const onKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Escape") {
      setOpen(false);
      return;
    }
    if (e.key === "ArrowDown" || e.key === "ArrowUp") {
      if (!open) focusOpen();
      if (filtered.length > 0) {
        const dir = e.key === "ArrowDown" ? 1 : -1;
        setActive((i) => (i + dir + filtered.length) % filtered.length);
      }
      e.preventDefault();
    } else if (e.key === "Enter" && open && filtered.length > 0) {
      const choice = filtered[active];
      if (choice) {
        select(choice);
        e.preventDefault();
      }
    }
  };

  const showMenu = open && Boolean(baseURL) && (loading || filtered.length > 0 || models.length > 0);

  return (
    <div>
      <Label>Model</Label>
      <div ref={containerRef} className="relative">
        <Input
          value={value}
          placeholder="e.g. gpt-4o-mini"
          role="combobox"
          aria-expanded={open}
          autoComplete="off"
          className="pr-9"
          onChange={(e) => {
            onChange(e.target.value);
            setActive(0);
            setOpen(true);
            void ensureLoaded();
          }}
          onFocus={focusOpen}
          onKeyDown={onKeyDown}
        />
        <span className="pointer-events-none absolute inset-y-0 right-2.5 flex items-center text-muted-foreground">
          {loading ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            <ChevronDown className={cn("h-4 w-4 transition-transform", open && "rotate-180")} />
          )}
        </span>

        {showMenu && (
          <ul
            role="listbox"
            className="absolute z-50 mt-1 max-h-60 w-full overflow-auto rounded-lg border bg-popover p-1 text-popover-foreground shadow-lg"
          >
            {loading && (
              <li className="flex items-center gap-2 px-2.5 py-2 text-xs text-muted-foreground">
                <Loader2 className="h-3.5 w-3.5 animate-spin" /> Loading models…
              </li>
            )}
            {!loading &&
              filtered.map((m, i) => {
                const selected = m === value;
                return (
                  <li key={m}>
                    <button
                      type="button"
                      role="option"
                      aria-selected={selected}
                      onMouseEnter={() => setActive(i)}
                      onMouseDown={(e) => e.preventDefault()}
                      onClick={() => select(m)}
                      className={cn(
                        "flex w-full items-center justify-between gap-2 rounded-md px-2.5 py-1.5 text-left font-mono text-xs transition-colors",
                        i === active ? "bg-accent text-accent-foreground" : "hover:bg-accent",
                      )}
                    >
                      <span className="truncate">{m}</span>
                      {selected && <Check className="h-3.5 w-3.5 shrink-0" />}
                    </button>
                  </li>
                );
              })}
            {!loading && filtered.length === 0 && models.length > 0 && (
              <li className="px-2.5 py-2 text-xs text-muted-foreground">
                No match — “{value}” will be used as a custom model.
              </li>
            )}
          </ul>
        )}
      </div>
      {hint && <p className="mt-1 text-xs text-muted-foreground">{hint}</p>}
    </div>
  );
}
