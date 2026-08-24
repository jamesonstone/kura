package worktree

import "time"

const sweepReportSchemaVersion = 1

type SweepState string

const (
	SweepRemoveReady        SweepState = "remove-ready"
	SweepMergedLocalFiles   SweepState = "merged-local-files"
	SweepMergedLocalCommits SweepState = "merged-local-commits"
	SweepProtectedActive    SweepState = "protected-active"
	SweepUnproven           SweepState = "unproven-not-merged"
	SweepStaleMetadata      SweepState = "stale-metadata"
)

var sweepStateOrder = []SweepState{
	SweepRemoveReady,
	SweepMergedLocalFiles,
	SweepMergedLocalCommits,
	SweepProtectedActive,
	SweepUnproven,
	SweepStaleMetadata,
}

type SweepOptions struct {
	Auto         bool
	Interactive  bool
	JSON         bool
	DryRun       bool
	ConfigPath   string
	Roots        []string
	ProjectRoots []string
	ExcludeRoots []string
	Only         SweepState
	Sort         string
	NoSizes      bool
	Color        string
	Jobs         int
	Timeout      time.Duration
	Verbose      bool
	Explain      string
}

type SweepConfig struct {
	ConfigPath   string
	Roots        []string
	ProjectRoots []string
	ExcludeRoots []string
	ProcessCheck string
	Jobs         int
	Timeout      time.Duration
	Sizes        bool
	SizeJobs     int
	StateRoot    string
	processPaths []string
	processError string
}

type SweepProcessEvidence struct {
	State  string `json:"state"`
	Detail string `json:"detail,omitempty"`
}

type SweepStatus struct {
	Lines       []string `json:"lines,omitempty"`
	Tracked     int      `json:"tracked"`
	Staged      int      `json:"staged"`
	Untracked   int      `json:"untracked"`
	Ignored     int      `json:"ignored"`
	Submodules  int      `json:"submodules"`
	Fingerprint string   `json:"fingerprint,omitempty"`
}

func (status SweepStatus) Dirty() bool {
	return len(status.Lines) != 0
}

type SweepCandidate struct {
	ID              string               `json:"id"`
	Repository      string               `json:"repository"`
	CommonDir       string               `json:"common_dir"`
	PrimaryPath     string               `json:"primary_path"`
	Path            string               `json:"path"`
	Branch          string               `json:"branch,omitempty"`
	HeadOID         string               `json:"head_oid,omitempty"`
	DefaultBranch   string               `json:"default_branch,omitempty"`
	State           SweepState           `json:"state"`
	Reason          string               `json:"reason"`
	Detail          string               `json:"detail,omitempty"`
	Selectable      bool                 `json:"selectable"`
	AutoRemovable   bool                 `json:"auto_removable"`
	ForceWorktree   bool                 `json:"force_worktree"`
	ForceBranch     bool                 `json:"force_branch"`
	SizeBytes       int64                `json:"size_bytes,omitempty"`
	LastUpdated     string               `json:"last_updated,omitempty"`
	Stale           bool                 `json:"stale"`
	StaleRetirable  bool                 `json:"stale_retirable"`
	Status          SweepStatus          `json:"status"`
	ExtraCommits    []string             `json:"extra_commits,omitempty"`
	ProcessEvidence SweepProcessEvidence `json:"process_evidence"`
	PullRequest     *SyncPullRequest     `json:"pull_request,omitempty"`
	Snapshot        string               `json:"snapshot"`
}

type SweepFailure struct {
	Operation  string `json:"operation"`
	Repository string `json:"repository,omitempty"`
	Path       string `json:"path,omitempty"`
	Error      string `json:"error"`
}

type SweepAction struct {
	CandidateID string `json:"candidate_id"`
	Path        string `json:"path"`
	Action      string `json:"action"`
	Status      string `json:"status"`
	Detail      string `json:"detail,omitempty"`
}

type SweepReport struct {
	SchemaVersion int              `json:"schema_version"`
	RunID         string           `json:"run_id"`
	Result        string           `json:"result"`
	GeneratedAt   string           `json:"generated_at"`
	ConfigPath    string           `json:"config_path,omitempty"`
	Roots         []string         `json:"roots"`
	ProjectRoots  []string         `json:"project_roots"`
	Candidates    []SweepCandidate `json:"candidates"`
	Failures      []SweepFailure   `json:"failures"`
	Actions       []SweepAction    `json:"actions"`
}

func (report *SweepReport) addFailure(operation, repository, path string, err error) {
	report.Failures = append(report.Failures, SweepFailure{
		Operation: operation, Repository: repository, Path: path, Error: err.Error(),
	})
}

func sweepStateLabel(state SweepState) string {
	switch state {
	case SweepRemoveReady:
		return "REMOVE READY"
	case SweepMergedLocalFiles:
		return "MERGED + LOCAL FILES"
	case SweepMergedLocalCommits:
		return "MERGED + LOCAL COMMITS"
	case SweepProtectedActive:
		return "PROTECTED / ACTIVE"
	case SweepStaleMetadata:
		return "STALE METADATA"
	default:
		return "UNPROVEN / NOT MERGED"
	}
}
