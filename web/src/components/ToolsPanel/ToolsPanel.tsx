import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import styles from './ToolsPanel.module.css'
import ui from '../../styles/ui.module.css'
import { useCollectionsOverview } from '../../hooks/useCollectionsOverview'
import { useIndexUsage } from '../../hooks/useIndexUsage'
import { useCurrentOps } from '../../hooks/useCurrentOps'
import { formatBytes } from '../../api/format'

interface Props {
  db: string
}

type OverviewSortKey = 'name' | 'count' | 'sizeBytes' | 'storageBytes' | 'indexBytes' | 'avgObjSize' | 'indexCount'

export function ToolsPanel({ db }: Props) {
  const { t } = useTranslation()
  const {
    data: overview,
    isLoading: overviewLoading,
    isError: overviewIsError,
    error: overviewError,
  } = useCollectionsOverview(db)
  const { data: usage, isLoading: usageLoading, isError: usageIsError, error: usageError } = useIndexUsage(db)
  const { data: ops, isLoading: opsLoading, isError: opsIsError, error: opsError } = useCurrentOps()

  // Collection is the default sort — clicking any other column header
  // switches to it (always starting ascending); clicking the active
  // column again flips direction. Same single rule for all seven columns,
  // no special-casing.
  const [overviewSort, setOverviewSort] = useState<{ key: OverviewSortKey; dir: 1 | -1 }>({ key: 'name', dir: 1 })

  function toggleOverviewSort(key: OverviewSortKey) {
    setOverviewSort((prev) => (prev.key === key ? { key, dir: (prev.dir * -1) as 1 | -1 } : { key, dir: 1 }))
  }

  function overviewSortArrow(key: OverviewSortKey): string {
    if (overviewSort.key !== key) return ''
    return overviewSort.dir === 1 ? ' ▲' : ' ▼'
  }

  // Communicates the sort to assistive tech; the ▲/▼ glyph alone is
  // invisible to a screen reader.
  function overviewAriaSort(key: OverviewSortKey): 'ascending' | 'descending' | 'none' {
    if (overviewSort.key !== key) return 'none'
    return overviewSort.dir === 1 ? 'ascending' : 'descending'
  }

  const sortedOverview = overview
    ? [...overview].sort((a, b) => {
        const { key, dir } = overviewSort
        if (key === 'name') return dir * (a.name ?? '').localeCompare(b.name ?? '')
        return dir * (a[key] - b[key])
      })
    : []
  const sortedUsage = usage
    ? [...usage].sort((a, b) => a.collection.localeCompare(b.collection) || a.index.localeCompare(b.index))
    : []

  return (
    <div className={styles.panel}>
      <div className={styles.section}>
        <div className={styles.sectionTitle}>{t('toolsPanel.collectionsOverview')}</div>
        {overviewLoading ? (
          <div className={styles.loading}>{t('toolsPanel.loading')}</div>
        ) : overviewIsError ? (
          <div className={styles.error}>
            {t('toolsPanel.error', { message: overviewError instanceof Error ? overviewError.message : String(overviewError) })}
          </div>
        ) : sortedOverview.length === 0 ? (
          <div className={styles.empty}>{t('toolsPanel.noCollections')}</div>
        ) : (
          <table className={`${ui.table} ${ui.tableNowrap}`}>
            <thead>
              <tr>
                <th aria-sort={overviewAriaSort('name')}>
                  <button
                    type="button"
                    className={styles.sortableHeader}
                    onClick={() => toggleOverviewSort('name')}
                  >
                    {t('toolsPanel.collection')}
                    {overviewSortArrow('name')}
                  </button>
                </th>
                <th aria-sort={overviewAriaSort('count')}>
                  <button
                    type="button"
                    className={styles.sortableHeader}
                    onClick={() => toggleOverviewSort('count')}
                  >
                    {t('toolsPanel.documents')}
                    {overviewSortArrow('count')}
                  </button>
                </th>
                <th aria-sort={overviewAriaSort('sizeBytes')}>
                  <button
                    type="button"
                    className={styles.sortableHeader}
                    onClick={() => toggleOverviewSort('sizeBytes')}
                  >
                    {t('toolsPanel.dataSize')}
                    {overviewSortArrow('sizeBytes')}
                  </button>
                </th>
                <th aria-sort={overviewAriaSort('storageBytes')}>
                  <button
                    type="button"
                    className={styles.sortableHeader}
                    onClick={() => toggleOverviewSort('storageBytes')}
                  >
                    {t('toolsPanel.storageSize')}
                    {overviewSortArrow('storageBytes')}
                  </button>
                </th>
                <th aria-sort={overviewAriaSort('indexBytes')}>
                  <button
                    type="button"
                    className={styles.sortableHeader}
                    onClick={() => toggleOverviewSort('indexBytes')}
                  >
                    {t('toolsPanel.indexSize')}
                    {overviewSortArrow('indexBytes')}
                  </button>
                </th>
                <th aria-sort={overviewAriaSort('avgObjSize')}>
                  <button
                    type="button"
                    className={styles.sortableHeader}
                    onClick={() => toggleOverviewSort('avgObjSize')}
                  >
                    {t('toolsPanel.avgObjSize')}
                    {overviewSortArrow('avgObjSize')}
                  </button>
                </th>
                <th aria-sort={overviewAriaSort('indexCount')}>
                  <button
                    type="button"
                    className={styles.sortableHeader}
                    onClick={() => toggleOverviewSort('indexCount')}
                  >
                    {t('toolsPanel.indexCount')}
                    {overviewSortArrow('indexCount')}
                  </button>
                </th>
              </tr>
            </thead>
            <tbody>
              {sortedOverview.map((s) => (
                <tr key={s.name}>
                  <td>{s.name}</td>
                  <td>{s.count.toLocaleString()}</td>
                  <td>{formatBytes(s.sizeBytes)}</td>
                  <td>{formatBytes(s.storageBytes)}</td>
                  <td>{formatBytes(s.indexBytes)}</td>
                  <td>{formatBytes(s.avgObjSize)}</td>
                  <td>{s.indexCount}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      <div className={styles.section}>
        <div className={styles.sectionTitle}>{t('toolsPanel.indexUsage')}</div>
        {usageLoading ? (
          <div className={styles.loading}>{t('toolsPanel.loading')}</div>
        ) : usageIsError ? (
          <div className={styles.error}>
            {t('toolsPanel.error', { message: usageError instanceof Error ? usageError.message : String(usageError) })}
          </div>
        ) : sortedUsage.length === 0 ? (
          <div className={styles.empty}>{t('toolsPanel.noIndexes')}</div>
        ) : (
          <table className={`${ui.table} ${ui.tableNowrap}`}>
            <thead>
              <tr>
                <th>{t('toolsPanel.collection')}</th>
                <th>{t('toolsPanel.index')}</th>
                <th>{t('toolsPanel.ops')}</th>
                <th>{t('toolsPanel.since')}</th>
              </tr>
            </thead>
            <tbody>
              {sortedUsage.map((u) => (
                <tr key={`${u.collection}.${u.index}`}>
                  <td>{u.collection}</td>
                  <td>{u.index}</td>
                  <td className={u.ops === 0 ? styles.zeroOps : undefined}>{u.ops.toLocaleString()}</td>
                  <td>{new Date(u.since).toLocaleString()}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      <div className={styles.section}>
        <div className={styles.sectionTitle}>{t('toolsPanel.currentOps')}</div>
        {opsLoading ? (
          <div className={styles.loading}>{t('toolsPanel.loading')}</div>
        ) : opsIsError ? (
          <div className={styles.error}>
            {t('toolsPanel.error', { message: opsError instanceof Error ? opsError.message : String(opsError) })}
          </div>
        ) : !ops || ops.length === 0 ? (
          <div className={styles.empty}>{t('toolsPanel.noCurrentOps')}</div>
        ) : (
          <table className={`${ui.table} ${ui.tableNowrap}`}>
            <thead>
              <tr>
                <th>{t('toolsPanel.namespace')}</th>
                <th>{t('toolsPanel.operation')}</th>
                <th>{t('toolsPanel.secsRunning')}</th>
                <th>{t('toolsPanel.client')}</th>
              </tr>
            </thead>
            <tbody>
              {ops.map((op) => (
                <tr key={op.opid}>
                  <td>{op.namespace}</td>
                  <td>{op.op}</td>
                  <td>{op.secsRunning}</td>
                  <td>{op.client}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  )
}
