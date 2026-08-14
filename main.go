// Command mongo-express-go serves a web UI for exploring a MongoDB
// deployment. main only handles flag parsing and process lifecycle; the
// server itself is built in pkg/api.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/laubstein/mongo-express-go/pkg/api"
	"github.com/laubstein/mongo-express-go/pkg/client"
	"github.com/laubstein/mongo-express-go/pkg/command"
	"github.com/laubstein/mongo-express-go/pkg/session"
)

func main() {
	opts, err := command.Parse(os.Args[1:])
	if err != nil {
		log.Fatalf("parse flags: %v", err)
	}

	registry := session.NewRegistry()

	if !opts.Sessions {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		cl, err := client.Connect(ctx, opts.MongoURI())
		cancel()
		if err != nil {
			log.Fatalf("connect to mongodb: %v", err)
		}
		registry.Put(session.DefaultID, cl, session.Info{ID: session.DefaultID, Name: "default"})
	}

	app := api.New(api.Config{
		Registry: registry,
		Sessions: opts.Sessions,
		Readonly: opts.Readonly,
		AuthUser: opts.AuthUser,
		AuthPass: opts.AuthPass,
	})

	addr := fmt.Sprintf("%s:%d", opts.Bind, opts.HTTPPort)
	log.Printf("mongo-express-go listening on %s", addr)
	if err := app.Listen(addr); err != nil {
		log.Fatalf("serve: %v", err)
	}
}
