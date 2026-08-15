package redis

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/laubstein/gonosequel/pkg/driver"
)

// parseInfo parses Redis's INFO reply ("key:value\r\n" lines, with "#
// Section" comment lines) into a flat map.
func parseInfo(raw string) map[string]string {
	out := map[string]string{}
	for _, line := range strings.Split(raw, "\r\n") {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if k, v, ok := strings.Cut(line, ":"); ok {
			out[k] = v
		}
	}
	return out
}

// ServerStatus reports version/uptime/connections from Redis's INFO
// command. Opcounters is a best-effort, lossy mapping: Redis's own
// operation counters (total_commands_processed, keyspace hits/misses)
// don't correspond 1:1 to MongoDB's insert/query/update/delete/getmore/
// command breakdown, so most fields here are left at zero rather than
// guessed at.
func (c *Client) ServerStatus(ctx context.Context) (driver.ServerStatus, error) {
	rc, err := c.conn(0)
	if err != nil {
		return driver.ServerStatus{}, err
	}
	raw, err := rc.Info(ctx, "server", "clients", "stats").Result()
	if err != nil {
		return driver.ServerStatus{}, fmt.Errorf("info: %w", err)
	}
	info := parseInfo(raw)

	uptime, _ := strconv.ParseInt(info["uptime_in_seconds"], 10, 64)
	current, _ := strconv.ParseInt(info["connected_clients"], 10, 64)
	maxClients, _ := strconv.ParseInt(info["maxclients"], 10, 64)
	commands, _ := strconv.ParseInt(info["total_commands_processed"], 10, 64)

	process := "redis"
	if v := info["redis_mode"]; v != "" {
		process = "redis (" + v + ")"
	}

	return driver.ServerStatus{
		Version:    info["redis_version"],
		Host:       info["tcp_port"],
		Process:    process,
		UptimeSecs: uptime,
		Connections: driver.ServerConnections{
			Current:   current,
			Available: maxClients - current,
		},
		Opcounters: driver.ServerOpCounters{
			Command: commands,
		},
	}, nil
}
