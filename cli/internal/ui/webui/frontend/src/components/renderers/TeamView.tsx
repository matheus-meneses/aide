import { useState } from "react";
import { ChevronDown, ChevronRight } from "lucide-react";

interface Member {
  id: number;
  name: string;
  registration: string;
  role: string;
  department: string;
  branch: string;
  email: string;
  manager_id: number | null;
  source: string;
}

interface Props {
  data: {
    members?: Member[];
    view: string;
  };
}

function buildTree(members: Member[]): Map<number | null, Member[]> {
  const tree = new Map<number | null, Member[]>();
  for (const m of members) {
    const parentId = m.manager_id ?? null;
    if (!tree.has(parentId)) tree.set(parentId, []);
    tree.get(parentId)?.push(m);
  }
  return tree;
}

function TreeNode({
  member,
  tree,
  depth,
}: {
  member: Member;
  tree: Map<number | null, Member[]>;
  depth: number;
}) {
  const children = tree.get(member.id) ?? [];
  const [open, setOpen] = useState(depth < 2);

  return (
    <div>
      <div
        className="flex items-start gap-1.5 py-1.5 px-2 rounded hover:bg-accent/40 cursor-pointer select-none"
        style={{ paddingLeft: `${8 + depth * 20}px` }}
        onClick={() => children.length > 0 && setOpen((o) => !o)}
      >
        <span className="mt-0.5 w-4 shrink-0 text-muted-foreground">
          {children.length > 0 ? (
            open ? (
              <ChevronDown className="w-3.5 h-3.5" />
            ) : (
              <ChevronRight className="w-3.5 h-3.5" />
            )
          ) : null}
        </span>
        <div className="flex flex-col min-w-0">
          <span className="text-sm font-medium leading-tight truncate">
            {member.name}
            {member.registration && (
              <span className="ml-1.5 text-xs text-muted-foreground font-normal">
                ({member.registration})
              </span>
            )}
          </span>
          {member.role && (
            <span className="text-xs text-muted-foreground leading-tight truncate">
              {member.role}
            </span>
          )}
        </div>
        {member.source && member.source !== "config" && (
          <span className="ml-auto shrink-0 text-[10px] px-1.5 py-0.5 rounded bg-muted text-muted-foreground">
            {member.source}
          </span>
        )}
      </div>
      {open &&
        children.map((child) => (
          <TreeNode key={child.id} member={child} tree={tree} depth={depth + 1} />
        ))}
    </div>
  );
}

function FlatRow({ member, byId }: { member: Member; byId: Map<number, Member> }) {
  const manager = member.manager_id != null ? byId.get(member.manager_id) : undefined;
  return (
    <div className="flex items-center gap-2 px-3 py-1.5 text-sm border-b last:border-0 hover:bg-accent/30">
      <span className="flex-1 font-medium truncate">
        {member.name}
        {member.registration && (
          <span className="ml-1.5 text-xs text-muted-foreground font-normal">
            ({member.registration})
          </span>
        )}
      </span>
      <span className="shrink-0 text-xs text-muted-foreground w-48 truncate text-right">
        {member.role || "—"}
      </span>
      <span className="shrink-0 text-xs text-muted-foreground w-40 truncate text-right">
        {manager?.name ?? "—"}
      </span>
    </div>
  );
}

export function TeamView({ data }: Props) {
  const { members, view } = data;
  const [mode, setMode] = useState<"tree" | "flat">(view === "flat" ? "flat" : "tree");

  if (!members || members.length === 0) {
    return (
      <div className="rounded-lg border bg-card p-4 text-sm text-muted-foreground">
        No team members found.
      </div>
    );
  }

  const tree = buildTree(members);
  const byId = new Map(members.map((m) => [m.id, m]));
  const roots = tree.get(null) ?? [];

  return (
    <div className="rounded-lg border bg-card overflow-hidden">
      <div className="px-3 py-2 border-b bg-accent/30 flex items-center justify-between">
        <span className="text-xs font-medium">Team ({members.length})</span>
        <div className="flex gap-1">
          <button
            onClick={() => setMode("tree")}
            className={`text-[10px] px-2 py-0.5 rounded transition-colors ${mode === "tree" ? "bg-primary text-primary-foreground" : "bg-muted text-muted-foreground hover:text-foreground"}`}
          >
            tree
          </button>
          <button
            onClick={() => setMode("flat")}
            className={`text-[10px] px-2 py-0.5 rounded transition-colors ${mode === "flat" ? "bg-primary text-primary-foreground" : "bg-muted text-muted-foreground hover:text-foreground"}`}
          >
            flat
          </button>
        </div>
      </div>

      {mode === "tree" ? (
        <div className="py-1">
          {roots.map((root) => (
            <TreeNode key={root.id} member={root} tree={tree} depth={0} />
          ))}
          {members
            .filter((m) => m.manager_id == null && !roots.find((r) => r.id === m.id))
            .map((m) => (
              <TreeNode key={m.id} member={m} tree={tree} depth={0} />
            ))}
        </div>
      ) : (
        <div>
          <div className="flex items-center gap-2 px-3 py-1 text-[10px] uppercase tracking-wider text-muted-foreground border-b bg-accent/20">
            <span className="flex-1">Name</span>
            <span className="shrink-0 w-48 text-right">Role</span>
            <span className="shrink-0 w-40 text-right">Manager</span>
          </div>
          {members.map((m) => (
            <FlatRow key={m.id} member={m} byId={byId} />
          ))}
        </div>
      )}
    </div>
  );
}
