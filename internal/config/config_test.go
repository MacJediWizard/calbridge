package config

import (
	"os"
	"testing"
)

func TestEnvironmentMethods(t *testing.T) {
	t.Run("IsDevelopment returns true for development", func(t *testing.T) {
		cfg := &Config{
			Server: ServerConfig{
				Environment: EnvDevelopment,
			},
		}
		if !cfg.IsDevelopment() {
			t.Error("expected IsDevelopment() to be true")
		}
		if cfg.IsProduction() {
			t.Error("expected IsProduction() to be false")
		}
	})

	t.Run("IsProduction returns true for production", func(t *testing.T) {
		cfg := &Config{
			Server: ServerConfig{
				Environment: EnvProduction,
			},
		}
		if !cfg.IsProduction() {
			t.Error("expected IsProduction() to be true")
		}
		if cfg.IsDevelopment() {
			t.Error("expected IsDevelopment() to be false")
		}
	})
}

func TestGetMissingRequired(t *testing.T) {
	t.Run("returns empty list when all required fields set", func(t *testing.T) {
		cfg := &Config{
			Server: ServerConfig{
				BaseURL: "https://example.com",
			},
			OIDC: OIDCConfig{
				Issuer:       "https://auth.example.com",
				ClientID:     "client-id",
				ClientSecret: "client-secret",
				RedirectURL:  "https://example.com/callback",
			},
			Security: SecurityConfig{
				EncryptionKey: make([]byte, 32),
				SessionSecret: "session-secret",
			},
			CalDAV: CalDAVConfig{
				DefaultDestURL: "https://caldav.example.com",
			},
		}

		missing := cfg.getMissingRequired()
		if len(missing) != 0 {
			t.Errorf("expected no missing fields, got %v", missing)
		}
	})

	t.Run("returns all missing fields", func(t *testing.T) {
		cfg := &Config{}

		missing := cfg.getMissingRequired()

		expectedMissing := []string{
			"BASE_URL",
			"OIDC_ISSUER",
			"OIDC_CLIENT_ID",
			"OIDC_CLIENT_SECRET",
			"OIDC_REDIRECT_URL",
			"ENCRYPTION_KEY",
			"SESSION_SECRET",
			"DEFAULT_DEST_URL",
		}

		if len(missing) != len(expectedMissing) {
			t.Errorf("expected %d missing fields, got %d: %v", len(expectedMissing), len(missing), missing)
		}

		for _, expected := range expectedMissing {
			found := false
			for _, m := range missing {
				if m == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected %q to be in missing list", expected)
			}
		}
	})

	t.Run("detects partial missing fields", func(t *testing.T) {
		cfg := &Config{
			Server: ServerConfig{
				BaseURL: "https://example.com",
			},
			OIDC: OIDCConfig{
				Issuer: "https://auth.example.com",
				// Missing ClientID, ClientSecret, RedirectURL
			},
			Security: SecurityConfig{
				EncryptionKey: make([]byte, 32),
				// Missing SessionSecret
			},
			// Missing CalDAV.DefaultDestURL
		}

		missing := cfg.getMissingRequired()

		// Missing: OIDC_CLIENT_ID, OIDC_CLIENT_SECRET, OIDC_REDIRECT_URL, SESSION_SECRET, DEFAULT_DEST_URL
		if len(missing) != 5 {
			t.Errorf("expected 5 missing fields, got %d: %v", len(missing), missing)
		}
	})
}

func TestGetEnvFunctions(t *testing.T) {
	// Save and restore environment
	cleanup := func(keys []string) func() {
		saved := make(map[string]string)
		for _, key := range keys {
			saved[key] = os.Getenv(key)
		}
		return func() {
			for key, val := range saved {
				if val == "" {
					os.Unsetenv(key)
				} else {
					os.Setenv(key, val)
				}
			}
		}
	}

	t.Run("getEnv returns default when not set", func(t *testing.T) {
		restore := cleanup([]string{"TEST_VAR"})
		defer restore()

		os.Unsetenv("TEST_VAR")
		result := getEnv("TEST_VAR", "default-value")
		if result != "default-value" {
			t.Errorf("expected 'default-value', got %q", result)
		}
	})

	t.Run("getEnv returns value when set", func(t *testing.T) {
		restore := cleanup([]string{"TEST_VAR"})
		defer restore()

		os.Setenv("TEST_VAR", "actual-value")
		result := getEnv("TEST_VAR", "default-value")
		if result != "actual-value" {
			t.Errorf("expected 'actual-value', got %q", result)
		}
	})

	t.Run("getEnvInt returns default when not set", func(t *testing.T) {
		restore := cleanup([]string{"TEST_INT"})
		defer restore()

		os.Unsetenv("TEST_INT")
		result, err := getEnvInt("TEST_INT", 42)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != 42 {
			t.Errorf("expected 42, got %d", result)
		}
	})

	t.Run("getEnvInt parses integer value", func(t *testing.T) {
		restore := cleanup([]string{"TEST_INT"})
		defer restore()

		os.Setenv("TEST_INT", "123")
		result, err := getEnvInt("TEST_INT", 42)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != 123 {
			t.Errorf("expected 123, got %d", result)
		}
	})

	t.Run("getEnvInt returns error for invalid integer", func(t *testing.T) {
		restore := cleanup([]string{"TEST_INT"})
		defer restore()

		os.Setenv("TEST_INT", "not-a-number")
		_, err := getEnvInt("TEST_INT", 42)
		if err == nil {
			t.Error("expected error for invalid integer")
		}
	})

	t.Run("getEnvFloat returns default when not set", func(t *testing.T) {
		restore := cleanup([]string{"TEST_FLOAT"})
		defer restore()

		os.Unsetenv("TEST_FLOAT")
		result, err := getEnvFloat("TEST_FLOAT", 3.14)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != 3.14 {
			t.Errorf("expected 3.14, got %f", result)
		}
	})

	t.Run("getEnvFloat parses float value", func(t *testing.T) {
		restore := cleanup([]string{"TEST_FLOAT"})
		defer restore()

		os.Setenv("TEST_FLOAT", "2.718")
		result, err := getEnvFloat("TEST_FLOAT", 3.14)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != 2.718 {
			t.Errorf("expected 2.718, got %f", result)
		}
	})

	t.Run("getEnvFloat returns error for invalid float", func(t *testing.T) {
		restore := cleanup([]string{"TEST_FLOAT"})
		defer restore()

		os.Setenv("TEST_FLOAT", "not-a-float")
		_, err := getEnvFloat("TEST_FLOAT", 3.14)
		if err == nil {
			t.Error("expected error for invalid float")
		}
	})
}

func TestEnvironmentConstants(t *testing.T) {
	t.Run("environment constants are correct", func(t *testing.T) {
		if EnvDevelopment != "development" {
			t.Errorf("expected EnvDevelopment to be 'development', got %q", EnvDevelopment)
		}
		if EnvProduction != "production" {
			t.Errorf("expected EnvProduction to be 'production', got %q", EnvProduction)
		}
	})
}

func TestErrorConstants(t *testing.T) {
	t.Run("error constants are not nil", func(t *testing.T) {
		errors := []error{
			ErrMissingConfig,
			ErrInvalidConfig,
			ErrEncryptionKeySize,
			ErrSessionSecretSize,
			ErrValidationFailed,
		}

		for _, err := range errors {
			if err == nil {
				t.Error("expected error constant to be non-nil")
			}
		}
	})
}
