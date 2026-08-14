// Types mirroring the JSON shapes returned by pkg/api handlers. Document
// bodies are Extended JSON and therefore handled as raw strings/unknown at
// this layer — see api/extjson.ts for parsing.

export interface DatabaseInfo {
  name: string
  sizeBytes: number
}

export interface CollectionInfo {
  name: string
  type: string
}

export interface CollectionStats {
  count: number
  sizeBytes: number
  storageBytes: number
  avgObjSize: number
  indexCount: number
}

export interface IndexInfo {
  name: string
  keys: Record<string, number>
  unique: boolean
}

export interface FieldType {
  type: string
  count: number
}

export interface SchemaField {
  path: string
  types: FieldType[]
}

export interface SessionInfo {
  id: string
  uri: string
  name: string
}

// ExtJSONDocument is a MongoDB document represented as parsed Extended
// JSON: values are either primitives or wrapper objects like
// {"$oid": "..."} / {"$numberLong": "..."}. Rendered opaquely by the UI's
// JSON view and edited as raw text, never re-interpreted as plain JSON.
export type ExtJSONDocument = Record<string, unknown>

export interface FindResult {
  documents: ExtJSONDocument[]
  total: number
  totalIsEstimate: boolean
}

export interface HistoryEntry {
  database: string
  collection: string
  filter: string
  at: string
}

export interface FindQuery {
  filter?: string
  projection?: string
  sort?: string
  skip?: number
  limit?: number
}
