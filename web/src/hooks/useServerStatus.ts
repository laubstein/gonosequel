import { useQuery } from '@tanstack/react-query'
import { api } from '../api/client'

export function useServerStatus() {
  return useQuery({ queryKey: ['serverStatus'], queryFn: api.serverStatus })
}
