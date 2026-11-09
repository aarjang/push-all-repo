package model

// Status is the outcome class of one repository.
type Status int

const (
	Pushed Status = iota
	UpToDate
	NotPushed
	SkippedByUser
	NotProcessed
)

// StatusOrder is the fixed display order of groups in every report.
var StatusOrder = []Status{Pushed, UpToDate, NotPushed, SkippedByUser, NotProcessed}

// Title is the group heading used in reports.
func (s Status) Title() string {
	switch s {
	case Pushed:
		return "Pushed"
	case UpToDate:
		return "Already up to date"
	case NotPushed:
		return "Not pushed"
	case SkippedByUser:
		return "Skipped by user"
	case NotProcessed:
		return "Not processed"
	}
	return "Unknown"
}

func (s Status) String() string { return s.Title() }

// Reason explains why a repository was not pushed.
type Reason int

const (
	ReasonNone Reason = iota
	NotARepo
	NestedRepo
	NoRemote
	NoRemoteLoginRequired
	NoRemoteDeclined
	CreateRepoFailed
	DetachedHead
	OperationInProgress
	IdentityMissing
	CommitFailed
	AuthFailed
	Rejected
	NetworkError
	Timeout
	Interrupted
	GitError
	// AnswerRequired: a prompt was needed but no terminal is attached.
	AnswerRequired
)

// AllReasons lists every Reason except ReasonNone, for tests.
var AllReasons = []Reason{
	NotARepo, NestedRepo, NoRemote, NoRemoteLoginRequired, NoRemoteDeclined,
	CreateRepoFailed, DetachedHead, OperationInProgress, IdentityMissing,
	CommitFailed, AuthFailed, Rejected, NetworkError, Timeout, Interrupted, GitError, AnswerRequired,
}

// Text is the human string for the reason without any detail.
func (r Reason) Text() string {
	switch r {
	case NotARepo:
		return "not a Git repository"
	case NestedRepo:
		return "inside another repository (not a top-level project)"
	case NoRemote:
		return "no remote configured"
	case NoRemoteLoginRequired:
		return "no remote (GitHub login required to create one)"
	case NoRemoteDeclined:
		return "no remote (user declined to create one)"
	case CreateRepoFailed:
		return "could not create GitHub repository"
	case DetachedHead:
		return "no current branch (detached HEAD)"
	case OperationInProgress:
		return "operation in progress"
	case IdentityMissing:
		return "cannot commit: Git identity not configured"
	case CommitFailed:
		return "commit failed"
	case AuthFailed:
		return "authentication failed"
	case Rejected:
		return "push rejected by remote (fetch first / non-fast-forward)"
	case NetworkError:
		return "could not reach remote"
	case Timeout:
		return "push exceeded the timeout"
	case Interrupted:
		return "interrupted by user"
	case GitError:
		return "git error"
	case AnswerRequired:
		return "answer required (no terminal)"
	}
	return ""
}

// TextWith renders the reason together with its detail:
// Timeout → "push exceeded <detail>", OperationInProgress → "<detail> in progress",
// everything else → Text() plus ": <detail>" when detail is non-empty.
func (r Reason) TextWith(detail string) string {
	switch r {
	case Timeout:
		if detail != "" {
			return "push exceeded " + detail
		}
		return r.Text()
	case OperationInProgress:
		if detail != "" {
			return detail + " in progress"
		}
		return r.Text()
	}
	if detail == "" {
		return r.Text()
	}
	return r.Text() + ": " + detail
}

func (r Reason) String() string { return r.Text() }
