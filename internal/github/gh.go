package github

import (
	"context"
	"errors"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/aarjang/push-all-repo/internal/gitx"
)

// ErrNameTaken means a repository with the proposed name already exists.
var ErrNameTaken = errors.New("a GitHub repository with that name already exists")

// Client is what the orchestrator needs from GitHub.
type Client interface {
	// Available reports whether the gh executable is installed.
	Available() bool
	// LoggedIn reports whether gh has a github.com login.
	LoggedIn(ctx context.Context) bool
	// CreateRepo creates the repository, adds it as origin of the repo at
	// path, pushes, and returns the new repository's URL.
	CreateRepo(ctx context.Context, path, name string, private bool, timeout time.Duration) (url string, err error)
}

// GhClient shells out to the GitHub CLI.
type GhClient struct {
	R       gitx.Runner
	program string
}

// NewGhClient looks up gh on PATH. The client is still usable when gh is
// missing; Available then reports false.
func NewGhClient(verbose io.Writer) *GhClient {
	p, err := exec.LookPath("gh")
	if err != nil {
		return &GhClient{}
	}
	return &GhClient{R: gitx.NewProgramRunner(p, verbose), program: p}
}

// Available implements Client.
func (c *GhClient) Available() bool { return c != nil && c.R != nil }

// LoggedIn implements Client.
func (c *GhClient) LoggedIn(ctx context.Context) bool {
	if !c.Available() {
		return false
	}
	_, _, err := c.R.Run(ctx, "", 15*time.Second, "auth", "status", "--hostname", "github.com")
	return err == nil
}

// CreateRepo implements Client.
func (c *GhClient) CreateRepo(ctx context.Context, path, name string, private bool, timeout time.Duration) (string, error) {
	if !c.Available() {
		return "", errors.New("gh is not installed")
	}
	vis := "--public"
	if private {
		vis = "--private"
	}
	_, stderr, err := c.R.Run(ctx, path, timeout, "repo", "create", name, vis, "--source", ".", "--remote", "origin", "--push")
	if err != nil {
		low := strings.ToLower(stderr)
		if strings.Contains(low, "already exists") || strings.Contains(low, "name already exists") {
			return "", ErrNameTaken
		}
		var ee *gitx.ExitError
		if errors.As(err, &ee) && (ee.TimedOut || ee.Cancelled) {
			return "", err
		}
		if line := gitx.LastLine(stderr); line != "" {
			return "", errors.New(line)
		}
		return "", err
	}
	out, _, verr := c.R.Run(ctx, path, 15*time.Second, "repo", "view", "--json", "url", "-q", ".url")
	if verr != nil {
		return "", nil
	}
	return strings.TrimSpace(out), nil
}
