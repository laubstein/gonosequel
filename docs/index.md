---
layout: home

hero:
  name: Go NoSequel
  text: A web-based NoSQL explorer
  image:
    src: /gopher.png
    alt: Go NoSequel gopher mascot
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
  - title: Redis & Valkey, too
    details: A redis-cli-like command console with autocomplete, and a per-type editor for strings/hashes/lists/sets/zsets — pick the backend with --driver.
---

## Supported databases

Go NoSequel connects to **MongoDB** and **Redis/Valkey** today. The name and the architecture
are deliberately database-agnostic — the goal is to grow into a single explorer for several
NoSQL engines, not just these.

<div class="db-support">
  <span class="db-badge db-badge--supported">MongoDB<small>supported</small></span>
  <span class="db-badge db-badge--supported">Redis/Valkey<small>supported</small></span>
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

/* VitePress's default home hero puts the image after the text (on the
   right). The gopher in gopher.png faces right in the artwork, so left
   unchanged it reads as looking away, off the edge of the page, instead
   of at the hero text next to it — reversed here (desktop widths only;
   the hero already stacks vertically below that, where left/right order
   doesn't apply) so the gopher faces the text instead. */
/* !important: VitePress's own scoped Hero component style
   (.container[data-v-xxxxxxxx]{flex-direction:row}) has the same
   specificity as a plain class-based override from here, and wins ties by
   appearing later in the built stylesheet — there's no selector-only way
   to beat it from outside the component. */
@media (min-width: 960px) {
  .VPHero .container {
    flex-direction: row-reverse !important;
  }
}
</style>
