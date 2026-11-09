package gitx

import (
	"context"
	"errors"
	"strings"

	"github.com/aarjang/push-all-repo/internal/model"
)

// OpError is a classified failure of a mutating git operation.
type OpError struct {
	Reason model.Reason
	Detail string
}

func (e *OpError) Error() string { return e.Reason.TextWith(e.Detail) }

type pattern struct {
	subs   []string
	reason model.Reason
}

// Order matters: first hit wins (contracts/git-commands.md).
var patterns = []pattern{
	{[]string{"permission denied", "authentication failed", "could not read username", "could not read password", "publickey", "403", "terminal prompts disabled", "invalid username or token"}, model.AuthFailed},
	{[]string{"rejected", "non-fast-forward", "fetch first", "failed to push some refs"}, model.Rejected},
	{[]string{"could not resolve host", "connection timed out", "connection refused", "network is unreachable", "unable to access", "ssh: connect to host", "does not appear to be a git repository", "could not read from remote repository"}, model.NetworkError},
}

// Classify maps an error and its stderr to a Reason and optional detail.
// Timeout and cancellation win over text matching; the fallback is GitError
// with the last non-empty stderr line as detail.
func Classify(err error, stderr string) (model.Reason, string) {
	var ee *ExitError
	if errors.As(err, &ee) {
		if ee.TimedOut {
			return model.Timeout, ""
		}
		if ee.Cancelled {
			return model.Interrupted, ""
		}
		if stderr == "" {
			stderr = ee.Stderr
		}
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return model.Timeout, ""
	}
	if errors.Is(err, context.Canceled) {
		return model.Interrupted, ""
	}
	low := strings.ToLower(stderr)
	for _, p := range patterns {
		for _, s := range p.subs {
			if strings.Contains(low, s) {
				return p.reason, ""
			}
		}
	}
	detail := LastLine(stderr)
	if detail == "" && err != nil {
		detail = err.Error()
	}
	return model.GitError, detail
}
