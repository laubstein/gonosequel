import { useTranslation } from 'react-i18next'
import styles from './HistoryPanel.module.css'
import { useHistory } from '../../hooks/useHistory'
import type { HistoryEntry } from '../../types'

interface Props {
  onReplay?: (entry: HistoryEntry) => void
}

export function HistoryPanel({ onReplay }: Props) {
  const { t } = useTranslation()
  const { data, isLoading } = useHistory()

  if (isLoading) return <div className={styles.empty}>{t('historyPanel.loading')}</div>
  if (!data || data.length === 0) return <div className={styles.empty}>{t('historyPanel.empty')}</div>

  return (
    <div className={styles.panel}>
      <table>
        <thead>
          <tr>
            <th>{t('historyPanel.database')}</th>
            <th>{t('historyPanel.collection')}</th>
            <th>{t('historyPanel.filter')}</th>
          </tr>
        </thead>
        <tbody>
          {data.map((entry, i) => (
            <tr key={i} onClick={() => onReplay?.(entry)}>
              <td>{entry.database}</td>
              <td>{entry.collection}</td>
              <td>{entry.filter}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
