package integration

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/aarjang/push-all-repo/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func singleOpts(path string) model.Options {
	o := folderOpts(path)
	o.Mode = model.SingleProject
	o.NoReview = false
	return o
}

func TestSingleModePush(t *testing.T) {
	requireGit(t)
	base := t.TempDir()
	remote := mkBareRemote(t, base, "gamma")
	gamma := mkRepo(t, base, "gamma", remote)
	addCommit(t, gamma, "second")

	h := newHarness()
	r, code := h.run(t, context.Background(), singleOpts(gamma))
	assert.Equal(t, 0, code)
	require.Len(t, r.Results, 1)
	assert.Equal(t, model.Pushed, r.Results[0].Status)
	assert.Equal(t, localHead(t, gamma), remoteHead(t, remote))
	assert.Equal(t, 0, h.prompter.ReviewCalls, "no review step in single mode")
	assert.Equal(t, 0, h.prompter.ConfirmCalls)
	assert.Contains(t, h.out.String(), "single mode")
	assert.NotContains(t, h.out.String(), "Found ")
}

func TestSingleModeNotARepo(t *testing.T) {
	requireGit(t)
	base := t.TempDir()
	notes := mkPlainFolder(t, base, "notes")
	h := newHarness()
	r, code := h.run(t, context.Background(), singleOpts(notes))
	assert.Equal(t, 2, code)
	assert.Empty(t, r.Results)
	assert.Contains(t, h.errOut.String(), "not a Git repository")
	assert.NotContains(t, h.out.String(), "Push report")
}

func TestSingleModeNestedPath(t *testing.T) {
	requireGit(t)
	base := t.TempDir()
	repo := mkRepo(t, base, "app", "")
	sub := filepath.Join(repo, "src")
	require.NoError(t, mkdir(sub))
	h := newHarness()
	_, code := h.run(t, context.Background(), singleOpts(sub))
	assert.Equal(t, 2, code)
	assert.Contains(t, h.errOut.String(), "is inside repository")
	assert.Contains(t, h.errOut.String(), "run on")
}

func TestSingleModePromptedPathRetries(t *testing.T) {
	requireGit(t)
	base := t.TempDir()
	remote := mkBareRemote(t, base, "p")
	repo := mkRepo(t, base, "p", remote)
	addCommit(t, repo, "x")
	h := newHarness()
	h.prompter.PathAnswers = []string{filepath.Join(base, "nope"), repo}
	h.prompter.ModeAnswer = model.SingleProject
	opts := singleOpts("")
	opts.Mode = model.ModeUnset
	r, code := h.run(t, context.Background(), opts)
	assert.Equal(t, 0, code)
	require.Len(t, r.Results, 1)
	assert.Equal(t, model.Pushed, r.Results[0].Status)
}
