package ui

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/huh"

	"github.com/aarjang/push-all-repo/internal/model"
)

// ErrAborted is returned when the user cancels a prompt (Ctrl+C / Esc).
var ErrAborted = errors.New("prompt aborted by user")

// Prompter is every question the tool can ask. The orchestrator only talks
// to this interface so it can be driven by scripts in tests.
type Prompter interface {
	Mode() (model.Mode, error)
	// Path asks for a path; validate is called on each attempt and its error
	// is shown to the user until it returns nil.
	Path(mode model.Mode, validate func(string) error) (string, error)
	// Review shows the discovered candidates and returns the ones to keep.
	Review(cands []model.Candidate) ([]model.Candidate, error)
	ConfirmStart(n int) (bool, error)
	DirtyDecision(c model.Candidate, defaultMessage string) (model.Decision, error)
	// CreateRepoOffer asks whether to create a GitHub repository. nameTaken is
	// true when a previous attempt failed because the name already exists.
	CreateRepoOffer(c model.Candidate, defaultName string, vis model.Visibility, nameTaken bool) (model.Decision, error)
	// SaveReport returns the path to save to and whether to save at all.
	SaveReport(defaultPath string) (string, bool, error)
}

// HuhPrompter asks with charmbracelet/huh forms.
type HuhPrompter struct {
	Style Style
	Out   io.Writer
}

// NewHuhPrompter returns an interactive prompter.
func NewHuhPrompter(style Style, out io.Writer) *HuhPrompter {
	return &HuhPrompter{Style: style, Out: out}
}

func (p *HuhPrompter) runForm(fields ...huh.Field) error {
	form := huh.NewForm(huh.NewGroup(fields...))
	if !p.Style.Color {
		form = form.WithTheme(huh.ThemeBase())
	}
	err := form.Run()
	if errors.Is(err, huh.ErrUserAborted) {
		return ErrAborted
	}
	return err
}

// Mode implements Prompter.
func (p *HuhPrompter) Mode() (model.Mode, error) {
	var m model.Mode
	err := p.runForm(huh.NewSelect[model.Mode]().
		Title("What do you want to push?").
		Options(
			huh.NewOption("A folder that contains my projects", model.FolderOfProjects),
			huh.NewOption("A single project", model.SingleProject),
		).Value(&m))
	return m, err
}

// Path implements Prompter.
func (p *HuhPrompter) Path(mode model.Mode, validate func(string) error) (string, error) {
	title := "Path to the folder that contains your projects"
	if mode == model.SingleProject {
		title = "Path to the project (a Git repository)"
	}
	var s string
	err := p.runForm(huh.NewInput().Title(title).Placeholder("~/projects").
		Validate(func(v string) error { return validate(strings.TrimSpace(v)) }).Value(&s))
	return strings.TrimSpace(s), err
}

// Review implements Prompter (US3).
func (p *HuhPrompter) Review(cands []model.Candidate) ([]model.Candidate, error) {
	var opts []huh.Option[int]
	var selected []int
	for i, c := range cands {
		if !c.IsRepo {
			continue // plain folders are reported, not chosen
		}
		opts = append(opts, huh.NewOption(FormatReviewRow(c, p.Style), i).Selected(true))
		selected = append(selected, i)
	}
	if len(opts) == 0 {
		return cands, nil
	}
	err := p.runForm(huh.NewMultiSelect[int]().
		Title("Repositories to push (space toggles, enter confirms)").
		Options(opts...).Value(&selected))
	if err != nil {
		return nil, err
	}
	keep := map[int]bool{}
	for _, i := range selected {
		keep[i] = true
	}
	out := make([]model.Candidate, 0, len(cands))
	for i, c := range cands {
		if !c.IsRepo || keep[i] {
			out = append(out, c)
		}
	}
	return out, nil
}

// ConfirmStart implements Prompter.
func (p *HuhPrompter) ConfirmStart(n int) (bool, error) {
	ok := true
	word := "repositories"
	if n == 1 {
		word = "repository"
	}
	err := p.runForm(huh.NewConfirm().Title(fmt.Sprintf("Push %d %s now?", n, word)).
		Affirmative("Yes, push").Negative("No, quit").Value(&ok))
	return ok, err
}

