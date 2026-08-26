import { useTranslation } from 'react-i18next'
import styles from './ToolsPanel.module.css'
import { useCollectionsOverview } from '../../hooks/useCollectionsOverview'
import { useIndexUsage } from '../../hooks/useIndexUsage'
import { useCurrentOps } from '../../hooks/useCurrentOps'

interface Props {
  db: string
}

function formatBytes(n: number): string {
  if (n < 1024) return `${n} B`
  const units = ['KB', 'MB', 'GB', 'TB']
  let value = n / 1024
  let i = 0
  while (value >= 1024 && i < units.length - 1) {
    value /= 1024
    i++
  }
  return `${value.toFixed(1)} ${units[i]}`
}

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

  const sortedOverview = overview ? [...overview].sort((a, b) => b.storageBytes - a.storageBytes) : []
  const sortedUsage = usage ? [...usage].sort((a, b) => a.ops - b.ops) : []

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
          <table>
            <thead>
              <tr>
                <th>{t('toolsPanel.collection')}</th>
                <th>{t('toolsPanel.documents')}</th>
                <th>{t('toolsPanel.dataSize')}</th>
                <th>{t('toolsPanel.storageSize')}</th>
                <th>{t('toolsPanel.indexSize')}</th>
                <th>{t('toolsPanel.avgObjSize')}</th>
                <th>{t('toolsPanel.indexCount')}</th>
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
          <table>
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
          <table>
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
