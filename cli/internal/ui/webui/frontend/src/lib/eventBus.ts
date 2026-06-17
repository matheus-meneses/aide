// A single shared EventSource for the agent event stream. Every consumer
// subscribes through this module so the app holds exactly one connection to
// /api/events, instead of each hook opening its own (which multiplied
// connections and could drop events). The connection is opened lazily on the
// first subscription and kept alive for the app's lifetime.

type EventHandler = (data: string) => void;
type ConnectionHandler = (connected: boolean) => void;

const ENDPOINT = "/api/events";

let source: EventSource | null = null;
let connected = false;
const eventHandlers = new Map<string, Set<EventHandler>>();
const attachedTypes = new Set<string>();
const connectionHandlers = new Set<ConnectionHandler>();

function setConnected(value: boolean): void {
  connected = value;
  for (const handler of connectionHandlers) handler(value);
}

function attachType(type: string): void {
  if (!source || attachedTypes.has(type)) return;
  attachedTypes.add(type);
  source.addEventListener(type, (e: MessageEvent<string>) => {
    const set = eventHandlers.get(type);
    if (!set) return;
    for (const handler of set) handler(e.data);
  });
}

function ensureSource(): void {
  if (source) return;
  const es = new EventSource(ENDPOINT);
  source = es;
  es.onopen = () => setConnected(true);
  es.onerror = () => setConnected(false);
  for (const type of eventHandlers.keys()) attachType(type);
}

// subscribeEvent registers a handler for a named SSE event and returns an
// unsubscribe function.
export function subscribeEvent(type: string, handler: EventHandler): () => void {
  ensureSource();
  let set = eventHandlers.get(type);
  if (!set) {
    set = new Set();
    eventHandlers.set(type, set);
  }
  set.add(handler);
  attachType(type);
  return () => {
    set.delete(handler);
  };
}

// subscribeConnection reports connection state changes and immediately invokes
// the handler with the current state. Returns an unsubscribe function.
export function subscribeConnection(handler: ConnectionHandler): () => void {
  ensureSource();
  connectionHandlers.add(handler);
  handler(connected);
  return () => {
    connectionHandlers.delete(handler);
  };
}
