interface Props {
  db: string
  coll: string
}

export function Results(_props: Props) {
  return <div style={{ flex: 1, padding: 12, overflow: 'auto' }}>Resultados</div>
}
