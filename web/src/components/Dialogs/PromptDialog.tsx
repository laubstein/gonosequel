import { useEffect, useState, type FormEvent } from 'react'
import styles from './Dialogs.module.css'

interface Props {
  title: string
  label: string
  defaultValue?: string
  confirmLabel: string
  cancelLabel: string
  onConfirm: (value: string) => void
  onCancel: () => void
}

// PromptDialog replaces window.prompt() for free-text input (new
// database/collection name, rename) with something styled consistently
// with the rest of the app, instead of the browser's own unstyled dialog.
export function PromptDialog({ title, label, defaultValue, confirmLabel, cancelLabel, onConfirm, onCancel }: Props) {
  const [value, setValue] = useState(defaultValue ?? '')

  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape') onCancel()
    }
    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [onCancel])

  function handleSubmit(e: FormEvent) {
    e.preventDefault()
    const trimmed = value.trim()
    if (trimmed) onConfirm(trimmed)
  }

  return (
    <div className={styles.overlay} onClick={onCancel}>
      <div className={styles.modal} onClick={(e) => e.stopPropagation()}>
        <div className={styles.title}>{title}</div>
        <form onSubmit={handleSubmit}>
          <div className={styles.field}>
            <label className={styles.label}>{label}</label>
            <input
              className={styles.input}
              value={value}
              onChange={(e) => setValue(e.target.value)}
              autoFocus
              autoComplete="off"
              spellCheck={false}
              onFocus={(e) => e.currentTarget.select()}
            />
          </div>
          <div className={styles.footer}>
            <button type="button" className={styles.button} onClick={onCancel}>
              {cancelLabel}
            </button>
            <button type="submit" className={styles.buttonPrimary} disabled={!value.trim()}>
              {confirmLabel}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
