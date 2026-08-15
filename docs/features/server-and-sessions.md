# Server & sessions

## Connection info

Shows the active connection's name and URI (password redacted, both server-side and in what
reaches the browser).

## Server status

Version, host, process name, uptime, connection pool usage (current/available), and cumulative
operation counters (insert/query/update/delete/getmore/command) for the connected `mongod`.

For a Redis/Valkey connection this reports the equivalent fields from Redis's own `INFO`
command (`redis_version`, `connected_clients`, `maxclients`, ...) instead — Redis's own
operation counters don't map cleanly onto MongoDB's insert/query/update/delete/getmore/command
breakdown, so only `command` (`total_commands_processed`) is populated there; the rest read
zero rather than being guessed at.

## Disconnect

Drops the current session and brings back the connection screen. Works in both
single-connection and `--sessions` mode — see [Connecting](/connecting).

## Active connections (multi-session)

In `--sessions` mode, lists every connection the server is currently holding open. Click **Use
this connection** to switch — this only repoints which session your browser talks to (via a
request header) and reloads data under it; the connection itself is already open server-side,
so switching is instant, not a reconnect. **+ Add connection** opens a second, dismissible
connection screen to add another without losing the current one.

In single-connection mode there's always exactly one entry here, and no switcher UI beyond
Disconnect/reconnect.
