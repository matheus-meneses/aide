import { isDesktopApp } from "./platform";

export function openExternal(url: string): void {
  if (!isDesktopApp) {
    window.open(url, "_blank", "noopener,noreferrer");
    return;
  }
  void fetch("/api/open", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ url }),
  }).catch((err: unknown) => {
    console.warn("failed to open external url:", err);
  });
}

export function handleExternalClick(
  e: React.MouseEvent<HTMLAnchorElement>,
  url: string | undefined,
): void {
  if (!isDesktopApp || !url) return;
  e.preventDefault();
  openExternal(url);
}
