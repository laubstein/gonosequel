// sortFieldsDeep returns value with every object's keys in alphabetical
// order, recursing into embedded documents and into the documents inside
// arrays.
//
// Two things are deliberately left alone:
//
//   - Array element order. An array's order is part of its value in
//     MongoDB, not an arbitrary presentation detail, so only the keys
//     *within* each element are sorted.
//   - Extended JSON wrappers ({"$oid": ...}, {"$regularExpression":
//     {"pattern": ..., "options": ...}}, ...). A wrapper is a scalar
//     value that happens to be spelled as an object; reordering its
//     internals would be rewriting the encoding of a value rather than
//     presenting a document's fields.
export function sortFieldsDeep(value: unknown): unknown {
  if (Array.isArray(value)) return value.map(sortFieldsDeep)
  if (value === null || typeof value !== 'object') return value

  const obj = value as Record<string, unknown>
  const keys = Object.keys(obj)
  if (keys.length === 1 && keys[0].startsWith('$')) return value

  const out: Record<string, unknown> = {}
  for (const key of [...keys].sort((a, b) => a.localeCompare(b))) {
    out[key] = sortFieldsDeep(obj[key])
  }
  return out
}

// sortPaths orders the results table's dotted column paths alphabetically,
// segment by segment, so a parent sorts immediately before its own
// children ("SO" then "SO.nome" then "SO.versao") rather than by raw
// string comparison, where a "SO2" column would land between them.
export function sortPaths(paths: string[]): string[] {
  return [...paths].sort((a, b) => {
    const as = a.split('.')
    const bs = b.split('.')
    for (let i = 0; i < Math.min(as.length, bs.length); i++) {
      const cmp = as[i].localeCompare(bs[i])
      if (cmp !== 0) return cmp
    }
    return as.length - bs.length
  })
}
