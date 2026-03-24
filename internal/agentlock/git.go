package agentlock

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

func collectGitInfo(ctx context.Context, workDir string) (GitInfo, error) {
	dir := workDir
	if dir == "" {
		dir = "."
	}

	if _, err := exec.CommandContext(ctx, "git", "-C", dir, "rev-parse", "--is-inside-work-tree").Output(); err != nil {
		return GitInfo{Present: false}, nil
	}

	head, err := commandOutput(ctx, dir, "git", "rev-parse", "--short", "HEAD")
	if err != nil {
		return GitInfo{}, fmt.Errorf("collect git head failed: %w", err)
	}

	status, err := commandOutput(ctx, dir, "git", "status", "--porcelain")
	if err != nil {
		return GitInfo{}, fmt.Errorf("collect git status failed: %w", err)
	}

	diffStat, err := commandOutput(ctx, dir, "git", "diff", "--stat")
	if err != nil {
		return GitInfo{}, fmt.Errorf("collect git diff failed: %w", err)
	}

	changed := 0
	if strings.TrimSpace(status) != "" {
		changed = len(strings.Split(strings.TrimSpace(status), "\n"))
	}

	return GitInfo{
		Present:      true,
		Head:         strings.TrimSpace(head),
		Dirty:        changed > 0,
		ChangedFiles: changed,
		DiffStat:     strings.TrimSpace(diffStat),
	}, nil
}

func commandOutput(ctx context.Context, dir string, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stdout

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s %s failed: %w", filepath.Base(name), strings.Join(args, " "), err)
	}

	return stdout.String(), nil
}

