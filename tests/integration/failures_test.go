package integration

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/aarjang/push-all-repo/internal/github"
	"github.com/aarjang/push-all-repo/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFailureIsolation(t *testing.T) {
	requireGit(t)
	base := t.TempDir()
	alpha := mkRepo(t, base, "alpha", mkBareRemote(t, base, "alpha"))
	addCommit(t, alpha, "again")
	gitRun(t, alpha, "remote", "set-url", "origin", filepath.Join(base, "does-not-exist.git"))
	betaRemote := mkBareRemote(t, base, "beta")
	beta := mkRepo(t, base, "beta", betaRemote)
	addCommit(t, beta, "ok")

	h := newHarness()
	r, code := h.run(t, context.Background(), folderOpts(projectsDir(base)))
	assert.Equal(t, 1, code)
	a := byName(r, "alpha")
	require.Equal(t, model.NotPushed, a.Status)
	assert.Contains(t, []model.Reason{model.NetworkError, model.GitError}, a.Reason)
	assert.Equal(t, model.Pushed, byName(r, "beta").Status, "the next repository is still processed")
	assert.Equal(t, localHead(t, beta), remoteHead(t, betaRemote))
}

func TestDetachedHead(t *testing.T) {
	requireGit(t)
	base := t.TempDir()
	gamma := mkRepo(t, base, "gamma", mkBareRemote(t, base, "gamma"))
	detach(t, gamma)
	h := newHarness()
	r, code := h.run(t, context.Background(), folderOpts(projectsDir(base)))
	assert.Equal(t, 1, code)
	res := byName(r, "gamma")
	assert.Equal(t, model.NotPushed, res.Status)
	assert.Equal(t, model.DetachedHead, res.Reason)
	assert.Contains(t, h.out.String(), "no current branch (detached HEAD)")
}

func TestMergeInProgress(t *testing.T) {
	requireGit(t)
	base := t.TempDir()
	repo := mkRepo(t, base, "conflicted", mkBareRemote(t, base, "conflicted"))
	startMerge(t, repo)
	h := newHarness()
	r, code := h.run(t, context.Background(), folderOpts(projectsDir(base)))
	assert.Equal(t, 1, code)
	res := byName(r, "conflicted")
	assert.Equal(t, model.OperationInProgress, res.Reason)
	assert.Equal(t, "merge", res.Detail)
	assert.Contains(t, h.out.String(), "merge in progress")
}

func TestRemoteWithoutOriginAndNoUpstream(t *testing.T) {
	requireGit(t)
	base := t.TempDir()
	repo := mkRepo(t, base, "odd", "")
	gitRun(t, repo, "remote", "add", "backup", mkBareRemote(t, base, "odd"))
	h := newHarness()
	r, code := h.run(t, context.Background(), folderOpts(projectsDir(base)))
	assert.Equal(t, 1, code)
	res := byName(r, "odd")
	assert.Equal(t, model.NotPushed, res.Status)
	assert.Equal(t, model.NoRemoteDeclined, res.Reason, "falls into the no-remote flow; --create-repo no declines")
	assert.Contains(t, res.Detail, "no remote named origin")

	// With a GitHub login and --create-repo yes the flow creates origin.
	h2 := newHarness()
	h2.gh = &github.FakeClient{Installed: true, Logged: true}
	opts := folderOpts(projectsDir(base))
	opts.CreateRepo = model.CreateRepoYes
	r2, code2 := h2.run(t, context.Background(), opts)
	assert.Equal(t, 0, code2)
	assert.Equal(t, model.Pushed, byName(r2, "odd").Status)
	require.Len(t, h2.gh.Calls, 1)
	assert.Equal(t, "odd", h2.gh.Calls[0].Name)
}

func TestRejectedPush(t *testing.T) {
	requireGit(t)
	base := t.TempDir()
	remote := mkBareRemote(t, base, "shared")
	a := mkRepo(t, base, "shared", remote)
	// Another clone pushes first so our push is non-fast-forward.
	other := filepath.Join(base, "other")
	gitRun(t, base, "clone", "-q", remote, other)
	gitRun(t, other, "config", "user.name", "Other")
	gitRun(t, other, "config", "user.email", "other@example.com")
	gitRun(t, other, "config", "commit.gpgsign", "false")
	addCommit(t, other, "theirs")
	gitRun(t, other, "push", "-q", "origin", "main")
	addCommit(t, a, "ours")

	h := newHarness()
	r, code := h.run(t, context.Background(), folderOpts(projectsDir(base)))
	assert.Equal(t, 1, code)
	assert.Equal(t, model.Rejected, byName(r, "shared").Reason)
}
