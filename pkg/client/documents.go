package client

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/laubstein/gonosequel/pkg/driver"
)

// Find runs a query against a collection with pagination.
func (c *Client) Find(ctx context.Context, dbName, collName string, opts driver.FindOptions) (driver.FindResult, error) {
	coll := c.collection(dbName, collName)

	filter := toBSON(opts.Filter)
	if filter == nil {
		filter = bson.M{}
	}

	findOpts := options.Find().SetSkip(opts.Skip)
	if opts.Limit > 0 {
		findOpts.SetLimit(opts.Limit)
	}
	if opts.Sort != nil {
		findOpts.SetSort(toBSOND(opts.Sort))
	}
	if opts.Projection != nil {
		findOpts.SetProjection(toBSON(opts.Projection))
	}

	cur, err := coll.Find(ctx, filter, findOpts)
	if err != nil {
		return driver.FindResult{}, fmt.Errorf("find %q.%q: %w", dbName, collName, err)
	}
	defer cur.Close(ctx)

	docs := []driver.Doc{}
	for cur.Next(ctx) {
		var doc bson.M
		if err := cur.Decode(&doc); err != nil {
			return driver.FindResult{}, fmt.Errorf("decode document: %w", err)
		}
		docs = append(docs, toDoc(doc))
	}
	if err := cur.Err(); err != nil {
		return driver.FindResult{}, fmt.Errorf("iterate documents: %w", err)
	}

	total, estimate, err := c.count(ctx, coll, filter)
	if err != nil {
		return driver.FindResult{}, err
	}

	return driver.FindResult{Documents: docs, Total: total, TotalIsEstimate: estimate}, nil
}

// count returns the total matching document count. An empty filter uses
// EstimatedDocumentCount, an O(1) metadata read; a non-empty filter uses
// CountDocuments, an exact but collection-scan-cost operation.
func (c *Client) count(ctx context.Context, coll *mongo.Collection, filter bson.M) (total int64, estimate bool, err error) {
	if len(filter) == 0 {
		n, err := coll.EstimatedDocumentCount(ctx)
		if err != nil {
			return 0, true, fmt.Errorf("estimated count: %w", err)
		}
		return n, true, nil
	}
	n, err := coll.CountDocuments(ctx, filter)
	if err != nil {
		return 0, false, fmt.Errorf("count: %w", err)
	}
	return n, false, nil
}

// FindOne fetches a single document by its _id.
func (c *Client) FindOne(ctx context.Context, dbName, collName string, id any) (driver.Doc, error) {
	var doc bson.M
	err := c.collection(dbName, collName).FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, driver.ErrNotFound
		}
		return nil, fmt.Errorf("find one %q.%q: %w", dbName, collName, err)
	}
	return toDoc(doc), nil
}

// InsertOne inserts a document and returns its assigned _id.
func (c *Client) InsertOne(ctx context.Context, dbName, collName string, doc driver.Doc) (any, error) {
	res, err := c.collection(dbName, collName).InsertOne(ctx, toBSON(doc))
	if err != nil {
		return nil, fmt.Errorf("insert into %q.%q: %w", dbName, collName, err)
	}
	return res.InsertedID, nil
}

// ReplaceOne replaces a document identified by _id with a new body. The _id
// in doc, if present, is ignored in favor of id.
func (c *Client) ReplaceOne(ctx context.Context, dbName, collName string, id any, doc driver.Doc) error {
	body := toBSON(doc)
	delete(body, "_id")
	res, err := c.collection(dbName, collName).ReplaceOne(ctx, bson.M{"_id": id}, body)
	if err != nil {
		return fmt.Errorf("replace in %q.%q: %w", dbName, collName, err)
	}
	if res.MatchedCount == 0 {
		return driver.ErrNotFound
	}
	return nil
}

// DeleteOne deletes a document identified by _id.
func (c *Client) DeleteOne(ctx context.Context, dbName, collName string, id any) error {
	res, err := c.collection(dbName, collName).DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("delete from %q.%q: %w", dbName, collName, err)
	}
	if res.DeletedCount == 0 {
		return driver.ErrNotFound
	}
	return nil
}
