import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import styles from './ServerPanel.module.css'
import { useConnectionInfo } from '../../hooks/useConnectionInfo'
import { useServerStatus } from '../../hooks/useServerStatus'
import { useSessions } from '../../hooks/useSessions'
import { api } from '../../api/client'
import { getSessionId, setSessionId } from '../../api/http'
import { ConnectionModal } from '../ConnectionModal/ConnectionModal'

function formatUptime(seconds: number): string {
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  if (days > 0) return `${days}d ${hours}h`
  if (hours > 0) return `${hours}h ${minutes}m`
  return `${minutes}m`
}

export function ServerPanel() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [addingConnection, setAddingConnection] = useState(false)

  const { data: connection } = useConnectionInfo()
  const { data: status, isLoading: statusLoading } = useServerStatus()
  const { data: sessions } = useSessions()

  const disconnect = useMutation({
    mutationFn: () => api.disconnect(),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: ['sessions'] }),
  })

  function switchSession(id: string) {
    setSessionId(id)
    void queryClient.invalidateQueries()
  }

  const activeId = getSessionId()

  return (
    <div className={styles.panel}>
      <div className={styles.section}>
        <div className={styles.sectionTitle}>{t('serverPanel.connection')}</div>
        {connection && (
          <div className={styles.row}>
            <div className={styles.sessionInfo}>
              <span className={styles.sessionName}>{connection.name}</span>
              <span className={styles.sessionUri}>{connection.uri}</span>
            </div>
            <button className={styles.buttonDanger} onClick={() => disconnect.mutate()}>
              {t('serverPanel.disconnect')}
            </button>
          </div>
        )}
      </div>

      <div className={styles.section}>
        <div className={styles.sectionTitle}>{t('app.tabs.server')}</div>
        {statusLoading || !status ? (
          <div className={styles.loading}>{t('serverPanel.loading')}</div>
        ) : (
          <>
            <div className={styles.grid}>
              <div className={styles.stat}>
                <span className={styles.statLabel}>{t('serverPanel.version')}</span>
                <span className={styles.statValue}>{status.version}</span>
              </div>
              <div className={styles.stat}>
                <span className={styles.statLabel}>{t('serverPanel.host')}</span>
                <span className={styles.statValue}>{status.host}</span>
              </div>
              <div className={styles.stat}>
                <span className={styles.statLabel}>{t('serverPanel.process')}</span>
                <span className={styles.statValue}>{status.process}</span>
              </div>
              <div className={styles.stat}>
                <span className={styles.statLabel}>{t('serverPanel.uptime')}</span>
                <span className={styles.statValue}>{formatUptime(status.uptimeSeconds)}</span>
              </div>
              <div className={styles.stat}>
                <span className={styles.statLabel}>{t('serverPanel.connections')}</span>
                <span className={styles.statValue}>
                  {status.connections.current} / {status.connections.available}
                </span>
              </div>
            </div>

            <div className={styles.sectionTitle} style={{ marginTop: 16 }}>
              {t('serverPanel.opcounters')}
            </div>
            <div className={styles.grid}>
              <div className={styles.stat}>
                <span className={styles.statLabel}>{t('serverPanel.insert')}</span>
                <span className={styles.statValue}>{status.opcounters.insert.toLocaleString()}</span>
              </div>
              <div className={styles.stat}>
                <span className={styles.statLabel}>{t('serverPanel.query')}</span>
                <span className={styles.statValue}>{status.opcounters.query.toLocaleString()}</span>
              </div>
              <div className={styles.stat}>
                <span className={styles.statLabel}>{t('serverPanel.update')}</span>
                <span className={styles.statValue}>{status.opcounters.update.toLocaleString()}</span>
              </div>
              <div className={styles.stat}>
                <span className={styles.statLabel}>{t('serverPanel.delete')}</span>
                <span className={styles.statValue}>{status.opcounters.delete.toLocaleString()}</span>
              </div>
              <div className={styles.stat}>
                <span className={styles.statLabel}>{t('serverPanel.getmore')}</span>
                <span className={styles.statValue}>{status.opcounters.getmore.toLocaleString()}</span>
              </div>
              <div className={styles.stat}>
                <span className={styles.statLabel}>{t('serverPanel.command')}</span>
                <span className={styles.statValue}>{status.opcounters.command.toLocaleString()}</span>
              </div>
            </div>
          </>
        )}
      </div>

      <div className={styles.section}>
        <div className={styles.row}>
          <div className={styles.sectionTitle} style={{ marginBottom: 0 }}>
            {t('serverPanel.activeConnections')}
          </div>
          <button className={styles.button} onClick={() => setAddingConnection(true)}>
            {t('serverPanel.addConnection')}
          </button>
        </div>
        {sessions?.map((s) => (
          <div key={s.id} className={styles.sessionItem}>
            <div className={styles.sessionInfo}>
              <span className={styles.sessionName}>{s.name}</span>
              <span className={styles.sessionUri}>{s.uri}</span>
            </div>
            {s.id === activeId ? (
              <span className={styles.activeBadge}>{t('serverPanel.active')}</span>
            ) : (
              <button className={styles.button} onClick={() => switchSession(s.id)}>
                {t('serverPanel.useConnection')}
              </button>
            )}
          </div>
        ))}
      </div>

      {addingConnection && (
        <ConnectionModal
          onCancel={() => setAddingConnection(false)}
          onConnected={() => {
            setAddingConnection(false)
            void queryClient.invalidateQueries()
          }}
        />
      )}
    </div>
  )
}
