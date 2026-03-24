<p align="center">
  <h1 align="center">driftcheck</h1>
  <p align="center">
    <strong>Audit the gap between what you declared and what you locked.</strong>
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
Each tool has a version declared in a manifest. Each version is pinned in a lockfile.

When someone edits the manifest without regenerating the lockfile — or regenerates the lockfile without updating the manifest — the two files silently diverge. Your CI still passes because nobody checks the gap.

**driftcheck** is a declaration drift audit tool. It compares your manifest (what you declared) against your lockfile (what you pinned) and reports every inconsistency: missing entries, stale entries, version mismatches, and digest changes.

This is the same problem `package-lock.json` solved for npm and `Cargo.lock` solved for Rust — but for AI agent dependency declarations.

> **What driftcheck does NOT do:** Runtime detection is not in scope. driftcheck audits what you declared, not what is installed. It cannot verify that the versions in your manifest match what is actually running on your machine. For that, you need runtime version checks or canary health endpoints.

## What driftcheck Does

You write a JSON manifest listing your agent tools and their versions. driftcheck compares it against a lockfile and tells you exactly what drifted:

```
$ driftcheck --manifest driftcheck.json --lock driftcheck.lock.json

{
  "status": "fail",
  "summary": {
    "targets": 3,
    "rules": 2,
    "canaries": 0,
    "findings": 4,
    "errors": 3,
    "warnings": 1
  },
  "findings": [
    {
      "code": "version_mismatch",
      "severity": "error",
      "subject": "claude",
      "message": "Manifest declares version 1.0.0 but lockfile has 0.9.0.",
      "fix": "Update the manifest or regenerate the lockfile to resolve the version conflict."
    },
    {
      "code": "lock_missing_target",
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

This creates `driftcheck.lock.json` — a snapshot of your current agent declarations with SHA-256 digests. **Commit this file to git.** It is your source of truth.

If a lockfile already exists, `--write-lock` overwrites it with a fresh snapshot generated from the current manifest.

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
+-----------------------+       +-----------------------+
| targets:              |       | targets:              |
|   - codex v5.0.0      |  -->  |   - codex v5.0.0  OK  |
|   - claude v1.0.0     |  -->  |   - claude v0.9.0  !!  |  version mismatch
|   - mcp-hub v0.8.0    |  -->  |   (missing)        !!  |  not locked
| rules:                |       |   - old-bridge     !!  |  stale entry
|   - codex: allow      |       +-----------------------+
|   - claude: ask       |
|   - mcp-*: deny       |
+-----------------------+

driftcheck detects:
  1. Version mismatches between manifest and lockfile (same name, different version)
  2. Targets in manifest but missing from lockfile (new, never locked)
  3. Stale targets in lockfile that are no longer in manifest
  4. Digest changes (same name+version but different content hash)
  5. Manifest/rules hash mismatches (lockfile out of date)
  6. Missing lockfile (not yet generated)
  7. Policy violations (deny rules, ask rules needing review)
  8. Invalid rules (empty patterns, bad globs, unknown decisions)
  9. Canary failures (HTTP health checks returning unexpected status)
```

## Policy Rules

Rules let your team control which agent tools are approved:

| Decision | Behavior | Use case |
|----------|----------|----------|
| `allow` | Passes silently | Approved tools |
| `ask` | Warns (non-blocking by default) | Tools needing team review |
| `deny` | Errors (blocks CI) | Banned or untested tools |

Rules use glob patterns, so `mcp-*` matches `mcp-hub`, `mcp-proxy`, etc.

### Rule Evaluation

- Rules are evaluated in declaration order (first match wins)
- Glob patterns use Go's `filepath.Match` syntax
- Each target is matched against its name, then kind, then source — the first field that matches a rule determines the decision
- If no rule matches a target, it is **allowed by default**
- Invalid glob patterns are reported as errors during rule validation

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
| `driftcheck --write-lock` | Generate a new lockfile from manifest (overwrites existing) |
| `driftcheck --fail-on-warning` | Exit code 1 for warnings too (strict mode) |
| `driftcheck --workdir DIR` | Set working directory for git collection (default: `.`) |
| `driftcheck --git=false` | Skip git evidence collection |
| `driftcheck --canary=false` | Skip HTTP health checks |
| `driftcheck --env=false` | Skip environment hash |

### Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Pass or warn — all declarations match, or only warnings present |
| `1` | Fail — drift detected, policy violation, or warnings with `--fail-on-warning` |
| `2` | System error — file not found, JSON parse failure |

## Finding Codes

| Code | Severity | Description |
|------|----------|-------------|
| `version_mismatch` | error | Manifest and lockfile declare different versions for the same target |
| `lock_missing_target` | error | Target exists in manifest but not in lockfile |
| `lock_extra_target` | error | Target exists in lockfile but not in manifest |
| `lock_digest_mismatch` | error | Same target and version but content hash differs |
| `lock_manifest_hash_mismatch` | error | Lockfile manifest hash does not match current manifest |
| `lock_rules_hash_mismatch` | error | Lockfile rules hash does not match current rules |
| `lock_missing` | error | Lockfile does not exist |
| `rule_deny` | error | Target matches a deny rule |
| `rule_ask` | warning | Target matches an ask rule (requires manual review) |
| `rule_missing_pattern` | error | Rule has an empty pattern |
| `rule_invalid_glob` | error | Rule pattern is not valid `filepath.Match` syntax |
| `rule_invalid_decision` | error | Rule decision is not `allow`, `deny`, or `ask` |
| `canary_failed` | error | Health check endpoint returned unexpected status |

## Development

```bash
git clone https://github.com/ratelworks/driftcheck.git
cd driftcheck
make build       # build binary to bin/driftcheck
make test        # go test ./...
make test-race   # go test -race ./...
make vet         # go vet ./...
make lint        # golangci-lint run
```

## Contributing

1. Fork the repository
2. Make a focused change with tests
3. Run `make test` and `make lint`
4. Open a pull request with a clear description

## License

MIT — see [LICENSE](LICENSE).
