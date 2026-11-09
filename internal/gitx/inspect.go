package gitx

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/aarjang/push-all-repo/internal/model"
)

// InspectTimeout bounds every read-only git call.
const InspectTimeout = 15 * time.Second

// Inspector fills a Candidate from a path.
type Inspector interface {
	Inspect(ctx context.Context, path string) (model.Candidate, error)
}

// Client is everything the orchestrator needs from git.
type Client interface {
	Inspector
	AddAll(ctx context.Context, path string) error
	Commit(ctx context.Context, path, message string) error
	Push(ctx context.Context, c model.Candidate, timeout time.Duration) error
}

// Git implements Client on top of a Runner.
type Git struct {
	R Runner
}

// New returns a Git client.
func New(r Runner) *Git { return &Git{R: r} }

func (g *Git) run(ctx context.Context, dir string, args ...string) (string, error) {
	out, _, err := g.R.Run(ctx, dir, InspectTimeout, args...)
	return strings.TrimSpace(out), err
}

// TopLevel returns the repository root containing path, or isRepo=false.
func (g *Git) TopLevel(ctx context.Context, path string) (top string, isRepo bool) {
	out, err := g.run(ctx, path, "rev-parse", "--show-toplevel")
	if err != nil || out == "" {
		return "", false
	}
	return filepath.Clean(filepath.FromSlash(out)), true
}

// CurrentBranch returns the branch name, or "" when HEAD is detached.
func (g *Git) CurrentBranch(ctx context.Context, path string) string {
	out, err := g.run(ctx, path, "symbolic-ref", "--quiet", "--short", "HEAD")
	if err != nil {
		return ""
	}
	return out
}

// Upstream returns "remote/branch" or "".
func (g *Git) Upstream(ctx context.Context, path string) string {
	out, err := g.run(ctx, path, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}")
	if err != nil {
		return ""
	}
	return out
}

// Remotes lists configured remote names.
func (g *Git) Remotes(ctx context.Context, path string) []string {
	out, err := g.run(ctx, path, "remote")
	if err != nil || out == "" {
		return nil
	}
	var names []string
	for _, l := range strings.Split(out, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			names = append(names, l)
		}
	}
	return names
}

// AheadCount returns commits ahead of upstream; -1 when it cannot be told.
func (g *Git) AheadCount(ctx context.Context, path string) int {
	out, err := g.run(ctx, path, "rev-list", "--count", "@{u}..HEAD")
	if err != nil {
		return -1
	}
	n, err := strconv.Atoi(out)
	if err != nil {
		return -1
	}
	return n
}

// StatusLines returns porcelain v1 status lines (empty when clean).
func (g *Git) StatusLines(ctx context.Context, path string) []string {
	out, err := g.run(ctx, path, "status", "--porcelain=v1", "--untracked-files=normal")
	if err != nil || out == "" {
		return nil
	}
	var lines []string
	for _, l := range strings.Split(strings.ReplaceAll(out, "\r\n", "\n"), "\n") {
		if l != "" {
			lines = append(lines, l)
		}
	}
	return lines
}

// InProgressOp reports which operation (merge, rebase or cherry-pick) is in progress, or "".
func (g *Git) InProgressOp(ctx context.Context, path string) string {
	// Display names of operations, never passed to git.
	checks := []struct{ file, op string }{
		{"MERGE_HEAD", "merge"},
		{"rebase-merge", "rebase"},          // not a git argument
		{"rebase-apply", "rebase"},          // not a git argument
		{"CHERRY_PICK_HEAD", "cherry-pick"}, // not a git argument
	}
	for _, c := range checks {
		out, err := g.run(ctx, path, "rev-parse", "--git-path", c.file)
		if err != nil || out == "" {
			continue
		}
		p := filepath.FromSlash(out)
		if !filepath.IsAbs(p) {
			p = filepath.Join(path, p)
		}
		if _, err := os.Stat(p); err == nil {
			return c.op
		}
	}
	return ""
}

// IdentityOK reports whether user.name and user.email are configured.
func (g *Git) IdentityOK(ctx context.Context, path string) bool {
	name, err1 := g.run(ctx, path, "config", "user.name")
	email, err2 := g.run(ctx, path, "config", "user.email")
	return err1 == nil && err2 == nil && name != "" && email != ""
}

// Inspect fills a Candidate for path.
func (g *Git) Inspect(ctx context.Context, path string) (model.Candidate, error) {
	c := model.Candidate{Path: path, Name: filepath.Base(path), AheadCount: -1}
	top, isRepo := g.TopLevel(ctx, path)
	if !isRepo {
		return c, nil
	}
	c.TopLevel = top
	if !SamePath(top, path) {
		c.Nested = true
		return c, nil
	}
	c.IsRepo = true
	c.Branch = g.CurrentBranch(ctx, path)
	c.Remotes = g.Remotes(ctx, path)
	c.Upstream = g.Upstream(ctx, path)
	if c.Upstream != "" {
		c.AheadCount = g.AheadCount(ctx, path)
	}
	c.DirtyFiles = g.StatusLines(ctx, path)
	c.Dirty = len(c.DirtyFiles) > 0
	c.InProgressOp = g.InProgressOp(ctx, path)
	c.IdentityOK = g.IdentityOK(ctx, path)
	return c, nil
}

// SamePath reports whether a and b name the same directory, resolving
// symlinks and ignoring case on case-insensitive platforms.
func SamePath(a, b string) bool {
	norm := func(p string) string {
		p = filepath.Clean(p)
		if r, err := filepath.EvalSymlinks(p); err == nil {
			p = r
		}
		if abs, err := filepath.Abs(p); err == nil {
			p = abs
		}
		return p
	}
	na, nb := norm(a), norm(b)
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		return strings.EqualFold(na, nb)
	}
	return na == nb
}
