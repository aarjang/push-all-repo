package integration

import (
	"context"
	"testing"

	"github.com/aarjang/push-all-repo/internal/github"
	"github.com/aarjang/push-all-repo/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func noRemoteFixture(t *testing.T) (base string) {
	t.Helper()
	base = t.TempDir()
	mkRepo(t, base, "delta", "")
	return
}

func TestNoRemoteFlagNo(t *testing.T) {
	requireGit(t)
	base := noRemoteFixture(t)
	h := newHarness()
	r, code := h.run(t, context.Background(), folderOpts(projectsDir(base)))
	assert.Equal(t, 1, code)
	res := byName(r, "delta")
	assert.Equal(t, model.NotPushed, res.Status)
	assert.Equal(t, model.NoRemoteDeclined, res.Reason)
	assert.Contains(t, h.out.String(), "no remote (user declined to create one)")
}

func TestNoRemoteLoginRequired(t *testing.T) {
	requireGit(t)
	base := noRemoteFixture(t)
	h := newHarness()
	h.gh = &github.FakeClient{Installed: true, Logged: false}
	opts := folderOpts(projectsDir(base))
	opts.CreateRepo = model.CreateRepoAsk
	r, code := h.run(t, context.Background(), opts)
	assert.Equal(t, 1, code)
	assert.Equal(t, model.NoRemoteLoginRequired, byName(r, "delta").Reason)
	assert.Contains(t, h.errOut.String(), "not logged in to GitHub")
	assert.Equal(t, 0, h.prompter.CreateCalls, "no offer when creation is impossible")
}

func TestNoRemoteGhMissing(t *testing.T) {
	requireGit(t)
	base := noRemoteFixture(t)
	h := newHarness()
	h.gh = &github.FakeClient{Installed: false}
	opts := folderOpts(projectsDir(base))
	opts.CreateRepo = model.CreateRepoAsk
	r, _ := h.run(t, context.Background(), opts)
	assert.Equal(t, model.NoRemoteLoginRequired, byName(r, "delta").Reason)
	assert.Contains(t, h.errOut.String(), "gh) is not installed")
}

func TestNoRemoteDeclined(t *testing.T) {
	requireGit(t)
	base := noRemoteFixture(t)
	h := newHarness()
	h.gh = &github.FakeClient{Installed: true, Logged: true}
	h.prompter.Create = []model.Decision{{CreateRepo: false}}
	opts := folderOpts(projectsDir(base))
	opts.CreateRepo = model.CreateRepoAsk
	r, _ := h.run(t, context.Background(), opts)
	assert.Equal(t, model.NoRemoteDeclined, byName(r, "delta").Reason)
	assert.Empty(t, h.gh.Calls)
}

func TestNoRemoteAccepted(t *testing.T) {
	requireGit(t)
	base := noRemoteFixture(t)
	h := newHarness()
	h.gh = &github.FakeClient{Installed: true, Logged: true}
	h.prompter.Create = []model.Decision{{CreateRepo: true, Visibility: model.Private}}
	opts := folderOpts(projectsDir(base))
	opts.CreateRepo = model.CreateRepoAsk
	r, code := h.run(t, context.Background(), opts)
	assert.Equal(t, 0, code)
	res := byName(r, "delta")
	assert.Equal(t, model.Pushed, res.Status)
	assert.Equal(t, "https://github.com/me/delta", res.CreatedRemote)
	require.Len(t, h.gh.Calls, 1)
	assert.Equal(t, "delta", h.gh.Calls[0].Name)
	assert.True(t, h.gh.Calls[0].Private)
	assert.Contains(t, h.out.String(), "pushed to new GitHub repository https://github.com/me/delta (private)")
}

func TestNoRemoteFlagYes(t *testing.T) {
	requireGit(t)
	base := noRemoteFixture(t)
	h := newHarness()
	h.gh = &github.FakeClient{Installed: true, Logged: true}
	opts := folderOpts(projectsDir(base))
	opts.CreateRepo = model.CreateRepoYes
	opts.Visibility = model.Public
	_, code := h.run(t, context.Background(), opts)
	assert.Equal(t, 0, code)
	require.Len(t, h.gh.Calls, 1)
	assert.False(t, h.gh.Calls[0].Private)
	assert.Equal(t, 0, h.prompter.CreateCalls)
}

func TestNoRemoteNameTaken(t *testing.T) {
	requireGit(t)
	base := noRemoteFixture(t)
	h := newHarness()
	h.gh = &github.FakeClient{Installed: true, Logged: true, Errs: []error{github.ErrNameTaken}}
	h.prompter.Create = []model.Decision{
		{CreateRepo: true},  // first offer: accept default name
		{CreateRepo: false}, // asked again after the clash: decline
	}
	opts := folderOpts(projectsDir(base))
	opts.CreateRepo = model.CreateRepoAsk
	r, code := h.run(t, context.Background(), opts)
	assert.Equal(t, 1, code)
	res := byName(r, "delta")
	assert.Equal(t, model.CreateRepoFailed, res.Reason)
	assert.Contains(t, res.Detail, "already exists")
	require.Equal(t, 2, h.prompter.CreateCalls)
	assert.True(t, h.prompter.NameTaken[1])
}

func TestNoRemoteNameTakenRetrySucceeds(t *testing.T) {
	requireGit(t)
	base := noRemoteFixture(t)
	h := newHarness()
	h.gh = &github.FakeClient{Installed: true, Logged: true, Errs: []error{github.ErrNameTaken, nil}}
	h.prompter.Create = []model.Decision{
		{CreateRepo: true},
		{CreateRepo: true, RepoName: "delta-2"},
	}
	opts := folderOpts(projectsDir(base))
	opts.CreateRepo = model.CreateRepoAsk
	r, code := h.run(t, context.Background(), opts)
	assert.Equal(t, 0, code)
	assert.Equal(t, model.Pushed, byName(r, "delta").Status)
	require.Len(t, h.gh.Calls, 2)
	assert.Equal(t, "delta-2", h.gh.Calls[1].Name)
}
