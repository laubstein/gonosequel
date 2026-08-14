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
