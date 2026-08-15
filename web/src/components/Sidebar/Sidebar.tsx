import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import styles from './Sidebar.module.css'
import { useDatabases } from '../../hooks/useDatabases'
import { useCollections } from '../../hooks/useCollections'
import { useCollectionStats } from '../../hooks/useCollectionStats'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '../../api/client'

interface Props {
  selectedDb: string | null
  onSelectDb: (db: string | null) => void
  selection: { db: string; coll: string } | null
  onSelect: (db: string, coll: string) => void
  onCollectionRenamed?: (oldName: string, newName: string) => void
  // The connected backend, e.g. "mongodb" or "redis" — collections are a
  // real, independently-creatable object in MongoDB but only a derived
  // grouping by key prefix in Redis/Valkey, so "+ New collection" means
  // something different (or nothing) there; see onNewKey.
  driver?: string
  // Opens the key editor directly (no collection selected yet) — the
  // Redis/Valkey equivalent of "+ New collection", since a collection
  // there only exists once a key with that prefix does.
  onNewKey?: () => void
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

export function Sidebar({ selectedDb, onSelectDb, selection, onSelect, onCollectionRenamed, driver, onNewKey }: Props) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const { data: databases, isLoading: dbsLoading } = useDatabases()
  const [filter, setFilter] = useState('')
  const isKeyValueDriver = driver === 'redis' || driver === 'valkey'

  const { data: collections, isLoading: collsLoading } = useCollections(selectedDb)
  const { data: stats } = useCollectionStats(selectedDb, selection?.coll ?? null)

  const filtered = useMemo(() => {
    if (!collections) return []
    if (!filter.trim()) return collections
    const needle = filter.toLowerCase()
    return collections.filter((c) => c.name.toLowerCase().includes(needle))
  }, [collections, filter])

  function refresh() {
    void queryClient.invalidateQueries({ queryKey: ['databases'] })
    void queryClient.invalidateQueries({ queryKey: ['collections', selectedDb] })
  }

  const createDatabase = useMutation({
    mutationFn: (name: string) => api.createDatabase(name),
    onSuccess: (_r, name) => {
      void queryClient.invalidateQueries({ queryKey: ['databases'] })
      onSelectDb(name)
    },
  })

  const dropDatabase = useMutation({
    mutationFn: (name: string) => api.dropDatabase(name),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['databases'] })
      onSelectDb(null)
    },
  })

  const createCollection = useMutation({
    mutationFn: (name: string) => api.createCollection(selectedDb as string, name),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: ['collections', selectedDb] }),
  })

  const dropCollection = useMutation({
    mutationFn: (name: string) => api.dropCollection(selectedDb as string, name),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: ['collections', selectedDb] }),
  })

  const renameCollection = useMutation({
    mutationFn: ({ oldName, newName }: { oldName: string; newName: string }) =>
      api.renameCollection(selectedDb as string, oldName, newName),
    onSuccess: (_r, { oldName, newName }) => {
      void queryClient.invalidateQueries({ queryKey: ['collections', selectedDb] })
      onCollectionRenamed?.(oldName, newName)
    },
  })

  function handleCreateDatabase() {
    const name = window.prompt(t('sidebar.promptNewDatabaseName'))
    if (name) createDatabase.mutate(name)
  }

  function handleDropDatabase() {
    if (!selectedDb) return
    if (window.confirm(t('sidebar.confirmDropDatabase', { name: selectedDb }))) {
      dropDatabase.mutate(selectedDb)
    }
  }

  function handleCreateCollection() {
    if (!selectedDb) return
    const name = window.prompt(t('sidebar.promptNewCollectionName'))
    if (name) createCollection.mutate(name)
  }

  function handleDropCollection(name: string) {
    if (window.confirm(t('sidebar.confirmDropCollection', { name }))) {
      dropCollection.mutate(name)
    }
  }

  function handleRenameCollection(name: string) {
    const newName = window.prompt(t('sidebar.promptRenameCollection', { name }), name)
    if (newName && newName !== name) {
      renameCollection.mutate({ oldName: name, newName })
    }
  }

  return (
    <aside className={styles.sidebar}>
      <div className={styles.dbRow}>
        <select
          className={styles.dbSelect}
          value={selectedDb ?? ''}
          onChange={(e) => onSelectDb(e.target.value || null)}
        >
          <option value="" disabled>
            {dbsLoading ? t('sidebar.loading') : t('sidebar.selectDatabase')}
          </option>
          {databases?.map((db) => (
            <option key={db.name} value={db.name}>
              {db.name}
            </option>
          ))}
        </select>
        <button className={styles.iconButton} onClick={handleCreateDatabase} title={t('sidebar.newDatabase')} aria-label={t('sidebar.newDatabase')}>
          +
        </button>
        <button
          className={styles.iconButton}
          onClick={handleDropDatabase}
          title={t('sidebar.dropDatabase')}
          aria-label={t('sidebar.dropDatabase')}
          disabled={!selectedDb}
        >
          −
        </button>
        <button className={styles.iconButton} onClick={refresh} title={t('sidebar.refresh')} aria-label={t('sidebar.refresh')}>
          ⟳
        </button>
      </div>

      <input
        className={styles.filterInput}
        placeholder={t('sidebar.filterCollections')}
        value={filter}
        onChange={(e) => setFilter(e.target.value)}
      />

      {selectedDb && (
        <div className={styles.newCollectionRow}>
          <button
            className={styles.newCollectionButton}
            onClick={isKeyValueDriver ? onNewKey : handleCreateCollection}
          >
            {isKeyValueDriver ? t('sidebar.newKey') : t('sidebar.newCollection')}
          </button>
        </div>
      )}

      <ul className={styles.collectionList}>
        {collsLoading && <li className={styles.empty}>{t('sidebar.loading')}</li>}
        {!collsLoading && selectedDb && filtered.length === 0 && (
          <li className={styles.empty}>{t('sidebar.noCollections')}</li>
        )}
        {filtered.map((c) => (
          <li key={c.name} className={styles.collectionRow}>
            <span
              className={
                selection?.db === selectedDb && selection?.coll === c.name
                  ? styles.collectionItemActive
                  : styles.collectionItem
              }
              onClick={() => selectedDb && onSelect(selectedDb, c.name)}
            >
              {c.name}
            </span>
            <button
              className={styles.dropButton}
              onClick={() => handleRenameCollection(c.name)}
              title={t('sidebar.renameCollection', { name: c.name })}
              aria-label={t('sidebar.renameCollection', { name: c.name })}
            >
              ✎
            </button>
            <button
              className={styles.dropButton}
              onClick={() => handleDropCollection(c.name)}
              title={t('sidebar.dropCollection', { name: c.name })}
              aria-label={t('sidebar.dropCollection', { name: c.name })}
            >
              ✕
            </button>
          </li>
        ))}
      </ul>

      {stats && (
        <div className={styles.statsPanel}>
          <span>{t('sidebar.documentCount', { count: stats.count, formattedCount: stats.count.toLocaleString() })}</span>
          <span>{formatBytes(stats.sizeBytes)}</span>
          <span>{t('sidebar.indexCount', { count: stats.indexCount })}</span>
        </div>
      )}
    </aside>
  )
}
