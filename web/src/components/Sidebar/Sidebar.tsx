import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import styles from './Sidebar.module.css'
import { useDatabases } from '../../hooks/useDatabases'
import { useCollections } from '../../hooks/useCollections'
import { useCollectionStats } from '../../hooks/useCollectionStats'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '../../api/client'
import { ConfirmDialog } from '../Dialogs/ConfirmDialog'
import { PromptDialog } from '../Dialogs/PromptDialog'

// One dialog can be open at a time — replaces window.prompt/window.confirm
// (unstyled, and can't require typed confirmation) for every destructive
// or free-text action in this sidebar.
type DialogState =
  | { kind: 'none' }
  | { kind: 'newDatabase' }
  | { kind: 'dropDatabase' }
  | { kind: 'newCollection' }
  | { kind: 'dropCollection'; name: string }
  | { kind: 'renameCollection'; name: string }

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
  const [dialog, setDialog] = useState<DialogState>({ kind: 'none' })
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

  function closeDialog() {
    setDialog({ kind: 'none' })
  }

  const createDatabase = useMutation({
    mutationFn: (name: string) => api.createDatabase(name),
    onSuccess: (_r, name) => {
      void queryClient.invalidateQueries({ queryKey: ['databases'] })
      onSelectDb(name)
      closeDialog()
    },
  })

  const dropDatabase = useMutation({
    mutationFn: (name: string) => api.dropDatabase(name),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['databases'] })
      onSelectDb(null)
      closeDialog()
    },
  })

  const createCollection = useMutation({
    mutationFn: (name: string) => api.createCollection(selectedDb as string, name),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['collections', selectedDb] })
      closeDialog()
    },
  })

  const dropCollection = useMutation({
    mutationFn: (name: string) => api.dropCollection(selectedDb as string, name),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['collections', selectedDb] })
      closeDialog()
    },
  })

  const renameCollection = useMutation({
    mutationFn: ({ oldName, newName }: { oldName: string; newName: string }) =>
      api.renameCollection(selectedDb as string, oldName, newName),
    onSuccess: (_r, { oldName, newName }) => {
      void queryClient.invalidateQueries({ queryKey: ['collections', selectedDb] })
      onCollectionRenamed?.(oldName, newName)
      closeDialog()
    },
  })

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
        <button
          className={styles.iconButton}
          onClick={() => setDialog({ kind: 'newDatabase' })}
          title={t('sidebar.newDatabase')}
          aria-label={t('sidebar.newDatabase')}
        >
          +
        </button>
        <button
          className={styles.iconButton}
          onClick={() => setDialog({ kind: 'dropDatabase' })}
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
            onClick={isKeyValueDriver ? onNewKey : () => setDialog({ kind: 'newCollection' })}
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
              onClick={() => setDialog({ kind: 'renameCollection', name: c.name })}
              title={t('sidebar.renameCollection', { name: c.name })}
              aria-label={t('sidebar.renameCollection', { name: c.name })}
            >
              ✎
            </button>
            <button
              className={styles.dropButton}
              onClick={() => setDialog({ kind: 'dropCollection', name: c.name })}
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

      {dialog.kind === 'newDatabase' && (
        <PromptDialog
          title={t('sidebar.newDatabaseTitle')}
          label={t('sidebar.promptNewDatabaseName')}
          confirmLabel={t('dialog.create')}
          cancelLabel={t('dialog.cancel')}
          onConfirm={(name) => createDatabase.mutate(name)}
          onCancel={closeDialog}
        />
      )}

      {dialog.kind === 'dropDatabase' && selectedDb && (
        <ConfirmDialog
          title={t('sidebar.dropDatabaseTitle')}
          message={t('sidebar.confirmDropDatabase', { name: selectedDb })}
          confirmLabel={t('dialog.delete')}
          cancelLabel={t('dialog.cancel')}
          danger
          onConfirm={() => dropDatabase.mutate(selectedDb)}
          onCancel={closeDialog}
        />
      )}

      {dialog.kind === 'newCollection' && selectedDb && (
        <PromptDialog
          title={t('sidebar.newCollectionTitle')}
          label={t('sidebar.promptNewCollectionName')}
          confirmLabel={t('dialog.create')}
          cancelLabel={t('dialog.cancel')}
          onConfirm={(name) => createCollection.mutate(name)}
          onCancel={closeDialog}
        />
      )}

      {dialog.kind === 'dropCollection' && (
        <ConfirmDialog
          title={t('sidebar.dropCollectionTitle')}
          message={t('sidebar.confirmDropCollection', { name: dialog.name })}
          confirmLabel={t('dialog.delete')}
          cancelLabel={t('dialog.cancel')}
          danger
          requireText={dialog.name}
          onConfirm={() => dropCollection.mutate(dialog.name)}
          onCancel={closeDialog}
        />
      )}

      {dialog.kind === 'renameCollection' && (
        <PromptDialog
          title={t('sidebar.renameCollectionTitle')}
          label={t('sidebar.promptRenameCollection', { name: dialog.name })}
          defaultValue={dialog.name}
          confirmLabel={t('dialog.rename')}
          cancelLabel={t('dialog.cancel')}
          onConfirm={(newName) => {
            if (newName !== dialog.name) renameCollection.mutate({ oldName: dialog.name, newName })
            else closeDialog()
          }}
          onCancel={closeDialog}
        />
      )}
    </aside>
  )
}
