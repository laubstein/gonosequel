// Driver values the server currently accepts via --driver / the connect
// form — keep in sync with command.SupportedDrivers in pkg/command/options.go.
export const SUPPORTED_DRIVERS = ['mongodb', 'redis', 'valkey'] as const
export type DriverName = (typeof SUPPORTED_DRIVERS)[number]

// Display names for driver ids, used both in the tab bar's "connected to"
// badge and the connection form's type selector. Falls back to the raw
// value for anything not listed here (e.g. couchdb, planned but not
// implemented), so a future backend shows up correctly even before this
// map is updated for it.
export const DRIVER_LABEL: Record<string, string> = {
  mongodb: 'MongoDB',
  redis: 'Redis',
  valkey: 'Valkey',
  couchdb: 'CouchDB',
}

// The port a driver listens on by default, when the user hasn't typed one
// — mirrors pkg/command/options.go's defaultPort map. Used only to fill
// the port field's placeholder, not as a silent fallback value.
export const DEFAULT_PORT: Record<DriverName, number> = {
  mongodb: 27017,
  redis: 6379,
  valkey: 6379,
}
