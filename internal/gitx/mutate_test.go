package gitx

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aarjang/push-all-repo/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func opErr(t *testing.T, err error) *OpError {
	t.Helper()
	var oe *OpError
	require.True(t, errors.As(err, &oe), "expected *OpError, got %T", err)
	return oe
}

func TestAddAllCancelledIsInterrupted(t *testing.T) {
	f := NewFakeRunner().On("add -A", Resp{Block: true})
	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(50 * time.Millisecond); cancel() }()
	err := New(f).AddAll(ctx, "/repo")
	assert.Equal(t, model.Interrupted, opErr(t, err).Reason)
}

func TestAddAllFailureIsCommitFailed(t *testing.T) {
	f := NewFakeRunner().On("add -A", Resp{Code: 128, Stderr: "fatal: index.lock exists"})
	err := New(f).AddAll(context.Background(), "/repo")
	oe := opErr(t, err)
	assert.Equal(t, model.CommitFailed, oe.Reason)
	assert.Equal(t, "fatal: index.lock exists", oe.Detail)
}

func TestCommitCancelledIsInterrupted(t *testing.T) {
	f := NewFakeRunner().On("commit -m msg", Resp{Block: true})
	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(50 * time.Millisecond); cancel() }()
	err := New(f).Commit(ctx, "/repo", "msg")
	assert.Equal(t, model.Interrupted, opErr(t, err).Reason)
}

func TestPushChoosesCommand(t *testing.T) {
	f := NewFakeRunner().On("push", Resp{}).On("push --set-upstream origin main", Resp{})
	g := New(f)
	require.NoError(t, g.Push(context.Background(), model.Candidate{Path: "/r", Branch: "main", Upstream: "origin/main", Remotes: []string{"origin"}}, time.Second))
	require.NoError(t, g.Push(context.Background(), model.Candidate{Path: "/r", Branch: "main", Remotes: []string{"origin"}}, time.Second))
	err := g.Push(context.Background(), model.Candidate{Path: "/r", Branch: "main", Remotes: []string{"backup"}}, time.Second)
	assert.Equal(t, model.NoRemote, opErr(t, err).Reason)
	assert.Equal(t, []string{"push", "push --set-upstream origin main"}, f.Calls)
}

func TestPushTimeoutDetail(t *testing.T) {
	f := NewFakeRunner().On("push", Resp{Block: true})
	err := New(f).Push(context.Background(), model.Candidate{Path: "/r", Branch: "main", Upstream: "origin/main"}, 50*time.Millisecond)
	oe := opErr(t, err)
	assert.Equal(t, model.Timeout, oe.Reason)
	assert.Equal(t, "0 s", oe.Detail)
}
