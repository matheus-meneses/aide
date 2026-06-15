export interface SlashCommand {
  name: string;
  description: string;
  clientOnly?: boolean;
}

export const COMMANDS: SlashCommand[] = [
  { name: "memory", description: "Show agent's last memory" },
  { name: "status", description: "Current counts and health" },
  { name: "today", description: "Today's meetings" },
  { name: "items", description: "Open items (optional: /items jira)" },
  { name: "scrape", description: "Trigger scrape now (optional: /scrape outlook)" },
  { name: "health", description: "Source health status" },
  { name: "stats", description: "Token usage statistics (7-day chart)" },
  { name: "whoami", description: "Show your identity" },
  { name: "team", description: "Show org tree (optional: /team flat or /team --source rh_portal)" },
  { name: "ack", description: "Acknowledge an alert (e.g. /ack fingerprint)" },
  { name: "prune", description: "Delete old data (e.g. /prune 2 keeps 2 days)" },
  { name: "command", description: "Run any aide CLI command (e.g. /command report)" },
  { name: "clear", description: "Clear chat", clientOnly: true },
  { name: "help", description: "List available commands", clientOnly: true },
];

export interface ExecResult {
  type: "memory" | "status" | "schedule" | "items" | "scrape" | "stats" | "team" | "text";
  data?: Record<string, unknown>;
  text?: string;
}

export async function execCommand(command: string): Promise<ExecResult> {
  const resp = await fetch("/api/exec", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ command }),
  });
  if (!resp.ok) {
    const text = await resp.text().catch(() => "");
    throw new Error(text.trim() || `HTTP ${resp.status}`);
  }
  return resp.json() as Promise<ExecResult>;
}

export function generateHelpText(): string {
  const lines = COMMANDS.map((c) => `- \`/${c.name}\` — ${c.description}`);
  return `**Available Commands**\n\n${lines.join("\n")}`;
}

export function isSlashCommand(input: string): boolean {
  return input.startsWith("/");
}

export function parseCommand(input: string): { name: string; args: string } {
  const trimmed = input.slice(1).trim();
  const spaceIdx = trimmed.indexOf(" ");
  if (spaceIdx === -1) {
    return { name: trimmed, args: "" };
  }
  return { name: trimmed.slice(0, spaceIdx), args: trimmed.slice(spaceIdx + 1).trim() };
}
