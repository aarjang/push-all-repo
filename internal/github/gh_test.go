package github

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aarjang/push-all-repo/internal/gitx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGhClientNotInstalled(t *testing.T) {
	c := &GhClient{}
	assert.False(t, c.Available())
	assert.False(t, c.LoggedIn(context.Background()))
	_, err := c.CreateRepo(context.Background(), ".", "x", true, time.Second)
	assert.Error(t, err)
}

func TestGhClientCreate(t *testing.T) {
	f := gitx.NewFakeRunner().
		On("auth status --hostname github.com", gitx.Resp{}).
		On("repo create demo --private --source . --remote origin --push", gitx.Resp{}).
		On("repo view --json url -q .url", gitx.Resp{Stdout: "https://github.com/me/demo\n"})
	c := &GhClient{R: f, program: "gh"}
	assert.True(t, c.Available())
	assert.True(t, c.LoggedIn(context.Background()))
	url, err := c.CreateRepo(context.Background(), ".", "demo", true, time.Second)
	require.NoError(t, err)
	assert.Equal(t, "https://github.com/me/demo", url)
}

func TestGhClientNameTaken(t *testing.T) {
	f := gitx.NewFakeRunner().
		On("repo create demo --public --source . --remote origin --push", gitx.Resp{Code: 1, Stderr: "GraphQL: Name already exists on this account (createRepository)"})
	c := &GhClient{R: f, program: "gh"}
	_, err := c.CreateRepo(context.Background(), ".", "demo", false, time.Second)
	assert.True(t, errors.Is(err, ErrNameTaken))
}

func TestGhClientOtherError(t *testing.T) {
	f := gitx.NewFakeRunner().
		On("repo create demo --private --source . --remote origin --push", gitx.Resp{Code: 1, Stderr: "error connecting to api.github.com\ncheck your internet connection"})
	c := &GhClient{R: f, program: "gh"}
	_, err := c.CreateRepo(context.Background(), ".", "demo", true, time.Second)
	require.Error(t, err)
	assert.Equal(t, "check your internet connection", err.Error())
}

func TestFakeClient(t *testing.T) {
	f := &FakeClient{Installed: true, Logged: true, Errs: []error{ErrNameTaken, nil}}
	_, err := f.CreateRepo(context.Background(), "/p", "a", true, 0)
	assert.True(t, errors.Is(err, ErrNameTaken))
	url, err := f.CreateRepo(context.Background(), "/p", "b", false, 0)
	require.NoError(t, err)
	assert.Equal(t, "https://github.com/me/b", url)
	assert.Len(t, f.Calls, 2)
}
