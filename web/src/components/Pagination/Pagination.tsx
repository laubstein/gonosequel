import { useTranslation } from 'react-i18next'
import styles from './Pagination.module.css'
import { limitOptions } from '../../api/sizeGuard'
import type { FindQuery } from '../../types'

interface Props {
  query: FindQuery
  total: number
  totalIsEstimate: boolean
  onChange: (skip: number, limit: number) => void
  // The selected collection's average document size, if known — widens
  // the per-page dropdown down to 1 when documents are large enough that
  // even the smallest normal option risks a large transfer (see
  // api/sizeGuard.ts). Omitted (or 0) keeps the standard options only.
  avgObjSize?: number
}

export function Pagination({ query, total, totalIsEstimate, onChange, avgObjSize }: Props) {
  const { t } = useTranslation()
  const skip = query.skip ?? 0
  const limit = query.limit ?? 50
  const page = Math.floor(skip / limit) + 1
  const lastPage = Math.max(1, Math.ceil(total / limit))
  const options = limitOptions(avgObjSize ?? 0)

  // An estimated total (EstimatedDocumentCount, used when there's no
  // filter — see AGENTS.md on why counting every document per page is not
  // an option) must not be presented as an exact page count. Results
  // already prefixes its document count with "~"; without the same marker
  // here, "page 3 of 412" reads as a fact derived from a guess.
  const pageLabel = t('pagination.page', { page, total: lastPage })

  return (
    <div className={styles.bar}>
      <button
        className={styles.button}
        disabled={skip === 0}
        onClick={() => onChange(Math.max(0, skip - limit), limit)}
        aria-label={t('pagination.previous')}
        title={t('pagination.previous')}
      >
        ◀
      </button>
      <span className={styles.pageLabel}>
        {totalIsEstimate ? `~${pageLabel}` : pageLabel}
      </span>
      <button
        className={styles.button}
        // With an estimated total this bound is itself approximate, so it
        // can hide a real next page. Trust it only for an exact count;
        // otherwise let the user page on and hit an empty page instead.
        disabled={totalIsEstimate ? total > 0 && skip + limit >= total + limit : skip + limit >= total}
        onClick={() => onChange(skip + limit, limit)}
        aria-label={t('pagination.next')}
        title={t('pagination.next')}
      >
        ▶
      </button>
      <div className={styles.spacer} />
      <select className={styles.limitSelect} value={limit} onChange={(e) => onChange(0, Number(e.target.value))}>
        {options.map((n) => (
          <option key={n} value={n}>
            {t('pagination.perPage', { count: n })}
          </option>
        ))}
      </select>
    </div>
  )
}
