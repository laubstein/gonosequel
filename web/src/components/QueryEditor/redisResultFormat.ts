// formatRedisResult renders a single command's result (already decoded
// from JSON — see pkg/api/handlers_command.go's commandResult and
// pkg/redis/command.go's normalizeCommandResult) as redis-cli-style plain
// text: (nil), (integer) N, quoted bulk strings except a small set of
// known bare status replies (OK, PONG), and numbered — recursively, for
// nested arrays like SCAN's [cursor, [keys...]] — lists for array replies.
//
// This is an approximation, not a byte-for-byte match: go-redis's generic
// Do() doesn't preserve the RESP wire distinction between a simple string
// and a bulk string (both decode to a Go string), which is exactly the
// distinction real redis-cli uses to decide whether to print a value bare
// or quoted. Known status strings are special-cased instead.
const BARE_STATUS_REPLIES = new Set(['OK', 'PONG'])

export function formatRedisResult(result: unknown): string {
  return formatValue(result)
}

function formatValue(value: unknown): string {
  if (value === null || value === undefined) return '(nil)'
  if (typeof value === 'number') return `(integer) ${value}`
  if (typeof value === 'boolean') return value ? '(integer) 1' : '(integer) 0'
  if (typeof value === 'string') {
    return BARE_STATUS_REPLIES.has(value) ? value : JSON.stringify(value)
  }
  if (Array.isArray(value)) return formatList(value)
  if (typeof value === 'object') {
    // A map-shaped reply (e.g. HGETALL, normalized server-side from
    // RESP3's map type) — flatten back to alternating field/value bulk
    // strings, matching how classic (RESP2) redis-cli prints it.
    const flat: unknown[] = []
    for (const [k, v] of Object.entries(value as Record<string, unknown>)) {
      flat.push(k, v)
    }
    return formatList(flat)
  }
  return String(value)
}

// formatList numbers each element ("1) ", "2) ", ...); a nested
// array/object's own lines are indented to align under its parent
// marker — recursively, so arbitrarily nested replies (SCAN's
// [cursor, [keys...]] shape, at minimum) stay readable instead of only
// the nested block's first line lining up correctly.
function formatList(items: unknown[]): string {
  if (items.length === 0) return '(empty array)'
  return items
    .map((item, i) => {
      const marker = `${i + 1}) `
      const lines = formatValue(item).split('\n')
      const continuationIndent = ' '.repeat(marker.length)
      return [marker + lines[0], ...lines.slice(1).map((line) => continuationIndent + line)].join('\n')
    })
    .join('\n')
}
