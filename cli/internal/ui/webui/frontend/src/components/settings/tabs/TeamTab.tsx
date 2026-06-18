import { useEffect, useMemo, useState } from "react";
import { ChevronDown, ChevronRight, Plus, Trash2, UserPlus, Users } from "lucide-react";
import * as api from "@/lib/api";
import { Field } from "@/components/forms/Field";
import { Badge, Button, Card, EmptyState, Label, Select, Skeleton, useToast } from "@/components/ui";
import { APP_NAME } from "@/lib/brand";

const MANUAL = "manual";

interface Draft {
  id?: number;
  name: string;
  email: string;
  role: string;
  department: string;
  branch: string;
  registration: string;
  aliases: string;
  managerId: number | null;
}

function emptyDraft(managerId: number | null): Draft {
  return {
    name: "",
    email: "",
    role: "",
    department: "",
    branch: "",
    registration: "",
    aliases: "",
    managerId,
  };
}

function toDraft(m: api.TeamMember): Draft {
  return {
    id: m.id,
    name: m.name,
    email: m.email ?? "",
    role: m.role ?? "",
    department: m.department ?? "",
    branch: m.branch ?? "",
    registration: m.registration ?? "",
    aliases: (m.aliases ?? []).join(", "),
    managerId: m.manager_id ?? null,
  };
}

function managerLabel(m: api.TeamMember): string {
  const parts = [m.name];
  if (m.registration) parts[0] = `${m.name} (${m.registration})`;
  if (m.role) parts.push(m.role);
  return parts.join(" · ");
}

function TreeNode({
  member,
  childrenOf,
  depth,
  onEdit,
  onDelete,
  onAddReport,
}: {
  member: api.TeamMember;
  childrenOf: Map<number | null, api.TeamMember[]>;
  depth: number;
  onEdit: (m: api.TeamMember) => void;
  onDelete: (m: api.TeamMember) => void;
  onAddReport: (m: api.TeamMember) => void;
}) {
  const children = childrenOf.get(member.id) ?? [];
  const [open, setOpen] = useState(depth < 2);
  const editable = member.source === MANUAL;

  return (
    <div>
      <div
        className="group flex items-center gap-2 rounded px-2 py-1.5 hover:bg-accent/40"
        style={{ paddingLeft: `${8 + depth * 20}px` }}
      >
        <button
          type="button"
          className="w-4 shrink-0 text-muted-foreground"
          onClick={() => children.length > 0 && setOpen((o) => !o)}
          aria-label={children.length > 0 ? (open ? "Collapse" : "Expand") : undefined}
        >
          {children.length > 0 ? (
            open ? <ChevronDown className="h-3.5 w-3.5" /> : <ChevronRight className="h-3.5 w-3.5" />
          ) : null}
        </button>
        <div className="min-w-0 flex-1">
          <div className="truncate text-sm font-medium">
            {member.name}
            {member.registration && (
              <span className="ml-1.5 text-xs font-normal text-muted-foreground">({member.registration})</span>
            )}
          </div>
          <div className="truncate text-xs text-muted-foreground">
            {[member.role, member.department, member.email].filter(Boolean).join(" · ") || "—"}
          </div>
        </div>
        {member.source !== MANUAL && <Badge tone="muted">{member.source}</Badge>}
        <div className="flex items-center gap-0.5 opacity-0 transition-opacity group-hover:opacity-100">
          <Button variant="ghost" size="icon" aria-label={`Add report to ${member.name}`} onClick={() => onAddReport(member)}>
            <UserPlus className="h-4 w-4" />
          </Button>
          {editable && (
            <>
              <Button variant="ghost" size="sm" onClick={() => onEdit(member)}>
                Edit
              </Button>
              <Button variant="ghost" size="icon" aria-label={`Remove ${member.name}`} onClick={() => onDelete(member)}>
                <Trash2 className="h-4 w-4 text-destructive" />
              </Button>
            </>
          )}
        </div>
      </div>
      {open &&
        children.map((child) => (
          <TreeNode
            key={child.id}
            member={child}
            childrenOf={childrenOf}
            depth={depth + 1}
            onEdit={onEdit}
            onDelete={onDelete}
            onAddReport={onAddReport}
          />
        ))}
    </div>
  );
}

