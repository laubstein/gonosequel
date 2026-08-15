// A curated subset of common Redis/Valkey commands, for the command-name
// autocomplete in RedisCommandRunner — the same level of help redis-cli's
// own tab-completion gives (the command name, not per-argument-position
// hints). Not exhaustive; covers the commands someone browsing with this
// tool is likely to reach for.
export interface RedisCommandHint {
  name: string
  syntax: string
}

export const REDIS_COMMANDS: RedisCommandHint[] = [
  { name: 'GET', syntax: 'GET key' },
  { name: 'SET', syntax: 'SET key value [EX seconds] [PX ms] [NX|XX]' },
  { name: 'DEL', syntax: 'DEL key [key ...]' },
  { name: 'EXISTS', syntax: 'EXISTS key [key ...]' },
  { name: 'EXPIRE', syntax: 'EXPIRE key seconds' },
  { name: 'PERSIST', syntax: 'PERSIST key' },
  { name: 'TTL', syntax: 'TTL key' },
  { name: 'TYPE', syntax: 'TYPE key' },
  { name: 'RENAME', syntax: 'RENAME key newkey' },
  { name: 'KEYS', syntax: 'KEYS pattern (blocks the server — prefer SCAN)' },
  { name: 'SCAN', syntax: 'SCAN cursor [MATCH pattern] [COUNT count]' },
  { name: 'DBSIZE', syntax: 'DBSIZE' },
  { name: 'PING', syntax: 'PING [message]' },
  { name: 'ECHO', syntax: 'ECHO message' },
  { name: 'HGET', syntax: 'HGET key field' },
  { name: 'HSET', syntax: 'HSET key field value [field value ...]' },
  { name: 'HGETALL', syntax: 'HGETALL key' },
  { name: 'HDEL', syntax: 'HDEL key field [field ...]' },
  { name: 'HKEYS', syntax: 'HKEYS key' },
  { name: 'HLEN', syntax: 'HLEN key' },
  { name: 'LPUSH', syntax: 'LPUSH key value [value ...]' },
  { name: 'RPUSH', syntax: 'RPUSH key value [value ...]' },
  { name: 'LRANGE', syntax: 'LRANGE key start stop' },
  { name: 'LLEN', syntax: 'LLEN key' },
  { name: 'LPOP', syntax: 'LPOP key' },
  { name: 'RPOP', syntax: 'RPOP key' },
  { name: 'SADD', syntax: 'SADD key member [member ...]' },
  { name: 'SMEMBERS', syntax: 'SMEMBERS key' },
  { name: 'SREM', syntax: 'SREM key member [member ...]' },
  { name: 'SCARD', syntax: 'SCARD key' },
  { name: 'ZADD', syntax: 'ZADD key score member [score member ...]' },
  { name: 'ZRANGE', syntax: 'ZRANGE key start stop [WITHSCORES]' },
  { name: 'ZSCORE', syntax: 'ZSCORE key member' },
  { name: 'ZREM', syntax: 'ZREM key member [member ...]' },
  { name: 'INCR', syntax: 'INCR key' },
  { name: 'DECR', syntax: 'DECR key' },
  { name: 'INCRBY', syntax: 'INCRBY key increment' },
  { name: 'APPEND', syntax: 'APPEND key value' },
  { name: 'STRLEN', syntax: 'STRLEN key' },
]
