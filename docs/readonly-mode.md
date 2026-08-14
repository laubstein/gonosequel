# Read-only mode

```bash
./gonosequel --url mongodb://localhost:27017 --readonly
```

`--readonly` is enforced by server-side middleware, not by hiding buttons in the UI: every
request that isn't `GET` or `HEAD` is rejected with `403` before it reaches any handler. A
banner appears at the top of the app whenever the connected server is running in this mode, so
write actions aren't a silent trap — you know upfront, not after clicking Save and getting a
403.

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
