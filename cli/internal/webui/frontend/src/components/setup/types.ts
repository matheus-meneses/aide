export type Step = "bootstrap" | "source" | "provider" | "done";

export interface Progress {
  setupLines: string[];
  setupDone: boolean;
  setupError: string;
  installLines: string[];
  installDone: string;
  installError: string;
}

export const emptyProgress: Progress = {
  setupLines: [],
  setupDone: false,
  setupError: "",
  installLines: [],
  installDone: "",
  installError: "",
};

export function parseMessage(raw: string): string {
  let inner = raw;
  try {
    const envelope = JSON.parse(raw) as { data?: string };
    if (typeof envelope.data === "string") inner = envelope.data;
  } catch {
    return raw;
  }
  try {
    return (JSON.parse(inner) as { message?: string }).message ?? inner;
  } catch {
    return inner;
  }
}
