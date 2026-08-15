# Connecting

## Picking a backend

`--driver` selects MongoDB (the default), Redis, or Valkey — see [Redis & Valkey](/features/redis-valkey)
for how the UI maps onto a key-value store. Redis and Valkey are wire-compatible and share a
driver; the flag only records which name you typed.

## By URL

```bash
./gonosequel --url "mongodb://user:pass@host:27017/mydb"
./gonosequel --driver redis --url "redis://user:pass@host:6379/0"
```

## By discrete flags

If you don't pass `--url`, the connection string is built from `--host` (default
`localhost`), `--port` (default depends on `--driver` — `27017` for MongoDB, `6379` for
Redis/Valkey), `--user`/`--pass`, and `--db` (a database name for MongoDB, a numbered database
0-15 for Redis/Valkey):

```bash
./gonosequel --host db.internal --port 27018 --user admin --pass secret --db mydb
```

`--url` always wins if both are given.

## Bookmarks

Save a named connection so you don't have to retype it. Bookmarks are TOML files under
`~/.gonosequel/bookmarks/` — the backend is inferred from the URL's own scheme
(`mongodb://` or `redis://`), no separate field needed:

```toml
# ~/.gonosequel/bookmarks/prod.toml
url = "mongodb://user:pass@prod.example.com:27017"
```

```toml
# ~/.gonosequel/bookmarks/cache.toml
url = "redis://user:pass@cache.example.com:6379"
```

```bash
./gonosequel --bookmark prod
```

`--bookmark` takes priority over `--url` and the discrete flags.

## Multi-session mode

By default the server connects to exactly one backend at startup and stays connected to it for
its whole run. Pass `--sessions` to change that: the server starts with **no** connection, the
UI shows a connection screen on load, and once connected you can open additional connections
— MongoDB and Redis/Valkey alike, mixed freely — from the **Server** tab's
*"+ Add connection"* button and switch between them without losing the others.

The connection screen's Standard tab has its own database-type selector (mirroring `--driver`)
with fields that adapt to the chosen backend, plus a **Read-only connection** checkbox that
opts that one connection into read-only independent of the server-wide `--readonly` flag — see
[Read-only mode](/readonly-mode). Saved bookmarks show up as one-click options on the
connection screen in both modes; the bookmark's URL — password included — is resolved on the
server and never sent to the browser, only picking a bookmark by name crosses the wire.

## Disconnecting

The **Server** tab has a **Disconnect** button. In single-connection mode this drops the only
connection and brings back the connection screen so you can reconnect — the same screen
`--sessions` mode shows before its first connection.
