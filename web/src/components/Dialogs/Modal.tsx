import { useEffect, useRef, type ReactNode } from 'react'
import styles from './Dialogs.module.css'

const FOCUSABLE =
  'a[href], button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])'

// Every open modal, innermost last. Modals nest — the document editor
// opens a delete confirmation over itself — and each one listens for
// Escape on the document, so without knowing who is on top a single
// Escape closed the confirmation *and* the editor beneath it, discarding
// unsaved edits. Only the topmost entry acts.
const modalStack: symbol[] = []

function isTopmost(id: symbol): boolean {
  return modalStack[modalStack.length - 1] === id
}

interface Props {
  // id of the element holding the dialog's title, for aria-labelledby.
  labelledBy: string
  // Called on Escape. Omit to make the dialog non-dismissible (the
  // connection-lost overlay, which must not be escapable).
  onEscape?: () => void
  // CSS width for the modal card, e.g. 'min(720px, 92vw)'.
  width?: string
  // Extra class on the card, for a consumer whose inner layout needs it
  // (the document editor constrains its height and scrolls its body).
  className?: string
  // 'opaque' hides whatever is behind entirely, for a modal that replaces
  // the app rather than sitting over it (the initial connect gate). The
  // default scrim is for everything else.
  backdrop?: 'scrim' | 'opaque'
  children: ReactNode
}

// Modal is the shared shell behind every overlay in the app: the scrim,
// Escape handling, a focus trap, and the dialog ARIA roles that none of
// these dialogs had.
//
// It deliberately does NOT close on a click on the scrim. Every consumer
// is either destructive or holds text the user typed — a stray click must
// not discard a carefully typed confirmation string or a half-written
// document. Cancel and Escape remain the ways out.
export function Modal({ labelledBy, onEscape, width, className, backdrop = 'scrim', children }: Props) {
  const ref = useRef<HTMLDivElement>(null)
  const idRef = useRef<symbol>(undefined as unknown as symbol)
  if (idRef.current === undefined) idRef.current = Symbol('modal')

  useEffect(() => {
    const id = idRef.current
    modalStack.push(id)
    // Restore focus to whatever opened the dialog, so dismissing it
    // doesn't dump the user back at the top of the page.
    const previouslyFocused = document.activeElement as HTMLElement | null

    const first = ref.current?.querySelector<HTMLElement>(FOCUSABLE)
    // An autoFocus'd field inside the dialog wins: it's the more specific
    // intent (a type-to-confirm input, a name field).
    if (first && !ref.current?.querySelector('[autofocus]')) first.focus()

    function handleKeyDown(e: KeyboardEvent) {
      // Ignore keys aimed at a modal stacked above this one.
      if (!isTopmost(id)) return
      if (e.key === 'Escape') {
        onEscape?.()
        return
      }
      if (e.key !== 'Tab' || !ref.current) return

      const items = Array.from(ref.current.querySelectorAll<HTMLElement>(FOCUSABLE))
      if (items.length === 0) return
      const firstItem = items[0]
      const lastItem = items[items.length - 1]
      // Wrap at both ends, so Tab can't walk out of the dialog into the
      // page behind it.
      if (!e.shiftKey && document.activeElement === lastItem) {
        e.preventDefault()
        firstItem.focus()
      } else if (e.shiftKey && document.activeElement === firstItem) {
        e.preventDefault()
        lastItem.focus()
      }
    }

    document.addEventListener('keydown', handleKeyDown)
    return () => {
      document.removeEventListener('keydown', handleKeyDown)
      const at = modalStack.lastIndexOf(id)
      if (at !== -1) modalStack.splice(at, 1)
      previouslyFocused?.focus?.()
    }
  }, [onEscape])

  return (
    <div className={backdrop === 'opaque' ? styles.overlayOpaque : styles.overlay}>
      <div
        ref={ref}
        className={className ? `${styles.modal} ${className}` : styles.modal}
        style={width ? { width } : undefined}
        role="dialog"
        aria-modal="true"
        aria-labelledby={labelledBy}
      >
        {children}
      </div>
    </div>
  )
}
