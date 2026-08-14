// Thin fetch wrapper: JSON in and out, with the server's uniform
// {"error": "..."} body turned into a thrown Error carrying that message.
let sessionId: string | null = null

export function setSessionId(id: string | null): void {
  sessionId = id
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
