import { useQuery } from '@tanstack/react-query'
import { api } from '../api/client'
import type { FindQuery } from '../types'

export function useDocuments(db: string | null, coll: string | null, query: FindQuery, enabled = true) {
  return useQuery({
    queryKey: ['documents', db, coll, query],
    queryFn: () => api.findDocuments(db as string, coll as string, query),
    enabled: db !== null && coll !== null && enabled,
    placeholderData: (prev) => prev,
  })
}
