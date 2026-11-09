package report

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aarjang/push-all-repo/internal/model"
	"github.com/aarjang/push-all-repo/internal/ui"
	"github.com/stretchr/testify/require"
)

var update = flag.Bool("update", false, "rewrite golden files")

// fixtureRun is shared by the text and Markdown golden tests.
func fixtureRun() model.Run {
	start := time.Date(2026, 11, 9, 14, 3, 17, 0, time.UTC)
	cand := func(name, branch string, ahead int, upstream string) model.Candidate {
		return model.Candidate{Path: "/Users/me/projects/" + name, Name: name, IsRepo: true, Branch: branch, Upstream: upstream, AheadCount: ahead, Remotes: []string{"origin"}}
	}
	return model.Run{
		Mode:        model.FolderOfProjects,
		RootPath:    "/Users/me/projects",
		StartedAt:   start,
		FinishedAt:  start.Add(41 * time.Second),
		ToolVersion: "0.1.0",
		Results: []model.Result{
			{Candidate: cand("api-server", "main", 2, "origin/main"), Status: model.Pushed},
			{Candidate: cand("website", "develop", 1, "origin/develop"), Status: model.Pushed, Committed: true, CommitMessage: "chore: bulk push on 2026-11-09 14:03"},
			{Candidate: cand("new-tool", "main", -1, ""), Status: model.Pushed, CreatedRemote: "https://github.com/me/new-tool", CreatedVisibility: model.Private},
			{Candidate: cand("dotfiles", "main", 0, "origin/main"), Status: model.UpToDate},
			{Candidate: cand("notes", "main", 0, "origin/main"), Status: model.UpToDate, Detail: "uncommitted changes left untouched"},
			{Candidate: cand("legacy-app", "main", 3, "origin/main"), Status: model.NotPushed, Reason: model.AuthFailed},
			{Candidate: cand("scratch", "", -1, ""), Status: model.NotPushed, Reason: model.DetachedHead},
			{Candidate: cand("experiments", "main", 1, "origin/main"), Status: model.SkippedByUser, Detail: "excluded in review"},
			{Candidate: model.Candidate{Path: "/Users/me/projects/Downloads", Name: "Downloads"}, Status: model.NotPushed, Reason: model.NotARepo},
			{Candidate: model.Candidate{Path: "/Users/me/projects/screenshots", Name: "screenshots"}, Status: model.NotPushed, Reason: model.NotARepo},
		},
	}
}

func checkGolden(t *testing.T, name string, got []byte) {
	t.Helper()
	path := filepath.Join("testdata", name)
	if *update {
		require.NoError(t, os.WriteFile(path, got, 0o644))
	}
	want, err := os.ReadFile(path)
	require.NoError(t, err, "run with -update to create the golden file")
	// Tolerate a CRLF checkout of the golden file; the renderer itself must
	// still emit LF only, which the Markdown test asserts separately.
	wantLF := strings.ReplaceAll(string(want), "\r\n", "\n")
	require.Equal(t, wantLF, string(got))
}

func TestRenderTextGolden(t *testing.T) {
	var buf bytes.Buffer
	RenderText(&buf, Build(fixtureRun()), ui.PlainStyle())
	checkGolden(t, "report.txt", buf.Bytes())
}

func TestRenderTextInterrupted(t *testing.T) {
	run := fixtureRun()
	run.Interrupted = true
	run.Results = append(run.Results, model.Result{Candidate: model.Candidate{Name: "zzz", IsRepo: true, Branch: "main"}, Status: model.NotProcessed, Reason: model.Interrupted})
	var buf bytes.Buffer
	RenderText(&buf, Build(run), ui.PlainStyle())
	require.Contains(t, buf.String(), "Run was interrupted.")
	require.Contains(t, buf.String(), "Not processed (1)")
	require.Contains(t, buf.String(), "1 not processed")
}

func TestTruncate(t *testing.T) {
	long := "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz"
	require.Len(t, []rune(truncate(long)), MaxNameWidth)
}
