import { useQuery } from '@tanstack/react-query'
import { api } from '../api/client'

export function useConnectionInfo() {
  return useQuery({ queryKey: ['connection'], queryFn: api.connectionInfo })
}
