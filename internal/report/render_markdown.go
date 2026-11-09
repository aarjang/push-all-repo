package report

import (
	"fmt"
	"io"
	"strings"

	"github.com/aarjang/push-all-repo/internal/model"
)

// RenderMarkdown writes the file report (contracts/report-format.md).
// It always uses "\n" so the output is identical on every OS.
func RenderMarkdown(w io.Writer, rep Report) {
	var b strings.Builder
	b.WriteString("# Push report\n\n")
	fmt.Fprintf(&b, "- **Date**: %s\n", rep.Run.StartedAt.Format(timeLayout))
	fmt.Fprintf(&b, "- **Root**: %s\n", rep.Run.RootPath)
	fmt.Fprintf(&b, "- **Mode**: %s\n", rep.Run.Mode.String())
	fmt.Fprintf(&b, "- **Tool**: pushall %s\n", displayVersion(rep.Run.ToolVersion))
	fmt.Fprintf(&b, "- **Duration**: %s\n", FormatDuration(rep.Run.Duration()))
	if rep.Run.Interrupted {
		b.WriteString("- **Interrupted**: yes\n")
	}
	fmt.Fprintf(&b, "- **Summary**: %s · %s\n", strings.ReplaceAll(Summary(rep), ", ", " · "), Folders(rep.Folders))

	for _, st := range model.StatusOrder {
		rows := rep.Groups[st]
		if len(rows) == 0 {
			continue
		}
		fmt.Fprintf(&b, "\n## %s (%d)\n\n", st.Title(), len(rows))
		col := "Details"
		if st == model.NotPushed || st == model.SkippedByUser || st == model.NotProcessed {
			col = "Reason"
		}
		fmt.Fprintf(&b, "| Repository | Branch | %s |\n|------------|--------|---------|\n", col)
		for _, r := range rows {
			branch := r.Candidate.Branch
			if branch == "" {
				branch = "—"
			}
			fmt.Fprintf(&b, "| %s | %s | %s |\n", cell(r.Candidate.Name), cell(branch), cell(Details(r)))
		}
	}
	if len(rep.NotRepos) > 0 {
		var names []string
		for _, r := range rep.NotRepos {
			names = append(names, r.Candidate.Name)
		}
		fmt.Fprintf(&b, "\n## Not a Git repository (%d)\n\n%s\n", len(names), strings.Join(names, ", "))
	}
	out := strings.ReplaceAll(b.String(), "\r\n", "\n")
	_, _ = io.WriteString(w, out)
}

// displayVersion prefixes release numbers with "v" and leaves "dev" alone.
func displayVersion(v string) string {
	if v == "" {
		return "dev"
	}
	if v[0] >= '0' && v[0] <= '9' {
		return "v" + v
	}
	return v
}

// Folders renders "1 folder" / "N folders".
func Folders(n int) string {
	if n == 1 {
		return "1 folder"
	}
	return fmt.Sprintf("%d folders", n)
}

func cell(s string) string {
	s = strings.ReplaceAll(s, "|", "\\|")
	s = strings.ReplaceAll(s, "\n", " ")
	return s
}
