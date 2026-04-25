# LLM.md - github.com/hanzoai/rollout

Shared rollout primitive used by `~/work/hanzo/operator` and `~/work/liquidity/operator`.

## What

Progressive deployment across environments (devnet -> testnet -> mainnet) with
health gates and automatic rollback. Reads YAML service manifests, compares
desired vs running images, and rolls services forward via `kubectl set image`
+ `rollout status` with port-forward health checks.

## Why standalone

Both `hanzo-operator` and `liquid-operator` (Go binary side) ship a CLI
subcommand that drives identical logic. Keeping the source duplicated invited
drift; a single module that both import is the simpler shape.

## Public API

- `rollout.Run(ctx, cfg)` -- entry point.
- `rollout.LoadAll(dir)` -- read dev.yml/test.yml/main.yml.
- `rollout.Manifest`, `rollout.Service`, `rollout.ServiceDiff`,
  `rollout.Env`, `rollout.PromotionOrder`.
- `rollout.HealthPath(name)`, `rollout.HealthPort(name)`,
  `rollout.IsBaseService(name)` -- health-check helpers keyed on
  service name conventions (ats/bd/ta = base trio; iam/kms/gateway etc).
- `rollout.GetCurrentImage`, `rollout.SetImage`, `rollout.WaitRollout`,
  `rollout.PortForward` -- thin kubectl shims.
- `rollout.CheckHealth` -- port-forward + HTTP probe with retries.

## Manifest shape

```
version: "1"
context: gke_<project>_<region>_<env>      # kubectl context
project: <gcp/aws project>
domain:  <env domain>
replicas: 1
services:
  <name>:
    image: <registry>/<repo>:vX.Y.Z         # tag MUST be semver vX.Y.Z
    namespace: <ns>
    kind: deployment | statefulset
```

Files: `dev.yml`, `test.yml`, `main.yml` for devnet/testnet/mainnet.

## Tests

- `go test ./...` -- pure unit tests, no cluster required.
- `go test -race ./...` -- race detector clean.

## Consumers

- `~/work/hanzo/operator` (Go) -- imports as
  `github.com/hanzoai/operator/internal/rollout` shim that re-exports
  `github.com/hanzoai/rollout`.
- `~/work/liquidity/operator` (Go binary at `cmd/rollout/`) -- same
  pattern.
