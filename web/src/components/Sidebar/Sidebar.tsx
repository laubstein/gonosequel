import { useMemo, useState } from 'react'
import styles from './Sidebar.module.css'
import { useDatabases } from '../../hooks/useDatabases'
import { useCollections } from '../../hooks/useCollections'
import { useCollectionStats } from '../../hooks/useCollectionStats'
import { useQueryClient } from '@tanstack/react-query'

interface Props {
  selection: { db: string; coll: string } | null
  onSelect: (db: string, coll: string) => void
}

function formatBytes(n: number): string {
  if (n < 1024) return `${n} B`
  const units = ['KB', 'MB', 'GB', 'TB']
  let value = n / 1024
  let i = 0
  while (value >= 1024 && i < units.length - 1) {
    value /= 1024
    i++
  }
  return `${value.toFixed(1)} ${units[i]}`
}

export function Sidebar({ selection, onSelect }: Props) {
  const queryClient = useQueryClient()
  const { data: databases, isLoading: dbsLoading } = useDatabases()
  const [currentDb, setCurrentDb] = useState<string | null>(selection?.db ?? null)
  const [filter, setFilter] = useState('')

  const { data: collections, isLoading: collsLoading } = useCollections(currentDb)
  const { data: stats } = useCollectionStats(currentDb, selection?.coll ?? null)

  const filtered = useMemo(() => {
    if (!collections) return []
    if (!filter.trim()) return collections
    const needle = filter.toLowerCase()
    return collections.filter((c) => c.name.toLowerCase().includes(needle))
  }, [collections, filter])

  function refresh() {
    void queryClient.invalidateQueries({ queryKey: ['databases'] })
    void queryClient.invalidateQueries({ queryKey: ['collections', currentDb] })
  }

  return (
    <aside className={styles.sidebar}>
      <div className={styles.dbRow}>
        <select
          className={styles.dbSelect}
          value={currentDb ?? ''}
          onChange={(e) => setCurrentDb(e.target.value || null)}
        >
          <option value="" disabled>
            {dbsLoading ? 'Carregando…' : 'Selecione um banco'}
          </option>
          {databases?.map((db) => (
            <option key={db.name} value={db.name}>
              {db.name}
            </option>
          ))}
        </select>
        <button className={styles.iconButton} onClick={refresh} title="Atualizar" aria-label="Atualizar">
          ⟳
        </button>
      </div>

      <input
        className={styles.filterInput}
        placeholder="filtrar coleções…"
        value={filter}
        onChange={(e) => setFilter(e.target.value)}
      />

      <ul className={styles.collectionList}>
        {collsLoading && <li className={styles.empty}>Carregando…</li>}
        {!collsLoading && currentDb && filtered.length === 0 && (
          <li className={styles.empty}>Nenhuma coleção</li>
        )}
        {filtered.map((c) => (
          <li
            key={c.name}
            className={
              selection?.db === currentDb && selection?.coll === c.name
                ? styles.collectionItemActive
                : styles.collectionItem
            }
            onClick={() => currentDb && onSelect(currentDb, c.name)}
          >
            {c.name}
          </li>
        ))}
      </ul>

      {stats && (
        <div className={styles.statsPanel}>
          <span>{stats.count.toLocaleString()} documentos</span>
          <span>{formatBytes(stats.sizeBytes)}</span>
          <span>{stats.indexCount} índices</span>
        </div>
      )}
    </aside>
  )
}
