# Tools

Pick any database in the sidebar (no collection needs to be open) to see the **Tools** tab —
three read-only panels for spotting problem hotspots without hand-writing queries against
`admin`.

## Collections overview

Every collection in the selected database, with document count, data size, storage size,
index size, average document size, and index count — sorted by storage size descending, so
bloated collections (storage size far exceeding data size) surface first without clicking
anything. A collection that can't report stats (a view, for instance, or one dropped mid-scan)
is skipped rather than hiding every other collection's numbers behind it.

## Index usage

`$indexStats` for every index in every collection of the database, sorted by operation count
ascending. An index with **0** operations (highlighted in red) hasn't been used by any query
since the server last restarted — a real candidate for dropping, not a guess. As with the
collections overview, a collection `$indexStats` can't run against is skipped, not fatal to the
whole tab.

A failed request (a network error, an unreachable server) shows as a distinct error message —
never silently as "no collections"/"no indexes", which would look identical to an empty
database.

## Running operations

Active server operations that have been running for at least one second, refreshed every five
seconds. This is instance-wide, not scoped to the open database — each operation's namespace
shows which database and collection it belongs to.

## What's not here (yet)

Two more powerful diagnostics were deliberately left out:

- **Slow query log via the database profiler** — enabling it is a write, which would be the
  one exception to `--readonly` covering every dangerous action without exception, and it adds
  performance overhead to the monitored server.
- **`$queryStats`** — aggregated per-query-shape statistics, MongoDB 7.0+ only. The closest
  thing to a `pg_stat_statements` equivalent, and the natural next step for this tab, but it
  needs server-version detection this project doesn't have yet.
