import { useQuery } from '@tanstack/react-query'
import { api } from '../api/client'

export function useIndexUsage(db: string | null) {
  return useQuery({
    queryKey: ['toolsIndexUsage', db],
    queryFn: () => api.indexUsage(db as string),
    enabled: db !== null,
  })
}
