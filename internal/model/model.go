package model

import "time"

// Mode is how the user pointed the tool at their work.
type Mode int

const (
	ModeUnset Mode = iota
	// FolderOfProjects: the path is a folder whose children are projects.
	FolderOfProjects
	// SingleProject: the path is one Git repository.
	SingleProject
)

func (m Mode) String() string {
	switch m {
	case FolderOfProjects:
		return "folder of projects"
	case SingleProject:
		return "single project"
	}
	return "unset"
}

// Short is the label used in the report header.
func (m Mode) Short() string {
	switch m {
	case FolderOfProjects:
		return "folder mode"
	case SingleProject:
		return "single mode"
	}
	return ""
}

// DirtyChoice is the user's answer for a repository with uncommitted changes.
type DirtyChoice int

const (
	DirtyUnset DirtyChoice = iota
	CommitAndPush
	PushOnly
	Skip
)

// Visibility of a GitHub repository the tool creates.
type Visibility int

const (
	Private Visibility = iota
	Public
)

func (v Visibility) String() string {
	if v == Public {
		return "public"
	}
	return "private"
}

// DecisionKind says which prompt a Decision answers.
type DecisionKind int

const (
	DecisionDirty DecisionKind = iota
	DecisionCreateRepo
	DecisionSaveReport
)

// Decision is an answer the user gave at a prompt during the run.
type Decision struct {
	Kind          DecisionKind
	DirtyChoice   DirtyChoice
	CommitMessage string
	CreateRepo    bool
	RepoName      string
	Visibility    Visibility
	// ApplyToAll remembers this answer for the remaining repositories.
	// It never applies to RepoName.
	ApplyToAll bool
}

// Candidate is a folder examined during discovery.
type Candidate struct {
	Path   string
	Name   string
	IsRepo bool
	// Nested is true when the folder lives inside another repository's
	// working tree; TopLevel then holds that repository's root.
	Nested   bool
	TopLevel string
	// Branch is empty when HEAD is detached.
	Branch string
	// Upstream like "origin/main"; empty when none.
	Upstream string
	Remotes  []string
	// AheadCount is commits on Branch not on Upstream; -1 when no upstream.
	AheadCount int
	Dirty      bool
	DirtyFiles []string
	// InProgressOp is "merge", "rebase", "cherry-pick" or empty.
	InProgressOp string
	// IdentityOK: user.name and user.email are both configured.
	IdentityOK bool
	// Excluded is set by the user in the review step.
	Excluded bool
}

// NeedsPush reports whether the current branch has anything to push.
func (c Candidate) NeedsPush() bool {
	return c.IsRepo && c.Branch != "" && (c.Upstream == "" || c.AheadCount > 0)
}

// HasRemote reports whether at least one remote is configured.
func (c Candidate) HasRemote() bool { return len(c.Remotes) > 0 }

// HasOrigin reports whether a remote named origin exists.
func (c Candidate) HasOrigin() bool {
	for _, r := range c.Remotes {
		if r == "origin" {
			return true
		}
	}
	return false
}

// MaxDetail is the longest Detail kept on a Result.
const MaxDetail = 200

// Result is the outcome for one Candidate.
type Result struct {
	Candidate Candidate
	Status    Status
	// Reason is required when Status is NotPushed or NotProcessed.
	Reason Reason
	// Detail is extra context (last stderr line, op name, timeout), ≤ MaxDetail.
	Detail        string
	Committed     bool
	CommitMessage string
	// CreatedRemote is the URL of a GitHub repository the tool created.
	CreatedRemote     string
	CreatedVisibility Visibility
	Duration          time.Duration
}

// SetDetail stores s trimmed to MaxDetail characters.
func (r *Result) SetDetail(s string) {
	rs := []rune(s)
	if len(rs) > MaxDetail {
		rs = rs[:MaxDetail]
	}
	r.Detail = string(rs)
}

// DirtyPolicy is the --dirty flag value.
type DirtyPolicy string

const (
	DirtyAsk      DirtyPolicy = "ask"
	DirtyCommit   DirtyPolicy = "commit"
	DirtyPushOnly DirtyPolicy = "push-only"
	DirtySkip     DirtyPolicy = "skip"
)

// Choice maps a non-ask policy to the DirtyChoice it implies.
func (p DirtyPolicy) Choice() DirtyChoice {
	switch p {
	case DirtyCommit:
		return CommitAndPush
	case DirtyPushOnly:
		return PushOnly
	case DirtySkip:
		return Skip
	}
	return DirtyUnset
}

// CreateRepoPolicy is the --create-repo flag value.
type CreateRepoPolicy string

const (
	CreateRepoAsk CreateRepoPolicy = "ask"
	CreateRepoYes CreateRepoPolicy = "yes"
	CreateRepoNo  CreateRepoPolicy = "no"
)

// DefaultTimeout is the per-push timeout when --timeout is not given.
const DefaultTimeout = 120 * time.Second

// Options are the resolved command-line flags.
type Options struct {
	Mode       Mode
	Path       string
	Dirty      DirtyPolicy
	Message    string
	CreateRepo CreateRepoPolicy
	Visibility Visibility
	NoReview   bool
	// ReportPath: "" → prompt, "-" → never save, else save there.
	ReportPath string
	Timeout    time.Duration
	Strict     bool
	NoColor    bool
	Verbose    bool
}

// DefaultOptions returns Options with every default from contracts/cli.md.
func DefaultOptions() Options {
	return Options{
		Dirty:      DirtyAsk,
		CreateRepo: CreateRepoAsk,
		Visibility: Private,
		Timeout:    DefaultTimeout,
	}
}

// Run is one execution of the tool.
type Run struct {
	Mode        Mode
	RootPath    string
	StartedAt   time.Time
	FinishedAt  time.Time
	Interrupted bool
	Options     Options
	Results     []Result
	ToolVersion string
}

// Duration of the run (zero until finished).
func (r Run) Duration() time.Duration {
	if r.FinishedAt.IsZero() {
		return 0
	}
	return r.FinishedAt.Sub(r.StartedAt)
}
