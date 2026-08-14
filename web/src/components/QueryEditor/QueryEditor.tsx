import { useState } from 'react'
import styles from './QueryEditor.module.css'
import type { FindQuery } from '../../types'
import { exportURL } from '../../api/http'

interface Props {
  db: string
  coll: string
  query: FindQuery
  onRun: (filter: string, sort: string) => void
  standalone?: boolean
}

// A plain textarea today; step 8 of the build plan swaps this for
// CodeMirror with schema-driven autocomplete without changing the
// filter/sort contract this component exposes to App.
export function QueryEditor({ db, coll, query, onRun }: Props) {
  const [filterText, setFilterText] = useState(query.filter ?? '{}')
  const [sortText, setSortText] = useState(query.sort ?? '')
  const [error, setError] = useState<string | null>(null)

  function run() {
    try {
      if (filterText.trim()) JSON.parse(filterText)
      if (sortText.trim()) JSON.parse(sortText)
      setError(null)
      onRun(filterText.trim() || '{}', sortText.trim())
    } catch (e) {
      setError(e instanceof Error ? e.message : 'JSON inválido')
    }
  }

  return (
    <div className={styles.editor}>
      <textarea
        className={styles.textarea}
        value={filterText}
        onChange={(e) => setFilterText(e.target.value)}
        placeholder='{ "status": "ativo" }'
        onKeyDown={(e) => {
          if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') run()
        }}
      />
      <div className={styles.row}>
        <input
          className={styles.textarea}
          style={{ minHeight: 'unset', flex: 1 }}
          value={sortText}
          onChange={(e) => setSortText(e.target.value)}
          placeholder='sort: { "campo": 1 }'
        />
      </div>
      <div className={styles.row}>
        <button className={styles.button} onClick={run}>
          Executar
        </button>
        {error && <span className={styles.error}>{error}</span>}
        <div className={styles.spacer} />
        <a className={styles.exportLink} href={exportURL(db, coll, 'json', { filter: filterText })} download>
          Exportar JSON
        </a>
        <a className={styles.exportLink} href={exportURL(db, coll, 'csv', { filter: filterText })} download>
          Exportar CSV
        </a>
      </div>
    </div>
  )
}
