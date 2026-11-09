package integration

import (
	"context"
	"testing"

	"github.com/aarjang/push-all-repo/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func dirtyFixture(t *testing.T) (base, repo, remote string) {
	t.Helper()
	base = t.TempDir()
	remote = mkBareRemote(t, base, "beta")
	repo = mkRepo(t, base, "beta", remote)
	makeDirty(t, repo)
	return
}

func TestDirtyCommitAndPush(t *testing.T) {
	requireGit(t)
	base, repo, remote := dirtyFixture(t)
	h := newHarness()
	h.prompter.Dirty = []model.Decision{{DirtyChoice: model.CommitAndPush, CommitMessage: "wip: from test"}}
	opts := folderOpts(projectsDir(base))
	opts.Dirty = model.DirtyAsk

	r, code := h.run(t, context.Background(), opts)
	assert.Equal(t, 0, code)
	res := byName(r, "beta")
	assert.Equal(t, model.Pushed, res.Status)
	assert.True(t, res.Committed)
	assert.Equal(t, "wip: from test", res.CommitMessage)
	assert.Equal(t, "wip: from test", remoteLastMessage(t, remote))
	assert.Equal(t, localHead(t, repo), remoteHead(t, remote))
	assert.Equal(t, 1, h.prompter.DirtyCalls)
}

func TestDirtyPushOnly(t *testing.T) {
	requireGit(t)
	base, _, remote := dirtyFixture(t)
	before := remoteHead(t, remote)
	h := newHarness()
	h.prompter.Dirty = []model.Decision{{DirtyChoice: model.PushOnly}}
	opts := folderOpts(projectsDir(base))
	opts.Dirty = model.DirtyAsk

	r, code := h.run(t, context.Background(), opts)
	assert.Equal(t, 0, code)
	res := byName(r, "beta")
	assert.Equal(t, model.UpToDate, res.Status)
	assert.Equal(t, "uncommitted changes left untouched", res.Detail)
	assert.Equal(t, before, remoteHead(t, remote))
	assert.Contains(t, h.out.String(), "(uncommitted changes left untouched)")
}

func TestDirtySkip(t *testing.T) {
	requireGit(t)
	base, _, _ := dirtyFixture(t)
	h := newHarness()
	h.prompter.Dirty = []model.Decision{{DirtyChoice: model.Skip}}
	opts := folderOpts(projectsDir(base))
	opts.Dirty = model.DirtyAsk

	r, code := h.run(t, context.Background(), opts)
	assert.Equal(t, 0, code)
	assert.Equal(t, model.SkippedByUser, byName(r, "beta").Status)

	opts.Strict = true
	h2 := newHarness()
	h2.prompter.Dirty = []model.Decision{{DirtyChoice: model.Skip}}
	_, code = h2.run(t, context.Background(), opts)
	assert.Equal(t, 1, code, "--strict turns user skips into failures")
}

func TestDirtyApplyToAll(t *testing.T) {
	requireGit(t)
	base := t.TempDir()
	for _, n := range []string{"one", "two", "three"} {
		repo := mkRepo(t, base, n, mkBareRemote(t, base, n))
		makeDirty(t, repo)
	}
	h := newHarness()
	h.prompter.Dirty = []model.Decision{{DirtyChoice: model.Skip, ApplyToAll: true}}
	opts := folderOpts(projectsDir(base))
	opts.Dirty = model.DirtyAsk
	r, _ := h.run(t, context.Background(), opts)
	assert.Equal(t, 1, h.prompter.DirtyCalls, "answer remembered for the rest of the run")
	for _, res := range r.Results {
		assert.Equal(t, model.SkippedByUser, res.Status)
	}
}

func TestDirtyCommitWithoutIdentity(t *testing.T) {
	requireGit(t)
	base, repo, remote := dirtyFixture(t)
	gitRun(t, repo, "config", "--unset", "user.email")
	// Make sure no global identity leaks in.
	t.Setenv("GIT_CONFIG_GLOBAL", t.TempDir()+"/none")
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	before := remoteHead(t, remote)

	h := newHarness()
	h.prompter.Dirty = []model.Decision{{DirtyChoice: model.CommitAndPush}}
	opts := folderOpts(projectsDir(base))
	opts.Dirty = model.DirtyAsk
	r, code := h.run(t, context.Background(), opts)
	assert.Equal(t, 1, code)
	res := byName(r, "beta")
	require.Equal(t, model.NotPushed, res.Status)
	assert.Equal(t, model.IdentityMissing, res.Reason)
	assert.Equal(t, before, remoteHead(t, remote), "nothing was committed or pushed")
	assert.Equal(t, "init", gitRun(t, repo, "log", "-1", "--pretty=%s"))
}

func TestDirtyFlagCommitUsesMessage(t *testing.T) {
	requireGit(t)
	base, _, remote := dirtyFixture(t)
	h := newHarness()
	opts := folderOpts(projectsDir(base))
	opts.Dirty = model.DirtyCommit
	opts.Message = "chore: scripted"
	_, code := h.run(t, context.Background(), opts)
	assert.Equal(t, 0, code)
	assert.Equal(t, "chore: scripted", remoteLastMessage(t, remote))
	assert.Equal(t, 0, h.prompter.DirtyCalls, "flag answers the prompt")
}
