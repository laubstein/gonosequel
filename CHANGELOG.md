# Changelog

All notable changes to this project are documented here. Format based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [0.0.14] - 2026-08-27

### Added

- **A–Z toggle** that sorts field names alphabetically, nested fields included. Documents in one
  collection need not share a field order, so the same field can otherwise sit in a different
  column position in every row. One preference covers all three views at once — the results
  table's columns, the results JSON view and the document editor — and is remembered in the
  browser.

  Array element order is never touched (it is part of an array's value, not a presentation
  detail), and Extended JSON wrappers like `{"$oid": ...}` are left intact. Table columns sort
  segment by segment, so a parent sorts immediately before its own children. In the editor the
  toggle disables itself once you have typed something, rather than discarding the edit.

## [0.0.13] - 2026-08-27

Styling foundations, and backend capabilities the UI never reached.

### Fixed

- **The table styles were leaking globally.** CSS Modules only scopes class selectors, so the
  bare `table {}` / `th, td {}` blocks in five panels were emitted unscoped and all applied to
  every table in the app, with CSS source order picking the winner. Each panel's appearance
  depended on Vite's chunk emission order rather than on its own stylesheet — every table
  inherited the index list's margin, the results grid's sticky headers and 320px truncation, and
  the history list's row hover, whether or not that made sense. They now share one scoped table
  style with opt-in modifiers, so row hover appears where rows actually do something (the index
  list had it only by accident) and truncation only where values can be long.
- Native scrollbars, `<select>` popups and the editor's own scrollbar rendered light in dark
  mode, because `color-scheme` was never set.
- The modal scrim and drop shadows were the only colours in the app blind to the theme: over a
  dark background the scrim barely separated the dialog from the page, and the shadow was
  invisible. Both are theme-aware now.

### Changed

- Each colour is declared once instead of twice. The light and dark palettes were byte-identical
  copies under two selectors, so changing a token meant changing it in both — and dark mode could
  disagree with itself depending on how it had been activated.
- The command console follows the backend's declared `command` capability instead of a hardcoded
  driver-name check, which would have left a third key-value backend without a console.
- The index usage table is sortable. Finding unused indexes is the point of it, and the zero-ops
  highlight already existed to mark them, but with a fixed order you had to hunt for the red rows.
- The connect form takes an optional **name**, shown in the connection list. Without it every
  session was labelled by its own redacted URI — least useful when several are open.
- The tab bar wraps instead of overflowing and the sidebar collapses below 900px, so a
  half-screen desktop window stays usable. This is deliberately not a mobile layout.

## [0.0.12] - 2026-08-27

Nested fields, keyboard access, and the documentation gaps.

### Fixed

- **Nested fields were unusable, at three layers that weren't the query.** MongoDB accepted
  dotted paths like `SO.nome` in filters and projections all along, but nothing in the UI helped
  you find, type, or see them:
  - Schema inference never descended into embedded documents, so `SO.nome` was invisible to
    autocomplete and the Schema tab — which also displayed the raw Go type name `bson.D` as the
    "observed type" of any subdocument.
  - The results table collapsed an embedded document into "{2 fields}", so projecting
    `{"SO.nome": 1, "SO.versao": 1}` worked and looked like it hadn't. Nested fields now get
    their own dotted columns, and the context menu stages the dotted path.
  - Typing `{SO.nome: 1}` — the natural form — was rejected outright, because a dotted key isn't
    a valid identifier and so fails even the relaxed JSON5 parse behind the auto-fix. It is now
    accepted.
- An unfixable Sort/Projection reported the text you typed back at you as the error message,
  instead of saying what was wrong with it.
- Pressing Escape on a confirmation opened over the document editor closed **both**, discarding
  unsaved edits. Only the topmost dialog responds now.

### Changed

- **The app is usable from the keyboard.** Six clickable things weren't focusable — worst, the
  results JSON view, where clicking is the only way to open a document, leaving that mode with no
  keyboard path at all. Result rows, history rows, sidebar collections, sortable headers and
  saved connections are now proper controls, sortable headers report `aria-sort`, the tab bar has
  tab roles, and the app has a visible focus indicator, which it previously had nowhere.
- The document editor, Redis key editor and connect dialog now use the shared modal shell, so
  they gain dialog roles, a focus trap and focus restoration; the connect dialog gains Escape.
- The results context menu stays inside the viewport instead of running off the edge, and its
  disabled entries explain why they're disabled.

### Documentation

- **Update mode (`updateMany`) is documented** — a destructive bulk write that previously
  appeared nowhere.
- New pages/sections for the History tab, Fix JSON, nested and dotted field paths, regex
  filters, theming and language, the connection-lost overlay, editor drafts, and the
  confirmation policy.
- Corrected statements that no longer matched the app, including the maximum page size and the
  results table's and export's own descriptions.

## [0.0.11] - 2026-08-27

Second batch from the interface audit: destructive actions and silent failures.

### Changed

- **Destructive actions now scale with blast radius.** They previously followed three different
  standards, and the most destructive one was the easiest to fire — dropping a collection made you
  type its name, while dropping an entire database was a single click, and deleting an index,
  deleting a document, deleting a Redis key, running `updateMany` and running `FLUSHALL` had no
  confirmation at all.
  - Type-the-name: drop database (new), drop collection, `FLUSHALL`/`FLUSHDB` (new).
  - Plain confirm: delete document, delete Redis key, drop index.
  - `updateMany` confirms **and first counts how many documents match**, so the number is visible
    before committing.
  - Disconnect stays unguarded: it destroys nothing, and editor drafts already persist.
- Dialogs no longer fail silently. A failed create/rename/drop used to leave the dialog sitting
  there having done nothing visible; it now stays open showing the server's reason, with what you
  typed intact. Buttons are disabled while a request is in flight, so a double click can't submit
  twice.
- Dialogs gained `role="dialog"`, `aria-modal`, a focus trap and focus restoration — none of which
  any modal in the app had. They no longer close on a click on the backdrop, which used to discard
  a carefully typed confirmation or a half-written name.

### Fixed

- **A failed fetch no longer looks like an empty result.** A failed schema fetch said "No data to
  infer schema", a failed index list rendered as "no indexes", a failed history fetch said "No
  queries yet", a failed collection list said "No collections", and a failed bookmarks fetch hid
  the saved connections section entirely.
- The sidebar's collection filter survived a database switch, silently showing a filtered subset
  of the new database.
- Clicking a saved connection and then Connect fired two competing connections; Connect on the URL
  tab was also enabled with an empty field.

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
