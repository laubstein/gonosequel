import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import CodeMirror from '@uiw/react-codemirror'
import { json } from '@codemirror/lang-json'
import { autocompletion } from '@codemirror/autocomplete'
import styles from './QueryEditor.module.css'
import type { FindQuery } from '../../types'
import { exportURL } from '../../api/http'
import { api } from '../../api/client'
import { useCollectionSchema } from '../../hooks/useCollectionSchema'
import { fieldCompletionSource } from './fieldCompletion'

interface Props {
  db: string
  coll: string
  query: FindQuery
  onRun: (filter: string, sort: string) => void
  onNewDocument: () => void
  standalone?: boolean
}

export function QueryEditor({ db, coll, query, onRun, onNewDocument }: Props) {
  const { t } = useTranslation()
  const [filterText, setFilterText] = useState(query.filter ?? '{}')
  const [sortText, setSortText] = useState(query.sort ?? '')
  const [error, setError] = useState<string | null>(null)
  const [explainResult, setExplainResult] = useState<string | null>(null)
  const [explaining, setExplaining] = useState(false)

  const { data: schemaFields } = useCollectionSchema(db, coll)
  const extensions = [
    json(),
    autocompletion({ override: [fieldCompletionSource(schemaFields ?? [])] }),
  ]

  function run() {
    try {
      if (filterText.trim()) JSON.parse(filterText)
      if (sortText.trim()) JSON.parse(sortText)
      setError(null)
      setExplainResult(null)
      onRun(filterText.trim() || '{}', sortText.trim())
    } catch (e) {
      setError(e instanceof Error ? e.message : t('queryEditor.invalidJson'))
    }
  }

  async function explain() {
    try {
      if (filterText.trim()) JSON.parse(filterText)
      setError(null)
      setExplaining(true)
      const result = await api.explain(db, coll, filterText.trim() || '{}')
      setExplainResult(JSON.stringify(result, null, 2))
    } catch (e) {
      setError(e instanceof Error ? e.message : t('queryEditor.invalidJson'))
    } finally {
      setExplaining(false)
    }
  }

  // A native, window-level capture-phase listener — not a CodeMirror
  // keymap extension, and not a React onKeyDownCapture prop either.
  // CodeMirror 6 installs its own native keydown handler directly on its
  // content DOM node, which fires and calls stopPropagation before a
  // React synthetic capture handler (attached at the React root, not
  // document) ever sees the event; a keymap.of([...]) extension has the
  // same problem competing against autocomplete's/basicSetup's own
  // keymaps for facet precedence. Capturing at `window` runs before both.
  const wrapperRef = useRef<HTMLDivElement>(null)
  const runRef = useRef(run)
  runRef.current = run

  useEffect(() => {
    function onKeyDown(e: KeyboardEvent) {
      if (!(e.ctrlKey || e.metaKey) || e.key !== 'Enter') return
      if (!wrapperRef.current?.contains(e.target as Node)) return
      e.preventDefault()
      e.stopPropagation()
      runRef.current()
    }
    window.addEventListener('keydown', onKeyDown, true)
    return () => window.removeEventListener('keydown', onKeyDown, true)
  }, [])

  return (
    <div className={styles.editor} ref={wrapperRef}>
      <CodeMirror
        value={filterText}
        height="80px"
        extensions={extensions}
        onChange={(value) => setFilterText(value)}
        placeholder={t('queryEditor.filterPlaceholder')}
        basicSetup={{ lineNumbers: false, foldGutter: false }}
      />
      <div className={styles.row}>
        <input
          className={styles.textarea}
          style={{ minHeight: 'unset', flex: 1 }}
          value={sortText}
          onChange={(e) => setSortText(e.target.value)}
          placeholder={t('queryEditor.sortPlaceholder')}
        />
      </div>
      <div className={styles.row}>
        <button className={styles.button} onClick={run}>
          {t('queryEditor.run')}
        </button>
        <button className={styles.button} onClick={() => void explain()} disabled={explaining}>
          {explaining ? t('queryEditor.explaining') : t('queryEditor.explain')}
        </button>
        <button className={styles.button} onClick={onNewDocument}>
          {t('queryEditor.newDocument')}
        </button>
        {error && <span className={styles.error}>{error}</span>}
        <div className={styles.spacer} />
        <a className={styles.exportLink} href={exportURL(db, coll, 'json', { filter: filterText })} download>
          {t('queryEditor.exportJson')}
        </a>
        <a className={styles.exportLink} href={exportURL(db, coll, 'csv', { filter: filterText })} download>
          {t('queryEditor.exportCsv')}
        </a>
      </div>
      {explainResult && (
        <pre className={styles.explainOutput}>{explainResult}</pre>
      )}
    </div>
  )
}
