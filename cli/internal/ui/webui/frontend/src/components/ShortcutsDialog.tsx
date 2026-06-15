import { Dialog } from "@/components/ui";
import { isDesktopApp } from "@/lib/platform";

interface Props {
  open: boolean;
  onClose: () => void;
}

const SHORTCUTS: { keys: string; label: string; desktopOnly?: boolean }[] = [
  { keys: "⌘ ,", label: "Toggle settings" },
  { keys: "⌘ L", label: "Toggle logs", desktopOnly: true },
  { keys: "Esc", label: "Close panel / dialog" },
  { keys: "?", label: "Show this cheat sheet" },
  { keys: "/", label: "Slash commands in chat" },
  { keys: "/help", label: "List all chat commands" },
];

export function ShortcutsDialog({ open, onClose }: Props) {
  const shortcuts = SHORTCUTS.filter((s) => isDesktopApp || !s.desktopOnly);
  return (
    <Dialog open={open} onClose={onClose} title="Keyboard shortcuts">
      <dl className="flex flex-col gap-1.5">
        {shortcuts.map((s) => (
          <div key={s.keys} className="flex items-center justify-between gap-4 text-sm">
            <dt className="text-muted-foreground">{s.label}</dt>
            <dd>
              <kbd className="rounded border bg-muted px-1.5 py-0.5 font-mono text-xs">{s.keys}</kbd>
            </dd>
          </div>
        ))}
      </dl>
    </Dialog>
  );
}
