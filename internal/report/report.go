package report

import "github.com/aarjang/push-all-repo/internal/model"

// Report is the grouped view of a Run.
type Report struct {
	Run    model.Run
	Groups map[model.Status][]model.Result
	Counts map[model.Status]int
	// NotRepos are NotPushed results whose reason is NotARepo or NestedRepo;
	// they are listed compactly and do not count as failures.
	NotRepos []model.Result
	// Folders is the number of folders considered (all results).
	Folders int
}

// Build groups the run's results in the fixed status order.
func Build(run model.Run) Report {
	r := Report{
		Run:    run,
		Groups: map[model.Status][]model.Result{},
		Counts: map[model.Status]int{},
	}
	for _, res := range run.Results {
		r.Folders++
		if res.Status == model.NotPushed && isFolderOnly(res.Reason) {
			r.NotRepos = append(r.NotRepos, res)
			continue
		}
		r.Groups[res.Status] = append(r.Groups[res.Status], res)
		r.Counts[res.Status]++
	}
	return r
}

func isFolderOnly(reason model.Reason) bool {
	return reason == model.NotARepo || reason == model.NestedRepo
}

// ExitCode per data-model.md: 130 interrupted, 1 any not pushed (or any
// user-skipped with strict), else 0. Plain folders never fail the run.
func (r Report) ExitCode(strict bool) int {
	if r.Run.Interrupted {
		return 130
	}
	if r.Counts[model.NotPushed] > 0 || r.Counts[model.NotProcessed] > 0 {
		return 1
	}
	if strict && r.Counts[model.SkippedByUser] > 0 {
		return 1
	}
	return 0
}
