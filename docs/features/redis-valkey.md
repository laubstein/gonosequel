# Redis & Valkey

Redis and Valkey are wire-compatible and share the same driver — `--driver redis` and
`--driver valkey` behave identically, the flag just records which one you typed. Connecting
works the same way as MongoDB (see [Connecting](/connecting)); what's different is how the UI
maps Redis's own data model onto the app's database/collection/document layout, and which tabs
are available.

## Databases and collections

- **Database** = Redis's numbered database (`0`–`15`).
- **Collection** = the part of a key before its first `:`. A key with no `:` at all falls
  under a synthetic `(no prefix)` collection.

Neither is a real Redis concept — Redis has no equivalent of `CREATE DATABASE`/`CREATE
COLLECTION`, so "+ New database" and "+ New collection" don't apply here. Instead:

- The database selector lists all 16 numbered databases directly; there's no create/drop.
- The sidebar's create button becomes **+ New key** — since a "collection" only exists once a
  key with that prefix does, creating one means writing its first key.

## Editing a key

Clicking a key opens a form specific to its Redis type, instead of the raw-JSON editor
MongoDB documents use:

| Type | Editor |
|---|---|
| `string` | A single text value. |
| `hash` | Field/value rows. |
| `list` | An ordered list of values. |
| `set` | An unordered list of values. |
| `zset` | Member/score rows. |

TTL is shown (in seconds; `-1` means no expiration) but not editable from this form — set it
with `EXPIRE`/`PERSIST` in the command console below.

## Command console

Redis has no equivalent to MongoDB's filter/aggregate query editor, so the Documents tab shows
a `redis-cli`-like console instead: a multi-line box where each line is one raw command, run in
sequence (a failing line doesn't stop the rest, matching `redis-cli`'s own pipe/batch
behavior), with command-name autocomplete and results formatted like the real `redis-cli`
(`OK`, `(integer) N`, `(nil)`, numbered list replies, `(error) ...`).

Write commands are allowed here — this is a full console, not a read-only query box — so it's
subject to `--readonly` like everything else that writes (see
[Read-only mode](/readonly-mode)). Selecting part of a line before running executes only the
selection, same as the Mongo query editor's Find/Explain boxes.

## What's not available

Aggregate, Explain, Indexes, and Schema have no Redis equivalent and don't show as tabs for a
Redis/Valkey connection — the backend reports which capabilities it supports (see
[Server & sessions](/features/server-and-sessions)'s connection info), and the UI hides what
isn't there rather than showing it and failing on click. Bulk `updateMany` is MongoDB-only for
the same reason: Redis has no concept of a filter matching multiple keys at once.
