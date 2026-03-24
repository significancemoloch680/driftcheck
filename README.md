<p align="center">
  <h1 align="center">agentlock</h1>
  <p align="center">
    <strong>Stop agent dependency drift before it breaks your pipeline.</strong>
  </p>
  <p align="center">
    <a href="https://github.com/ratelworks/agentlock/actions/workflows/ci.yml"><img src="https://github.com/ratelworks/agentlock/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
    <a href="https://github.com/ratelworks/agentlock/releases/latest"><img src="https://img.shields.io/github/v/release/ratelworks/agentlock" alt="Release"></a>
    <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-green" alt="License"></a>
    <a href="https://pkg.go.dev/github.com/ratelworks/agentlock"><img src="https://pkg.go.dev/badge/github.com/ratelworks/agentlock.svg" alt="Go Reference"></a>
  </p>
</p>

---

Your team runs Codex, Claude, MCP servers, and OpenClaw skills — each pinned to a version today, drifted tomorrow.
**agentlock** reads a JSON manifest, compares it against a lockfile, applies `allow`/`deny`/`ask` rules, and reports exactly what changed — in machine-readable JSON that CI can act on.

## Quick Start

```bash
# Install
go install github.com/ratelworks/agentlock@latest

# Run against the example manifest
agentlock --manifest examples/agentlock.json --lock examples/agentlock.lock.json
```

## What It Catches

```
$ agentlock --manifest examples/agentlock.json \
            --lock examples/agentlock.lock.json \
            --git=false --canary=false --env=false
```

```json
{
  "status": "fail",
  "summary": {
    "targets": 3,
    "rules": 3,
    "findings": 4,
    "errors": 3,
    "warnings": 1
  },
  "findings": [
    {
      "severity": "error",
      "subject": "mcp-hub",
      "message": "The lockfile does not include this target.",
      "fix": "Regenerate the lockfile so the target is pinned."
    },
    {
      "severity": "error",
      "subject": "old-bridge",
      "message": "The lockfile contains an extra target that is not in the manifest.",
      "fix": "Delete the stale lock entry or regenerate the lockfile from the manifest."
    },
    {
      "severity": "warning",
      "subject": "claude",
      "message": "Target claude requires manual review.",
      "fix": "Review the target and confirm that the ask rule is acceptable."
    },
    {
      "severity": "error",
      "subject": "mcp-hub",
      "message": "Target mcp-hub matches a deny rule.",
      "fix": "Remove or rename the denied target before you refresh the lockfile."
    }
  ]
}
```

Every finding includes a `fix` field — no guessing what to do next.

## How It Works

```
agentlock.json (manifest)     agentlock.lock.json (lockfile)
┌─────────────────────┐       ┌─────────────────────┐
│ targets:             │       │ targets:             │
│   - codex v5.0.0     │  ──→  │   - codex v5.0.0 ✓   │
│   - claude v1.0.0    │  ──→  │   - claude v0.9.0 ✗   │  ← version mismatch
│   - mcp-hub v0.8.0   │  ──→  │   (missing) ✗         │  ← not locked
│ rules:               │       │   - old-bridge ✗       │  ← stale entry
│   - codex: allow     │       └─────────────────────┘
│   - claude: ask      │
│   - mcp-*: deny      │
└─────────────────────┘
```

1. **Manifest** declares what agent targets your project depends on
2. **Lockfile** pins the exact versions and digests
3. **Rules** control which targets are `allow`ed, `deny`ed, or require `ask` review
4. **agentlock** compares them and reports drift, missing targets, stale entries, and policy violations

## Features

| Feature | Description |
|---------|-------------|
| **Manifest ↔ Lock diff** | Detects missing targets, stale entries, digest mismatches |
| **Policy rules** | `allow` / `deny` / `ask` glob patterns on target names |
| **Environment hash** | Redacted SHA-256 of env vars — secrets are never stored |
| **Git evidence** | HEAD commit, dirty state, changed file count |
| **HTTP canaries** | Health check endpoints defined in the manifest |
| **Machine-readable** | JSON-only output — pipe to `jq`, store as CI artifact |
| **Zero config** | No server, no database — local files only |

## Manifest Format

Create `agentlock.json` in your project root:

```json
{
  "name": "my-agent-stack",
  "targets": [
    {
      "name": "codex",
      "kind": "cli",
      "source": "github.com/openai/codex",
      "version": "5.0.0"
    },
    {
      "name": "claude",
      "kind": "cli",
      "source": "anthropic/claude-code",
      "version": "1.0.0"
    }
  ],
  "rules": [
    { "pattern": "codex", "decision": "allow" },
    { "pattern": "claude", "decision": "ask" }
  ],
  "canaries": [
    {
      "name": "mcp-health",
      "url": "http://localhost:3000/health",
      "expected_status": 200,
      "timeout_millis": 5000
    }
  ]
}
```

## CLI Reference

```bash
# Basic audit
agentlock --manifest agentlock.json --lock agentlock.lock.json

# Bootstrap a lockfile from manifest
agentlock --manifest agentlock.json --write-lock

# Skip network checks (offline mode)
agentlock --canary=false --git=false --env=false

# Fail CI on warnings too
agentlock --fail-on-warning
```

### Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Pass — no errors |
| `1` | Fail — manifest/lock drift or policy violation |
| `2` | System error — file not found, parse failure |

## CI Integration

### GitHub Actions

```yaml
- name: Audit agent dependencies
  run: |
    go install github.com/ratelworks/agentlock@latest
    agentlock --manifest agentlock.json --lock agentlock.lock.json --fail-on-warning
```

### Pre-commit Hook

```bash
#!/bin/sh
agentlock --canary=false --env=false || exit 1
```

## Why agentlock?

AI agent toolchains (Codex, Claude Code, MCP servers, OpenClaw skills) are growing fast.
Today you pin versions manually. Tomorrow someone upgrades a provider, changes an MCP endpoint, or adds a new skill — and your pipeline breaks at 2 AM.

**agentlock** is `package-lock.json` for agent dependencies:
- Declare what you depend on (manifest)
- Pin exact versions (lockfile)
- Enforce team policies (rules)
- Catch drift before it reaches production (CI)

## Development

```bash
make build    # → bin/agentlock
make test     # go test -race ./...
make lint     # go vet ./...
```

## Contributing

1. Fork the repository
2. Make a focused change with tests
3. Run `go test ./...` and `go vet ./...`
4. Open a pull request

## License

MIT — see [LICENSE](LICENSE).
