package gitx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNoForbiddenGitArguments guards FR-015: the tool never force-pushes,
// rewrites history, switches branches or touches the stash.
func TestNoForbiddenGitArguments(t *testing.T) {
	forbidden := []string{
		`"--force"`, `"--force-with-lease"`, `"-f"`, `"--delete"`, `"--no-verify"`,
		`"reset"`, `"rebase"`, `"checkout"`, `"switch"`, `"clean"`, `"stash"`,
	}
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for i, line := range strings.Split(string(src), "\n") {
			if strings.Contains(line, "// not a git argument") {
				continue
			}
			for _, word := range forbidden {
				if strings.Contains(line, word) {
					t.Errorf("%s:%d contains forbidden git argument %s", f, i+1, word)
				}
			}
		}
	}
}
