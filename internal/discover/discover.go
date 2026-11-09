package discover

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/aarjang/push-all-repo/internal/gitx"
	"github.com/aarjang/push-all-repo/internal/model"
)

// Discover lists candidate projects under root (research R7): the root itself
// first when it is a repository, then its direct child directories sorted
// case-insensitively. Hidden folders are skipped, symlinked directories are
// followed, and nothing is scanned recursively. Children that are merely
// sub-folders of a root repository are dropped because they belong to it.
func Discover(ctx context.Context, root string, g gitx.Inspector) ([]model.Candidate, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	rootCand, err := g.Inspect(ctx, root)
	if err != nil {
		return nil, err
	}

	var names []string
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		full := filepath.Join(root, name)
		if e.IsDir() {
			names = append(names, name)
			continue
		}
		if e.Type()&os.ModeSymlink != 0 {
			if fi, err := os.Stat(full); err == nil && fi.IsDir() {
				names = append(names, name)
			}
		}
	}
	sort.SliceStable(names, func(i, j int) bool {
		li, lj := strings.ToLower(names[i]), strings.ToLower(names[j])
		if li != lj {
			return li < lj
		}
		return names[i] < names[j]
	})

	out := make([]model.Candidate, 0, len(names)+1)
	if rootCand.IsRepo {
		out = append(out, rootCand)
	}
	for _, name := range names {
		if ctx.Err() != nil {
			return out, ctx.Err()
		}
		c, err := g.Inspect(ctx, filepath.Join(root, name))
		if err != nil {
			return nil, err
		}
		if c.Nested && rootCand.IsRepo && gitx.SamePath(c.TopLevel, root) {
			continue // part of the root project, not a project of its own
		}
		out = append(out, c)
	}
	return out, nil
}
