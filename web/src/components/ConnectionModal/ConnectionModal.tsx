import { useState } from 'react'
import { useMutation, useQuery } from '@tanstack/react-query'
import styles from './ConnectionModal.module.css'
import { api } from '../../api/client'
import { setSessionId } from '../../api/http'

interface Props {
  onConnected: () => void
}

export function ConnectionModal({ onConnected }: Props) {
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
    <div className={styles.overlay}>
      <div className={styles.card}>
        <div className={styles.title}>Conectar ao MongoDB</div>

        <div className={styles.field}>
          <label className={styles.label}>URL de conexão</label>
          <input
            className={styles.input}
            value={url}
            onChange={(e) => setUrl(e.target.value)}
            placeholder="mongodb://usuário:senha@host:27017"
          />
        </div>

        <button className={styles.button} onClick={() => connect.mutate(url)} disabled={connect.isPending}>
          {connect.isPending ? 'Conectando…' : 'Conectar'}
        </button>
        {error && <div className={styles.error}>{error}</div>}

        {bookmarkList && bookmarkList.length > 0 && (
          <div className={styles.bookmarks}>
            <div className={styles.label}>Conexões salvas</div>
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
