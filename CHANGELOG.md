# Changelog

All notable changes to this project are documented here. Format based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [0.0.10] - 2026-08-27

First batch from a full interface audit: the things that were simply wrong.

### Fixed

- **Export now exports what's on screen.** It received only the filter, and the *draft* filter at
  that — so hidden fields reappeared, sort order was lost, and exporting mid-edit produced a
  different set of documents than the one being looked at. It now sends the applied query's
  filter, sort and projection.
- **Export works in `--sessions` mode.** The download was a plain browser navigation, which
  cannot carry the `X-Session-Id` header, so every export failed with "no active connection for
  this session". Downloads are now authorized by a single-use, 30-second ticket fetched over a
  normal request. The direct, header-authenticated route is unchanged for scripted use.
- **A failed document load could overwrite the real document.** Neither the document editor nor
  the Redis key editor handled the fetch failing: the editor sat on its initial empty state,
  looking like a document that had loaded empty, and saving wrote that emptiness over the stored
  value. Both now show the error and disable Save.
- **The index editor no longer loses an index when a recreate fails.** Editing is drop-then-create
  (MongoDB cannot alter or rename an index in place); if the create failed, the original was
  already gone with nothing said about it. The original is now restored, and the message
  distinguishes "restored unchanged" from "the index really is missing".
- **Explain described a query you never ran** — it took only the filter, so with a sort or
  projection set, both the plan and the collection-scan warning were about something else.
- Redis commands typed in the console left the whole UI stale: a `DEL` or `FLUSHDB` never
  refreshed the key table or the collection list.
- Every database created from the UI was born with a stray collection called `_init`. The new
  database dialog now asks what the first collection should be called.
- The results grid showed the previous page's rows with no indication they were stale while the
  next page loaded, and claimed a count of 0 when paging past the end.
- An estimated document count was presented as an exact page count ("page 3 of 412" from a
  guess); it is now marked as an estimate, and an approximate total no longer decides that
  there is no next page.
- Removing one projection chip wiped the entire projection whenever the text wasn't strict JSON.
- A trailing space in the projection pinned Run in its "unapplied changes" state forever.
- The theme tooltip showed the raw setting name, so the Portuguese UI read "Tema: system".
- Every downloaded document was named after its collection, so saving several produced files
  that overwrote each other.
- Dropping an index and changing a TTL failed silently; the TTL field also discarded what was
  typed when the request failed.

### Added

- **A Redis key's TTL can be set.** It was displayed but read-only, and no write path read the
  field back — there was no way to set an expiry at all. Note every Redis write recreates the
  key, so a key's expiry is now preserved across saves instead of being silently dropped.
- The preset dropdown clears when switching editor modes, instead of labelling contents it no
  longer describes.

## [0.0.9] - 2026-08-27

### Added

- Results table context menu gained a third action, **Exclude value** — the negative of "Filter by
  value", staging `{ "field": { "$ne": <value> } }` to drop rows carrying that value.
- Sort and Projection now have visible labels, and both accept JS-object-literal shorthand: typing
  `{cpu: 1}` and pressing Run rewrites it to `{"cpu": 1}` automatically, with no "Fix JSON" click.

### Changed

- The document editor shows and downloads numbers unwrapped — a field stored as int32 reads as
  `"cpu": 1` instead of `{"$numberInt": "1"}`, so a downloaded document is the plain JSON a script
  would expect. Non-numeric wrappers (`$oid`, `$date`, …) are untouched, and the document is still
  fetched in canonical Extended JSON, so the exact BSON type survives the trip.
- Saving a document warns first when a value can't round-trip exactly as displayed: a `Long`
  beyond JavaScript's safe integer range, or any `Decimal128` (plain JSON has no literal for it).
  The warning names the fields and explains that rewriting one in its wrapped form
  (`{"$numberLong": "..."}`) keeps it exact. Everything else saves unchanged, as before.

## [0.0.8] - 2026-08-27

### Added

- Results table context menu: right-click a value for **Filter by value** (fills the filter
  editor without running) or **Hide field** (excludes that field from the query via a MongoDB
  projection, not just from the display). Disabled for cells with no single value to filter by
  (an array or a multi-field subdocument).
- Projection field in the find form, in the same raw-JSON style as Sort — the direct way to
  include or exclude fields before they're ever fetched, not just after seeing them once via
  "Hide field". Every exclusion currently in it shows as a removable chip above Run.
- Run button shows a highlighted outline (and a tooltip) whenever the prepared filter/sort/
  projection no longer match what's actually shown in the results table.
- Per-page dropdown offers 1 through 9, in addition to the usual 10/25/50/100/250, when a
  collection's average document size is large enough that even 10/page would risk a large
  transfer — and the size guard's "reduce and load" suggestion can recommend one of them.
- Tools tab's Collections overview table can now be sorted by any column (click the header;
  click again to reverse) — Collection name stays the default.

### Fixed

- The index editor's "Editing ..." form (and its warning banner) stayed open after switching to
  a different collection, even though the index being edited didn't exist there.

## [0.0.7] - 2026-08-26

### Added

