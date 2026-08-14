import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useQueryClient } from '@tanstack/react-query'
import styles from './App.module.css'
import { Sidebar } from './components/Sidebar/Sidebar'
import { QueryEditor } from './components/QueryEditor/QueryEditor'
import { Results } from './components/Results/Results'
import { Pagination } from './components/Pagination/Pagination'
import { IndexPanel } from './components/IndexPanel/IndexPanel'
import { SchemaPanel } from './components/SchemaPanel/SchemaPanel'
import { HistoryPanel } from './components/HistoryPanel/HistoryPanel'
import { ServerPanel } from './components/ServerPanel/ServerPanel'
import { DocumentEditor, type EditorTarget } from './components/DocumentEditor/DocumentEditor'
import { ConnectionModal } from './components/ConnectionModal/ConnectionModal'
import { useDocuments } from './hooks/useDocuments'
import { useTheme } from './hooks/useTheme'
import { useSessions } from './hooks/useSessions'
import { useInfo } from './hooks/useInfo'
import { docId } from './api/extjson'
import type { ExtJSONDocument, FindQuery, HistoryEntry, Preset } from './types'

const THEME_ICON = { light: '☀', dark: '☾', system: '◐' } as const

type Tab = 'documents' | 'schema' | 'indexes' | 'history' | 'server'

const TAB_IDS: Tab[] = ['documents', 'schema', 'indexes', 'history', 'server']

const DEFAULT_QUERY: FindQuery = { filter: '{}', sort: '', skip: 0, limit: 50 }

export default function App() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const { theme, cycle } = useTheme()
  const { data: sessions, isLoading: sessionsLoading } = useSessions()
  const { data: info } = useInfo()

  const [tab, setTab] = useState<Tab>('documents')
  const [selection, setSelection] = useState<{ db: string; coll: string } | null>(null)
  const [query, setQuery] = useState<FindQuery>(DEFAULT_QUERY)
  const [editorTarget, setEditorTarget] = useState<EditorTarget | null>(null)
  const [replayNonce, setReplayNonce] = useState(0)
  const [aggregateResult, setAggregateResult] = useState<ExtJSONDocument[] | null>(null)
  const [preset, setPreset] = useState<Preset | null>(null)

  const { data } = useDocuments(selection?.db ?? null, selection?.coll ?? null, query)

  function selectCollection(db: string, coll: string) {
    setSelection({ db, coll })
    setQuery(DEFAULT_QUERY)
    setAggregateResult(null)
    setPreset(null)
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
    setSelection({ db: entry.database, coll: entry.collection })
    setQuery({ ...DEFAULT_QUERY, filter: entry.filter, skip: 0 })
    setAggregateResult(null)
    setPreset(null)
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
      <ConnectionModal onConnected={() => void queryClient.invalidateQueries({ queryKey: ['sessions'] })} />
    )
  }

  return (
    <div className={styles.app}>
      {info?.readonly && <div className={styles.readonlyBanner}>{t('app.readonlyBanner')}</div>}

      <div className={styles.tabbar}>
        {TAB_IDS.map((id) => (
          <button
            key={id}
            className={tab === id ? styles.tabActive : styles.tab}
            onClick={() => setTab(id)}
          >
            {t(`app.tabs.${id}`)}
          </button>
        ))}
        <div className={styles.spacer} />
        <button
          className={styles.tab}
          onClick={cycle}
          title={t('app.themeTitle', { theme })}
          aria-label={t('app.themeToggle')}
        >
          {THEME_ICON[theme]}
        </button>
      </div>

      <div className={styles.layout}>
        <Sidebar selection={selection} onSelect={selectCollection} onCollectionRenamed={collectionRenamed} />

        <div className={styles.main}>
          {!selection && tab !== 'history' && tab !== 'server' && (
            <div style={{ padding: 16, color: 'var(--color-text-muted)' }}>
              {t('app.selectCollectionHint')}
            </div>
          )}

          {selection && tab === 'documents' && (
            <>
              <QueryEditor
                key={`${selection.db}:${selection.coll}:${replayNonce}`}
                db={selection.db}
                coll={selection.coll}
                query={query}
                onRun={runQuery}
                onNewDocument={() => setEditorTarget({ mode: 'new' })}
                onAggregateResult={setAggregateResult}
                preset={preset}
              />
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
          {tab === 'history' && <HistoryPanel onReplay={replayHistory} />}
          {tab === 'server' && <ServerPanel />}
        </div>
      </div>

      {selection && editorTarget && (
        <DocumentEditor
          db={selection.db}
          coll={selection.coll}
          target={editorTarget}
          onClose={() => setEditorTarget(null)}
        />
      )}
    </div>
  )
}
