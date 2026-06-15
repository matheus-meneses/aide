import { createContext, useContext } from "react";
import type { Message } from "@/hooks/useChatStream";

export interface ChatContextValue {
  messages: Message[];
  isStreaming: boolean;
  input: string;
  setInput: (value: string) => void;
  send: (message: string) => Promise<void>;
  cancel: () => void;
  retry: () => void;
  injectMessage: (msg: Message) => void;
  updateMessage: (id: string, patch: Partial<Message>) => void;
  clearMessages: () => void;
}

export const ChatContext = createContext<ChatContextValue | null>(null);

export function useChat(): ChatContextValue {
  const ctx = useContext(ChatContext);
  if (!ctx) {
    throw new Error("useChat must be used within a ChatProvider");
  }
  return ctx;
}
