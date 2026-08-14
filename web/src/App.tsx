import { useState } from 'react'
import styles from './App.module.css'
import { Sidebar } from './components/Sidebar/Sidebar'
import { QueryEditor } from './components/QueryEditor/QueryEditor'
import { Results } from './components/Results/Results'
import { Pagination } from './components/Pagination/Pagination'
import { IndexPanel } from './components/IndexPanel/IndexPanel'
import { SchemaPanel } from './components/SchemaPanel/SchemaPanel'
import { HistoryPanel } from './components/HistoryPanel/HistoryPanel'
import { DocumentEditor, type EditorTarget } from './components/DocumentEditor/DocumentEditor'
import { useDocuments } from './hooks/useDocuments'
import { docId } from './api/extjson'
import type { ExtJSONDocument, FindQuery, HistoryEntry } from './types'

type Tab = 'documents' | 'schema' | 'indexes' | 'history' | 'server'

const TABS: { id: Tab; label: string }[] = [
  { id: 'documents', label: 'Documentos' },
  { id: 'schema', label: 'Schema' },
  { id: 'indexes', label: 'Índices' },
  { id: 'history', label: 'Histórico' },
  { id: 'server', label: 'Servidor' },
]

const DEFAULT_QUERY: FindQuery = { filter: '{}', sort: '', skip: 0, limit: 50 }

export default function App() {
  const [tab, setTab] = useState<Tab>('documents')
  const [selection, setSelection] = useState<{ db: string; coll: string } | null>(null)
  const [query, setQuery] = useState<FindQuery>(DEFAULT_QUERY)
  const [editorTarget, setEditorTarget] = useState<EditorTarget | null>(null)
  const [replayNonce, setReplayNonce] = useState(0)

  const { data } = useDocuments(selection?.db ?? null, selection?.coll ?? null, query)

  function selectCollection(db: string, coll: string) {
    setSelection({ db, coll })
    setQuery(DEFAULT_QUERY)
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
    setReplayNonce((n) => n + 1)
    setTab('documents')
  }

  return (
    <div className={styles.app}>
      <div className={styles.tabbar}>
        {TABS.map((t) => (
          <button
            key={t.id}
            className={tab === t.id ? styles.tabActive : styles.tab}
            onClick={() => setTab(t.id)}
          >
            {t.label}
          </button>
        ))}
        <div className={styles.spacer} />
      </div>

      <div className={styles.layout}>
        <Sidebar selection={selection} onSelect={selectCollection} />

        <div className={styles.main}>
          {!selection && tab !== 'history' && (
            <div style={{ padding: 16, color: 'var(--color-text-muted)' }}>
              Selecione uma coleção na barra lateral.
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
              />
              <Results db={selection.db} coll={selection.coll} query={query} onOpenDocument={openDocument} />
              <Pagination
                query={query}
                total={data?.total ?? 0}
                totalIsEstimate={data?.totalIsEstimate ?? false}
                onChange={paginate}
              />
            </>
          )}
          {selection && tab === 'schema' && <SchemaPanel db={selection.db} coll={selection.coll} />}
          {selection && tab === 'indexes' && <IndexPanel db={selection.db} coll={selection.coll} />}
          {tab === 'history' && <HistoryPanel onReplay={replayHistory} />}
          {selection && tab === 'server' && <div style={{ padding: 16 }}>Informações do servidor</div>}
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
