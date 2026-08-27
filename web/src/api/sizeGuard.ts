// Same page-size options as Pagination.tsx's own dropdown — duplicated
// here rather than imported, since exporting it from there only for this
// would be more coupling than the four-line list is worth.
const LIMIT_OPTIONS = [10, 25, 50, 100, 250]

// Above this estimated transfer size (avgObjSize * page limit), Results.tsx
// holds off firing the documents query until the user confirms — a
// collection with large documents (tens of MB each) could otherwise turn a
// routine "list the first page" into a multi-hundred-MB request with no
// warning.
export const SIZE_GUARD_THRESHOLD_BYTES = 5 * 1024 * 1024 // 5 MB

// Largest LIMIT_OPTIONS value whose estimated transfer (avgObjSize * that
// value) still fits under the threshold — undefined if even the smallest
// option (10) doesn't fit, meaning there's no "reduce and load" suggestion
// to offer, only "load anyway".
export function safeLimitSuggestion(avgObjSize: number): number | undefined {
  if (avgObjSize <= 0) return undefined
  let best: number | undefined
  for (const option of LIMIT_OPTIONS) {
    if (avgObjSize * option <= SIZE_GUARD_THRESHOLD_BYTES) best = option
  }
  return best
}
