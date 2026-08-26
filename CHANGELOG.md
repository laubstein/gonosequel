# Changelog

All notable changes to this project are documented here. Format based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

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
