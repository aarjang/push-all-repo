package ui

import (
	"os"

	"github.com/charmbracelet/lipgloss"
)

// IsTTY reports whether f is attached to a terminal.
func IsTTY(f *os.File) bool {
	if f == nil {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// Symbols are the per-status markers used in progress lines and the report.
type Symbols struct {
	Pushed, UpToDate, Skipped, NotPushed, NotProcessed, Ahead string
}

var unicodeSymbols = Symbols{"✔", "=", "–", "✘", "○", "↑"}
var asciiSymbols = Symbols{"+", "=", "-", "x", "o", "^"}

// Style wraps the colouring decisions so callers never touch lipgloss.
type Style struct {
	Color   bool
	Unicode bool
	Sym     Symbols
	ok      lipgloss.Style
	warn    lipgloss.Style
	bad     lipgloss.Style
	dim     lipgloss.Style
	bold    lipgloss.Style
}

// NewStyle builds a Style. noColor disables colour and Unicode symbols;
// NO_COLOR in the environment behaves like noColor.
func NewStyle(noColor bool) Style {
	if os.Getenv("NO_COLOR") != "" {
		noColor = true
	}
	s := Style{Color: !noColor, Unicode: !noColor && consoleIsUTF8()}
	if s.Unicode {
		s.Sym = unicodeSymbols
	} else {
		s.Sym = asciiSymbols
	}
	if s.Color {
		s.ok = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
		s.warn = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
		s.bad = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
		s.dim = lipgloss.NewStyle().Faint(true)
		s.bold = lipgloss.NewStyle().Bold(true)
	}
	return s
}

// PlainStyle is colourless ASCII, used for files and tests.
func PlainStyle() Style { return Style{Sym: asciiSymbols} }

func (s Style) apply(st lipgloss.Style, text string) string {
	if !s.Color {
		return text
	}
	return st.Render(text)
}

// OK colours a success.
func (s Style) OK(t string) string { return s.apply(s.ok, t) }

// Warn colours a warning.
func (s Style) Warn(t string) string { return s.apply(s.warn, t) }

// Bad colours a failure.
func (s Style) Bad(t string) string { return s.apply(s.bad, t) }

// Dim de-emphasises text.
func (s Style) Dim(t string) string { return s.apply(s.dim, t) }

// Bold emphasises text.
func (s Style) Bold(t string) string { return s.apply(s.bold, t) }
