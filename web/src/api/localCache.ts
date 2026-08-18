// Thin JSON read/write wrapper around localStorage, shared by anything
// that needs to survive a page refresh (or a reconnect, since these keys
// aren't tied to a session ID) without a server round trip — e.g. the
// query/command editors' in-progress drafts. Failures (storage disabled,
// quota exceeded, corrupt JSON) are swallowed: callers treat a miss the
// same as "nothing saved yet" rather than crashing over a convenience
// feature.
export function readLocal<T>(key: string): T | undefined {
  try {
    const raw = localStorage.getItem(key)
    if (raw === null) return undefined
    return JSON.parse(raw) as T
  } catch {
    return undefined
  }
}

export function writeLocal<T>(key: string, value: T): void {
  try {
    localStorage.setItem(key, JSON.stringify(value))
  } catch {
    // Value still works for the current page load via the caller's own
    // in-memory cache — it just won't survive a refresh.
  }
}
