package command

import (
	"strings"
	"testing"
)

// setEnv sets an environment variable for the duration of the test,
// restoring (or unsetting) whatever was there before on cleanup — tests in
// this file mutate real process environment since Parse reads it via
// os.LookupEnv/os.ExpandEnv directly, not through an injectable source.
func setEnv(t *testing.T, key, value string) {
	t.Helper()
	t.Setenv(key, value)
}

func TestParseEnvGNSPrefix(t *testing.T) {
	setEnv(t, "GNS_URL", "mongodb://gns-host:27017")

	opts, err := Parse(nil)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if opts.URL != "mongodb://gns-host:27017" {
		t.Errorf("URL = %q, want mongodb://gns-host:27017", opts.URL)
	}
}

func TestParseEnvMEFallback(t *testing.T) {
	setEnv(t, "ME_URL", "mongodb://me-host:27017")

	opts, err := Parse(nil)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if opts.URL != "mongodb://me-host:27017" {
		t.Errorf("URL = %q, want mongodb://me-host:27017 (ME_ fallback)", opts.URL)
	}
}

func TestParseEnvGNSTakesPriorityOverME(t *testing.T) {
	setEnv(t, "ME_URL", "mongodb://me-host:27017")
	setEnv(t, "GNS_URL", "mongodb://gns-host:27017")

	opts, err := Parse(nil)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if opts.URL != "mongodb://gns-host:27017" {
		t.Errorf("URL = %q, want mongodb://gns-host:27017 (GNS_ should win)", opts.URL)
	}
}

func TestParseEnvExpandsVarReferences(t *testing.T) {
	setEnv(t, "MONGO_URL", "mongodb://expanded-host:27017")
	setEnv(t, "GNS_URL", "$MONGO_URL")

	opts, err := Parse(nil)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if opts.URL != "mongodb://expanded-host:27017" {
		t.Errorf("URL = %q, want mongodb://expanded-host:27017 (expanded from $MONGO_URL)", opts.URL)
	}
}

func TestParseEnvExpandsVarReferencesFromMEFallback(t *testing.T) {
	setEnv(t, "MONGO_URL", "mongodb://expanded-host:27017")
	setEnv(t, "ME_URL", "$MONGO_URL")

	opts, err := Parse(nil)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if opts.URL != "mongodb://expanded-host:27017" {
		t.Errorf("URL = %q, want mongodb://expanded-host:27017 (expanded from $MONGO_URL via ME_ fallback)", opts.URL)
	}
}

func TestParseEnvPortFallback(t *testing.T) {
	setEnv(t, "GNS_DRIVER", "mongodb")
	setEnv(t, "GNS_PORT", "27018")

	opts, err := Parse(nil)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if opts.Port != 27018 {
		t.Errorf("Port = %d, want 27018", opts.Port)
	}
}

// --- MongoDB-only MONGODB_* fallback tier for host/port/user/pass ---

func TestParseMongodbTierAppliesWhenNothingElseSet(t *testing.T) {
	setEnv(t, "MONGODB_HOST", "mongo-tier-host")
	setEnv(t, "MONGODB_PORT", "27019")
	setEnv(t, "MONGODB_USERNAME", "mongo-tier-user")
	setEnv(t, "MONGODB_PASSWORD", "mongo-tier-pass")

	opts, err := Parse(nil)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if opts.Host != "mongo-tier-host" {
		t.Errorf("Host = %q, want mongo-tier-host", opts.Host)
	}
	if opts.Port != 27019 {
		t.Errorf("Port = %d, want 27019", opts.Port)
	}
	if opts.User != "mongo-tier-user" {
		t.Errorf("User = %q, want mongo-tier-user", opts.User)
	}
	if opts.Pass != "mongo-tier-pass" {
		t.Errorf("Pass = %q, want mongo-tier-pass", opts.Pass)
	}
}

func TestParseGNSHostBeatsMongodbHost(t *testing.T) {
	setEnv(t, "MONGODB_HOST", "mongo-tier-host")
	setEnv(t, "GNS_HOST", "gns-host")

	opts, err := Parse(nil)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if opts.Host != "gns-host" {
		t.Errorf("Host = %q, want gns-host (GNS_HOST should win over MONGODB_HOST)", opts.Host)
	}
}

