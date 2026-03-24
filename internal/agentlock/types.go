package agentlock

const (
	defaultManifestFile = "agentlock.json"
	defaultLockFile     = "agentlock.lock.json"

	decisionAllow = "allow"
	decisionDeny  = "deny"
	decisionAsk   = "ask"

	severityInfo    = "info"
	severityWarning = "warning"
	severityError   = "error"

	statusPass = "pass"
	statusWarn = "warn"
	statusFail = "fail"

	exitCodeSuccess = 0
	exitCodeUser    = 1
	exitCodeSystem  = 2
)

// Manifest describes the input state to audit.
type Manifest struct {
	Name     string   `json:"name"`
	Targets  []Target `json:"targets"`
	Rules    []Rule   `json:"rules"`
	Canaries []Canary `json:"canaries"`
}

// Target describes one pinned agent-related dependency or service.
type Target struct {
	Name    string `json:"name"`
	Kind    string `json:"kind"`
	Source  string `json:"source"`
	Version string `json:"version"`
}

// Rule describes how a target should be handled.
type Rule struct {
	Pattern  string `json:"pattern"`
	Decision string `json:"decision"`
}

// Canary describes an HTTP health check for the manifest.
type Canary struct {
	Name           string `json:"name"`
	URL            string `json:"url"`
	ExpectedStatus int    `json:"expected_status"`
	TimeoutMillis  int    `json:"timeout_millis"`
}

// Lockfile captures the pinned state generated from a manifest.
type Lockfile struct {
	ManifestHash string         `json:"manifest_hash"`
	RulesHash    string         `json:"rules_hash"`
	GeneratedAt  string         `json:"generated_at"`
	Targets      []LockedTarget `json:"targets"`
}

// LockedTarget stores the pinned digest for a target.
type LockedTarget struct {
	Name    string `json:"name"`
	Kind    string `json:"kind"`
	Source  string `json:"source"`
	Version string `json:"version"`
	Digest  string `json:"digest"`
}

// Finding describes a concrete issue or policy result.
type Finding struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Subject  string `json:"subject"`
	Message  string `json:"message"`
	Fix      string `json:"fix"`
}

// GitInfo summarizes git evidence collected from the workspace.
type GitInfo struct {
	Present      bool   `json:"present"`
	Head         string `json:"head"`
	Dirty        bool   `json:"dirty"`
	ChangedFiles int    `json:"changed_files"`
	DiffStat     string `json:"diff_stat"`
}

// CanaryResult reports the outcome of a canary request.
type CanaryResult struct {
	Name           string `json:"name"`
	URL            string `json:"url"`
	ExpectedStatus int    `json:"expected_status"`
	StatusCode     int    `json:"status_code"`
	Healthy        bool   `json:"healthy"`
	DurationMillis int64  `json:"duration_millis"`
	Error          string `json:"error,omitempty"`
}

// EnvSnapshot records a redacted hash of the current environment.
type EnvSnapshot struct {
	Hash     string `json:"hash"`
	Total    int    `json:"total"`
	Redacted int    `json:"redacted"`
}

// Summary aggregates the report counts.
type Summary struct {
	Targets  int `json:"targets"`
	Rules    int `json:"rules"`
	Canaries int `json:"canaries"`
	Findings int `json:"findings"`
	Errors   int `json:"errors"`
	Warnings int `json:"warnings"`
}

// Report is the machine-readable result emitted by the CLI.
type Report struct {
	Status        string         `json:"status"`
	Summary       Summary        `json:"summary"`
	ManifestPath  string         `json:"manifest_path"`
	LockPath      string         `json:"lock_path"`
	ManifestHash  string         `json:"manifest_hash"`
	LockHash      string         `json:"lock_hash"`
	ConfigHash    string         `json:"config_hash"`
	Env           EnvSnapshot    `json:"env"`
	Git           GitInfo        `json:"git"`
	Canaries      []CanaryResult `json:"canaries"`
	Findings      []Finding      `json:"findings"`
	GeneratedLock *Lockfile      `json:"generated_lock,omitempty"`
}

// AuditConfig controls one audit run.
type AuditConfig struct {
	ManifestPath    string
	LockPath        string
	WorkDir         string
	WriteLock       bool
	IncludeGit      bool
	IncludeCanaries bool
	IncludeEnv      bool
	FailOnWarning   bool
}
