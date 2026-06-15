import type { CSSProperties } from "react";

const dragStyle = { "--wails-draggable": "drag" } as CSSProperties;

export function TitleBar() {
  return <div className="h-7 shrink-0 bg-background" style={dragStyle} aria-hidden="true" />;
}
