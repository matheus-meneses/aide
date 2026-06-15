export function LogPanel({ lines }: { lines: string[] }) {
  if (lines.length === 0) return null;
  return (
    <div className="mt-3 max-h-40 overflow-auto rounded-md border bg-muted/40 p-3 font-mono text-xs text-muted-foreground">
      {lines.map((l, i) => (
        <div key={i}>{l}</div>
      ))}
    </div>
  );
}
