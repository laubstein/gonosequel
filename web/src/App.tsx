import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import styles from './App.module.css'
import { Sidebar } from './components/Sidebar/Sidebar'
import { QueryEditor } from './components/QueryEditor/QueryEditor'
import { RedisCommandRunner } from './components/QueryEditor/RedisCommandRunner'
import { Results } from './components/Results/Results'
import { Pagination } from './components/Pagination/Pagination'
import { IndexPanel } from './components/IndexPanel/IndexPanel'
import { SchemaPanel } from './components/SchemaPanel/SchemaPanel'
import { HistoryPanel } from './components/HistoryPanel/HistoryPanel'
import { ServerPanel } from './components/ServerPanel/ServerPanel'
import { ToolsPanel } from './components/ToolsPanel/ToolsPanel'
import { DocumentEditor, type EditorTarget } from './components/DocumentEditor/DocumentEditor'
import { RedisValueEditor } from './components/RedisValueEditor/RedisValueEditor'
import { ConnectionModal } from './components/ConnectionModal/ConnectionModal'
import { useDocuments } from './hooks/useDocuments'
import { useTheme } from './hooks/useTheme'
import { useSessions } from './hooks/useSessions'
import { useInfo } from './hooks/useInfo'
import { useConnectionInfo } from './hooks/useConnectionInfo'
import { docId } from './api/extjson'
import { api } from './api/client'
import { DRIVER_LABEL } from './drivers'
import type { Capability, ExtJSONDocument, FindQuery, HistoryEntry } from './types'

const THEME_ICON = { light: '☀', dark: '☾', system: '◐' } as const

type Tab = 'documents' | 'schema' | 'indexes' | 'tools' | 'history' | 'server'

const TAB_IDS: Tab[] = ['documents', 'schema', 'indexes', 'tools', 'history', 'server']

// Tabs that only make sense when the connected backend reports the
// matching capability (see pkg/driver's Cap* constants) — 'documents',
// 'history', and 'server' have no backend-specific requirement and always
// show. Hiding rather than showing-then-erroring keeps the same "you know
// upfront" philosophy as the --readonly banner.
const TAB_CAPABILITY: Partial<Record<Tab, Capability>> = {
  schema: 'schema',
  indexes: 'indexes',
  tools: 'tools',
}

const DEFAULT_QUERY: FindQuery = { filter: '{}', sort: '', skip: 0, limit: 50 }

