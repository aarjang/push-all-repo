package gitx

import (
	"context"
	"testing"

	"github.com/aarjang/push-all-repo/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestClassifyPatterns(t *testing.T) {
	cases := []struct {
		stderr string
		want   model.Reason
	}{
		{"git@github.com: Permission denied (publickey).", model.AuthFailed},
		{"fatal: Authentication failed for 'https://x'", model.AuthFailed},
		{"fatal: could not read Username for 'https://github.com': terminal prompts disabled", model.AuthFailed},
		{"remote: 403 Forbidden", model.AuthFailed},
		{"! [rejected] main -> main (fetch first)", model.Rejected},
		{"error: failed to push some refs to 'x'", model.Rejected},
		{"hint: Updates were rejected because the tip is non-fast-forward", model.Rejected},
		{"ssh: Could not resolve hostname github.com", model.NetworkError},
		{"fatal: unable to access 'https://x/': Connection timed out", model.NetworkError},
		{"ssh: connect to host github.com port 22: Connection refused", model.NetworkError},
		{"fatal: '/tmp/nope.git' does not appear to be a git repository", model.NetworkError},
		{"error: something odd happened\n", model.GitError},
	}
	for _, tc := range cases {
		got, _ := Classify(&ExitError{Code: 1, Stderr: tc.stderr}, tc.stderr)
		assert.Equal(t, tc.want, got, "stderr: %s", tc.stderr)
	}
}

func TestClassifyFallbackDetail(t *testing.T) {
	r, d := Classify(&ExitError{Code: 1}, "line one\nlast line here\n\n")
	assert.Equal(t, model.GitError, r)
	assert.Equal(t, "last line here", d)
}

func TestClassifyTimeoutAndCancel(t *testing.T) {
	r, _ := Classify(&ExitError{TimedOut: true}, "")
	assert.Equal(t, model.Timeout, r)
	r, _ = Classify(&ExitError{Cancelled: true}, "rejected") // cancel wins over text
	assert.Equal(t, model.Interrupted, r)
	r, _ = Classify(context.DeadlineExceeded, "")
	assert.Equal(t, model.Timeout, r)
	r, _ = Classify(context.Canceled, "")
	assert.Equal(t, model.Interrupted, r)
}

func TestClassifyFirstHitWins(t *testing.T) {
	// auth pattern appears before rejected pattern in the table
	r, _ := Classify(&ExitError{Code: 1}, "rejected: permission denied")
	assert.Equal(t, model.AuthFailed, r)
}
