interface Props {
  selection: { db: string; coll: string } | null
  onSelect: (db: string, coll: string) => void
}

export function Sidebar(_props: Props) {
  return <aside style={{ borderRight: '1px solid var(--color-border)', padding: 12 }}>Sidebar</aside>
}
