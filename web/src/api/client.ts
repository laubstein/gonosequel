import type {
  DatabaseInfo,
  CollectionInfo,
  CollectionStats,
  IndexInfo,
  SchemaField,
  SessionInfo,
  ExtJSONDocument,
  FindResult,
  HistoryEntry,
  FindQuery,
  AppInfo,
  ServerStatus,
  IndexUsageStat,
  CurrentOp,
} from '../types'
import { apiGet, apiSend } from './http'

export const api = {
  info: () => apiGet<AppInfo>('/api/info'),

  sessions: () => apiGet<SessionInfo[]>('/api/sessions'),
  bookmarks: () => apiGet<{ name: string; uri: string }[]>('/api/bookmarks'),
  connect: (url: string, name?: string) =>
    apiSend<{ sessionId: string }>('POST', '/api/connect', { url, name }),
  connectBookmark: (bookmark: string) =>
    apiSend<{ sessionId: string }>('POST', '/api/connect', { bookmark }),
  disconnect: () => apiSend<{ ok: true }>('POST', '/api/disconnect'),
  connectionInfo: () => apiGet<SessionInfo>('/api/connection'),
  serverStatus: () => apiGet<ServerStatus>('/api/server_status'),
  history: () => apiGet<HistoryEntry[]>('/api/history'),

  listDatabases: () => apiGet<DatabaseInfo[]>('/api/databases'),
  createDatabase: (name: string, initialCollection?: string) =>
    apiSend<{ ok: true }>('POST', '/api/databases', { name, initialCollection }),
  dropDatabase: (db: string) => apiSend<{ ok: true }>('DELETE', `/api/databases/${encodeURIComponent(db)}`),

  listCollections: (db: string) =>
    apiGet<CollectionInfo[]>(`/api/databases/${encodeURIComponent(db)}/collections`),
  createCollection: (db: string, name: string, opts?: { capped?: boolean; maxSizeByte?: number; maxDocs?: number }) =>
    apiSend<{ ok: true }>('POST', `/api/databases/${encodeURIComponent(db)}/collections`, { name, ...opts }),
  dropCollection: (db: string, coll: string) =>
    apiSend<{ ok: true }>('DELETE', `/api/databases/${encodeURIComponent(db)}/collections/${encodeURIComponent(coll)}`),
  renameCollection: (db: string, coll: string, newName: string) =>
    apiSend<{ ok: true }>(
      'POST',
      `/api/databases/${encodeURIComponent(db)}/collections/${encodeURIComponent(coll)}/rename`,
      { newName },
    ),
  collectionStats: (db: string, coll: string) =>
    apiGet<CollectionStats>(`/api/databases/${encodeURIComponent(db)}/collections/${encodeURIComponent(coll)}`),
  collectionSchema: (db: string, coll: string) =>
    apiGet<SchemaField[]>(`/api/databases/${encodeURIComponent(db)}/collections/${encodeURIComponent(coll)}/schema`),

  findDocuments: (db: string, coll: string, query: FindQuery) =>
    apiGet<FindResult>(
      `/api/databases/${encodeURIComponent(db)}/collections/${encodeURIComponent(coll)}/documents`,
      query as Record<string, string | number | undefined>,
    ),
  explain: (db: string, coll: string, filter: string) =>
    apiGet<ExtJSONDocument>(
      `/api/databases/${encodeURIComponent(db)}/collections/${encodeURIComponent(coll)}/explain`,
      { filter },
    ),
  aggregate: (db: string, coll: string, pipeline: string) =>
    apiSend<{ documents: ExtJSONDocument[] }>(
      'POST',
      `/api/databases/${encodeURIComponent(db)}/collections/${encodeURIComponent(coll)}/aggregate`,
      JSON.parse(pipeline),
    ),
  getDocument: (db: string, coll: string, encodedId: string) =>
    apiGet<ExtJSONDocument>(
      `/api/databases/${encodeURIComponent(db)}/collections/${encodeURIComponent(coll)}/documents/${encodedId}`,
    ),
  insertDocument: (db: string, coll: string, doc: ExtJSONDocument) =>
    apiSend<{ id: string }>(
      'POST',
      `/api/databases/${encodeURIComponent(db)}/collections/${encodeURIComponent(coll)}/documents`,
      doc,
    ),
  replaceDocument: (db: string, coll: string, encodedId: string, doc: ExtJSONDocument) =>
    apiSend<{ ok: true }>(
      'PUT',
      `/api/databases/${encodeURIComponent(db)}/collections/${encodeURIComponent(coll)}/documents/${encodedId}`,
      doc,
    ),
  deleteDocument: (db: string, coll: string, encodedId: string) =>
    apiSend<{ ok: true }>(
      'DELETE',
      `/api/databases/${encodeURIComponent(db)}/collections/${encodeURIComponent(coll)}/documents/${encodedId}`,
    ),

  listIndexes: (db: string, coll: string) =>
    apiGet<IndexInfo[]>(`/api/databases/${encodeURIComponent(db)}/collections/${encodeURIComponent(coll)}/indexes`),
  createIndex: (db: string, coll: string, keys: Record<string, number>, unique: boolean) =>
    apiSend<{ name: string }>(
      'POST',
      `/api/databases/${encodeURIComponent(db)}/collections/${encodeURIComponent(coll)}/indexes`,
      { keys, unique },
    ),
  dropIndex: (db: string, coll: string, name: string) =>
    apiSend<{ ok: true }>(
      'DELETE',
      `/api/databases/${encodeURIComponent(db)}/collections/${encodeURIComponent(coll)}/indexes/${encodeURIComponent(name)}`,
    ),

  collectionsOverview: (db: string) =>
    apiGet<CollectionStats[]>(`/api/databases/${encodeURIComponent(db)}/tools/collections-overview`),
  indexUsage: (db: string) =>
    apiGet<IndexUsageStat[]>(`/api/databases/${encodeURIComponent(db)}/tools/index-usage`),
  currentOps: (minSecs: number) => apiGet<CurrentOp[]>('/api/tools/current-ops', { minSecs }),
}
