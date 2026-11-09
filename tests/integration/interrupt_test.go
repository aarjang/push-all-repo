package integration

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/aarjang/push-all-repo/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// installSlowPrePush makes every push in repo block for secs seconds.
func installSlowPrePush(t *testing.T, repo string, secs int) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell hooks are not reliable on Windows runners")
	}
	hooks := filepath.Join(repo, ".git", "hooks")
	require.NoError(t, os.MkdirAll(hooks, 0o755))
	script := "#!/bin/sh\nsleep " + itoa(secs) + "\nexit 0\n"
	require.NoError(t, os.WriteFile(filepath.Join(hooks, "pre-push"), []byte(script), 0o755))
}

func itoa(n int) string { return string(rune('0' + n)) }

func TestInterruptMidRun(t *testing.T) {
	requireGit(t)
	base := t.TempDir()
	one := mkRepo(t, base, "one", mkBareRemote(t, base, "one"))
	two := mkRepo(t, base, "two", mkBareRemote(t, base, "two"))
	three := mkRepo(t, base, "three", mkBareRemote(t, base, "three"))
	for _, r := range []string{one, two, three} {
		addCommit(t, r, "work")
	}
	installSlowPrePush(t, three, 5) // sorted order: one, three, two → "three" is second

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(1500 * time.Millisecond)
		cancel()
	}()
	h := newHarness()
	start := time.Now()
	r, code := h.run(t, ctx, folderOpts(projectsDir(base)))
	assert.Less(t, time.Since(start), 4*time.Second, "the blocked push was killed")

	assert.Equal(t, 130, code)
	assert.True(t, r.Interrupted)
	require.Len(t, r.Results, 3)
	assert.Equal(t, model.Pushed, byName(r, "one").Status)
	assert.Equal(t, model.NotPushed, byName(r, "three").Status)
	assert.Equal(t, model.Interrupted, byName(r, "three").Reason)
	assert.Equal(t, model.NotProcessed, byName(r, "two").Status)
	assert.Contains(t, h.out.String(), "Run was interrupted.")
	assert.Contains(t, h.out.String(), "Not processed (1)")
}

func TestPushTimeout(t *testing.T) {
	requireGit(t)
	base := t.TempDir()
	slow := mkRepo(t, base, "slow", mkBareRemote(t, base, "slow"))
	addCommit(t, slow, "work")
	installSlowPrePush(t, slow, 5)
	fastRemote := mkBareRemote(t, base, "zfast")
	fast := mkRepo(t, base, "zfast", fastRemote)
	addCommit(t, fast, "work")

	h := newHarness()
	opts := folderOpts(projectsDir(base))
	opts.Timeout = time.Second
	r, code := h.run(t, context.Background(), opts)
	assert.Equal(t, 1, code)
	s := byName(r, "slow")
	assert.Equal(t, model.NotPushed, s.Status)
	assert.Equal(t, model.Timeout, s.Reason)
	assert.Equal(t, "1 s", s.Detail)
	assert.Contains(t, h.out.String(), "push exceeded 1 s")
	assert.Equal(t, model.Pushed, byName(r, "zfast").Status, "the next repository is still processed")
	assert.Equal(t, localHead(t, fast), remoteHead(t, fastRemote))
}
