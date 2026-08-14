import styles from './Pagination.module.css'
import type { FindQuery } from '../../types'

interface Props {
  query: FindQuery
  total: number
  totalIsEstimate: boolean
  onChange: (skip: number, limit: number) => void
}

const LIMIT_OPTIONS = [10, 25, 50, 100, 250]

export function Pagination({ query, total, onChange }: Props) {
  const skip = query.skip ?? 0
  const limit = query.limit ?? 50
  const page = Math.floor(skip / limit) + 1
  const lastPage = Math.max(1, Math.ceil(total / limit))

  return (
    <div className={styles.bar}>
      <button className={styles.button} disabled={skip === 0} onClick={() => onChange(Math.max(0, skip - limit), limit)}>
        ◀
      </button>
      <span className={styles.pageLabel}>
        página {page} de {lastPage}
      </span>
      <button
        className={styles.button}
        disabled={skip + limit >= total}
        onClick={() => onChange(skip + limit, limit)}
      >
        ▶
      </button>
      <div className={styles.spacer} />
      <select className={styles.limitSelect} value={limit} onChange={(e) => onChange(0, Number(e.target.value))}>
        {LIMIT_OPTIONS.map((n) => (
          <option key={n} value={n}>
            {n} por página
          </option>
        ))}
      </select>
    </div>
  )
}
