---
layout: home

hero:
  name: Go NoSequel
  text: A web-based MongoDB explorer
  tagline: pgweb's interface, mongo-express's feature set, one self-contained binary.
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
