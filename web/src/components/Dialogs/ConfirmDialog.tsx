import { useId, useState, type FormEvent, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import styles from './Dialogs.module.css'
import { Modal } from './Modal'

interface Props {
  title: string
  message: ReactNode
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
  // pending disables both buttons and swaps the confirm label while the
  // caller's mutation is in flight. Without it every dialog in the app
  // double-submits on a double click.
  pending?: boolean
  // error keeps the dialog open and shows why the action failed. Pass the
  // caller's own mutation error; the dialog closes only from the caller's
  // onSuccess, never on failure.
  error?: string | null
  onConfirm: () => void
  onCancel: () => void
}

export function ConfirmDialog({
  title,
  message,
  confirmLabel,
  cancelLabel,
  danger,
  requireText,
  pending,
  error,
  onConfirm,
  onCancel,
}: Props) {
  const { t } = useTranslation()
  const titleId = useId()
  const [typed, setTyped] = useState('')
  // Trimmed: a trailing space from a paste otherwise blocks confirmation
  // with no explanation of what's wrong.
  const canConfirm = !requireText || typed.trim() === requireText

  function handleSubmit(e: FormEvent) {
    e.preventDefault()
    if (canConfirm && !pending) onConfirm()
  }

  return (
    <Modal labelledBy={titleId} onEscape={pending ? undefined : onCancel}>
      <div className={styles.title} id={titleId}>
        {title}
      </div>
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
              disabled={pending}
              autoFocus
              autoComplete="off"
              spellCheck={false}
            />
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
          <button
            type="submit"
            className={danger ? styles.buttonDanger : styles.buttonPrimary}
            disabled={!canConfirm || pending}
          >
            {pending ? t('dialog.working') : confirmLabel}
          </button>
        </div>
      </form>
    </Modal>
  )
}
