import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import styles from './Results.module.css'
import { useDocuments } from '../../hooks/useDocuments'
import { summarizeValue, docId, rawFilterValue } from '../../api/extjson'
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
  onFilterByValue: (field: string, value: unknown) => void
  onHideField: (field: string) => void
  onExcludeValue: (field: string, value: unknown) => void
}

interface MenuState {
  x: number
  y: number
  field: string
  value: unknown
}

// Columns are capped so a document with a large nested structure can't
// produce a table hundreds of columns wide; the JSON view is the way to
// read those.
const MAX_NESTED_DEPTH = 3
const MAX_COLUMNS = 60

// collectColumns lists the table's columns, descending into embedded
// documents so a nested field gets its own dotted column ("SO.nome") —
// the same path MongoDB accepts in a filter or projection.
//
// Without this, projecting {"SO.nome": 1, "SO.versao": 1} produced a
// single "SO" column reading "{2 fields}": the projection worked, but its
// whole point was invisible. Arrays and Extended JSON wrappers ($oid,
// $date, ...) stay whole — an array has no fixed shape to spread across
// columns, and a wrapper is a scalar value, not a subdocument.
function collectColumns(docs: ExtJSONDocument[]): string[] {
  const seen = new Set<string>()
  const ordered: string[] = []

  function add(path: string) {
    if (seen.has(path) || ordered.length >= MAX_COLUMNS) return
    seen.add(path)
    ordered.push(path)
  }

  function walk(value: unknown, prefix: string, depth: number) {
    if (!isPlainSubdocument(value) || depth > MAX_NESTED_DEPTH) {
      add(prefix)
      return
    }
    const entries = Object.entries(value as Record<string, unknown>)
    // An empty subdocument still deserves a column of its own, else the
    // field vanishes from the table entirely.
    if (entries.length === 0) {
      add(prefix)
      return
    }
    for (const [key, child] of entries) {
      walk(child, `${prefix}.${key}`, depth + 1)
    }
  }

  for (const doc of docs) {
    for (const [key, value] of Object.entries(doc)) {
      walk(value, key, 1)
    }
  }
  return ordered
}

// isPlainSubdocument distinguishes a real embedded document from an
// Extended JSON scalar wrapper ({"$oid": ...}), which must not be spread
// into columns.
function isPlainSubdocument(value: unknown): boolean {
  if (value === null || typeof value !== 'object' || Array.isArray(value)) return false
  const keys = Object.keys(value as Record<string, unknown>)
  return !(keys.length === 1 && keys[0].startsWith('$'))
}

// valueAtPath reads a dotted column path out of a document. Returns
// undefined when any segment is missing, which renders as an empty cell —
// documents in one collection need not share a shape.
function valueAtPath(doc: ExtJSONDocument, path: string): unknown {
  let cur: unknown = doc
  for (const segment of path.split('.')) {
    if (cur === null || typeof cur !== 'object' || Array.isArray(cur)) return undefined
    cur = (cur as Record<string, unknown>)[segment]
    if (cur === undefined) return undefined
  }
  return cur
}

// hasPath reports whether the document actually carries the column, so a
// missing field renders blank rather than as the string "null".
function hasPath(doc: ExtJSONDocument, path: string): boolean {
  let cur: unknown = doc
  const segments = path.split('.')
  for (let i = 0; i < segments.length; i++) {
    if (cur === null || typeof cur !== 'object' || Array.isArray(cur)) return false
    const obj = cur as Record<string, unknown>
    if (!(segments[i] in obj)) return false
    cur = obj[segments[i]]
  }
  return true
}

