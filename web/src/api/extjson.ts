// Helpers for working with the Extended JSON documents the backend sends
// and expects (see pkg/client/extjson.go on the Go side).

import type { ExtJSONDocument } from '../types'

// encodeDocId mirrors the Go server's EncodeDocID: base64url of the JSON
// text `{"_id": <value>}`. This matches the server's canonical encoding
// for the identifier types actually used as _id in practice — ObjectId,
// string, UUID, Date, Decimal128, all of which render identically in
// relaxed and canonical Extended JSON. A bare numeric _id (int32/int64)
// is the one case where relaxed and canonical diverge, and is not
// supported by this encoder; MongoDB's own default (ObjectId) and every
// other common identifier type are unaffected.
export function encodeDocId(idValue: unknown): string {
  const json = JSON.stringify({ _id: idValue })
  const bytes = new TextEncoder().encode(json)
  let binary = ''
  bytes.forEach((b) => {
    binary += String.fromCharCode(b)
  })
  return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '')
}

export function docId(doc: ExtJSONDocument): string {
  return encodeDocId(doc['_id'])
}

// summarizeValue renders a single Extended JSON value as a short display
// string for the table view (e.g. an ObjectId wrapper -> its hex string).
export function summarizeValue(value: unknown): string {
  if (value === null || value === undefined) return 'null'
  if (typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean') {
    return String(value)
  }
  if (Array.isArray(value)) return `[${value.length} items]`
  if (typeof value === 'object') {
    const obj = value as Record<string, unknown>
    const keys = Object.keys(obj)
    if (keys.length === 1 && keys[0].startsWith('$')) {
      return String(obj[keys[0]])
    }
    return `{${keys.length} fields}`
  }
  return String(value)
}

const NUMBER_WRAPPER_KEYS = new Set(['$numberInt', '$numberLong', '$numberDouble', '$numberDecimal'])

// unwrapNumberWrappers recursively replaces every numeric Extended JSON
// wrapper ({"$numberInt"|"$numberLong"|"$numberDouble"|"$numberDecimal":
// "N"}) in a canonical document with a bare JS number — used to show/
// download the document editor's canonical form (see handleGetDocument's
// doc comment on the Go side for why canonical is used at all) as
// something that looks like the plain JSON document it actually is. A
// downstream script that does `doc.cpu === 1` breaks on
// `{"cpu": {"$numberInt": "1"}}`, so this trades type-preservation for
// plain-JSON compatibility on this view: a Long or Decimal128 outside JS's
// safe integer/double precision can come back with different digits after
// a save — the same lossy behavior every other relaxed-Extended-JSON
// surface in this app (the results table, JSON exports) already has for
// those types. A $numberDouble/$numberDecimal that doesn't parse to a
// finite number (NaN/±Infinity, or a Decimal128 outside double's range) is
// left wrapped, since plain JSON has no literal for it and unwrapping
// would silently corrupt the value to `null`.
export function unwrapNumberWrappers(value: unknown): unknown {
  if (Array.isArray(value)) return value.map(unwrapNumberWrappers)
  if (value !== null && typeof value === 'object') {
    const obj = value as Record<string, unknown>
    const keys = Object.keys(obj)
    if (keys.length === 1 && NUMBER_WRAPPER_KEYS.has(keys[0])) {
      const raw = obj[keys[0]]
      if (typeof raw === 'string') {
        const n = Number(raw)
        if (Number.isFinite(n)) return n
      }
      return value
    }
    const result: Record<string, unknown> = {}
    for (const k of keys) result[k] = unwrapNumberWrappers(obj[k])
    return result
  }
  return value
}

export interface RiskyNumberField {
  path: (string | number)[]
  wrapperType: '$numberLong' | '$numberDecimal'
  displayed: number
}

// findRiskyNumberFields walks a canonical Extended JSON document (before
// unwrapNumberWrappers runs) and reports every $numberLong/$numberDecimal
// field where showing it as a bare number (as unwrapNumberWrappers does)
// is not a lossless round trip if saved back unedited:
//   - $numberDecimal always changes BSON type on save (Decimal128 ->
//     Double) — Extended JSON has no bare-number form that means
//     "Decimal128", so there is no magnitude below which this is safe.
//   - $numberLong only loses its exact value once it falls outside the
//     range a JS double can represent every integer in
//     (Number.isSafeInteger) — below that threshold, the round trip
//     through a bare JS number is exact.
// Used to warn before a save silently changes one of these, rather than
// to decide what unwrapNumberWrappers itself unwraps (it always unwraps
// all four types, matching how the document editor displays/downloads).
export function findRiskyNumberFields(value: unknown, path: (string | number)[] = []): RiskyNumberField[] {
  const out: RiskyNumberField[] = []
  walk(value, path)
  return out

  function walk(v: unknown, p: (string | number)[]) {
    if (Array.isArray(v)) {
      v.forEach((item, i) => walk(item, [...p, i]))
      return
    }
    if (v !== null && typeof v === 'object') {
      const obj = v as Record<string, unknown>
      const keys = Object.keys(obj)
      if (keys.length === 1 && (keys[0] === '$numberLong' || keys[0] === '$numberDecimal')) {
        const wrapperType = keys[0] as '$numberLong' | '$numberDecimal'
        const raw = obj[wrapperType]
        if (typeof raw === 'string') {
          const n = Number(raw)
          if (Number.isFinite(n) && (wrapperType === '$numberDecimal' || !Number.isSafeInteger(n))) {
            out.push({ path: p, wrapperType, displayed: n })
          }
        }
        return
      }
      for (const k of keys) walk(obj[k], [...p, k])
      return
    }
  }
}

// getAtPath reads the value at a findRiskyNumberFields path out of a plain
// JS value (e.g. the document parsed from the editor's current text) —
// used to check whether the user has left a risky field exactly as
// auto-unwrapped (still risky) or has since rewrapped/edited it (safe to
// save as-is).
export function getAtPath(value: unknown, path: (string | number)[]): unknown {
  let cur = value
  for (const key of path) {
    if (cur === null || typeof cur !== 'object') return undefined
    cur = (cur as Record<string | number, unknown>)[key]
  }
  return cur
}

// pathToLabel renders a findRiskyNumberFields path as a dotted/bracketed
// field reference for display in the pre-save warning, e.g. "items[2].price".
export function pathToLabel(path: (string | number)[]): string {
  return path.map((seg, i) => (typeof seg === 'number' ? `[${seg}]` : i === 0 ? seg : `.${seg}`)).join('')
}

// rawFilterValue returns the value a "Filter by value" context-menu action
// should use — the actual Extended JSON value (not summarizeValue's lossy
// display string), or undefined when the cell has no single sensible value
// to filter by (an array, or a multi-field subdocument).
export function rawFilterValue(value: unknown): unknown {
  if (value === null || typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean') {
    return value
  }
  if (Array.isArray(value)) return undefined
  if (typeof value === 'object') {
    const keys = Object.keys(value as Record<string, unknown>)
    // A single-key "$"-prefixed wrapper (ObjectId, Date, Decimal128, Long,
    // ...) — keep the whole wrapper, not summarizeValue's unwrapped inner
    // string, so the resulting filter is still valid Extended JSON the
    // backend understands (e.g. {"createdAt": {"$date": "..."}}).
    if (keys.length === 1 && keys[0].startsWith('$')) return value
    return undefined
  }
  return undefined
}
