package driftcheck

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
				t.Fatalf("findings = %d, want %d\nfindings: %+v", len(findings), tt.wantFindings, findings)
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

// --- 수정 3: version mismatch 테스트 ---

func TestVersionMismatchDetection(t *testing.T) {
	t.Parallel()

	manifest := Manifest{
		Name: "version-check",
		Targets: []Target{
			{Name: "claude", Kind: "cli", Source: "anthropic/claude-code", Version: "1.0.0"},
		},
	}

	// lockfile에는 같은 name|kind|source인데 version만 다른 항목
	lock := Lockfile{
		ManifestHash: "", // 해시 비교 건너뜀
		Targets: []LockedTarget{
			{
				Name:    "claude",
				Kind:    "cli",
				Source:  "anthropic/claude-code",
				Version: "0.9.0",
				Digest:  targetDigest(Target{Name: "claude", Kind: "cli", Source: "anthropic/claude-code", Version: "0.9.0"}),
			},
		},
	}

	generatedLock, err := generateLock(manifest, time.Now())
	if err != nil {
		t.Fatalf("generateLock failed: %v", err)
	}

	findings := compareManifestToLock(manifest, lock, generatedLock, true)

	if len(findings) != 1 {
		t.Fatalf("findings = %d, want 1\nfindings: %+v", len(findings), findings)
	}
	if findings[0].Code != "version_mismatch" {
		t.Fatalf("code = %q, want %q", findings[0].Code, "version_mismatch")
	}
	if findings[0].Subject != "claude" {
		t.Fatalf("subject = %q, want %q", findings[0].Subject, "claude")
	}
}

func TestVersionMismatchMultipleTargets(t *testing.T) {
	t.Parallel()

	manifest := Manifest{
		Name: "multi-version",
		Targets: []Target{
			{Name: "codex", Kind: "cli", Source: "openai/codex", Version: "5.0.0"},
			{Name: "claude", Kind: "cli", Source: "anthropic/claude-code", Version: "2.0.0"},
		},
	}

	lock := Lockfile{
		Targets: []LockedTarget{
			{
				Name:    "codex",
				Kind:    "cli",
				Source:  "openai/codex",
				Version: "5.0.0",
				Digest:  targetDigest(Target{Name: "codex", Kind: "cli", Source: "openai/codex", Version: "5.0.0"}),
			},
			{
				Name:    "claude",
				Kind:    "cli",
				Source:  "anthropic/claude-code",
				Version: "1.0.0", // version mismatch
				Digest:  targetDigest(Target{Name: "claude", Kind: "cli", Source: "anthropic/claude-code", Version: "1.0.0"}),
			},
		},
	}

	generatedLock, err := generateLock(manifest, time.Now())
	if err != nil {
		t.Fatalf("generateLock failed: %v", err)
	}

	findings := compareManifestToLock(manifest, lock, generatedLock, true)

	// codex는 OK, claude만 version_mismatch + manifest_hash_mismatch 가능
	versionMismatches := 0
	for _, f := range findings {
		if f.Code == "version_mismatch" {
			versionMismatches++
		}
	}
	if versionMismatches != 1 {
		t.Fatalf("version_mismatch findings = %d, want 1\nfindings: %+v", versionMismatches, findings)
	}
}

// --- 수정 2: --write-lock 덮어쓰기 테스트 ---

