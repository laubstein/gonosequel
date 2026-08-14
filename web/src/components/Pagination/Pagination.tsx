interface Props {
  db: string
  coll: string
}

export function Pagination(_props: Props) {
  return <div style={{ padding: 12, borderTop: '1px solid var(--color-border)' }}>Paginação</div>
}
