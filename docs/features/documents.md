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

## Export

**Export JSON** / **Export CSV** links next to the query editor download the *current filter's*
full result set (not just the current page) as a stream — CSV flattens nested paths into
dotted column names (e.g. `address.city`).
