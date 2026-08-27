import { useEffect, useState, type FormEvent } from 'react'
import styles from './Dialogs.module.css'

interface Props {
  title: string
  label: string
  defaultValue?: string
  confirmLabel: string
  cancelLabel: string
  // An optional second input, for the cases where one value alone can't
  // create the thing — MongoDB materializes a database only once it holds
  // a collection, so "new database" genuinely needs two names.
  secondLabel?: string
  secondDefaultValue?: string
  secondHint?: string
  onConfirm: (value: string, second: string) => void
  onCancel: () => void
}

// PromptDialog replaces window.prompt() for free-text input (new
// database/collection name, rename) with something styled consistently
// with the rest of the app, instead of the browser's own unstyled dialog.
export function PromptDialog({
  title,
  label,
  defaultValue,
  confirmLabel,
  cancelLabel,
  secondLabel,
  secondDefaultValue,
  secondHint,
  onConfirm,
  onCancel,
}: Props) {
  const [value, setValue] = useState(defaultValue ?? '')
  const [second, setSecond] = useState(secondDefaultValue ?? '')

  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape') onCancel()
    }
    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [onCancel])

  const canConfirm = value.trim() !== '' && (secondLabel === undefined || second.trim() !== '')

  function handleSubmit(e: FormEvent) {
    e.preventDefault()
    if (canConfirm) onConfirm(value.trim(), second.trim())
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
          {secondLabel !== undefined && (
            <div className={styles.field}>
              <label className={styles.label}>{secondLabel}</label>
              <input
                className={styles.input}
                value={second}
                onChange={(e) => setSecond(e.target.value)}
                autoComplete="off"
                spellCheck={false}
              />
              {secondHint && <div className={styles.hint}>{secondHint}</div>}
            </div>
          )}
          <div className={styles.footer}>
            <button type="button" className={styles.button} onClick={onCancel}>
              {cancelLabel}
            </button>
            <button type="submit" className={styles.buttonPrimary} disabled={!canConfirm}>
              {confirmLabel}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
