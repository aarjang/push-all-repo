package run

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/aarjang/push-all-repo/internal/discover"
	"github.com/aarjang/push-all-repo/internal/github"
	"github.com/aarjang/push-all-repo/internal/gitx"
	"github.com/aarjang/push-all-repo/internal/model"
	"github.com/aarjang/push-all-repo/internal/report"
	"github.com/aarjang/push-all-repo/internal/ui"
)

// Deps are the collaborators the orchestrator talks to.
type Deps struct {
	Git      gitx.Client
	GitHub   github.Client
	Prompter ui.Prompter
	Progress *ui.Progress
	// Out receives the report; Err receives warnings and errors.
	Out, Err io.Writer
	Style    ui.Style
	Now      func() time.Time
	Version  string
	// Cwd is where the default report file goes.
	Cwd string
}

// Exit codes (contracts/cli.md).
const (
	ExitOK          = 0
	ExitNotPushed   = 1
	ExitUsage       = 2
	ExitInterrupted = 130
)

type runner struct {
	ctx  context.Context
	opts model.Options
	d    Deps
	run  model.Run

	dirtyAll   *model.Decision
	createAll  *model.Decision
	warnedGh   bool
	warnedFlag bool
}

// Run executes the whole flow and returns the finished Run and exit code.
func Run(ctx context.Context, opts model.Options, d Deps) (model.Run, int) {
	if d.Now == nil {
		d.Now = time.Now
	}
	if d.Err == nil {
		d.Err = io.Discard
	}
	if d.Out == nil {
		d.Out = io.Discard
	}
	r := &runner{ctx: ctx, opts: opts, d: d}
	r.run = model.Run{Options: opts, StartedAt: d.Now(), ToolVersion: d.Version}
	code := r.execute()
	return r.run, code
}

func (r *runner) fail(err error) int {
	var nf *ui.NeedFlagError
	switch {
	case errors.Is(err, ui.ErrAborted), errors.Is(err, context.Canceled):
		fmt.Fprintln(r.d.Err, "pushall: cancelled")
		return ExitInterrupted
	case errors.As(err, &nf):
		fmt.Fprintf(r.d.Err, "pushall: %v\n", err)
		return ExitUsage
	default:
		fmt.Fprintf(r.d.Err, "pushall: %v\n", err)
		return ExitUsage
	}
}

