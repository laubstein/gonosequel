import type { ReactNode } from 'react'
import styles from './JsonView.module.css'

// Matches, in order of preference: a quoted string (optionally followed by
// the colon that makes it an object key), true/false, null, or a number.
// Applied with matchAll rather than a naive replace so keys and string
// values just re-use one shared "string" branch, split by whether the
// match happens to end with a colon.
const TOKEN_RE =
  /("(?:\\u[a-fA-F0-9]{4}|\\.|[^"\\])*"(?:\s*:)?|\btrue\b|\bfalse\b|\bnull\b|-?\d+(?:\.\d+)?(?:[eE][+-]?\d+)?)/g

function classify(token: string): string {
  if (token.startsWith('"')) return token.trimEnd().endsWith(':') ? styles.key : styles.string
  if (token === 'true' || token === 'false') return styles.boolean
  if (token === 'null') return styles.null
  return styles.number
}

// highlightJSON tokenizes an already-formatted JSON string into plain text
// interspersed with classified <span>s, rather than building an HTML
// string and using dangerouslySetInnerHTML — simpler to reason about and
// there's no escaping step to get right.
function highlightJSON(json: string): ReactNode[] {
  const nodes: ReactNode[] = []
  let lastIndex = 0
  let key = 0

  for (const match of json.matchAll(TOKEN_RE)) {
    const token = match[0]
    const index = match.index
    if (index > lastIndex) {
      nodes.push(json.slice(lastIndex, index))
    }
    nodes.push(
      <span key={key++} className={classify(token)}>
        {token}
      </span>,
    )
    lastIndex = index + token.length
  }
  if (lastIndex < json.length) {
    nodes.push(json.slice(lastIndex))
  }
  return nodes
}

interface Props {
  value: unknown
  onClick?: () => void
}

// Renders a value as indented, syntax-highlighted JSON. `value` is
// expected to already be a plain object/array (e.g. a parsed Extended
// JSON document) — this only formats and highlights, it doesn't interpret
// $oid/$numberLong/etc wrappers specially.
export function JsonView({ value, onClick }: Props) {
  const text = JSON.stringify(value, null, 2)
  return (
    <pre className={styles.pre} onClick={onClick}>
      {highlightJSON(text)}
    </pre>
  )
}
