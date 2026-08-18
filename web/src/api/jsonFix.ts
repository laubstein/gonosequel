import JSON5 from 'json5'

// JSON5 is a strict superset of JSON: it also accepts the JS-object-literal
// syntax people naturally type by hand (unquoted keys, single-quoted
// strings, trailing commas) — content real JSON parsers reject. When `text`
// fails JSON.parse but succeeds under JSON5, it's very likely one of those
// cases rather than genuinely malformed input, so callers can offer to
// re-serialize it as strict JSON instead of making the user hand-fix
// quoting.
//
// Returns null when `text` is already valid JSON (nothing to fix), a
// string with the re-serialized strict-JSON form when it's fixable, or
// undefined when it's broken beyond what JSON5's relaxed grammar covers.
export function computeJsonFix(text: string): string | null | undefined {
  try {
    JSON.parse(text)
    return null
  } catch {
    try {
      return JSON.stringify(JSON5.parse(text), null, 2)
    } catch {
      return undefined
    }
  }
}
