import { cn } from "@/lib/cn";

type Tone = "default" | "muted" | "success" | "warning" | "destructive" | "info";

const tones: Record<Tone, string> = {
  default: "bg-primary/10 text-primary",
  muted: "bg-muted text-muted-foreground",
  success: "bg-success/15 text-success",
  warning: "bg-warning/15 text-warning",
  destructive: "bg-destructive/15 text-destructive",
  info: "bg-info/15 text-info",
};

export function Badge({
  tone = "default",
  className,
  ...props
}: React.HTMLAttributes<HTMLSpanElement> & { tone?: Tone }) {
  return (
    <span
      className={cn(
        "inline-flex items-center rounded-full px-2 py-0.5 text-[11px] font-medium",
        tones[tone],
        className,
      )}
      {...props}
    />
  );
}
