import { useMemo } from "react";
import { Plus, X } from "lucide-react";
import type { ManifestField } from "@/lib/api";
import { Button, Input, Textarea, Label } from "@/components/ui";

export function Field({
  label,
  value,
  onChange,
  secret,
  required,
  placeholder,
  numeric,
}: {
  label: string;
  value: string;
  onChange: (v: string) => void;
  secret?: boolean;
  required?: boolean;
  placeholder?: string;
  numeric?: boolean;
}) {
  return (
    <div>
      <Label>
        {label}{" "}
        {required && (
          <>
            <span className="text-destructive" aria-hidden="true">
              *
            </span>
            <span className="sr-only">required</span>
          </>
        )}
      </Label>
      <Input
        type={secret ? "password" : numeric ? "number" : "text"}
        inputMode={numeric ? "numeric" : undefined}
        required={required}
        aria-required={required || undefined}
        value={value}
        placeholder={placeholder}
        onChange={(e) => onChange(e.target.value)}
      />
    </div>
  );
}

export function ConfigField({
  field,
  value,
  onChange,
}: {
  field: ManifestField;
  value: string;
  onChange: (v: string) => void;
}) {
  const label = field.label || field.key;

  if (field.type === "object_list") {
    return <ObjectListField field={field} value={value} onChange={onChange} />;
  }

  if (field.type === "string_list") {
    return (
      <div>
        <Label>
          {label} {field.required && <span className="text-destructive">*</span>}
          <span className="ml-1 font-normal">(one per line)</span>
        </Label>
        <Textarea
          value={value}
          rows={3}
          placeholder={"e.g.\nCalendário\nWork"}
          onChange={(e) => onChange(e.target.value)}
        />
      </div>
    );
  }

  return (
    <Field
      label={label}
      required={field.required}
      value={value}
      numeric={field.type === "integer"}
      placeholder={field.default ? `default: ${field.default}` : undefined}
      onChange={onChange}
    />
  );
}

type Row = Record<string, string>;

function cellToText(v: unknown): string {
  if (v == null) return "";
  if (typeof v === "string") return v;
  if (typeof v === "number" || typeof v === "boolean") return String(v);
  return JSON.stringify(v);
}

function ObjectListField({
  field,
  value,
  onChange,
}: {
  field: ManifestField;
  value: string;
  onChange: (v: string) => void;
}) {
  const label = field.label || field.key;
  const subFields = useMemo(() => field.fields ?? [], [field.fields]);

  const rows = useMemo<Row[]>(() => {
    if (!value.trim()) return [];
    try {
      const parsed: unknown = JSON.parse(value);
      if (!Array.isArray(parsed)) return [];
      return parsed.map((entry) => {
        const obj = (entry ?? {}) as Record<string, unknown>;
        const row: Row = {};
        subFields.forEach((sf) => {
          row[sf.key] = cellToText(obj[sf.key]);
        });
        return row;
      });
    } catch {
      return [];
    }
  }, [value, subFields]);

  const commit = (next: Row[]) => onChange(next.length === 0 ? "" : JSON.stringify(next));

  const addRow = () => {
    const blank: Row = {};
    subFields.forEach((sf) => (blank[sf.key] = sf.default));
    commit([...rows, blank]);
  };

  const updateCell = (idx: number, key: string, v: string) =>
    commit(rows.map((r, i) => (i === idx ? { ...r, [key]: v } : r)));

  const removeRow = (idx: number) => commit(rows.filter((_, i) => i !== idx));

  return (
    <div>
      <Label>
        {label}{" "}
        {field.required && (
          <>
            <span className="text-destructive" aria-hidden="true">
              *
            </span>
            <span className="sr-only">required</span>
          </>
        )}
      </Label>
      <div className="mt-1 space-y-2">
        {rows.length === 0 && (
          <p className="text-xs text-muted-foreground">No entries yet — add at least one.</p>
        )}
        {rows.map((row, idx) => (
          <div key={idx} className="relative rounded-md border p-3 pr-9">
            <button
              type="button"
              onClick={() => removeRow(idx)}
              aria-label={`Remove ${label} entry ${idx + 1}`}
              className="absolute right-2 top-2 text-muted-foreground transition-colors hover:text-destructive"
            >
              <X className="h-4 w-4" />
            </button>
            <div className="space-y-2">
              {subFields.map((sf) => (
                <Field
                  key={sf.key}
                  label={sf.label || sf.key}
                  required={sf.required}
                  value={row[sf.key] ?? ""}
                  placeholder={sf.default ? `default: ${sf.default}` : undefined}
                  onChange={(v) => updateCell(idx, sf.key, v)}
                />
              ))}
            </div>
          </div>
        ))}
        <Button type="button" variant="outline" size="sm" onClick={addRow}>
          <Plus className="h-3.5 w-3.5" /> Add entry
        </Button>
      </div>
    </div>
  );
}
