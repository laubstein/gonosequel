import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import styles from './Sidebar.module.css'
import { useDatabases } from '../../hooks/useDatabases'
import { useCollections } from '../../hooks/useCollections'
import { useCollectionStats } from '../../hooks/useCollectionStats'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '../../api/client'
import { formatBytes } from '../../api/format'
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
  // Whether the backend is key-value (it declares the 'command'
  // capability). A collection is a real, independently-creatable object in
  // MongoDB but only a derived grouping by key prefix in Redis/Valkey, so
  // "+ New collection" means something different (or nothing) there; see
  // onNewKey.
  keyValueBackend?: boolean
  // Opens the key editor directly (no collection selected yet) — the
  // Redis/Valkey equivalent of "+ New collection", since a collection
  // there only exists once a key with that prefix does.
  onNewKey?: () => void
}

export function Sidebar({
  selectedDb,
  onSelectDb,
  selection,
  onSelect,
  onCollectionRenamed,
  keyValueBackend,
  onNewKey,
}: Props) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const { data: databases, isLoading: dbsLoading } = useDatabases()
  const [filter, setFilter] = useState('')
  const [dialog, setDialog] = useState<DialogState>({ kind: 'none' })
  const isKeyValueDriver = keyValueBackend ?? false

  const { data: collections, isLoading: collsLoading, isError: collsError, error: collsErr } = useCollections(selectedDb)
  const { data: stats } = useCollectionStats(selectedDb, selection?.coll ?? null)

  // The filter is about one database's collection list, so it must not
  // survive a switch — the same cross-switch state leak already fixed in
  // IndexPanel. Left alone, changing database silently showed a filtered
  // subset of the new one (or "no collections"), with the reason sitting
  // in a box the user had stopped looking at.
  useEffect(() => {
    setFilter('')
  }, [selectedDb])

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

  // The mutation objects live at component level and so outlive any one
  // dialog. Without clearing them on open, a failure from last time is
  // still showing the next time the dialog appears. Reset on open rather
  // than in closeDialog, so it can't race with a mutation's own onSuccess.
  function openDialog(next: DialogState) {
    createDatabase.reset()
    dropDatabase.reset()
    createCollection.reset()
    dropCollection.reset()
    renameCollection.reset()
    setDialog(next)
  }

  // Every dialog reports its own mutation's failure in place and stays
  // open. http.ts's handle() always throws an Error carrying the server's
  // {"error": ...} text, so .message is the useful part.
  function errorOf(e: unknown): string | null {
    if (!e) return null
    return e instanceof Error ? e.message : String(e)
  }

  // MongoDB has no "create database" command — a database exists once it
  // holds a collection, so one has to be created alongside it. The UI used
  // to send no name for it, which meant every database created here was
  // born with a collection literally called "_init" (the server's
  // fallback) that the user never asked for and had to clean up.
  const createDatabase = useMutation({
    mutationFn: ({ name, initialCollection }: { name: string; initialCollection: string }) =>
      api.createDatabase(name, initialCollection),
    onSuccess: (_r, { name }) => {
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
          onClick={() => openDialog({ kind: 'newDatabase' })}
          title={t('sidebar.newDatabase')}
          aria-label={t('sidebar.newDatabase')}
        >
          +
        </button>
        <button
          className={styles.iconButton}
          onClick={() => openDialog({ kind: 'dropDatabase' })}
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
            onClick={isKeyValueDriver ? onNewKey : () => openDialog({ kind: 'newCollection' })}
          >
            {isKeyValueDriver ? t('sidebar.newKey') : t('sidebar.newCollection')}
          </button>
        </div>
      )}

      <ul className={styles.collectionList}>
        {collsLoading && <li className={styles.empty}>{t('sidebar.loading')}</li>}
        {/* A failed list used to fall through to "No collections", so a
            server error was indistinguishable from an empty database. */}
        {collsError && (
          <li className={styles.error}>
            {collsErr instanceof Error ? collsErr.message : t('sidebar.collectionsFailed')}
          </li>
        )}
        {!collsLoading && !collsError && selectedDb && filtered.length === 0 && (
          <li className={styles.empty}>{t('sidebar.noCollections')}</li>
        )}
        {filtered.map((c) => (
          <li key={c.name} className={styles.collectionRow}>
            <button
              type="button"
              className={
                selection?.db === selectedDb && selection?.coll === c.name
                  ? styles.collectionItemActive
                  : styles.collectionItem
              }
              onClick={() => selectedDb && onSelect(selectedDb, c.name)}
              aria-current={selection?.db === selectedDb && selection?.coll === c.name ? 'true' : undefined}
            >
              {c.name}
            </button>
            <button
              className={styles.dropButton}
              onClick={() => openDialog({ kind: 'renameCollection', name: c.name })}
              title={t('sidebar.renameCollection', { name: c.name })}
              aria-label={t('sidebar.renameCollection', { name: c.name })}
            >
              ✎
            </button>
            <button
              className={styles.dropButton}
              onClick={() => openDialog({ kind: 'dropCollection', name: c.name })}
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
          <span>{t('sidebar.dataSize', { size: formatBytes(stats.sizeBytes) })}</span>
          <span>{t('sidebar.storageSize', { size: formatBytes(stats.storageBytes) })}</span>
          <span>{t('sidebar.indexCount', { count: stats.indexCount })}</span>
          <span>{t('sidebar.indexSize', { size: formatBytes(stats.indexBytes) })}</span>
          <span>{t('sidebar.avgObjSize', { size: formatBytes(stats.avgObjSize) })}</span>
        </div>
      )}

      {dialog.kind === 'newDatabase' && (
        <PromptDialog
          title={t('sidebar.newDatabaseTitle')}
          label={t('sidebar.promptNewDatabaseName')}
          confirmLabel={t('dialog.create')}
          cancelLabel={t('dialog.cancel')}
          secondLabel={t('sidebar.promptInitialCollectionName')}
          secondDefaultValue="data"
          secondHint={t('sidebar.initialCollectionHint')}
          pending={createDatabase.isPending}
          error={errorOf(createDatabase.error)}
          onConfirm={(name, initialCollection) => createDatabase.mutate({ name, initialCollection })}
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
          // Dropping a database destroys every collection in it, so it
          // gets at least the guard dropping a single collection already
          // had — previously the more destructive action was the easier
          // one to fire.
          requireText={selectedDb}
          pending={dropDatabase.isPending}
          error={errorOf(dropDatabase.error)}
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
          pending={createCollection.isPending}
          error={errorOf(createCollection.error)}
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
          pending={dropCollection.isPending}
          error={errorOf(dropCollection.error)}
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
          pending={renameCollection.isPending}
          error={errorOf(renameCollection.error)}
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
