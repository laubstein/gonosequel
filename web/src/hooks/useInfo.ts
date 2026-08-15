import { useQuery } from '@tanstack/react-query'
import { api } from '../api/client'

// SERVER_HEALTH_POLL_MS is how often /api/info — an unauthenticated,
// database-independent liveness check (see pkg/api/handlers_info.go's own
// doc comment) — is re-fetched. App.tsx uses a failing fetch here to
// detect the server itself being unreachable (not a backend/database
// issue, which /api/info never touches) and show the connection-lost
// placeholder; the interval is the "de tempos em tempos" automatic retry
// that placeholder's own Retry button supplements with an immediate one.
const SERVER_HEALTH_POLL_MS = 5000

export function useInfo() {
  return useQuery({
    queryKey: ['info'],
    queryFn: api.info,
    refetchInterval: SERVER_HEALTH_POLL_MS,
    // The interval above is already the retry loop; react-query's own
    // exponential-backoff retries on top of it would just delay when a
    // failure actually surfaces as isError.
    retry: false,
  })
}
