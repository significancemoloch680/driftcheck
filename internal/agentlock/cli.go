package agentlock

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

// Execute runs the agentlock CLI and returns an exit code.
func Execute(args []string, stdout io.Writer, stderr io.Writer) int {
	exitCode := exitCodeSuccess
	cmd := newRootCommand(stdout, stderr, &exitCode)
	cmd.SetArgs(args)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	if _, err := cmd.ExecuteC(); err != nil {
		code := errorCode(err)
		writeError(stdout, stderr, err, code)
		return code
	}

	return exitCode
}

func newRootCommand(stdout io.Writer, stderr io.Writer, exitCode *int) *cobra.Command {
	var manifestPath string
	var lockPath string
	var workDir string
	var writeLock bool
	var includeGit bool
	var includeCanaries bool
	var includeEnv bool
	var failOnWarning bool

	cmd := &cobra.Command{
		Use:   "agentlock",
		Short: "Audit agent manifests against a lockfile",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			report, err := Audit(context.Background(), AuditConfig{
				ManifestPath:    manifestPath,
				LockPath:        lockPath,
				WorkDir:         workDir,
				WriteLock:       writeLock,
				IncludeGit:      includeGit,
				IncludeCanaries: includeCanaries,
				IncludeEnv:      includeEnv,
				FailOnWarning:   failOnWarning,
			})
			if err != nil {
				code := errorCode(err)
				writeError(stdout, stderr, err, code)
				*exitCode = code
				return
			}

			if err := writeJSON(stdout, report); err != nil {
				writeError(stdout, stderr, err, exitCodeSystem)
				*exitCode = exitCodeSystem
				return
			}

			switch report.Status {
			case statusPass:
				*exitCode = exitCodeSuccess
			case statusWarn, statusFail:
				*exitCode = exitCodeUser
			default:
				*exitCode = exitCodeSystem
			}
		},
	}

	cmd.Flags().StringVar(&manifestPath, "manifest", defaultManifestFile, "Path to the manifest JSON file")
	cmd.Flags().StringVar(&lockPath, "lock", defaultLockFile, "Path to the lockfile JSON file")
	cmd.Flags().StringVar(&workDir, "workdir", ".", "Working directory for git collection")
	cmd.Flags().BoolVar(&writeLock, "write-lock", false, "Write a new lockfile when the lockfile is missing")
	cmd.Flags().BoolVar(&includeGit, "git", true, "Collect git evidence")
	cmd.Flags().BoolVar(&includeCanaries, "canary", true, "Run HTTP canary checks from the manifest")
	cmd.Flags().BoolVar(&includeEnv, "env", true, "Include a redacted environment hash")
	cmd.Flags().BoolVar(&failOnWarning, "fail-on-warning", false, "Return a failure exit code for warnings")

	return cmd
}

func writeJSON(out io.Writer, value any) error {
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		return fmt.Errorf("write json failed: %w", err)
	}

	return nil
}

func writeError(stdout io.Writer, stderr io.Writer, err error, code int) {
	payload := map[string]any{
		"status": "error",
		"error": map[string]any{
			"code":    code,
			"message": err.Error(),
		},
	}

	if writeJSON(stdout, payload) == nil {
		return
	}

	_ = writeJSON(stderr, payload)
}
