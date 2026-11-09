package gitx

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Resp is a scripted response for FakeRunner.
type Resp struct {
	Stdout string
	Stderr string
	Code   int
	// Block, when set, waits for ctx to end and then fails as cancelled/timed out.
	Block bool
}

// FakeRunner answers commands from a script keyed by the joined arguments.
type FakeRunner struct {
	Responses map[string]Resp
	Calls     []string
}

// NewFakeRunner returns an empty fake.
func NewFakeRunner() *FakeRunner { return &FakeRunner{Responses: map[string]Resp{}} }

// On scripts a response for the given argument list.
func (f *FakeRunner) On(args string, r Resp) *FakeRunner {
	f.Responses[args] = r
	return f
}

// Run implements Runner.
func (f *FakeRunner) Run(ctx context.Context, dir string, timeout time.Duration, args ...string) (string, string, error) {
	key := strings.Join(args, " ")
	f.Calls = append(f.Calls, key)
	r, ok := f.Responses[key]
	if !ok {
		return "", "", &ExitError{Code: 128, Stderr: fmt.Sprintf("fake: unscripted command %q", key)}
	}
	if r.Block {
		if timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, timeout)
			defer cancel()
		}
		<-ctx.Done()
		ee := &ExitError{Code: -1}
		if ctx.Err() == context.DeadlineExceeded {
			ee.TimedOut = true
		} else {
			ee.Cancelled = true
		}
		return "", "", ee
	}
	if r.Code != 0 {
		return r.Stdout, r.Stderr, &ExitError{Code: r.Code, Stderr: r.Stderr}
	}
	return r.Stdout, r.Stderr, nil
}
