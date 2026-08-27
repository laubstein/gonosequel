import JSON5 from 'json5'

// Quotes unquoted object keys that JSON5 itself won't accept, specifically
// dotted paths: `{SO.nome: 1}` is the natural way to write a MongoDB
// nested field, but `SO.nome` is not a valid ECMAScript identifier, so
// JSON5 rejects it just as JSON does. Matches only in key position (after
// `{` or `,`, before `:`).
//
// Applied only after JSON5 has already failed, and the result is kept only
// if it then parses — so a false match inside a string literal can't make
// things worse than the failure we already had.
const BARE_KEY = /([{,]\s*)([A-Za-z_$][A-Za-z0-9_$]*(?:\.[A-Za-z0-9_$]+)+)(\s*:)/g

function quoteDottedKeys(text: string): string {
  return text.replace(BARE_KEY, '$1"$2"$3')
}

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
    // fall through to the repair attempts
  }

  try {
    return JSON.stringify(JSON5.parse(text), null, 2)
  } catch {
    // fall through: may still be a dotted-key case JSON5 can't express
  }

  const quoted = quoteDottedKeys(text)
  if (quoted === text) return undefined
  try {
    return JSON.stringify(JSON5.parse(quoted), null, 2)
  } catch {
    return undefined
  }
}
