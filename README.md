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
  "manifest_hash": "651207116b5c03074e215f78d899283eef32de1d0a708e031f03a9253bd5368b",
  "lock_hash": "3e0eaa8f728b185d5c357a01fcb51ebd1b05baf7c7d501abf0a1d8dac6294811",
  "config_hash": "a1e8293c08a8530db69bd99eb739a5ffa938f0c1f7e6b7ace9cdd371f7ce608b",
  "env": {
    "hash": "ad6792bda5db53ca800f6c73c7087c2ff875fed21ea7f38d766d7ec3013c4f96",
    "total": 64,
    "redacted": 1
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
      "code": "lock_missing_target",
      "severity": "error",
      "subject": "mcp-hub",
      "message": "The lockfile does not include this target.",
      "fix": "Regenerate the lockfile so the target is pinned."
    },
    {
      "code": "lock_extra_target",
      "severity": "error",
      "subject": "old-bridge",
      "message": "The lockfile contains an extra target that is not in the manifest.",
      "fix": "Delete the stale lock entry or regenerate the lockfile from the manifest."
    },
    {
      "code": "rule_ask",
      "severity": "warning",
      "subject": "claude",
      "message": "Target claude requires manual review.",
      "fix": "Review the target and confirm that the ask rule is acceptable."
    },
    {
      "code": "rule_deny",
      "severity": "error",
      "subject": "mcp-hub",
      "message": "Target mcp-hub matches a deny rule.",
      "fix": "Remove or rename the denied target before you refresh the lockfile."
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
