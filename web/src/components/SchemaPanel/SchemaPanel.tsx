import { useTranslation } from 'react-i18next'
import styles from './SchemaPanel.module.css'
import ui from '../../styles/ui.module.css'
import { useCollectionSchema } from '../../hooks/useCollectionSchema'
import { QueryState } from '../QueryState/QueryState'

interface Props {
  db: string
  coll: string
}

export function SchemaPanel({ db, coll }: Props) {
  const { t } = useTranslation()
  const { data, isLoading, isError, error } = useCollectionSchema(db, coll)

  return (
    <QueryState
      isLoading={isLoading}
      isError={isError}
      error={error}
      isEmpty={!data || data.length === 0}
      emptyLabel={t('schemaPanel.noData')}
      loadingLabel={t('schemaPanel.loading')}
    >
      <div className={styles.panel}>
        <table className={ui.table}>
          <thead>
            <tr>
              <th>{t('schemaPanel.field')}</th>
              <th>{t('schemaPanel.observedTypes')}</th>
            </tr>
          </thead>
          <tbody>
            {data?.map((field) => (
              <tr key={field.path}>
                <td>{field.path}</td>
                <td>
                  {/* Named `ft`, not `t`: shadowing the translation
                      function inside this map is a footgun waiting for
                      the first translated string added here. */}
                  {field.types.map((ft) => (
                    <span key={ft.type} className={styles.typeTag}>
                      {ft.type} ({ft.count})
                    </span>
                  ))}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </QueryState>
  )
}
