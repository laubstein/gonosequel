// Command gonosequel serves a web UI for exploring a MongoDB
// deployment. main only handles flag parsing, process lifecycle, and
// embedding the built frontend; the server itself is built in pkg/api.
package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"os"
	"time"

	"github.com/laubstein/gonosequel/pkg/api"
	"github.com/laubstein/gonosequel/pkg/bookmarks"
	"github.com/laubstein/gonosequel/pkg/client"
	"github.com/laubstein/gonosequel/pkg/command"
	"github.com/laubstein/gonosequel/pkg/driver"
	"github.com/laubstein/gonosequel/pkg/redis"
	"github.com/laubstein/gonosequel/pkg/session"
)

// distFS embeds the built frontend. It is empty until `cd web && npm run
// build` has populated web/dist — see the Makefile's build target, which
// always runs the frontend build before `go build`.
//
//go:embed all:web/dist
var distFS embed.FS

// docsFS embeds the built documentation site. Empty until `cd docs && npm
// run docs:build` has populated docs/.vitepress/dist — same two-stage
// build pattern as distFS above.
//
//go:embed all:docs/.vitepress/dist
var docsFS embed.FS

func main() {
	opts, err := command.Parse(os.Args[1:])
	if err != nil {
		log.Fatalf("parse flags: %v", err)
	}

	bookmarksDir, err := bookmarks.DefaultDir()
	if err != nil {
		log.Fatalf("resolve bookmarks directory: %v", err)
	}

	registry := session.NewRegistry()

	if !opts.Sessions {
		uri, err := resolveURI(opts, bookmarksDir)
		if err != nil {
			log.Fatalf("resolve connection: %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		cl, err := connect(ctx, opts.Driver, uri)
		cancel()
		if err != nil {
			log.Fatalf("connect: %v", err)
		}
		registry.Put(session.DefaultID, cl, session.Info{ID: session.DefaultID, Name: "default"})
	}

	app := api.New(api.Config{
		Registry:     registry,
		Driver:       opts.Driver,
		Sessions:     opts.Sessions,
		Readonly:     opts.Readonly,
		AuthUser:     opts.AuthUser,
		AuthPass:     opts.AuthPass,
		Assets:       assetsFS(),
		DevProxy:     opts.DevProxy,
		BookmarksDir: bookmarksDir,
		Docs:         docsFSSub(),
	})

	addr := fmt.Sprintf("%s:%d", opts.Bind, opts.HTTPPort)
	log.Printf("gonosequel listening on %s", addr)
	if err := app.Listen(addr); err != nil {
		log.Fatalf("serve: %v", err)
	}
}

// connect dials the backend named by driverName, returning it as a
// driver.Driver so the rest of the app never needs to know which concrete
// backend package handled the connection.
func connect(ctx context.Context, driverName, uri string) (driver.Driver, error) {
	switch driverName {
	case "mongodb":
		return client.Connect(ctx, uri)
	case "redis", "valkey":
		return redis.Connect(ctx, uri)
	default:
		return nil, fmt.Errorf("unknown driver %q", driverName)
	}
}

// resolveURI prefers a saved --bookmark over discrete flags/--url.
func resolveURI(opts *command.Options, bookmarksDir string) (string, error) {
	if opts.Bookmark == "" {
		return opts.URI(), nil
	}
	b, err := bookmarks.Load(bookmarksDir, opts.Bookmark)
	if err != nil {
		return "", fmt.Errorf("load bookmark %q: %w", opts.Bookmark, err)
	}
	return b.URL, nil
}

// assetsFS strips the "web/dist" embed prefix so paths inside it match
// what static.New expects (e.g. "index.html", not "web/dist/index.html").
func assetsFS() fs.FS {
	sub, err := fs.Sub(distFS, "web/dist")
	if err != nil {
		log.Fatalf("embed web/dist: %v", err)
	}
	return sub
}

// docsFSSub strips the "docs/.vitepress/dist" embed prefix, same reason
// as assetsFS above.
func docsFSSub() fs.FS {
	sub, err := fs.Sub(docsFS, "docs/.vitepress/dist")
	if err != nil {
		log.Fatalf("embed docs/.vitepress/dist: %v", err)
	}
	return sub
}
