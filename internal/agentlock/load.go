package agentlock

import (
	"fmt"
	"time"
)

func loadManifest(path string) (Manifest, []byte, error) {
	var manifest Manifest
	data, err := readJSONFile(path, &manifest)
	if err != nil {
		return Manifest{}, nil, err
	}

	return manifest, data, nil
}

func loadLock(path string) (Lockfile, []byte, error) {
	var lock Lockfile
	data, err := readJSONFile(path, &lock)
	if err != nil {
		return Lockfile{}, nil, err
	}

	return lock, data, nil
}

func generateLock(manifest Manifest, generatedAt time.Time) (Lockfile, error) {
	normalized := normalizeManifest(manifest)
	manifestHash, err := hashJSON(normalized)
	if err != nil {
		return Lockfile{}, fmt.Errorf("hash manifest failed: %w", err)
	}

	rulesHash, err := hashJSON(normalized.Rules)
	if err != nil {
		return Lockfile{}, fmt.Errorf("hash rules failed: %w", err)
	}

	targets := make([]LockedTarget, 0, len(normalized.Targets))
	for _, target := range normalized.Targets {
		targets = append(targets, LockedTarget{
			Name:    target.Name,
			Kind:    target.Kind,
			Source:  target.Source,
			Version: target.Version,
			Digest:  targetDigest(target),
		})
	}

	return Lockfile{
		ManifestHash: manifestHash,
		RulesHash:    rulesHash,
		GeneratedAt:  generatedAt.UTC().Format(time.RFC3339),
		Targets:      targets,
	}, nil
}