export function TeamTab() {
  const { toast } = useToast();
  const [members, setMembers] = useState<api.TeamMember[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [draft, setDraft] = useState<Draft | null>(null);

  const load = async () => {
    try {
      setMembers(await api.fetchTeam());
    } catch (e) {
      toast(String(e), "error");
    }
  };

  useEffect(() => {
    void load().finally(() => setLoading(false));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const byId = useMemo(() => new Map(members.map((m) => [m.id, m])), [members]);

  const childrenOf = useMemo(() => {
    const map = new Map<number | null, api.TeamMember[]>();
    for (const m of members) {
      const key = m.manager_id != null && byId.has(m.manager_id) ? m.manager_id : null;
      if (!map.has(key)) map.set(key, []);
      map.get(key)?.push(m);
    }
    return map;
  }, [members, byId]);

  const roots = childrenOf.get(null) ?? [];

  const managerOptions = useMemo(() => {
    if (!draft?.id) return members;
    const blocked = new Set<number>([draft.id]);
    const stack = [draft.id];
    while (stack.length) {
      const cur = stack.pop() as number;
      for (const child of childrenOf.get(cur) ?? []) {
        if (!blocked.has(child.id)) {
          blocked.add(child.id);
          stack.push(child.id);
        }
      }
    }
    return members.filter((m) => !blocked.has(m.id));
  }, [draft, members, childrenOf]);

  const submit = async () => {
    if (!draft || !draft.name.trim()) {
      toast("Name is required", "error");
      return;
    }
    setSaving(true);
    const manager = draft.managerId != null ? byId.get(draft.managerId) : undefined;
    const input: api.TeamMemberInput = {
      name: draft.name.trim(),
      email: draft.email,
      role: draft.role,
      department: draft.department,
      branch: draft.branch,
      registration: draft.registration,
      aliases: draft.aliases
        .split(",")
        .map((a) => a.trim())
        .filter(Boolean),
      manager_id: draft.managerId,
      manager_registration: manager?.registration ?? "",
    };
    try {
      if (draft.id != null) await api.updateTeamMember(draft.id, input);
      else await api.addTeamMember(input);
      await load();
      setDraft(null);
      toast("Team saved", "success");
    } catch (e) {
      toast(String(e), "error");
    } finally {
      setSaving(false);
    }
  };

  const remove = async (m: api.TeamMember) => {
    try {
      await api.deleteTeamMember(m.id);
      await load();
      toast(`Removed ${m.name}`, "success");
    } catch (e) {
      toast(String(e), "error");
    }
  };

  if (loading) return <Skeleton className="h-64 w-full" />;

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-sm font-semibold">Team</h2>
          <p className="text-xs text-muted-foreground">
            The org chart {APP_NAME} works with. Add people manually or let plugins sync them.
          </p>
        </div>
        <Button size="sm" onClick={() => setDraft(emptyDraft(null))}>
          <Plus className="h-3.5 w-3.5" /> Add member
        </Button>
      </div>

      {draft && (
        <Card className="space-y-3 p-4">
          <div className="text-sm font-medium">{draft.id != null ? "Edit member" : "New member"}</div>
          <div className="grid gap-3 sm:grid-cols-2">
            <Field label="Name" required value={draft.name} onChange={(v) => setDraft({ ...draft, name: v })} />
            <Field label="Email" value={draft.email} onChange={(v) => setDraft({ ...draft, email: v })} />
            <Field label="Role" value={draft.role} onChange={(v) => setDraft({ ...draft, role: v })} />
            <Field label="Department" value={draft.department} onChange={(v) => setDraft({ ...draft, department: v })} />
            <Field label="Branch" value={draft.branch} onChange={(v) => setDraft({ ...draft, branch: v })} />
            <Field
              label="Registration"
              value={draft.registration}
              onChange={(v) => setDraft({ ...draft, registration: v })}
            />
            <div>
              <Label>Manager</Label>
              <Select
                value={draft.managerId ?? ""}
                onChange={(e) =>
                  setDraft({ ...draft, managerId: e.target.value === "" ? null : Number(e.target.value) })
                }
              >
                <option value="">— No manager (top level) —</option>
                {managerOptions.map((m) => (
                  <option key={m.id} value={m.id}>
                    {managerLabel(m)}
                  </option>
                ))}
              </Select>
            </div>
            <Field
              label="Aliases (comma-separated)"
              value={draft.aliases}
              onChange={(v) => setDraft({ ...draft, aliases: v })}
            />
          </div>
          <div className="flex justify-end gap-2">
            <Button variant="ghost" onClick={() => setDraft(null)} disabled={saving}>
              Cancel
            </Button>
            <Button onClick={() => void submit()} loading={saving}>
              Save
            </Button>
          </div>
        </Card>
      )}

      {members.length === 0 && !draft ? (
        <EmptyState icon={Users} title="No team members" description={`Add people for ${APP_NAME} to keep an eye on.`} />
      ) : (
        <Card className="py-1">
          {roots.map((root) => (
            <TreeNode
              key={root.id}
              member={root}
              childrenOf={childrenOf}
              depth={0}
              onEdit={(m) => setDraft(toDraft(m))}
              onDelete={(m) => void remove(m)}
              onAddReport={(m) => setDraft(emptyDraft(m.id))}
            />
          ))}
        </Card>
      )}
    </div>
  );
}
