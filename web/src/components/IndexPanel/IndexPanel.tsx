import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import styles from './IndexPanel.module.css'
import { api } from '../../api/client'

interface Props {
  db: string
  coll: string
}

export function IndexPanel({ db, coll }: Props) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const { data: indexes, isLoading } = useQuery({
    queryKey: ['indexes', db, coll],
    queryFn: () => api.listIndexes(db, coll),
  })

  const [field, setField] = useState('')
  const [direction, setDirection] = useState<'1' | '-1'>('1')
  const [unique, setUnique] = useState(false)
  const [error, setError] = useState<string | null>(null)

  function invalidate() {
    void queryClient.invalidateQueries({ queryKey: ['indexes', db, coll] })
  }

  const create = useMutation({
    mutationFn: () => api.createIndex(db, coll, { [field]: Number(direction) }, unique),
    onSuccess: () => {
      invalidate()
      setField('')
      setError(null)
    },
    onError: (e) => setError(e instanceof Error ? e.message : String(e)),
  })

  const drop = useMutation({
    mutationFn: (name: string) => api.dropIndex(db, coll, name),
    onSuccess: invalidate,
  })

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
              <tr key={idx.name}>
                <td>{idx.name}</td>
                <td>{Object.entries(idx.keys ?? {}).map(([k, v]) => `${k}: ${v}`).join(', ')}</td>
                <td>{idx.unique ? t('indexPanel.yes') : t('indexPanel.no')}</td>
                <td>
                  {idx.name !== '_id_' && (
                    <button className={styles.dropButton} onClick={() => drop.mutate(idx.name)}>
                      {t('indexPanel.drop')}
                    </button>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      <div className={styles.form}>
        <input
          className={styles.input}
          placeholder={t('indexPanel.fieldPlaceholder')}
          value={field}
          onChange={(e) => setField(e.target.value)}
        />
        <select className={styles.input} value={direction} onChange={(e) => setDirection(e.target.value as '1' | '-1')}>
          <option value="1">{t('indexPanel.ascending')}</option>
          <option value="-1">{t('indexPanel.descending')}</option>
        </select>
        <label>
          <input type="checkbox" checked={unique} onChange={(e) => setUnique(e.target.checked)} /> {t('indexPanel.uniqueLabel')}
        </label>
        <button className={styles.button} onClick={() => field && create.mutate()} disabled={!field || create.isPending}>
          {t('indexPanel.create')}
        </button>
      </div>
      {error && <div className={styles.error}>{error}</div>}
    </div>
  )
}
