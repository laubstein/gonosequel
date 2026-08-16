package command

import "testing"

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
