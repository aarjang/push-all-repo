package ui

import "github.com/aarjang/push-all-repo/internal/model"

// ScriptedPrompter answers from queues; the last value repeats when a queue
// runs dry. It records how often each prompt was shown.
type ScriptedPrompter struct {
	ModeAnswer  model.Mode
	PathAnswers []string
	// ReviewFn, when set, decides what to keep; nil keeps everything.
	ReviewFn      func([]model.Candidate) []model.Candidate
	Confirm       bool
	Dirty         []model.Decision
	Create        []model.Decision
	SavePath      string
	Save          bool
	ReviewCalls   int
	ConfirmCalls  int
	DirtyCalls    int
	CreateCalls   int
	SaveCalls     int
	CreateOffered []string // defaultName per CreateRepoOffer call
	NameTaken     []bool
}

// NewScripted returns a prompter that confirms everything and keeps all.
func NewScripted() *ScriptedPrompter {
	return &ScriptedPrompter{Confirm: true}
}

func (s *ScriptedPrompter) Mode() (model.Mode, error) { return s.ModeAnswer, nil }

func (s *ScriptedPrompter) Path(_ model.Mode, validate func(string) error) (string, error) {
	var last string
	for i, p := range s.PathAnswers {
		last = p
		if err := validate(p); err == nil || i == len(s.PathAnswers)-1 {
			if err != nil {
				return "", err
			}
			return p, nil
		}
	}
	return last, nil
}

func (s *ScriptedPrompter) Review(c []model.Candidate) ([]model.Candidate, error) {
	s.ReviewCalls++
	if s.ReviewFn != nil {
		return s.ReviewFn(c), nil
	}
	return c, nil
}

func (s *ScriptedPrompter) ConfirmStart(int) (bool, error) {
	s.ConfirmCalls++
	return s.Confirm, nil
}

func pop(q *[]model.Decision) model.Decision {
	if len(*q) == 0 {
		return model.Decision{}
	}
	d := (*q)[0]
	if len(*q) > 1 {
		*q = (*q)[1:]
	}
	return d
}

func (s *ScriptedPrompter) DirtyDecision(_ model.Candidate, defaultMessage string) (model.Decision, error) {
	s.DirtyCalls++
	d := pop(&s.Dirty)
	if d.DirtyChoice == model.CommitAndPush && d.CommitMessage == "" {
		d.CommitMessage = defaultMessage
	}
	return d, nil
}

func (s *ScriptedPrompter) CreateRepoOffer(_ model.Candidate, defaultName string, vis model.Visibility, nameTaken bool) (model.Decision, error) {
	s.CreateCalls++
	s.CreateOffered = append(s.CreateOffered, defaultName)
	s.NameTaken = append(s.NameTaken, nameTaken)
	d := pop(&s.Create)
	if d.RepoName == "" {
		d.RepoName = defaultName
	}
	return d, nil
}

func (s *ScriptedPrompter) SaveReport(defaultPath string) (string, bool, error) {
	s.SaveCalls++
	if s.SavePath != "" {
		return s.SavePath, s.Save, nil
	}
	return defaultPath, s.Save, nil
}
