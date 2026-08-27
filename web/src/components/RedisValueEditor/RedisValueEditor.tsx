import { useEffect, useId, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import docStyles from '../DocumentEditor/DocumentEditor.module.css'
import styles from './RedisValueEditor.module.css'
import { api } from '../../api/client'
import { ConfirmDialog } from '../Dialogs/ConfirmDialog'
import { Modal } from '../Dialogs/Modal'
import type { EditorTarget } from '../DocumentEditor/DocumentEditor'

type ValueType = 'string' | 'hash' | 'list' | 'set' | 'zset'

interface HashRow {
  field: string
  value: string
}

interface ZRow {
  member: string
  score: string
}

interface Props {
  db: string
  coll: string
  target: EditorTarget
  onClose: () => void
}

// RedisValueEditor replaces DocumentEditor's free-form JSON textarea with a
// dedicated form per Redis value type (string/hash/list/set/zset), matching
// the {_id, type, ttl, value} shape pkg/redis's driver.Driver implementation
// reads and writes — see pkg/redis/documents.go's readKeyDoc/writeKeyDoc.
export function RedisValueEditor({ db, coll, target, onClose }: Props) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()

  const [key, setKey] = useState('')
  // Seconds until expiry, as text so the field can be cleared while
  // typing. Empty or <= 0 means "no expiry" (Redis PERSIST), matching the
  // -1 readKeyDoc reports for a key without one.
  const [ttl, setTtl] = useState('')
  const [type, setType] = useState<ValueType>('string')
  const [stringValue, setStringValue] = useState('')
  const [hashRows, setHashRows] = useState<HashRow[]>([{ field: '', value: '' }])
  const [listItems, setListItems] = useState<string[]>([''])
  const [setItems, setSetItems] = useState<string[]>([''])
  const [zsetRows, setZsetRows] = useState<ZRow[]>([{ member: '', score: '0' }])
  const [error, setError] = useState<string | null>(null)
  // Deleting a whole key used to fire straight from the footer button.
  const [confirmingDelete, setConfirmingDelete] = useState(false)
  const titleId = useId()

  const existing = useQuery({
    queryKey: ['document', db, coll, target.mode === 'edit' ? target.encodedId : null],
    queryFn: () => api.getDocument(db, coll, target.mode === 'edit' ? target.encodedId : ''),
    enabled: target.mode === 'edit',
  })

  // A failed fetch must not let Save through: the form would still hold its
  // defaults (an empty `string`-typed value), and saving that overwrites
  // the real key with an empty string.
  const loadFailed = target.mode === 'edit' && existing.isError
  const canSave = target.mode === 'new' || existing.data !== undefined

  useEffect(() => {
    if (target.mode !== 'edit' || !existing.data) return
    const doc = existing.data
    setKey(typeof doc._id === 'string' ? doc._id : '')
    setTtl(typeof doc.ttl === 'number' && doc.ttl > 0 ? String(doc.ttl) : '')
    const docType = (typeof doc.type === 'string' ? doc.type : 'string') as ValueType
    setType(docType)
    const value = doc.value

    if (docType === 'string') {
      setStringValue(typeof value === 'string' ? value : '')
    } else if (docType === 'hash' && value && typeof value === 'object') {
      const rows = Object.entries(value as Record<string, unknown>).map(([field, v]) => ({
        field,
        value: String(v),
      }))
      setHashRows(rows.length ? rows : [{ field: '', value: '' }])
    } else if (docType === 'list' && Array.isArray(value)) {
      setListItems(value.length ? value.map(String) : [''])
    } else if (docType === 'set' && Array.isArray(value)) {
      setSetItems(value.length ? value.map(String) : [''])
    } else if (docType === 'zset' && Array.isArray(value)) {
      const rows = (value as { member: unknown; score: unknown }[]).map((z) => ({
        member: String(z.member),
        score: String(z.score),
      }))
      setZsetRows(rows.length ? rows : [{ member: '', score: '0' }])
    }
  }, [target, existing.data])

  function invalidate() {
    void queryClient.invalidateQueries({ queryKey: ['documents', db, coll] })
    // A saved key can create or empty out a "collection" (a derived key
    // prefix grouping, not a real Redis concept — see pkg/redis), so the
    // sidebar's collection list needs a refresh too, not just this
    // collection's own document list.
    void queryClient.invalidateQueries({ queryKey: ['collections', db] })
  }

  function buildValue(): unknown {
    switch (type) {
      case 'string':
        return stringValue
      case 'hash': {
        const obj: Record<string, string> = {}
        for (const row of hashRows) if (row.field) obj[row.field] = row.value
        return obj
      }
      case 'list':
        return listItems.filter((v) => v !== '')
      case 'set':
        return setItems.filter((v) => v !== '')
      case 'zset':
        return zsetRows
          .filter((r) => r.member !== '')
          .map((r) => ({ member: r.member, score: Number(r.score) || 0 }))
    }
  }

  const save = useMutation({
    mutationFn: async () => {
      // ttl is always sent, including as 0 to mean "no expiry": every
      // write path here recreates the key, which drops whatever expiry it
      // had, so omitting the field would silently make an expiring key
      // permanent on every save.
      const doc: Record<string, unknown> = {
        type,
        value: buildValue(),
        ttl: Number(ttl) > 0 ? Number(ttl) : 0,
      }
      if (target.mode === 'new') {
        if (key) doc._id = key
        // coll can be "" when creating the first key of a not-yet-existing
        // collection (see App.tsx's newKeyDb flow) — the route needs a
        // non-empty path segment regardless, and the server ignores it
        // once doc._id is set, so derive it from the key's own prefix
        // rather than sending an empty segment (which 405s, since
        // /collections//documents doesn't match the :coll route param).
        const urlColl = coll || key.split(':')[0] || '_'
        return api.insertDocument(db, urlColl, doc)
      }
      return api.replaceDocument(db, coll, target.encodedId, doc)
    },
    onSuccess: () => {
      invalidate()
      onClose()
    },
    onError: (e) => setError(e instanceof Error ? e.message : String(e)),
  })

  function handleSave() {
    setError(null)
    // With no collection context (creating the first key of a new
    // collection from the sidebar's "+ New key"), falling back to an
    // auto-generated id would produce a key starting with ":" — a
    // collection name can't be inferred from nothing, so require one here.
    if (target.mode === 'new' && !coll && !key) {
      setError(t('redisEditor.keyRequired'))
      return
    }
    save.mutate()
  }

  // The failure shows inside the confirm dialog, which stays open — so no
  // onError writing to the editor-level error behind it.
  const remove = useMutation({
    mutationFn: async () => {
      if (target.mode !== 'edit') throw new Error(t('redisEditor.noKeyToDelete'))
      return api.deleteDocument(db, coll, target.encodedId)
    },
    onSuccess: () => {
      invalidate()
      onClose()
    },
  })

  return (
    <>
      <Modal labelledBy={titleId} onEscape={onClose} width="min(720px, 92vw)" className={docStyles.card}>
        <div className={docStyles.header} id={titleId}>
          {target.mode === 'new' ? t('documentEditor.newTitle') : t('documentEditor.editTitle')}
          <div className={docStyles.headerSpacer} />
          <button className={docStyles.closeButton} onClick={onClose} aria-label={t('documentEditor.close')}>
            ✕
          </button>
        </div>

        <div className={docStyles.body}>
          {loadFailed ? (
            <div className={docStyles.error}>
              {existing.error instanceof Error ? existing.error.message : t('redisEditor.loadFailed')}
            </div>
          ) : target.mode === 'edit' && existing.isLoading ? (
            <div>{t('documentEditor.loading')}</div>
          ) : (
            <>
              <div className={styles.row}>
                <label className={styles.label}>{t('redisEditor.key')}</label>
                <input
                  className={styles.input}
                  value={key}
                  onChange={(e) => setKey(e.target.value)}
                  disabled={target.mode === 'edit'}
                  placeholder={coll ? `${coll}:...` : t('redisEditor.keyPlaceholder')}
                />
              </div>
              <div className={styles.row}>
                <label className={styles.label}>{t('redisEditor.type')}</label>
                <select
                  className={styles.input}
                  value={type}
                  onChange={(e) => setType(e.target.value as ValueType)}
                  disabled={target.mode === 'edit'}
                >
                  <option value="string">string</option>
                  <option value="hash">hash</option>
                  <option value="list">list</option>
                  <option value="set">set</option>
                  <option value="zset">zset</option>
                </select>
              </div>

              {type === 'string' && (
                <textarea
                  className={docStyles.textarea}
                  value={stringValue}
                  onChange={(e) => setStringValue(e.target.value)}
                  spellCheck={false}
                />
              )}

              {type === 'hash' && (
                <div className={styles.rows}>
                  {hashRows.map((row, i) => (
                    <div key={i} className={styles.row}>
                      <input
                        className={styles.input}
                        placeholder={t('redisEditor.field')}
                        value={row.field}
                        onChange={(e) =>
                          setHashRows(hashRows.map((r, j) => (j === i ? { ...r, field: e.target.value } : r)))
                        }
                      />
                      <input
                        className={styles.input}
                        placeholder={t('redisEditor.value')}
                        value={row.value}
                        onChange={(e) =>
                          setHashRows(hashRows.map((r, j) => (j === i ? { ...r, value: e.target.value } : r)))
                        }
                      />
                      <button className={styles.removeButton} onClick={() => setHashRows(hashRows.filter((_, j) => j !== i))}>
                        ✕
                      </button>
                    </div>
                  ))}
                  <button className={docStyles.button} onClick={() => setHashRows([...hashRows, { field: '', value: '' }])}>
                    {t('redisEditor.addField')}
                  </button>
                </div>
              )}

              {(type === 'list' || type === 'set') && (
                <div className={styles.rows}>
                  {(type === 'list' ? listItems : setItems).map((item, i) => (
                    <div key={i} className={styles.row}>
                      <input
                        className={styles.input}
                        value={item}
                        onChange={(e) => {
                          const next = [...(type === 'list' ? listItems : setItems)]
                          next[i] = e.target.value
                          if (type === 'list') setListItems(next)
                          else setSetItems(next)
                        }}
                      />
                      <button
                        className={styles.removeButton}
                        onClick={() => {
                          const next = (type === 'list' ? listItems : setItems).filter((_, j) => j !== i)
                          if (type === 'list') setListItems(next)
                          else setSetItems(next)
                        }}
                      >
                        ✕
                      </button>
                    </div>
                  ))}
                  <button
                    className={docStyles.button}
                    onClick={() => (type === 'list' ? setListItems([...listItems, '']) : setSetItems([...setItems, '']))}
                  >
                    {t('redisEditor.addItem')}
                  </button>
                </div>
              )}

              {type === 'zset' && (
                <div className={styles.rows}>
                  {zsetRows.map((row, i) => (
                    <div key={i} className={styles.row}>
                      <input
                        className={styles.input}
                        placeholder={t('redisEditor.member')}
                        value={row.member}
                        onChange={(e) =>
                          setZsetRows(zsetRows.map((r, j) => (j === i ? { ...r, member: e.target.value } : r)))
                        }
                      />
                      <input
                        className={styles.input}
                        placeholder={t('redisEditor.score')}
                        value={row.score}
                        onChange={(e) =>
                          setZsetRows(zsetRows.map((r, j) => (j === i ? { ...r, score: e.target.value } : r)))
                        }
                      />
                      <button className={styles.removeButton} onClick={() => setZsetRows(zsetRows.filter((_, j) => j !== i))}>
                        ✕
                      </button>
                    </div>
                  ))}
                  <button className={docStyles.button} onClick={() => setZsetRows([...zsetRows, { member: '', score: '0' }])}>
                    {t('redisEditor.addMember')}
                  </button>
                </div>
              )}

              <div className={styles.row}>
                <label className={styles.label}>{t('redisEditor.ttlLabel')}</label>
                <input
                  className={styles.input}
                  type="number"
                  min={0}
                  value={ttl}
                  onChange={(e) => setTtl(e.target.value)}
                  placeholder={t('redisEditor.ttlPlaceholder')}
                />
              </div>
            </>
          )}
          {error && <div className={docStyles.error}>{error}</div>}
        </div>

        <div className={docStyles.footer}>
          {target.mode === 'edit' && (
            <button
              className={docStyles.buttonDanger}
              onClick={() => setConfirmingDelete(true)}
              disabled={remove.isPending}
            >
              {remove.isPending ? t('documentEditor.deleting') : t('documentEditor.delete')}
            </button>
          )}
          <div className={docStyles.footerSpacer} />
          <button className={docStyles.button} onClick={onClose}>
            {t('documentEditor.cancel')}
          </button>
          <button className={docStyles.buttonPrimary} onClick={handleSave} disabled={save.isPending || !canSave}>
            {save.isPending ? t('documentEditor.saving') : t('documentEditor.save')}
          </button>
        </div>
      </Modal>

      {confirmingDelete && (
        <ConfirmDialog
          title={t('redisEditor.confirmDeleteTitle')}
          message={t('redisEditor.confirmDelete', { name: key })}
          confirmLabel={t('dialog.delete')}
          cancelLabel={t('dialog.cancel')}
          danger
          pending={remove.isPending}
          error={remove.error instanceof Error ? remove.error.message : null}
          onConfirm={() => remove.mutate()}
          onCancel={() => {
            remove.reset()
            setConfirmingDelete(false)
          }}
        />
      )}
    </>
  )
}
