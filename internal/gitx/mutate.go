package gitx

import (
	"context"
	"fmt"
	"time"

	"github.com/aarjang/push-all-repo/internal/model"
)

// MutateTimeout bounds local operations (add, commit).
const MutateTimeout = 60 * time.Second

// AddAll stages every change that is not ignored (git add -A).
func (g *Git) AddAll(ctx context.Context, path string) error {
	_, stderr, err := g.R.Run(ctx, path, MutateTimeout, "add", "-A")
	if err != nil {
		return &OpError{Reason: model.CommitFailed, Detail: LastLine(stderr)}
	}
	return nil
}

// Commit records staged changes. Hooks run; --no-verify is never used.
func (g *Git) Commit(ctx context.Context, path, message string) error {
	_, stderr, err := g.R.Run(ctx, path, MutateTimeout, "commit", "-m", message)
	if err != nil {
		reason, detail := Classify(err, stderr)
		if reason == model.GitError {
			reason = model.CommitFailed
		}
		return &OpError{Reason: reason, Detail: detail}
	}
	return nil
}

// Push sends the current branch to its upstream, or publishes it to origin
// when it has none. It never forces.
func (g *Git) Push(ctx context.Context, c model.Candidate, timeout time.Duration) error {
	var args []string
	switch {
	case c.Upstream != "":
		args = []string{"push"}
	case c.HasOrigin():
		args = []string{"push", "--set-upstream", "origin", c.Branch}
	default:
		return &OpError{Reason: model.NoRemote, Detail: "no upstream and no remote named origin"}
	}
	_, stderr, err := g.R.Run(ctx, c.Path, timeout, args...)
	if err == nil {
		return nil
	}
	reason, detail := Classify(err, stderr)
	if reason == model.Timeout {
		detail = fmt.Sprintf("%d s", int(timeout/time.Second))
	}
	return &OpError{Reason: reason, Detail: detail}
}
