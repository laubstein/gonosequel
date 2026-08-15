// Package command parses CLI flags and environment variables into a single
// Options value used to configure the rest of the application.
package command

import (
	"flag"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"
)

// SupportedDrivers lists the backend values --driver currently accepts.
// "redis" and "valkey" both route to the same driver package — the two
// are wire-compatible, --driver just records which one the user typed.
// CouchDB support is planned but not implemented yet.
var SupportedDrivers = []string{"mongodb", "redis", "valkey"}

// defaultPort is the connection port assumed for a driver when --port
// isn't explicitly set — each backend has its own conventional default,
// unlike --host/--user/--pass/--db, which mean the same thing everywhere.
var defaultPort = map[string]int{
	"mongodb": 27017,
	"redis":   6379,
	"valkey":  6379,
}

// Options holds every runtime setting the application accepts, sourced from
// CLI flags with environment variable fallbacks.
type Options struct {
	Driver   string
	URL      string
	Host     string
	Port     int
	User     string
	Pass     string
	DB       string
	Bind     string
	HTTPPort int
	Bookmark string
	Sessions bool
	AuthUser string
	AuthPass string
	Readonly bool
	DevProxy string

	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// Parse builds an Options value from the given CLI arguments (typically
// os.Args[1:]), falling back to environment variables when a flag is unset.
func Parse(args []string) (*Options, error) {
	fs := flag.NewFlagSet("gonosequel", flag.ContinueOnError)

	opts := &Options{}
	fs.StringVar(&opts.Driver, "driver", envOr("ME_DRIVER", "mongodb"), "database backend to connect to ("+strings.Join(SupportedDrivers, ", ")+")")
	fs.StringVar(&opts.URL, "url", envOr("ME_URL", ""), "connection URL (mongodb://... or redis://...)")
	fs.StringVar(&opts.Host, "host", envOr("ME_HOST", ""), "backend host")
	fs.IntVar(&opts.Port, "port", 0, "backend port (default depends on --driver: mongodb 27017, redis/valkey 6379)")
	fs.StringVar(&opts.User, "user", envOr("ME_USER", ""), "backend username")
	fs.StringVar(&opts.Pass, "pass", envOr("ME_PASS", ""), "backend password")
	fs.StringVar(&opts.DB, "db", envOr("ME_DB", ""), "default database (MongoDB database name, or Redis/Valkey numbered database)")

	fs.StringVar(&opts.Bind, "bind", envOr("ME_BIND", "127.0.0.1"), "address to bind the HTTP server")
	fs.IntVar(&opts.HTTPPort, "http-port", envIntOr("ME_HTTP_PORT", 8081), "HTTP server port")

	fs.StringVar(&opts.Bookmark, "bookmark", envOr("ME_BOOKMARK", ""), "load connection from a saved bookmark")
	fs.BoolVar(&opts.Sessions, "sessions", envBoolOr("ME_SESSIONS", false), "enable multi-session mode")
	fs.StringVar(&opts.AuthUser, "auth-user", envOr("ME_AUTH_USER", ""), "basic auth username for the web UI")
	fs.StringVar(&opts.AuthPass, "auth-pass", envOr("ME_AUTH_PASS", ""), "basic auth password for the web UI")
	fs.BoolVar(&opts.Readonly, "readonly", envBoolOr("ME_READONLY", false), "reject all non-GET requests")
	fs.StringVar(&opts.DevProxy, "dev-proxy", envOr("ME_DEV_PROXY", ""), "proxy non-API requests to this URL (dev mode)")

	readTimeout := fs.Duration("read-timeout", 30*time.Second, "HTTP read timeout")
	writeTimeout := fs.Duration("write-timeout", 30*time.Second, "HTTP write timeout")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	if !slices.Contains(SupportedDrivers, opts.Driver) {
		return nil, fmt.Errorf("unsupported --driver %q (supported: %s)", opts.Driver, strings.Join(SupportedDrivers, ", "))
	}

	// --port has no fixed flag default (see above) because the right
	// default depends on which --driver was chosen, known only after
	// parsing. ME_MONGO_PORT is kept as a MongoDB-specific fallback for
	// backward compatibility; anything else falls back to defaultPort.
	if opts.Port == 0 {
		if v, ok := os.LookupEnv("ME_MONGO_PORT"); ok {
			if n, err := strconv.Atoi(v); err == nil {
				opts.Port = n
			}
		}
	}
	if opts.Port == 0 {
		opts.Port = defaultPort[opts.Driver]
	}

	opts.ReadTimeout = *readTimeout
	opts.WriteTimeout = *writeTimeout

	return opts, nil
}

// URI builds the connection string for the selected --driver from the
// discrete host/port/user/pass/db flags, preferring the explicit --url
// flag when set. mongodb and redis/valkey each get their own scheme.
func (o *Options) URI() string {
	if o.URL != "" {
		return o.URL
	}

	host := o.Host
	if host == "" {
		host = "localhost"
	}

	scheme := "mongodb"
	if o.Driver == "redis" || o.Driver == "valkey" {
		scheme = "redis"
	}

	auth := ""
	if o.User != "" {
		auth = fmt.Sprintf("%s:%s@", o.User, o.Pass)
	}

	uri := fmt.Sprintf("%s://%s%s:%d", scheme, auth, host, o.Port)
	if o.DB != "" {
		uri += "/" + o.DB
	}
	return uri
}

func envOr(key, def string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return def
}

func envIntOr(key string, def int) int {
	v, ok := os.LookupEnv(key)
	if !ok {
		return def
	}
	var n int
	if _, err := fmt.Sscanf(v, "%d", &n); err != nil {
		return def
	}
	return n
}

func envBoolOr(key string, def bool) bool {
	v, ok := os.LookupEnv(key)
	if !ok {
		return def
	}
	return v == "1" || v == "true" || v == "yes"
}
