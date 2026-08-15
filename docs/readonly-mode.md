# Read-only mode

```bash
./gonosequel --url mongodb://localhost:27017 --readonly
```

`--readonly` is enforced by server-side middleware, not by hiding buttons in the UI: every
request that isn't `GET` or `HEAD` is rejected with `403` before it reaches any handler. A
banner appears at the top of the app whenever the connected server is running in this mode, so
write actions aren't a silent trap — you know upfront, not after clicking Save and getting a
403.

## Per-session read-only

`--readonly` is server-wide — every session on that server is read-only, no exceptions. In
`--sessions` mode, a single connection can also be opted into read-only individually from the
connect screen's **Read-only connection** checkbox, independent of the server-wide flag (see
[Connecting](/connecting)). If the server itself was started with `--readonly`, that checkbox
is forced checked and disabled — a tampered request can't downgrade it, since the server forces
the session read-only regardless of what the connect request actually asked for.

## Redis/Valkey's command console

The [command console](/features/redis-valkey#command-console) is a full `redis-cli`-like
console, not a read-only query box — it accepts write commands like any other. It's a `POST`
request like everything else that writes, so `--readonly` (server-wide or per-session) blocks
it entirely, the same way it blocks aggregation pipelines below.

## Aggregation pipelines are blocked entirely

This is worth calling out because it's not obvious from "read-only": running an aggregation
pipeline is a `POST` request, and `--readonly` rejects it like any other `POST` — **even a
pipeline that only reads**, such as `[{ "$match": { ... } }, { "$limit": 10 }]`.

This is deliberate. Unlike a find filter, a pipeline can itself write, via `$out` or `$merge`
stages. There's no cheap, reliable way to tell a read-only pipeline from one that isn't short
of inspecting every stage — so `--readonly` blocks aggregate entirely rather than trying to
allow "safe" pipelines through. If you need to run aggregations against a server you don't
want written to, use a MongoDB user with read-only privileges instead of relying on
`--readonly` to distinguish query intent.

## What's unaffected

Everything read-only continues to work normally: browsing documents, running find queries,
viewing schema/indexes/Tools/Server tabs, and Explain (which itself only plans and reports,
never writes).
