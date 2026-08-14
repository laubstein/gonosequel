package api

import "github.com/gofiber/fiber/v3"

// registerRoutes mounts every route group on the app. Route handlers are
// implemented in the handlers_*.go files, grouped by domain.
func registerRoutes(app *fiber.App, d *deps) {
	api := app.Group("/api")

	api.Get("/info", d.handleInfo)
	api.Get("/sessions", d.handleListSessions)
	api.Get("/bookmarks", d.handleListBookmarks)
	api.Post("/connect", d.handleConnect)
	api.Post("/disconnect", d.handleDisconnect)

	scoped := api.Group("", withSession(d))

	scoped.Get("/connection", d.handleConnectionInfo)
	scoped.Get("/server_status", d.handleServerStatus)
	scoped.Get("/history", d.handleHistory)

	scoped.Get("/databases", d.handleListDatabases)
	scoped.Post("/databases", d.handleCreateDatabase)
	scoped.Delete("/databases/:db", d.handleDropDatabase)

	scoped.Get("/databases/:db/collections", d.handleListCollections)
	scoped.Post("/databases/:db/collections", d.handleCreateCollection)
	scoped.Get("/databases/:db/collections/:coll", d.handleCollectionStats)
	scoped.Delete("/databases/:db/collections/:coll", d.handleDropCollection)
	scoped.Post("/databases/:db/collections/:coll/rename", d.handleRenameCollection)
	scoped.Get("/databases/:db/collections/:coll/schema", d.handleCollectionSchema)

	scoped.Get("/databases/:db/collections/:coll/documents", d.handleListDocuments)
	scoped.Get("/databases/:db/collections/:coll/explain", d.handleExplainQuery)
	scoped.Post("/databases/:db/collections/:coll/aggregate", d.handleAggregate)
	scoped.Post("/databases/:db/collections/:coll/documents", d.handleInsertDocument)
	scoped.Get("/databases/:db/collections/:coll/documents/:id", d.handleGetDocument)
	scoped.Put("/databases/:db/collections/:coll/documents/:id", d.handleReplaceDocument)
	scoped.Delete("/databases/:db/collections/:coll/documents/:id", d.handleDeleteDocument)

	scoped.Get("/databases/:db/collections/:coll/indexes", d.handleListIndexes)
	scoped.Post("/databases/:db/collections/:coll/indexes", d.handleCreateIndex)
	scoped.Delete("/databases/:db/collections/:coll/indexes/:name", d.handleDropIndex)

	scoped.Get("/databases/:db/collections/:coll/export", d.handleExport)
}
