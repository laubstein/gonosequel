interface Props {
  db: string
  coll: string
  standalone?: boolean
}

export function QueryEditor(_props: Props) {
  return <div style={{ padding: 12, borderBottom: '1px solid var(--color-border)' }}>Editor de query</div>
}