func TestParseMEHostBeatsMongodbHost(t *testing.T) {
	setEnv(t, "MONGODB_HOST", "mongo-tier-host")
	setEnv(t, "ME_HOST", "me-host")

	opts, err := Parse(nil)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if opts.Host != "me-host" {
		t.Errorf("Host = %q, want me-host (ME_HOST should win over MONGODB_HOST)", opts.Host)
	}
}

func TestParseMongodbTierIgnoredForNonMongoDriver(t *testing.T) {
	setEnv(t, "MONGODB_HOST", "mongo-tier-host")
	setEnv(t, "MONGODB_PORT", "27019")
	setEnv(t, "MONGODB_USERNAME", "mongo-tier-user")
	setEnv(t, "MONGODB_PASSWORD", "mongo-tier-pass")

	opts, err := Parse([]string{"-driver", "redis"})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if opts.Host != "" {
		t.Errorf("Host = %q, want empty (MONGODB_HOST must not apply to --driver redis)", opts.Host)
	}
	if opts.User != "" {
		t.Errorf("User = %q, want empty (MONGODB_USERNAME must not apply to --driver redis)", opts.User)
	}
	if opts.Pass != "" {
		t.Errorf("Pass = %q, want empty (MONGODB_PASSWORD must not apply to --driver redis)", opts.Pass)
	}
	// Port still falls back to the driver's own default (6379), just not
	// from MONGODB_PORT.
	if opts.Port != 6379 {
		t.Errorf("Port = %d, want 6379 (MONGODB_PORT must not apply to --driver redis)", opts.Port)
	}
}

func TestParseFlagOverridesMongodbTier(t *testing.T) {
	setEnv(t, "MONGODB_HOST", "mongo-tier-host")

	opts, err := Parse([]string{"-host", "flag-host"})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if opts.Host != "flag-host" {
		t.Errorf("Host = %q, want flag-host (explicit flag should win over MONGODB_HOST)", opts.Host)
	}
}

func TestParseMongodbTierExpandsVarReferences(t *testing.T) {
	setEnv(t, "REAL_HOST", "expanded-mongo-host")
	setEnv(t, "MONGODB_HOST", "$REAL_HOST")

	opts, err := Parse(nil)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if opts.Host != "expanded-mongo-host" {
		t.Errorf("Host = %q, want expanded-mongo-host (expanded from $REAL_HOST)", opts.Host)
	}
}

// --- driver-agnostic compat tier for auth/TLS/session-secret ---

func TestParseCompatTierAppliesWhenNothingElseSet(t *testing.T) {
	setEnv(t, "ME_CONFIG_BASICAUTH_USERNAME", "compat-user")
	setEnv(t, "ME_CONFIG_BASICAUTH_PASSWORD", "compat-pass")
	setEnv(t, "ME_CONFIG_SITE_SSL_CRT_PATH", "/compat/cert.pem")
	setEnv(t, "ME_CONFIG_SITE_SSL_KEY_PATH", "/compat/key.pem")
	setEnv(t, "ME_CONFIG_SITE_SESSIONSECRET", "compat-secret")

	opts, err := Parse(nil)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if opts.AuthUser != "compat-user" {
		t.Errorf("AuthUser = %q, want compat-user", opts.AuthUser)
	}
	if opts.AuthPass != "compat-pass" {
		t.Errorf("AuthPass = %q, want compat-pass", opts.AuthPass)
	}
	if opts.TLSCert != "/compat/cert.pem" {
		t.Errorf("TLSCert = %q, want /compat/cert.pem", opts.TLSCert)
	}
	if opts.TLSKey != "/compat/key.pem" {
		t.Errorf("TLSKey = %q, want /compat/key.pem", opts.TLSKey)
	}
	if opts.SessionSecret != "compat-secret" {
		t.Errorf("SessionSecret = %q, want compat-secret", opts.SessionSecret)
	}
}