func TestWriteLockOverwritesExistingLockfile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "driftcheck.json")
	lockPath := filepath.Join(dir, "driftcheck.lock.json")

	// 초기 manifest와 lockfile 생성
	initialManifest := Manifest{
		Name: "overwrite-test",
		Targets: []Target{
			{Name: "codex", Kind: "cli", Source: "openai/codex", Version: "4.0.0"},
		},
	}
	writeTestJSON(t, manifestPath, initialManifest)

	// 초기 lockfile을 --write-lock으로 생성
	_, err := Audit(context.Background(), AuditConfig{
		ManifestPath:    manifestPath,
		LockPath:        lockPath,
		WriteLock:       true,
		IncludeGit:      false,
		IncludeCanaries: false,
		IncludeEnv:      false,
	})
	if err != nil {
		t.Fatalf("initial audit failed: %v", err)
	}

	// lockfile이 생성되었는지 확인
	var initialLock Lockfile
	readTestJSON(t, lockPath, &initialLock)
	if len(initialLock.Targets) != 1 || initialLock.Targets[0].Version != "4.0.0" {
		t.Fatalf("initial lock target version = %q, want %q", initialLock.Targets[0].Version, "4.0.0")
	}

	// manifest를 변경
	updatedManifest := Manifest{
		Name: "overwrite-test",
		Targets: []Target{
			{Name: "codex", Kind: "cli", Source: "openai/codex", Version: "5.0.0"},
		},
	}
	writeTestJSON(t, manifestPath, updatedManifest)

	// --write-lock으로 기존 lockfile 덮어쓰기
	_, err = Audit(context.Background(), AuditConfig{
		ManifestPath:    manifestPath,
		LockPath:        lockPath,
		WriteLock:       true,
		IncludeGit:      false,
		IncludeCanaries: false,
		IncludeEnv:      false,
	})
	if err != nil {
		t.Fatalf("overwrite audit failed: %v", err)
	}

	// lockfile이 덮어쓰여졌는지 확인
	var updatedLock Lockfile
	readTestJSON(t, lockPath, &updatedLock)
	if len(updatedLock.Targets) != 1 || updatedLock.Targets[0].Version != "5.0.0" {
		t.Fatalf("updated lock target version = %q, want %q", updatedLock.Targets[0].Version, "5.0.0")
	}
}

// --- 수정 4: glob 패턴 검증 테스트 ---

func TestInvalidGlobPatternDetection(t *testing.T) {
	t.Parallel()

	rules := []Rule{
		{Pattern: "[invalid", Decision: decisionAllow},
	}

	findings := validateRules(rules)

	if len(findings) != 1 {
		t.Fatalf("findings = %d, want 1\nfindings: %+v", len(findings), findings)
	}
	if findings[0].Code != "rule_invalid_glob" {
		t.Fatalf("code = %q, want %q", findings[0].Code, "rule_invalid_glob")
	}
}

func TestValidGlobPatternsPass(t *testing.T) {
	t.Parallel()

	rules := []Rule{
		{Pattern: "mcp-*", Decision: decisionAllow},
		{Pattern: "claude", Decision: decisionAsk},
		{Pattern: "test-[abc]", Decision: decisionDeny},
	}

	findings := validateRules(rules)

	if len(findings) != 0 {
		t.Fatalf("findings = %d, want 0\nfindings: %+v", len(findings), findings)
	}
}

func TestNoRuleMatchDefaultsToAllow(t *testing.T) {
	t.Parallel()

	targets := []Target{
		{Name: "unknown-tool", Kind: "cli", Source: "example.com/unknown", Version: "1.0.0"},
	}

	rules := []Rule{
		{Pattern: "codex", Decision: decisionDeny},
	}

	findings := evaluateTargetPolicies(targets, rules)

	// unknown-tool은 어떤 rule에도 매칭되지 않으므로 기본 allow → finding 없음
	if len(findings) != 0 {
		t.Fatalf("findings = %d, want 0 (default allow)\nfindings: %+v", len(findings), findings)
	}
}

// --- 수정 5: canary 실패 테스트 (mock HTTP server) ---

func TestCanaryFailureProducesFinding(t *testing.T) {
	t.Parallel()

	// 500을 반환하는 mock 서버
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "driftcheck.json")
	lockPath := filepath.Join(dir, "driftcheck.lock.json")

	manifest := Manifest{
		Name: "canary-fail",
		Targets: []Target{
			{Name: "dummy", Kind: "cli", Source: "example.com/dummy", Version: "1.0.0"},
		},
		Canaries: []Canary{
			{Name: "bad-service", URL: server.URL, ExpectedStatus: 200, TimeoutMillis: 5000},
		},
	}
	writeTestJSON(t, manifestPath, manifest)

	report, err := Audit(context.Background(), AuditConfig{
		ManifestPath:    manifestPath,
		LockPath:        lockPath,
		WriteLock:       true,
		IncludeGit:      false,
		IncludeCanaries: true,
		IncludeEnv:      false,
	})
	if err != nil {
		t.Fatalf("audit failed: %v", err)
	}

	canaryFindings := 0
	for _, f := range report.Findings {
		if f.Code == "canary_failed" {
			canaryFindings++
		}
	}
	if canaryFindings != 1 {
		t.Fatalf("canary_failed findings = %d, want 1\nfindings: %+v", canaryFindings, report.Findings)
	}
}

