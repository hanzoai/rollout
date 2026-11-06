package rollout

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// A single-env run must not require the other envs' manifests to exist:
// targeting mainnet cannot fail because dev.yml is absent.
func TestRunSingleEnvDoesNotRequireOtherManifests(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.yml"), []byte(testManifest), 0644); err != nil {
		t.Fatal(err)
	}
	err := Run(context.Background(), Config{ManifestDir: dir, Env: "mainnet", DryRun: true})
	if err != nil {
		t.Fatalf("single-env dry run failed without dev/test manifests: %v", err)
	}
}

// A single-env run validates only the target env's tags: a non-semver tag
// in an env that is not being deployed must not block the run.
func TestRunSingleEnvIgnoresOtherEnvTagErrors(t *testing.T) {
	dir := t.TempDir()
	bad := `version: "1"
context: kind-test-devnet
project: acme-devnet
domain: dev.example.com
replicas: 1

services:
  gateway:
    image: example.com/gateway:latest
    namespace: acme
`
	if err := os.WriteFile(filepath.Join(dir, "dev.yml"), []byte(bad), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.yml"), []byte(testManifest), 0644); err != nil {
		t.Fatal(err)
	}
	err := Run(context.Background(), Config{ManifestDir: dir, Env: "mainnet", DryRun: true})
	if err != nil {
		t.Fatalf("single-env run blocked by another env's bad tag: %v", err)
	}
}

// A single-env run still rejects an unknown env name.
func TestRunSingleEnvUnknownEnv(t *testing.T) {
	dir := writeTestManifests(t)
	err := Run(context.Background(), Config{ManifestDir: dir, Env: "staging", DryRun: true})
	if err == nil {
		t.Fatal("expected unknown env error, got nil")
	}
}
