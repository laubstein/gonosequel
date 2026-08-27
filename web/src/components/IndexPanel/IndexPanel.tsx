import { Fragment, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import styles from './IndexPanel.module.css'
import { api } from '../../api/client'
import type { CreateIndexOptions, IndexInfo } from '../../types'

interface Props {
  db: string
  coll: string
}

interface FieldRow {
  field: string
  direction: '1' | '-1'
}

function emptyFieldRow(): FieldRow {
  return { field: '', direction: '1' }
}

// buildFormState reads an existing index back into the create form's
// shape, for the "Edit" (recreate) flow — everything except TTL is
// immutable in MongoDB, so editing anything else means dropping this
// index and creating a new one with the edited spec.
function buildFormState(idx: IndexInfo): {
  fields: FieldRow[]
  unique: boolean
  sparse: boolean
  ttl: string
  partialFilter: string
} {
  const fields = (idx.keys ?? []).map(({ key, value }) => ({
    field: key,
    direction: (value < 0 ? '-1' : '1') as '1' | '-1',
  }))
  return {
    fields: fields.length > 0 ? fields : [emptyFieldRow()],
    unique: idx.unique,
    sparse: idx.sparse ?? false,
    ttl: idx.expireAfterSeconds != null ? String(idx.expireAfterSeconds) : '',
    partialFilter: idx.partialFilterExpression ? JSON.stringify(idx.partialFilterExpression, null, 2) : '',
  }
}

// indexKeysToRequest converts an existing index's key spec into the
// ordered {field, direction} array createIndex expects — the two sides use
// different shapes for the same thing (see AGENTS.md on why key order is
// carried as an array rather than an object).
function indexKeysToRequest(idx: IndexInfo): { field: string; direction: number }[] {
  return (idx.keys ?? []).map(({ key, value }) => ({ field: key, direction: value < 0 ? -1 : 1 }))
}

// optionsOf recovers the create-time options of an existing index, for
// restoring it after a failed recreate.
function optionsOf(idx: IndexInfo): CreateIndexOptions {
  const opts: CreateIndexOptions = { unique: idx.unique, sparse: idx.sparse ?? false }
  if (idx.expireAfterSeconds != null) opts.expireAfterSeconds = idx.expireAfterSeconds
  if (idx.partialFilterExpression) opts.partialFilterExpression = idx.partialFilterExpression
  return opts
}

export function IndexPanel({ db, coll }: Props) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const { data: indexes, isLoading } = useQuery({
    queryKey: ['indexes', db, coll],
    queryFn: () => api.listIndexes(db, coll),
  })

  const [fields, setFields] = useState<FieldRow[]>([emptyFieldRow()])
  const [unique, setUnique] = useState(false)
  const [sparse, setSparse] = useState(false)
  const [ttl, setTtl] = useState('')
  const [partialFilter, setPartialFilter] = useState('')
  const [editingName, setEditingName] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [expanded, setExpanded] = useState<string | null>(null)
  const [ttlEditValue, setTtlEditValue] = useState('')

  function invalidate() {
    void queryClient.invalidateQueries({ queryKey: ['indexes', db, coll] })
  }

  function resetForm() {
    setFields([emptyFieldRow()])
    setUnique(false)
    setSparse(false)
    setTtl('')
    setPartialFilter('')
    setEditingName(null)
    setError(null)
  }

  // IndexPanel isn't remounted on collection switch (App.tsx doesn't key
  // it by db/coll), so without this the create/edit form — including an
  // in-progress "Edit" targeting a specific index name — would keep
  // showing after navigating to a different collection, where that index
  // doesn't even exist. Unlike QueryEditor's drafts, none of this form
  // state is meant to survive a collection switch.
  useEffect(() => {
    resetForm()
    setExpanded(null)
    setTtlEditValue('')
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [db, coll])

  function parseOptions(): CreateIndexOptions | null {
    const opts: CreateIndexOptions = { unique, sparse }
    if (ttl.trim()) {
      const seconds = Number(ttl)
      if (!Number.isFinite(seconds) || seconds < 0) {
        setError(t('indexPanel.invalidTtl'))
        return null
      }
      opts.expireAfterSeconds = seconds
    }
    if (partialFilter.trim()) {
      try {
        opts.partialFilterExpression = JSON.parse(partialFilter) as Record<string, unknown>
      } catch {
        setError(t('indexPanel.invalidPartialFilter'))
        return null
      }
    }
    return opts
  }

  const save = useMutation({
    mutationFn: async () => {
      const keys = fields
        .filter((row) => row.field.trim())
        .map((row) => ({ field: row.field.trim(), direction: Number(row.direction) }))
      const opts = parseOptions()
      if (opts === null) throw new Error(t('indexPanel.invalidForm'))

      if (!editingName) {
        return api.createIndex(db, coll, keys, opts)
      }

      // Editing is drop-then-create: MongoDB can't alter an index in place
      // (except a TTL's expireAfterSeconds, handled separately) and can't
      // rename one, so there is no atomic path. If the create fails the
      // original is already gone, so put it back rather than leaving the
      // collection with no index at all — and if that restore also fails,
      // say so explicitly instead of reporting only the first error.
      const original = indexes?.find((i) => i.name === editingName)
      await api.dropIndex(db, coll, editingName)
      try {
        return await api.createIndex(db, coll, keys, opts)
      } catch (createErr) {
        const message = createErr instanceof Error ? createErr.message : String(createErr)
        if (!original) throw createErr
        try {
          await api.createIndex(db, coll, indexKeysToRequest(original), optionsOf(original))
          throw new Error(t('indexPanel.recreateFailedRestored', { message }))
        } catch (restoreErr) {
          if (restoreErr instanceof Error && restoreErr.message.includes(message)) throw restoreErr
          throw new Error(t('indexPanel.recreateFailedAndLost', { name: editingName, message }))
        }
      }
    },
    onSuccess: () => {
      invalidate()
      resetForm()
    },
    onError: (e) => {
      setError(e instanceof Error ? e.message : String(e))
      // The index list is whatever the failed sequence left behind —
      // recreated, or genuinely dropped. Either way it is not what's on
      // screen any more.
      invalidate()
    },
  })

  const drop = useMutation({
    mutationFn: (name: string) => api.dropIndex(db, coll, name),
    onSuccess: invalidate,
    onError: (e) => setError(e instanceof Error ? e.message : String(e)),
  })

  const updateTtl = useMutation({
    mutationFn: ({ name, seconds }: { name: string; seconds: number }) => api.updateIndexTTL(db, coll, name, seconds),
    onSuccess: invalidate,
    onError: (e) => setError(e instanceof Error ? e.message : String(e)),
  })

  function startEdit(idx: IndexInfo) {
    const state = buildFormState(idx)
    setFields(state.fields)
    setUnique(state.unique)
    setSparse(state.sparse)
    setTtl(state.ttl)
    setPartialFilter(state.partialFilter)
    setEditingName(idx.name)
    setError(null)
  }

  function addFieldRow() {
    setFields((f) => [...f, emptyFieldRow()])
  }

  function removeFieldRow(i: number) {
    setFields((f) => f.filter((_, idx) => idx !== i))
  }

  function updateFieldRow(i: number, patch: Partial<FieldRow>) {
    setFields((f) => f.map((row, idx) => (idx === i ? { ...row, ...patch } : row)))
  }

  const hasAnyField = fields.some((f) => f.field.trim())

  return (
    <div className={styles.panel}>
      {isLoading ? (
        <div>{t('indexPanel.loading')}</div>
      ) : (
        <table>
          <thead>
            <tr>
              <th>{t('indexPanel.name')}</th>
              <th>{t('indexPanel.fields')}</th>
              <th>{t('indexPanel.unique')}</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {indexes?.map((idx) => (
              <Fragment key={idx.name}>
                <tr>
                  <td>
                    <button
                      type="button"
                      className={styles.expandButton}
                      onClick={() => setExpanded(expanded === idx.name ? null : idx.name)}
                    >
                      {expanded === idx.name ? '▾' : '▸'} {idx.name}
                    </button>
                  </td>
                  <td>{(idx.keys ?? []).map(({ key, value }) => `${key}: ${value}`).join(', ')}</td>
                  <td>{idx.unique ? t('indexPanel.yes') : t('indexPanel.no')}</td>
                  <td>
                    {idx.name !== '_id_' && (
                      <>
                        <button className={styles.editButton} onClick={() => startEdit(idx)}>
                          {t('indexPanel.edit')}
                        </button>
                        <button className={styles.dropButton} onClick={() => drop.mutate(idx.name)}>
                          {t('indexPanel.drop')}
                        </button>
                      </>
                    )}
                  </td>
                </tr>
                {expanded === idx.name && (
                  <tr>
                    <td colSpan={4} className={styles.details}>
                      <div>{t('indexPanel.sparse')}: {idx.sparse ? t('indexPanel.yes') : t('indexPanel.no')}</div>
                      <div>
                        {t('indexPanel.ttl')}:{' '}
                        {idx.expireAfterSeconds != null ? (
                          <span className={styles.ttlEditor}>
                            <input
                              className={styles.input}
                              type="number"
                              min={0}
                              value={ttlEditValue}
                              placeholder={String(idx.expireAfterSeconds)}
                              onChange={(e) => setTtlEditValue(e.target.value)}
                            />
                            <button
                              className={styles.button}
                              disabled={!ttlEditValue.trim() || updateTtl.isPending}
                              onClick={() => {
                                const seconds = Number(ttlEditValue)
                                if (Number.isFinite(seconds) && seconds >= 0) {
                                  // Cleared on success only: clearing here
                                  // discarded the typed value even when the
                                  // request failed, with nothing reported.
                                  updateTtl.mutate(
                                    { name: idx.name, seconds },
                                    { onSuccess: () => setTtlEditValue('') },
                                  )
                                }
                              }}
                            >
                              {t('indexPanel.saveTtl')}
                            </button>
                          </span>
                        ) : (
                          t('indexPanel.no')
                        )}
                      </div>
                      {idx.partialFilterExpression && (
                        <div>
                          {t('indexPanel.partialFilter')}:
                          <pre className={styles.jsonPreview}>{JSON.stringify(idx.partialFilterExpression, null, 2)}</pre>
                        </div>
                      )}
                    </td>
                  </tr>
                )}
              </Fragment>
            ))}
          </tbody>
        </table>
      )}

      <div className={styles.form}>
        {editingName && (
          <div className={styles.editNotice}>
            {t('indexPanel.editingNotice', { name: editingName })}
            <button className={styles.cancelEditButton} onClick={resetForm}>
              {t('indexPanel.cancelEdit')}
            </button>
          </div>
        )}

        {fields.map((row, i) => (
          <div key={i} className={styles.fieldRow}>
            <input
              className={styles.input}
              placeholder={t('indexPanel.fieldPlaceholder')}
              value={row.field}
              onChange={(e) => updateFieldRow(i, { field: e.target.value })}
            />
            <select
              className={styles.input}
              value={row.direction}
              onChange={(e) => updateFieldRow(i, { direction: e.target.value as '1' | '-1' })}
            >
              <option value="1">{t('indexPanel.ascending')}</option>
              <option value="-1">{t('indexPanel.descending')}</option>
            </select>
            {fields.length > 1 && (
              <button type="button" className={styles.removeFieldButton} onClick={() => removeFieldRow(i)}>
                ✕
              </button>
            )}
          </div>
        ))}
        <button type="button" className={styles.addFieldButton} onClick={addFieldRow}>
          {t('indexPanel.addField')}
        </button>

        <label>
          <input type="checkbox" checked={unique} onChange={(e) => setUnique(e.target.checked)} /> {t('indexPanel.uniqueLabel')}
        </label>
        <label>
          <input type="checkbox" checked={sparse} onChange={(e) => setSparse(e.target.checked)} /> {t('indexPanel.sparseLabel')}
        </label>
        <input
          className={styles.input}
          type="number"
          min={0}
          placeholder={t('indexPanel.ttlPlaceholder')}
          value={ttl}
          onChange={(e) => setTtl(e.target.value)}
        />
        <textarea
          className={styles.textarea}
          placeholder={t('indexPanel.partialFilterPlaceholder')}
          value={partialFilter}
          onChange={(e) => setPartialFilter(e.target.value)}
        />
        <button className={styles.button} onClick={() => hasAnyField && save.mutate()} disabled={!hasAnyField || save.isPending}>
          {editingName ? t('indexPanel.saveEdit') : t('indexPanel.create')}
        </button>
      </div>
      {error && <div className={styles.error}>{error}</div>}
    </div>
  )
}
