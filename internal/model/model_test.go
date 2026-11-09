package model

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNeedsPush(t *testing.T) {
	cases := []struct {
		name string
		c    Candidate
		want bool
	}{
		{"no upstream", Candidate{IsRepo: true, Branch: "main", AheadCount: -1}, true},
		{"ahead 0", Candidate{IsRepo: true, Branch: "main", Upstream: "origin/main", AheadCount: 0}, false},
		{"ahead 2", Candidate{IsRepo: true, Branch: "main", Upstream: "origin/main", AheadCount: 2}, true},
		{"detached", Candidate{IsRepo: true, Branch: "", AheadCount: -1}, false},
		{"not repo", Candidate{IsRepo: false, Branch: "main", AheadCount: -1}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.c.NeedsPush())
		})
	}
}

func TestReasonText(t *testing.T) {
	for _, r := range AllReasons {
		assert.NotEmpty(t, r.Text(), "reason %d has no text", int(r))
	}
	assert.Equal(t, "push exceeded 1 s", Timeout.TextWith("1 s"))
	assert.Equal(t, "merge in progress", OperationInProgress.TextWith("merge"))
	assert.Equal(t, "git error: boom", GitError.TextWith("boom"))
	assert.Equal(t, "authentication failed", AuthFailed.TextWith(""))
	assert.Equal(t, "", ReasonNone.Text())
}

func TestSetDetailTrims(t *testing.T) {
	var r Result
	r.SetDetail(strings.Repeat("x", 500))
	assert.Len(t, []rune(r.Detail), MaxDetail)
}

func TestDefaults(t *testing.T) {
	o := DefaultOptions()
	assert.Equal(t, DirtyAsk, o.Dirty)
	assert.Equal(t, CreateRepoAsk, o.CreateRepo)
	assert.Equal(t, Private, o.Visibility)
	assert.Equal(t, DefaultTimeout, o.Timeout)
	assert.Equal(t, CommitAndPush, DirtyCommit.Choice())
	assert.Equal(t, DirtyUnset, DirtyAsk.Choice())
}