func TestCanarySuccessNoneFinding(t *testing.T) {
	t.Parallel()

	// 200을 반환하는 mock 서버
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "driftcheck.json")
	lockPath := filepath.Join(dir, "driftcheck.lock.json")

	manifest := Manifest{
		Name: "canary-pass",
		Targets: []Target{
			{Name: "dummy", Kind: "cli", Source: "example.com/dummy", Version: "1.0.0"},
		},
		Canaries: []Canary{
			{Name: "good-service", URL: server.URL, ExpectedStatus: 200, TimeoutMillis: 5000},
		},
	}
	writeTestJSON(t, manifestPath, manifest)

	report, err := Audit(context.Background(), AuditConfig{
		ManifestPath:    manifestPath,
		LockPath:        lockPath,
		WriteLock:       true,
		IncludeGit:      false,
		IncludeCanaries: true,
		IncludeEnv:      false,
	})
	if err != nil {
		t.Fatalf("audit failed: %v", err)
	}

	for _, f := range report.Findings {
		if f.Code == "canary_failed" {
			t.Fatalf("unexpected canary_failed finding: %+v", f)
		}
	}
}

// --- 빈 manifest 테스트 ---

func TestEmptyManifestTargets(t *testing.T) {
	t.Parallel()

	manifest := Manifest{
		Name:    "empty",
		Targets: []Target{},
		Rules:   []Rule{},
	}

	generatedLock, err := generateLock(manifest, time.Now())
	if err != nil {
		t.Fatalf("generateLock failed: %v", err)
	}

	findings := make([]Finding, 0)
	findings = append(findings, validateRules(manifest.Rules)...)
	findings = append(findings, compareManifestToLock(manifest, generatedLock, generatedLock, true)...)
	findings = append(findings, evaluateTargetPolicies(manifest.Targets, manifest.Rules)...)

	if len(findings) != 0 {
		t.Fatalf("findings = %d, want 0 for empty manifest\nfindings: %+v", len(findings), findings)
	}

	status := summarizeStatus(findings, false)
	if status != statusPass {
		t.Fatalf("status = %q, want %q", status, statusPass)
	}
}

// --- malformed JSON 테스트 ---

func TestMalformedManifestJSON(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "driftcheck.json")
	lockPath := filepath.Join(dir, "driftcheck.lock.json")

	// 잘못된 JSON 작성
	if err := os.WriteFile(manifestPath, []byte(`{invalid json`), 0o644); err != nil {
		t.Fatalf("write manifest failed: %v", err)
	}

	_, err := Audit(context.Background(), AuditConfig{
		ManifestPath:    manifestPath,
		LockPath:        lockPath,
		IncludeGit:      false,
		IncludeCanaries: false,
		IncludeEnv:      false,
	})

	if err == nil {
		t.Fatal("expected error for malformed manifest JSON, got nil")
	}
}

func TestMalformedLockfileJSON(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "driftcheck.json")
	lockPath := filepath.Join(dir, "driftcheck.lock.json")

	manifest := Manifest{
		Name: "malformed-lock",
		Targets: []Target{
			{Name: "codex", Kind: "cli", Source: "openai/codex", Version: "5.0.0"},
		},
	}
	writeTestJSON(t, manifestPath, manifest)

	// 잘못된 lockfile JSON 작성
	if err := os.WriteFile(lockPath, []byte(`not valid json!`), 0o644); err != nil {
		t.Fatalf("write lock failed: %v", err)
	}

	_, err := Audit(context.Background(), AuditConfig{
		ManifestPath:    manifestPath,
		LockPath:        lockPath,
		IncludeGit:      false,
		IncludeCanaries: false,
		IncludeEnv:      false,
	})

	if err == nil {
		t.Fatal("expected error for malformed lockfile JSON, got nil")
	}
}

// --- lock 없이 --write-lock 미사용 시 lock_missing finding 테스트 ---

