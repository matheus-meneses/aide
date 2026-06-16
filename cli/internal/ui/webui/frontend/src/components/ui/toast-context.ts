import { createContext, useContext } from "react";

export type ToastTone = "success" | "error" | "info";

export interface ToastContextValue {
  toast: (message: string, tone?: ToastTone) => void;
}

export const ToastContext = createContext<ToastContextValue | null>(null);

export function useToast(): ToastContextValue {
  const ctx = useContext(ToastContext);
  if (!ctx) throw new Error("useToast must be used within <ToastProvider>");
  return ctx;
}
