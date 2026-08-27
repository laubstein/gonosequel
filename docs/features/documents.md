# Documents

Select a database and a collection in the sidebar to open the **Documents** tab.

This page describes MongoDB's Documents tab. For a Redis/Valkey connection this tab shows a
key browser plus a per-type value editor and command console instead — see
[Redis & Valkey](/features/redis-valkey).

## Table and JSON views

Results toggle between a flattened table (one column per top-level field, values summarized
for readability) and a raw JSON view with syntax highlighting. Nested documents don't fit a
table cleanly, so the JSON view is where you'd inspect those.

Clicking a row (in either view) opens the document editor.

## Extended JSON, and why it matters

BSON doesn't map onto JSON without loss — an `ObjectId`, a `Date`, a `Decimal128`, and a 64-bit
integer would otherwise all collapse into ambiguous plain numbers or strings. Documents are
therefore shown and edited as MongoDB's Extended JSON: a document listed in the table or JSON
view uses the relaxed form (readable — dates as ISO strings, not `{"$date": ...}` wrappers),
while opening a document *for editing* fetches it fresh in canonical form (every type spelled
out explicitly). This means a save round-trip can't silently turn a `Long` into a `Double`, or
a `Decimal128` into a plain float.

When editing, the textarea expects the same Extended JSON syntax you see — `{"$oid": "..."}`
for ObjectIds, `{"$numberLong": "..."}` for 64-bit integers, and so on.

## Creating and deleting

**+ New document** opens the same editor, empty, in insert mode. Deleting is a button inside
the document editor itself, not a separate action in the table — this avoids an accidental
click deleting something.

## Downloading a document

The document editor has a **Download** button that saves the document currently shown (in
Extended JSON, exactly as displayed — canonical form when editing an existing document) as a
`.json` file, entirely client-side. Handy for a document too large to comfortably read inline,
or to keep a copy before editing it.

## Pagination and the large-document size guard

The page size defaults to 50 documents per page (up to 1000 via the per-page dropdown). For a
collection whose documents are small this is a non-issue, but some collections average tens of
megabytes *per document* — a page of 50 of those could mean transferring hundreds of megabytes
in one request without any warning.

To avoid that, the Documents tab checks the collection's average document size (the same
`avgObjSize` already shown in the sidebar's stats and in the Tools tab's collections overview)
against the requested page size. If the estimated transfer for the current page size would
exceed roughly 5 MB, the page doesn't load automatically — instead you get a warning naming the
average document size and the estimated transfer, with two ways forward: **Load anyway**, or, if
a smaller page size would fit comfortably, **Show N per page instead**, which switches to that
size and loads immediately (now safely under the threshold). A collection with normal-sized
documents never sees this — pages load exactly as before.

This only applies to the Documents tab's own paginated list; opening a single document in the
editor (an explicit, one-at-a-time action) and the Aggregate pipeline's output (which hides
pagination entirely, since skip/limit don't mean anything for a pipeline — see
[Query editor](/features/query-editor)) are both unaffected.

## Export

**Export JSON** / **Export CSV** links next to the query editor download the *current filter's*
full result set (not just the current page) as a stream — CSV flattens nested paths into
dotted column names (e.g. `address.city`).
