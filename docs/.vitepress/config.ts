import { defineConfig } from 'vitepress'

export default defineConfig({
  title: 'Go NoSequel',
  description: 'A web-based NoSQL explorer (MongoDB, Redis/Valkey) — usage and configuration',
  // Served under /doc by the Go binary (pkg/api/assets.go), embedded via
  // go:embed alongside the main React app served at /. The GitHub Pages
  // workflow (.github/workflows/pages.yml) builds this same site a second
  // time with DOCS_BASE set to /<repo>/ instead, since a project Pages
  // site is served from a subpath rather than the domain root.
  base: process.env.DOCS_BASE || '/doc/',
  // Left off deliberately: cleanUrls relies on server-side rewriting
  // (e.g. Netlify/Vercel's "no extension -> try adding .html"), which
  // Fiber's static.New here doesn't do — it serves by exact path and
  // falls back to index.html on any miss. With cleanUrls on, a link to
  // /doc/getting-started (no .html) would silently render the docs
  // homepage instead of 404ing or finding the real page. Links keep
  // their .html suffix instead, which the static handler resolves
  // exactly.

  themeConfig: {
    // Files under docs/public/ are served at the site root and copied
    // verbatim into the build — VitePress rewrites this path with `base`
    // (/doc/) itself, same as it does for page links, so this stays
    // '/gopher.png' regardless of where the docs are actually mounted.
    logo: '/gopher.png',
    nav: [
      { text: 'Guide', link: '/getting-started' },
      { text: 'CLI reference', link: '/cli-reference' },
      { text: 'Features', link: '/features/documents' },
    ],

    sidebar: [
      {
        text: 'Guide',
        items: [
          { text: 'Getting started', link: '/getting-started' },
          { text: 'Connecting', link: '/connecting' },
          { text: 'CLI reference', link: '/cli-reference' },
          { text: 'Read-only mode', link: '/readonly-mode' },
        ],
      },
      {
        text: 'Features',
        items: [
          { text: 'Documents', link: '/features/documents' },
          { text: 'Query editor', link: '/features/query-editor' },
          { text: 'Schema & indexes', link: '/features/schema-and-indexes' },
          { text: 'Tools', link: '/features/tools' },
          { text: 'Server & sessions', link: '/features/server-and-sessions' },
          { text: 'Redis & Valkey', link: '/features/redis-valkey' },
        ],
      },
      {
        text: 'More',
        items: [{ text: 'Contributing', link: '/contributing' }],
      },
    ],
  },
})
