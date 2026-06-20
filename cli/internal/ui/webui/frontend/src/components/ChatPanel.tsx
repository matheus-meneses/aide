import { useEffect, useRef, useState } from "react";
import { ChatMessage } from "./ChatMessage";
import { ChatComposer } from "./ChatComposer";
import { isSlashCommand } from "@/lib/commands";
import { APP_NAME } from "@/lib/brand";
import { fetchWhoami } from "@/lib/api";
import { useChatScroll } from "@/hooks/useChatScroll";
import { useSlashCommands } from "@/hooks/useSlashCommands";
import { useChat } from "@/hooks/useChat";

const SUGGESTIONS = [
  "What meetings do I have today?",
  "Show me my open tasks",
  "Summarize unread notifications",
  "What should I focus on right now?",
];

interface Props {
  onConfigure?: () => void;
}

export function ChatPanel({ onConfigure }: Props) {
  const {
    messages,
    send,
    isStreaming,
    cancel,
    injectMessage,
    updateMessage,
    clearMessages,
    retry,
    input,
    setInput,
  } = useChat();
  const [userName, setUserName] = useState("");
  const inputRef = useRef<HTMLTextAreaElement>(null);

  const { scrollRef, handleScroll, markAtBottom } = useChatScroll(messages);
  const {
    showCommands,
    setShowCommands,
    selectedIdx,
    setSelectedIdx,
    filteredCommands,
    handleSlashCommand,
  } = useSlashCommands({
    input,
    injectMessage,
    updateMessage,
    clearMessages,
    markAtBottom,
  });

  useEffect(() => {
    fetchWhoami()
      .then((p) => {
        if (p.preferred_name) setUserName(p.preferred_name);
      })
      .catch((err: unknown) => {
        console.warn("failed to load identity:", err);
      });
  }, []);

  const handleSubmit = (e: React.SyntheticEvent) => {
    e.preventDefault();
    const text = input.trim();
    if (!text || isStreaming) return;
    setInput("");
    setShowCommands(false);

    if (isSlashCommand(text)) {
      void handleSlashCommand(text);
    } else {
      void send(text);
      markAtBottom();
    }
    setTimeout(() => inputRef.current?.focus(), 0);
  };

  const selectCommand = (name: string) => {
    setInput(`/${name} `);
    setShowCommands(false);
    inputRef.current?.focus();
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (showCommands && filteredCommands.length > 0) {
      if (e.key === "ArrowDown") {
        e.preventDefault();
        setSelectedIdx((i) => Math.min(i + 1, filteredCommands.length - 1));
        return;
      }
      if (e.key === "ArrowUp") {
        e.preventDefault();
        setSelectedIdx((i) => Math.max(i - 1, 0));
        return;
      }
      if (e.key === "Tab" || (e.key === "Enter" && !e.shiftKey)) {
        e.preventDefault();
        const cmd = filteredCommands[selectedIdx];
        if (cmd) {
          setInput(`/${cmd.name} `);
          setShowCommands(false);
        }
        return;
      }
      if (e.key === "Escape") {
        setShowCommands(false);
        return;
      }
    }

    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSubmit(e);
    }
  };

  const handleSuggestion = (text: string) => {
    if (isStreaming) return;
    void send(text);
    markAtBottom();
  };

  return (
    <div className="flex flex-col h-full">
      <div
        ref={scrollRef}
        onScroll={handleScroll}
        className="flex-1 overflow-y-auto scrollbar-thin py-4"
        aria-live="polite"
        aria-relevant="additions"
      >
        <div className="max-w-3xl mx-auto">
          {messages.length === 0 && (
            <div className="flex flex-col items-center justify-center h-full min-h-[60vh] text-muted-foreground">
              <div className="text-2xl font-light mb-2">{userName ? `Hi ${userName}` : APP_NAME}</div>
              <div className="text-sm mb-6">
                {userName
                  ? "How can I help you today?"
                  : "Your personal work assistant. Ask about tasks, meetings, or items."}
              </div>
              <div className="flex flex-wrap justify-center gap-2 max-w-lg">
                {SUGGESTIONS.map((s) => (
                  <button
                    key={s}
                    onClick={() => handleSuggestion(s)}
                    className="px-3 py-1.5 text-xs rounded-full border bg-card hover:bg-accent transition-colors"
                  >
                    {s}
                  </button>
                ))}
              </div>
            </div>
          )}
          {messages.map((msg, i) => (
            <ChatMessage
              key={msg.id}
              role={msg.role}
              content={msg.content}
              timestamp={msg.timestamp}
              isError={msg.isError}
              format={msg.format}
              data={msg.data}
              pending={msg.pending}
              pendingLabel={msg.pendingLabel}
              needsConfig={msg.needsConfig}
              isStreaming={isStreaming && i === messages.length - 1 && msg.role === "assistant"}
              onSuggestionClick={handleSuggestion}
              onRetry={msg.isError && !msg.needsConfig ? retry : undefined}
              onConfigure={onConfigure}
            />
          ))}
        </div>
      </div>

      <ChatComposer
        input={input}
        setInput={setInput}
        isStreaming={isStreaming}
        inputRef={inputRef}
        onSubmit={handleSubmit}
        onCancel={cancel}
        onKeyDown={handleKeyDown}
        showCommands={showCommands}
        filteredCommands={filteredCommands}
        selectedIdx={selectedIdx}
        onSelectCommand={selectCommand}
      />
    </div>
  );
}
