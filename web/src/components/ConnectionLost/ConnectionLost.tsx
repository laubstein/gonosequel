import { useTranslation } from 'react-i18next'
import styles from '../Dialogs/Dialogs.module.css'

interface Props {
  onRetry: () => void
  onDisconnect: () => void
  retrying: boolean
}

// ConnectionLost is a blocking placeholder shown when /api/info — a
// database-independent liveness check, see pkg/api/handlers_info.go —
// stops responding, i.e. the browser can't reach the gonosequel server
// itself (not a backend/database connectivity problem, which this never
// detects). useInfo() already re-polls this endpoint every few seconds on
// its own, so Retry here is just an immediate manual nudge rather than the
// only way recovery happens. There's no Escape/click-outside dismissal —
// unlike the app's other modals, there is nothing to fall back to while
// the server is unreachable.
export function ConnectionLost({ onRetry, onDisconnect, retrying }: Props) {
  const { t } = useTranslation()

  return (
    <div className={styles.overlay}>
      <div className={styles.modal}>
        <div className={styles.title}>{t('connectionLost.title')}</div>
        <div className={styles.message}>{t('connectionLost.message')}</div>
        <div className={styles.footer}>
          <button className={styles.buttonDanger} onClick={onDisconnect}>
            {t('connectionLost.disconnect')}
          </button>
          <button className={styles.buttonPrimary} onClick={onRetry} disabled={retrying}>
            {retrying ? t('connectionLost.retrying') : t('connectionLost.retry')}
          </button>
        </div>
      </div>
    </div>
  )
}
