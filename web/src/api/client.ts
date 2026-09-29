import axios from 'axios';

const TOKEN_KEY = 'gocrewwai.builder.token';

const client = axios.create({
  baseURL: '/api/v1',
  headers: {
    'Content-Type': 'application/json',
  },
});

// Bearer tokens live in sessionStorage: they survive a reload but are gone
// when the tab closes, which bounds the lifetime of a token left in a
// shared browser. Never log the token.
export function getToken(): string | null {
  try {
    return sessionStorage.getItem(TOKEN_KEY);
  } catch {
    return null;
  }
}

export function setToken(token: string | null): void {
  try {
    if (token) {
      sessionStorage.setItem(TOKEN_KEY, token);
    } else {
      sessionStorage.removeItem(TOKEN_KEY);
    }
  } catch {
    // Storage unavailable (private mode); the token still applies to this
    // page lifetime via the in-memory fallback below.
  }
  memoryToken = token;
}

let memoryToken: string | null = null;

function currentToken(): string | null {
  return memoryToken ?? getToken();
}

client.interceptors.request.use((config) => {
  const token = currentToken();
  if (token) {
    config.headers.set('Authorization', `Bearer ${token}`);
  }
  return config;
});

export interface KickoffRequest {
  session_id: string;
  agent_role: string;
  agent_goal?: string;
  agent_backstory?: string;
  agent_model?: string;
  task_description: string;
  task_expected_output?: string;
  crew_process?: string;
}

export interface KickoffResponse {
  message: string;
  session_id: string;
}

export interface SessionState {
  session_id: string;
  status: string;
  result?: Record<string, unknown>;
  error?: string;
}

// kickoffCrew starts one crew run. Every call mints a fresh Idempotency-Key
// so a retried click cannot silently collapse into an earlier run; the
// backend still collapses genuinely concurrent duplicates to one execution.
export const kickoffCrew = async (payload: KickoffRequest): Promise<KickoffResponse> => {
  const key =
    typeof crypto !== 'undefined' && 'randomUUID' in crypto
      ? crypto.randomUUID()
      : `builder-${Date.now()}-${Math.floor(Math.random() * 1e9)}`;
  const response = await client.post<KickoffResponse>('/crews/kickoff', payload, {
    headers: { 'Idempotency-Key': key },
  });
  return response.data;
};

export const getSession = async (id: string): Promise<SessionState> => {
  const response = await client.get<SessionState>(`/sessions/${encodeURIComponent(id)}`);
  return response.data;
};

// authHeaders is for transports that cannot use the axios interceptor, such
// as the fetch-based SSE stream.
export function authHeaders(): Record<string, string> {
  const token = currentToken();
  if (!token) return {};
  return { Authorization: `Bearer ${token}` };
}

export function hasToken(): boolean {
  return currentToken() !== null && currentToken() !== '';
}

export default client;
