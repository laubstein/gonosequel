import type { Preset, SchemaField } from '../../types'

const MAX_FIELD_PRESETS = 6

function dominantType(field: SchemaField): string {
  return field.types.reduce((a, b) => (b.count > a.count ? b : a)).type
}

// buildPresets turns a collection's inferred schema (see
// pkg/client/schema.go's $sample-based inference, surfaced here through
// useCollectionSchema) into a set of ready-to-run queries: a handful of
// generic ones that don't need any field knowledge, plus a couple per
// field for the first few non-_id, non-nested fields, chosen by that
// field's most common observed type. Capped at MAX_FIELD_PRESETS fields
// so a wide collection doesn't produce an unusable, giant dropdown.
export function buildPresets(schema: SchemaField[]): Preset[] {
  const presets: Preset[] = [
    { labelKey: 'presets.allDocuments', mode: 'find', filter: '{}' },
    { labelKey: 'presets.sortByIdDesc', mode: 'find', filter: '{}', sort: '{ "_id": -1 }' },
    { labelKey: 'presets.findById', mode: 'find', filter: '{ "_id": { "$oid": "" } }' },
    { labelKey: 'presets.sample10', mode: 'aggregate', pipeline: '[\n  { "$sample": { "size": 10 } }\n]' },
    { labelKey: 'presets.countAll', mode: 'aggregate', pipeline: '[\n  { "$count": "total" }\n]' },
  ]

  const fields = schema.filter((f) => f.path !== '_id' && !f.path.includes('.')).slice(0, MAX_FIELD_PRESETS)

  for (const field of fields) {
    switch (dominantType(field)) {
      case 'string':
        presets.push({
          labelKey: 'presets.findFieldEquals',
          labelParams: { field: field.path },
          mode: 'find',
          filter: `{ "${field.path}": "" }`,
        })
        break
      case 'bool':
        presets.push({
          labelKey: 'presets.whereFieldTrue',
          labelParams: { field: field.path },
          mode: 'find',
          filter: `{ "${field.path}": true }`,
        })
        break
      case 'int':
      case 'long':
      case 'double':
      case 'decimal':
        presets.push({
          labelKey: 'presets.sortByFieldDesc',
          labelParams: { field: field.path },
          mode: 'find',
          filter: '{}',
          sort: `{ "${field.path}": -1 }`,
        })
        break
      case 'date': {
        const since = new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString()
        presets.push({
          labelKey: 'presets.fieldLast24h',
          labelParams: { field: field.path },
          mode: 'find',
          filter: `{ "${field.path}": { "$gte": { "$date": "${since}" } } }`,
        })
        break
      }
    }
  }

  const groupCandidate = fields.find((f) => dominantType(f) === 'string' || dominantType(f) === 'bool')
  if (groupCandidate) {
    presets.push({
      labelKey: 'presets.countGroupedByField',
      labelParams: { field: groupCandidate.path },
      mode: 'aggregate',
      pipeline: `[\n  { "$group": { "_id": "$${groupCandidate.path}", "count": { "$sum": 1 } } },\n  { "$sort": { "count": -1 } }\n]`,
    })
  }

  return presets
}

// buildRedisPresets is buildPresets' Redis/Valkey equivalent. Every preset
// above assumes MongoDB: ObjectId/_id-based lookups, and several run in
// "aggregate" mode — a capability Redis doesn't have (see
// pkg/redis/aggregate.go), so applying one would 501. Redis's Find only
// understands one filter shape at all, "$keyPattern" (see
// pkg/redis/documents.go), so there is no per-field preset generation the
// way buildPresets does from a sampled schema — there's nothing to sample
// that would be a valid filter target.
export function buildRedisPresets(coll: string): Preset[] {
  return [
    { labelKey: 'presets.redisAllKeys', mode: 'find', filter: '{}' },
    {
      labelKey: 'presets.redisKeyPattern',
      labelParams: { pattern: `${coll}:*` },
      mode: 'find',
      filter: `{ "$keyPattern": "${coll}:*" }`,
    },
  ]
}
