package integration

import (
	"bytes"
	"context"
	"testing"

	"github.com/aarjang/push-all-repo/internal/github"
	"github.com/aarjang/push-all-repo/internal/gitx"
	"github.com/aarjang/push-all-repo/internal/model"
	"github.com/aarjang/push-all-repo/internal/run"
	"github.com/aarjang/push-all-repo/internal/ui"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// harness bundles the real git client with scripted collaborators.
type harness struct {
	prompter *ui.ScriptedPrompter
	gh       *github.FakeClient
	out      bytes.Buffer
	errOut   bytes.Buffer
}

func newHarness() *harness {
	return &harness{prompter: ui.NewScripted(), gh: &github.FakeClient{}}
}

func (h *harness) run(t *testing.T, ctx context.Context, opts model.Options) (model.Run, int) {
	t.Helper()
	deps := run.Deps{
		Git:      gitx.New(gitx.NewExecRunner(nil)),
		GitHub:   h.gh,
		Prompter: h.prompter,
		Out:      &h.out,
		Err:      &h.errOut,
		Style:    ui.PlainStyle(),
		Version:  "test",
		Cwd:      t.TempDir(),
	}
	return run.Run(ctx, opts, deps)
}

func folderOpts(path string) model.Options {
	o := model.DefaultOptions()
	o.Mode = model.FolderOfProjects
	o.Path = path
	o.Dirty = model.DirtyPushOnly
	o.CreateRepo = model.CreateRepoNo
	o.NoReview = true
	o.ReportPath = "-"
	return o
}

func byName(r model.Run, name string) model.Result {
	for _, res := range r.Results {
		if res.Candidate.Name == name {
			return res
		}
	}
	return model.Result{}
}

func TestFolderModeHappyPath(t *testing.T) {
	requireGit(t)
	base := t.TempDir()
	alphaRemote := mkBareRemote(t, base, "alpha")
	gammaRemote := mkBareRemote(t, base, "gamma")
	alpha := mkRepo(t, base, "alpha", alphaRemote)
	mkRepo(t, base, "gamma", gammaRemote)
	mkPlainFolder(t, base, "notes")
	addCommit(t, alpha, "unpushed")
	require.NotEqual(t, localHead(t, alpha), remoteHead(t, alphaRemote))

	h := newHarness()
	r, code := h.run(t, context.Background(), folderOpts(projectsDir(base)))

	assert.Equal(t, 0, code)
	require.Len(t, r.Results, 3)
	assert.Equal(t, []string{"alpha", "gamma", "notes"}, []string{r.Results[0].Candidate.Name, r.Results[1].Candidate.Name, r.Results[2].Candidate.Name})
	assert.Equal(t, model.Pushed, byName(r, "alpha").Status)
	assert.Equal(t, localHead(t, alpha), remoteHead(t, alphaRemote))
	assert.Equal(t, model.UpToDate, byName(r, "gamma").Status)
	assert.Equal(t, model.NotPushed, byName(r, "notes").Status)
	assert.Equal(t, model.NotARepo, byName(r, "notes").Reason)
	assert.Contains(t, h.out.String(), "Found 2 repositories under")
	assert.Contains(t, h.out.String(), "Not a Git repository: notes")
	assert.Contains(t, h.out.String(), "Summary: 1 pushed, 1 up to date, 0 not pushed, 0 skipped by user")
}

func TestFolderModeNoUpstreamPublishesBranch(t *testing.T) {
	requireGit(t)
	base := t.TempDir()
	remote := mkBareRemote(t, base, "pub")
	repo := mkRepo(t, base, "pub", "")
	gitRun(t, repo, "remote", "add", "origin", remote)

	h := newHarness()
	r, code := h.run(t, context.Background(), folderOpts(projectsDir(base)))
	assert.Equal(t, 0, code)
	res := byName(r, "pub")
	assert.Equal(t, model.Pushed, res.Status)
	assert.Equal(t, localHead(t, repo), remoteHead(t, remote))
	assert.Equal(t, "origin/main", gitRun(t, repo, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}"))
}

func TestFolderModeEmpty(t *testing.T) {
	requireGit(t)
	base := t.TempDir()
	h := newHarness()
	r, code := h.run(t, context.Background(), folderOpts(base))
	assert.Equal(t, 0, code)
	assert.Empty(t, r.Results)
	assert.Contains(t, h.out.String(), "Found 0 repositories")
}

func TestInvalidPathExit2(t *testing.T) {
	requireGit(t)
	h := newHarness()
	_, code := h.run(t, context.Background(), folderOpts(t.TempDir()+"/does-not-exist"))
	assert.Equal(t, 2, code)
	assert.Contains(t, h.errOut.String(), "does not exist")
}
