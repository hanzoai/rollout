package rollout

import (
	"path/filepath"
	"testing"
)

// Pure-function invariants. No filesystem. No brand strings.

func TestServiceTagEmptyImage(t *testing.T) {
	if got := (Service{Image: ""}).Tag(); got != "" {
		t.Errorf("Tag() on empty image = %q, want empty", got)
	}
	if err := (Service{Image: ""}).ValidateTag("test"); err == nil {
		t.Error("ValidateTag on empty image: expected error, got nil")
	}
}

func TestEnvFileNames(t *testing.T) {
	cases := map[Env]string{
		Devnet:    "dev.yml",
		Testnet:   "test.yml",
		Mainnet:   "main.yml",
		Env(""):   "",
		Env("xx"): "",
	}
	for env, want := range cases {
		if got := envFile(env); got != want {
			t.Errorf("envFile(%q) = %q, want %q", env, got, want)
		}
	}
}

func TestLoadManifestMissingFile(t *testing.T) {
	dir := t.TempDir()
	// dev.yml does not exist.
	if _, err := LoadManifest(dir, Devnet); err == nil {
		t.Errorf("LoadManifest on missing %s: expected error, got nil", filepath.Join(dir, "dev.yml"))
	}
}