export default function App() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const { theme, cycle } = useTheme()
  const { data: sessions, isLoading: sessionsLoading } = useSessions()
  const { data: info } = useInfo()
  const { data: connection } = useConnectionInfo()
  const isKeyValueDriver = connection?.driver === 'redis' || connection?.driver === 'valkey'

  const disconnect = useMutation({
    mutationFn: () => api.disconnect(),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: ['sessions'] }),
  })

  const visibleTabs = TAB_IDS.filter((id) => {
    const cap = TAB_CAPABILITY[id]
    return !cap || !connection || connection.capabilities.includes(cap)
  })

  const [tab, setTab] = useState<Tab>('documents')
  const [selectedDb, setSelectedDb] = useState<string | null>(null)
  const [selection, setSelection] = useState<{ db: string; coll: string } | null>(null)
  const [query, setQuery] = useState<FindQuery>(DEFAULT_QUERY)
  const [editorTarget, setEditorTarget] = useState<EditorTarget | null>(null)
  // Set while creating the very first key of a not-yet-existing Redis
  // "collection" (a collection is only a derived grouping by key prefix —
  // it can't be created empty the way a MongoDB collection can, so there's
  // no selection.coll to open the usual "+ New document" editor against
  // yet). Holds the database to create the key in; RedisValueEditor's own
  // Key field is where the user types the full key, prefix included.
  const [newKeyDb, setNewKeyDb] = useState<string | null>(null)
  const [replayNonce, setReplayNonce] = useState(0)
  const [aggregateResult, setAggregateResult] = useState<ExtJSONDocument[] | null>(null)

  const { data } = useDocuments(selection?.db ?? null, selection?.coll ?? null, query)

  // If the active tab loses its capability (e.g. switching to a
  // lower-capability connection), fall back to Documents rather than
  // leaving the content area stuck on a tab whose button just disappeared.
  useEffect(() => {
    if (!visibleTabs.includes(tab)) setTab('documents')
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [visibleTabs.join(',')])

  function selectDatabase(db: string | null) {
    setSelectedDb(db)
    setSelection(null)
    setQuery(DEFAULT_QUERY)
    setAggregateResult(null)
  }

  function selectCollection(db: string, coll: string) {
    setSelectedDb(db)
    setSelection({ db, coll })
    setQuery(DEFAULT_QUERY)
    setAggregateResult(null)
  }

  function collectionRenamed(oldName: string, newName: string) {
    setSelection((s) => (s && s.coll === oldName ? { ...s, coll: newName } : s))
  }

  function runQuery(filter: string, sort: string) {
    setQuery((q) => ({ ...q, filter, sort, skip: 0 }))
  }

  function paginate(skip: number, limit: number) {
    setQuery((q) => ({ ...q, skip, limit }))
  }

  function openDocument(doc: ExtJSONDocument) {
    setEditorTarget({ mode: 'edit', encodedId: docId(doc) })
  }

  function replayHistory(entry: HistoryEntry) {
    setSelectedDb(entry.database)
    setSelection({ db: entry.database, coll: entry.collection })
    setQuery({ ...DEFAULT_QUERY, filter: entry.filter, skip: 0 })
    setAggregateResult(null)
    setReplayNonce((n) => n + 1)
    setTab('documents')
  }

  // In --sessions mode the server starts with no active connection, so
  // GET /api/sessions comes back empty until the user connects through
  // this modal. In single-connection mode a "default" session is always
  // pre-registered at startup, so this never renders — except right after
  // disconnecting from the Server tab, which drops back to zero sessions
  // in either mode and reuses this same gate to reconnect.
  if (!sessionsLoading && sessions && sessions.length === 0) {
    return (
      <ConnectionModal onConnected={() => void queryClient.invalidateQueries()} />
    )
  }

  return (
    <div className={styles.app}>
      {info?.readonly && <div className={styles.readonlyBanner}>{t('app.readonlyBanner')}</div>}

      <div className={styles.tabbar}>
        {visibleTabs.map((id) => (
          <button
            key={id}
            className={tab === id ? styles.tabActive : styles.tab}
            onClick={() => setTab(id)}
          >
            {t(`app.tabs.${id}`)}
          </button>
        ))}
        <div className={styles.spacer} />
        {connection?.driver && (
          <span className={styles.connectionLabel} title={t('app.driverTitle')}>
            {DRIVER_LABEL[connection.driver] ?? connection.driver}
          </span>
        )}
        <button
          className={styles.tab}
          onClick={cycle}
          title={t('app.themeTitle', { theme })}
          aria-label={t('app.themeToggle')}
        >
          {THEME_ICON[theme]}
        </button>
        <a
          className={styles.tab}
          href="/doc"
          target="_blank"
          rel="noreferrer"
          title={t('app.helpTitle')}
          aria-label={t('app.helpTitle')}
        >
          ?
        </a>
        <button
          className={styles.tab}
          onClick={() => disconnect.mutate()}
          title={t('app.disconnect')}
          aria-label={t('app.disconnect')}
        >
          ⏻
        </button>
      </div>

      <div className={styles.layout}>
        <Sidebar
          selectedDb={selectedDb}
          onSelectDb={selectDatabase}
          selection={selection}
          onSelect={selectCollection}
          onCollectionRenamed={collectionRenamed}
          driver={connection?.driver}
          onNewKey={() => setNewKeyDb(selectedDb)}
        />

        <div className={styles.main}>
          {!selectedDb && tab === 'tools' && (
            <div style={{ padding: 16, color: 'var(--color-text-muted)' }}>
              {t('app.selectDatabaseHint')}
            </div>
          )}
          {!selection && tab !== 'history' && tab !== 'server' && tab !== 'tools' && (
            <div style={{ padding: 16, color: 'var(--color-text-muted)' }}>
              {t('app.selectCollectionHint')}
            </div>
          )}

          {selection && tab === 'documents' && (
            <>
              {isKeyValueDriver ? (
                <RedisCommandRunner db={selection.db} coll={selection.coll} />
              ) : (
                <QueryEditor
                  key={`${selection.db}:${selection.coll}:${replayNonce}`}
                  db={selection.db}
                  coll={selection.coll}
                  query={query}
                  onRun={runQuery}
                  onNewDocument={() => setEditorTarget({ mode: 'new' })}
                  onAggregateResult={setAggregateResult}
                />
              )}
              <Results
                db={selection.db}
                coll={selection.coll}
                query={query}
                onOpenDocument={openDocument}
                overrideDocuments={aggregateResult}
              />
              {!aggregateResult && (
                <Pagination
                  query={query}
                  total={data?.total ?? 0}
                  totalIsEstimate={data?.totalIsEstimate ?? false}
                  onChange={paginate}
                />
              )}
            </>
          )}
          {selection && tab === 'schema' && <SchemaPanel db={selection.db} coll={selection.coll} />}
          {selection && tab === 'indexes' && <IndexPanel db={selection.db} coll={selection.coll} />}
          {selectedDb && tab === 'tools' && <ToolsPanel db={selectedDb} />}
          {tab === 'history' && <HistoryPanel onReplay={replayHistory} />}
          {tab === 'server' && <ServerPanel />}
        </div>
      </div>

      {selection &&
        editorTarget &&
        (isKeyValueDriver ? (
          <RedisValueEditor
            db={selection.db}
            coll={selection.coll}
            target={editorTarget}
            onClose={() => setEditorTarget(null)}
          />
        ) : (
          <DocumentEditor
            db={selection.db}
            coll={selection.coll}
            target={editorTarget}
            onClose={() => setEditorTarget(null)}
          />
        ))}

      {newKeyDb && (
        <RedisValueEditor
          db={newKeyDb}
          coll=""
          target={{ mode: 'new' }}
          onClose={() => setNewKeyDb(null)}
        />
      )}
    </div>
  )
}
