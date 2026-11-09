package ui

import (
	"fmt"
	"io"

	"github.com/aarjang/push-all-repo/internal/model"
)

// Progress prints one line per repository as it is processed.
type Progress struct {
	W     io.Writer
	Style Style
}

// Start announces that repository i of n is being processed.
func (p *Progress) Start(i, n int, name string) {
	if p == nil || p.W == nil {
		return
	}
	fmt.Fprintf(p.W, "[%d/%d] %s ... ", i, n, name)
}

// Done appends the outcome and ends the line.
func (p *Progress) Done(r model.Result) {
	if p == nil || p.W == nil {
		return
	}
	fmt.Fprintln(p.W, Outcome(r, p.Style))
}

// Outcome renders the short suffix for a result.
func Outcome(r model.Result, s Style) string {
	switch r.Status {
	case model.Pushed:
		txt := "pushed"
		if r.CreatedRemote != "" {
			txt += " to new GitHub repository"
		} else if r.Candidate.AheadCount > 0 {
			txt += fmt.Sprintf(" (%s)", plural(r.Candidate.AheadCount, "commit"))
		}
		return s.OK(s.Sym.Pushed + " " + txt)
	case model.UpToDate:
		txt := "up to date"
		if r.Detail != "" {
			txt += " (" + r.Detail + ")"
		}
		return s.Dim(s.Sym.UpToDate + " " + txt)
	case model.NotPushed:
		txt := " not pushed: " + r.Reason.TextWith(r.Detail)
		if r.Committed {
			txt += " (local commit created)"
		}
		return s.Bad(s.Sym.NotPushed + txt)
	case model.SkippedByUser:
		txt := "skipped"
		if r.Detail != "" {
			txt += " (" + r.Detail + ")"
		}
		return s.Warn(s.Sym.Skipped + " " + txt)
	case model.NotProcessed:
		return s.Dim(s.Sym.NotProcessed + " not processed")
	}
	return ""
}

func plural(n int, word string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, word)
	}
	return fmt.Sprintf("%d %ss", n, word)
}
