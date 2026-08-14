import { useState } from 'react'
import styles from './App.module.css'
import { Sidebar } from './components/Sidebar/Sidebar'
import { QueryEditor } from './components/QueryEditor/QueryEditor'
import { Results } from './components/Results/Results'
import { Pagination } from './components/Pagination/Pagination'
import { IndexPanel } from './components/IndexPanel/IndexPanel'
import { SchemaPanel } from './components/SchemaPanel/SchemaPanel'
import { HistoryPanel } from './components/HistoryPanel/HistoryPanel'

type Tab = 'documents' | 'schema' | 'indexes' | 'query' | 'history' | 'server'

const TABS: { id: Tab; label: string }[] = [
  { id: 'documents', label: 'Documentos' },
  { id: 'schema', label: 'Schema' },
  { id: 'indexes', label: 'Índices' },
  { id: 'query', label: 'Query' },
  { id: 'history', label: 'Histórico' },
  { id: 'server', label: 'Servidor' },
]

export default function App() {
  const [tab, setTab] = useState<Tab>('documents')
  const [selection, setSelection] = useState<{ db: string; coll: string } | null>(null)

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
        <Sidebar selection={selection} onSelect={(db, coll) => setSelection({ db, coll })} />

        <div className={styles.main}>
          {tab === 'documents' && selection && (
            <>
              <QueryEditor db={selection.db} coll={selection.coll} />
              <Results db={selection.db} coll={selection.coll} />
              <Pagination db={selection.db} coll={selection.coll} />
            </>
          )}
          {tab === 'schema' && selection && <SchemaPanel db={selection.db} coll={selection.coll} />}
          {tab === 'indexes' && selection && <IndexPanel db={selection.db} coll={selection.coll} />}
          {tab === 'query' && selection && <QueryEditor db={selection.db} coll={selection.coll} standalone />}
          {tab === 'history' && <HistoryPanel />}
          {tab === 'server' && <div style={{ padding: 16 }}>Informações do servidor</div>}
          {!selection && tab !== 'history' && tab !== 'server' && (
            <div style={{ padding: 16, color: 'var(--color-text-muted)' }}>
              Selecione uma coleção na barra lateral.
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