func (r *runner) execute() int {
	// 1. mode
	mode := r.opts.Mode
	if mode == model.ModeUnset {
		m, err := r.d.Prompter.Mode()
		if err != nil {
			return r.fail(err)
		}
		mode = m
	}
	r.run.Mode = mode

	// 2. path
	path := r.opts.Path
	if path == "" {
		p, err := r.d.Prompter.Path(mode, func(s string) error {
			_, err := r.validatePath(mode, s)
			return err
		})
		if err != nil {
			return r.fail(err)
		}
		path = p
	}
	abs, err := r.validatePath(mode, path)
	if err != nil {
		return r.fail(err)
	}
	r.run.RootPath = abs

	// 3. discovery
	var cands []model.Candidate
	if mode == model.SingleProject {
		c, err := r.d.Git.Inspect(r.ctx, abs)
		if err != nil {
			return r.fail(err)
		}
		cands = []model.Candidate{c}
	} else {
		cs, err := discover.Discover(r.ctx, abs, r.d.Git)
		if err != nil {
			if r.ctx.Err() != nil {
				// Interrupted during discovery: report what was found so far.
				fmt.Fprintln(r.d.Err, "pushall: interrupted during discovery")
				r.run.Interrupted = true
				for _, c := range cs {
					r.run.Results = append(r.run.Results, model.Result{Candidate: c, Status: model.NotProcessed, Reason: model.Interrupted})
				}
				r.run.FinishedAt = r.d.Now()
				rep := report.Build(r.run)
				fmt.Fprintln(r.d.Out)
				report.RenderText(r.d.Out, rep, r.d.Style)
				return rep.ExitCode(r.opts.Strict)
			}
			return r.fail(fmt.Errorf("cannot read %s: %w", abs, err))
		}
		cands = cs
		repos := 0
		for _, c := range cs {
			if c.IsRepo {
				repos++
			}
		}
		fmt.Fprintf(r.d.Out, "Found %d %s under %s\n", repos, plural(repos, "repository", "repositories"), abs)

		// 4. review + confirm (US3)
		if !r.opts.NoReview {
			kept, err := r.d.Prompter.Review(cands)
			if err != nil {
				return r.fail(err)
			}
			keep := map[string]bool{}
			for _, k := range kept {
				keep[k.Path] = true
			}
			n := 0
			for i := range cands {
				if cands[i].IsRepo && !keep[cands[i].Path] {
					cands[i].Excluded = true
				} else if cands[i].IsRepo {
					n++
				}
			}
			if n > 0 {
				ok, err := r.d.Prompter.ConfirmStart(n)
				if err != nil {
					return r.fail(err)
				}
				if !ok {
					r.run.FinishedAt = r.d.Now()
					rep := report.Build(r.run)
					report.RenderText(r.d.Out, rep, r.d.Style)
					return ExitOK
				}
			}
		}
	}

	// 5. the loop
	total := len(cands)
	for i, c := range cands {
		if r.ctx.Err() != nil {
			r.run.Interrupted = true
			for _, rest := range cands[i:] {
				r.run.Results = append(r.run.Results, model.Result{Candidate: rest, Status: model.NotProcessed, Reason: model.Interrupted})
			}
			break
		}
		r.d.Progress.Start(i+1, total, c.Name)
		res := r.processOne(c)
		r.d.Progress.Done(res)
		r.run.Results = append(r.run.Results, res)
		if res.Reason == model.Interrupted && res.Status == model.NotPushed {
			r.run.Interrupted = true
			for _, rest := range cands[i+1:] {
				r.run.Results = append(r.run.Results, model.Result{Candidate: rest, Status: model.NotProcessed, Reason: model.Interrupted})
			}
			break
		}
	}
	r.run.FinishedAt = r.d.Now()

	// 6. report
	rep := report.Build(r.run)
	fmt.Fprintln(r.d.Out)
	report.RenderText(r.d.Out, rep, r.d.Style)
	r.saveReport(rep)
	return rep.ExitCode(r.opts.Strict)
}

// validatePath implements FR-003 and the single-mode checks of US2.
func (r *runner) validatePath(mode model.Mode, p string) (string, error) {
	if p == "" {
		return "", errors.New("path is empty")
	}
	p = expandHome(p)
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	fi, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("path does not exist: %s", abs)
		}
		return "", fmt.Errorf("cannot access %s: %w", abs, err)
	}
	if !fi.IsDir() {
		return "", fmt.Errorf("not a folder: %s", abs)
	}
	if f, err := os.Open(abs); err != nil {
		return "", fmt.Errorf("cannot read %s: %w", abs, err)
	} else {
		_ = f.Close()
	}
	if mode == model.SingleProject {
		c, err := r.d.Git.Inspect(r.ctx, abs)
		if err != nil {
			return "", err
		}
		if c.Nested {
			return "", fmt.Errorf("%s is inside repository %s; run on %s instead", abs, c.TopLevel, c.TopLevel)
		}
		if !c.IsRepo {
			return "", fmt.Errorf("not a Git repository: %s", abs)
		}
	}
	return abs, nil
}

func expandHome(p string) string {
	if len(p) > 0 && p[0] == '~' && (len(p) == 1 || os.IsPathSeparator(p[1])) {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, p[1:])
		}
	}
	return p
}

