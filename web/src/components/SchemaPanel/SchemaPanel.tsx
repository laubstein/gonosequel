import { useTranslation } from 'react-i18next'
import styles from './SchemaPanel.module.css'
import { useCollectionSchema } from '../../hooks/useCollectionSchema'

interface Props {
  db: string
  coll: string
}

export function SchemaPanel({ db, coll }: Props) {
  const { t } = useTranslation()
  const { data, isLoading } = useCollectionSchema(db, coll)

  if (isLoading) return <div className={styles.empty}>{t('schemaPanel.loading')}</div>
  if (!data || data.length === 0) return <div className={styles.empty}>{t('schemaPanel.noData')}</div>

  return (
    <div className={styles.panel}>
      <table>
        <thead>
          <tr>
            <th>{t('schemaPanel.field')}</th>
            <th>{t('schemaPanel.observedTypes')}</th>
          </tr>
        </thead>
        <tbody>
          {data.map((field) => (
            <tr key={field.path}>
              <td>{field.path}</td>
              <td>
                {field.types.map((t) => (
                  <span key={t.type} className={styles.typeTag}>
                    {t.type} ({t.count})
                  </span>
                ))}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
