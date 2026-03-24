package driftcheck

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

func hashString(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func hashJSON(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("marshal json failed: %w", err)
	}

	return hashString(string(data)), nil
}

// targetIdentityKey returns a key that identifies a target by name, kind, and source
// without including the version. Used for matching targets between manifest and lockfile
// so that version changes are detected as explicit mismatches rather than missing targets.
func targetIdentityKey(target Target) string {
	return strings.Join([]string{target.Name, target.Kind, target.Source}, "|")
}

// lockedTargetIdentityKey returns a version-independent identity key for a locked target.
func lockedTargetIdentityKey(target LockedTarget) string {
	return strings.Join([]string{target.Name, target.Kind, target.Source}, "|")
}

// targetKey returns a full key including version, used for digest computation.
func targetKey(target Target) string {
	return strings.Join([]string{target.Name, target.Kind, target.Source, target.Version}, "|")
}

// lockedTargetKey returns a full key including version for a locked target.
func lockedTargetKey(target LockedTarget) string {
	return strings.Join([]string{target.Name, target.Kind, target.Source, target.Version}, "|")
}

func targetDigest(target Target) string {
	return hashString(targetKey(target))
}

func normalizeManifest(manifest Manifest) Manifest {
	normalized := manifest

	normalized.Targets = append([]Target(nil), manifest.Targets...)
	sort.SliceStable(normalized.Targets, func(i, j int) bool {
		return targetKey(normalized.Targets[i]) < targetKey(normalized.Targets[j])
	})

	normalized.Rules = append([]Rule(nil), manifest.Rules...)
	sort.SliceStable(normalized.Rules, func(i, j int) bool {
		return normalized.Rules[i].Pattern < normalized.Rules[j].Pattern || (normalized.Rules[i].Pattern == normalized.Rules[j].Pattern && normalized.Rules[i].Decision < normalized.Rules[j].Decision)
	})

	normalized.Canaries = append([]Canary(nil), manifest.Canaries...)
	sort.SliceStable(normalized.Canaries, func(i, j int) bool {
		return normalized.Canaries[i].Name < normalized.Canaries[j].Name
	})

	return normalized
}

func normalizeLock(lock Lockfile) Lockfile {
	normalized := lock
	normalized.Targets = append([]LockedTarget(nil), lock.Targets...)
	sort.SliceStable(normalized.Targets, func(i, j int) bool {
		return lockedTargetKey(normalized.Targets[i]) < lockedTargetKey(normalized.Targets[j])
	})
	return normalized
}
