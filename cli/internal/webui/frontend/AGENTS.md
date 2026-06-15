# frontend (React UI)

## Purpose

Single-page React application served by the Go agent HTTP server. Provides real-time chat with the AI agent, activity feed (notifications), item browsing, and system status display.

## Tech Stack

- React 18 + TypeScript
- Vite (build tool)
- Tailwind CSS (styling)
- Lucide React (icons)
- No component library — all components are custom

## Key Components

| Component | Role |
|-----------|------|
| `App.tsx` | Root layout: sidebar + main content, SSE connection |
| `ChatPanel.tsx` | Chat interface with slash commands, streaming responses, suggestions |
| `ChatMessage.tsx` | Message bubble rendering (user/assistant/error/rich content) |
| `NotificationFeed.tsx` | Activity sidebar with dismiss/acknowledge |
| `StatusBar.tsx` | Header: connection status, source filters, dark mode, update banner |
| `ItemsView.tsx` | Browsable list of open items filtered by source |
| `renderers/*` | Rich content renderers (Markdown, Schedule, Items, Status, Stats, Memory) |

## Hooks

| Hook | Role |
|------|------|
| `useSSE.ts` | Single EventSource connection to `/api/events`; manages notifications + browser Notification API; forwards chat_message events via callback |
| `useChatStream.ts` | Chat message state, streaming POST to `/api/chat`, session hydration, retry logic |

## Important Invariants

- Single `EventSource` connection (in `useSSE`) — `useChatStream` receives chat events via callback, not a second connection.
- Browser notifications only fire when `Notification.permission === 'granted'`.
- Notification dedup uses `eventKey` (fingerprint or type+timestamp+data prefix).
- Events list capped at 50 in state.
- Dismiss tracking uses stable keys (fingerprint-based), not array indices.
- All API calls go through `lib/api.ts` which checks `resp.ok` before parsing JSON.
- `aria-live="polite"` on the chat scroll container for screen reader support.

## Pitfalls

- `EventSource` auto-reconnects on disconnect but events during the gap are lost (no Last-Event-ID replay yet).
- Notification events are persisted in `localStorage` (key `aide-notifications`, capped at 100) via `lib/eventStore.ts` and reloaded on page refresh.
- No service worker — notifications only work while tab is open.
- `fetchStatus` error sets `statusError` state; UI shows fallback text instead of infinite skeleton.
- Chat retry only resends the last user message — does not handle multi-turn context loss.

## Build

```bash
cd cli/internal/webui/frontend
npm install
npm run build   # outputs to dist/, embedded by Go at compile time
```

## Relations

- Communicates with Go agent via: `GET /api/events` (SSE), `POST /api/chat` (streaming), REST endpoints (`/api/items`, `/api/status`, `/api/today`, etc.)
- Embedded into Go binary via `//go:embed frontend/dist` in `internal/webui/embed.go`
