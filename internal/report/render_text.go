package report

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aarjang/push-all-repo/internal/model"
	"github.com/aarjang/push-all-repo/internal/ui"
)

// MaxNameWidth caps the repository column.
const MaxNameWidth = 40

const timeLayout = "2006-01-02 15:04:05"

// RenderText writes the on-screen report (contracts/report-format.md).
func RenderText(w io.Writer, rep Report, s ui.Style) {
	sep, dot := " — ", " · "
	if !s.Unicode {
		sep, dot = " - ", " | "
	}
	fmt.Fprintf(w, "%s%s%s%s%s%s%s\n", s.Bold("Push report"), sep, rep.Run.StartedAt.Format(timeLayout), sep, rep.Run.RootPath, sep, rep.Run.Mode.Short())
	if rep.Run.Interrupted {
		fmt.Fprintln(w, s.Warn("Run was interrupted."))
	}

	nameW, branchW := columnWidths(rep)
	for _, st := range model.StatusOrder {
		rows := rep.Groups[st]
		if len(rows) == 0 {
			continue
		}
		fmt.Fprintf(w, "\n%s (%d)\n", s.Bold(st.Title()), len(rows))
		for _, r := range rows {
			fmt.Fprintf(w, "  %s %s  %s  %s\n", symbol(st, s), pad(truncate(r.Candidate.Name), nameW), pad(branchOrDash(r), branchW), strings.TrimSpace(Details(r)))
		}
	}
	if len(rep.NotRepos) > 0 {
		var names []string
		for _, r := range rep.NotRepos {
			names = append(names, r.Candidate.Name)
		}
		fmt.Fprintf(w, "\nNot a Git repository: %s\n", strings.Join(names, ", "))
	}
	fmt.Fprintf(w, "\n%s %s%s%s%s%s\n", s.Bold("Summary:"), Summary(rep), dot, Folders(rep.Folders), dot, FormatDuration(rep.Run.Duration()))
}

// Summary is "N pushed, N up to date, N not pushed, N skipped by user[, N not processed]".
func Summary(rep Report) string {
	parts := []string{
		fmt.Sprintf("%d pushed", rep.Counts[model.Pushed]),
		fmt.Sprintf("%d up to date", rep.Counts[model.UpToDate]),
		fmt.Sprintf("%d not pushed", rep.Counts[model.NotPushed]),
		fmt.Sprintf("%d skipped by user", rep.Counts[model.SkippedByUser]),
	}
	if n := rep.Counts[model.NotProcessed]; n > 0 {
		parts = append(parts, fmt.Sprintf("%d not processed", n))
	}
	return strings.Join(parts, ", ")
}

// Details is the right-hand column for one result.
func Details(r model.Result) string {
	switch r.Status {
	case model.Pushed:
		var d string
		switch {
		case r.CreatedRemote != "":
			d = fmt.Sprintf("pushed to new GitHub repository %s (%s)", r.CreatedRemote, r.CreatedVisibility)
		case r.Candidate.Upstream == "":
			d = "branch published (upstream set)"
		case r.Candidate.AheadCount == 1:
			d = "1 commit pushed"
		case r.Candidate.AheadCount > 1:
			d = fmt.Sprintf("%d commits pushed", r.Candidate.AheadCount)
		default:
			d = "pushed"
		}
		if r.Committed {
			d += fmt.Sprintf("; committed by tool: %q", r.CommitMessage)
		}
		if r.Detail != "" {
			d += "; " + r.Detail
		}
		return d
	case model.UpToDate:
		if r.Detail != "" {
			return "(" + r.Detail + ")"
		}
		return ""
	case model.NotPushed:
		d := r.Reason.TextWith(r.Detail)
		if r.Committed {
			d += fmt.Sprintf("; a local commit was created but not pushed: %q", r.CommitMessage)
		}
		return d
	case model.SkippedByUser:
		return r.Detail
	case model.NotProcessed:
		return "interrupted before reaching this repository"
	}
	return ""
}

// FormatDuration renders whole seconds, e.g. "41 s".
func FormatDuration(d time.Duration) string {
	return fmt.Sprintf("%d s", int(d.Round(time.Second)/time.Second))
}

func symbol(st model.Status, s ui.Style) string {
	switch st {
	case model.Pushed:
		return s.OK(s.Sym.Pushed)
	case model.UpToDate:
		return s.Dim(s.Sym.UpToDate)
	case model.NotPushed:
		return s.Bad(s.Sym.NotPushed)
	case model.SkippedByUser:
		return s.Warn(s.Sym.Skipped)
	case model.NotProcessed:
		return s.Dim(s.Sym.NotProcessed)
	}
	return " "
}

func branchOrDash(r model.Result) string {
	if r.Candidate.Branch == "" {
		return "-"
	}
	return r.Candidate.Branch
}

func columnWidths(rep Report) (nameW, branchW int) {
	nameW, branchW = 4, 4
	for _, st := range model.StatusOrder {
		for _, r := range rep.Groups[st] {
			if n := len([]rune(truncate(r.Candidate.Name))); n > nameW {
				nameW = n
			}
			if n := len([]rune(branchOrDash(r))); n > branchW {
				branchW = n
			}
		}
	}
	return nameW, branchW
}

func truncate(name string) string {
	rs := []rune(name)
	if len(rs) <= MaxNameWidth {
		return name
	}
	return string(rs[:MaxNameWidth-1]) + "…"
}

func pad(s string, w int) string {
	n := len([]rune(s))
	if n >= w {
		return s
	}
	return s + strings.Repeat(" ", w-n)
}
