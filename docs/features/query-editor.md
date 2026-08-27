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

Regular expressions work as they do in the shell, in either form MongoDB accepts:

```json
{ "nome": { "$regex": "^srv", "$options": "i" } }
```

Press <kbd>Ctrl+Enter</kbd> (or <kbd>Cmd+Enter</kbd> on macOS) to run without leaving the
keyboard.

## Projection — choosing which fields come back

A **Projection** field sits below Sort, taking the same raw MongoDB shape as `find()`'s own
projection argument — exclude fields with `{ "field": 0 }`, or switch to include-only with
`{ "field": 1, "otherField": 1 }` (MongoDB's own rule applies: don't mix `1`s and `0`s in the
same projection, except `_id` can always be excluded alongside inclusions). Typing here is the
most direct way to keep a large or irrelevant field from ever being fetched in the first place —
unlike the context-menu route below, it doesn't require the field to have already come back once.

Every exclusion currently in the Projection field also shows as a removable chip above the Run
button; removing one (✕) edits the Projection text to drop it. This box, and the chip row,
share the same state — editing one updates the other.

Sort and Projection both accept JS-object-literal shorthand: type `{cpu: 1}` and Run rewrites it
to `{"cpu": 1}` for you. Unlike the filter editor's "Fix JSON" button, this happens
automatically on Run, with no extra click.

## Filtering and hiding fields from the results table

Right-click a value in the results table (not the JSON view) for three actions:

- **Filter by value** — replaces the filter editor's text with `{ "field": <that value> }`. It
  only edits the filter, it does **not** run the query — Run gets a highlighted outline (and a
  tooltip explaining why) whenever the prepared filter, sort, or projection no longer match
  what's actually shown, so it's always clear when the results on screen are stale relative to
  what's staged. Only available for a cell holding a single plain value (a string, number,
  boolean, `null`, or an ObjectId/Date/Decimal128/Long) — a nested subdocument with several
  fields, or an array, has no single value to filter by, so the option is disabled there.
- **Exclude value** — the negative of the above: stages
  `{ "field": { "$ne": <that value> } }`, to drop the rows carrying that value out of the
  results. Same staging behavior and the same restriction on which cells it works for.
- **Hide field** — adds `"field": 0` to the Projection field above (creating it if empty),
  excluding that field from the query itself, not just from the display. Like the other two,
  this only stages the change — Run applies it.

## Update mode — editing many documents at once

The third mode turns the editor into a bulk write: the top box is a filter, the second is a
MongoDB update document, and Run applies that update to **every** matching document via
`updateMany`.

```json
{ "$set": { "status": "archived" } }
```

Because this writes to an unbounded number of documents from the same Run button that
harmlessly runs a find one mode-toggle away, it asks first — and the confirmation tells you how
many documents currently match, so the blast radius is visible before you commit. There is no
undo.

The result line afterwards reports how many documents matched and how many were actually
modified; those differ when a document already held the values being set.

Update mode is MongoDB-only, and rejected entirely under `--readonly`.

## Fix JSON

MongoDB's shell accepts JavaScript object syntax, so `{status: 'active'}` is what most people
type from memory — but it isn't valid JSON, which is what the filter box expects. When the
editor sees input that parses as a JS object literal but not as JSON (unquoted keys, single
quotes, trailing commas), it offers **Fix JSON** to rewrite it properly rather than making you
hunt for the missing quotes.

The Sort and Projection fields don't need the button: they repair the same input automatically
when you press Run, including dotted paths like `{SO.nome: 1}` that even JSON5 rejects.

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

## Nested fields

MongoDB addresses a field inside an embedded document by its dotted path, and every box here
takes that path directly — filter, sort and projection alike:

```json
{ "SO.nome": "Ubuntu" }
```

```json
{ "SO.nome": 1, "SO.versao": 1 }
```

The results table shows nested fields as their own `SO.nome` / `SO.versao` columns rather than
collapsing the parent into "{2 fields}", so a projection like the one above shows exactly the
fields it selected. Right-clicking one of those cells stages the dotted path too, so "Filter by
value" and "Hide field" work on a nested field the same way they do on a top-level one. The
Schema tab and autocomplete list the dotted paths as well.

Deeply nested structures are spread only a few levels into columns, and the column count is
capped — the JSON view is the way to read a large document in full.

## Explain

Runs the current query through MongoDB's query planner at `executionStats` verbosity — the
query does execute, gathering real execution statistics alongside the chosen plan, rather than
just reporting what plan *would* be used. Sort and projection are part of what's explained, not
just the filter: a sort can select a different index, or add a blocking in-memory sort stage
that explaining the filter alone would never show. If the winning plan is a full collection scan
(`COLLSCAN`) anywhere in it — even nested behind a sort or projection stage — a warning banner
appears above the explain output.
