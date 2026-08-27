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
    a.download = `${coll}.json`
    a.click()
    URL.revokeObjectURL(url)
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

  const remove = useMutation({
    mutationFn: async () => {
      if (target.mode !== 'edit') throw new Error('no document to delete')
      return api.deleteDocument(db, coll, target.encodedId)
    },
    onSuccess: () => {
      invalidate()
      onClose()
    },
    onError: (e) => setError(e instanceof Error ? e.message : String(e)),
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
            {target.mode === 'edit' && existing.isLoading ? (
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
              <button className={styles.buttonDanger} onClick={() => remove.mutate()} disabled={remove.isPending}>
                {t('documentEditor.delete')}
              </button>
            )}
            {!(target.mode === 'edit' && existing.isLoading) && (
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
            <button className={styles.buttonPrimary} onClick={handleSave} disabled={save.isPending}>
              {t('documentEditor.save')}
            </button>
          </div>
        </div>
      </div>

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
