# Schema & indexes

## Schema

MongoDB collections have no declared schema, so this tab infers one: it samples up to 100
documents (`$sample`) and reports, per field path, every BSON type observed and how often. A
field that's sometimes a string and sometimes missing entirely, or that switched types over
time, shows up exactly as that — multiple types listed with their counts, not silently
collapsed into one.

This inferred schema also drives the query editor's autocomplete and its preset queries.

## Indexes

Lists every index on the selected collection, in a table that expands per-row (click the name)
to show its full spec: keys, uniqueness, sparse, TTL (`expireAfterSeconds`), and partial filter
expression. Create a compound index (any number of fields, each with its own direction),
optionally unique, sparse, TTL, or filtered by a partial expression — MongoDB itself rejects
combining `sparse` with a partial filter expression on the same index, so that error surfaces as
a normal create failure rather than being pre-validated client-side. Drop any index except
`_id_`, which MongoDB doesn't allow removing.

**Editing an existing index**: MongoDB only allows changing one property of an existing index in
place — a TTL index's `expireAfterSeconds` (via `collMod`), edited directly from the expanded
row. Everything else about an index (keys, unique, sparse, partial filter) is immutable once
created; **Edit** pre-fills the create form with the index's current spec, and saving drops the
old index and creates the new one — the form says so explicitly, since that's a brief window
where the index doesn't exist.

For usage stats across *every* index in the database — not just this collection — see the
[Tools](/features/tools) tab.
