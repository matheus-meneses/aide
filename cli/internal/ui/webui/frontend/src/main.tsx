import React from "react";
import ReactDOM from "react-dom/client";
import App from "./App";
import { ErrorBoundary } from "@/components/ErrorBoundary";
import { ToastProvider } from "@/components/ui";
import { TrayPanel } from "@/components/TrayPanel";
import { isTrayPanel } from "@/lib/platform";
import "./index.css";

const rootEl = document.getElementById("root");
if (!rootEl) throw new Error("Root element not found");

if (isTrayPanel) {
  document.documentElement.classList.add("tray-panel");
}

ReactDOM.createRoot(rootEl).render(
  <React.StrictMode>
    <ErrorBoundary>
      {isTrayPanel ? (
        <TrayPanel />
      ) : (
        <ToastProvider>
          <App />
        </ToastProvider>
      )}
    </ErrorBoundary>
  </React.StrictMode>,
);
