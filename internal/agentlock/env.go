package agentlock

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
)

func snapshotEnv(entries []string) EnvSnapshot {
	normalized := make([]string, 0, len(entries))
	redactedCount := 0

	for _, entry := range entries {
		key, value, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}

		if shouldRedactEnvKey(key) {
			value = "<redacted>"
			redactedCount++
		}

		normalized = append(normalized, key+"="+value)
	}

	sort.Strings(normalized)
	sum := sha256.Sum256([]byte(strings.Join(normalized, "\n")))

	return EnvSnapshot{
		Hash:     hex.EncodeToString(sum[:]),
		Total:    len(normalized),
		Redacted: redactedCount,
	}
}

func shouldRedactEnvKey(key string) bool {
	upper := strings.ToUpper(key)
	switch {
	case strings.Contains(upper, "SECRET"):
		return true
	case strings.Contains(upper, "TOKEN"):
		return true
	case strings.Contains(upper, "PASSWORD"):
		return true
	case strings.Contains(upper, "PASS"):
		return true
	case strings.Contains(upper, "KEY"):
		return true
	case strings.Contains(upper, "COOKIE"):
		return true
	case strings.Contains(upper, "AUTH"):
		return true
	default:
		return false
	}
}
