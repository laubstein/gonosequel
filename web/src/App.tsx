import { useEffect, useRef, useState } from 'react'
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
import { ConnectionLost } from './components/ConnectionLost/ConnectionLost'
import { useDocuments } from './hooks/useDocuments'
import { useCollectionStats } from './hooks/useCollectionStats'
import { SIZE_GUARD_THRESHOLD_BYTES } from './api/sizeGuard'
import { useTheme } from './hooks/useTheme'
import { useSessions } from './hooks/useSessions'
import { useInfo } from './hooks/useInfo'
import { useConnectionInfo } from './hooks/useConnectionInfo'
import { docId } from './api/extjson'
import { api } from './api/client'
import { getSessionId, setSessionId } from './api/http'
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
  const infoQuery = useInfo()
  const { data: info } = infoQuery
  const { data: connection } = useConnectionInfo()
  const isKeyValueDriver = connection?.driver === 'redis' || connection?.driver === 'valkey'

  // Set when the user clicks "Disconnect" on the connection-lost
  // placeholder below: the server is unreachable, so there's no request
  // to make (unlike the header's own Disconnect button, which calls
  // api.disconnect()) — this just clears the local session and drops back
  // to the connect screen, same as the sessions-gate below does after a
  // normal disconnect.
  const [forcedDisconnect, setForcedDisconnect] = useState(false)

  function disconnectAfterConnectionLost() {
    setSessionId(null)
    setForcedDisconnect(true)
  }

  // The session ID persisted by api/http.ts survives a refresh, but the
  // session it names might not (server restarted, or disconnected from
  // another tab) — once the real session list is in, drop back to the
  // connect screen instead of leaving every session-scoped request 400ing
  // against a session ID that no longer exists anywhere.
  useEffect(() => {
    if (sessionsLoading || !sessions) return
    const current = getSessionId()
    if (current && !sessions.some((s) => s.id === current)) {
      setSessionId(null)
      setForcedDisconnect(true)
    }
  }, [sessions, sessionsLoading])

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

  // Size guard: a collection with very large documents (tens of MB each)
  // could turn "list the first page" into a multi-hundred-MB request with
  // no warning, so the documents query doesn't fire until either the
  // estimate is safe or the user confirms — see Results.tsx for the UI.
  // Lives here, not in Results.tsx, because this same "should we fetch
  // right now" decision also has to gate the useDocuments call below,
  // whose only purpose is feeding Pagination's total count.
  const stats = useCollectionStats(selection?.db ?? null, selection?.coll ?? null)
  const docLimit = query.limit ?? 50
  const avgObjSize = stats.data?.avgObjSize ?? 0
  const isDangerous = !aggregateResult && avgObjSize > 0 && avgObjSize * docLimit > SIZE_GUARD_THRESHOLD_BYTES
  const sizeGuardKey = `${selection?.db}:${selection?.coll}:${docLimit}`
  const [confirmedSizeGuardKey, setConfirmedSizeGuardKey] = useState<string | null>(null)
  const sizeGuardConfirmed = confirmedSizeGuardKey === sizeGuardKey
  const statsSettled = stats.isSuccess || stats.isError
  const documentsEnabled = statsSettled && (!isDangerous || sizeGuardConfirmed)

  const { data } = useDocuments(selection?.db ?? null, selection?.coll ?? null, query, documentsEnabled)

  // If the active tab loses its capability (e.g. switching to a
  // lower-capability connection), fall back to Documents rather than
  // leaving the content area stuck on a tab whose button just disappeared.
  useEffect(() => {
    if (!visibleTabs.includes(tab)) setTab('documents')
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [visibleTabs.join(',')])

  // Browser back/forward navigation: every database/collection/tab pick
  // pushes a history entry carrying that selection, and a popstate listener
  // restores it when the user navigates with the browser's own Back/Forward
  // buttons (or a bookmarked/shared deep link on first load). `poppingRef`
  // suppresses the push that would otherwise happen while a popstate
  // handler itself is updating state — without it, hitting Back would
  // immediately push a fresh entry forward again and the button would feel
  // like it does nothing.
  const poppingRef = useRef(false)

  function historyUrl(next: { db: string | null; coll: string | null; tab: Tab }): string {
    const params = new URLSearchParams()
    if (next.db) params.set('db', next.db)
    if (next.coll) params.set('coll', next.coll)
    if (next.tab !== 'documents') params.set('tab', next.tab)
    const search = params.toString()
    return window.location.pathname + (search ? `?${search}` : '')
  }

  function pushHistory(next: { db: string | null; coll: string | null; tab: Tab }) {
    if (poppingRef.current) return
    window.history.pushState(next, '', historyUrl(next))
  }

  useEffect(() => {
    // Normalize the current URL into a history state object on first load,
    // so the very first Back press has something concrete to return to
    // (rather than the `null` state a plain page load starts with), and
    // pick up a deep link's db/coll/tab if one was in the URL.
    const params = new URLSearchParams(window.location.search)
    const initialDb = params.get('db')
    const initialColl = params.get('coll')
    const initialTabParam = params.get('tab') as Tab | null
    const initialTab = initialTabParam && TAB_IDS.includes(initialTabParam) ? initialTabParam : 'documents'
    if (initialDb) setSelectedDb(initialDb)
    if (initialDb && initialColl) setSelection({ db: initialDb, coll: initialColl })
    setTab(initialTab)
    window.history.replaceState({ db: initialDb, coll: initialColl, tab: initialTab }, '', historyUrl({ db: initialDb, coll: initialColl, tab: initialTab }))

    function onPopState(e: PopStateEvent) {
      poppingRef.current = true
      const state = e.state as { db: string | null; coll: string | null; tab: Tab } | null
      const params = new URLSearchParams(window.location.search)
      const db = state?.db ?? params.get('db')
      const coll = state?.coll ?? params.get('coll')
      const tabParam = state?.tab ?? (params.get('tab') as Tab | null)
      const tab = tabParam && TAB_IDS.includes(tabParam) ? tabParam : 'documents'
      setSelectedDb(db)
      setSelection(db && coll ? { db, coll } : null)
      setQuery(DEFAULT_QUERY)
      setAggregateResult(null)
      setTab(tab)
      poppingRef.current = false
    }
    window.addEventListener('popstate', onPopState)
    return () => window.removeEventListener('popstate', onPopState)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  function selectTab(next: Tab) {
    setTab(next)
    pushHistory({ db: selectedDb, coll: selection?.coll ?? null, tab: next })
  }

  function selectDatabase(db: string | null) {
    setSelectedDb(db)
    setSelection(null)
    setQuery(DEFAULT_QUERY)
    setAggregateResult(null)
    pushHistory({ db, coll: null, tab })
  }

  function selectCollection(db: string, coll: string) {
    setSelectedDb(db)
    setSelection({ db, coll })
    setQuery(DEFAULT_QUERY)
    setAggregateResult(null)
    pushHistory({ db, coll, tab })
  }

  function collectionRenamed(oldName: string, newName: string) {
    setSelection((s) => (s && s.coll === oldName ? { ...s, coll: newName } : s))
  }

  function runQuery(filter: string, sort: string, projection?: string) {
    setQuery((q) => ({ ...q, filter, sort, projection: projection || undefined, skip: 0 }))
  }

  // Bridges Results' right-click context menu (where a value/field is
  // picked) to QueryEditor's own draft state (where the filter/hidden
  // fields actually live) — the two are siblings, so App.tsx is the only
  // place that can connect them. Mirrors replayNonce's same "external
  // source overwrites the draft" pattern used for history replay.
  const [externalDraftPatch, setExternalDraftPatch] = useState<{ filterText?: string; hideField?: string } | null>(
    null,
  )
  const [externalDraftNonce, setExternalDraftNonce] = useState(0)

  function filterByValue(field: string, value: unknown) {
    setExternalDraftPatch({ filterText: JSON.stringify({ [field]: value }, null, 2) })
    setExternalDraftNonce((n) => n + 1)
  }

  function hideField(field: string) {
    setExternalDraftPatch({ hideField: field })
    setExternalDraftNonce((n) => n + 1)
  }

  function excludeByValue(field: string, value: unknown) {
    setExternalDraftPatch({ filterText: JSON.stringify({ [field]: { $ne: value } }, null, 2) })
    setExternalDraftNonce((n) => n + 1)
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
    pushHistory({ db: entry.database, coll: entry.collection, tab: 'documents' })
  }

  // /api/info never touches the database (see its own doc comment) — a
  // failing fetch here means the browser can't reach the gonosequel
  // server itself, not a backend/database problem. Checked before the
  // sessions gate below: while the server is unreachable, GET
  // /api/sessions is failing too, so `sessions` never becomes the
  // zero-length array that gate looks for — this placeholder is what
  // actually surfaces the outage instead of the app silently hanging on
  // stale or empty data.
  if (infoQuery.isError && !forcedDisconnect) {
    return (
      <ConnectionLost
        onRetry={() => void infoQuery.refetch()}
        onDisconnect={disconnectAfterConnectionLost}
        retrying={infoQuery.isFetching}
      />
    )
  }

  // In --sessions mode the server starts with no active connection, so
  // GET /api/sessions comes back empty until the user connects through
  // this modal. In single-connection mode a "default" session is always
  // pre-registered at startup, so this never renders — except right after
  // disconnecting (from the Server tab, or forcedDisconnect above), which
  // drops back to zero sessions in either mode and reuses this same gate
  // to reconnect.
  if (forcedDisconnect || (!sessionsLoading && sessions && sessions.length === 0)) {
    return (
      <ConnectionModal
        onConnected={() => {
          setForcedDisconnect(false)
          void queryClient.invalidateQueries()
        }}
      />
    )
  }

  return (
    <div className={styles.app}>
      {(info?.readonly || connection?.readonly) && (
        <div className={styles.readonlyBanner}>{t('app.readonlyBanner')}</div>
      )}

      <div className={styles.tabbar}>
        {visibleTabs.map((id) => (
          <button
            key={id}
            className={tab === id ? styles.tabActive : styles.tab}
            onClick={() => selectTab(id)}
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
          title={t('app.themeTitle', { theme: t(`app.themeNames.${theme}`) })}
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
          className={styles.disconnectButton}
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
            <div className={styles.selectCollectionHint}>{t('app.selectCollectionHint')}</div>
          )}

          {selection && tab === 'documents' && (
            <>
              {isKeyValueDriver ? (
                <RedisCommandRunner db={selection.db} coll={selection.coll} />
              ) : (
                <QueryEditor
                  db={selection.db}
                  coll={selection.coll}
                  query={query}
                  replayNonce={replayNonce}
                  onRun={runQuery}
                  onNewDocument={() => setEditorTarget({ mode: 'new' })}
                  onAggregateResult={setAggregateResult}
                  externalDraftPatch={externalDraftPatch}
                  externalDraftNonce={externalDraftNonce}
                />
              )}
              <Results
                db={selection.db}
                coll={selection.coll}
                query={query}
                onOpenDocument={openDocument}
                overrideDocuments={aggregateResult}
                enabled={documentsEnabled}
                onPaginate={paginate}
                onFilterByValue={filterByValue}
                onHideField={hideField}
                onExcludeValue={excludeByValue}
                sizeGuard={
                  aggregateResult
                    ? undefined
                    : {
                        avgObjSize,
                        isDangerous,
                        confirmed: sizeGuardConfirmed,
                        settled: statsSettled,
                        onConfirm: () => setConfirmedSizeGuardKey(sizeGuardKey),
                      }
                }
              />
              {!aggregateResult && (
                <Pagination
                  query={query}
                  total={data?.total ?? 0}
                  totalIsEstimate={data?.totalIsEstimate ?? false}
                  onChange={paginate}
                  avgObjSize={avgObjSize}
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
