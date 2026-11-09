package report

import (
	"bytes"
	"os"
	"path/filepath"
	"time"
)

// DefaultReportPath is "push-report-YYYYMMDD-HHMMSS.md" for the given time.
func DefaultReportPath(now time.Time) string {
	return "push-report-" + now.Format("20060102-150405") + ".md"
}

// Save writes the Markdown report to path (UTF-8, LF line endings on every
// OS), creating parent directories as needed.
func Save(path string, rep Report) error {
	var buf bytes.Buffer
	RenderMarkdown(&buf, rep)
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}
