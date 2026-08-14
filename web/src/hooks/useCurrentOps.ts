import { useQuery } from '@tanstack/react-query'
import { api } from '../api/client'

const MIN_SECS = 1

export function useCurrentOps() {
  return useQuery({
    queryKey: ['toolsCurrentOps'],
    queryFn: () => api.currentOps(MIN_SECS),
    refetchInterval: 5000,
  })
}
