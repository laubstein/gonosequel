# Schema & indexes

## Schema

MongoDB collections have no declared schema, so this tab infers one: it samples up to 100
documents (`$sample`) and reports, per field path, every BSON type observed and how often. A
field that's sometimes a string and sometimes missing entirely, or that switched types over
time, shows up exactly as that — multiple types listed with their counts, not silently
collapsed into one.

This inferred schema also drives the query editor's autocomplete and its preset queries.

## Indexes

Lists every index on the selected collection with its key spec and uniqueness. Create a new
single-field index by name, direction, and optional uniqueness; drop any index except `_id_`,
which MongoDB doesn't allow removing.

For usage stats across *every* index in the database — not just this collection — see the
[Tools](/features/tools) tab.
