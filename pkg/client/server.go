package client

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/laubstein/gonosequel/pkg/driver"
)

// ServerStatus reports version, host, uptime, connection pool usage, and
// cumulative operation counters for the connected server.
func (c *Client) ServerStatus(ctx context.Context) (driver.ServerStatus, error) {
	var raw struct {
		Version     string  `bson:"version"`
		Host        string  `bson:"host"`
		Process     string  `bson:"process"`
		Uptime      float64 `bson:"uptime"`
		Connections struct {
			Current   int64 `bson:"current"`
			Available int64 `bson:"available"`
		} `bson:"connections"`
		Opcounters struct {
			Insert  int64 `bson:"insert"`
			Query   int64 `bson:"query"`
			Update  int64 `bson:"update"`
			Delete  int64 `bson:"delete"`
			Getmore int64 `bson:"getmore"`
			Command int64 `bson:"command"`
		} `bson:"opcounters"`
	}

	cmd := bson.D{{Key: "serverStatus", Value: 1}}
	if err := c.mongo.Database("admin").RunCommand(ctx, cmd).Decode(&raw); err != nil {
		return driver.ServerStatus{}, fmt.Errorf("serverStatus: %w", err)
	}

	return driver.ServerStatus{
		Version:    raw.Version,
		Host:       raw.Host,
		Process:    raw.Process,
		UptimeSecs: int64(raw.Uptime),
		Connections: driver.ServerConnections{
			Current:   raw.Connections.Current,
			Available: raw.Connections.Available,
		},
		Opcounters: driver.ServerOpCounters{
			Insert:  raw.Opcounters.Insert,
			Query:   raw.Opcounters.Query,
			Update:  raw.Opcounters.Update,
			Delete:  raw.Opcounters.Delete,
			Getmore: raw.Opcounters.Getmore,
			Command: raw.Opcounters.Command,
		},
	}, nil
}
