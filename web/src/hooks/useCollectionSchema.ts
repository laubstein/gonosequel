import { useQuery } from '@tanstack/react-query'
import { api } from '../api/client'

export function useCollectionSchema(db: string | null, coll: string | null) {
  return useQuery({
    queryKey: ['schema', db, coll],
    queryFn: () => api.collectionSchema(db as string, coll as string),
    enabled: db !== null && coll !== null,
  })
}
