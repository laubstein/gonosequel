import { useQuery } from '@tanstack/react-query'
import { api } from '../api/client'

export function useCollectionStats(db: string | null, coll: string | null) {
  return useQuery({
    queryKey: ['collectionStats', db, coll],
    queryFn: () => api.collectionStats(db as string, coll as string),
    enabled: db !== null && coll !== null,
  })
}
