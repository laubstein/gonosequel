import { useTranslation } from 'react-i18next'
import styles from './HistoryPanel.module.css'
import ui from '../../styles/ui.module.css'
import { useHistory } from '../../hooks/useHistory'
import { QueryState } from '../QueryState/QueryState'
import type { HistoryEntry } from '../../types'

interface Props {
  onReplay?: (entry: HistoryEntry) => void
}

export function HistoryPanel({ onReplay }: Props) {
  const { t } = useTranslation()
  const { data, isLoading, isError, error } = useHistory()

  return (
    <QueryState
      isLoading={isLoading}
      isError={isError}
      error={error}
      isEmpty={!data || data.length === 0}
      emptyLabel={t('historyPanel.empty')}
      loadingLabel={t('historyPanel.loading')}
    >
      <div className={styles.panel}>
        <table className={`${ui.table} ${ui.tableRows}`}>
          <thead>
            <tr>
              <th>{t('historyPanel.database')}</th>
              <th>{t('historyPanel.collection')}</th>
              <th>{t('historyPanel.filter')}</th>
            </tr>
          </thead>
          <tbody>
            {data?.map((entry, i) => (
              <tr
                key={i}
                onClick={() => onReplay?.(entry)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' || e.key === ' ') {
                    e.preventDefault()
                    onReplay?.(entry)
                  }
                }}
                role="button"
                tabIndex={0}
                title={t('historyPanel.replayHint')}
              >
                <td>{entry.database}</td>
                <td>{entry.collection}</td>
                <td>{entry.filter}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </QueryState>
  )
}
