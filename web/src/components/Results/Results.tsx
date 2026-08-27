import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import styles from './Results.module.css'
import { useDocuments } from '../../hooks/useDocuments'
import { summarizeValue, docId } from '../../api/extjson'
import { formatBytes } from '../../api/format'
import { safeLimitSuggestion } from '../../api/sizeGuard'
import { JsonView } from '../JsonView/JsonView'
import type { ExtJSONDocument, FindQuery } from '../../types'

type ViewMode = 'table' | 'json'

// Owned by App.tsx (see its own comment for why) — Results only renders
// the warning UI and reports the user's choice back up via onConfirm.
interface SizeGuard {
  avgObjSize: number
  isDangerous: boolean
  confirmed: boolean
  // Whether the collection-stats fetch that avgObjSize/isDangerous depend
  // on has resolved yet (success or error) — while false, it's not yet
  // known whether this page is safe to auto-fetch.
  settled: boolean
  onConfirm: () => void
}

interface Props {
  db: string
  coll: string
  query: FindQuery
  onOpenDocument: (doc: ExtJSONDocument) => void
  // When set (aggregate mode), these documents are rendered directly
  // instead of fetching via query — and rows aren't click-to-edit, since
  // an aggregation stage like $group can produce documents that don't
  // correspond to any real stored document (a synthetic _id, or none of
  // the original fields at all).
  overrideDocuments?: ExtJSONDocument[] | null
  // Whether the documents query should actually run — false while the
  // size guard (sizeGuard, below) hasn't been confirmed yet. Always true
  // in aggregate mode (overrideDocuments set), where sizeGuard is absent.
  enabled: boolean
  onPaginate: (skip: number, limit: number) => void
  sizeGuard?: SizeGuard
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

export function Results({ db, coll, query, onOpenDocument, overrideDocuments, enabled, onPaginate, sizeGuard }: Props) {
  const { t } = useTranslation()
  const [mode, setMode] = useState<ViewMode>('table')
  const fetched = useDocuments(db, coll, query, enabled)

  const editable = overrideDocuments == null
  const documents = overrideDocuments ?? fetched.data?.documents ?? []
  const total = overrideDocuments ? overrideDocuments.length : (fetched.data?.total ?? 0)
  const totalIsEstimate = overrideDocuments ? false : (fetched.data?.totalIsEstimate ?? false)

  const columns = useMemo(() => collectColumns(documents), [documents])

  if (sizeGuard && !sizeGuard.settled) {
    return <div className={styles.empty}>{t('results.loading')}</div>
  }
  if (sizeGuard && sizeGuard.isDangerous && !sizeGuard.confirmed) {
    const limit = query.limit ?? 50
    const suggestion = safeLimitSuggestion(sizeGuard.avgObjSize)
    return (
      <div className={styles.empty}>
        <div>
          {t('results.sizeGuardWarning', {
            avgSize: formatBytes(sizeGuard.avgObjSize),
            estimatedSize: formatBytes(sizeGuard.avgObjSize * limit),
            count: limit,
          })}
        </div>
        <div className={styles.sizeGuardActions}>
          <button className={styles.sizeGuardButton} onClick={sizeGuard.onConfirm}>
            {t('results.sizeGuardLoadAnyway')}
          </button>
          {suggestion !== undefined && suggestion < limit && (
            <button className={styles.sizeGuardButton} onClick={() => onPaginate(0, suggestion)}>
              {t('results.sizeGuardReduceTo', { count: suggestion })}
            </button>
          )}
        </div>
      </div>
    )
  }
  if (!overrideDocuments && fetched.isLoading) {
    return <div className={styles.empty}>{t('results.loading')}</div>
  }
  if (!overrideDocuments && fetched.isError) {
    return (
      <div className={styles.empty}>
        {fetched.error instanceof Error ? fetched.error.message : t('results.queryError')}
      </div>
    )
  }
  if (documents.length === 0) {
    return (
      <div className={styles.container}>
        <Toolbar mode={mode} setMode={setMode} total={0} estimate={false} />
        <div className={styles.empty}>{t('results.noDocuments')}</div>
      </div>
    )
  }

  return (
    <div className={styles.container}>
      <Toolbar mode={mode} setMode={setMode} total={total} estimate={totalIsEstimate} />

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
              {documents.map((doc, i) => (
                <tr key={editable ? docId(doc) : i} onClick={editable ? () => onOpenDocument(doc) : undefined}>
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
          {documents.map((doc, i) => (
            <div key={editable ? docId(doc) : i} className={editable ? styles.jsonDoc : styles.jsonDocReadonly}>
              <JsonView value={doc} onClick={editable ? () => onOpenDocument(doc) : undefined} />
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
