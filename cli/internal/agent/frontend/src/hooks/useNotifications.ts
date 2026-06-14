import { useCallback, useEffect, useRef, useState } from "react";
import { type AgentEvent } from "@/lib/eventStore";
import { fetchRuntime } from "@/lib/api";
import {
  GROUPING_WINDOW_MS,
  type NotificationState,
  currentPermission,
  flushGroupedBuffer,
  requestNotificationPermission,
  showBrowserNotification,
} from "@/lib/notifications";

export function useNotifications() {
  const groupBufferRef = useRef<AgentEvent[]>([]);
  const groupTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const nativeRef = useRef(false);
  const [notificationPermission, setNotificationPermission] =
    useState<NotificationState>("unknown");

  const enableNotifications = useCallback(async () => {
    const result = await requestNotificationPermission();
    setNotificationPermission(result);
    return result;
  }, []);

  const queueBrowserNotification = useCallback((event: AgentEvent) => {
    if (nativeRef.current) return;

    const priority = event.priority || "normal";
    if (priority === "urgent") {
      showBrowserNotification(event);
      return;
    }

    groupBufferRef.current.push(event);
    if (!groupTimerRef.current) {
      groupTimerRef.current = setTimeout(() => {
        flushGroupedBuffer(groupBufferRef.current);
        groupBufferRef.current = [];
        groupTimerRef.current = null;
      }, GROUPING_WINDOW_MS);
    }
  }, []);

  useEffect(() => {
    let cancelled = false;
    fetchRuntime()
      .then((info) => {
        if (cancelled) return;
        if (info.native_notifications) {
          nativeRef.current = true;
          setNotificationPermission("native");
        } else {
          setNotificationPermission(currentPermission());
        }
      })
      .catch(() => {
        if (!cancelled) setNotificationPermission(currentPermission());
      });
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    if (notificationPermission !== "default") return;

    const onGesture = () => {
      if (currentPermission() === "default") {
        void enableNotifications();
      }
      window.removeEventListener("pointerdown", onGesture);
      window.removeEventListener("keydown", onGesture);
    };

    window.addEventListener("pointerdown", onGesture, { once: true });
    window.addEventListener("keydown", onGesture, { once: true });
    return () => {
      window.removeEventListener("pointerdown", onGesture);
      window.removeEventListener("keydown", onGesture);
    };
  }, [notificationPermission, enableNotifications]);

  const cleanupGrouping = useCallback(() => {
    if (groupTimerRef.current) {
      clearTimeout(groupTimerRef.current);
      groupTimerRef.current = null;
    }
    groupBufferRef.current = [];
  }, []);

  return { notificationPermission, enableNotifications, queueBrowserNotification, cleanupGrouping };
}
