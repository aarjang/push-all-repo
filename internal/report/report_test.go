package report

import (
	"testing"

	"github.com/aarjang/push-all-repo/internal/model"
	"github.com/stretchr/testify/assert"
)

func res(name string, st model.Status, reason model.Reason) model.Result {
	return model.Result{Candidate: model.Candidate{Name: name, Path: "/x/" + name}, Status: st, Reason: reason}
}

func TestBuildGroupsAndCounts(t *testing.T) {
	run := model.Run{Results: []model.Result{
		res("b", model.Pushed, model.ReasonNone),
		res("a", model.Pushed, model.ReasonNone),
		res("c", model.UpToDate, model.ReasonNone),
		res("d", model.NotPushed, model.AuthFailed),
		res("e", model.NotPushed, model.NotARepo),
		res("f", model.SkippedByUser, model.ReasonNone),
	}}
	r := Build(run)
	assert.Equal(t, 2, r.Counts[model.Pushed])
	assert.Equal(t, 1, r.Counts[model.UpToDate])
	assert.Equal(t, 1, r.Counts[model.NotPushed], "plain folders are not counted as not pushed")
	assert.Equal(t, 1, r.Counts[model.SkippedByUser])
	assert.Len(t, r.NotRepos, 1)
	assert.Equal(t, 6, r.Folders)
	assert.Equal(t, "b", r.Groups[model.Pushed][0].Candidate.Name, "processing order is kept")
}

func TestExitCodes(t *testing.T) {
	ok := Build(model.Run{Results: []model.Result{res("a", model.Pushed, 0), res("b", model.UpToDate, 0), res("n", model.NotPushed, model.NotARepo)}})
	assert.Equal(t, 0, ok.ExitCode(false))
	assert.Equal(t, 0, ok.ExitCode(true))

	failed := Build(model.Run{Results: []model.Result{res("a", model.Pushed, 0), res("b", model.NotPushed, model.Rejected)}})
	assert.Equal(t, 1, failed.ExitCode(false))

	skipped := Build(model.Run{Results: []model.Result{res("a", model.SkippedByUser, 0)}})
	assert.Equal(t, 0, skipped.ExitCode(false))
	assert.Equal(t, 1, skipped.ExitCode(true))

	interrupted := Build(model.Run{Interrupted: true, Results: []model.Result{res("a", model.Pushed, 0)}})
	assert.Equal(t, 130, interrupted.ExitCode(false))
}
