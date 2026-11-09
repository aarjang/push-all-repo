package ui

import (
	"fmt"

	"github.com/aarjang/push-all-repo/internal/model"
)

// NeedFlagError says which flag must supply an answer when no terminal is
// attached (contracts/cli.md "Non-interactive mode").
type NeedFlagError struct {
	What string
	Flag string
}

func (e *NeedFlagError) Error() string {
	return fmt.Sprintf("%s is required: pass %s when not running in a terminal", e.What, e.Flag)
}

// NonInteractivePrompter fails every prompt with a NeedFlagError, except the
// optional ones (review, confirmation, saving) which take the safe default.
type NonInteractivePrompter struct{}

func (NonInteractivePrompter) Mode() (model.Mode, error) {
	return model.ModeUnset, &NeedFlagError{"the mode", "--mode folder|single"}
}

func (NonInteractivePrompter) Path(model.Mode, func(string) error) (string, error) {
	return "", &NeedFlagError{"a path", "PATH as the last argument"}
}

func (NonInteractivePrompter) Review(c []model.Candidate) ([]model.Candidate, error) {
	return c, nil
}

func (NonInteractivePrompter) ConfirmStart(int) (bool, error) { return true, nil }

func (NonInteractivePrompter) DirtyDecision(model.Candidate, string) (model.Decision, error) {
	return model.Decision{}, &NeedFlagError{"a decision for uncommitted changes", "--dirty commit|push-only|skip"}
}

func (NonInteractivePrompter) CreateRepoOffer(model.Candidate, string, model.Visibility, bool) (model.Decision, error) {
	return model.Decision{}, &NeedFlagError{"a decision for repositories without a remote", "--create-repo yes|no"}
}

// SaveReport never saves when no terminal is attached; use --report PATH.
func (NonInteractivePrompter) SaveReport(string) (string, bool, error) { return "", false, nil }
