package integration

import (
	"context"
	"os"
	"testing"

	"github.com/aarjang/push-all-repo/internal/model"
	"github.com/stretchr/testify/assert"
)

func mkdir(p string) error { return os.MkdirAll(p, 0o755) }

func threeRepos(t *testing.T) (base string) {
	t.Helper()
	base = t.TempDir()
	for _, n := range []string{"alpha", "beta", "gamma"} {
		repo := mkRepo(t, base, n, mkBareRemote(t, base, n))
		addCommit(t, repo, "work")
	}
	return
}

func TestReviewExcludes(t *testing.T) {
	requireGit(t)
	base := threeRepos(t)
	h := newHarness()
	h.prompter.ReviewFn = func(cs []model.Candidate) []model.Candidate {
		var keep []model.Candidate
		for _, c := range cs {
			if c.Name != "alpha" {
				keep = append(keep, c)
			}
		}
		return keep
	}
	opts := folderOpts(projectsDir(base))
	opts.NoReview = false
	r, code := h.run(t, context.Background(), opts)
	assert.Equal(t, 0, code)
	assert.Equal(t, 1, h.prompter.ReviewCalls)
	assert.Equal(t, 1, h.prompter.ConfirmCalls)
	a := byName(r, "alpha")
	assert.Equal(t, model.SkippedByUser, a.Status)
	assert.Equal(t, "excluded in review", a.Detail)
	assert.Equal(t, model.Pushed, byName(r, "beta").Status)
	assert.Equal(t, model.Pushed, byName(r, "gamma").Status)
	assert.Contains(t, h.out.String(), "Skipped by user (1)")
}

// An emptied review list (Ctrl+A in the form) must stop before touching any
// repository, not walk them all as "excluded in review".
func TestReviewExcludesAllStops(t *testing.T) {
	requireGit(t)
	base := threeRepos(t)
	h := newHarness()
	h.prompter.ReviewFn = func([]model.Candidate) []model.Candidate { return nil }
	opts := folderOpts(projectsDir(base))
	opts.NoReview = false
	r, code := h.run(t, context.Background(), opts)
	assert.Equal(t, 0, code)
	assert.Equal(t, 1, h.prompter.ReviewCalls)
	assert.Equal(t, 0, h.prompter.ConfirmCalls, "no confirmation for an empty selection")
	assert.Empty(t, r.Results)
	assert.Contains(t, h.out.String(), "No repositories selected; nothing to push.")
	assert.Contains(t, h.out.String(), "Summary: 0 pushed")
	assert.NotContains(t, h.out.String(), "excluded in review")
}

func TestNoReviewFlagSkipsPrompt(t *testing.T) {
	requireGit(t)
	base := threeRepos(t)
	h := newHarness()
	_, code := h.run(t, context.Background(), folderOpts(projectsDir(base)))
	assert.Equal(t, 0, code)
	assert.Equal(t, 0, h.prompter.ReviewCalls)
	assert.Equal(t, 0, h.prompter.ConfirmCalls)
}

func TestDeclineConfirmStart(t *testing.T) {
	requireGit(t)
	base := threeRepos(t)
	h := newHarness()
	h.prompter.Confirm = false
	opts := folderOpts(projectsDir(base))
	opts.NoReview = false
	r, code := h.run(t, context.Background(), opts)
	assert.Equal(t, 0, code)
	assert.Empty(t, r.Results)
	assert.Contains(t, h.out.String(), "Summary: 0 pushed")
}
