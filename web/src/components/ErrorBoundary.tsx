import { Component, type ErrorInfo, type ReactNode } from 'react'
import { withTranslation, type WithTranslation } from 'react-i18next'
import styles from './Dialogs/Dialogs.module.css'

interface Props extends WithTranslation {
  children: ReactNode
}

interface State {
  error: Error | null
}

// The app had no error boundary anywhere: an uncaught render error (a bad
// document shape, a stale localStorage draft, anything reaching a
// component in a state it wasn't written for) unmounts the whole React
// tree, leaving a blank page. Reloading doesn't help because the same
// server data — not something transient — triggers the same crash again.
//
// This catches that at the root and offers a real way out: "Try again"
// re-renders the existing tree (fixes anything that was a one-off), and
// "Reload" does a full navigation for state that needs a clean slate.
// Must be a class component — React only recognizes componentDidCatch/
// getDerivedStateFromError, there's no hook equivalent.
class ErrorBoundaryImpl extends Component<Props, State> {
  state: State = { error: null }

  static getDerivedStateFromError(error: Error): State {
    return { error }
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error('gonosequel: uncaught render error', error, info.componentStack)
  }

  render() {
    const { error } = this.state
    if (!error) return this.props.children

    const { t } = this.props
    return (
      <div className={styles.overlayOpaque}>
        <div className={styles.modal}>
          <div className={styles.title}>{t('errorBoundary.title')}</div>
          <div className={styles.message}>{t('errorBoundary.message')}</div>
          <div className={styles.message}>
            <code>{error.message}</code>
          </div>
          <div className={styles.footer}>
            <button className={styles.button} onClick={() => location.reload()}>
              {t('errorBoundary.reload')}
            </button>
            <button className={styles.buttonPrimary} onClick={() => this.setState({ error: null })}>
              {t('errorBoundary.tryAgain')}
            </button>
          </div>
        </div>
      </div>
    )
  }
}

export const ErrorBoundary = withTranslation()(ErrorBoundaryImpl)
