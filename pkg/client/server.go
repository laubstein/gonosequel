package client

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// ServerConnections reports the server's current connection pool usage.
type ServerConnections struct {
	Current   int64 `json:"current"`
	Available int64 `json:"available"`
}

// ServerOpCounters reports cumulative operation counts since the server
// started.
type ServerOpCounters struct {
	Insert  int64 `json:"insert"`
	Query   int64 `json:"query"`
	Update  int64 `json:"update"`
	Delete  int64 `json:"delete"`
	Getmore int64 `json:"getmore"`
	Command int64 `json:"command"`
}

// ServerStatus summarizes the target MongoDB server's identity and
// runtime state, as reported by the serverStatus command.
type ServerStatus struct {
	Version     string            `json:"version"`
	Host        string            `json:"host"`
	Process     string            `json:"process"`
	UptimeSecs  int64             `json:"uptimeSeconds"`
	Connections ServerConnections `json:"connections"`
	Opcounters  ServerOpCounters  `json:"opcounters"`
}

// ServerStatus reports version, host, uptime, connection pool usage, and
// cumulative operation counters for the connected server.
func (c *Client) ServerStatus(ctx context.Context) (ServerStatus, error) {
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
		return ServerStatus{}, fmt.Errorf("serverStatus: %w", err)
	}

	return ServerStatus{
		Version:    raw.Version,
		Host:       raw.Host,
		Process:    raw.Process,
		UptimeSecs: int64(raw.Uptime),
		Connections: ServerConnections{
			Current:   raw.Connections.Current,
			Available: raw.Connections.Available,
		},
		Opcounters: ServerOpCounters{
			Insert:  raw.Opcounters.Insert,
			Query:   raw.Opcounters.Query,
			Update:  raw.Opcounters.Update,
			Delete:  raw.Opcounters.Delete,
			Getmore: raw.Opcounters.Getmore,
			Command: raw.Opcounters.Command,
		},
	}, nil
}