// processOne walks the per-candidate state machine from data-model.md.
func (r *runner) processOne(c model.Candidate) (res model.Result) {
	start := r.d.Now()
	res = model.Result{Candidate: c}
	defer func() {
		res.Duration = r.d.Now().Sub(start)
		res.Candidate = c
	}()

	switch {
	case c.Excluded:
		res.Status, res.Detail = model.SkippedByUser, "excluded in review"
		return
	case !c.IsRepo && c.Nested:
		res.Status, res.Reason = model.NotPushed, model.NestedRepo
		return
	case !c.IsRepo:
		res.Status, res.Reason = model.NotPushed, model.NotARepo
		return
	case c.Branch == "":
		res.Status, res.Reason = model.NotPushed, model.DetachedHead
		return
	case c.InProgressOp != "":
		res.Status, res.Reason, res.Detail = model.NotPushed, model.OperationInProgress, c.InProgressOp
		return
	}

	// uncommitted changes (FR-007)
	if c.Dirty {
		dec, err := r.resolveDirty(c)
		if err != nil {
			return r.errResult(res, err)
		}
		switch dec.DirtyChoice {
		case model.Skip:
			res.Status, res.Detail = model.SkippedByUser, "uncommitted changes"
			return
		case model.CommitAndPush:
			if !c.IdentityOK {
				res.Status, res.Reason = model.NotPushed, model.IdentityMissing
				return
			}
			if err := r.d.Git.AddAll(r.ctx, c.Path); err != nil {
				return r.opErrResult(res, err)
			}
			if err := r.d.Git.Commit(r.ctx, c.Path, dec.CommitMessage); err != nil {
				return r.opErrResult(res, err)
			}
			res.Committed, res.CommitMessage = true, dec.CommitMessage
			if c.AheadCount >= 0 {
				c.AheadCount++
			}
			c.Dirty = false
		default: // PushOnly
			res.Detail = "uncommitted changes left untouched"
		}
	}

	// no remote (FR-008). A repository whose remotes are all named something
	// other than origin, with no upstream, has nowhere to push either: the
	// spec sends it through the same flow.
	if !c.HasRemote() {
		return r.handleNoRemote(c, res)
	}
	if c.Upstream == "" && !c.HasOrigin() {
		res.SetDetail("no upstream and no remote named origin")
		return r.handleNoRemote(c, res)
	}

	if c.Upstream != "" && c.AheadCount == 0 {
		res.Status = model.UpToDate
		return
	}

	if err := r.d.Git.Push(r.ctx, c, r.opts.Timeout); err != nil {
		return r.opErrResult(res, err)
	}
	res.Status = model.Pushed
	return
}

func (r *runner) errResult(res model.Result, err error) model.Result {
	if errors.Is(err, ui.ErrAborted) || errors.Is(err, context.Canceled) || r.ctx.Err() != nil {
		res.Status, res.Reason = model.NotPushed, model.Interrupted
		return res
	}
	var nf *ui.NeedFlagError
	if errors.As(err, &nf) {
		res.Status, res.Reason = model.NotPushed, model.AnswerRequired
		res.SetDetail("pass " + nf.Flag)
		if !r.warnedFlag {
			r.warnedFlag = true
			fmt.Fprintf(r.d.Err, "pushall: %v\n", err)
		}
		return res
	}
	res.Status, res.Reason = model.NotPushed, model.GitError
	res.SetDetail(err.Error())
	return res
}

func (r *runner) opErrResult(res model.Result, err error) model.Result {
	var oe *gitx.OpError
	if errors.As(err, &oe) {
		res.Status, res.Reason = model.NotPushed, oe.Reason
		res.SetDetail(oe.Detail)
		return res
	}
	reason, detail := gitx.Classify(err, "")
	res.Status, res.Reason = model.NotPushed, reason
	res.SetDetail(detail)
	return res
}

func (r *runner) defaultMessage() string {
	if r.opts.Message != "" {
		return r.opts.Message
	}
	return "chore: bulk push on " + r.d.Now().Format("2006-01-02 15:04")
}

func (r *runner) resolveDirty(c model.Candidate) (model.Decision, error) {
	if ch := r.opts.Dirty.Choice(); ch != model.DirtyUnset {
		return model.Decision{Kind: model.DecisionDirty, DirtyChoice: ch, CommitMessage: r.defaultMessage()}, nil
	}
	if r.dirtyAll != nil {
		d := *r.dirtyAll
		if d.DirtyChoice == model.CommitAndPush && d.CommitMessage == "" {
			d.CommitMessage = r.defaultMessage()
		}
		return d, nil
	}
	d, err := r.d.Prompter.DirtyDecision(c, r.defaultMessage())
	if err != nil {
		return d, err
	}
	if d.DirtyChoice == model.DirtyUnset {
		d.DirtyChoice = model.PushOnly
	}
	if d.DirtyChoice == model.CommitAndPush && d.CommitMessage == "" {
		d.CommitMessage = r.defaultMessage()
	}
	if d.ApplyToAll {
		keep := d
		r.dirtyAll = &keep
	}
	return d, nil
}

