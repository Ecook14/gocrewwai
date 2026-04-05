import { useEffect, useState } from 'react';

export interface TelemetryEvent {
  id: string;
  type: string;
  timestamp: string;
  agent_role?: string;
  payload: Record<string, any>;
}

export function useSSE(sessionId: string | null) {
  const [lastEvent, setLastEvent] = useState<TelemetryEvent | null>(null);
  const [status, setStatus] = useState<'connected' | 'disconnected' | 'connecting'>('disconnected');

  useEffect(() => {
    if (!sessionId) return;

    setStatus('connecting');
    const eventSource = new EventSource(`/api/v1/stream/${sessionId}`);

    eventSource.onopen = () => {
      console.log('🚀 SSE Connected to Crew-GO Engine');
      setStatus('connected');
    };

    eventSource.onmessage = (event) => {
      try {
        const parsed: TelemetryEvent = JSON.parse(event.data);
        setLastEvent(parsed);
      } catch (err) {
        console.error('❌ Failed to parse SSE event:', err);
      }
    };

    eventSource.onerror = (err) => {
      console.error('❌ SSE Connection Error:', err);
      setStatus('disconnected');
      eventSource.close();
    };

    return () => {
      eventSource.close();
      setStatus('disconnected');
    };
  }, [sessionId]);

  return { lastEvent, status };
}
