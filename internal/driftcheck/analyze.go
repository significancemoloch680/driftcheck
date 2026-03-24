package driftcheck

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Audit compares the manifest, lockfile, git state, env snapshot, and canary results.
func Audit(ctx context.Context, cfg AuditConfig) (Report, error) {
	if cfg.ManifestPath == "" {
		cfg.ManifestPath = defaultManifestFile
	}
	if cfg.LockPath == "" {
		cfg.LockPath = defaultLockFile
	}
	if cfg.WorkDir == "" {
		cfg.WorkDir = "."
	}

	manifest, _, err := loadManifest(cfg.ManifestPath)
	if err != nil {
		return Report{}, classifyLoadError(cfg.ManifestPath, err)
	}

	generatedLock, err := generateLock(manifest, time.Now())
	if err != nil {
		return Report{}, newSystemError("generate lock failed", err)
	}

	var lock Lockfile
	lockExists := true
	if loadedLock, _, err := loadLock(cfg.LockPath); err != nil {
		if isMissingFileError(err) && cfg.WriteLock {
			if err := writeJSONFile(cfg.LockPath, generatedLock); err != nil {
				return Report{}, newSystemError("write lock failed", err)
			}
			lock = generatedLock
			lockExists = false
		} else if isMissingFileError(err) {
			lockExists = false
		} else {
			return Report{}, classifyLoadError(cfg.LockPath, err)
		}
	} else {
		lock = loadedLock
	}

	findings := make([]Finding, 0)
	findings = append(findings, validateRules(manifest.Rules)...)
	findings = append(findings, compareManifestToLock(manifest, lock, generatedLock, lockExists)...)
	findings = append(findings, evaluateTargetPolicies(manifest.Targets, manifest.Rules)...)

	env := EnvSnapshot{}
	if cfg.IncludeEnv {
		env = snapshotEnv(os.Environ())
	}

	gitInfo := GitInfo{}
	if cfg.IncludeGit {
		gitInfo, err = collectGitInfo(ctx, cfg.WorkDir)
		if err != nil {
			return Report{}, newSystemError("collect git info failed", err)
		}
	}

	canaries := make([]CanaryResult, 0, len(manifest.Canaries))
	if cfg.IncludeCanaries {
		for _, canary := range manifest.Canaries {
			result, err := runCanary(ctx, canary)
			if err != nil {
				return Report{}, newSystemError("run canary failed", err)
			}
			canaries = append(canaries, result)
			if !result.Healthy {
				findings = append(findings, Finding{
					Code:     "canary_failed",
					Severity: severityError,
					Subject:  canary.Name,
					Message:  fmt.Sprintf("Canary %s returned %d instead of %d.", canary.Name, result.StatusCode, result.ExpectedStatus),
					Fix:      "Check the endpoint, network access, and service health before rerunning the audit.",
				})
			}
		}
	}

	manifestHash, err := hashJSON(normalizeManifest(manifest))
	if err != nil {
		return Report{}, newSystemError("hash manifest failed", err)
	}

	lockHash, err := hashJSON(normalizeLock(lock))
	if err != nil {
		return Report{}, newSystemError("hash lock failed", err)
	}

	configHash, err := hashJSON(struct {
		ManifestHash string `json:"manifest_hash"`
		LockHash     string `json:"lock_hash"`
		EnvHash      string `json:"env_hash"`
		GitHead      string `json:"git_head"`
	}{
		ManifestHash: manifestHash,
		LockHash:     lockHash,
		EnvHash:      env.Hash,
		GitHead:      gitInfo.Head,
	})
	if err != nil {
		return Report{}, newSystemError("hash config failed", err)
	}

	summary := summarize(findings, len(manifest.Targets), len(manifest.Rules), len(canaries))
	status := summarizeStatus(findings, cfg.FailOnWarning)

	report := Report{
		Status:       status,
		Summary:      summary,
		ManifestPath: cfg.ManifestPath,
		LockPath:     cfg.LockPath,
		ManifestHash: manifestHash,
		LockHash:     lockHash,
		ConfigHash:   configHash,
		Env:          env,
		Git:          gitInfo,
		Canaries:     canaries,
		Findings:     findings,
	}

	if cfg.WriteLock {
		report.GeneratedLock = &generatedLock
	}

	if !lockExists && !cfg.WriteLock {
		report.Findings = append(report.Findings, Finding{
			Code:     "lock_missing",
			Severity: severityError,
			Subject:  filepath.Base(cfg.LockPath),
			Message:  "The lockfile does not exist.",
			Fix:      "Run driftcheck with --write-lock to bootstrap the lockfile from the manifest.",
		})
		report.Summary = summarize(report.Findings, len(manifest.Targets), len(manifest.Rules), len(canaries))
		report.Status = summarizeStatus(report.Findings, cfg.FailOnWarning)
	}

	return report, nil
}

func summarize(findings []Finding, targetCount, ruleCount, canaryCount int) Summary {
	errors := 0
	warnings := 0
	for _, finding := range findings {
		switch finding.Severity {
		case severityError:
			errors++
		case severityWarning:
			warnings++
		}
	}

	return Summary{
		Targets:  targetCount,
		Rules:    ruleCount,
		Canaries: canaryCount,
		Findings: len(findings),
		Errors:   errors,
		Warnings: warnings,
	}
}

