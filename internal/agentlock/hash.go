package agentlock

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
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

func targetKey(target Target) string {
	return strings.Join([]string{target.Name, target.Kind, target.Source, target.Version}, "|")
}

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

func pathBase(name string) string {
	if name == "" {
		return ""
	}
	return filepath.Base(name)
}

