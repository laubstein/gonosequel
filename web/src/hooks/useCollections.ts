import { useQuery } from '@tanstack/react-query'
import { api } from '../api/client'

export function useCollections(db: string | null) {
  return useQuery({
    queryKey: ['collections', db],
    queryFn: () => api.listCollections(db as string),
    enabled: db !== null,
  })
}