func TestParseGNSAuthUserBeatsCompatTier(t *testing.T) {
	setEnv(t, "ME_CONFIG_BASICAUTH_USERNAME", "compat-user")
	setEnv(t, "GNS_AUTH_USER", "gns-user")

	opts, err := Parse(nil)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if opts.AuthUser != "gns-user" {
		t.Errorf("AuthUser = %q, want gns-user (GNS_AUTH_USER should win over compat tier)", opts.AuthUser)
	}
}

func TestParseMEAuthUserBeatsCompatTier(t *testing.T) {
	setEnv(t, "ME_CONFIG_BASICAUTH_USERNAME", "compat-user")
	setEnv(t, "ME_AUTH_USER", "me-user")

	opts, err := Parse(nil)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if opts.AuthUser != "me-user" {
		t.Errorf("AuthUser = %q, want me-user (ME_AUTH_USER should win over compat tier)", opts.AuthUser)
	}
}

func TestParseTLSCertWithoutKeyIsError(t *testing.T) {
	_, err := Parse([]string{"-tls-cert", "/tmp/cert.pem"})
	if err == nil {
		t.Fatal("Parse: want error when --tls-cert is set without --tls-key")
	}
}

func TestParseTLSKeyWithoutCertIsError(t *testing.T) {
	_, err := Parse([]string{"-tls-key", "/tmp/key.pem"})
	if err == nil {
		t.Fatal("Parse: want error when --tls-key is set without --tls-cert")
	}
}

func TestParseTLSCertAndKeyTogetherIsValid(t *testing.T) {
	opts, err := Parse([]string{"-tls-cert", "/tmp/cert.pem", "-tls-key", "/tmp/key.pem"})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if opts.TLSCert != "/tmp/cert.pem" || opts.TLSKey != "/tmp/key.pem" {
		t.Errorf("TLSCert/TLSKey = %q/%q, want /tmp/cert.pem//tmp/key.pem", opts.TLSCert, opts.TLSKey)
	}
}

// --- AuthEnabled/TLSEnabled: explicit-disable override, default true ---

func TestParseAuthEnabledDefaultsTrue(t *testing.T) {
	opts, err := Parse(nil)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !opts.AuthEnabled {
		t.Errorf("AuthEnabled = false, want true (default)")
	}
	if !opts.TLSEnabled {
		t.Errorf("TLSEnabled = false, want true (default)")
	}
}

func TestParseAuthEnabledFalseFromCompatVar(t *testing.T) {
	setEnv(t, "ME_CONFIG_BASICAUTH_ENABLED", "false")

	opts, err := Parse(nil)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if opts.AuthEnabled {
		t.Errorf("AuthEnabled = true, want false (ME_CONFIG_BASICAUTH_ENABLED=false)")
	}
}

func TestParseTLSEnabledFalseFromCompatVar(t *testing.T) {
	setEnv(t, "ME_CONFIG_SITE_SSL_ENABLED", "false")

	opts, err := Parse(nil)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if opts.TLSEnabled {
		t.Errorf("TLSEnabled = true, want false (ME_CONFIG_SITE_SSL_ENABLED=false)")
	}
}

func TestParseGNSAuthEnabledBeatsCompatVar(t *testing.T) {
	setEnv(t, "ME_CONFIG_BASICAUTH_ENABLED", "false")
	setEnv(t, "GNS_AUTH_ENABLED", "true")

	opts, err := Parse(nil)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !opts.AuthEnabled {
		t.Errorf("AuthEnabled = false, want true (GNS_AUTH_ENABLED should win over compat tier)")
	}
}

func TestParseFlagOverridesEnv(t *testing.T) {
	setEnv(t, "GNS_URL", "mongodb://env-host:27017")

	opts, err := Parse([]string{"-url", "mongodb://flag-host:27017"})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if opts.URL != "mongodb://flag-host:27017" {
		t.Errorf("URL = %q, want mongodb://flag-host:27017 (explicit flag should win over env)", opts.URL)
	}
}

// --- --verbose ---

func TestParseVerboseDefaultsFalse(t *testing.T) {
	opts, err := Parse(nil)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if opts.Verbose {
		t.Errorf("Verbose = true, want false (default)")
	}
}

