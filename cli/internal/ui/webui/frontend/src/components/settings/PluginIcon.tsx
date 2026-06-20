import { Package, type LucideIcon } from "lucide-react";

export function PluginIcon({
  icon,
  fallback: Fallback = Package,
}: {
  icon?: string;
  fallback?: LucideIcon;
}) {
  const wrapper =
    "flex h-9 w-9 shrink-0 items-center justify-center overflow-hidden rounded-lg bg-accent text-accent-foreground";
  if (icon && /^(https?:\/\/|data:image\/)/.test(icon)) {
    return (
      <div className={wrapper}>
        <img src={icon} alt="" className="h-5 w-5 object-contain" />
      </div>
    );
  }
  if (icon) {
    return (
      <div className={wrapper}>
        <span className="text-lg leading-none">{icon}</span>
      </div>
    );
  }
  return (
    <div className={wrapper}>
      <Fallback className="h-5 w-5" />
    </div>
  );
}
