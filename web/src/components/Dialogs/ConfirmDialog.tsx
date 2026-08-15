import { useEffect, useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import styles from './Dialogs.module.css'

interface Props {
  title: string
  message: string
  confirmLabel: string
  cancelLabel: string
  // danger renders the confirm button in the destructive (red) style —
  // set for anything that deletes/drops data.
  danger?: boolean
  // requireText, when set, is the exact string the user must type into a
  // confirmation field before the confirm button enables — used for
  // dropping a collection, where a single click (even behind a "are you
  // sure?" dialog) is too easy to fire by habit/muscle memory on the
  // wrong row.
  requireText?: string
  onConfirm: () => void
  onCancel: () => void
}

export function ConfirmDialog({ title, message, confirmLabel, cancelLabel, danger, requireText, onConfirm, onCancel }: Props) {
  const { t } = useTranslation()
  const [typed, setTyped] = useState('')
  const canConfirm = !requireText || typed === requireText

  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape') onCancel()
    }
    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [onCancel])

  function handleSubmit(e: FormEvent) {
    e.preventDefault()
    if (canConfirm) onConfirm()
  }

  return (
    <div className={styles.overlay} onClick={onCancel}>
      <div className={styles.modal} onClick={(e) => e.stopPropagation()}>
        <div className={styles.title}>{title}</div>
        <div className={styles.message}>{message}</div>
        {danger && <div className={styles.warning}>{t('dialog.irreversibleWarning')}</div>}
        <form onSubmit={handleSubmit}>
          {requireText && (
            <div className={styles.field}>
              <label className={styles.label}>{t('dialog.typeToConfirm', { name: requireText })}</label>
              <input
                className={styles.input}
                value={typed}
                onChange={(e) => setTyped(e.target.value)}
                placeholder={requireText}
                autoFocus
                autoComplete="off"
                spellCheck={false}
              />
            </div>
          )}
          <div className={styles.footer}>
            <button type="button" className={styles.button} onClick={onCancel}>
              {cancelLabel}
            </button>
            <button type="submit" className={danger ? styles.buttonDanger : styles.buttonPrimary} disabled={!canConfirm}>
              {confirmLabel}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
