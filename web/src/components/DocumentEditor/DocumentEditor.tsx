import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import CodeMirror from '@uiw/react-codemirror'
import { json } from '@codemirror/lang-json'
import styles from './DocumentEditor.module.css'
import { api } from '../../api/client'
import { computeJsonFix } from '../../api/jsonFix'
import { useIsDarkMode } from '../../hooks/useIsDarkMode'

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

  useEffect(() => {
    if (target.mode === 'edit' && existing.data) {
      setText(JSON.stringify(existing.data, null, 2))
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
    try {
      JSON.parse(text)
    } catch (e) {
      setError(e instanceof Error ? e.message : t('documentEditor.invalidJson'))
      return
    }
    save.mutate()
  }

  return (
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
  )
}