func TestParseVerboseEnvVar(t *testing.T) {
	setEnv(t, "GNS_VERBOSE", "true")

	opts, err := Parse(nil)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !opts.Verbose {
		t.Errorf("Verbose = false, want true (GNS_VERBOSE=true)")
	}
}

// --- Banner() ---

func TestBannerRedactsBackendPassword(t *testing.T) {
	opts := &Options{Driver: "mongodb", Host: "dbhost", Port: 27017, User: "dbuser", Pass: "s3cr3t-pass"}

	banner := opts.Banner()
	if strings.Contains(banner, "s3cr3t-pass") {
		t.Errorf("Banner() leaked the raw backend password: %s", banner)
	}
	if !strings.Contains(banner, "dbuser") {
		t.Errorf("Banner() should still show the username: %s", banner)
	}
	if !strings.Contains(banner, "****") {
		t.Errorf("Banner() should mask the password with ****: %s", banner)
	}
}

func TestBannerRedactsURLPassword(t *testing.T) {
	opts := &Options{Driver: "mongodb", URL: "mongodb://urluser:urlpass@host:27017/db"}

	banner := opts.Banner()
	if strings.Contains(banner, "urlpass") {
		t.Errorf("Banner() leaked the raw password from --url: %s", banner)
	}
	if !strings.Contains(banner, "urluser") {
		t.Errorf("Banner() should still show the username from --url: %s", banner)
	}
}

func TestBannerShowsBookmarkWithoutResolvingURL(t *testing.T) {
	opts := &Options{Driver: "mongodb", Bookmark: "prod"}

	banner := opts.Banner()
	if !strings.Contains(banner, `bookmark "prod"`) {
		t.Errorf("Banner() = %q, want it to mention bookmark \"prod\"", banner)
	}
}

func TestBannerSessionsModeSkipsConnectionURI(t *testing.T) {
	opts := &Options{Driver: "mongodb", Sessions: true, Host: "should-not-appear", User: "should-not-appear"}

	banner := opts.Banner()
	if strings.Contains(banner, "should-not-appear") {
		t.Errorf("Banner() in --sessions mode should not expose per-connection fields: %s", banner)
	}
	if !strings.Contains(banner, "--sessions mode") {
		t.Errorf("Banner() = %q, want a mention of --sessions mode", banner)
	}
}

func TestBannerRedactsAuthPassword(t *testing.T) {
	opts := &Options{Driver: "mongodb", AuthUser: "admin", AuthPass: "top-secret", AuthEnabled: true}

	banner := opts.Banner()
	if strings.Contains(banner, "top-secret") {
		t.Errorf("Banner() leaked the raw auth password: %s", banner)
	}
	if !strings.Contains(banner, "admin") {
		t.Errorf("Banner() should still show the auth username: %s", banner)
	}
}

func TestBannerRedactsSessionSecret(t *testing.T) {
	opts := &Options{Driver: "mongodb", SessionSecret: "my-signing-secret"}

	banner := opts.Banner()
	if strings.Contains(banner, "my-signing-secret") {
		t.Errorf("Banner() leaked the raw session secret: %s", banner)
	}
	if !strings.Contains(banner, "configured") {
		t.Errorf("Banner() should say the session secret is configured: %s", banner)
	}
}

func TestBannerAuthConfiguredButDisabled(t *testing.T) {
	opts := &Options{Driver: "mongodb", AuthUser: "admin", AuthPass: "pass", AuthEnabled: false}

	banner := opts.Banner()
	if !strings.Contains(banner, "configured but disabled") {
		t.Errorf("Banner() = %q, want it to say basic auth is configured but disabled", banner)
	}
}

func TestBannerTLSConfiguredButDisabled(t *testing.T) {
	opts := &Options{Driver: "mongodb", TLSCert: "/c.pem", TLSKey: "/k.pem", TLSEnabled: false}

	banner := opts.Banner()
	if !strings.Contains(banner, "configured but disabled") {
		t.Errorf("Banner() = %q, want it to say TLS is configured but disabled", banner)
	}
}
