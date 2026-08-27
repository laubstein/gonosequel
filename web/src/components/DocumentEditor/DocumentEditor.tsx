import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import CodeMirror from '@uiw/react-codemirror'
import { json } from '@codemirror/lang-json'
import styles from './DocumentEditor.module.css'
import { api } from '../../api/client'
import { computeJsonFix } from '../../api/jsonFix'
import { unwrapNumberWrappers, findRiskyNumberFields, getAtPath, pathToLabel } from '../../api/extjson'
import { useIsDarkMode } from '../../hooks/useIsDarkMode'
import { ConfirmDialog } from '../Dialogs/ConfirmDialog'

export type EditorTarget = { mode: 'new' } | { mode: 'edit'; encodedId: string }

interface Props {
  db: string
  coll: string
  target: EditorTarget
  onClose: () => void
}

const jsonExtensions = [json()]

export function DocumentEditor({ db, coll, target, onClose }: Props) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const isDark = useIsDarkMode()
  const [text, setText] = useState('{}')
  const [error, setError] = useState<string | null>(null)
  const [pendingRiskyFields, setPendingRiskyFields] = useState<string[] | null>(null)
  // Deleting used to fire straight from the footer button, with no
  // guard at all — while *saving* a Long went through a warning dialog.
  const [confirmingDelete, setConfirmingDelete] = useState(false)

  const fixedJson = useMemo(() => computeJsonFix(text), [text])

  function applyFix() {
    if (typeof fixedJson !== 'string') return
    setText(fixedJson)
    setError(null)
  }

  const existing = useQuery({
    queryKey: ['document', db, coll, target.mode === 'edit' ? target.encodedId : null],
    queryFn: () => api.getDocument(db, coll, target.mode === 'edit' ? target.encodedId : ''),
    enabled: target.mode === 'edit',
  })

  // A failed fetch must not let Save through: `text` would still hold the
  // initial '{}', and saving that replaces the real document with an empty
  // one. Loading is equally unsafe for the same reason, so the gate is
  // "we actually have the document" rather than "no error".
  const loadFailed = target.mode === 'edit' && existing.isError
  const canSave = target.mode === 'new' || existing.data !== undefined

  // Fields where existing.data (the original canonical fetch) has a Long
  // or Decimal128 that unwrapNumberWrappers displayed as a bare number
  // whose save can't round-trip exactly — see findRiskyNumberFields. Only
  // computed from the original fetch, not the live `text`, since that's
  // what determines whether saving these fields unedited is actually risky.
  const riskyFields = useMemo(
    () => (target.mode === 'edit' && existing.data ? findRiskyNumberFields(existing.data) : []),
    [target, existing.data],
  )

  useEffect(() => {
    if (target.mode === 'edit' && existing.data) {
      setText(JSON.stringify(unwrapNumberWrappers(existing.data), null, 2))
    }
    if (target.mode === 'new') {
      setText('{\n  \n}')
    }
  }, [target, existing.data])

  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape') onClose()
    }
    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [onClose])

  function invalidate() {
    void queryClient.invalidateQueries({ queryKey: ['documents', db, coll] })
  }

  function downloadDocument() {
    const blob = new Blob([text], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    // Named after the document, not just the collection — downloading
    // three documents from one collection used to produce three files
    // with the same name. Falls back to the collection name when the _id
    // isn't a string (or is missing, for an unsaved new document).
    a.download = `${coll}${documentIdSuffix()}.json`
    a.click()
    URL.revokeObjectURL(url)
  }

  // A filesystem-safe fragment of the document's _id, or '' if there
  // isn't one to use.
  function documentIdSuffix(): string {
    let id: unknown
    try {
      id = (JSON.parse(text) as Record<string, unknown>)._id
    } catch {
      return ''
    }
    if (id !== null && typeof id === 'object') {
      const keys = Object.keys(id as Record<string, unknown>)
      if (keys.length === 1 && keys[0].startsWith('$')) {
        id = (id as Record<string, unknown>)[keys[0]]
      }
    }
    if (typeof id !== 'string' && typeof id !== 'number') return ''
    const safe = String(id).replace(/[^A-Za-z0-9._-]/g, '_').slice(0, 64)
    return safe ? `.${safe}` : ''
  }

  const save = useMutation({
    mutationFn: async () => {
      const doc = JSON.parse(text)
      if (target.mode === 'new') {
        return api.insertDocument(db, coll, doc)
      }
      return api.replaceDocument(db, coll, target.encodedId, doc)
    },
    onSuccess: () => {
      invalidate()
      onClose()
    },
    onError: (e) => setError(e instanceof Error ? e.message : String(e)),
  })

  // The failure shows inside the confirm dialog, which stays open — so no
  // onError writing to the editor-level error behind it.
  const remove = useMutation({
    mutationFn: async () => {
      if (target.mode !== 'edit') throw new Error(t('documentEditor.noDocumentToDelete'))
      return api.deleteDocument(db, coll, target.encodedId)
    },
    onSuccess: () => {
      invalidate()
      onClose()
    },
  })

  function handleSave() {
    setError(null)
    let doc: unknown
    try {
      doc = JSON.parse(text)
    } catch (e) {
      setError(e instanceof Error ? e.message : t('documentEditor.invalidJson'))
      return
    }
    // A risky field is still risky only if it's still exactly the bare
    // number unwrapNumberWrappers displayed — if the user has since
    // rewrapped it (e.g. back to {"$numberLong": "..."}) or changed it
    // intentionally, saving it as typed is not a silent conversion.
    const stillRisky = riskyFields.filter((f) => getAtPath(doc, f.path) === f.displayed)
    if (stillRisky.length > 0) {
      setPendingRiskyFields(stillRisky.map((f) => pathToLabel(f.path)))
      return
    }
    save.mutate()
  }

  return (
    <>
      <div className={styles.overlay} onClick={onClose}>
        <div className={styles.modal} onClick={(e) => e.stopPropagation()}>
          <div className={styles.header}>
            {target.mode === 'new' ? t('documentEditor.newTitle') : t('documentEditor.editTitle')}
            <div className={styles.headerSpacer} />
            <button className={styles.closeButton} onClick={onClose} aria-label={t('documentEditor.close')}>
              ✕
            </button>
          </div>

          <div className={styles.body}>
            {loadFailed ? (
              <div className={styles.error}>
                {existing.error instanceof Error ? existing.error.message : t('documentEditor.loadFailed')}
              </div>
            ) : target.mode === 'edit' && existing.isLoading ? (
              <div>{t('documentEditor.loading')}</div>
            ) : (
              <CodeMirror
                className={styles.codeEditor}
                value={text}
                minHeight="320px"
                extensions={jsonExtensions}
                theme={isDark ? 'dark' : 'light'}
                onChange={setText}
                basicSetup={{ lineNumbers: true, foldGutter: false, closeBrackets: false }}
              />
            )}
            {error && <div className={styles.error}>{error}</div>}
            {typeof fixedJson === 'string' && <div className={styles.hint}>{t('documentEditor.fixJsonHint')}</div>}
          </div>

          <div className={styles.footer}>
            {target.mode === 'edit' && (
              <button
                className={styles.buttonDanger}
                onClick={() => setConfirmingDelete(true)}
                disabled={remove.isPending}
              >
                {remove.isPending ? t('documentEditor.deleting') : t('documentEditor.delete')}
              </button>
            )}
            {canSave && (
              <button className={styles.button} onClick={downloadDocument}>
                {t('documentEditor.download')}
              </button>
            )}
            <div className={styles.footerSpacer} />
            {typeof fixedJson === 'string' && (
              <button className={styles.buttonDanger} onClick={applyFix}>
                {t('documentEditor.fixJson')}
              </button>
            )}
            <button className={styles.button} onClick={onClose}>
              {t('documentEditor.cancel')}
            </button>
            <button className={styles.buttonPrimary} onClick={handleSave} disabled={save.isPending || !canSave}>
              {save.isPending ? t('documentEditor.saving') : t('documentEditor.save')}
            </button>
          </div>
        </div>
      </div>

      {confirmingDelete && (
        <ConfirmDialog
          title={t('documentEditor.confirmDeleteTitle')}
          message={t('documentEditor.confirmDelete')}
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

      {pendingRiskyFields && (
        <ConfirmDialog
          title={t('documentEditor.riskyNumberTitle')}
          message={
            <>
              {pendingRiskyFields.map((f, i) => (
                <span key={f}>
                  {i > 0 && ', '}
                  <code>{f}</code>
                </span>
              ))}{' '}
              {t('documentEditor.riskyNumberWarning')}
            </>
          }
          confirmLabel={t('documentEditor.riskyNumberConfirm')}
          cancelLabel={t('dialog.cancel')}
          onConfirm={() => {
            setPendingRiskyFields(null)
            save.mutate()
          }}
          onCancel={() => setPendingRiskyFields(null)}
        />
      )}
    </>
  )
}
