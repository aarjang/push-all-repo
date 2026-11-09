package run

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aarjang/push-all-repo/internal/model"
	"github.com/aarjang/push-all-repo/internal/ui"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeGit is a gitx.Client whose Inspect can cancel the run part-way.
type fakeGit struct {
	repos    map[string]bool
	cancelOn string
	cancel   context.CancelFunc
	inspects []string
}

func (f *fakeGit) Inspect(_ context.Context, path string) (model.Candidate, error) {
	f.inspects = append(f.inspects, filepath.Base(path))
	c := model.Candidate{Path: path, Name: filepath.Base(path), AheadCount: -1}
	if f.repos[filepath.Base(path)] {
		c.IsRepo, c.Branch, c.Remotes, c.Upstream, c.AheadCount = true, "main", []string{"origin"}, "origin/main", 1
	}
	if filepath.Base(path) == f.cancelOn && f.cancel != nil {
		f.cancel()
	}
	return c, nil
}

func (f *fakeGit) AddAll(context.Context, string) error                       { return nil }
func (f *fakeGit) Commit(context.Context, string, string) error               { return nil }
func (f *fakeGit) Push(context.Context, model.Candidate, time.Duration) error { return nil }

func TestInterruptDuringDiscoveryPrintsPartialReport(t *testing.T) {
	root := t.TempDir()
	for _, d := range []string{"a", "b", "c"} {
		require.NoError(t, os.Mkdir(filepath.Join(root, d), 0o755))
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	g := &fakeGit{repos: map[string]bool{"a": true, "b": true, "c": true}, cancelOn: "b", cancel: cancel}

	opts := model.DefaultOptions()
	opts.Mode, opts.Path, opts.NoReview, opts.ReportPath = model.FolderOfProjects, root, true, "-"
	var out, errOut bytes.Buffer
	run, code := Run(ctx, opts, Deps{Git: g, Prompter: ui.NewScripted(), Out: &out, Err: &errOut, Style: ui.PlainStyle()})

	assert.Equal(t, ExitInterrupted, code)
	assert.True(t, run.Interrupted)
	require.Len(t, run.Results, 2, "a and b were discovered before the cancel took effect")
	for _, r := range run.Results {
		assert.Equal(t, model.NotProcessed, r.Status)
	}
	assert.Contains(t, out.String(), "Run was interrupted.")
	assert.Contains(t, out.String(), "Not processed (2)")
	assert.Contains(t, errOut.String(), "interrupted during discovery")
}

func TestNoOriginNoUpstreamGoesToNoRemoteFlow(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(root, "odd"), 0o755))
	g := &fakeGit{repos: map[string]bool{}}
	// Hand-craft the candidate through a wrapper Inspect.
	wrapped := &oddGit{fakeGit: g}

	opts := model.DefaultOptions()
	opts.Mode, opts.Path, opts.NoReview, opts.ReportPath = model.FolderOfProjects, root, true, "-"
	opts.CreateRepo = model.CreateRepoNo
	var out bytes.Buffer
	run, code := Run(context.Background(), opts, Deps{Git: wrapped, Prompter: ui.NewScripted(), Out: &out, Style: ui.PlainStyle()})
	assert.Equal(t, ExitNotPushed, code)
	require.Len(t, run.Results, 1)
	assert.Equal(t, model.NoRemoteDeclined, run.Results[0].Reason)
	assert.Contains(t, out.String(), "no upstream and no remote named origin")
}

type oddGit struct{ *fakeGit }

func (o *oddGit) Inspect(ctx context.Context, path string) (model.Candidate, error) {
	c, _ := o.fakeGit.Inspect(ctx, path)
	if filepath.Base(path) == "odd" {
		c.IsRepo, c.Branch, c.Remotes, c.Upstream, c.AheadCount = true, "main", []string{"backup"}, "", -1
	}
	return c, nil
}
