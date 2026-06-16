import { type ReactNode, useEffect, useMemo, useRef, useState } from "react";
import { useChatStream } from "@/hooks/useChatStream";
import { type AgentEvent, describeEvent } from "@/hooks/useSSE";
import { ChatContext, type ChatContextValue } from "@/hooks/useChat";

function pendingEventKey(event: AgentEvent): string {
  if (event.id) return String(event.id);
  try {
    const parsed = JSON.parse(event.data) as Record<string, unknown>;
    if (typeof parsed.fingerprint === "string") return parsed.fingerprint;
  } catch {
    // fall through to type/timestamp key
  }
  return `${event.type}-${event.timestamp}`;
}

interface Props {
  children: ReactNode;
  pendingEvent?: AgentEvent | null;
  onEventConsumed?: () => void;
  registerChatMessage?: (cb: (event: AgentEvent) => void) => void;
}

export function ChatProvider({ children, pendingEvent, onEventConsumed, registerChatMessage }: Props) {
  const {
    messages,
    send,
    isStreaming,
    cancel,
    injectMessage,
    updateMessage,
    clearMessages,
    appendAssistantFromSSE,
    retry,
  } = useChatStream("/api/chat");
  const [input, setInput] = useState("");
  const messagesRef = useRef(messages);
  const processedEventsRef = useRef<Set<string>>(new Set());

  useEffect(() => {
    messagesRef.current = messages;
  }, [messages]);

  useEffect(() => {
    if (registerChatMessage) {
      registerChatMessage((event) => {
        if (event.data) appendAssistantFromSSE(event.data);
      });
    }
  }, [registerChatMessage, appendAssistantFromSSE]);

  useEffect(() => {
    if (!pendingEvent) return;

    const key = pendingEventKey(pendingEvent);
    const { title, body } = describeEvent(pendingEvent);
    const content = title ? `**${title}**\n\n${body}` : body;

    if (content.trim() === "") {
      onEventConsumed?.();
      return;
    }

    const matchText = (body || content).trim();
    const existsInChat =
      matchText.length > 0 &&
      messagesRef.current.some((m) => m.role === "assistant" && m.content.includes(matchText));

    if (existsInChat || processedEventsRef.current.has(key)) {
      onEventConsumed?.();
      return;
    }

    processedEventsRef.current.add(key);
    injectMessage({
      id: `event-${key}`,
      role: "assistant",
      content,
      timestamp: new Date(pendingEvent.timestamp).getTime() || Date.now(),
    });
    onEventConsumed?.();
  }, [pendingEvent, injectMessage, onEventConsumed]);

  const value = useMemo<ChatContextValue>(
    () => ({
      messages,
      isStreaming,
      input,
      setInput,
      send,
      cancel,
      retry,
      injectMessage,
      updateMessage,
      clearMessages,
    }),
    [
      messages,
      isStreaming,
      input,
      send,
      cancel,
      retry,
      injectMessage,
      updateMessage,
      clearMessages,
    ],
  );

  return <ChatContext.Provider value={value}>{children}</ChatContext.Provider>;
}
