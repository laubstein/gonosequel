# Changelog

All notable changes to this project are documented here. Format based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

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
