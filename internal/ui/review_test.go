package ui

import (
	"testing"

	"github.com/aarjang/push-all-repo/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestFormatReviewRow(t *testing.T) {
	plain := PlainStyle()
	uni := Style{Unicode: true, Sym: unicodeSymbols}
	base := model.Candidate{Name: "app", IsRepo: true, Branch: "main", Remotes: []string{"origin"}}

	ahead := base
	ahead.Upstream, ahead.AheadCount = "origin/main", 2
	assert.Equal(t, "app  [main]  ^2", FormatReviewRow(ahead, plain))
	assert.Equal(t, "app  [main]  ↑2", FormatReviewRow(ahead, uni))

	upToDate := base
	upToDate.Upstream, upToDate.AheadCount = "origin/main", 0
	assert.Equal(t, "app  [main]  up to date", FormatReviewRow(upToDate, plain))

	noUpstream := base
	noUpstream.AheadCount = -1
	assert.Equal(t, "app  [main]  no upstream", FormatReviewRow(noUpstream, plain))

	noRemote := base
	noRemote.Remotes, noRemote.AheadCount = nil, -1
	assert.Equal(t, "app  [main]  no remote", FormatReviewRow(noRemote, plain))

	dirty := upToDate
	dirty.Dirty = true
	assert.Equal(t, "app  [main]  up to date  dirty", FormatReviewRow(dirty, plain))

	detached := base
	detached.Branch = ""
	assert.Equal(t, "app  [detached]  detached", FormatReviewRow(detached, plain))

	merging := upToDate
	merging.InProgressOp = "merge"
	assert.Equal(t, "app  [main]  merge in progress  up to date", FormatReviewRow(merging, plain))
}

func TestOutcome(t *testing.T) {
	s := PlainStyle()
	assert.Equal(t, "+ pushed (2 commits)", Outcome(model.Result{Status: model.Pushed, Candidate: model.Candidate{AheadCount: 2}}, s))
	assert.Equal(t, "+ pushed (1 commit)", Outcome(model.Result{Status: model.Pushed, Candidate: model.Candidate{AheadCount: 1}}, s))
	assert.Equal(t, "= up to date", Outcome(model.Result{Status: model.UpToDate}, s))
	assert.Equal(t, "x not pushed: authentication failed", Outcome(model.Result{Status: model.NotPushed, Reason: model.AuthFailed}, s))
	assert.Equal(t, "- skipped (excluded in review)", Outcome(model.Result{Status: model.SkippedByUser, Detail: "excluded in review"}, s))
	assert.Equal(t, "o not processed", Outcome(model.Result{Status: model.NotProcessed}, s))
}

func TestNonInteractivePrompter(t *testing.T) {
	var p NonInteractivePrompter
	_, err := p.Mode()
	var nf *NeedFlagError
	assert.ErrorAs(t, err, &nf)
	assert.Contains(t, err.Error(), "--mode")
	_, err = p.DirtyDecision(model.Candidate{}, "")
	assert.ErrorAs(t, err, &nf)
	ok, _ := p.ConfirmStart(3)
	assert.True(t, ok)
	_, save, _ := p.SaveReport("x")
	assert.False(t, save)
}