func TestMissingLockWithoutWriteLockProducesFinding(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "driftcheck.json")
	lockPath := filepath.Join(dir, "driftcheck.lock.json")

	manifest := Manifest{
		Name: "no-lock",
		Targets: []Target{
			{Name: "codex", Kind: "cli", Source: "openai/codex", Version: "5.0.0"},
		},
	}
	writeTestJSON(t, manifestPath, manifest)

	report, err := Audit(context.Background(), AuditConfig{
		ManifestPath:    manifestPath,
		LockPath:        lockPath,
		WriteLock:       false,
		IncludeGit:      false,
		IncludeCanaries: false,
		IncludeEnv:      false,
	})
	if err != nil {
		t.Fatalf("audit failed: %v", err)
	}

	found := false
	for _, f := range report.Findings {
		if f.Code == "lock_missing" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected lock_missing finding, got: %+v", report.Findings)
	}
}

// --- deny rule 테스트 ---

func TestDenyRuleBlocksTarget(t *testing.T) {
	t.Parallel()

	targets := []Target{
		{Name: "banned-tool", Kind: "cli", Source: "evil.com/tool", Version: "1.0.0"},
		{Name: "safe-tool", Kind: "cli", Source: "good.com/tool", Version: "1.0.0"},
	}

	rules := []Rule{
		{Pattern: "banned-*", Decision: decisionDeny},
		{Pattern: "safe-*", Decision: decisionAllow},
	}

	findings := evaluateTargetPolicies(targets, rules)

	if len(findings) != 1 {
		t.Fatalf("findings = %d, want 1\nfindings: %+v", len(findings), findings)
	}
	if findings[0].Code != "rule_deny" {
		t.Fatalf("code = %q, want %q", findings[0].Code, "rule_deny")
	}
	if findings[0].Subject != "banned-tool" {
		t.Fatalf("subject = %q, want %q", findings[0].Subject, "banned-tool")
	}
}

// --- 복합 검증: invalid glob + invalid decision ---

func TestMultipleRuleValidationErrors(t *testing.T) {
	t.Parallel()

	rules := []Rule{
		{Pattern: "[bad-glob", Decision: decisionAllow},
		{Pattern: "valid", Decision: "nope"},
		{Pattern: "", Decision: ""},
	}

	findings := validateRules(rules)

	// [bad-glob → rule_invalid_glob
	// valid + nope → rule_invalid_decision
	// empty pattern → rule_missing_pattern
	if len(findings) != 3 {
		t.Fatalf("findings = %d, want 3\nfindings: %+v", len(findings), findings)
	}

	codes := make(map[string]int)
	for _, f := range findings {
		codes[f.Code]++
	}

	want := map[string]int{
		"rule_invalid_glob":     1,
		"rule_invalid_decision": 1,
		"rule_missing_pattern":  1,
	}
	for code, wantCount := range want {
		if codes[code] != wantCount {
			t.Fatalf("code %q count = %d, want %d", code, codes[code], wantCount)
		}
	}
}

// --- applyRules first-match-wins 테스트 ---

func TestApplyRulesFirstMatchWins(t *testing.T) {
	t.Parallel()

	target := Target{Name: "claude", Kind: "cli", Source: "anthropic/claude-code", Version: "1.0.0"}

	rules := []Rule{
		{Pattern: "claude", Decision: decisionDeny},
		{Pattern: "claude", Decision: decisionAllow},
	}

	decision := applyRules(target, rules)
	if decision != decisionDeny {
		t.Fatalf("decision = %q, want %q (first match wins)", decision, decisionDeny)
	}
}

// --- targetIdentityKey 테스트 ---

func TestTargetIdentityKeyIgnoresVersion(t *testing.T) {
	t.Parallel()

	a := Target{Name: "claude", Kind: "cli", Source: "anthropic", Version: "1.0.0"}
	b := Target{Name: "claude", Kind: "cli", Source: "anthropic", Version: "2.0.0"}

	if targetIdentityKey(a) != targetIdentityKey(b) {
		t.Fatalf("identity keys should match regardless of version: %q vs %q",
			targetIdentityKey(a), targetIdentityKey(b))
	}

	if targetKey(a) == targetKey(b) {
		t.Fatal("full keys should differ when versions differ")
	}
}

// --- 헬퍼 함수 ---

func writeTestJSON(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatalf("marshal json failed: %v", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		t.Fatalf("write %s failed: %v", path, err)
	}
}

func readTestJSON(t *testing.T, path string, target any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s failed: %v", path, err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		t.Fatalf("parse %s failed: %v", path, err)
	}
}

