package main

import (
	"bytes"
	"testing"
	"time"

	"github.com/aarjang/push-all-repo/internal/model"
	"github.com/stretchr/testify/assert"
)

func parse(args ...string) (model.Options, int, bool, string, string) {
	var out, errb bytes.Buffer
	o, c, d := parseFlags(args, &out, &errb)
	return o, c, d, out.String(), errb.String()
}

func TestDefaults(t *testing.T) {
	o, code, done, _, _ := parse()
	assert.False(t, done)
	assert.Equal(t, 0, code)
	assert.Equal(t, model.ModeUnset, o.Mode)
	assert.Equal(t, model.DirtyAsk, o.Dirty)
	assert.Equal(t, model.CreateRepoAsk, o.CreateRepo)
	assert.Equal(t, model.Private, o.Visibility)
	assert.Equal(t, 120*time.Second, o.Timeout)
	assert.Equal(t, "", o.ReportPath)
}

func TestAllFlags(t *testing.T) {
	o, _, done, _, _ := parse("--mode", "folder", "--dirty", "commit", "--message", "m", "--create-repo", "yes",
		"--visibility", "public", "--no-review", "--report", "r.md", "--timeout", "7", "--strict", "--no-color", "--verbose", "/some/path")
	assert.False(t, done)
	assert.Equal(t, model.FolderOfProjects, o.Mode)
	assert.Equal(t, model.DirtyCommit, o.Dirty)
	assert.Equal(t, "m", o.Message)
	assert.Equal(t, model.CreateRepoYes, o.CreateRepo)
	assert.Equal(t, model.Public, o.Visibility)
	assert.True(t, o.NoReview)
	assert.Equal(t, "r.md", o.ReportPath)
	assert.Equal(t, 7*time.Second, o.Timeout)
	assert.True(t, o.Strict && o.NoColor && o.Verbose)
	assert.Equal(t, "/some/path", o.Path)
}

func TestFlagsAfterPath(t *testing.T) {
	o, code, done, _, _ := parse("--mode", "folder", "/p", "--dirty", "skip", "--report", "-")
	assert.False(t, done)
	assert.Equal(t, 0, code)
	assert.Equal(t, "/p", o.Path)
	assert.Equal(t, model.DirtySkip, o.Dirty)
	assert.Equal(t, "-", o.ReportPath)

	o, _, done, _, _ = parse("/p", "--mode", "single")
	assert.False(t, done)
	assert.Equal(t, model.SingleProject, o.Mode)
	assert.Equal(t, "/p", o.Path)
}

func TestInvalidValuesExit2(t *testing.T) {
	for _, args := range [][]string{
		{"--mode", "bogus"},
		{"--dirty", "maybe"},
		{"--create-repo", "sometimes"},
		{"--visibility", "hidden"},
		{"--timeout", "0"},
		{"--unknown-flag"},
		{"a", "b"},
	} {
		_, code, done, _, errOut := parse(args...)
		assert.True(t, done, "%v", args)
		assert.Equal(t, 2, code, "%v", args)
		assert.Contains(t, errOut, "pushall:", "%v", args)
	}
}

func TestVersionAndHelp(t *testing.T) {
	_, code, done, out, _ := parse("--version")
	assert.True(t, done)
	assert.Equal(t, 0, code)
	assert.Contains(t, out, "pushall dev")

	_, code, done, out, _ = parse("--help")
	assert.True(t, done)
	assert.Equal(t, 0, code)
	assert.Contains(t, out, "--mode folder|single")
}
