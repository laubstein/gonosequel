# Query editor

## Find mode

The default mode. A filter (JSON object) and an optional sort, matching MongoDB's own `find()`
shape:

```json
{ "status": "active", "price": { "$gt": 100 } }
```

Autocomplete inside the filter editor is schema-aware: typing `"` inside a JSON key position
suggests the collection's actual field names, sourced from the same inferred schema shown on
the **Schema** tab.

Press <kbd>Ctrl+Enter</kbd> (or <kbd>Cmd+Enter</kbd> on macOS) to run without leaving the
keyboard.

## Aggregate mode

Switches the editor to hold a full aggregation pipeline (a JSON array) instead of a filter:

```json
[
  { "$group": { "_id": "$category", "total": { "$sum": "$price" } } },
  { "$sort": { "total": -1 } }
]
```

Sort, Explain, and export don't apply to an arbitrary pipeline with the current backend, so
they're hidden in this mode. Results from an aggregate run aren't click-to-edit — a `$group`
output document doesn't correspond to any real stored document, so there's nothing sensible to
open in the document editor. Pagination is hidden too, since `skip`/`limit` don't mean
anything for a pipeline's output; use `$limit`/`$skip` stages in the pipeline itself if needed.

Aggregate is a `POST` and is rejected entirely under `--readonly` — see [Read-only
mode](/readonly-mode) for why.

## Useful queries (presets)

The dropdown above the editor offers ready-to-run queries generated from the selected
collection's inferred schema: a handful of generic ones (all documents, sort by `_id`
descending, find by `_id`, sample 10, count all) plus a couple per field based on that field's
most common observed type — a string field gets a `find where X = ...` template, a boolean
gets `where X is true`, a number gets `sort by X descending`, a date field gets `X in the last
24 hours`, and the first string/boolean field also gets a `count grouped by X` aggregate
preset. Selecting one fills the editor and, if needed, switches mode — it doesn't run
automatically, so you can review or tweak it first.

## Explain

Runs the current filter through MongoDB's query planner at `executionStats` verbosity — the
query does execute, gathering real execution statistics alongside the chosen plan, rather than
just reporting what plan *would* be used. If the winning plan is a full collection scan
(`COLLSCAN`) anywhere in it — even nested behind a sort or projection stage — a warning banner
appears above the explain output.
