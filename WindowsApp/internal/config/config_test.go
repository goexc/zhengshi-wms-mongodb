package config

import "testing"

func TestEnvironmentOverridesAPIBaseURL(t *testing.T) {
	t.Setenv(APIBaseURLEnv, "https://test.example.invalid")
	cfg := withEnvironment(Default())
	if cfg.APIBaseURL != "https://test.example.invalid" {
		t.Fatalf("APIBaseURL = %q", cfg.APIBaseURL)
	}
}

func TestDefaultKeepsLoginEnabled(t *testing.T) {
	if !Default().KeepLoggedIn {
		t.Fatal("keep login should default to enabled")
	}
}

func TestImageBaseURLCanBeOverridden(t *testing.T) {
	t.Setenv(ImageBaseURLEnv, "https://files.example.invalid/materials/")
	if got := ImageBaseURL(); got != "https://files.example.invalid/materials/" {
		t.Fatalf("ImageBaseURL = %q", got)
	}
}
