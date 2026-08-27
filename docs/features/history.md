# Query history

The **History** tab lists the queries run in this session, with the database, collection and
filter for each. Clicking a row loads that query back into the editor for the collection it came
from, switching the selection if needed — it fills the editor rather than running immediately,
so you can adjust it first.

Only filtered queries are recorded. Paging through a collection with no filter would otherwise
drown the list in entries nobody wants to replay.

The history is per session and lives in memory on the server: it is gone when the server stops,
and it isn't shared between connections in `--sessions` mode. Nothing is written to disk.

The tab is available for Redis/Valkey connections too, where it records the same thing for the
key browser's own queries.

::: tip
This is separate from the editor's *draft* persistence, which is what keeps the query you were
typing when you switch collections or reload the page. Drafts live in the browser
(`localStorage`), survive a refresh and a reconnect, and are per collection — see
[Documents](/features/documents).
:::
