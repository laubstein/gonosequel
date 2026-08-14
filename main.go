// Command mongo-express-go serves a web UI for exploring a MongoDB
// deployment. main only handles flag parsing and process lifecycle; the
// server itself is built in pkg/api.
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/laubstein/mongo-express-go/pkg/api"
	"github.com/laubstein/mongo-express-go/pkg/command"
)

func main() {
	opts, err := command.Parse(os.Args[1:])
	if err != nil {
		log.Fatalf("parse flags: %v", err)
	}

	app := api.New(api.Config{
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
