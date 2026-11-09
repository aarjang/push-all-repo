package integration

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aarjang/push-all-repo/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReportSavedToFlagPath(t *testing.T) {
	requireGit(t)
	base := threeRepos(t)
	mkPlainFolder(t, base, "notes")
	h := newHarness()
	opts := folderOpts(projectsDir(base))
	opts.ReportPath = filepath.Join(t.TempDir(), "out", "report.md")
	_, code := h.run(t, context.Background(), opts)
	assert.Equal(t, 0, code)
	data, err := os.ReadFile(opts.ReportPath)
	require.NoError(t, err)
	s := string(data)
	for _, n := range []string{"alpha", "beta", "gamma", "notes"} {
		assert.Equal(t, 1, strings.Count(s, n), "%s appears once", n)
	}
	assert.Contains(t, s, "## Pushed (3)")
	assert.Contains(t, s, "## Not a Git repository (1)")
	assert.NotContains(t, s, "\r\n")
	assert.Contains(t, h.out.String(), "Report saved to "+opts.ReportPath)
	assert.Equal(t, 0, h.prompter.SaveCalls)
}

func TestReportDashDisables(t *testing.T) {
	requireGit(t)
	base := threeRepos(t)
	h := newHarness()
	_, _ = h.run(t, context.Background(), folderOpts(projectsDir(base)))
	assert.Equal(t, 0, h.prompter.SaveCalls)
	assert.NotContains(t, h.out.String(), "Report saved")
}

func TestReportPromptDefaultPath(t *testing.T) {
	requireGit(t)
	base := threeRepos(t)
	h := newHarness()
	h.prompter.Save = true
	opts := folderOpts(projectsDir(base))
	opts.ReportPath = ""
	cwd := t.TempDir()
	// run() uses its own Cwd; mimic by capturing the printed path
	_ = cwd
	_, code := h.run(t, context.Background(), opts)
	assert.Equal(t, 0, code)
	assert.Equal(t, 1, h.prompter.SaveCalls)
	out := h.out.String()
	idx := strings.Index(out, "Report saved to ")
	require.Greater(t, idx, -1)
	path := strings.TrimSpace(strings.SplitN(out[idx+len("Report saved to "):], "\n", 2)[0])
	assert.Regexp(t, `push-report-\d{8}-\d{6}\.md$`, path)
	_, err := os.Stat(path)
	assert.NoError(t, err)
}

func TestReportPromptDeclined(t *testing.T) {
	requireGit(t)
	base := threeRepos(t)
	h := newHarness()
	h.prompter.Save = false
	opts := folderOpts(projectsDir(base))
	opts.ReportPath = ""
	_, _ = h.run(t, context.Background(), opts)
	assert.Equal(t, 1, h.prompter.SaveCalls)
	assert.NotContains(t, h.out.String(), "Report saved")
}

func TestReportMarkdownMatchesScreen(t *testing.T) {
	requireGit(t)
	base := threeRepos(t)
	h := newHarness()
	opts := folderOpts(projectsDir(base))
	opts.ReportPath = filepath.Join(t.TempDir(), "r.md")
	r, _ := h.run(t, context.Background(), opts)
	data, err := os.ReadFile(opts.ReportPath)
	require.NoError(t, err)
	for _, res := range r.Results {
		assert.Contains(t, h.out.String(), res.Candidate.Name)
		assert.Contains(t, string(data), "| "+res.Candidate.Name+" |")
	}
	assert.Equal(t, model.FolderOfProjects, r.Mode)
}
