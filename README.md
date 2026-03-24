<p align="center">
  <h1 align="center">driftcheck</h1>
  <p align="center">
    <strong>Stop agent dependency drift before it breaks your pipeline.</strong>
  </p>
  <p align="center">
    <a href="https://github.com/ratelworks/driftcheck/actions/workflows/ci.yml"><img src="https://github.com/ratelworks/driftcheck/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
    <a href="https://github.com/ratelworks/driftcheck/releases/latest"><img src="https://img.shields.io/github/v/release/ratelworks/driftcheck" alt="Release"></a>
    <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-green" alt="License"></a>
    <a href="https://pkg.go.dev/github.com/ratelworks/driftcheck"><img src="https://pkg.go.dev/badge/github.com/ratelworks/driftcheck.svg" alt="Go Reference"></a>
  </p>
</p>

---

## The Problem

Your team uses AI agent tools — Codex, Claude Code, MCP servers, OpenClaw skills.
Each tool has a version. Each version behaves differently.

On Monday, everything works. On Wednesday, someone updates Claude Code. On Friday, your CI pipeline breaks at 2 AM and nobody knows why.

**The root cause?** No one tracked which versions were running, and no one verified that the versions matched what was tested.

This is the same problem `package-lock.json` solved for npm, and `Cargo.lock` solved for Rust.
**driftcheck** solves it for AI agent dependencies.

## What driftcheck Does

You write a simple JSON file listing your agent tools and their versions. driftcheck compares it against a lockfile and tells you exactly what drifted:

```
$ driftcheck --manifest driftcheck.json --lock driftcheck.lock.json

{
  "status": "fail",
  "summary": { "findings": 4, "errors": 3, "warnings": 1 },
  "findings": [
    {
      "severity": "error",
      "subject": "mcp-hub",
      "message": "The lockfile does not include this target.",
      "fix": "Regenerate the lockfile so the target is pinned."
    }
  ]
}
```

Every finding tells you **what went wrong** and **how to fix it**. No guessing.

## Getting Started (5 Minutes)

### Step 1: Install

```bash
go install github.com/ratelworks/driftcheck@latest
```

### Step 2: Create a manifest

Create a file called `driftcheck.json` in your project root. This is where you declare the agent tools your project depends on:

```json
{
  "name": "my-project",
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
  "canaries": []
}
```

**What each field means:**

| Field | Purpose | Example |
|-------|---------|---------|
| `name` | Your project name | `"my-project"` |
| `targets` | Agent tools you depend on | Codex v5.0.0, Claude v1.0.0 |
| `targets[].kind` | Type of dependency | `"cli"`, `"bridge"`, `"skill"` |
| `targets[].source` | Where it comes from | `"github.com/openai/codex"` |
| `rules` | Team policies for each target | `allow`, `deny`, or `ask` |
| `canaries` | Health check URLs (optional) | `http://localhost:3000/health` |

### Step 3: Generate a lockfile

```bash
driftcheck --manifest driftcheck.json --write-lock
```

This creates `driftcheck.lock.json` — a snapshot of your current agent versions with SHA-256 digests. **Commit this file to git.** It's your source of truth.

### Step 4: Run the audit

```bash
driftcheck --manifest driftcheck.json --lock driftcheck.lock.json
```

If everything matches, you get `"status": "pass"`. If something drifted, you get a detailed report with fixes.

### Step 5: Add to CI

```yaml
# .github/workflows/ci.yml
- name: Audit agent dependencies
  run: |
    go install github.com/ratelworks/driftcheck@latest
    driftcheck --manifest driftcheck.json --lock driftcheck.lock.json --fail-on-warning
```

Now every PR is checked automatically. No more surprise drift.

## How It Works

```
driftcheck.json (manifest)     driftcheck.lock.json (lockfile)
┌─────────────────────┐       ┌─────────────────────┐
│ targets:             │       │ targets:             │
│   - codex v5.0.0     │  ──>  │   - codex v5.0.0  OK │
│   - claude v1.0.0    │  ──>  │   - claude v0.9.0  !! │  version mismatch
│   - mcp-hub v0.8.0   │  ──>  │   (missing)        !! │  not locked
│ rules:               │       │   - old-bridge     !! │  stale entry
│   - codex: allow     │       └─────────────────────┘
│   - claude: ask      │
│   - mcp-*: deny      │
└─────────────────────┘

driftcheck detects:
  1. Version mismatches between manifest and lockfile
  2. Targets in manifest but missing from lockfile
  3. Stale targets in lockfile that are no longer in manifest
  4. Policy violations (deny rules, ask rules needing review)
```

## Policy Rules

Rules let your team control which agent tools are approved:

| Decision | Behavior | Use case |
|----------|----------|----------|
| `allow` | Passes silently | Approved tools |
| `ask` | Warns (non-blocking by default) | Tools needing team review |
| `deny` | Errors (blocks CI) | Banned or untested tools |

Rules use glob patterns, so `mcp-*` matches `mcp-hub`, `mcp-proxy`, etc.

## Additional Features

### Environment Hash

driftcheck captures a SHA-256 hash of your environment variables (with secrets automatically redacted). This helps answer "was the environment the same when it worked?"

```bash
# Included by default. To skip:
driftcheck --env=false
```

### Git Evidence

Records the current HEAD commit and dirty state, so you can trace exactly which code version was audited.

```bash
# Included by default. To skip:
driftcheck --git=false
```

### Health Canaries

Define HTTP endpoints in your manifest to check if services are actually running:

```json
{
  "canaries": [
    {
      "name": "mcp-server",
      "url": "http://localhost:3000/health",
      "expected_status": 200,
      "timeout_millis": 5000
    }
  ]
}
```

```bash
# Included by default. To skip:
driftcheck --canary=false
```

## CLI Reference

| Command | Description |
|---------|-------------|
| `driftcheck` | Run audit with default paths |
| `driftcheck --manifest FILE` | Specify manifest path (default: `driftcheck.json`) |
| `driftcheck --lock FILE` | Specify lockfile path (default: `driftcheck.lock.json`) |
| `driftcheck --write-lock` | Generate a new lockfile from manifest |
| `driftcheck --fail-on-warning` | Exit code 1 for warnings too (strict mode) |
| `driftcheck --git=false` | Skip git evidence collection |
| `driftcheck --canary=false` | Skip HTTP health checks |
| `driftcheck --env=false` | Skip environment hash |

### Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Pass — all targets match, no policy violations |
| `1` | Fail — drift detected or policy violation |
| `2` | System error — file not found, JSON parse failure |

## Development

```bash
git clone https://github.com/ratelworks/driftcheck.git
cd driftcheck
make build    # build binary to bin/driftcheck
make test     # go test -race ./...
make lint     # go vet ./...
```

## Contributing

1. Fork the repository
2. Make a focused change with tests
3. Run `make test` and `make lint`
4. Open a pull request with a clear description

## License

MIT — see [LICENSE](LICENSE).
