// Package command parses CLI flags and environment variables into a single
// Options value used to configure the rest of the application.
package command

import (
	"flag"
	"fmt"
	"os"
	"time"
)

// Options holds every runtime setting the application accepts, sourced from
// CLI flags with environment variable fallbacks.
type Options struct {
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
	SkipOpen bool
	Prefix   string
	DevProxy string

	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// Parse builds an Options value from the given CLI arguments (typically
// os.Args[1:]), falling back to environment variables when a flag is unset.
func Parse(args []string) (*Options, error) {
	fs := flag.NewFlagSet("mongo-express-go", flag.ContinueOnError)

	opts := &Options{}
	fs.StringVar(&opts.URL, "url", envOr("ME_URL", ""), "MongoDB connection URL (mongodb://...)")
	fs.StringVar(&opts.Host, "host", envOr("ME_HOST", ""), "MongoDB host")
	fs.IntVar(&opts.Port, "port", envIntOr("ME_MONGO_PORT", 27017), "MongoDB port")
	fs.StringVar(&opts.User, "user", envOr("ME_USER", ""), "MongoDB username")
	fs.StringVar(&opts.Pass, "pass", envOr("ME_PASS", ""), "MongoDB password")
	fs.StringVar(&opts.DB, "db", envOr("ME_DB", ""), "MongoDB default database")

	fs.StringVar(&opts.Bind, "bind", envOr("ME_BIND", "127.0.0.1"), "address to bind the HTTP server")
	fs.IntVar(&opts.HTTPPort, "http-port", envIntOr("ME_HTTP_PORT", 8081), "HTTP server port")

	fs.StringVar(&opts.Bookmark, "bookmark", envOr("ME_BOOKMARK", ""), "load connection from a saved bookmark")
	fs.BoolVar(&opts.Sessions, "sessions", envBoolOr("ME_SESSIONS", false), "enable multi-session mode")
	fs.StringVar(&opts.AuthUser, "auth-user", envOr("ME_AUTH_USER", ""), "basic auth username for the web UI")
	fs.StringVar(&opts.AuthPass, "auth-pass", envOr("ME_AUTH_PASS", ""), "basic auth password for the web UI")
	fs.BoolVar(&opts.Readonly, "readonly", envBoolOr("ME_READONLY", false), "reject all non-GET requests")
	fs.BoolVar(&opts.SkipOpen, "skip-open", envBoolOr("ME_SKIP_OPEN", false), "do not open a browser on startup")
	fs.StringVar(&opts.Prefix, "prefix", envOr("ME_PREFIX", ""), "serve the app under a URL path prefix")
	fs.StringVar(&opts.DevProxy, "dev-proxy", envOr("ME_DEV_PROXY", ""), "proxy non-API requests to this URL (dev mode)")

	readTimeout := fs.Duration("read-timeout", 30*time.Second, "HTTP read timeout")
	writeTimeout := fs.Duration("write-timeout", 30*time.Second, "HTTP write timeout")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	opts.ReadTimeout = *readTimeout
	opts.WriteTimeout = *writeTimeout

	return opts, nil
}

// MongoURI builds a mongodb:// connection string from Opts, preferring the
// explicit --url flag when set.
func (o *Options) MongoURI() string {
	if o.URL != "" {
		return o.URL
	}

	host := o.Host
	if host == "" {
		host = "localhost"
	}

	auth := ""
	if o.User != "" {
		auth = fmt.Sprintf("%s:%s@", o.User, o.Pass)
	}

	uri := fmt.Sprintf("mongodb://%s%s:%d", auth, host, o.Port)
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
