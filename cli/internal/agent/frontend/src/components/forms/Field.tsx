import type { ManifestField } from "@/lib/api";
import { Input, Textarea, Label } from "@/components/ui";

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
        {label} {required && <span className="text-destructive">*</span>}
      </Label>
      <Input
        type={secret ? "password" : numeric ? "number" : "text"}
        inputMode={numeric ? "numeric" : undefined}
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
    return (
      <div className="rounded-md border border-dashed p-3 text-xs text-muted-foreground">
        <span className="font-medium text-foreground">{label}</span> is a structured list — configure
        it with <code className="rounded bg-muted px-1">aide source add {field.key}</code> after setup.
      </div>
    );
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
