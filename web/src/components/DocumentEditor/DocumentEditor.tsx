import { useEffect, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import styles from './DocumentEditor.module.css'
import { api } from '../../api/client'

export type EditorTarget = { mode: 'new' } | { mode: 'edit'; encodedId: string }

interface Props {
  db: string
  coll: string
  target: EditorTarget
  onClose: () => void
}

export function DocumentEditor({ db, coll, target, onClose }: Props) {
  const queryClient = useQueryClient()
  const [text, setText] = useState('{}')
  const [error, setError] = useState<string | null>(null)

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
      setError(e instanceof Error ? e.message : 'JSON inválido')
      return
    }
    save.mutate()
  }

  return (
    <div className={styles.overlay} onClick={onClose}>
      <div className={styles.modal} onClick={(e) => e.stopPropagation()}>
        <div className={styles.header}>
          {target.mode === 'new' ? 'Novo documento' : 'Editar documento'}
          <div className={styles.headerSpacer} />
          <button className={styles.closeButton} onClick={onClose} aria-label="Fechar">
            ✕
          </button>
        </div>

        <div className={styles.body}>
          {target.mode === 'edit' && existing.isLoading ? (
            <div>Carregando…</div>
          ) : (
            <textarea className={styles.textarea} value={text} onChange={(e) => setText(e.target.value)} spellCheck={false} />
          )}
          {error && <div className={styles.error}>{error}</div>}
        </div>

        <div className={styles.footer}>
          {target.mode === 'edit' && (
            <button className={styles.buttonDanger} onClick={() => remove.mutate()} disabled={remove.isPending}>
              Excluir
            </button>
          )}
          <div className={styles.footerSpacer} />
          <button className={styles.button} onClick={onClose}>
            Cancelar
          </button>
          <button className={styles.buttonPrimary} onClick={handleSave} disabled={save.isPending}>
            Salvar
          </button>
        </div>
      </div>
    </div>
  )
}
