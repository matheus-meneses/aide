import { useCallback, useEffect, useRef, useState } from "react";
import { fetchNotifications } from "@/lib/api";
import { subscribeConnection, subscribeEvent } from "@/lib/eventBus";
import {
  type AgentEvent,
  MAX_EVENTS,
  eventKey,
  loadFromStorage,
  safeParse,
  saveToStorage,
} from "@/lib/eventStore";
import { shouldNotify } from "@/lib/notifications";
import { useNotifications } from "@/hooks/useNotifications";

export type { AgentEvent } from "@/lib/eventStore";
export { describeEvent } from "@/lib/notifications";

export function useSSE(url: string) {
  const [events, setEvents] = useState<AgentEvent[]>(loadFromStorage);
  const [connected, setConnected] = useState(false);
  const chatCallbackRef = useRef<((event: AgentEvent) => void) | null>(null);
  const lastEventIdRef = useRef<number>(0);
  const { notificationPermission, enableNotifications, queueBrowserNotification, cleanupGrouping } =
    useNotifications();

  const onChatMessage = useCallback((cb: (event: AgentEvent) => void) => {
    chatCallbackRef.current = cb;
  }, []);

  useEffect(() => {
    fetchNotifications(MAX_EVENTS)
      .then((data) => {
        if (data.events.length) {
          setEvents((prev) => {
            const existingKeys = new Set(prev.map(eventKey));
            const newOnes = data.events.filter((e: AgentEvent) => !existingKeys.has(eventKey(e)));
            if (newOnes.length === 0) return prev;
            const merged = [...newOnes, ...prev].slice(0, MAX_EVENTS);
            saveToStorage(merged);
            const maxId = Math.max(...merged.map((e) => e.id || 0));
            if (maxId > lastEventIdRef.current) lastEventIdRef.current = maxId;
            return merged;
          });
        }
      })
      .catch((err: unknown) => {
        console.warn("failed to load notifications:", err);
      });
  }, []);

  useEffect(() => {
    const stored = loadFromStorage();
    if (stored.length > 0) {
      const maxId = Math.max(...stored.map((e) => e.id || 0));
      if (maxId > lastEventIdRef.current) lastEventIdRef.current = maxId;
    }

    const appendEvent = (event: AgentEvent) => {
      const key = eventKey(event);
      setEvents((prev) => {
        if (prev.some((existing) => eventKey(existing) === key)) return prev;
        const updated = [event, ...prev].slice(0, MAX_EVENTS);
        saveToStorage(updated);
        if (event.id && event.id > lastEventIdRef.current) {
          lastEventIdRef.current = event.id;
        }
        return updated;
      });
    };

    const handleNotifiable = (data: string) => {
      const event = safeParse(data);
      if (!event) return;
      appendEvent(event);
      if (shouldNotify(event)) {
        queueBrowserNotification(event);
      }
    };

    const handleSilent = (data: string) => {
      const event = safeParse(data);
      if (!event) return;
      appendEvent(event);
    };

    const handleChatMessage = (data: string) => {
      const event = safeParse(data);
      if (event) {
        chatCallbackRef.current?.(event);
      }
    };

    const unsubs = [
      subscribeConnection(setConnected),
      subscribeEvent("notification", handleNotifiable),
      subscribeEvent("briefing", handleNotifiable),
      subscribeEvent("scrape_complete", handleSilent),
      subscribeEvent("cycle_error", handleSilent),
      subscribeEvent("chat_message", handleChatMessage),
    ];

    return () => {
      unsubs.forEach((u) => u());
      cleanupGrouping();
    };
  }, [url, queueBrowserNotification, cleanupGrouping]);

  const clear = useCallback(() => {
    setEvents([]);
    saveToStorage([]);
  }, []);

  const dismiss = useCallback((event: AgentEvent) => {
    const key = eventKey(event);
    setEvents((prev) => {
      const updated = prev.filter((e) => eventKey(e) !== key);
      saveToStorage(updated);
      return updated;
    });
  }, []);

  return {
    events,
    connected,
    clear,
    dismiss,
    onChatMessage,
    notificationPermission,
    enableNotifications,
  };
}
