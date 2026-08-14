import { useQuery } from '@tanstack/react-query'
import { api } from '../api/client'

export function useCollectionsOverview(db: string | null) {
  return useQuery({
    queryKey: ['toolsCollectionsOverview', db],
    queryFn: () => api.collectionsOverview(db as string),
    enabled: db !== null,
  })
}
