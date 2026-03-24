package agentlock

import (
	"encoding/json"
	"fmt"
	"os"
)

func readJSONFile(path string, target any) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s failed: %w", path, err)
	}

	if err := json.Unmarshal(data, target); err != nil {
		return nil, fmt.Errorf("parse %s failed: %w", path, err)
	}

	return data, nil
}

func writeJSONFile(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json failed: %w", err)
	}

	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("write %s failed: %w", path, err)
	}

	return nil
}

