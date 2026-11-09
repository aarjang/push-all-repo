package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// requireGit skips the test when git is unavailable.
func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
}

// gitRun runs git in dir and fails the test on error.
func gitRun(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "LC_ALL=C")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %s in %s:\n%s", strings.Join(args, " "), dir, out)
	return strings.TrimSpace(string(out))
}

// gitTry runs git and returns combined output and error without failing.
func gitTry(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "LC_ALL=C")
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// mkBareRemote creates base/remotes/<name>.git and returns its path.
func mkBareRemote(t *testing.T, base, name string) string {
	t.Helper()
	dir := filepath.Join(base, "remotes", name+".git")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	gitRun(t, dir, "init", "-q", "--bare", "-b", "main")
	return dir
}

// mkRepo creates base/projects/<name> with an initial commit. When remote is
// non-empty it is added as origin and the branch is pushed with -u.
func mkRepo(t *testing.T, base, name, remote string) string {
	t.Helper()
	dir := filepath.Join(base, "projects", name)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	gitRun(t, dir, "init", "-q", "-b", "main")
	gitRun(t, dir, "config", "user.name", "Test User")
	gitRun(t, dir, "config", "user.email", "test@example.com")
	gitRun(t, dir, "config", "commit.gpgsign", "false")
	gitRun(t, dir, "commit", "-q", "--allow-empty", "-m", "init")
	if remote != "" {
		gitRun(t, dir, "remote", "add", "origin", remote)
		gitRun(t, dir, "push", "-q", "-u", "origin", "main")
	}
	return dir
}

// mkPlainFolder creates a non-repository folder with one file.
func mkPlainFolder(t *testing.T, base, name string) string {
	t.Helper()
	dir := filepath.Join(base, "projects", name)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("hi\n"), 0o644))
	return dir
}

// projectsDir returns the folder that holds all repos made by mkRepo.
func projectsDir(base string) string { return filepath.Join(base, "projects") }

func addCommit(t *testing.T, repo, msg string) {
	t.Helper()
	gitRun(t, repo, "commit", "-q", "--allow-empty", "-m", msg)
}

func makeDirty(t *testing.T, repo string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(repo, "dirty.txt"), []byte("x\n"), 0o644))
}

func detach(t *testing.T, repo string) {
	t.Helper()
	gitRun(t, repo, "checkout", "-q", "--detach")
}

// startMerge leaves repo with an unresolved merge conflict.
func startMerge(t *testing.T, repo string) {
	t.Helper()
	f := filepath.Join(repo, "conflict.txt")
	require.NoError(t, os.WriteFile(f, []byte("base\n"), 0o644))
	gitRun(t, repo, "add", "conflict.txt")
	gitRun(t, repo, "commit", "-q", "-m", "base")
	gitRun(t, repo, "checkout", "-q", "-b", "other")
	require.NoError(t, os.WriteFile(f, []byte("other\n"), 0o644))
	gitRun(t, repo, "commit", "-q", "-am", "other")
	gitRun(t, repo, "checkout", "-q", "main")
	require.NoError(t, os.WriteFile(f, []byte("main\n"), 0o644))
	gitRun(t, repo, "commit", "-q", "-am", "main")
	_, err := gitTry(repo, "merge", "other")
	require.Error(t, err, "merge should conflict")
}

func remoteHead(t *testing.T, bare string) string {
	t.Helper()
	return gitRun(t, bare, "rev-parse", "main")
}

func localHead(t *testing.T, repo string) string {
	t.Helper()
	return gitRun(t, repo, "rev-parse", "HEAD")
}

func remoteLastMessage(t *testing.T, bare string) string {
	t.Helper()
	return gitRun(t, bare, "log", "-1", "--pretty=%s", "main")
}
