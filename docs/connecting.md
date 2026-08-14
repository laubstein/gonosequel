# Connecting

## By URL

```bash
./gonosequel --url "mongodb://user:pass@host:27017/mydb"
```

## By discrete flags

If you don't pass `--url`, the connection string is built from `--host` (default
`localhost`), `--port` (default `27017`), `--user`/`--pass`, and `--db`:

```bash
./gonosequel --host db.internal --port 27018 --user admin --pass secret --db mydb
```

`--url` always wins if both are given.

## Bookmarks

Save a named connection so you don't have to retype it. Bookmarks are TOML files under
`~/.gonosequel/bookmarks/`:

```toml
# ~/.gonosequel/bookmarks/prod.toml
url = "mongodb://user:pass@prod.example.com:27017"
```

```bash
./gonosequel --bookmark prod
```

`--bookmark` takes priority over `--url` and the discrete flags.

## Multi-session mode

By default the server connects to exactly one MongoDB at startup and stays connected to it
for its whole run. Pass `--sessions` to change that: the server starts with **no** connection,
the UI shows a connection screen on load (by URL or by picking a saved bookmark), and once
connected you can open additional connections from the **Server** tab's *"+ Add connection"*
button and switch between them without losing the others.

Saved bookmarks show up as one-click options on the connection screen in both modes. The
bookmark's URL — password included — is resolved on the server and never sent to the browser;
only picking a bookmark by name crosses the wire.

## Disconnecting

The **Server** tab has a **Disconnect** button. In single-connection mode this drops the only
connection and brings back the connection screen so you can reconnect — the same screen
`--sessions` mode shows before its first connection.
