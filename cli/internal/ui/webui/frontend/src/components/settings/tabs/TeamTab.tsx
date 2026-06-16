import { useEffect, useState } from "react";
import { Plus, Trash2, Users } from "lucide-react";
import * as api from "@/lib/api";
import { Field } from "@/components/forms/Field";
import { Button, Card, EmptyState, Skeleton, useToast } from "@/components/ui";
import { APP_NAME } from "@/lib/brand";

const blank: api.TeamMember = { name: "" };

export function TeamTab() {
  const { toast } = useToast();
  const [members, setMembers] = useState<api.TeamMember[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [draft, setDraft] = useState<api.TeamMember | null>(null);
  const [editIndex, setEditIndex] = useState<number | null>(null);

  useEffect(() => {
    api
      .fetchTeam()
      .then(setMembers)
      .catch((e: unknown) => toast(String(e), "error"))
      .finally(() => setLoading(false));
  }, [toast]);

  const persist = async (next: api.TeamMember[]) => {
    setSaving(true);
    try {
      await api.setTeam(next);
      setMembers(next);
      setDraft(null);
      setEditIndex(null);
      toast("Team saved", "success");
    } catch (e) {
      toast(String(e), "error");
    } finally {
      setSaving(false);
    }
  };

  const remove = (idx: number) => persist(members.filter((_, i) => i !== idx));

  const commit = () => {
    if (!draft || !draft.name.trim()) {
      toast("Name is required", "error");
      return;
    }
    const next = [...members];
    if (editIndex != null) next[editIndex] = draft;
    else next.push(draft);
    void persist(next);
  };

  if (loading) return <Skeleton className="h-64 w-full" />;

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-sm font-semibold">Team</h2>
          <p className="text-xs text-muted-foreground">People {APP_NAME} tracks from your config.</p>
        </div>
        <Button
          size="sm"
          onClick={() => {
            setDraft({ ...blank });
            setEditIndex(null);
          }}
        >
          <Plus className="h-3.5 w-3.5" /> Add member
        </Button>
      </div>

      {draft && (
        <Card className="space-y-3 p-4">
          <div className="text-sm font-medium">{editIndex != null ? "Edit member" : "New member"}</div>
          <div className="grid gap-3 sm:grid-cols-2">
            <Field label="Name" required value={draft.name} onChange={(v) => setDraft({ ...draft, name: v })} />
            <Field label="Email" value={draft.email ?? ""} onChange={(v) => setDraft({ ...draft, email: v })} />
            <Field label="Role" value={draft.role ?? ""} onChange={(v) => setDraft({ ...draft, role: v })} />
            <Field label="Department" value={draft.department ?? ""} onChange={(v) => setDraft({ ...draft, department: v })} />
            <Field label="Branch" value={draft.branch ?? ""} onChange={(v) => setDraft({ ...draft, branch: v })} />
            <Field label="Registration" value={draft.registration ?? ""} onChange={(v) => setDraft({ ...draft, registration: v })} />
            <Field label="Manager" value={draft.manager ?? ""} onChange={(v) => setDraft({ ...draft, manager: v })} />
            <Field
              label="Aliases (comma-separated)"
              value={(draft.aliases ?? []).join(", ")}
              onChange={(v) =>
                setDraft({ ...draft, aliases: v.split(",").map((a) => a.trim()).filter(Boolean) })
              }
            />
          </div>
          <div className="flex justify-end gap-2">
            <Button variant="ghost" onClick={() => setDraft(null)} disabled={saving}>
              Cancel
            </Button>
            <Button onClick={commit} loading={saving}>
              Save
            </Button>
          </div>
        </Card>
      )}

      {members.length === 0 && !draft ? (
        <EmptyState icon={Users} title="No team members" description={`Add people for ${APP_NAME} to keep an eye on.`} />
      ) : (
        <div className="grid gap-2">
          {members.map((m, idx) => (
            <Card key={`${m.name}-${idx}`} className="flex items-center gap-3 p-3">
              <div className="flex-1">
                <div className="text-sm font-medium">{m.name}</div>
                <div className="text-xs text-muted-foreground">
                  {[m.role, m.department, m.email].filter(Boolean).join(" · ") || "—"}
                </div>
              </div>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => {
                  setDraft({ ...m });
                  setEditIndex(idx);
                }}
              >
                Edit
              </Button>
              <Button variant="ghost" size="icon" aria-label={`Remove ${m.name}`} onClick={() => void remove(idx)}>
                <Trash2 className="h-4 w-4 text-destructive" />
              </Button>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}
