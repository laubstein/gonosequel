// Pagination.tsx's normal per-page options.
const STANDARD_LIMIT_OPTIONS = [10, 25, 50, 100, 250]

// Offered in addition to STANDARD_LIMIT_OPTIONS when documents are large
// enough that even the smallest standard option (10/page) risks a large
// transfer (see needsFineLimitOptions) — lets the user go as low as one
// document per page instead of being stuck at 10.
const FINE_LIMIT_OPTIONS = [1, 2, 3, 4, 5, 6, 7, 8, 9]

// Above this estimated transfer size (avgObjSize * page limit), Results.tsx
// holds off firing the documents query until the user confirms — a
// collection with large documents (tens of MB each) could otherwise turn a
// routine "list the first page" into a multi-hundred-MB request with no
// warning.
export const SIZE_GUARD_THRESHOLD_BYTES = 5 * 1024 * 1024 // 5 MB

// Whether Pagination.tsx's dropdown should also offer 1-9 per page, not
// just the standard options — when even 10/page (the finest standard
// option) would exceed the size guard threshold.
export function needsFineLimitOptions(avgObjSize: number): boolean {
  return avgObjSize > 0 && avgObjSize * STANDARD_LIMIT_OPTIONS[0] > SIZE_GUARD_THRESHOLD_BYTES
}

// The full set of per-page options to offer, given the collection's
// average document size.
export function limitOptions(avgObjSize: number): number[] {
  return needsFineLimitOptions(avgObjSize) ? [...FINE_LIMIT_OPTIONS, ...STANDARD_LIMIT_OPTIONS] : STANDARD_LIMIT_OPTIONS
}

// Largest offered option whose estimated transfer (avgObjSize * that
// value) still fits under the threshold — undefined if even the smallest
// available option doesn't fit, meaning there's no "reduce and load"
// suggestion to offer, only "load anyway".
export function safeLimitSuggestion(avgObjSize: number): number | undefined {
  if (avgObjSize <= 0) return undefined
  let best: number | undefined
  for (const option of limitOptions(avgObjSize)) {
    if (avgObjSize * option <= SIZE_GUARD_THRESHOLD_BYTES) best = option
  }
  return best
}
