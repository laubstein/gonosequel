import { useTranslation } from 'react-i18next'
import styles from './QueryState.module.css'
import type { ReactNode } from 'react'

interface Props {
  isLoading: boolean
  isError: boolean
  error?: unknown
  // True when the query succeeded but returned nothing.
  isEmpty: boolean
  emptyLabel: string
  loadingLabel?: string
  children: ReactNode
}

// QueryState renders the loading / error / empty ladder every panel needs
// around a fetched list, so a failure can't be mistaken for an empty
// result. Most panels used to destructure only `isLoading` and fall
// straight through to their empty state, which meant "the server errored"
// and "there is nothing here" rendered identically — the difference that
// matters most when something is wrong.
//
// Returns children only once the data is actually present.
export function QueryState({ isLoading, isError, error, isEmpty, emptyLabel, loadingLabel, children }: Props) {
  const { t } = useTranslation()

  if (isLoading) return <div className={styles.loading}>{loadingLabel ?? t('common.loading')}</div>
  if (isError) {
    return (
      <div className={styles.error} role="alert">
        {error instanceof Error ? error.message : t('common.loadFailed')}
      </div>
    )
  }
  if (isEmpty) return <div className={styles.empty}>{emptyLabel}</div>
  return <>{children}</>
}
