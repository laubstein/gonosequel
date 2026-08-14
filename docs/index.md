---
layout: home

hero:
  name: Go NoSequel
  text: A web-based NoSQL explorer
  tagline: pgweb's interface, mongo-express's feature set, one self-contained binary. MongoDB today, more NoSQL databases planned.
  actions:
    - theme: brand
      text: Get started
      link: /getting-started
    - theme: alt
      text: CLI reference
      link: /cli-reference

features:
  - title: One binary, no dependencies
    details: The frontend is built and embedded into the Go executable via go:embed — nothing else to install or run in production.
  - title: Extended JSON, done right
    details: Documents round-trip through canonical Extended JSON, so editing a Long or a Decimal128 never silently turns it into a Double.
  - title: Find and Aggregate
    details: A schema-aware query editor with ready-made presets, Explain with a COLLSCAN warning, and a full aggregation pipeline mode.
  - title: Hotspot tools
    details: Collection size overview, index usage stats, and currently running operations — all read-only, all safe under --readonly.
---

## Supported databases

Go NoSequel connects to **MongoDB** today. The name and the architecture are deliberately
database-agnostic — the goal is to grow into a single explorer for several NoSQL engines, not
just this one.

<div class="db-support">
  <span class="db-badge db-badge--supported">MongoDB<small>supported</small></span>
  <span class="db-badge db-badge--planned">Redis<small>planned</small></span>
  <span class="db-badge db-badge--planned">CouchDB<small>planned</small></span>
</div>

<style>
.db-support {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  margin: 24px 0 40px;
}
.db-badge {
  display: inline-flex;
  align-items: baseline;
  gap: 8px;
  padding: 6px 14px;
  border-radius: 999px;
  font-weight: 600;
  font-size: 14px;
  border: 1px solid var(--vp-c-divider);
  background: var(--vp-c-bg-soft);
  color: var(--vp-c-text-1);
}
.db-badge small {
  font-weight: 500;
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--vp-c-text-2);
}
.db-badge--supported {
  border-color: var(--vp-c-brand-1);
}
.db-badge--supported small {
  color: var(--vp-c-brand-1);
}
.db-badge--planned {
  opacity: 0.7;
}
</style>
