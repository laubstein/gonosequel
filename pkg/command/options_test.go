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

func TestParseEnvMongoPortFallback(t *testing.T) {
	setEnv(t, "GNS_DRIVER", "mongodb")
	setEnv(t, "GNS_MONGO_PORT", "27018")

	opts, err := Parse(nil)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if opts.Port != 27018 {
		t.Errorf("Port = %d, want 27018", opts.Port)
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
