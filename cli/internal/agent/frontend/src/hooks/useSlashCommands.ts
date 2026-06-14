import { useEffect, useState } from "react";
import { COMMANDS, execCommand, generateHelpText, parseCommand } from "@/lib/commands";
import { type Message } from "@/hooks/useChatStream";

interface Params {
  input: string;
  injectMessage: (msg: Message) => void;
  updateMessage: (id: string, patch: Partial<Message>) => void;
  clearMessages: () => void;
  markAtBottom: () => void;
}

const PENDING_LABELS: Record<string, string> = {
  scrape: "Collecting items from your sources…",
  command: "Running command…",
  prune: "Pruning old data…",
};

export function useSlashCommands({
  input,
  injectMessage,
  updateMessage,
  clearMessages,
  markAtBottom,
}: Params) {
  const [showCommands, setShowCommands] = useState(false);
  const [commandFilter, setCommandFilter] = useState("");
  const [selectedIdx, setSelectedIdx] = useState(0);

  const filteredCommands = COMMANDS.filter(
    (c) => commandFilter === "" || c.name.startsWith(commandFilter),
  );

  useEffect(() => {
    if (input === "/" || (input.startsWith("/") && !input.includes(" "))) {
      setShowCommands(true);
      setCommandFilter(input.slice(1));
      setSelectedIdx(0);
    } else {
      setShowCommands(false);
    }
  }, [input]);

  const handleSlashCommand = async (text: string) => {
    const { name, args } = parseCommand(text);
    const fullCommand = args ? `${name} ${args}` : name;

    const userMsg: Message = {
      id: `cmd-${Date.now()}`,
      role: "user",
      content: text,
      timestamp: Date.now(),
    };
    injectMessage(userMsg);
    markAtBottom();

    if (name === "clear") {
      clearMessages();
      return;
    }

    if (name === "help") {
      const helpMsg: Message = {
        id: `help-${Date.now()}`,
        role: "assistant",
        content: generateHelpText(),
        timestamp: Date.now(),
        format: "text",
      };
      injectMessage(helpMsg);
      return;
    }

    const pendingId = `exec-${Date.now()}`;
    const pendingLabel = PENDING_LABELS[name];
    if (pendingLabel) {
      injectMessage({
        id: pendingId,
        role: "assistant",
        content: "",
        timestamp: Date.now(),
        pending: true,
        pendingLabel,
      });
      markAtBottom();
    }

    try {
      const result = await execCommand(fullCommand);
      const responseMsg: Message = {
        id: pendingId,
        role: "assistant",
        content: result.text || "",
        timestamp: Date.now(),
        format: result.type,
        data: result.data,
      };
      if (pendingLabel) {
        updateMessage(pendingId, { ...responseMsg, pending: false, pendingLabel: undefined });
      } else {
        injectMessage(responseMsg);
      }
    } catch (err: unknown) {
      const message = `Command failed: ${err instanceof Error ? err.message : "network error"}`;
      if (pendingLabel) {
        updateMessage(pendingId, {
          content: message,
          timestamp: Date.now(),
          format: "text",
          isError: true,
          pending: false,
          pendingLabel: undefined,
        });
      } else {
        injectMessage({
          id: `exec-err-${Date.now()}`,
          role: "assistant",
          content: message,
          timestamp: Date.now(),
          format: "text",
          isError: true,
        });
      }
    }
  };

  return {
    showCommands,
    setShowCommands,
    commandFilter,
    selectedIdx,
    setSelectedIdx,
    filteredCommands,
    handleSlashCommand,
  };
}