func (r *runner) handleNoRemote(c model.Candidate, res model.Result) model.Result {
	if r.opts.CreateRepo == model.CreateRepoNo {
		res.Status, res.Reason = model.NotPushed, model.NoRemoteDeclined
		return res
	}
	if r.d.GitHub == nil || !r.d.GitHub.Available() || !r.d.GitHub.LoggedIn(r.ctx) {
		if !r.warnedGh {
			r.warnedGh = true
			if r.d.GitHub == nil || !r.d.GitHub.Available() {
				fmt.Fprintln(r.d.Err, "pushall: GitHub CLI (gh) is not installed; repositories without a remote cannot be created")
			} else {
				fmt.Fprintln(r.d.Err, "pushall: not logged in to GitHub (run `gh auth login`); repositories without a remote cannot be created")
			}
		}
		res.Status, res.Reason = model.NotPushed, model.NoRemoteLoginRequired
		return res
	}

	var dec model.Decision
	switch {
	case r.opts.CreateRepo == model.CreateRepoYes:
		dec = model.Decision{Kind: model.DecisionCreateRepo, CreateRepo: true, RepoName: c.Name, Visibility: r.opts.Visibility}
	case r.createAll != nil:
		dec = *r.createAll
		dec.RepoName = c.Name
	default:
		d, err := r.d.Prompter.CreateRepoOffer(c, c.Name, r.opts.Visibility, false)
		if err != nil {
			return r.errResult(res, err)
		}
		if d.RepoName == "" {
			d.RepoName = c.Name
		}
		if d.ApplyToAll {
			keep := d
			r.createAll = &keep
		}
		dec = d
	}
	if !dec.CreateRepo {
		res.Status, res.Reason = model.NotPushed, model.NoRemoteDeclined
		return res
	}

	for {
		url, err := r.d.GitHub.CreateRepo(r.ctx, c.Path, dec.RepoName, dec.Visibility == model.Private, r.opts.Timeout)
		if err == nil {
			res.Status = model.Pushed
			res.CreatedRemote = url
			if url == "" {
				res.CreatedRemote = "github.com/…/" + dec.RepoName
			}
			res.CreatedVisibility = dec.Visibility
			return res
		}
		if errors.Is(err, github.ErrNameTaken) {
			if r.opts.CreateRepo == model.CreateRepoYes {
				res.Status, res.Reason = model.NotPushed, model.CreateRepoFailed
				res.SetDetail(fmt.Sprintf("repository name %q already exists", dec.RepoName))
				return res
			}
			d, perr := r.d.Prompter.CreateRepoOffer(c, dec.RepoName, dec.Visibility, true)
			if perr != nil {
				return r.errResult(res, perr)
			}
			if !d.CreateRepo || d.RepoName == "" || d.RepoName == dec.RepoName {
				res.Status, res.Reason = model.NotPushed, model.CreateRepoFailed
				res.SetDetail(fmt.Sprintf("repository name %q already exists", dec.RepoName))
				return res
			}
			dec.RepoName, dec.Visibility = d.RepoName, d.Visibility
			continue
		}
		var ee *gitx.ExitError
		if errors.As(err, &ee) && (ee.TimedOut || ee.Cancelled) {
			return r.opErrResult(res, err)
		}
		res.Status, res.Reason = model.NotPushed, model.CreateRepoFailed
		res.SetDetail(err.Error())
		return res
	}
}

// saveReport implements US4: --report PATH, --report - or the prompt.
func (r *runner) saveReport(rep report.Report) {
	path := r.opts.ReportPath
	switch path {
	case "-":
		return
	case "":
		def := filepath.Join(r.d.Cwd, report.DefaultReportPath(r.d.Now()))
		p, ok, err := r.d.Prompter.SaveReport(def)
		if err != nil || !ok {
			return
		}
		path = p
	}
	if err := report.Save(path, rep); err != nil {
		fmt.Fprintf(r.d.Err, "pushall: could not save report: %v\n", err)
		return
	}
	fmt.Fprintf(r.d.Out, "Report saved to %s\n", path)
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}
