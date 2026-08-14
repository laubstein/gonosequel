import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useMutation, useQuery } from '@tanstack/react-query'
import styles from './ConnectionModal.module.css'
import { api } from '../../api/client'
import { setSessionId } from '../../api/http'

interface Props {
  onConnected: () => void
  // When set, a close button lets the user dismiss the modal without
  // connecting — used when opening it to add a connection alongside an
  // existing one. Omitted for the blocking initial gate (no session yet),
  // where there is nothing to cancel back to.
  onCancel?: () => void
}

export function ConnectionModal({ onConnected, onCancel }: Props) {
  const { t } = useTranslation()
  const [url, setUrl] = useState('mongodb://localhost:27017')
  const [error, setError] = useState<string | null>(null)

  const { data: bookmarkList } = useQuery({ queryKey: ['bookmarks'], queryFn: api.bookmarks })

  const connect = useMutation({
    mutationFn: (targetUrl: string) => api.connect(targetUrl),
    onSuccess: (res) => {
      setSessionId(res.sessionId)
      onConnected()
    },
    onError: (e) => setError(e instanceof Error ? e.message : String(e)),
  })

  const connectBookmark = useMutation({
    mutationFn: (name: string) => api.connectBookmark(name),
    onSuccess: (res) => {
      setSessionId(res.sessionId)
      onConnected()
    },
    onError: (e) => setError(e instanceof Error ? e.message : String(e)),
  })

  return (
    <div className={onCancel ? styles.overlayDialog : styles.overlay}>
      <div className={styles.card}>
        <div className={styles.title}>
          {t('connectionModal.title')}
          {onCancel && (
            <button className={styles.closeButton} onClick={onCancel} aria-label={t('connectionModal.cancel')}>
              ✕
            </button>
          )}
        </div>

        <div className={styles.field}>
          <label className={styles.label}>{t('connectionModal.urlLabel')}</label>
          <input
            className={styles.input}
            value={url}
            onChange={(e) => setUrl(e.target.value)}
            placeholder={t('connectionModal.urlPlaceholder')}
          />
        </div>

        <button className={styles.button} onClick={() => connect.mutate(url)} disabled={connect.isPending}>
          {connect.isPending ? t('connectionModal.connecting') : t('connectionModal.connect')}
        </button>
        {error && <div className={styles.error}>{error}</div>}

        {bookmarkList && bookmarkList.length > 0 && (
          <div className={styles.bookmarks}>
            <div className={styles.label}>{t('connectionModal.savedConnections')}</div>
            {bookmarkList.map((b) => (
              <div key={b.name} className={styles.bookmarkItem} onClick={() => connectBookmark.mutate(b.name)}>
                {b.name} — {b.uri}
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
