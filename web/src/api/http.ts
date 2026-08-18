// Thin fetch wrapper: JSON in and out, with the server's uniform
// {"error": "..."} body turned into a thrown Error carrying that message.
//
// sessionId is persisted to localStorage (--sessions mode's connection
// screen is what ever sets it — single-connection mode never calls
// setSessionId, so every request there goes out with no header and the
// server falls back to session.DefaultID) so a browser refresh keeps using
// the same backend session instead of losing track of it: without this,
// reloading the page in --sessions mode left every session-scoped request
// (database list, query history, ...) 400ing forever, since the server has
// no "default" session to fall back to in that mode. App.tsx validates the
// restored ID against the live session list on load and drops back to the
// connect screen if the session no longer exists (server restarted, or
// disconnected from elsewhere) rather than getting stuck like that.
const SESSION_STORAGE_KEY = 'gonosequel.sessionId'

function readStoredSessionId(): string | null {
  try {
    return localStorage.getItem(SESSION_STORAGE_KEY)
  } catch {
    return null
  }
}

let sessionId: string | null = readStoredSessionId()

export function setSessionId(id: string | null): void {
  sessionId = id
  try {
    if (id) localStorage.setItem(SESSION_STORAGE_KEY, id)
    else localStorage.removeItem(SESSION_STORAGE_KEY)
  } catch {
    // Storage disabled (private browsing, quota, ...) — sessionId still
    // works for the current page load via the in-memory variable, it just
    // won't survive a refresh.
  }
}

export function getSessionId(): string | null {
  return sessionId
}

function headers(extra?: Record<string, string>): Record<string, string> {
  const h: Record<string, string> = { 'Content-Type': 'application/json', ...extra }
  if (sessionId) h['X-Session-Id'] = sessionId
  return h
}

async function handle<T>(res: Response): Promise<T> {
  const text = await res.text()
  const body: unknown = text ? JSON.parse(text) : undefined
  if (!res.ok) {
    const message =
      body && typeof body === 'object' && 'error' in body
        ? String((body as { error: unknown }).error)
        : `request failed with status ${res.status}`
    throw new Error(message)
  }
  return body as T
}

export async function apiGet<T>(path: string, params?: Record<string, string | number | undefined>): Promise<T> {
  const url = new URL(path, window.location.origin)
  if (params) {
    for (const [key, value] of Object.entries(params)) {
      if (value !== undefined) url.searchParams.set(key, String(value))
    }
  }
  const res = await fetch(url.toString().replace(window.location.origin, ''), { headers: headers() })
  return handle<T>(res)
}

export async function apiSend<T>(method: 'POST' | 'PUT' | 'DELETE', path: string, body?: unknown): Promise<T> {
  const res = await fetch(path, {
    method,
    headers: headers(),
    body: body !== undefined ? JSON.stringify(body) : undefined,
  })
  return handle<T>(res)
}

// exportURL builds the download URL for a collection export; the browser
// navigates to it directly rather than going through fetch, since the
// response is a file attachment.
export function exportURL(
  db: string,
  coll: string,
  format: 'json' | 'csv',
  query?: Record<string, string | number | undefined>,
): string {
  const url = new URL(`/api/databases/${encodeURIComponent(db)}/collections/${encodeURIComponent(coll)}/export`, window.location.origin)
  url.searchParams.set('format', format)
  if (query) {
    for (const [key, value] of Object.entries(query)) {
      if (value !== undefined) url.searchParams.set(key, String(value))
    }
  }
  return url.toString()
}
