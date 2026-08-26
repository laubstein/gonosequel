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
  name?: string
  count: number
  sizeBytes: number
  storageBytes: number
  indexBytes: number
  avgObjSize: number
  indexCount: number
}

export interface IndexUsageStat {
  collection: string
  index: string
  ops: number
  since: string
}

export interface CurrentOp {
  opid: number
  namespace: string
  op: string
  secsRunning: number
  client: string
  description: string
}

export interface IndexKeyEntry {
  key: string
  value: number
}

export interface IndexInfo {
  name: string
  // An ordered array, not a plain object — index key order is
  // semantically significant (it decides which sorts/range queries a
  // compound index can serve), and JSON objects don't reliably preserve
  // key order the way an array does.
  keys: IndexKeyEntry[]
  unique: boolean
  sparse?: boolean
  expireAfterSeconds?: number
  partialFilterExpression?: Record<string, unknown>
}

export interface CreateIndexOptions {
  unique?: boolean
  sparse?: boolean
  expireAfterSeconds?: number
  partialFilterExpression?: Record<string, unknown>
}

export interface FieldType {
  type: string
  count: number
}

export interface SchemaField {
  path: string
  types: FieldType[]
}

// Capability values reported by the server for the current connection —
// keep in sync with the driver.Cap* constants in pkg/driver/driver.go.
export type Capability = 'find' | 'aggregate' | 'explain' | 'indexes' | 'schema' | 'tools' | 'command' | 'updateMany'

export interface SessionInfo {
  id: string
  uri: string
  name: string
  driver: string
  capabilities: Capability[]
  // readonly is per-session (see pkg/session.Info.Readonly) — independent
  // of, and possibly stricter than, AppInfo.readonly (the server-wide
  // --readonly flag). A session can opt into read-only from the connect
  // form even when the server default is read-write, but never the
  // reverse: the server forces this true whenever its own --readonly flag
  // is set, regardless of what the connect request asked for.
  readonly: boolean
}

export interface AppInfo {
  app: string
  version: string
  driver: string
  readonly: boolean
}

export interface ServerConnections {
  current: number
  available: number
}

export interface ServerOpCounters {
  insert: number
  query: number
  update: number
  delete: number
  getmore: number
  command: number
}

export interface ServerStatus {
  version: string
  host: string
  process: string
  uptimeSeconds: number
  connections: ServerConnections
  opcounters: ServerOpCounters
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

// Preset is a ready-made query offered above the query editor, generated
// from the selected collection's inferred schema (see
// components/QueryEditor/presets.ts). find presets fill the filter/sort
// fields; aggregate presets fill the pipeline editor; update presets fill
// the filter and update-document editors. The label is a translation key
// (+ params for field-specific presets) rather than literal text, so
// presets render in whichever language the UI is in.
export interface Preset {
  labelKey: string
  labelParams?: Record<string, string>
  mode: 'find' | 'aggregate' | 'update'
  filter?: string
  sort?: string
  pipeline?: string
  update?: string
}

// CommandResult is the outcome of one line of a redis-cli-like script run
// via POST /api/databases/:db/command — see pkg/api/handlers_command.go.
export interface CommandResult {
  command: string
  result?: unknown
  error?: string
}