- Size guard for large documents: the Documents tab now checks the collection's average
  document size (`avgObjSize`, already shown in the sidebar and Tools tab) against the current
  page size before fetching. If the estimated transfer would exceed ~5 MB, the page doesn't load
  automatically — a warning shows the average and estimated size, with "Load anyway" or (when a
  smaller page size would fit) "Show N per page instead". Collections with normal-sized
  documents are unaffected; opening a single document and the Aggregate pipeline's output are
  unaffected too.
- Document editor: a **Download** button saves the currently shown document (Extended JSON, as
  displayed) as a `.json` file, client-side — no server round-trip.

## [0.0.6] - 2026-08-26

### Changed

- Tools tab: "Collections overview" now sorts alphabetically by collection name, and "Index
  usage" alphabetically by collection then index name — both previously sorted by a size/usage
  metric (storage size descending, operation count ascending).

## [0.0.5] - 2026-08-26

### Added

- Databases and collections are now listed alphabetically (case-insensitive) — this also fixes
  Redis/Valkey's collection list, previously built from a Go map and shown in nondeterministic
  order that reshuffled on every refresh.
- Index details: sparse, TTL (`expireAfterSeconds`), and partial filter expression are now shown
  per index, and the create-index form supports compound indexes (any number of fields) plus
  sparse/TTL/partial filter options — previously limited to a single field.
- Index editing: a TTL index's `expireAfterSeconds` can now be changed in place (MongoDB's
  `collMod`, the one index property it allows altering without recreating the index). Any other
  change reuses the create form, pre-filled with the index's current spec, and drops + recreates
  the index on save — the UI says so explicitly.
- The Collections overview (Tools tab) now also shows average document size.
- Sidebar's per-collection quick stats now also show storage size, index size, and average
  document size (previously only document count, data size, and index count).

### Fixed

- The Tools tab showed "No collections"/"No indexes" even when data existed, whenever a single
  collection failed to report stats (e.g. a view) — one bad collection no longer hides every
  other collection's numbers, and a genuine request failure now shows as a distinct error instead
  of looking identical to an empty database.
- Compound index creation lost field order: the API accepted index keys as a JSON object, which
  neither Go's `map[string]int` nor JSON itself reliably preserves the order of — order matters
  for a compound index's usability. Keys now travel as an ordered array.
- The index list's "Fields" column rendered as `0: [object Object]` — the API and frontend
  disagreed on whether an index's key spec was a JSON object or an ordered array.

## [0.0.4] - 2026-08-26

### Fixed

- Startup panic ("decode SHA256 password: illegal base64 data at input byte 1") whenever
  `--auth-user`/`--auth-pass` were set: fiber v3.5.0's `basicauth.Config.Users` holds a *hash* of
  the password, not the plaintext — the plaintext was being passed straight through, which fiber
  then tried (and failed) to decode as a hex/base64 SHA-256 digest. Basic auth now hashes the
  configured password with SHA-256 before handing it to fiber.

## [0.0.3] - 2026-08-26

### Added

- Startup banner: gonosequel now always prints the effective configuration on startup (driver,
  connection target, bind address, whether sessions mode/readonly/basic auth/TLS/the session
  secret are on). Any credential-shaped value — backend password, basic auth password, session
  secret — is masked as `****`, never printed in the clear.
- `--verbose` (`GNS_VERBOSE`/`ME_VERBOSE`, default `false`): prints extra diagnostic logging
  beyond the banner — bookmarks directory, the initial connect lifecycle in main.go, and session
  connect/disconnect events in `--sessions` mode.

## [0.0.2] - 2026-08-25

### Added

- HTTPS support: `--tls-cert`/`--tls-key` serve the UI over TLS instead of plain HTTP, enabled
  by the mere presence of both. `--tls-enabled` (default `true`) lets an imported
  `ME_CONFIG_SITE_SSL_ENABLED=false` disable TLS while keeping the cert/key paths configured.
- `--session-secret`: HMAC-signs the session ID handed out by `/api/connect` in `--sessions`
  mode, so a client can't forge or guess another session's ID. Unset keeps today's unsigned,
  opaque session IDs — see [Server & sessions](docs/features/server-and-sessions.md).
- `--auth-enabled` (default `true`) mirrors `--tls-enabled` for basic auth: lets an imported
  `ME_CONFIG_BASICAUTH_ENABLED=false` disable it while `--auth-user`/`--auth-pass` stay
  configured.
- Basic auth (`--auth-user`/`--auth-pass`, already implemented) now also recognizes
  mongo-express's own real env var names as a compat fallback tier
  (`ME_CONFIG_BASICAUTH_USERNAME`/`_PASSWORD`), same convention used for the new TLS and
  session-secret flags (`ME_CONFIG_SITE_SSL_CRT_PATH`/`_KEY_PATH`,
  `ME_CONFIG_SITE_SESSIONSECRET`, `ME_CONFIG_SITE_SSL_ENABLED`, `ME_CONFIG_BASICAUTH_ENABLED`).

## [0.0.1] - 2026-08-25

Initial tagged release. MongoDB and Redis/Valkey explorer with a pgweb-style UI, single-session
and `--sessions` (multi-connection) modes, read-only mode, saved bookmarks, JSON/CSV export, and
a GitHub Actions release + Pages pipeline.
