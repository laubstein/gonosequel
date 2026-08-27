# Documents

Select a database and a collection in the sidebar to open the **Documents** tab.

This page describes MongoDB's Documents tab. For a Redis/Valkey connection this tab shows a
key browser plus a per-type value editor and command console instead — see
[Redis & Valkey](/features/redis-valkey).

## Table and JSON views

Results toggle between a flattened table and a raw JSON view with syntax highlighting. The
table gives each field its own column, descending a few levels into embedded documents so a
nested field appears as `SO.nome` rather than collapsing its parent into "{2 fields}". Arrays
are summarized rather than spread, and the column count is capped — the JSON view is where to
read a large or deeply nested document in full.

Clicking a row (in either view) opens the document editor; both are reachable from the keyboard.

**A–Z** sorts field names alphabetically, nested fields included — useful for a wide or
inconsistently-ordered collection, where the same field can otherwise sit in a different place in
every document. It applies to all three views at once (table columns, JSON view and the document
editor) and is remembered in the browser. Array element order is never touched: an array's order
is part of its value, not a presentation detail.

In the editor the toggle rebuilds the text from the loaded document, so it turns itself off once
you have typed something rather than discarding the edit — reopen the document to sort it. Note
that saving a sorted document stores the fields in that order, since a save replaces the whole
document.

## Extended JSON, and why it matters

BSON doesn't map onto JSON without loss — an `ObjectId`, a `Date`, a `Decimal128`, and a 64-bit
integer would otherwise all collapse into ambiguous plain numbers or strings. Documents are
therefore shown and edited as MongoDB's Extended JSON: a document listed in the table or JSON
view uses the relaxed form (readable — dates as ISO strings, not `{"$date": ...}` wrappers),
while opening a document *for editing* fetches it fresh in canonical form, so the exact BSON
type survives the trip.

Numbers, however, are shown **unwrapped** in the editor: a field stored as an `int32` reads as
`"cpu": 1`, not `"cpu": {"$numberInt": "1"}`. The document you see and download is the plain
JSON a downstream script would expect. Non-numeric wrappers stay as they are — `{"$oid": "..."}`
for ObjectIds, `{"$date": "..."}` for dates — and the editor accepts that same syntax back.

Two numeric types can't survive that unwrapping exactly: a `Long` larger than JavaScript's safe
integer range, and any `Decimal128` (plain JSON simply has no literal that means "Decimal128").
Saving one of those as displayed would change its value or its type. The editor detects this and
**warns before saving**, naming the affected fields. To keep such a value exactly as stored,
rewrite the field in its wrapped form — `{"$numberLong": "..."}` or `{"$numberDecimal": "..."}` —
before saving; the editor accepts wrapped and bare numbers side by side in the same document.

## Creating and deleting

**+ New document** opens the same editor, empty, in insert mode. Deleting is a button inside
the document editor itself, not a separate action in the table — this avoids an accidental
click deleting something.

## Downloading a document

The document editor has a **Download** button that saves the document currently shown — exactly
as displayed, numbers unwrapped — as a `.json` file, entirely client-side. Handy for a document
too large to comfortably read inline, to keep a copy before editing it, or to feed straight into
a script that expects ordinary JSON numbers.

## Pagination and the large-document size guard

The page size defaults to 50 documents per page (up to 250 via the per-page dropdown). For a
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

When documents are large enough that even the smallest normal page size (10 per page) would
still risk a large transfer, the per-page dropdown itself grows to also offer 1 through 9 —
letting you go as low as one document per page instead of being stuck at 10. Normal collections
keep the usual 10/25/50/100/250 options only.

This only applies to the Documents tab's own paginated list; opening a single document in the
editor (an explicit, one-at-a-time action) and the Aggregate pipeline's output (which hides
pagination entirely, since skip/limit don't mean anything for a pipeline — see
[Query editor](/features/query-editor)) are both unaffected.

## Export

**Export JSON** / **Export CSV** next to the query editor download the full result set of the
query currently shown — filter, sort and projection included, not just the current page — as a
stream. CSV flattens nested paths into dotted column names (e.g. `address.city`).

The export reflects the query that produced the results on screen, not unapplied edits sitting
in the editor above: press Run first if you want those included.

## Deleting, and what asks first

Actions that destroy data confirm first, scaled to how much they can destroy:

| Action | What it asks |
|---|---|
| Delete a database, delete a collection, Redis `FLUSHALL`/`FLUSHDB` | type the name to confirm |
| Delete a document, delete a Redis key, delete an index | a plain confirmation |
| `updateMany` (Update mode) | a confirmation showing how many documents match |
| Disconnect | nothing — it destroys nothing, and drafts are kept |

A failed action keeps its dialog open and shows why, rather than closing as though it had
worked.

## Editor drafts

Whatever is typed in the query editor (or the Redis command runner) is kept per collection, in
the browser, and comes back when you return to that collection — including after a page reload
or a reconnect. It is deliberately not tied to the session, so reconnecting to the same
database doesn't lose the query you were in the middle of writing.

Drafts are what you typed, not what you ran; the [History](/features/history) tab is the record
of queries actually executed.
