package gitx

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// scriptRepo returns a fake runner scripted for a healthy repo at dir.
func scriptRepo(dir string) *FakeRunner {
	return NewFakeRunner().
		On("rev-parse --show-toplevel", Resp{Stdout: filepath.ToSlash(dir) + "\n"}).
		On("symbolic-ref --quiet --short HEAD", Resp{Stdout: "main\n"}).
		On("remote", Resp{Stdout: "origin\n"}).
		On("rev-parse --abbrev-ref --symbolic-full-name @{u}", Resp{Stdout: "origin/main\n"}).
		On("rev-list --count @{u}..HEAD", Resp{Stdout: "2\n"}).
		On("status --porcelain=v1 --untracked-files=normal", Resp{Stdout: ""}).
		On("rev-parse --git-path MERGE_HEAD", Resp{Stdout: ".git/MERGE_HEAD\n"}).
		On("rev-parse --git-path rebase-merge", Resp{Stdout: ".git/rebase-merge\n"}).
		On("rev-parse --git-path rebase-apply", Resp{Stdout: ".git/rebase-apply\n"}).
		On("rev-parse --git-path CHERRY_PICK_HEAD", Resp{Stdout: ".git/CHERRY_PICK_HEAD\n"}).
		On("config user.name", Resp{Stdout: "Me\n"}).
		On("config user.email", Resp{Stdout: "me@example.com\n"})
}

func TestInspectHealthy(t *testing.T) {
	dir := t.TempDir()
	g := New(scriptRepo(dir))
	c, err := g.Inspect(context.Background(), dir)
	require.NoError(t, err)
	assert.True(t, c.IsRepo)
	assert.False(t, c.Nested)
	assert.Equal(t, "main", c.Branch)
	assert.Equal(t, "origin/main", c.Upstream)
	assert.Equal(t, []string{"origin"}, c.Remotes)
	assert.Equal(t, 2, c.AheadCount)
	assert.False(t, c.Dirty)
	assert.Equal(t, "", c.InProgressOp)
	assert.True(t, c.IdentityOK)
	assert.True(t, c.NeedsPush())
}

func TestInspectDetached(t *testing.T) {
	dir := t.TempDir()
	f := scriptRepo(dir).On("symbolic-ref --quiet --short HEAD", Resp{Code: 1})
	c, _ := New(f).Inspect(context.Background(), dir)
	assert.True(t, c.IsRepo)
	assert.Equal(t, "", c.Branch)
	assert.False(t, c.NeedsPush())
}

func TestInspectNoUpstream(t *testing.T) {
	dir := t.TempDir()
	f := scriptRepo(dir).On("rev-parse --abbrev-ref --symbolic-full-name @{u}", Resp{Code: 128, Stderr: "fatal: no upstream configured"})
	c, _ := New(f).Inspect(context.Background(), dir)
	assert.Equal(t, "", c.Upstream)
	assert.Equal(t, -1, c.AheadCount)
	assert.True(t, c.NeedsPush())
}

func TestInspectNoRemotes(t *testing.T) {
	dir := t.TempDir()
	f := scriptRepo(dir).On("remote", Resp{Stdout: ""}).
		On("rev-parse --abbrev-ref --symbolic-full-name @{u}", Resp{Code: 128})
	c, _ := New(f).Inspect(context.Background(), dir)
	assert.Empty(t, c.Remotes)
	assert.False(t, c.HasRemote())
	assert.False(t, c.HasOrigin())
}

func TestInspectDirty(t *testing.T) {
	dir := t.TempDir()
	f := scriptRepo(dir).On("status --porcelain=v1 --untracked-files=normal", Resp{Stdout: " M a.go\n?? b.txt\nA  c.md\n"})
	c, _ := New(f).Inspect(context.Background(), dir)
	assert.True(t, c.Dirty)
	assert.Len(t, c.DirtyFiles, 3)
}

func TestInspectMergeInProgress(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".git", "MERGE_HEAD"), []byte("abc\n"), 0o644))
	c, _ := New(scriptRepo(dir)).Inspect(context.Background(), dir)
	assert.Equal(t, "merge", c.InProgressOp)
}

func TestInspectNested(t *testing.T) {
	parent := t.TempDir()
	child := filepath.Join(parent, "sub")
	require.NoError(t, os.Mkdir(child, 0o755))
	f := NewFakeRunner().On("rev-parse --show-toplevel", Resp{Stdout: filepath.ToSlash(parent) + "\n"})
	c, _ := New(f).Inspect(context.Background(), child)
	assert.False(t, c.IsRepo)
	assert.True(t, c.Nested)
	assert.True(t, SamePath(parent, c.TopLevel))
}

func TestInspectNotARepo(t *testing.T) {
	dir := t.TempDir()
	f := NewFakeRunner().On("rev-parse --show-toplevel", Resp{Code: 128, Stderr: "fatal: not a git repository"})
	c, _ := New(f).Inspect(context.Background(), dir)
	assert.False(t, c.IsRepo)
	assert.False(t, c.Nested)
	assert.Equal(t, filepath.Base(dir), c.Name)
}

func TestInspectIdentityMissing(t *testing.T) {
	dir := t.TempDir()
	f := scriptRepo(dir).On("config user.email", Resp{Code: 1})
	c, _ := New(f).Inspect(context.Background(), dir)
	assert.False(t, c.IdentityOK)
}
