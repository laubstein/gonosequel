// Tools-tab support: database-wide (not single-collection) diagnostics for
// spotting problem hotspots — bloated collections, unused indexes, and
// operations running right now. Everything here is read-only.
package client

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// CollectionsOverview reports size and count stats for every collection in
// a database, so bloat (storage size far exceeding data size) or unusually
// large collections stand out without inspecting them one at a time.
func (c *Client) CollectionsOverview(ctx context.Context, dbName string) ([]CollectionStats, error) {
	colls, err := c.ListCollections(ctx, dbName)
	if err != nil {
		return nil, fmt.Errorf("collections overview: %w", err)
	}

	out := make([]CollectionStats, 0, len(colls))
	for _, coll := range colls {
		stats, err := c.Stats(ctx, dbName, coll.Name)
		if err != nil {
			return nil, fmt.Errorf("collections overview: stats for %q: %w", coll.Name, err)
		}
		stats.Name = coll.Name
		out = append(out, stats)
	}
	return out, nil
}

// IndexUsageStat reports how many operations have used a single index
// since the server last restarted.
type IndexUsageStat struct {
	Collection string    `json:"collection"`
	Index      string    `json:"index"`
	Ops        int64     `json:"ops"`
	Since      time.Time `json:"since"`
}

// IndexUsage reports $indexStats for every index in every collection of a
// database. An index with Ops == 0 has not been used by any operation
// since the server started — a candidate for dropping.
func (c *Client) IndexUsage(ctx context.Context, dbName string) ([]IndexUsageStat, error) {
	colls, err := c.ListCollections(ctx, dbName)
	if err != nil {
		return nil, fmt.Errorf("index usage: %w", err)
	}

	var out []IndexUsageStat
	for _, coll := range colls {
		docs, err := c.Aggregate(ctx, dbName, coll.Name, bson.A{bson.M{"$indexStats": bson.M{}}})
		if err != nil {
			return nil, fmt.Errorf("index usage: $indexStats for %q: %w", coll.Name, err)
		}
		for _, doc := range docs {
			var raw struct {
				Name     string `bson:"name"`
				Accesses struct {
					Ops   int64     `bson:"ops"`
					Since time.Time `bson:"since"`
				} `bson:"accesses"`
			}
			b, err := bson.Marshal(doc)
			if err != nil {
				return nil, fmt.Errorf("index usage: marshal $indexStats result: %w", err)
			}
			if err := bson.Unmarshal(b, &raw); err != nil {
				return nil, fmt.Errorf("index usage: decode $indexStats result: %w", err)
			}
			out = append(out, IndexUsageStat{
				Collection: coll.Name,
				Index:      raw.Name,
				Ops:        raw.Accesses.Ops,
				Since:      raw.Accesses.Since,
			})
		}
	}
	return out, nil
}

// CurrentOp describes a single in-progress server operation.
type CurrentOp struct {
	OpID        int64  `json:"opid"`
	Namespace   string `json:"namespace"`
	Op          string `json:"op"`
	SecsRunning int64  `json:"secsRunning"`
	Client      string `json:"client"`
	Description string `json:"description"`
}

// CurrentOps lists active operations that have been running for at least
// minSecs, filtered server-side by the currentOp command itself rather
// than in Go, so idle/background operations never cross the wire.
func (c *Client) CurrentOps(ctx context.Context, minSecs int64) ([]CurrentOp, error) {
	cmd := bson.D{
		{Key: "currentOp", Value: 1},
		{Key: "active", Value: true},
		{Key: "secs_running", Value: bson.M{"$gte": minSecs}},
	}

	var raw struct {
		InProg []struct {
			OpID        int64  `bson:"opid"`
			NS          string `bson:"ns"`
			Op          string `bson:"op"`
			SecsRunning int64  `bson:"secs_running"`
			Client      string `bson:"client"`
			Desc        string `bson:"desc"`
		} `bson:"inprog"`
	}
	if err := c.mongo.Database("admin").RunCommand(ctx, cmd).Decode(&raw); err != nil {
		return nil, fmt.Errorf("currentOp: %w", err)
	}

	out := make([]CurrentOp, 0, len(raw.InProg))
	for _, op := range raw.InProg {
		out = append(out, CurrentOp{
			OpID:        op.OpID,
			Namespace:   op.NS,
			Op:          op.Op,
			SecsRunning: op.SecsRunning,
			Client:      op.Client,
			Description: op.Desc,
		})
	}
	return out, nil
}
