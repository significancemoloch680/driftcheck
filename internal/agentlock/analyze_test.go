package agentlock

import (
	"testing"
	"time"
)

func TestAuditAnalysis(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		manifest     Manifest
		lockBuilder  func(t *testing.T, manifest Manifest) Lockfile
		lockExists   bool
		wantStatus   string
		wantFindings int
		wantCounts   map[string]int
	}{
		{
			name: "matching manifest and lock",
			manifest: Manifest{
				Name: "match",
				Targets: []Target{
					{Name: "codex", Kind: "cli", Source: "github.com/openai/codex", Version: "5.0.0"},
				},
				Rules: []Rule{
					{Pattern: "codex", Decision: decisionAllow},
				},
			},
			lockBuilder: func(t *testing.T, manifest Manifest) Lockfile {
				t.Helper()
				lock, err := generateLock(manifest, time.Date(2026, time.March, 24, 0, 0, 0, 0, time.UTC))
				if err != nil {
					t.Fatalf("generateLock failed: %v", err)
				}
				return lock
			},
			lockExists:   true,
			wantStatus:   statusPass,
			wantFindings: 0,
			wantCounts:   map[string]int{},
		},
		{
			name: "missing lock target and ask rule",
			manifest: Manifest{
				Name: "drift",
				Targets: []Target{
					{Name: "claude", Kind: "cli", Source: "anthropic/claude-code", Version: "1.0.0"},
					{Name: "mcp-hub", Kind: "bridge", Source: "localhost/mcp", Version: "0.8.0"},
				},
				Rules: []Rule{
					{Pattern: "claude", Decision: decisionAsk},
				},
			},
			lockBuilder: func(t *testing.T, manifest Manifest) Lockfile {
				t.Helper()
				lock, err := generateLock(manifest, time.Date(2026, time.March, 24, 0, 0, 0, 0, time.UTC))
				if err != nil {
					t.Fatalf("generateLock failed: %v", err)
				}
				lock.Targets = []LockedTarget{
					{
						Name:    "claude",
						Kind:    "cli",
						Source:  "anthropic/claude-code",
						Version: "1.0.0",
						Digest:  targetDigest(Target{Name: "claude", Kind: "cli", Source: "anthropic/claude-code", Version: "1.0.0"}),
					},
				}
				return lock
			},
			lockExists:   true,
			wantStatus:   statusFail,
			wantFindings: 2,
			wantCounts:   map[string]int{severityError: 1, severityWarning: 1},
		},
		{
			name: "extra lock target and invalid rule",
			manifest: Manifest{
				Name: "invalid",
				Targets: []Target{
					{Name: "codex", Kind: "cli", Source: "github.com/openai/codex", Version: "5.0.0"},
				},
				Rules: []Rule{
					{Pattern: "", Decision: "maybe"},
				},
			},
			lockBuilder: func(t *testing.T, manifest Manifest) Lockfile {
				t.Helper()
				lock, err := generateLock(manifest, time.Date(2026, time.March, 24, 0, 0, 0, 0, time.UTC))
				if err != nil {
					t.Fatalf("generateLock failed: %v", err)
				}
				lock.Targets = []LockedTarget{
					{
						Name:    "codex",
						Kind:    "cli",
						Source:  "github.com/openai/codex",
						Version: "5.0.0",
						Digest:  targetDigest(Target{Name: "codex", Kind: "cli", Source: "github.com/openai/codex", Version: "5.0.0"}),
					},
					{
						Name:    "old-bridge",
						Kind:    "bridge",
						Source:  "localhost/old",
						Version: "1.0.0",
						Digest:  "deadbeef",
					},
				}
				return lock
			},
			lockExists:   true,
			wantStatus:   statusFail,
			wantFindings: 2,
			wantCounts:   map[string]int{severityError: 2},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			lock := tt.lockBuilder(t, tt.manifest)
			generatedLock, err := generateLock(tt.manifest, time.Date(2026, time.March, 24, 0, 0, 0, 0, time.UTC))
			if err != nil {
				t.Fatalf("generateLock failed: %v", err)
			}

			findings := make([]Finding, 0)
			findings = append(findings, validateRules(tt.manifest.Rules)...)
			findings = append(findings, compareManifestToLock(tt.manifest, lock, generatedLock, tt.lockExists)...)
			findings = append(findings, evaluateTargetPolicies(tt.manifest.Targets, tt.manifest.Rules)...)

			if got := summarizeStatus(findings, false); got != tt.wantStatus {
				t.Fatalf("status = %q, want %q", got, tt.wantStatus)
			}
			if len(findings) != tt.wantFindings {
				t.Fatalf("findings = %d, want %d", len(findings), tt.wantFindings)
			}

			counts := make(map[string]int)
			for _, finding := range findings {
				counts[finding.Severity]++
			}

			for severity, want := range tt.wantCounts {
				if counts[severity] != want {
					t.Fatalf("severity %q count = %d, want %d", severity, counts[severity], want)
				}
			}
		})
	}
}

func TestSnapshotEnvRedactsSecrets(t *testing.T) {
	t.Parallel()

	snapshot := snapshotEnv([]string{
		"API_KEY=secret",
		"SAFE=value",
		"SESSION_TOKEN=another-secret",
	})

	if snapshot.Redacted != 2 {
		t.Fatalf("redacted = %d, want 2", snapshot.Redacted)
	}
	if snapshot.Total != 3 {
		t.Fatalf("total = %d, want 3", snapshot.Total)
	}
}

func TestGenerateLockIsStable(t *testing.T) {
	t.Parallel()

	manifest := Manifest{
		Name: "stable",
		Targets: []Target{
			{Name: "b", Kind: "cli", Source: "two", Version: "2"},
			{Name: "a", Kind: "cli", Source: "one", Version: "1"},
		},
		Rules: []Rule{
			{Pattern: "b", Decision: decisionAllow},
			{Pattern: "a", Decision: decisionAsk},
		},
	}

	first, err := generateLock(manifest, time.Date(2026, time.March, 24, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("generateLock failed: %v", err)
	}
	second, err := generateLock(manifest, time.Date(2026, time.March, 24, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("generateLock failed: %v", err)
	}

	if first.ManifestHash != second.ManifestHash {
		t.Fatalf("manifest hash changed between runs")
	}
	if first.RulesHash != second.RulesHash {
		t.Fatalf("rules hash changed between runs")
	}
	if len(first.Targets) != 2 || first.Targets[0].Name != "a" || first.Targets[1].Name != "b" {
		t.Fatalf("targets were not sorted as expected: %#v", first.Targets)
	}
}
