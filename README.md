agentlock
Agent manifests drift when lockfiles, git state, and health checks are not verified together.
agentlock is a single-binary Go CLI that audits that drift and emits machine-readable JSON for CI and local checks.

[![CI](https://github.com/ratelworks/agentlock/actions/workflows/ci.yml/badge.svg)](https://github.com/ratelworks/agentlock/actions/workflows/ci.yml)

## What it does

`agentlock` reads a JSON manifest, compares it with a lockfile, applies `allow` / `deny` / `ask` rules, records redacted environment and git evidence, and can run optional HTTP canaries.

The output is JSON only, so it is easy to pipe into CI, archives, or later diff checks.

## Install

```bash
go install github.com/ratelworks/agentlock@latest
```

## Usage

Audit a manifest and lockfile:

```bash
agentlock --manifest examples/agentlock.json --lock examples/agentlock.lock.json --git=false --canary=false
```

Example output:

```json
{
  "status": "fail",
  "summary": {
    "targets": 3,
    "rules": 3,
    "canaries": 0,
    "findings": 4,
    "errors": 3,
    "warnings": 1
  },
  "manifest_path": "examples/agentlock.json",
  "lock_path": "examples/agentlock.lock.json",
  "manifest_hash": "b0f8f0ca53f2c0e5f6f0f2fca5b5d84b0f7d967bce18c2fd41c9cb2f6c6408b3",
  "lock_hash": "98ebd27fbe2d3f2b70b9f0c0a62b9b9c4a2a97f6c6ce2b0fe9f4ce2d9a4b5f75",
  "config_hash": "9cc6f4c928fdc8cfac5d4c8f2f9a0b6af5f1e6f4a7e3e2d1c9b8a7f6e5d4c3b2",
  "env": {
    "hash": "0cc175b9c0f1b6a831c399e269772661",
    "total": 0,
    "redacted": 0
  },
  "git": {
    "present": false,
    "head": "",
    "dirty": false,
    "changed_files": 0,
    "diff_stat": ""
  },
  "canaries": [],
  "findings": [
    {
      "code": "rule_ask",
      "severity": "warning",
      "subject": "claude",
      "message": "Target claude requires manual review.",
      "fix": "Review the claude target and confirm that the ask rule is acceptable."
    },
    {
      "code": "rule_deny",
      "severity": "error",
      "subject": "mcp-hub",
      "message": "Target mcp-hub matches a deny rule.",
      "fix": "Remove or rename the denied target before you refresh the lockfile."
    },
    {
      "code": "lock_missing_target",
      "severity": "error",
      "subject": "mcp-hub",
      "message": "The lockfile does not include the mcp-hub target.",
      "fix": "Run agentlock again with --write-lock to bootstrap the lockfile."
    },
    {
      "code": "lock_extra_target",
      "severity": "error",
      "subject": "old-bridge",
      "message": "The lockfile contains an extra target that is not in the manifest.",
      "fix": "Delete the stale lock entry or regenerate the lockfile from the manifest."
    }
  ]
}
```

## Contributing

1. Fork the repository.
2. Make a focused change.
3. Run `go test ./...` and `go vet ./...`.
4. Open a pull request with a clear description of the behavior change.

## License

MIT, see [LICENSE](LICENSE).