func summarizeStatus(findings []Finding, failOnWarning bool) string {
	hasError := false
	hasWarning := false
	for _, finding := range findings {
		switch finding.Severity {
		case severityError:
			hasError = true
		case severityWarning:
			hasWarning = true
		}
	}

	switch {
	case hasError:
		return statusFail
	case hasWarning && failOnWarning:
		return statusFail
	case hasWarning:
		return statusWarn
	default:
		return statusPass
	}
}

func validateRules(rules []Rule) []Finding {
	findings := make([]Finding, 0)
	for _, rule := range rules {
		if rule.Pattern == "" {
			findings = append(findings, Finding{
				Code:     "rule_missing_pattern",
				Severity: severityError,
				Subject:  rule.Decision,
				Message:  "A rule pattern is empty.",
				Fix:      "Set a non-empty glob pattern on the rule.",
			})
			continue
		}

		switch rule.Decision {
		case decisionAllow, decisionDeny, decisionAsk:
		default:
			findings = append(findings, Finding{
				Code:     "rule_invalid_decision",
				Severity: severityError,
				Subject:  rule.Pattern,
				Message:  "A rule decision is invalid.",
				Fix:      "Use allow, deny, or ask.",
			})
		}
	}

	return findings
}

func compareManifestToLock(manifest Manifest, lock Lockfile, generatedLock Lockfile, lockExists bool) []Finding {
	findings := make([]Finding, 0)
	if !lockExists {
		return findings
	}

	if lock.ManifestHash != "" && lock.ManifestHash != generatedLock.ManifestHash {
		findings = append(findings, Finding{
			Code:     "lock_manifest_hash_mismatch",
			Severity: severityError,
			Subject:  "manifest_hash",
			Message:  "The lockfile manifest hash does not match the generated manifest hash.",
			Fix:      "Regenerate the lockfile from the current manifest.",
		})
	}

	if lock.RulesHash != "" && lock.RulesHash != generatedLock.RulesHash {
		findings = append(findings, Finding{
			Code:     "lock_rules_hash_mismatch",
			Severity: severityError,
			Subject:  "rules_hash",
			Message:  "The lockfile rules hash does not match the generated rules hash.",
			Fix:      "Regenerate the lockfile from the current manifest rules.",
		})
	}

	lockMap := make(map[string]LockedTarget, len(lock.Targets))
	for _, target := range lock.Targets {
		lockMap[targetKey(Target{
			Name:    target.Name,
			Kind:    target.Kind,
			Source:  target.Source,
			Version: target.Version,
		})] = target
	}

	for _, target := range manifest.Targets {
		key := targetKey(target)
		locked, ok := lockMap[key]
		if !ok {
			findings = append(findings, Finding{
				Code:     "lock_missing_target",
				Severity: severityError,
				Subject:  target.Name,
				Message:  "The lockfile does not include this target.",
				Fix:      "Regenerate the lockfile so the target is pinned.",
			})
			continue
		}

		if locked.Digest != targetDigest(target) {
			findings = append(findings, Finding{
				Code:     "lock_digest_mismatch",
				Severity: severityError,
				Subject:  target.Name,
				Message:  "The lock digest does not match the manifest target.",
				Fix:      "Regenerate the lockfile and commit the updated digest.",
			})
		}
	}

	manifestMap := make(map[string]Target, len(manifest.Targets))
	for _, target := range manifest.Targets {
		manifestMap[targetKey(target)] = target
	}

	for _, locked := range lock.Targets {
		key := lockedTargetKey(locked)
		if _, ok := manifestMap[key]; !ok {
			findings = append(findings, Finding{
				Code:     "lock_extra_target",
				Severity: severityError,
				Subject:  locked.Name,
				Message:  "The lockfile contains an extra target that is not in the manifest.",
				Fix:      "Delete the stale lock entry or regenerate the lockfile from the manifest.",
			})
		}
	}

	return findings
}

func evaluateTargetPolicies(targets []Target, rules []Rule) []Finding {
	findings := make([]Finding, 0)
	for _, target := range targets {
		decision := applyRules(target, rules)
		switch decision {
		case decisionAllow:
			continue
		case decisionAsk:
			findings = append(findings, Finding{
				Code:     "rule_ask",
				Severity: severityWarning,
				Subject:  target.Name,
				Message:  fmt.Sprintf("Target %s requires manual review.", target.Name),
				Fix:      "Review the target and confirm that the ask rule is acceptable.",
			})
		case decisionDeny:
			findings = append(findings, Finding{
				Code:     "rule_deny",
				Severity: severityError,
				Subject:  target.Name,
				Message:  fmt.Sprintf("Target %s matches a deny rule.", target.Name),
				Fix:      "Remove or rename the denied target before you refresh the lockfile.",
			})
		default:
			continue
		}

	}

	return findings
}

func applyRules(target Target, rules []Rule) string {
	subjects := []string{target.Name, target.Kind, target.Source}
	for _, rule := range rules {
		for _, subject := range subjects {
			if subject == "" {
				continue
			}
			match, err := filepath.Match(rule.Pattern, subject)
			if err != nil {
				continue
			}
			if match {
				return strings.ToLower(rule.Decision)
			}
		}
	}

	return decisionAllow
}

func classifyLoadError(path string, err error) error {
	return newUserError(fmt.Sprintf("load %s failed", path), err)
}

func isMissingFileError(err error) bool {
	return errors.Is(err, os.ErrNotExist)
}
