import { useEffect, useRef, useState } from 'react';
import { authHeaders } from '../api/client';

// Backend event envelope, matching pkg/events.Event:
//   { "type": "agent.thinking", "timestamp": "...", "source": "role",
//     "session_id": "...", "payload": {...} }
// Frames arrive as SSE data lines: "data: {...}\n\n" with no event: field.
export interface StreamEvent {
  id?: string;
  type: string;
  timestamp: string;
  source?: string;
  session_id?: string;
  payload?: Record<string, unknown>;
}

export type StreamStatus = 'connected' | 'disconnected' | 'connecting' | 'unauthorized';

// Terminal crew lifecycle events. When one arrives for our session the run
// is over and the stream can close.
const TERMINAL_TYPES = new Set([
  'crew.kickoff.completed',
  'crew.kickoff.failed',
]);

// useStream opens the session event stream with fetch instead of EventSource
// because EventSource cannot send an Authorization header, and the backend
// requires a bearer token on /stream/:id. It parses SSE data frames from the
// response body and closes itself on terminal crew events or abort.
export function useStream(sessionId: string | null) {
  const [lastEvent, setLastEvent] = useState<StreamEvent | null>(null);
  const [status, setStatus] = useState<StreamStatus>('disconnected');
  const abortRef = useRef<AbortController | null>(null);

  useEffect(() => {
    if (!sessionId) return;
    setStatus('connecting');
    const abort = new AbortController();
    abortRef.current = abort;

    const run = async () => {
      let resp: Response;
      try {
        resp = await fetch(`/api/v1/stream/${encodeURIComponent(sessionId)}`, {
          headers: { Accept: 'text/event-stream', ...authHeaders() },
          signal: abort.signal,
        });
      } catch (err) {
        if (!abort.signal.aborted) setStatus('disconnected');
        return;
      }
      if (resp.status === 401) {
        setStatus('unauthorized');
        return;
      }
      if (!resp.ok || !resp.body) {
        setStatus('disconnected');
        return;
      }
      setStatus('connected');

      const reader = resp.body.getReader();
      const decoder = new TextDecoder();
      let buffer = '';
      const handleFrame = (frame: string) => {
        const lines = frame.split('\n');
        for (const line of lines) {
          const text = line.startsWith(':') ? '' : line.replace(/^data:\s?/, '');
          if (!line.startsWith('data:') || !text) continue;
          try {
            const parsed = JSON.parse(text) as StreamEvent;
            if (parsed && typeof parsed.type === 'string') {
              setLastEvent(parsed);
              if (parsed.session_id === sessionId && TERMINAL_TYPES.has(parsed.type)) {
                abort.abort();
                setStatus('disconnected');
                return;
              }
            }
          } catch {
            // Skip malformed frames; the stream continues.
          }
        }
      };

      try {
        for (;;) {
          const { done, value } = await reader.read();
          if (done) break;
          buffer += decoder.decode(value, { stream: true });
          let idx: number;
          while ((idx = buffer.indexOf('\n\n')) >= 0) {
            const frame = buffer.slice(0, idx);
            buffer = buffer.slice(idx + 2);
            handleFrame(frame);
            if (abort.signal.aborted) return;
          }
        }
      } catch {
        // Aborted or connection dropped.
      } finally {
        if (!abort.signal.aborted) setStatus('disconnected');
        try {
          reader.releaseLock();
        } catch {
          // Already released.
        }
      }
    };

    void run();
    return () => abort.abort();
  }, [sessionId]);

  return { lastEvent, status };
}

// Backwards-compatible alias for the previous EventSource-based hook name.
export const useSSE = useStream;
export type { StreamEvent as TelemetryEvent };
