import { useId, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useMutation, useQuery } from '@tanstack/react-query'
import styles from './ConnectionModal.module.css'
import { api } from '../../api/client'
import { Modal } from '../Dialogs/Modal'
import { setSessionId } from '../../api/http'
import { useInfo } from '../../hooks/useInfo'
import { SUPPORTED_DRIVERS, DRIVER_LABEL, DEFAULT_PORT, type DriverName } from '../../drivers'

interface Props {
  onConnected: () => void
  // When set, a close button lets the user dismiss the modal without
  // connecting — used when opening it to add a connection alongside an
  // existing one. Omitted for the blocking initial gate (no session yet),
  // where there is nothing to cancel back to.
  onCancel?: () => void
}

type Tab = 'standard' | 'url'

// inferDriverLabel guesses a display name from a raw connection URL's
// scheme, for the URL tab — there's no type selector there, just a pasted
// string, but the title still reacts to what the user typed.
function inferDriverLabel(url: string): string | null {
  if (url.startsWith('redis://') || url.startsWith('rediss://')) return DRIVER_LABEL.redis
  if (url.startsWith('mongodb://') || url.startsWith('mongodb+srv://')) return DRIVER_LABEL.mongodb
  return null
}

export function ConnectionModal({ onConnected, onCancel }: Props) {
  const { t } = useTranslation()
  const [tab, setTab] = useState<Tab>('standard')

  // Standard (form) tab fields.
  const [driver, setDriver] = useState<DriverName>('mongodb')
  const [host, setHost] = useState('localhost')
  const [port, setPort] = useState('')
  const [requiresAuth, setRequiresAuth] = useState(false)
  const [user, setUser] = useState('')
  const [pass, setPass] = useState('')
  const [dbName, setDbName] = useState('')
  const [extraParams, setExtraParams] = useState('')

  // URL (raw string) tab field.
  const [url, setUrl] = useState('')

  // Read-only, shared across both tabs (placed outside the tab-specific
  // content below). When the server itself was started with --readonly,
  // AppInfo.readonly is true — the checkbox is then forced checked and
  // disabled, since a session on this server can never actually be
  // read-write no matter what the form sends (rejectWrites enforces that
  // server-side regardless of the request body). effectiveReadonly is
  // what actually gets sent, so a stale unchecked box can't slip through
  // even if requiresReadonly briefly disagrees with it during a render.
  const [readonly, setReadonly] = useState(false)
  const { data: info } = useInfo()
  const serverForcesReadonly = info?.readonly ?? false
  const effectiveReadonly = serverForcesReadonly || readonly

  const [error, setError] = useState<string | null>(null)

  const {
    data: bookmarkList,
    isError: bookmarksFailed,
  } = useQuery({ queryKey: ['bookmarks'], queryFn: api.bookmarks })


  const connect = useMutation({
    mutationFn: (args: { url: string; driver?: string; readonly: boolean }) =>
      // The name is what labels the session in the Server tab's list. The
      // UI never sent one, so the backend always fell back to the redacted
      // URI and every session was labelled by its own address — unhelpful
      // precisely when several are open.
      api.connect(args.url, args.driver, name.trim() || undefined, args.readonly),
    onSuccess: (res) => {
      setSessionId(res.sessionId)
      onConnected()
    },
    onError: (e) => setError(e instanceof Error ? e.message : String(e)),
  })

  const connectBookmark = useMutation({
    mutationFn: (args: { name: string; readonly: boolean }) => api.connectBookmark(args.name, args.readonly),
    onSuccess: (res) => {
      setSessionId(res.sessionId)
      onConnected()
    },
    onError: (e) => setError(e instanceof Error ? e.message : String(e)),
  })

  // Either connect path in flight blocks the other: clicking a saved
  // connection and then Connect used to fire two competing connects.
  const connecting = connect.isPending || connectBookmark.isPending

  function buildStandardURL(): string {
    const scheme = driver === 'mongodb' ? 'mongodb' : 'redis'
    const auth = requiresAuth && user ? `${encodeURIComponent(user)}:${encodeURIComponent(pass)}@` : ''
    const effectivePort = port || DEFAULT_PORT[driver]
    let built = `${scheme}://${auth}${host || 'localhost'}:${effectivePort}`
    if (dbName) built += `/${encodeURIComponent(dbName)}`
    if (extraParams) built += `?${extraParams}`
    return built
  }

  function handleStandardConnect() {
    connect.mutate({ url: buildStandardURL(), driver, readonly: effectiveReadonly })
  }

  function handleUrlConnect() {
    connect.mutate({ url, readonly: effectiveReadonly })
  }

  const titleDriverLabel = tab === 'standard' ? DRIVER_LABEL[driver] : inferDriverLabel(url)
  const titleId = useId()
  const [name, setName] = useState('')

  return (
    <Modal
      labelledBy={titleId}
      // The initial gate has nowhere to go back to, so it is deliberately
      // not escapable; the "add a connection" variant is.
      onEscape={onCancel}
      backdrop={onCancel ? 'scrim' : 'opaque'}
      width="min(420px, 92vw)"
      className={styles.card}
    >
      <>
        <div className={styles.title} id={titleId}>
          {titleDriverLabel
            ? t('connectionModal.titleFor', { driver: titleDriverLabel })
            : t('connectionModal.title')}
          {onCancel && (
            <button className={styles.closeButton} onClick={onCancel} aria-label={t('connectionModal.cancel')}>
              ✕
            </button>
          )}
        </div>

        <div className={styles.tabs}>
          <button
            className={tab === 'standard' ? styles.tabActive : styles.tab}
            onClick={() => setTab('standard')}
          >
            {t('connectionModal.tabStandard')}
          </button>
          <button className={tab === 'url' ? styles.tabActive : styles.tab} onClick={() => setTab('url')}>
            {t('connectionModal.tabUrl')}
          </button>
        </div>

        <div className={styles.field}>
          <label className={styles.checkboxLabel}>
            <input
              type="checkbox"
              checked={effectiveReadonly}
              disabled={serverForcesReadonly}
              onChange={(e) => setReadonly(e.target.checked)}
            />
            {t('connectionModal.readonlyLabel')}
          </label>
          {serverForcesReadonly && <span className={styles.hint}>{t('connectionModal.readonlyForcedHint')}</span>}
        </div>

        <div className={styles.field}>
          <label className={styles.label}>{t('connectionModal.nameLabel')}</label>
          <input
            className={styles.input}
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder={t('connectionModal.namePlaceholder')}
            autoComplete="off"
          />
        </div>

        {tab === 'standard' ? (
          <>
            <div className={styles.field}>
              <label className={styles.label}>{t('connectionModal.driverLabel')}</label>
              <select className={styles.input} value={driver} onChange={(e) => setDriver(e.target.value as DriverName)}>
                {SUPPORTED_DRIVERS.map((d) => (
                  <option key={d} value={d}>
                    {DRIVER_LABEL[d]}
                  </option>
                ))}
              </select>
            </div>

            <div className={styles.row}>
              <div className={styles.field} style={{ flex: 2 }}>
                <label className={styles.label}>{t('connectionModal.hostLabel')}</label>
                <input className={styles.input} value={host} onChange={(e) => setHost(e.target.value)} placeholder="localhost" />
              </div>
              <div className={styles.field} style={{ flex: 1 }}>
                <label className={styles.label}>{t('connectionModal.portLabel')}</label>
                <input
                  className={styles.input}
                  value={port}
                  onChange={(e) => setPort(e.target.value)}
                  placeholder={String(DEFAULT_PORT[driver])}
                  inputMode="numeric"
                />
              </div>
            </div>

            <div className={styles.field}>
              <label className={styles.checkboxLabel}>
                <input type="checkbox" checked={requiresAuth} onChange={(e) => setRequiresAuth(e.target.checked)} />
                {t('connectionModal.requiresAuth')}
              </label>
            </div>

            {requiresAuth && (
              <div className={styles.row}>
                <div className={styles.field}>
                  <label className={styles.label}>{t('connectionModal.userLabel')}</label>
                  <input className={styles.input} value={user} onChange={(e) => setUser(e.target.value)} />
                </div>
                <div className={styles.field}>
                  <label className={styles.label}>{t('connectionModal.passLabel')}</label>
                  <input
                    className={styles.input}
                    type="password"
                    value={pass}
                    onChange={(e) => setPass(e.target.value)}
                  />
                </div>
              </div>
            )}

            <div className={styles.field}>
              <label className={styles.label}>
                {driver === 'mongodb' ? t('connectionModal.dbLabelMongo') : t('connectionModal.dbLabelRedis')}
              </label>
              <input className={styles.input} value={dbName} onChange={(e) => setDbName(e.target.value)} />
            </div>

            <div className={styles.field}>
              <label className={styles.label}>{t('connectionModal.extraParamsLabel')}</label>
              <input
                className={styles.input}
                value={extraParams}
                onChange={(e) => setExtraParams(e.target.value)}
                placeholder={
                  driver === 'mongodb' ? 'authSource=admin&replicaSet=rs0' : 'dial_timeout=5s'
                }
              />
              <span className={styles.hint}>{t('connectionModal.extraParamsHint')}</span>
            </div>

            <button className={styles.button} onClick={handleStandardConnect} disabled={connecting}>
              {connecting ? t('connectionModal.connecting') : t('connectionModal.connect')}
            </button>
          </>
        ) : (
          <>
            <div className={styles.field}>
              <label className={styles.label}>{t('connectionModal.urlLabel')}</label>
              <input
                className={styles.input}
                value={url}
                onChange={(e) => setUrl(e.target.value)}
                placeholder={t('connectionModal.urlPlaceholder')}
              />
            </div>
            <button className={styles.button} onClick={handleUrlConnect} disabled={connecting || !url.trim()}>
              {connecting ? t('connectionModal.connecting') : t('connectionModal.connect')}
            </button>
          </>
        )}

        {error && <div className={styles.error}>{error}</div>}

        {/* A failed fetch used to hide the saved-connections section
            entirely, so the feature just looked absent. */}
        {bookmarksFailed && <div className={styles.hint}>{t('connectionModal.bookmarksFailed')}</div>}

        {bookmarkList && bookmarkList.length > 0 && (
          <div className={styles.bookmarks}>
            <div className={styles.label}>{t('connectionModal.savedConnections')}</div>
            {bookmarkList.map((b) => (
              <button
                key={b.name}
                type="button"
                className={styles.bookmarkItem}
                disabled={connecting}
                onClick={() => connectBookmark.mutate({ name: b.name, readonly: effectiveReadonly })}
              >
                {b.name} — {b.uri}
              </button>
            ))}
          </div>
        )}
      </>
    </Modal>
  )
}
