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

export function Pagination({ query, total, onChange, avgObjSize }: Props) {
  const { t } = useTranslation()
  const skip = query.skip ?? 0
  const limit = query.limit ?? 50
  const page = Math.floor(skip / limit) + 1
  const lastPage = Math.max(1, Math.ceil(total / limit))
  const options = limitOptions(avgObjSize ?? 0)

  return (
    <div className={styles.bar}>
      <button className={styles.button} disabled={skip === 0} onClick={() => onChange(Math.max(0, skip - limit), limit)}>
        ◀
      </button>
      <span className={styles.pageLabel}>{t('pagination.page', { page, total: lastPage })}</span>
      <button
        className={styles.button}
        disabled={skip + limit >= total}
        onClick={() => onChange(skip + limit, limit)}
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
