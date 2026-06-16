import { useEffect, useState } from "react";
import * as api from "@/lib/api";
import { Field } from "@/components/forms/Field";
import { Button, Skeleton, useToast } from "@/components/ui";
import { APP_NAME } from "@/lib/brand";

export function IdentityTab() {
  const { toast } = useToast();
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [preferred, setPreferred] = useState("");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    api
      .fetchWhoami()
      .then((p) => {
        setName(p.name ?? "");
        setEmail(p.email ?? "");
        setPreferred(p.preferred_name ?? "");
      })
      .catch((e: unknown) => toast(String(e), "error"))
      .finally(() => setLoading(false));
  }, [toast]);

  const save = async () => {
    if (!name.trim()) {
      toast("Name is required", "error");
      return;
    }
    setSaving(true);
    try {
      await api.setWhoami({ name, email, preferred_name: preferred });
      toast("Identity saved", "success");
    } catch (e) {
      toast(String(e), "error");
    } finally {
      setSaving(false);
    }
  };

  if (loading) return <Skeleton className="h-40 w-full" />;

  return (
    <div className="max-w-lg space-y-3">
      <div>
        <h2 className="text-sm font-semibold">Profile</h2>
        <p className="text-xs text-muted-foreground">
          Tell {APP_NAME} who you are so it can personalize your experience.
        </p>
      </div>
      <Field label="Full name" value={name} onChange={setName} placeholder="e.g. John Doe" />
      <Field
        label="Email"
        value={email}
        onChange={setEmail}
        placeholder="e.g. john@company.com"
      />
      <Field
        label={`How should ${APP_NAME} call you?`}
        value={preferred}
        onChange={setPreferred}
        placeholder="Leave blank to use your first name"
      />
      <div className="flex justify-end">
        <Button onClick={() => void save()} loading={saving}>
          Save
        </Button>
      </div>
    </div>
  );
}
