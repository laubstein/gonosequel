# Contributing

This page is a pointer, not a duplicate — the authoritative contributor documentation lives in
the repository itself, next to the code it describes, so it can't drift out of sync the way a
copy here would:

- **`AGENTS.md`** — package layout, fixed dependency versions, Effective Go conventions used
  throughout, and the invariants that matter most if you're touching `pkg/client` or `pkg/api`
  (how Extended JSON is handled, how `_id` is encoded for routes, how `--readonly` is enforced).
- **`PLAN.md`** — the original design rationale and phased build order, for the "why" behind
  decisions that aren't obvious from the code alone.

## Quick reference

```bash
make lint          # gofmt, go vet, staticcheck, errcheck
make test           # unit + integration tests, starts/stops MongoDB via Docker itself
make test-short      # unit tests only, no Docker required
```

Every backend change that touches `pkg/client` or `pkg/api` should have integration test
coverage against a real MongoDB (via `testcontainers-go`, already wired into `TestMain` in
both packages) — this project has caught real bugs that unit tests with mocks would have
missed, precisely because they only surface against a live server.
