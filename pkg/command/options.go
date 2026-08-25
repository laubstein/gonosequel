// Package command parses CLI flags and environment variables into a single
// Options value used to configure the rest of the application.
package command

import (
	"flag"
	"fmt"
	"os"
	"slices"
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
	Driver        string
	URL           string
	Host          string
	Port          int
	User          string
	Pass          string
	DB            string
	Bind          string
	HTTPPort      int
	Bookmark      string
	Sessions      bool
	AuthUser      string
	AuthPass      string
	Readonly      bool
	DevProxy      string
	TLSCert       string
	TLSKey        string
	SessionSecret string
	// AuthEnabled/TLSEnabled default true and only matter as an explicit
	// override: mongo-express's own ME_CONFIG_BASICAUTH_ENABLED/
	// ME_CONFIG_SITE_SSL_ENABLED default *false*, gating on/off a
	// username+password or cert+key that may already be sitting in the
	// environment. gonosequel's own convention is simpler — basic
	// auth/TLS activate from the mere presence of --auth-user or
	// --tls-cert+--tls-key, no separate enable flag needed — so these two
	// exist only so an imported ME_CONFIG_*_ENABLED=false is honored
	// (main.go clears the corresponding credentials/cert when false)
	// rather than silently ignored.
	AuthEnabled bool
	TLSEnabled  bool

	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// Parse builds an Options value from the given CLI arguments (typically
// os.Args[1:]), falling back to environment variables when a flag is unset.
func Parse(args []string) (*Options, error) {
	fs := flag.NewFlagSet("gonosequel", flag.ContinueOnError)

	// Flag defaults are built before fs.Parse below runs, so opts.Driver
	// isn't populated yet — but the MONGODB_* fallback tier for
	// host/port/user/pass only applies for driver "mongodb", so that has
	// to be known now. resolveDriver mirrors --driver's own default
	// resolution (env, else "mongodb") and additionally honors an
	// explicit -driver/--driver in args, without needing a full parse.
	driver := resolveDriver(args)

	opts := &Options{}
	fs.StringVar(&opts.Driver, "driver", driver, "database backend to connect to ("+strings.Join(SupportedDrivers, ", ")+")")
	fs.StringVar(&opts.URL, "url", envOr("URL", ""), "connection URL (mongodb://... or redis://...)")
	fs.StringVar(&opts.Host, "host", envOrMongo(driver, "HOST", "MONGODB_HOST", ""), "backend host")
	fs.IntVar(&opts.Port, "port", envIntOrMongo(driver, "PORT", "MONGODB_PORT", 0), "backend port (default depends on --driver: mongodb 27017, redis/valkey 6379)")
	fs.StringVar(&opts.User, "user", envOrMongo(driver, "USER", "MONGODB_USERNAME", ""), "backend username")
	fs.StringVar(&opts.Pass, "pass", envOrMongo(driver, "PASS", "MONGODB_PASSWORD", ""), "backend password")
	fs.StringVar(&opts.DB, "db", envOr("DB", ""), "default database (MongoDB database name, or Redis/Valkey numbered database)")

	fs.StringVar(&opts.Bind, "bind", envOr("BIND", "127.0.0.1"), "address to bind the HTTP server")
	fs.IntVar(&opts.HTTPPort, "http-port", envIntOr("HTTP_PORT", 8081), "HTTP server port")

	fs.StringVar(&opts.Bookmark, "bookmark", envOr("BOOKMARK", ""), "load connection from a saved bookmark")
	fs.BoolVar(&opts.Sessions, "sessions", envBoolOr("SESSIONS", false), "enable multi-session mode")
	fs.StringVar(&opts.AuthUser, "auth-user", envOrCompat("AUTH_USER", "ME_CONFIG_BASICAUTH_USERNAME", ""), "basic auth username for the web UI")
	fs.StringVar(&opts.AuthPass, "auth-pass", envOrCompat("AUTH_PASS", "ME_CONFIG_BASICAUTH_PASSWORD", ""), "basic auth password for the web UI")
	fs.BoolVar(&opts.Readonly, "readonly", envBoolOr("READONLY", false), "reject all non-GET requests")
	fs.StringVar(&opts.DevProxy, "dev-proxy", envOr("DEV_PROXY", ""), "proxy non-API requests to this URL (dev mode)")
	fs.StringVar(&opts.TLSCert, "tls-cert", envOrCompat("TLS_CERT", "ME_CONFIG_SITE_SSL_CRT_PATH", ""), "path to TLS certificate file; enables HTTPS together with --tls-key")
	fs.StringVar(&opts.TLSKey, "tls-key", envOrCompat("TLS_KEY", "ME_CONFIG_SITE_SSL_KEY_PATH", ""), "path to TLS private key file; enables HTTPS together with --tls-cert")
	fs.StringVar(&opts.SessionSecret, "session-secret", envOrCompat("SESSION_SECRET", "ME_CONFIG_SITE_SESSIONSECRET", ""), "secret used to HMAC-sign session IDs handed out in --sessions mode; empty disables signing")
	fs.BoolVar(&opts.AuthEnabled, "auth-enabled", envBoolOrCompat("AUTH_ENABLED", "ME_CONFIG_BASICAUTH_ENABLED", true), "whether --auth-user/--auth-pass take effect; set to false to keep them configured but inactive")
	fs.BoolVar(&opts.TLSEnabled, "tls-enabled", envBoolOrCompat("TLS_ENABLED", "ME_CONFIG_SITE_SSL_ENABLED", true), "whether --tls-cert/--tls-key take effect; set to false to keep them configured but inactive")

	readTimeout := fs.Duration("read-timeout", 30*time.Second, "HTTP read timeout")
	writeTimeout := fs.Duration("write-timeout", 30*time.Second, "HTTP write timeout")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	if !slices.Contains(SupportedDrivers, opts.Driver) {
		return nil, fmt.Errorf("unsupported --driver %q (supported: %s)", opts.Driver, strings.Join(SupportedDrivers, ", "))
	}

	if (opts.TLSCert == "") != (opts.TLSKey == "") {
		return nil, fmt.Errorf("--tls-cert and --tls-key must be set together")
	}

	// --port has no fixed flag default (see above) because the right
	// default depends on which --driver was chosen — GNS_PORT/ME_PORT/
	// MONGODB_PORT (see the flag default above) may also have left it at
	// 0, so this is the final fallback, based on the driver actually
	// parsed rather than resolveDriver's pre-parse guess.
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

// envLookup resolves a setting from the environment: GNS_<key> first, then
// ME_<key> as a fallback for mongo-express's own variable names (this
// project's earlier working name), so an existing mongo-express deployment
// can switch images without also having to rewrite its environment. The
// value — from either prefix — is then run through os.ExpandEnv, so a
// value like "$MONGO_URL" resolves through to MONGO_URL's own value rather
// than being used as the literal string "$MONGO_URL" (the same expansion
// a shell would do, useful when the value is wired up from another
// variable already set by the surrounding deployment, e.g. a docker-compose
// service link).
func envLookup(key string) (string, bool) {
	if v, ok := os.LookupEnv("GNS_" + key); ok {
		return os.ExpandEnv(v), true
	}
	if v, ok := os.LookupEnv("ME_" + key); ok {
		return os.ExpandEnv(v), true
	}
	return "", false
}

// resolveDriver determines the effective --driver before the full
// flag.FlagSet has parsed args — needed because envOrMongo/envIntOrMongo's
// MONGODB_* fallback tier only applies for "mongodb", and that decision
// has to be made while building --host/--port/--user/--pass's own flag
// defaults, before fs.Parse (see Parse). Mirrors --driver's own default
// (env, else "mongodb"), overridden by an explicit -driver/--driver found
// in args. The real --driver flag is still parsed normally afterward and
// is what actually ends up in Options — this is only a best-effort
// pre-scan to gate a default value, not the source of truth.
func resolveDriver(args []string) string {
	if v, ok := scanFlagValue(args, "driver"); ok {
		return v
	}
	return envOr("DRIVER", "mongodb")
}

// scanFlagValue looks for -name/--name in args, in any of the standard
// library flag package's accepted forms (-name=value, --name=value, -name
// value, --name value), and returns its value. Used only by resolveDriver,
// ahead of the real fs.Parse call.
func scanFlagValue(args []string, name string) (string, bool) {
	eq1, eq2 := "-"+name+"=", "--"+name+"="
	for i, a := range args {
		switch {
		case strings.HasPrefix(a, eq1):
			return strings.TrimPrefix(a, eq1), true
		case strings.HasPrefix(a, eq2):
			return strings.TrimPrefix(a, eq2), true
		case a == "-"+name || a == "--"+name:
			if i+1 < len(args) {
				return args[i+1], true
			}
			return "", false
		}
	}
	return "", false
}

// envLookupMongo extends envLookup with a third fallback tier —
// MONGODB_<mongoKey> — consulted only when driver is "mongodb". These
// names (MONGODB_HOST, MONGODB_PORT, MONGODB_USERNAME, MONGODB_PASSWORD)
// match the convention used by official MongoDB Docker images and common
// Helm chart deployments, so gonosequel can pick up connection details
// already wired into the environment by those without any extra
// configuration — meaningless for Redis/Valkey, so not consulted then.
func envLookupMongo(driver, key, mongoKey string) (string, bool) {
	if v, ok := envLookup(key); ok {
		return v, true
	}
	if driver != "mongodb" {
		return "", false
	}
	if v, ok := os.LookupEnv(mongoKey); ok {
		return os.ExpandEnv(v), true
	}
	return "", false
}

// envLookupCompat extends envLookup with a third fallback tier —
// compatKey, looked up unconditionally (unlike envLookupMongo's MONGODB_*
// tier, which only applies for --driver mongodb). Used for settings that
// mean the same thing regardless of backend (basic auth, TLS, session
// secret), where compatKey is the name mongo-express itself used
// (ME_CONFIG_BASICAUTH_USERNAME, ME_CONFIG_SITE_SSL_CRT_PATH,
// ME_CONFIG_SITE_SESSIONSECRET, ...), so an existing mongo-express
// deployment's environment keeps working without renaming anything.
func envLookupCompat(key, compatKey string) (string, bool) {
	if v, ok := envLookup(key); ok {
		return v, true
	}
	if v, ok := os.LookupEnv(compatKey); ok {
		return os.ExpandEnv(v), true
	}
	return "", false
}

func envOrCompat(key, compatKey, def string) string {
	if v, ok := envLookupCompat(key, compatKey); ok {
		return v
	}
	return def
}

func envBoolOrCompat(key, compatKey string, def bool) bool {
	v, ok := envLookupCompat(key, compatKey)
	if !ok {
		return def
	}
	return v == "1" || v == "true" || v == "yes"
}

func envOrMongo(driver, key, mongoKey, def string) string {
	if v, ok := envLookupMongo(driver, key, mongoKey); ok {
		return v
	}
	return def
}

func envIntOrMongo(driver, key, mongoKey string, def int) int {
	v, ok := envLookupMongo(driver, key, mongoKey)
	if !ok {
		return def
	}
	var n int
	if _, err := fmt.Sscanf(v, "%d", &n); err != nil {
		return def
	}
	return n
}

func envOr(key, def string) string {
	if v, ok := envLookup(key); ok {
		return v
	}
	return def
}

func envIntOr(key string, def int) int {
	v, ok := envLookup(key)
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
	v, ok := envLookup(key)
	if !ok {
		return def
	}
	return v == "1" || v == "true" || v == "yes"
}
