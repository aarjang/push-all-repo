package discover

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/aarjang/push-all-repo/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeInspector struct {
	repos  map[string]bool   // path → is top-level repo
	nested map[string]string // path → top-level it belongs to
}

func (f fakeInspector) Inspect(_ context.Context, path string) (model.Candidate, error) {
	c := model.Candidate{Path: path, Name: filepath.Base(path), AheadCount: -1}
	if f.repos[filepath.Clean(path)] {
		c.IsRepo = true
		c.Branch = "main"
	} else if top, ok := f.nested[filepath.Clean(path)]; ok {
		c.Nested = true
		c.TopLevel = top
	}
	return c, nil
}

func names(cs []model.Candidate) []string {
	var out []string
	for _, c := range cs {
		out = append(out, c.Name)
	}
	return out
}

func TestDiscoverOrderAndFilters(t *testing.T) {
	root := t.TempDir()
	for _, d := range []string{"Zeta", "alpha", "beta", ".hidden", "notes"} {
		require.NoError(t, os.Mkdir(filepath.Join(root, d), 0o755))
	}
	require.NoError(t, os.WriteFile(filepath.Join(root, "file.txt"), []byte("x"), 0o644))

	f := fakeInspector{repos: map[string]bool{
		filepath.Join(root, "Zeta"):  true,
		filepath.Join(root, "alpha"): true,
		filepath.Join(root, "beta"):  true,
	}}
	cs, err := Discover(context.Background(), root, f)
	require.NoError(t, err)
	assert.Equal(t, []string{"alpha", "beta", "notes", "Zeta"}, names(cs))
	assert.True(t, cs[0].IsRepo)
	assert.False(t, cs[2].IsRepo, "notes is a plain folder")
}

func TestDiscoverRootIsRepo(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(root, "src"), 0o755))   // sub-folder of root repo
	require.NoError(t, os.Mkdir(filepath.Join(root, "child"), 0o755)) // its own repo
	f := fakeInspector{
		repos:  map[string]bool{filepath.Clean(root): true, filepath.Join(root, "child"): true},
		nested: map[string]string{filepath.Join(root, "src"): root},
	}
	cs, err := Discover(context.Background(), root, f)
	require.NoError(t, err)
	assert.Equal(t, []string{filepath.Base(root), "child"}, names(cs))
}

func TestDiscoverNestedInOtherRepoKept(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(root, "odd"), 0o755))
	f := fakeInspector{nested: map[string]string{filepath.Join(root, "odd"): "/somewhere/else"}}
	cs, err := Discover(context.Background(), root, f)
	require.NoError(t, err)
	require.Len(t, cs, 1)
	assert.True(t, cs[0].Nested)
}

func TestDiscoverSymlinkFollowed(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need privileges on Windows")
	}
	root := t.TempDir()
	target := t.TempDir()
	require.NoError(t, os.Symlink(target, filepath.Join(root, "linked")))
	f := fakeInspector{repos: map[string]bool{filepath.Join(root, "linked"): true}}
	cs, err := Discover(context.Background(), root, f)
	require.NoError(t, err)
	assert.Equal(t, []string{"linked"}, names(cs))
}

func TestDiscoverEmpty(t *testing.T) {
	cs, err := Discover(context.Background(), t.TempDir(), fakeInspector{})
	require.NoError(t, err)
	assert.Empty(t, cs)
}

func TestDiscoverUnreadable(t *testing.T) {
	_, err := Discover(context.Background(), filepath.Join(t.TempDir(), "missing"), fakeInspector{})
	assert.Error(t, err)
}
