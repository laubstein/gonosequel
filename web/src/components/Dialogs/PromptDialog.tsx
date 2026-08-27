import { useId, useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import styles from './Dialogs.module.css'
import { Modal } from './Modal'

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
  // See ConfirmDialog: pending gates double-submission, error keeps the
  // dialog open with the reason instead of failing silently.
  pending?: boolean
  error?: string | null
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
  pending,
  error,
  onConfirm,
  onCancel,
}: Props) {
  const { t } = useTranslation()
  const titleId = useId()
  const [value, setValue] = useState(defaultValue ?? '')
  const [second, setSecond] = useState(secondDefaultValue ?? '')

  const canConfirm = value.trim() !== '' && (secondLabel === undefined || second.trim() !== '')

  function handleSubmit(e: FormEvent) {
    e.preventDefault()
    if (canConfirm && !pending) onConfirm(value.trim(), second.trim())
  }

  return (
    <Modal labelledBy={titleId} onEscape={pending ? undefined : onCancel}>
      <div className={styles.title} id={titleId}>
        {title}
      </div>
      <form onSubmit={handleSubmit}>
        <div className={styles.field}>
          <label className={styles.label}>{label}</label>
          <input
            className={styles.input}
            value={value}
            onChange={(e) => setValue(e.target.value)}
            disabled={pending}
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
              disabled={pending}
              autoComplete="off"
              spellCheck={false}
            />
            {secondHint && <div className={styles.hint}>{secondHint}</div>}
          </div>
        )}
        {error && (
          <div className={styles.error} role="alert">
            {error}
          </div>
        )}
        <div className={styles.footer}>
          <button type="button" className={styles.button} onClick={onCancel} disabled={pending}>
            {cancelLabel}
          </button>
          <button type="submit" className={styles.buttonPrimary} disabled={!canConfirm || pending}>
            {pending ? t('dialog.working') : confirmLabel}
          </button>
        </div>
      </form>
    </Modal>
  )
}