// DirtyDecision implements Prompter (FR-007/FR-007a).
func (p *HuhPrompter) DirtyDecision(c model.Candidate, defaultMessage string) (model.Decision, error) {
	d := model.Decision{Kind: model.DecisionDirty}
	files := c.DirtyFiles
	const maxShow = 15
	shown := files
	more := ""
	if len(files) > maxShow {
		shown = files[:maxShow]
		more = fmt.Sprintf("\n… and %d more", len(files)-maxShow)
	}
	desc := strings.Join(shown, "\n") + more
	choice := model.PushOnly
	err := p.runForm(
		huh.NewNote().Title(fmt.Sprintf("%s has uncommitted changes", c.Name)).Description(desc),
		huh.NewSelect[model.DirtyChoice]().Title("What should happen?").
			Options(
				huh.NewOption("Commit all changes and push", model.CommitAndPush),
				huh.NewOption("Push existing commits only (leave changes untouched)", model.PushOnly),
				huh.NewOption("Skip this repository", model.Skip),
			).Value(&choice),
	)
	if err != nil {
		return d, err
	}
	d.DirtyChoice = choice
	if choice == model.CommitAndPush {
		msg := defaultMessage
		if err := p.runForm(huh.NewInput().Title("Commit message").Value(&msg).
			Validate(func(v string) error {
				if strings.TrimSpace(v) == "" {
					return errors.New("commit message cannot be empty")
				}
				return nil
			})); err != nil {
			return d, err
		}
		d.CommitMessage = strings.TrimSpace(msg)
	}
	all := false
	if err := p.runForm(huh.NewConfirm().Title("Apply this answer to all remaining repositories with uncommitted changes?").
		Affirmative("Yes, apply to all").Negative("No, ask each time").Value(&all)); err != nil {
		return d, err
	}
	d.ApplyToAll = all
	return d, nil
}

// CreateRepoOffer implements Prompter (FR-008).
func (p *HuhPrompter) CreateRepoOffer(c model.Candidate, defaultName string, vis model.Visibility, nameTaken bool) (model.Decision, error) {
	d := model.Decision{Kind: model.DecisionCreateRepo, RepoName: defaultName, Visibility: vis}
	title := fmt.Sprintf("%s has no remote. Create a GitHub repository and push?", c.Name)
	if nameTaken {
		title = fmt.Sprintf("A repository named %q already exists on your account. Try another name?", defaultName)
	}
	create := true
	name := defaultName
	err := p.runForm(
		huh.NewConfirm().Title(title).Affirmative("Yes, create").Negative("No, skip").Value(&create),
		huh.NewInput().Title("Repository name").Value(&name).
			Validate(func(v string) error {
				if strings.TrimSpace(v) == "" {
					return errors.New("name cannot be empty")
				}
				return nil
			}),
		huh.NewSelect[model.Visibility]().Title("Visibility").
			Options(
				huh.NewOption("Private (recommended)", model.Private),
				huh.NewOption("Public", model.Public),
			).Value(&vis),
	)
	if err != nil {
		return d, err
	}
	d.CreateRepo = create
	d.RepoName = strings.TrimSpace(name)
	d.Visibility = vis
	if nameTaken {
		return d, nil
	}
	all := false
	if err := p.runForm(huh.NewConfirm().Title("Apply this answer to all remaining repositories without a remote?").
		Affirmative("Yes, apply to all").Negative("No, ask each time").Value(&all)); err != nil {
		return d, err
	}
	d.ApplyToAll = all
	return d, nil
}

// SaveReport implements Prompter (US4).
func (p *HuhPrompter) SaveReport(defaultPath string) (string, bool, error) {
	save := false
	err := p.runForm(huh.NewConfirm().Title(fmt.Sprintf("Save this report to %s?", defaultPath)).
		Affirmative("Yes, save").Negative("No").Value(&save))
	return defaultPath, save, err
}
