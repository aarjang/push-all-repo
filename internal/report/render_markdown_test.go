package report

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderMarkdownGolden(t *testing.T) {
	var buf bytes.Buffer
	RenderMarkdown(&buf, Build(fixtureRun()))
	assert.NotContains(t, buf.String(), "\r\n")
	checkGolden(t, "report.md", buf.Bytes())
}

func TestDefaultReportPath(t *testing.T) {
	now := time.Date(2026, 11, 9, 14, 3, 17, 0, time.UTC)
	assert.Equal(t, "push-report-20261109-140317.md", DefaultReportPath(now))
}

func TestSaveCreatesDirs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "deeper", "r.md")
	require.NoError(t, Save(path, Build(fixtureRun())))
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	s := string(data)
	assert.True(t, strings.HasPrefix(s, "# Push report\n"))
	for _, name := range []string{"api-server", "website", "new-tool", "dotfiles", "notes", "legacy-app", "scratch", "experiments", "Downloads", "screenshots"} {
		re := regexp.MustCompile(`(^|[\s|])` + regexp.QuoteMeta(name) + `($|[\s|,])`)
		assert.Len(t, re.FindAllStringIndex(s, -1), 1, "%s should appear exactly once", name)
	}
}

func TestCellEscapesPipes(t *testing.T) {
	assert.Equal(t, "a \\| b", cell("a | b"))
}
