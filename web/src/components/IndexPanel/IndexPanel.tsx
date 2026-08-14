import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import styles from './IndexPanel.module.css'
import { api } from '../../api/client'

interface Props {
  db: string
  coll: string
}

export function IndexPanel({ db, coll }: Props) {
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
        <div>Carregando…</div>
      ) : (
        <table>
          <thead>
            <tr>
              <th>Nome</th>
              <th>Campos</th>
              <th>Único</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {indexes?.map((idx) => (
              <tr key={idx.name}>
                <td>{idx.name}</td>
                <td>{Object.entries(idx.keys ?? {}).map(([k, v]) => `${k}: ${v}`).join(', ')}</td>
                <td>{idx.unique ? 'sim' : 'não'}</td>
                <td>
                  {idx.name !== '_id_' && (
                    <button className={styles.dropButton} onClick={() => drop.mutate(idx.name)}>
                      Apagar
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
          placeholder="campo"
          value={field}
          onChange={(e) => setField(e.target.value)}
        />
        <select className={styles.input} value={direction} onChange={(e) => setDirection(e.target.value as '1' | '-1')}>
          <option value="1">crescente</option>
          <option value="-1">decrescente</option>
        </select>
        <label>
          <input type="checkbox" checked={unique} onChange={(e) => setUnique(e.target.checked)} /> único
        </label>
        <button className={styles.button} onClick={() => field && create.mutate()} disabled={!field || create.isPending}>
          Criar índice
        </button>
      </div>
      {error && <div className={styles.error}>{error}</div>}
    </div>
  )
}
