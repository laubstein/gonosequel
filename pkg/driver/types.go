package driver

import (
	"errors"
	"time"
)

// Sentinel errors returned by Driver implementations, checked with
// errors.Is.
var (
	// ErrNotFound is returned when a requested database, collection, or
	// document does not exist.
	ErrNotFound = errors.New("not found")
	// ErrAlreadyExists is returned when creating a collection or index
	// that already exists under that name.
	ErrAlreadyExists = errors.New("already exists")
)

// DatabaseInfo summarizes a single database.
type DatabaseInfo struct {
	Name      string `json:"name"`
	SizeBytes int64  `json:"sizeBytes"`
}

// CollectionInfo summarizes a single collection.
type CollectionInfo struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// CollectionStats reports size and document count metrics for a
// collection.
type CollectionStats struct {
	Name         string `json:"name,omitempty"`
	Count        int64  `json:"count"`
	SizeBytes    int64  `json:"sizeBytes"`
	StorageBytes int64  `json:"storageBytes"`
	IndexBytes   int64  `json:"indexBytes"`
	AvgObjSize   int64  `json:"avgObjSize"`
	IndexCount   int64  `json:"indexCount"`
}

// CreateCollectionOptions configures collection creation.
type CreateCollectionOptions struct {
	Capped      bool
	MaxSizeByte int64
	MaxDocs     int64
}

// FindOptions controls a document query: filter, projection, sort, and
// pagination.
type FindOptions struct {
	Filter     Doc
	Projection Doc
	Sort       OrderedDoc
	Skip       int64
	Limit      int64
}

// FindResult holds a page of documents plus the total matching count.
type FindResult struct {
	Documents []Doc
	// Total is the number of documents matching Filter. When Filter is
	// empty, this is typically an estimate rather than an exact count,
	// since counting the whole collection on every page load does not
	// scale.
	Total int64
	// TotalIsEstimate is true when Total is an estimate rather than an
	// exact count.
	TotalIsEstimate bool
}

// IndexInfo describes a single index.
type IndexInfo struct {
	Name   string     `json:"name"`
	Keys   OrderedDoc `json:"keys"`
	Unique bool       `json:"unique"`
}

// FieldType describes one observed value type at a field path, with how
// often it was seen in the sample.
type FieldType struct {
	Type  string `json:"type"`
	Count int    `json:"count"`
}

// SchemaField summarizes a single field path across the sampled documents.
type SchemaField struct {
	Path  string      `json:"path"`
	Types []FieldType `json:"types"`
}

// DefaultSchemaSampleSize is the number of documents sampled by
// InferSchema when the caller does not specify a size.
const DefaultSchemaSampleSize = 100

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

// ServerStatus summarizes the target server's identity and runtime state.
type ServerStatus struct {
	Version     string            `json:"version"`
	Host        string            `json:"host"`
	Process     string            `json:"process"`
	UptimeSecs  int64             `json:"uptimeSeconds"`
	Connections ServerConnections `json:"connections"`
	Opcounters  ServerOpCounters  `json:"opcounters"`
}

// IndexUsageStat reports how many operations have used a single index
// since the server last restarted.
type IndexUsageStat struct {
	Collection string    `json:"collection"`
	Index      string    `json:"index"`
	Ops        int64     `json:"ops"`
	Since      time.Time `json:"since"`
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
