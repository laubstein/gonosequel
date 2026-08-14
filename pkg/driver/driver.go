// Package driver defines the contract a NoSQL backend must satisfy to be
// explored through the application: MongoDB today, with Valkey/Redis and
// CouchDB planned. It has no knowledge of MongoDB, HTTP, or any concrete
// backend — pkg/client implements it, and pkg/session/pkg/api/pkg/export
// depend on it instead of a concrete backend package, so a future backend
// package never needs to import pkg/client just to share this vocabulary.
package driver

import "context"

// Doc is a single document/record, backend-agnostic: a MongoDB backend's
// bson.M, a CouchDB document, or a Valkey hash all fit here. It is a type
// alias (not a defined type) so a plain map[string]any built anywhere in
// the codebase is already a Doc, and so type switches on Doc also match
// map[string]any values built outside this package.
type Doc = map[string]any

// Entry is one ordered key/value pair. OrderedDoc is used exactly where
// key order is semantically significant — index key specs and sort specs
// — the generic equivalent of MongoDB's bson.D/bson.E.
type Entry struct {
	Key   string
	Value any
}

// OrderedDoc is an ordered sequence of key/value pairs; see Entry.
type OrderedDoc []Entry

// DatabaseAdmin creates, lists, and drops databases.
type DatabaseAdmin interface {
	ListDatabases(ctx context.Context) ([]DatabaseInfo, error)
	CreateDatabase(ctx context.Context, dbName, initialCollection string) error
	DropDatabase(ctx context.Context, dbName string) error
}

// CollectionAdmin creates, lists, renames, drops, and reports stats for
// collections within a database.
type CollectionAdmin interface {
	ListCollections(ctx context.Context, dbName string) ([]CollectionInfo, error)
	CreateCollection(ctx context.Context, dbName, collName string, opts CreateCollectionOptions) error
	DropCollection(ctx context.Context, dbName, collName string) error
	RenameCollection(ctx context.Context, dbName, oldName, newName string) error
	Stats(ctx context.Context, dbName, collName string) (CollectionStats, error)
}

// DocumentStore is CRUD on individual documents within a collection.
type DocumentStore interface {
	Find(ctx context.Context, dbName, collName string, opts FindOptions) (FindResult, error)
	FindOne(ctx context.Context, dbName, collName string, id any) (Doc, error)
	InsertOne(ctx context.Context, dbName, collName string, doc Doc) (any, error)
	ReplaceOne(ctx context.Context, dbName, collName string, id any, doc Doc) error
	DeleteOne(ctx context.Context, dbName, collName string, id any) error
}

// IndexAdmin creates, lists, and drops indexes on a collection.
type IndexAdmin interface {
	ListIndexes(ctx context.Context, dbName, collName string) ([]IndexInfo, error)
	CreateIndex(ctx context.Context, dbName, collName string, keys OrderedDoc, unique bool) (string, error)
	DropIndex(ctx context.Context, dbName, collName, name string) error
}

// SchemaInferrer samples a collection's documents to infer field types,
// since none of the target backends has a declared, authoritative schema.
type SchemaInferrer interface {
	InferSchema(ctx context.Context, dbName, collName string, sampleSize int64) ([]SchemaField, error)
}

// QueryExplainer reports how the backend would execute a filter.
type QueryExplainer interface {
	Explain(ctx context.Context, dbName, collName string, filter Doc) (Doc, error)
}

// Aggregator runs a multi-stage aggregation pipeline. Not every backend has
// an equivalent (Valkey, notably) — it is its own interface so a future
// driver that lacks the capability simply doesn't implement it, rather
// than having to fake it on the composed Driver.
type Aggregator interface {
	Aggregate(ctx context.Context, dbName, collName string, pipeline []Doc) ([]Doc, error)
}

// ServerDiagnostics backs the Tools tab: server identity/uptime and
// database-wide (not single-collection) hotspot diagnostics.
type ServerDiagnostics interface {
	ServerStatus(ctx context.Context) (ServerStatus, error)
	CollectionsOverview(ctx context.Context, dbName string) ([]CollectionStats, error)
	IndexUsage(ctx context.Context, dbName string) ([]IndexUsageStat, error)
	CurrentOps(ctx context.Context, minSecs int64) ([]CurrentOp, error)
}

// DocCodec is how a backend represents documents on the wire. MongoDB's
// implementation handles Extended JSON (ObjectId, Decimal128, ...); a
// CouchDB backend's would just be plain JSON with no surrogate types.
type DocCodec interface {
	// MarshalRelaxed renders doc as the human-readable form served to the
	// frontend for display.
	MarshalRelaxed(doc Doc) ([]byte, error)
	// MarshalCanonical renders doc as the type-preserving form served when
	// a document is opened for editing, so a view-edit-save round-trip
	// cannot silently change a value's type.
	MarshalCanonical(doc Doc) ([]byte, error)
	// UnmarshalDoc parses a single document sent by the frontend.
	UnmarshalDoc(raw []byte) (Doc, error)
	// UnmarshalDocArray parses a document array — an aggregation pipeline,
	// in practice — sent by the frontend.
	UnmarshalDocArray(raw []byte) ([]Doc, error)
	// EncodeDocID encodes a document's id (which may be any type the
	// backend supports, not just a string) into a URL-safe string
	// suitable for a route path segment.
	EncodeDocID(id any) (string, error)
	// DecodeDocID reverses EncodeDocID.
	DecodeDocID(encoded string) (any, error)
}

// Driver is a live connection to one NoSQL backend, composed from the
// narrow interfaces above. It is one interface, not several narrower ones
// at each consumption site, because pkg/session.Registry must hold a
// single value per connection regardless of which backend it is — the
// narrower interfaces above exist so tests and future callers can still
// depend on just the slice of capability they need.
//
// Not every backend will implement every embedded interface equally well
// (Valkey has no aggregation, schema inference, or query explain; CouchDB's
// index model is design documents/views, not CreateIndex(keys, unique)).
// That is expected and deliberately not solved here — a future backend
// package decides how to handle a capability it lacks (return an error, or
// let callers detect support via a type assertion such as
// `if agg, ok := d.(driver.Aggregator); ok`).
type Driver interface {
	// Close disconnects the backend.
	Close(ctx context.Context) error

	DatabaseAdmin
	CollectionAdmin
	DocumentStore
	IndexAdmin
	SchemaInferrer
	QueryExplainer
	Aggregator
	ServerDiagnostics
	DocCodec
}
