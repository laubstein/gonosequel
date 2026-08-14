import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import styles from './Results.module.css'
import { useDocuments } from '../../hooks/useDocuments'
import { summarizeValue, docId } from '../../api/extjson'
import type { ExtJSONDocument, FindQuery } from '../../types'

type ViewMode = 'table' | 'json'

interface Props {
  db: string
  coll: string
  query: FindQuery
  onOpenDocument: (doc: ExtJSONDocument) => void
}

function collectColumns(docs: ExtJSONDocument[]): string[] {
  const seen = new Set<string>()
  const ordered: string[] = []
  for (const doc of docs) {
    for (const key of Object.keys(doc)) {
      if (!seen.has(key)) {
        seen.add(key)
        ordered.push(key)
      }
    }
  }
  return ordered
}

export function Results({ db, coll, query, onOpenDocument }: Props) {
  const { t } = useTranslation()
  const [mode, setMode] = useState<ViewMode>('table')
  const { data, isLoading, isError, error } = useDocuments(db, coll, query)

  const columns = useMemo(() => collectColumns(data?.documents ?? []), [data])

  if (isLoading) {
    return <div className={styles.empty}>{t('results.loading')}</div>
  }
  if (isError) {
    return <div className={styles.empty}>{error instanceof Error ? error.message : t('results.queryError')}</div>
  }
  if (!data || data.documents.length === 0) {
    return (
      <div className={styles.container}>
        <Toolbar mode={mode} setMode={setMode} total={0} estimate={false} />
        <div className={styles.empty}>{t('results.noDocuments')}</div>
      </div>
    )
  }

  return (
    <div className={styles.container}>
      <Toolbar mode={mode} setMode={setMode} total={data.total} estimate={data.totalIsEstimate} />

      {mode === 'table' ? (
        <div className={styles.tableWrap}>
          <table>
            <thead>
              <tr>
                {columns.map((c) => (
                  <th key={c}>{c}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {data.documents.map((doc) => (
                <tr key={docId(doc)} onClick={() => onOpenDocument(doc)}>
                  {columns.map((c) => (
                    <td key={c}>{c in doc ? summarizeValue(doc[c]) : ''}</td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <div className={styles.jsonView}>
          {data.documents.map((doc) => (
            <div key={docId(doc)} className={styles.jsonDoc} onClick={() => onOpenDocument(doc)}>
              {JSON.stringify(doc, null, 2)}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

function Toolbar({
  mode,
  setMode,
  total,
  estimate,
}: {
  mode: ViewMode
  setMode: (m: ViewMode) => void
  total: number
  estimate: boolean
}) {
  const { t } = useTranslation()
  return (
    <div className={styles.toolbar}>
      <button className={mode === 'table' ? styles.toggleButtonActive : styles.toggleButton} onClick={() => setMode('table')}>
        {t('results.table')}
      </button>
      <button className={mode === 'json' ? styles.toggleButtonActive : styles.toggleButton} onClick={() => setMode('json')}>
        {t('results.json')}
      </button>
      <span className={styles.status}>
        {estimate ? '~' : ''}
        {t('results.documentCount', { count: total, formattedCount: total.toLocaleString() })}
      </span>
    </div>
  )
}