export function Results({
  db,
  coll,
  query,
  onOpenDocument,
  overrideDocuments,
  enabled,
  onPaginate,
  sizeGuard,
  onFilterByValue,
  onHideField,
  onExcludeValue,
}: Props) {
  const { t } = useTranslation()
  const [mode, setMode] = useState<ViewMode>('table')
  const [menu, setMenu] = useState<MenuState | null>(null)
  const fetched = useDocuments(db, coll, query, enabled)

  const editable = overrideDocuments == null
  const documents = overrideDocuments ?? fetched.data?.documents ?? []
  const total = overrideDocuments ? overrideDocuments.length : (fetched.data?.total ?? 0)
  const totalIsEstimate = overrideDocuments ? false : (fetched.data?.totalIsEstimate ?? false)

  const columns = useMemo(() => collectColumns(documents), [documents])

  // useDocuments keeps the previous page's data as placeholder data while
  // the next one loads, so isLoading is false on every fetch after the
  // first. Without consulting isFetching, changing page or filter silently
  // shows the old rows as if they were the new ones.
  const refetching = !overrideDocuments && fetched.isFetching && !fetched.isLoading

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
        <Toolbar mode={mode} setMode={setMode} total={total} estimate={totalIsEstimate} fetching={refetching} />
        <div className={styles.empty}>{t('results.noDocuments')}</div>
      </div>
    )
  }

  return (
    <div className={styles.container}>
      <Toolbar mode={mode} setMode={setMode} total={total} estimate={totalIsEstimate} fetching={refetching} />

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
                    <td
                      key={c}
                      onContextMenu={
                        editable
                          ? (e) => {
                              e.preventDefault()
                              e.stopPropagation()
                              // field is the dotted path, which is exactly what a
                              // MongoDB filter or projection takes — so "Filter by
                              // value" and "Hide field" work on a nested field too.
                              setMenu({ x: e.clientX, y: e.clientY, field: c, value: valueAtPath(doc, c) })
                            }
                          : undefined
                      }
                    >
                      {hasPath(doc, c) ? summarizeValue(valueAtPath(doc, c)) : ''}
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
          {menu && (
            <CellContextMenu
              menu={menu}
              onClose={() => setMenu(null)}
              onFilterByValue={(field, value) => {
                onFilterByValue(field, value)
                setMenu(null)
              }}
              onHideField={(field) => {
                onHideField(field)
                setMenu(null)
              }}
              onExcludeValue={(field, value) => {
                onExcludeValue(field, value)
                setMenu(null)
              }}
            />
          )}
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
  fetching,
}: {
  mode: ViewMode
  setMode: (m: ViewMode) => void
  total: number
  estimate: boolean
  fetching: boolean
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
      {fetching && <span className={styles.fetching}>{t('results.loading')}</span>}
    </div>
  )
}

function CellContextMenu({
  menu,
  onClose,
  onFilterByValue,
  onHideField,
  onExcludeValue,
}: {
  menu: MenuState
  onClose: () => void
  onFilterByValue: (field: string, value: unknown) => void
  onHideField: (field: string) => void
  onExcludeValue: (field: string, value: unknown) => void
}) {
  const { t } = useTranslation()

  useEffect(() => {
    document.addEventListener('click', onClose)
    document.addEventListener('contextmenu', onClose)
    document.addEventListener('scroll', onClose, true)
    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape') onClose()
    }
    document.addEventListener('keydown', handleKeyDown)
    return () => {
      document.removeEventListener('click', onClose)
      document.removeEventListener('contextmenu', onClose)
      document.removeEventListener('scroll', onClose, true)
      document.removeEventListener('keydown', handleKeyDown)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const raw = rawFilterValue(menu.value)

  return (
    <div className={styles.contextMenu} style={{ left: menu.x, top: menu.y }} onClick={(e) => e.stopPropagation()}>
      <button
        className={styles.contextMenuItem}
        disabled={raw === undefined}
        onClick={() => onFilterByValue(menu.field, raw)}
      >
        🔍 {t('results.filterByValue')}
      </button>
      <button
        className={styles.contextMenuItem}
        disabled={raw === undefined}
        onClick={() => onExcludeValue(menu.field, raw)}
      >
        🚫 {t('results.excludeValue')}
      </button>
      <button className={styles.contextMenuItem} onClick={() => onHideField(menu.field)}>
        ✕ {t('results.hideField')}
      </button>
    </div>
  )
}
