function detectDesktop(): boolean {
  if (typeof window === "undefined") return false;
  try {
    const params = new URLSearchParams(window.location.search);
    if (params.get("desktop") === "1") {
      sessionStorage.setItem("aide-desktop", "1");
    }
  } catch {
    // ignore inaccessible storage / URL
  }
  try {
    return sessionStorage.getItem("aide-desktop") === "1";
  } catch {
    return false;
  }
}

export const isDesktopApp = detectDesktop();
